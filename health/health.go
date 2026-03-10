// Package health provides a standard /health endpoint for PolyON modules.
//
// K8s readiness/liveness probes should point to this handler.
package health

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// CheckFunc is a health check function. Return nil for healthy, error for unhealthy.
type CheckFunc func() error

// Checker manages health check functions.
type Checker struct {
	mu     sync.RWMutex
	checks map[string]CheckFunc
}

// New creates a new Checker.
func New() *Checker {
	return &Checker{checks: map[string]CheckFunc{}}
}

// Add registers a named health check.
func (c *Checker) Add(name string, fn CheckFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.checks[name] = fn
}

// Handler returns an http.Handler that responds with health status.
//
//	200 {"status":"ok","checks":{...}} — all checks pass
//	503 {"status":"error","checks":{...}} — one or more checks failed
func (c *Checker) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.mu.RLock()
		checks := make(map[string]CheckFunc, len(c.checks))
		for k, v := range c.checks {
			checks[k] = v
		}
		c.mu.RUnlock()

		results := map[string]string{}
		healthy := true
		for name, fn := range checks {
			if err := fn(); err != nil {
				results[name] = err.Error()
				healthy = false
			} else {
				results[name] = "ok"
			}
		}

		status := "ok"
		code := http.StatusOK
		if !healthy {
			status = "error"
			code = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(map[string]any{
			"status":    status,
			"checks":    results,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})
}

// Handler returns a simple health handler with no checks (always 200).
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":    "ok",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})
}
