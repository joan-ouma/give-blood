package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joan-ouma/give-blood/internal/config"
	"github.com/joan-ouma/give-blood/internal/db"
	"github.com/joan-ouma/give-blood/internal/handlers"
	"github.com/joan-ouma/give-blood/internal/httpserver"
	"github.com/joan-ouma/give-blood/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	database, err := db.Connect(cfg.MongoURI)
	if err != nil {
		log.Fatalf("database connect error: %v", err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			log.Printf("error closing database: %v", err)
		}
	}()

	mongoDB := database.Client.Database("blood_donation")

	userService := service.NewUserService(mongoDB)
	locationService := service.NewLocationService(mongoDB)
	driveService := service.NewDriveService(mongoDB)
	donationService := service.NewDonationService(mongoDB)
	leaderboardService := service.NewLeaderboardService(mongoDB)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := userService.EnsureIndexes(ctx); err != nil {
		log.Fatalf("database index setup error (user): %v", err)
	}
	if err := locationService.EnsureIndexes(ctx); err != nil {
		log.Fatalf("database index setup error (location): %v", err)
	}
	if err := driveService.EnsureIndexes(ctx); err != nil {
		log.Fatalf("database index setup error (drive): %v", err)
	}
	if err := donationService.EnsureIndexes(ctx); err != nil {
		log.Fatalf("database index setup error (donation): %v", err)
	}
	if err := leaderboardService.EnsureIndexes(ctx); err != nil {
		log.Fatalf("database index setup error (leaderboard): %v", err)
	}

	tokenSvc := service.NewTokenService(cfg.JWTSecret)
	limiter := service.NewRateLimiter()
	donationLimiter := service.NewRateLimiter()

	authHandler := handlers.NewAuthHandler(userService, tokenSvc, limiter)
	locationHandler := handlers.NewLocationHandler(locationService)
	driveHandler := handlers.NewDriveHandler(driveService)
	donationHandler := handlers.NewDonationHandler(donationService, donationLimiter)
	leaderboardHandler := handlers.NewLeaderboardHandler(leaderboardService, tokenSvc)

	authMiddleware := service.Middleware(tokenSvc)

	srv := httpserver.New(
		cfg.Port,
		cfg.AllowedOrigin,
		authMiddleware,
		authHandler,
		locationHandler,
		driveHandler,
		donationHandler,
		leaderboardHandler,
	)

	go func() {
		if err := srv.Start(); err != nil && err != httpserver.ErrServerClosed {
			log.Fatalf("server listen error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown error: %v", err)
	}
}
