package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v2"
)

func startConfigWatcher(ctx context.Context, configPath string, api *API, hub *Hub) error {
	if configPath == "" || configPath == "DOCKER_HOSTS env" {
		log.Println("Config watcher: skipping (no file path)")
		return nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	configDir := filepath.Dir(configPath)
	configFile := filepath.Base(configPath)

	if err := watcher.Add(configDir); err != nil {
		watcher.Close()
		return fmt.Errorf("failed to watch directory %s: %w", configDir, err)
	}

	log.Printf("Config watcher: watching %s for changes to %s", configDir, configFile)

	go func() {
		defer watcher.Close()

		var debounceTimer *time.Timer
		var pending bool

		for {
			select {
			case <-ctx.Done():
				return

			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if filepath.Base(event.Name) != configFile {
					continue
				}

				if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
					log.Printf("Config watcher: %s changed (%v)", configFile, event.Op)

					if debounceTimer != nil {
						debounceTimer.Stop()
					}
					pending = true
					debounceTimer = time.AfterFunc(500*time.Millisecond, func() {
						if !pending {
							return
						}
						pending = false
						if err := reloadConfig(configPath, api, hub); err != nil {
							log.Printf("Config reload error: %v", err)
						}
					})
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("Config watcher error: %v", err)
			}
		}
	}()

	return nil
}

func reloadConfig(configPath string, api *API, hub *Hub) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var newConfig Config
	if err := yaml.Unmarshal(data, &newConfig); err != nil {
		return fmt.Errorf("failed to parse config YAML: %w", err)
	}

	newConfig.ProxyMappings = ensureMapping(newConfig.ProxyMappings)
	newConfig.HostAddresses = ensureMapping(newConfig.HostAddresses)
	newConfig.Hosts = ensureHostNames(newConfig.Hosts)
	newConfig.ParsedMappings = parseProxyMappings(newConfig.ProxyMappings)

	if len(newConfig.Hosts) == 0 {
		return fmt.Errorf("validation failed: no hosts configured")
	}

	api.configMutex.Lock()
	oldConfig := api.config
	api.config = newConfig
	api.computeProxyRoutes()
	api.configMutex.Unlock()

	api.notifyConfigChanged()

	hostsChanged := configHostsChanged(oldConfig, newConfig)
	hub.BroadcastConfigUpdate(newConfig, hostsChanged)

	log.Printf("Config reloaded: %d hosts, %d proxy mappings, base_domain=%s",
		len(newConfig.Hosts), len(newConfig.ProxyMappings), newConfig.Defaults.BaseDomain)

	return nil
}

func configHostsChanged(old, new Config) bool {
	if len(old.Hosts) != len(new.Hosts) {
		return true
	}

	oldMap := make(map[string]string)
	for _, h := range old.Hosts {
		oldMap[h.Name] = h.Endpoint
	}

	for _, h := range new.Hosts {
		if oldMap[h.Name] != h.Endpoint {
			return true
		}
	}

	return false
}
