package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/bongcloud/chess/internal/cache"
	"github.com/bongcloud/chess/internal/config"
	"github.com/bongcloud/chess/internal/database"
	"github.com/bongcloud/chess/internal/logger"
	"github.com/bongcloud/chess/internal/server"
	"github.com/bongcloud/chess/internal/ws"
)

func main() {
	//  1. Config
	cfg, err := config.Load()
	if err != nil {
		// Logger isn't ready yet — write to stderr and exit
		_, _ = os.Stderr.WriteString("FATAL: failed to load config: " + err.Error() + "\n")
		os.Exit(1)
	}

	//  2. Logger
	log := logger.Must(cfg.Log.Level, cfg.Log.Encoding, cfg.App.Name, cfg.App.Version)
	defer func() { _ = log.Sync() }()

	log.Info("BongCloud Chess starting up",
		zap.String("env", cfg.App.Env),
		zap.String("version", cfg.App.Version),
	)

	//  3. MongoDB
	mongo, err := database.NewMongoDB(cfg.Mongo, log)
	if err != nil {
		log.Fatal("failed to connect to MongoDB", zap.Error(err))
	}

	//  4. Redis
	redis, err := cache.NewRedisClient(cfg.Redis, log)
	if err != nil {
		log.Fatal("failed to connect to Redis", zap.Error(err))
	}

	//  5. WebSocket upgrader
	upgrader := ws.NewUpgrader(cfg.WebSocket, log)

	//  6. HTTP Server
	srv := server.New(server.Dependencies{
		Config:   cfg,
		Log:      log,
		Mongo:    mongo,
		Redis:    redis,
		Upgrader: upgrader,
	})

	//  7. Start + graceful shutdown
	// Run the server in a goroutine so we can listen for OS signals concurrently.
	serverErr := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		log.Error("server error", zap.Error(err))
	case sig := <-quit:
		log.Info("shutdown signal received", zap.String("signal", sig.String()))
	}

	// Give in-flight requests time to finish
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	log.Info("draining in-flight requests…")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown error", zap.Error(err))
	}

	// Close infrastructure connections
	if err := mongo.Disconnect(shutdownCtx); err != nil {
		log.Error("MongoDB disconnect error", zap.Error(err))
	}
	if err := redis.Close(); err != nil {
		log.Error("Redis close error", zap.Error(err))
	}

	log.Info("BongCloud Chess shut down cleanly")
}
