package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/harness-org/backend/internal/domain/capability"
	"github.com/harness-org/backend/internal/domain/identity"
	"github.com/harness-org/backend/internal/domain/layer"
	"github.com/harness-org/backend/internal/domain/organization"
	"github.com/harness-org/backend/internal/gateway"
	"github.com/harness-org/backend/internal/pkg/config"
	"github.com/harness-org/backend/internal/pkg/database"
	"github.com/harness-org/backend/internal/pkg/server"
)

func main() {
	cfg := config.Load()

	connCtx, connCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer connCancel()

	db, err := database.Connect(connCtx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()

	if err := database.RunMigrations(context.Background(), db, cfg.MigrationsPath); err != nil {
		log.Fatalf("migrations failed: %v", err)
	}

	identRepo := identity.NewRepository(db)
	identSvc := identity.NewService(identRepo, cfg.JWTSecret)
	identHandler := identity.NewHandler(identSvc)

	orgRepo := organization.NewRepository(db)
	orgSvc := organization.NewService(orgRepo)
	orgHandler := organization.NewHandler(orgSvc)

	layerRepo := layer.NewRepository(db)
	layerClassifier := layer.NewClassifierService(layerRepo)
	layerHandler := layer.NewHandler(layerClassifier)

	capRepo := capability.NewRepository(db)
	capRouter := capability.NewRouter(capRepo)
	capHandler := capability.NewHandler(capRepo, capRouter)

	router := server.NewRouter(cfg.CorsOrigins)
	gateway.RegisterRoutes(router, &gateway.Dependencies{
		IdentityHandler:     identHandler,
		OrganizationHandler: orgHandler,
		LayerHandler:        layerHandler,
		CapabilityHandler:   capHandler,
	})

	srv := server.New(router, cfg.ServerPort)
	go func() {
		log.Printf("server starting on :%d", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	srv.Shutdown(shutdownCtx)
}
