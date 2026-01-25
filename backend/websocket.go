package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type Hub struct {
	clients    map[*websocket.Conn]struct{}
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	broadcast  chan []byte
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]struct{}),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		broadcast:  make(chan []byte, 64),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case conn := <-h.register:
			h.clients[conn] = struct{}{}
			wsClientsConnected.Inc()
		case conn := <-h.unregister:
			delete(h.clients, conn)
			_ = conn.Close()
			wsClientsConnected.Dec()
		case message := <-h.broadcast:
			for conn := range h.clients {
				_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
				if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
					delete(h.clients, conn)
					_ = conn.Close()
					wsClientsConnected.Dec()
				}
			}
		}
	}
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	h.register <- conn
	go func() {
		defer func() { h.unregister <- conn }()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
}

func (h *Hub) BroadcastSnapshot(snapshot []Container) {
	payload := map[string]any{
		"type":    "containers_snapshot",
		"payload": snapshot,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	select {
	case h.broadcast <- data:
	default:
	}
}

func (h *Hub) BroadcastDiff(diff ContainerDiff) {
	payload := map[string]any{
		"type":    "containers_diff",
		"payload": diff,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	select {
	case h.broadcast <- data:
	default:
	}
}

func (h *Hub) BroadcastConfigUpdate(config Config, hostsChanged bool) {
	payload := map[string]any{
		"type": "config_updated",
		"payload": map[string]any{
			"config":        config,
			"hosts_changed": hostsChanged,
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	select {
	case h.broadcast <- data:
	default:
	}
}
