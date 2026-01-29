package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	linuxserverConfBaseURL = getLinuxserverConfURL()
	linuxserverConfSuffix  = ".subdomain.conf.sample"
	linuxserverRepoDir     = getLinuxserverRepoDir()
)

func getLinuxserverConfURL() string {
	if envURL := os.Getenv("LINUXSERVER_CONF_URL"); envURL != "" {
		return envURL
	}
	return "https://github.com/linuxserver/reverse-proxy-confs.git"
}

func getLinuxserverRepoDir() string {
	if envDir := os.Getenv("LINUXSERVER_REPO_DIR"); envDir != "" {
		return envDir
	}
	return "/tmp/linuxserver-reverse-proxy-confs"
}

type linuxserverConfigCache struct {
	mu      sync.RWMutex
	configs map[string]string
	fetched map[string]time.Time
	ttl     time.Duration
}

type linuxserverGitCache struct {
	mu      sync.RWMutex
	files   map[string]string
	fetched time.Time
	ttl     time.Duration
	syncErr error
	repoDir string
}

var lsConfigCache = &linuxserverConfigCache{
	configs: make(map[string]string),
	fetched: make(map[string]time.Time),
	ttl:     1 * time.Hour,
}

var lsGitCache = &linuxserverGitCache{
	files:   make(map[string]string),
	ttl:     24 * time.Hour,
	syncErr: fmt.Errorf("not yet synced"),
	repoDir: linuxserverRepoDir,
}

func fetchLinuxserverConfig(containerName string) (string, error) {
	lsConfigCache.mu.RLock()
	if config, ok := lsConfigCache.configs[containerName]; ok {
		if time.Since(lsConfigCache.fetched[containerName]) < lsConfigCache.ttl {
			lsConfigCache.mu.RUnlock()
			return config, nil
		}
	}
	lsConfigCache.mu.RUnlock()

	log.Printf("nginx-gen: fetching linuxserver config for %s from %s", containerName, linuxserverConfBaseURL)

	var config string
	var err error

	if strings.HasSuffix(linuxserverConfBaseURL, ".git") {
		config, err = fetchLinuxserverConfigFromGit(containerName)
	} else {
		config, err = fetchLinuxserverConfigFromRawURL(containerName)
	}

	if err != nil {
		return "", err
	}

	lsConfigCache.mu.Lock()
	lsConfigCache.configs[containerName] = config
	lsConfigCache.fetched[containerName] = time.Now()
	lsConfigCache.mu.Unlock()

	return config, nil
}

func fetchLinuxserverConfigFromRawURL(containerName string) (string, error) {
	url := linuxserverConfBaseURL + containerName + linuxserverConfSuffix
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return "", fmt.Errorf("config not found (404)")
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body failed: %w", err)
	}

	return string(body), nil
}

func fetchLinuxserverConfigFromGit(containerName string) (string, error) {
	if err := ensureGitRepoSynced(); err != nil {
		return "", err
	}

	lsGitCache.mu.RLock()
	defer lsGitCache.mu.RUnlock()

	configFileName := containerName + linuxserverConfSuffix
	if config, ok := lsGitCache.files[configFileName]; ok {
		return config, nil
	}

	return "", fmt.Errorf("config %s not found in git repository", configFileName)
}

func ensureGitRepoSynced() error {
	lsGitCache.mu.RLock()
	if lsGitCache.syncErr == nil && time.Since(lsGitCache.fetched) < lsGitCache.ttl {
		defer lsGitCache.mu.RUnlock()
		return nil
	}
	lsGitCache.mu.RUnlock()

	lsGitCache.mu.Lock()
	defer lsGitCache.mu.Unlock()

	if lsGitCache.syncErr == nil && time.Since(lsGitCache.fetched) < lsGitCache.ttl {
		return nil
	}

	log.Printf("nginx-gen: syncing linuxserver configs from %s to %s", linuxserverConfBaseURL, lsGitCache.repoDir)

	if _, err := os.Stat(lsGitCache.repoDir); os.IsNotExist(err) {
		if err := gitClone(linuxserverConfBaseURL, lsGitCache.repoDir); err != nil {
			lsGitCache.syncErr = fmt.Errorf("failed to clone repo: %w", err)
			return lsGitCache.syncErr
		}
	} else {
		if err := gitPull(lsGitCache.repoDir); err != nil {
			lsGitCache.syncErr = fmt.Errorf("failed to pull repo: %w", err)
			return lsGitCache.syncErr
		}
	}

	if err := loadConfigsFromGitRepo(lsGitCache.repoDir); err != nil {
		lsGitCache.syncErr = fmt.Errorf("failed to load configs: %w", err)
		return lsGitCache.syncErr
	}

	lsGitCache.fetched = time.Now()
	lsGitCache.syncErr = nil
	log.Printf("nginx-gen: synced %d linuxserver configs", len(lsGitCache.files))

	return nil
}

func gitClone(repoURL, targetDir string) error {
	cmd := exec.Command("git", "clone", "--depth=1", repoURL, targetDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone failed: %w\n%s", err, string(output))
	}
	return nil
}

func gitPull(repoDir string) error {
	cmd := exec.Command("git", "-C", repoDir, "pull", "--ff-only")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git pull failed: %w\n%s", err, string(output))
	}
	return nil
}

