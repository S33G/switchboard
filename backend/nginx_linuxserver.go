package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	linuxserverConfBaseURL = "https://raw.githubusercontent.com/linuxserver/reverse-proxy-confs/master/"
	linuxserverConfSuffix  = ".subdomain.conf.sample"
)

type linuxserverConfigCache struct {
	mu      sync.RWMutex
	configs map[string]string
	fetched map[string]time.Time
	ttl     time.Duration
}

var lsConfigCache = &linuxserverConfigCache{
	configs: make(map[string]string),
	fetched: make(map[string]time.Time),
	ttl:     1 * time.Hour,
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

	url := linuxserverConfBaseURL + containerName + linuxserverConfSuffix
	log.Printf("nginx-gen: fetching linuxserver config for %s from %s", containerName, url)

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

	config := string(body)

	lsConfigCache.mu.Lock()
	lsConfigCache.configs[containerName] = config
	lsConfigCache.fetched[containerName] = time.Now()
	lsConfigCache.mu.Unlock()

	return config, nil
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
