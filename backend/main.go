package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	// Create API and initialize proxied ports BEFORE first sync
	// so that convertPorts() can correctly mark proxied ports
	mux := http.NewServeMux()
	api := NewAPI(store, nil, config) // hub is nil for now, will use it later
	api.Register(mux)
	mux.Handle("/metrics", promhttp.Handler())
	setProxiedPorts(api.config.ProxyRoutes)

	// Now do the first sync with proxied ports initialized
	_ = syncAllHosts(ctx, manager, store, warns)

	hub := NewHub()
	go hub.Run()
	go startDiffBroadcaster(ctx, store, hub)

	// Update API's hub reference after creating it
	api.hub = hub

	go startHealthMonitor(ctx, manager, warns)
	go startCacheCleaner(ctx, manager)
	go startEventLoops(ctx, manager, store, hub, warns)

	log.Println("main: about to start nginx generator loop")
	go startNginxGeneratorLoop(ctx, store, api, warns)
	log.Println("main: nginx generator loop goroutine started")

	if err := startConfigWatcher(ctx, configSource, api, hub); err != nil {
		log.Printf("WARN failed to start config watcher: %v", err)
	}

	handler := http.Handler(mux)
	if envBool("ACCESS_LOGS") {
		handler = withAccessLogs(log.Default(), handler)
	}

	debugMux := http.NewServeMux()
	debugMux.Handle("/metrics", promhttp.Handler())
	debugServer := &http.Server{Addr: ":6060", Handler: debugMux}
	go func() {
		log.Println("Debug server (pprof + metrics) listening on :6060")
		if err := debugServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Debug server error: %v", err)
		}
	}()

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

func startCacheCleaner(ctx context.Context, manager *DockerClientManager) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			manager.cache.Clean()
		}
	}
}

func startHealthMonitor(ctx context.Context, manager *DockerClientManager, warns *warnLimiter) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, hostName := range manager.HostNames() {
				go func(host string) {
					pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
					defer cancel()

					if err := manager.Ping(pingCtx, host); err != nil {
						warns.Warnf("health|"+host, 2*time.Minute, "WARN health check failed for %s: %v, attempting reconnect", host, err)
						if reconnectErr := manager.ReconnectHost(ctx, host); reconnectErr != nil {
							warns.Warnf("reconnect|"+host, 2*time.Minute, "WARN reconnect failed for %s: %v", host, reconnectErr)
						} else {
							log.Printf("Successfully reconnected to host %s", host)
						}
					}
				}(hostName)
			}
		}
	}
}

func startDiffBroadcaster(ctx context.Context, store *StateStore, hub *Hub) {
	for {
		select {
		case <-ctx.Done():
			return
		case diff := <-store.Diffs():
			hub.BroadcastDiff(diff)
		}
	}
}

func startEventLoops(ctx context.Context, manager *DockerClientManager, store *StateStore, hub *Hub, warns *warnLimiter) {
	for _, host := range manager.HostNames() {
		hostName := host
		go func() {
			log.Printf("EVENT_LOOP_START: host=%s", hostName)
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				log.Printf("EVENT_LOOP_CONNECTING: host=%s", hostName)
				result, err := manager.Events(ctx, hostName)
				if err != nil {
					warns.Warnf("events|"+hostName, 30*time.Second, "WARN docker events %s: %v", hostName, err)
					time.Sleep(2 * time.Second)
					continue
				}
				log.Printf("EVENT_LOOP_CONNECTED: host=%s", hostName)
				for {
					select {
					case <-ctx.Done():
						return
					case event := <-result.Messages:
						if event.Actor.ID == "" {
							continue
						}

						dockerEventsTotal.WithLabelValues(hostName, string(event.Action)).Inc()

						log.Printf("EVENT: host=%s action=%s container=%s", hostName, event.Action, event.Actor.ID[:12])
						switch event.Action {
						case "start", "die", "stop", "pause", "unpause", "kill":
							manager.cache.Invalidate(event.Actor.ID)
							log.Printf("SYNC_SINGLE: host=%s container=%s", hostName, event.Actor.ID[:12])
							if syncErr := syncSingleContainer(ctx, manager, store, hub, hostName, event.Actor.ID); syncErr != nil {
								warns.Warnf("sync-single|"+hostName, 10*time.Second, "WARN sync single container %s/%s: %v, falling back to full sync", hostName, event.Actor.ID[:12], syncErr)
								if fullSyncErr := syncHost(ctx, manager, store, hub, hostName); fullSyncErr != nil {
									warns.Warnf("sync|"+hostName, 30*time.Second, "WARN docker sync %s: %v", hostName, fullSyncErr)
								}
							}
						case "create", "destroy", "rename":
							manager.cache.Invalidate(event.Actor.ID)
							log.Printf("SYNC_HOST: host=%s reason=%s", hostName, event.Action)
							if syncErr := syncHost(ctx, manager, store, hub, hostName); syncErr != nil {
								warns.Warnf("sync|"+hostName, 30*time.Second, "WARN docker sync %s: %v", hostName, syncErr)
							}
						default:
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

func syncSingleContainer(ctx context.Context, manager *DockerClientManager, store *StateStore, _ *Hub, hostName string, containerID string) error {
	start := time.Now()
	defer func() {
		syncDuration.WithLabelValues(hostName).Observe(time.Since(start).Seconds())
	}()

	summary, err := manager.InspectContainer(ctx, hostName, containerID)
	if err != nil {
		return err
	}

	store.UpdateSingleContainer(hostName, *summary)
	return nil
}

func syncHost(ctx context.Context, manager *DockerClientManager, store *StateStore, _ *Hub, hostName string) error {
	start := time.Now()
	defer func() {
		syncDuration.WithLabelValues(hostName).Observe(time.Since(start).Seconds())
	}()

	containers, err := manager.ListContainers(ctx, hostName)
	if err != nil {
		return err
	}
	store.UpdateFromHost(hostName, containers)

	stateCounts := make(map[string]int)
	for _, c := range containers {
		stateCounts[string(c.State)]++
	}
	for state, count := range stateCounts {
		containerCount.WithLabelValues(hostName, state).Set(float64(count))
	}

	return nil
}
