package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stwalsh4118/atlas/api/internal/config"
	"github.com/stwalsh4118/atlas/api/internal/database"
	"github.com/stwalsh4118/atlas/api/internal/handlers"
	"github.com/stwalsh4118/atlas/api/internal/logger"
	"github.com/stwalsh4118/atlas/api/internal/middleware"
	"github.com/stwalsh4118/atlas/api/internal/repository"
	"github.com/stwalsh4118/atlas/api/internal/services"
)

const (
	shutdownTimeout = 30 * time.Second
)

func main() {
	// Load configuration from environment variables
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize structured logger
	log := logger.New(cfg.Server.Env)
	log.Info("Starting Atlas API", map[string]interface{}{
		"version":     "0.1.0",
		"environment": cfg.Server.Env,
		"port":        cfg.Server.Port,
	})

	// Create database connection pool
	ctx := context.Background()
	db, err := database.NewPostgresPool(ctx, cfg.Database)
	if err != nil {
		log.Fatal("Failed to connect to database", err, map[string]interface{}{
			"host": cfg.Database.Host,
			"port": cfg.Database.Port,
			"name": cfg.Database.Name,
		})
	}
	defer db.Close()

	log.Info("Database connection established", map[string]interface{}{
		"host":     cfg.Database.Host,
		"port":     cfg.Database.Port,
		"database": cfg.Database.Name,
		"pool_min": cfg.Database.PoolMin,
		"pool_max": cfg.Database.PoolMax,
	})

	// Setup Gin router
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()

	// Add middleware in order: RequestID -> Logger -> Recovery -> CORS
	router.Use(middleware.RequestID())
	router.Use(middleware.Logger(log))
	router.Use(middleware.Recovery(log))
	router.Use(middleware.CORS(cfg.CORS.Origins))

	// Register health check routes
	healthHandler := handlers.NewHealthHandler(db, cfg.Server.Env)
	router.GET("/health", healthHandler.Health)
	router.GET("/health/ready", healthHandler.Ready)
	router.GET("/api/v1/info", healthHandler.Info)

	// Initialize repository and service layers
	parcelRepo := repository.NewParcelRepository(db)
	parcelService := services.NewParcelService(parcelRepo, log)

	// Initialize handlers
	parcelHandler := handlers.NewParcelHandler(parcelService)

	// Register API v1 routes
	v1 := router.Group("/api/v1")
	{
		parcels := v1.Group("/parcels")
		{
			parcels.GET("/at-point", parcelHandler.AtPoint)
		}
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Server.Port),
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		log.Info("Server listening", map[string]interface{}{
			"port": cfg.Server.Port,
			"addr": srv.Addr,
		})
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed to start", err, nil)
		}
	}()

	// Wait for interrupt signal (SIGINT or SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown
	log.Info("Shutting down server...", nil)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("Server forced to shutdown", err, map[string]interface{}{
			"timeout": shutdownTimeout.String(),
		})
	}

	log.Info("Server exited", nil)
}
