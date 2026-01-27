package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	dockercontainer "github.com/docker/docker/api/types/container"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/tufanbarisyildirim/gonginx/config"
	"github.com/tufanbarisyildirim/gonginx/dumper"
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
		if p.Public > 0 {
			return p.Public, true
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

	useLinuxserverConfs := envBoolDefault("NGINX_USE_LINUXSERVER_CONFS", true)

	sort.Slice(snapshot, func(i, j int) bool {
		if snapshot[i].Host == snapshot[j].Host {
			return snapshot[i].Name < snapshot[j].Name
		}
		return snapshot[i].Host < snapshot[j].Host
	})

	mappingDomains := make([]string, 0, len(cfg.ParsedMappings))
	for d := range cfg.ParsedMappings {
		mappingDomains = append(mappingDomains, d)
	}
	sort.Strings(mappingDomains)

	var customConfigs []string

	conf := &config.Config{
		Block: &config.Block{
			Directives: []config.IDirective{},
		},
	}

	for _, domain := range mappingDomains {
		target := cfg.ParsedMappings[domain]
		upstream, err := resolveTargetPort(target, cfg)
		if err != nil {
			log.Printf("WARN nginx-gen: skipping domain %s: %v", domain, err)
			continue
		}

		serverBlock := buildServerBlock(domain, upstream)
		conf.Block.Directives = append(conf.Block.Directives, serverBlock)
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

		if useLinuxserverConfs {
			if lsConfig := tryLinuxserverConfig(c.Name, hostAddr, int(port), cfg.Defaults.Scheme, fqdn); lsConfig != "" {
				customConfigs = append(customConfigs, lsConfig)
				continue
			}
		}

		serverBlock := buildServerBlock(fqdn, upstream)
		conf.Block.Directives = append(conf.Block.Directives, serverBlock)
	}

	style := &dumper.Style{
		SortDirectives:    false,
		SpaceBeforeBlocks: true,
		StartIndent:       0,
		Indent:            2,
	}
	generated := dumper.DumpConfig(conf, style)

	if len(customConfigs) > 0 {
		generated = generated + "\n" + strings.Join(customConfigs, "\n")
	}

	return generated, nil
}

func buildServerBlock(serverName, upstream string) *config.Directive {
	serverBlock := &config.Block{Directives: []config.IDirective{}}

	server := &config.Directive{
		Name:  "server",
		Block: serverBlock,
	}

	serverBlock.Directives = append(serverBlock.Directives, &config.Directive{
		Name:       "listen",
		Parameters: []config.Parameter{{Value: "80"}},
	})

	serverBlock.Directives = append(serverBlock.Directives, &config.Directive{
		Name:       "server_name",
		Parameters: []config.Parameter{{Value: serverName}},
	})

	locationBlock := &config.Block{Directives: []config.IDirective{}}
	location := &config.Directive{
		Name:       "location",
		Parameters: []config.Parameter{{Value: "/"}},
		Block:      locationBlock,
	}

	locationBlock.Directives = append(locationBlock.Directives, &config.Directive{
		Name:       "proxy_http_version",
		Parameters: []config.Parameter{{Value: "1.1"}},
	})

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
		locationBlock.Directives = append(locationBlock.Directives, &config.Directive{
			Name:       "proxy_set_header",
			Parameters: []config.Parameter{{Value: header.name}, {Value: header.value}},
		})
	}

	locationBlock.Directives = append(locationBlock.Directives, &config.Directive{
		Name:       "proxy_pass",
		Parameters: []config.Parameter{{Value: "http://" + upstream}},
	})

	serverBlock.Directives = append(serverBlock.Directives, location)
	return server
}

// nginxExecInContainer runs a command inside the nginx container via the Docker API.
// Returns combined stdout+stderr output and any error (including non-zero exit code).
func nginxExecInContainer(ctx context.Context, cli dockerclient.APIClient, containerName string, cmd []string) (string, error) {
	execCfg := dockercontainer.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := cli.ContainerExecCreate(ctx, containerName, execCfg)
	if err != nil {
		return "", fmt.Errorf("exec create: %w", err)
	}

	attachResp, err := cli.ContainerExecAttach(ctx, execResp.ID, dockercontainer.ExecAttachOptions{})
	if err != nil {
		return "", fmt.Errorf("exec attach: %w", err)
	}
	defer attachResp.Close()

	var stdout, stderr bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdout, &stderr, attachResp.Reader); err != nil {
		return "", fmt.Errorf("exec read output: %w", err)
	}

	inspectResp, err := cli.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return stdout.String() + stderr.String(), fmt.Errorf("exec inspect: %w", err)
	}

	combined := stdout.String() + stderr.String()
	if inspectResp.ExitCode != 0 {
		return combined, fmt.Errorf("exit code %d", inspectResp.ExitCode)
	}
	return combined, nil
}

