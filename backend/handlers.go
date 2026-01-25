package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"sync"
)

type API struct {
	store         *StateStore
	hub           *Hub
	config        Config
	configMutex   sync.RWMutex
	configChanged chan struct{}
}

func NewAPI(store *StateStore, hub *Hub, config Config) *API {
	api := &API{
		store:         store,
		hub:           hub,
		config:        config,
		configChanged: make(chan struct{}, 1),
	}
	api.computeProxyRoutes()
	return api
}

func (api *API) ConfigChanged() <-chan struct{} {
	return api.configChanged
}

func (api *API) notifyConfigChanged() {
	select {
	case api.configChanged <- struct{}{}:
	default:
	}
}

func (api *API) addProxyRoute(routes map[string]map[string][]string, key string, url string) {
	if routes[key] == nil {
		routes[key] = make(map[string][]string)
	}
	routes[key]["domains"] = append(routes[key]["domains"], url)
}

func (api *API) computeProxyRoutes() {
	proxyRoutes := make(map[string]map[string][]string)

	for domain, targetStr := range api.config.ProxyMappings {
		scheme := api.config.Defaults.Scheme
		if scheme == "" {
			scheme = "http"
		}
		url := scheme + "://" + domain

		api.addProxyRoute(proxyRoutes, targetStr, url)

		if target, ok := api.config.ParsedMappings[domain]; ok {
			containerKey := target.Host + "/" + target.Container
			api.addProxyRoute(proxyRoutes, containerKey, url)
		}
	}

	api.config.ProxyRoutes = proxyRoutes
}

func (api *API) Register(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", api.handleHealthz)
	mux.HandleFunc("/api/containers", api.handleContainers)
	mux.HandleFunc("/api/config", api.handleConfig)
	mux.HandleFunc("/ws", api.handleWebsocket)
	mux.Handle("/", http.FileServer(http.Dir("ui")))
}

// @Summary Health check
// @Success 200
// @Router /healthz [get]
func (api *API) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// @Summary List containers
// @Produce json
// @Success 200 {array} Container
// @Router /api/containers [get]
func (api *API) handleContainers(w http.ResponseWriter, r *http.Request) {
	snapshot := api.store.Snapshot()

	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 0
	offset := 0

	if limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			limit = val
		}
	}

	if offsetStr != "" {
		if val, err := strconv.Atoi(offsetStr); err == nil && val >= 0 {
			offset = val
		}
	}

	total := len(snapshot)

	if limit > 0 {
		if offset >= total {
			offset = 0
		}
		end := offset + limit
		if end > total {
			end = total
		}

		response := map[string]any{
			"containers": snapshot[offset:end],
			"total":      total,
			"limit":      limit,
			"offset":     offset,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(snapshot)
}

// @Summary Get or update configuration
// @Produce json
// @Accept json
// @Success 200 {object} Config
// @Router /api/config [get]
// @Router /api/config [put]
func (api *API) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		api.configMutex.RLock()
		config := api.config
		api.configMutex.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(config)

	case "PUT":
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization required", http.StatusUnauthorized)
			return
		}

		apiKey := os.Getenv("API_KEY")
		if apiKey != "" && authHeader != "Bearer "+apiKey {
			http.Error(w, "Invalid API key", http.StatusUnauthorized)
			return
		}

		var newConfig Config
		if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if len(newConfig.Hosts) == 0 {
			http.Error(w, "At least one host must be configured", http.StatusBadRequest)
			return
		}

		api.configMutex.Lock()
		api.config = newConfig
		api.computeProxyRoutes()
		api.configMutex.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(newConfig)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// @Summary WebSocket stream
// @Router /ws [get]
func (api *API) handleWebsocket(w http.ResponseWriter, r *http.Request) {
	api.hub.ServeWS(w, r)
}