func loadConfigsFromGitRepo(repoDir string) error {
	lsGitCache.files = make(map[string]string)

	err := filepath.Walk(repoDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasSuffix(path, linuxserverConfSuffix) {
			body, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			fileName := filepath.Base(path)
			lsGitCache.files[fileName] = string(body)
		}

		return nil
	})

	return err
}

func replaceLinuxserverVars(template, containerName, hostAddr string, port int, scheme, domain string) (string, error) {
	config := template

	hostOnly, _, _ := strings.Cut(hostAddr, ":")
	if hostOnly == "" {
		hostOnly = hostAddr
	}
	hostOnly = strings.TrimSpace(hostOnly)

	if hostOnly == "" {
		return "", fmt.Errorf("hostAddr is empty after trimming")
	}

	upstreamAppRe := regexp.MustCompile(`(\s*)set\s+\$upstream_app\s+([^;]+);`)
	config = upstreamAppRe.ReplaceAllStringFunc(config, func(match string) string {
		submatch := upstreamAppRe.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		indent := submatch[1]
		return fmt.Sprintf("%sset $upstream_app %s;", indent, hostOnly)
	})

	upstreamPortRe := regexp.MustCompile(`(\s*)set\s+\$upstream_port\s+([^;]+);`)
	config = upstreamPortRe.ReplaceAllStringFunc(config, func(match string) string {
		submatch := upstreamPortRe.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		indent := submatch[1]
		return fmt.Sprintf("%sset $upstream_port %d;", indent, port)
	})

	proto := strings.ToLower(strings.TrimSpace(scheme))
	if proto == "" {
		proto = "http"
	}
	upstreamProtoRe := regexp.MustCompile(`(\s*)set\s+\$upstream_proto\s+([^;]+);`)
	config = upstreamProtoRe.ReplaceAllStringFunc(config, func(match string) string {
		submatch := upstreamProtoRe.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		indent := submatch[1]
		return fmt.Sprintf("%sset $upstream_proto %s;", indent, proto)
	})

	serverNameRe := regexp.MustCompile(`server_name\s+` + regexp.QuoteMeta(containerName) + `\.\*;`)
	config = serverNameRe.ReplaceAllString(config, "server_name "+domain+";")

	if proto == "http" {
		config = convertHTTPStoHTTP(config)
	}

	config = replaceLinuxserverIncludes(config)

	lines := strings.Split(config, "\n")
	var filteredLines []string
	inServerBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "server {") {
			inServerBlock = true
		}
		if !inServerBlock && strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "## Version") {
			continue
		}
		filteredLines = append(filteredLines, line)
	}
	config = strings.Join(filteredLines, "\n")
	config = strings.TrimLeft(config, "\n")

	return config, nil
}

func convertHTTPStoHTTP(config string) string {
	config = regexp.MustCompile(`listen\s+443\s+ssl;`).ReplaceAllString(config, "listen 80;")
	config = regexp.MustCompile(`listen\s+443\s+quic;`).ReplaceAllString(config, "# listen 443 quic;")
	config = regexp.MustCompile(`listen\s+\[::\]:443\s+ssl;`).ReplaceAllString(config, "listen [::]:80;")
	config = regexp.MustCompile(`listen\s+\[::\]:443\s+quic;`).ReplaceAllString(config, "# listen [::]:443 quic;")
	config = regexp.MustCompile(`(\s+)include\s+/config/nginx/ssl\.conf;`).ReplaceAllString(config, "$1# include /config/nginx/ssl.conf;")
	return config
}

func replaceLinuxserverIncludes(config string) string {
	proxyConfRe := regexp.MustCompile(`(\s+)include\s+/config/nginx/proxy\.conf;`)
	config = proxyConfRe.ReplaceAllStringFunc(config, func(match string) string {
		indent := proxyConfRe.FindStringSubmatch(match)[1]
		return indent + "proxy_http_version 1.1;\n" +
			indent + "proxy_set_header Host $host;\n" +
			indent + "proxy_set_header X-Real-IP $remote_addr;\n" +
			indent + "proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n" +
			indent + "proxy_set_header X-Forwarded-Proto $scheme;\n" +
			indent + "proxy_set_header Upgrade $http_upgrade;\n" +
			indent + "proxy_set_header Connection $connection_upgrade;"
	})

	config = regexp.MustCompile(`(\s+)include\s+/config/nginx/resolver\.conf;`).ReplaceAllString(config, "$1# include /config/nginx/resolver.conf;")

	commentOutIncludes := []string{
		"ldap-server.conf",
		"ldap-location.conf",
		"authelia-server.conf",
		"authelia-location.conf",
		"authentik-server.conf",
		"authentik-location.conf",
		"tinyauth-server.conf",
		"tinyauth-location.conf",
	}

	for _, inc := range commentOutIncludes {
		pattern := regexp.MustCompile(`(\s+)#?include\s+/config/nginx/` + regexp.QuoteMeta(inc) + `;`)
		config = pattern.ReplaceAllString(config, "$1# include /config/nginx/"+inc+";")
	}

	return config
}

func tryLinuxserverConfig(containerName, hostAddr string, port int, scheme, domain string) string {
	template, err := fetchLinuxserverConfig(containerName)
	if err != nil {
		return ""
	}

	config, err := replaceLinuxserverVars(template, containerName, hostAddr, port, scheme, domain)
	if err != nil {
		log.Printf("WARN nginx-gen: failed to replace vars in linuxserver config for %s: %v", containerName, err)
		return ""
	}

	log.Printf("nginx-gen: using linuxserver config for %s", containerName)
	return config
}
