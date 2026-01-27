package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/felixge/httpsnoop"
)

func envBool(name string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	if v == "" {
		return false
	}
	switch v {
	case "1", "true", "t", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func envBoolDefault(name string, defaultValue bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	if v == "" {
		return defaultValue
	}
	switch v {
	case "1", "true", "t", "yes", "y", "on":
		return true
	case "0", "false", "f", "no", "n", "off":
		return false
	default:
		return defaultValue
	}
}

func withAccessLogs(logger *log.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		m := httpsnoop.CaptureMetrics(next, w, r)
		logger.Printf(
			"%s %s %d %d %s",
			r.Method,
			r.URL.RequestURI(),
			m.Code,
			m.Written,
			time.Since(start).Truncate(time.Microsecond),
		)
	})
}
