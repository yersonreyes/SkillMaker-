// cmd/api is the composition root for the SkillMaker backend.
// It wires infrastructure (config, logger, database, storage) and domain
// modules (users, auth) into a single Gin HTTP server with graceful shutdown.
//
// Dependency graph (explicit — no DI magic):
//
//	config → logger → db → storage
//	                    └→ usersRepo → usersSvc
//	                    └→ authRepo  → authSvc (needs usersSvc + authCfg)
//	                → router → authRoutes → srv
//
// @title           SkillMaker API
// @version         1.0
// @description     Plataforma interna de formación en video — LMS corporativo
// @host            localhost:3000
// @BasePath        /api
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/yersonreyes/SkillMaker-/backend/internal/middleware"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/auth"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/users"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/config"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/database"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/httpserver"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/logger"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/storage"
)

func main() {
	// ── 1. Config + logger ────────────────────────────────────────────────────
	cfg := config.MustLoad()

	log := logger.New(cfg.LogLevel, cfg.AppEnv)
	slog.SetDefault(log)

	// ── 2. Infrastructure ─────────────────────────────────────────────────────
	db, err := database.Open(cfg.DatabaseURL, cfg.DBMaxOpenConns, cfg.DBMaxIdleConns)
	if err != nil {
		log.Error("no se pudo abrir la base de datos", "err", err)
		os.Exit(1) //nolint:gocritic // exitAfterDefer: defer is unreachable; intentional early exit before server starts
	}
	defer database.Close(db)

	storageClient, err := storage.New(&cfg.Storage)
	if err != nil {
		log.Error("no se pudo inicializar storage", "err", err)
		os.Exit(1) //nolint:gocritic // exitAfterDefer: defer is unreachable; intentional early exit before server starts
	}

	// ── 3. Modules ────────────────────────────────────────────────────────────
	usersRepo := users.NewRepository(db)
	usersSvc := users.NewService(usersRepo)

	coursesRepo := courses.NewRepository(db)
	coursesSvc := courses.NewService(coursesRepo, storageClient, cfg.Storage.PresignTTL, cfg.Storage.MaxUploadBytes)

	authRepo := auth.NewRepository(db)
	authCfg := auth.Config{
		JWTSecret:             cfg.Auth.JWTSecret,
		JWTExpiresIn:          cfg.Auth.JWTExpiresIn,
		RefreshTokenExpiresIn: cfg.Auth.RefreshTokenExpiresIn,
		GoogleClientID:        cfg.Auth.GoogleClientID,
		GoogleHostedDomain:    cfg.Auth.GoogleHostedDomain,
	}
	authSvc := auth.NewService(authCfg, usersSvc, authRepo)

	// ── 4. HTTP server ────────────────────────────────────────────────────────
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := httpserver.NewRouter(&cfg, db, storageClient)
	api := router.Group("/api")
	auth.RegisterRoutes(api, authSvc)

	// Protected route groups — shared across modules that need JWT / RBAC.
	// usersRepo/usersSvc are already built above (lines 64-65); reused here.
	protected := api.Group("", middleware.JWT(cfg.Auth.JWTSecret))
	adminGrp := protected.Group("", middleware.RequireRole("administrador"))
	users.RegisterRoutes(adminGrp, protected, usersSvc)

	// Courses module — creador-only routes.
	creatorGrp := protected.Group("", middleware.RequireRole("creador"))
	courses.RegisterRoutes(creatorGrp, coursesSvc)
	// Additional modules (evaluations, approvals, certificates,
	// reporting) will be wired here as they are implemented in subsequent changes.

	srv := httpserver.NewServer(cfg.Port, router)

	// ── 5. Start + graceful shutdown ──────────────────────────────────────────
	go func() {
		log.Info("servidor escuchando", "port", cfg.Port, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Info("apagando servidor", "signal", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("shutdown forzado", "err", err)
	}
	log.Info("servidor apagado")
}
