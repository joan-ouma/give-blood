package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/joan-ouma/give-blood/internal/handlers"
)

type Server struct {
	server *http.Server
}

func New(port string, allowedOrigin string, authMiddleware func(http.Handler) http.Handler, authHandler *handlers.AuthHandler) *Server {
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

	mux.HandleFunc("/api/auth/register", authHandler.Register)
	mux.HandleFunc("/api/auth/login", authHandler.Login)
	mux.HandleFunc("/api/auth/refresh", authHandler.Refresh)

	mux.Handle("/api/auth/me", authMiddleware(http.HandlerFunc(authHandler.Me)))

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
