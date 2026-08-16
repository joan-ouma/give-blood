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
	donationHandler *handlers.DonationHandler,
	leaderboardHandler *handlers.LeaderboardHandler,
) *Server {
	mux := http.NewServeMux()

	cors := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "http://localhost:5173" || origin == "http://localhost:5174" || origin == allowedOrigin {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			} else {
				w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			}
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			// Security Headers
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; sandbox")

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

	// Locations
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

	// Drives
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

	// Donations routing
	mux.HandleFunc("/api/donations", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			authMiddleware(http.HandlerFunc(donationHandler.Create)).ServeHTTP(w, r)
			return
		}
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/api/donations/mine", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			authMiddleware(http.HandlerFunc(donationHandler.ListMine)).ServeHTTP(w, r)
			return
		}
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/api/agency/donations/pending", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			authMiddleware(http.HandlerFunc(donationHandler.ListPending)).ServeHTTP(w, r)
			return
		}
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/api/donations/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		// Expecting /api/donations/:id/verify or /api/donations/:id/reject
		if len(parts) < 5 || parts[2] == "" {
			http.Error(w, `{"error":"invalid resource path"}`, http.StatusBadRequest)
			return
		}

		action := parts[4]
		if r.Method == http.MethodPost {
			if action == "accept" {
				authMiddleware(http.HandlerFunc(donationHandler.Accept)).ServeHTTP(w, r)
				return
			}
			if action == "verify" {
				authMiddleware(http.HandlerFunc(donationHandler.Verify)).ServeHTTP(w, r)
				return
			}
			if action == "reject" {
				authMiddleware(http.HandlerFunc(donationHandler.Reject)).ServeHTTP(w, r)
				return
			}
		}
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	})

	// Eligibility & Leaderboard
	mux.HandleFunc("/api/donors/me/eligibility", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			authMiddleware(http.HandlerFunc(leaderboardHandler.GetEligibility)).ServeHTTP(w, r)
			return
		}
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/api/leaderboard", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			leaderboardHandler.GetLeaderboard(w, r)
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
