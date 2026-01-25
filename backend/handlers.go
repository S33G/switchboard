package main

import (
	"encoding/json"
	"net/http"
)

type API struct {
	store  *StateStore
	hub    *Hub
	config Config
}

func NewAPI(store *StateStore, hub *Hub, config Config) *API {
	api := &API{store: store, hub: hub, config: config}
	api.computeProxyRoutes()
	return api
}

func (api *API) computeProxyRoutes() {
	proxyRoutes := make(map[string]map[string][]string)

	for domain, containerName := range api.config.ProxyMappings {
		scheme := api.config.Defaults.Scheme
		if scheme == "" {
			scheme = "http"
		}
		url := scheme + "://" + domain

		if proxyRoutes[containerName] == nil {
			proxyRoutes[containerName] = make(map[string][]string)
		}
		proxyRoutes[containerName]["domains"] = append(proxyRoutes[containerName]["domains"], url)
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
func (api *API) handleContainers(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(api.store.Snapshot())
}

// @Summary Get configuration
// @Produce json
// @Success 200 {object} Config
// @Router /api/config [get]
func (api *API) handleConfig(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(api.config)
}

// @Summary WebSocket stream
// @Router /ws [get]
func (api *API) handleWebsocket(w http.ResponseWriter, r *http.Request) {
	api.hub.ServeWS(w, r)
}
