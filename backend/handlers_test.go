package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/moby/moby/api/types/container"
)

func TestHandlers(t *testing.T) {
	store := NewStateStore()
	store.UpdateFromHost("host-1", []container.Summary{
		{
			ID:     "abc",
			Names:  []string{"/web"},
			Image:  "repo/web:latest",
			State:  container.StateRunning,
			Status: "Up",
			Labels: map[string]string{"app": "web"},
		},
	})

	config := Config{Hosts: []Host{{Name: "host-1", Endpoint: "unix:///var/run/docker.sock"}}}
	hub := NewHub()
	api := NewAPI(store, hub, config)

	mux := http.NewServeMux()
	api.Register(mux)

	cases := []struct {
		path       string
		statusCode int
	}{
		{path: "/healthz", statusCode: http.StatusOK},
		{path: "/api/config", statusCode: http.StatusOK},
		{path: "/api/containers", statusCode: http.StatusOK},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, req)
		if recorder.Code != tc.statusCode {
			t.Fatalf("%s expected %d got %d", tc.path, tc.statusCode, recorder.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/containers", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, req)

	var payload []Container
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode containers: %v", err)
	}
	if len(payload) != 1 || payload[0].ID != "abc" {
		t.Fatalf("unexpected container payload: %+v", payload)
	}
}
