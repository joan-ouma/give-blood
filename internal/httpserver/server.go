package httpserver

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/joan-ouma/give-blood/internal/handlers"
)

type Server struct {
	server *http.Server
}

func New(
	port string,
	allowedOrigin string,
	authMiddleware func(http.Handler) http.Handler,
	authHandler *handlers.AuthHandler,
	locHandler *handlers.LocationHandler,
	driveHandler *handlers.DriveHandler,
) *Server {
	mux := http.NewServeMux()

	cors := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	// Auth Endpoints
	mux.HandleFunc("/api/auth/register", authHandler.Register)
	mux.HandleFunc("/api/auth/login", authHandler.Login)
	mux.HandleFunc("/api/auth/refresh", authHandler.Refresh)
	mux.Handle("/api/auth/me", authMiddleware(http.HandlerFunc(authHandler.Me)))

	// Locations & Drives routing with trailing slash parsing
	mux.HandleFunc("/api/locations", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			locHandler.List(w, r)
			return
		}
		if r.Method == http.MethodPost {
			authMiddleware(http.HandlerFunc(locHandler.Create)).ServeHTTP(w, r)
			return
		}
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/api/locations/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 4 || parts[3] == "" {
			http.Error(w, `{"error":"missing location id"}`, http.StatusBadRequest)
			return
		}

		if r.Method == http.MethodGet {
			locHandler.GetByID(w, r)
			return
		}
		if r.Method == http.MethodPut {
			authMiddleware(http.HandlerFunc(locHandler.Update)).ServeHTTP(w, r)
			return
		}
		if r.Method == http.MethodDelete {
			authMiddleware(http.HandlerFunc(locHandler.Delete)).ServeHTTP(w, r)
			return
		}
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/api/drives", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			driveHandler.List(w, r)
			return
		}
		if r.Method == http.MethodPost {
			authMiddleware(http.HandlerFunc(driveHandler.Create)).ServeHTTP(w, r)
			return
		}
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/api/drives/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 4 || parts[3] == "" {
			http.Error(w, `{"error":"missing drive id"}`, http.StatusBadRequest)
			return
		}

		if r.Method == http.MethodGet {
			driveHandler.GetByID(w, r)
			return
		}
		if r.Method == http.MethodPut {
			authMiddleware(http.HandlerFunc(driveHandler.Update)).ServeHTTP(w, r)
			return
		}
		if r.Method == http.MethodDelete {
			authMiddleware(http.HandlerFunc(driveHandler.Delete)).ServeHTTP(w, r)
			return
		}
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	})

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      cors(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return &Server{server: srv}
}

var ErrServerClosed = http.ErrServerClosed

func (s *Server) Start() error {
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
