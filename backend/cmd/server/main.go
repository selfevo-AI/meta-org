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
	"github.com/harness-org/backend/internal/domain/dashboard"
	"github.com/harness-org/backend/internal/domain/evolution"
	"github.com/harness-org/backend/internal/domain/governance"
	"github.com/harness-org/backend/internal/domain/identity"
	"github.com/harness-org/backend/internal/domain/layer"
	"github.com/harness-org/backend/internal/domain/observability"
	"github.com/harness-org/backend/internal/domain/organization"
	"github.com/harness-org/backend/internal/domain/project"
	"github.com/harness-org/backend/internal/domain/verification"
	"github.com/harness-org/backend/internal/domain/workflow"
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

	govRepo := governance.NewRepository(db)
	govSvc := governance.NewService(govRepo)
	govHandler := governance.NewHandler(govSvc)

	evoRepo := evolution.NewRepository(db)
	evoSvc := evolution.NewService(evoRepo)
	evoHandler := evolution.NewHandler(evoSvc)

	orgRepo := organization.NewRepository(db)
	orgSvc := organization.NewService(
		orgRepo,
		organization.WithGovernanceService(govSvc),
		organization.WithEvolutionService(evoSvc),
	)
	orgHandler := organization.NewHandler(orgSvc)

	layerRepo := layer.NewRepository(db)
	layerClassifier := layer.NewClassifierService(layerRepo)
	layerHandler := layer.NewHandler(layerClassifier)

	capRepo := capability.NewRepository(db)
	capRouter := capability.NewRouter(capRepo)
	capHandler := capability.NewHandler(capRepo, capRouter, evoSvc)

	dashRepo := dashboard.NewRepository(db)
	dashSvc := dashboard.NewService(dashRepo)
	dashHandler := dashboard.NewHandler(dashSvc)

	wfRepo := workflow.NewRepository(db)
	wfSvc := workflow.NewService(wfRepo)
	wfHandler := workflow.NewHandler(wfSvc)

	projectRepo := project.NewRepository(db)
	projectSvc := project.NewService(
		projectRepo,
		project.WithGovernanceService(govSvc),
		project.WithEvolutionService(evoSvc),
		project.WithOrganizationService(orgSvc),
		project.WithWorkflowService(wfSvc),
	)
	projectHandler := project.NewHandler(projectSvc)

	verRepo := verification.NewRepository(db)
	verSvc := verification.NewService(verRepo)
	verHandler := verification.NewHandler(verSvc)

	obsRepo := observability.NewRepository(db)
	obsSvc := observability.NewService(obsRepo)
	obsHandler := observability.NewHandler(obsSvc)

	router := server.NewRouter(cfg.CorsOrigins)
	gateway.RegisterRoutes(router, &gateway.Dependencies{
		JWTSecret:            cfg.JWTSecret,
		IdentityHandler:      identHandler,
		OrganizationHandler:  orgHandler,
		LayerHandler:         layerHandler,
		CapabilityHandler:    capHandler,
		DashboardHandler:     dashHandler,
		WorkflowHandler:      wfHandler,
		ProjectHandler:       projectHandler,
		ObservabilityHandler: obsHandler,
		VerificationHandler:  verHandler,
		GovernanceHandler:    govHandler,
		EvolutionHandler:     evoHandler,
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
