package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var dnsLabelRe = regexp.MustCompile(`[^a-z0-9-]`)

func sanitizeDNSLabel(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "_", "-")
	s = dnsLabelRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "x"
	}
	return s
}

func choosePublishedPort(c Container) (uint16, bool) {
	for _, p := range c.Ports {
		if p.PublicPort > 0 {
			return uint16(p.PublicPort), true
		}
	}
	return 0, false
}

func defaultPortForScheme(scheme string) int {
	if strings.ToLower(strings.TrimSpace(scheme)) == "https" {
		return 443
	}
	return 80
}

func resolveTargetPort(target ProxyTarget, cfg Config) (string, error) {
	hostAddr := strings.TrimSpace(cfg.HostAddresses[target.Host])
	if hostAddr == "" {
		return "", fmt.Errorf("host %q not found in host_addresses", target.Host)
	}

	port := target.Port
	if port == 0 {
		port = defaultPortForScheme(cfg.Defaults.Scheme)
	}

	return hostAddr + ":" + strconv.Itoa(port), nil
}

func renderNginxConfig(snapshot []Container, cfg Config) (string, error) {
	domain := strings.TrimSpace(cfg.Defaults.BaseDomain)
	if domain == "" {
		return "", fmt.Errorf("defaults.base_domain is empty")
	}
	if cfg.HostAddresses == nil {
		return "", fmt.Errorf("host_addresses is not configured")
	}

	sort.Slice(snapshot, func(i, j int) bool {
		if snapshot[i].Host == snapshot[j].Host {
			return snapshot[i].Name < snapshot[j].Name
		}
		return snapshot[i].Host < snapshot[j].Host
	})

	var b strings.Builder
	b.WriteString("# GENERATED FILE. DO NOT EDIT.\n")
	b.WriteString("# Source: Switchboard container snapshot\n\n")

	for domain, target := range cfg.ParsedMappings {
		upstream, err := resolveTargetPort(target, cfg)
		if err != nil {
			log.Printf("WARN nginx-gen: skipping domain %s: %v", domain, err)
			continue
		}

		b.WriteString("server {\n")
		b.WriteString("  listen 80;\n")
		b.WriteString("  server_name " + domain + ";\n")
		b.WriteString("\n")
		b.WriteString("  location / {\n")
		b.WriteString("    proxy_http_version 1.1;\n")
		b.WriteString("    proxy_set_header Host $host;\n")
		b.WriteString("    proxy_set_header X-Real-IP $remote_addr;\n")
		b.WriteString("    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
		b.WriteString("    proxy_set_header X-Forwarded-Proto $scheme;\n")
		b.WriteString("    proxy_set_header Upgrade $http_upgrade;\n")
		b.WriteString("    proxy_set_header Connection $connection_upgrade;\n")
		b.WriteString("    proxy_pass http://" + upstream + ";\n")
		b.WriteString("  }\n")
		b.WriteString("}\n\n")
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
		fqdn := fmt.Sprintf("%s.%s.%s", containerLabel, hostLabel, domain)
		upstream := hostAddr + ":" + strconv.Itoa(int(port))

		b.WriteString("server {\n")
		b.WriteString("  listen 80;\n")
		b.WriteString("  server_name " + fqdn + ";\n")
		b.WriteString("\n")
		b.WriteString("  location / {\n")
		b.WriteString("    proxy_http_version 1.1;\n")
		b.WriteString("    proxy_set_header Host $host;\n")
		b.WriteString("    proxy_set_header X-Real-IP $remote_addr;\n")
		b.WriteString("    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
		b.WriteString("    proxy_set_header X-Forwarded-Proto $scheme;\n")
		b.WriteString("    proxy_set_header Upgrade $http_upgrade;\n")
		b.WriteString("    proxy_set_header Connection $connection_upgrade;\n")
		b.WriteString("    proxy_pass http://" + upstream + ";\n")
		b.WriteString("  }\n")
		b.WriteString("}\n\n")
	}

	return b.String(), nil
}

func startNginxGeneratorLoop(ctx context.Context, store *StateStore, cfg Config, warns *warnLimiter) {
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

	log.Printf("nginx-gen: starting loop, target=%s", generatedPath)

	debounce := 1500 * time.Millisecond
	if v := strings.TrimSpace(os.Getenv("NGINX_RELOAD_DEBOUNCE")); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			debounce = d
		}
	}

	var lastApplied string
	var pending string
	var lastChange time.Time

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if pending == "" {
				snapshot := store.Snapshot()
				rendered, err := renderNginxConfig(snapshot, cfg)
				if err != nil {
					warns.Warnf("nginx-render", 30*time.Second, "WARN nginx-gen: %v", err)
					continue
				}
				log.Printf("nginx-gen: rendered %d bytes, lastApplied %d bytes", len(rendered), len(lastApplied))
				if rendered != lastApplied {
					pending = rendered
					lastChange = time.Now()
					log.Printf("nginx-gen: pending config set, will apply after debounce (%v)", debounce)
				}
			}

			if pending == "" {
				continue
			}
			if time.Since(lastChange) < debounce {
				continue
			}

			// Apply pending config.
			dir := filepath.Dir(generatedPath)
			tmp := filepath.Join(dir, ".switchboard.generated.conf.tmp")

			prevBytes, _ := os.ReadFile(generatedPath)
			if err := os.WriteFile(tmp, []byte(pending), 0644); err != nil {
				warns.Warnf("nginx-write", 30*time.Second, "WARN nginx-gen: write %s: %v", tmp, err)
				pending = ""
				continue
			}
			if err := os.Rename(tmp, generatedPath); err != nil {
				warns.Warnf("nginx-rename", 30*time.Second, "WARN nginx-gen: rename %s -> %s: %v", tmp, generatedPath, err)
				pending = ""
				continue
			}

			if err := exec.Command("nginx", "-t").Run(); err != nil {
				_ = os.WriteFile(generatedPath, prevBytes, 0644)
				warns.Warnf("nginx-test", 30*time.Second, "WARN nginx-gen: nginx -t failed (rolled back): %v", err)
				pending = ""
				continue
			}

			if err := exec.Command("nginx", "-s", "reload").Run(); err != nil {
				warns.Warnf("nginx-reload", 30*time.Second, "WARN nginx-gen: nginx reload failed: %v", err)
				pending = ""
				continue
			}

			lastApplied = pending
			pending = ""
			log.Printf("nginx-gen: applied %d bytes", len(lastApplied))
		}
	}
}