func startNginxGeneratorLoop(ctx context.Context, store *StateStore, api *API, warns *warnLimiter) {
	enabled := envBool("NGINX_CONF_GEN_ENABLED")

	log.Printf("nginx-gen: called, NGINX_CONF_GEN_ENABLED=%v (raw=%q)", enabled, os.Getenv("NGINX_CONF_GEN_ENABLED"))
	if !enabled {
		log.Println("nginx-gen: NGINX_CONF_GEN_ENABLED is false, exiting")
		return
	}

	generatedPath := strings.TrimSpace(os.Getenv("NGINX_GENERATED_CONF"))
	if generatedPath == "" {
		generatedPath = "/etc/nginx/conf.d/switchboard.generated.conf"
	}

	nginxContainer := strings.TrimSpace(os.Getenv("NGINX_CONTAINER_NAME"))
	if nginxContainer == "" {
		nginxContainer = "switchboard-nginx"
	}

	var dockerCli dockerclient.APIClient
	cli, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		warns.Warnf("nginx-docker-init", 30*time.Second, "WARN nginx-gen: failed to create Docker client: %v (reload via API disabled)", err)
	} else {
		dockerCli = cli
	}

	log.Printf("nginx-gen: starting loop, target=%s, nginx_container=%s", generatedPath, nginxContainer)

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

		api.configMutex.RLock()
		cfg := api.config
		api.configMutex.RUnlock()

		rendered, err := renderNginxConfig(snapshot, cfg)
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
		case <-api.ConfigChanged():
			log.Println("nginx-gen: config changed, re-rendering")
			render()
		case <-debounceTimer:
			if pending == "" {
				continue
			}
			if err := applyNginxConfig(ctx, generatedPath, pending, dockerCli, nginxContainer, warns); err == nil {
				lastAppliedHash = pendingHash
				log.Printf("nginx-gen: applied %d bytes", len(pending))
			}
			pending = ""
			debounceTimer = nil
		}
	}
}

func applyNginxConfig(ctx context.Context, generatedPath, configContent string, dockerCli dockerclient.APIClient, nginxContainer string, warns *warnLimiter) error {
	dir := filepath.Dir(generatedPath)
	tmp := filepath.Join(dir, ".switchboard.generated.conf.tmp")

	log.Printf("nginx-gen: applying config (%d bytes)", len(configContent))

	prevBytes, _ := os.ReadFile(generatedPath)
	if err := os.WriteFile(tmp, []byte(configContent), 0644); err != nil {
		warns.Warnf("nginx-write", 30*time.Second, "WARN nginx-gen: write %s: %v", tmp, err)
		return err
	}
	if err := os.Rename(tmp, generatedPath); err != nil {
		warns.Warnf("nginx-rename", 30*time.Second, "WARN nginx-gen: rename %s -> %s: %v", tmp, generatedPath, err)
		return err
	}

	if dockerCli == nil {
		warns.Warnf("nginx-no-docker", 30*time.Second, "WARN nginx-gen: no Docker client, config written but nginx not reloaded")
		return nil
	}

	output, err := nginxExecInContainer(ctx, dockerCli, nginxContainer, []string{"nginx", "-t"})
	if err != nil {
		_ = os.WriteFile(generatedPath, prevBytes, 0644)
		warns.Warnf("nginx-test", 30*time.Second, "WARN nginx-gen: nginx -t failed (rolled back): %v\n%s", err, output)
		return err
	}

	output, err = nginxExecInContainer(ctx, dockerCli, nginxContainer, []string{"nginx", "-s", "reload"})
	if err != nil {
		nginxReloadErrors.Inc()
		warns.Warnf("nginx-reload", 30*time.Second, "WARN nginx-gen: nginx reload failed: %v\n%s", err, output)
		return err
	}

	nginxReloadsTotal.Inc()
	return nil
}
