package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nginxinc/nginx-go-crossplane"
)

func renderNginxConfigCrossplane(snapshot []Container, cfg Config) (string, error) {
	domain := strings.TrimSpace(cfg.Defaults.BaseDomain)
	if domain == "" {
		return "", nil
	}
	if cfg.HostAddresses == nil {
		return "", nil
	}

	var directives crossplane.Directives

	directives = append(directives, &crossplane.Directive{
		Directive: "#",
		Line:      1,
		Args:      []string{"GENERATED FILE. DO NOT EDIT."},
		Comment:   stringPtr("GENERATED FILE. DO NOT EDIT."),
	})
	directives = append(directives, &crossplane.Directive{
		Directive: "#",
		Line:      2,
		Args:      []string{"Source: Switchboard container snapshot"},
		Comment:   stringPtr("Source: Switchboard container snapshot"),
	})

	lineNum := 3

	for domain, target := range cfg.ParsedMappings {
		upstream, err := resolveTargetPort(target, cfg)
		if err != nil {
			log.Printf("WARN nginx-gen: skipping domain %s: %v", domain, err)
			continue
		}

		serverBlock := &crossplane.Directive{
			Directive: "server",
			Line:      lineNum,
			Block:     crossplane.Directives{},
		}
		lineNum++

		serverBlock.Block = append(serverBlock.Block,
			&crossplane.Directive{
				Directive: "listen",
				Line:      lineNum,
				Args:      []string{"80"},
			},
		)
		lineNum++

		serverBlock.Block = append(serverBlock.Block,
			&crossplane.Directive{
				Directive: "server_name",
				Line:      lineNum,
				Args:      []string{domain},
			},
		)
		lineNum++

		locationBlock := &crossplane.Directive{
			Directive: "location",
			Line:      lineNum,
			Args:      []string{"/"},
			Block:     crossplane.Directives{},
		}
		lineNum++

		locationBlock.Block = append(locationBlock.Block,
			&crossplane.Directive{
				Directive: "proxy_http_version",
				Line:      lineNum,
				Args:      []string{"1.1"},
			},
		)
		lineNum++

		proxyHeaders := []struct {
			name  string
			value string
		}{
			{"Host", "$host"},
			{"X-Real-IP", "$remote_addr"},
			{"X-Forwarded-For", "$proxy_add_x_forwarded_for"},
			{"X-Forwarded-Proto", "$scheme"},
			{"Upgrade", "$http_upgrade"},
			{"Connection", "$connection_upgrade"},
		}

		for _, header := range proxyHeaders {
			locationBlock.Block = append(locationBlock.Block,
				&crossplane.Directive{
					Directive: "proxy_set_header",
					Line:      lineNum,
					Args:      []string{header.name, header.value},
				},
			)
			lineNum++
		}

		locationBlock.Block = append(locationBlock.Block,
			&crossplane.Directive{
				Directive: "proxy_pass",
				Line:      lineNum,
				Args:      []string{"http://" + upstream},
			},
		)
		lineNum++

		serverBlock.Block = append(serverBlock.Block, locationBlock)
		directives = append(directives, serverBlock)
	}

	for _, c := range snapshot {
		if strings.ToLower(c.State) != "running" {
			continue
		}

		port, ok := choosePublishedPort(c)
		if !ok {
			continue
		}

		hostAddr := strings.TrimSpace(cfg.HostAddresses[c.Host])
		if hostAddr == "" {
			continue
		}

		containerLabel := sanitizeDNSLabel(c.Name)
		hostLabel := sanitizeDNSLabel(c.Host)
		fqdn := containerLabel + "." + hostLabel + "." + domain

		serverBlock := &crossplane.Directive{
			Directive: "server",
			Line:      lineNum,
			Block:     crossplane.Directives{},
		}
		lineNum++

		serverBlock.Block = append(serverBlock.Block,
			&crossplane.Directive{
				Directive: "listen",
				Line:      lineNum,
				Args:      []string{"80"},
			},
		)
		lineNum++

		serverBlock.Block = append(serverBlock.Block,
			&crossplane.Directive{
				Directive: "server_name",
				Line:      lineNum,
				Args:      []string{fqdn},
			},
		)
		lineNum++

		locationBlock := &crossplane.Directive{
			Directive: "location",
			Line:      lineNum,
			Args:      []string{"/"},
			Block:     crossplane.Directives{},
		}
		lineNum++

		locationBlock.Block = append(locationBlock.Block,
			&crossplane.Directive{
				Directive: "proxy_http_version",
				Line:      lineNum,
				Args:      []string{"1.1"},
			},
		)
		lineNum++

		proxyHeaders := []struct {
			name  string
			value string
		}{
			{"Host", "$host"},
			{"X-Real-IP", "$remote_addr"},
			{"X-Forwarded-For", "$proxy_add_x_forwarded_for"},
			{"X-Forwarded-Proto", "$scheme"},
			{"Upgrade", "$http_upgrade"},
			{"Connection", "$connection_upgrade"},
		}

		for _, header := range proxyHeaders {
			locationBlock.Block = append(locationBlock.Block,
				&crossplane.Directive{
					Directive: "proxy_set_header",
					Line:      lineNum,
					Args:      []string{header.name, header.value},
				},
			)
			lineNum++
		}

		upstream := hostAddr + ":" + strconv.Itoa(int(port))
		locationBlock.Block = append(locationBlock.Block,
			&crossplane.Directive{
				Directive: "proxy_pass",
				Line:      lineNum,
				Args:      []string{"http://" + upstream},
			},
		)
		lineNum++

		serverBlock.Block = append(serverBlock.Block, locationBlock)
		directives = append(directives, serverBlock)
	}

	config := crossplane.Config{
		File:   "switchboard.generated.conf",
		Parsed: directives,
	}

	var buf bytes.Buffer
	err := crossplane.Build(&buf, config, &crossplane.BuildOptions{
		Indent: 2,
		Tabs:   false,
		Header: false,
	})
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func stringPtr(s string) *string {
	return &s
}

func startNginxGeneratorLoopCrossplane(ctx context.Context, store *StateStore, cfg Config, warns *warnLimiter) {
	enabled := envBool("NGINX_ENABLED")
	log.Printf("nginx-gen: called, NGINX_ENABLED=%v (raw=%q)", enabled, os.Getenv("NGINX_ENABLED"))
	if !enabled {
		log.Println("nginx-gen: NGINX_ENABLED is false, exiting")
		return
	}

	generatedPath := strings.TrimSpace(os.Getenv("NGINX_GENERATED_CONF"))
	if generatedPath == "" {
		generatedPath = "/etc/nginx/conf.d/switchboard.generated.conf"
	}

	log.Printf("nginx-gen: starting crossplane loop, target=%s", generatedPath)

	baseDebounce := 1500 * time.Millisecond
	if v := strings.TrimSpace(os.Getenv("NGINX_RELOAD_DEBOUNCE")); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			baseDebounce = d
		}
	}

	tracker := newDebounceTracker()
	var lastAppliedHash [32]byte
	var pending string
	var pendingHash [32]byte
	var debounceTimer <-chan time.Time

	render := func() {
		start := time.Now()
		snapshot := store.Snapshot()
		rendered, err := renderNginxConfigCrossplane(snapshot, cfg)
		nginxConfigGenDuration.Observe(time.Since(start).Seconds())
		if err != nil {
			warns.Warnf("nginx-render", 30*time.Second, "WARN nginx-gen: %v", err)
			return
		}
		hash := sha256.Sum256([]byte(rendered))
		if hash == lastAppliedHash {
			pending = ""
			return
		}
		pending = rendered
		pendingHash = hash

		tracker.recordChange()
		adaptiveDebounce := tracker.calculateDebounce(baseDebounce)
		debounceTimer = time.After(adaptiveDebounce)
		log.Printf("nginx-gen: config changed, will apply after adaptive debounce (%v)", adaptiveDebounce)
	}

	render()

	for {
		select {
		case <-ctx.Done():
			return
		case <-store.Changed():
			render()
		case <-debounceTimer:
			if pending == "" {
				continue
			}
			if err := applyNginxConfig(generatedPath, pending, warns); err == nil {
				lastAppliedHash = pendingHash
				log.Printf("nginx-gen: applied %d bytes using crossplane", len(pending))
			}
			pending = ""
			debounceTimer = nil
		}
	}
}
