package httpserver

import (
	"fmt"
	"net/http"
	"time"
)

const (
	defaultReadTimeout  = 5 * time.Second
	defaultWriteTimeout = 10 * time.Second
	defaultIdleTimeout  = 120 * time.Second
)

// NewServer constructs a *http.Server with sensible timeout defaults.
// Graceful shutdown is handled in the composition root (cmd/api/main.go)
// by calling srv.Shutdown(ctx) with a 15-second context.
func NewServer(port int, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      handler,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
		IdleTimeout:  defaultIdleTimeout,
	}
}
