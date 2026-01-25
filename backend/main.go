package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

type warnLimiter struct {
	mu   sync.Mutex
	last map[string]time.Time
}

func newWarnLimiter() *warnLimiter {
	return &warnLimiter{last: make(map[string]time.Time)}
}

func (w *warnLimiter) Warnf(key string, every time.Duration, format string, args ...any) {
	if w == nil {
		log.Printf(format, args...)
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if t, ok := w.last[key]; ok && time.Since(t) < every {
		return
	}
	w.last[key] = time.Now()
	log.Printf(format, args...)
}

func main() {
	initConfigPath := flag.String("init-config", "", "write a YAML config file to the given path (use '-' for stdout) and exit")
	flag.Parse()

	if strings.TrimSpace(*initConfigPath) != "" {
		cfg, _, err := loadConfigFromEnv()
		if err != nil {
			log.Fatal(err)
		}

		path := strings.TrimSpace(*initConfigPath)
		if path == "-" {
			if err := writeConfigYAML(os.Stdout, cfg); err != nil {
				log.Fatal(err)
			}
			return
		}
		f, err := os.Create(path)
		if err != nil {
			log.Fatal(err)
		}
		defer func() { _ = f.Close() }()
		if err := writeConfigYAML(f, cfg); err != nil {
			log.Fatal(err)
		}
		return
	}

	config, configSource, err := loadConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("=== Configuration loaded from: %s ===", configSource)
	log.Printf("Configured hosts (%d):", len(config.Hosts))
	for _, host := range config.Hosts {
		log.Printf("  - %s: %s", host.Name, host.Endpoint)
	}
	log.Printf("Base domain: %s", config.Defaults.BaseDomain)
	log.Printf("Scheme: %s", config.Defaults.Scheme)

	if len(config.ProxyMappings) > 0 {
		log.Println("Proxy Routes:")
		for domain, target := range config.ProxyMappings {
			log.Printf("  %s -> %s", domain, target)
		}
	} else {
		log.Println("Proxy Routes: none")
	}
	log.Println("==========================================")

	if configData, err := marshalConfigYAML(config); err == nil {
		log.Println("Full configuration:")
		log.Print(string(configData))
		log.Println("==========================================")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	warns := newWarnLimiter()

	manager := NewDockerClientManager(config.Hosts)
	if err := manager.Connect(ctx); err != nil {
		log.Printf("WARN %v", err)
	}

	store := NewStateStore()
	_ = syncAllHosts(ctx, manager, store, warns)

	hub := NewHub()
	go hub.Run()

	go startEventLoops(ctx, manager, store, hub, warns)
	log.Println("main: about to start nginx generator loop")
	go startNginxGeneratorLoop(ctx, store, config, warns)
	log.Println("main: nginx generator loop goroutine started")

	mux := http.NewServeMux()
	api := NewAPI(store, hub, config)
	api.Register(mux)

	handler := http.Handler(mux)
	if envBool("ACCESS_LOGS") {
		handler = withAccessLogs(log.Default(), handler)
	}

	port := strings.TrimSpace(os.Getenv("API_PORT"))
	if port == "" {
		port = "8069"
	}
	server := &http.Server{Addr: ":" + port, Handler: handler}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = server.Shutdown(shutdownCtx)
}

func startEventLoops(ctx context.Context, manager *DockerClientManager, store *StateStore, hub *Hub, warns *warnLimiter) {
	for _, host := range manager.HostNames() {
		hostName := host
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				result, err := manager.Events(ctx, hostName)
				if err != nil {
					warns.Warnf("events|"+hostName, 30*time.Second, "WARN docker events %s: %v", hostName, err)
					time.Sleep(2 * time.Second)
					continue
				}
				for {
					select {
					case <-ctx.Done():
						return
					case <-result.Messages:
						if syncErr := syncHost(ctx, manager, store, hub, hostName); syncErr != nil {
							warns.Warnf("sync|"+hostName, 30*time.Second, "WARN docker sync %s: %v", hostName, syncErr)
							continue
						}
					case <-result.Err:
						warns.Warnf("events-stream|"+hostName, 30*time.Second, "WARN docker events stream %s: disconnected", hostName)
						time.Sleep(2 * time.Second)
						goto restart
					}
				}
			restart:
			}
		}()
	}
}

func syncAllHosts(ctx context.Context, manager *DockerClientManager, store *StateStore, warns *warnLimiter) error {
	for _, host := range manager.HostNames() {
		if err := syncHost(ctx, manager, store, nil, host); err != nil {
			warns.Warnf("sync-init|"+host, 30*time.Second, "WARN docker initial sync %s: %v", host, err)
			continue
		}
	}
	return nil
}

func syncHost(ctx context.Context, manager *DockerClientManager, store *StateStore, hub *Hub, hostName string) error {
	containers, err := manager.ListContainers(ctx, hostName)
	if err != nil {
		return err
	}
	store.UpdateFromHost(hostName, containers)
	if hub != nil {
		hub.BroadcastSnapshot(store.Snapshot())
	}
	return nil
}
