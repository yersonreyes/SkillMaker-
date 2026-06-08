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
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/approvals"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/auth"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/certificates"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/notifications"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/reporting"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/users"
	usersService "github.com/yersonreyes/SkillMaker-/backend/internal/modules/users/service"
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

	// C8.1 — session management (caller-scoped) on the JWT-protected group.
	auth.RegisterSessionRoutes(protected, authSvc)

	adminGrp := protected.Group("", middleware.RequireRole("administrador"))
	users.RegisterRoutes(adminGrp, protected, usersSvc)

	// Courses module — creador-only routes.
	creatorGrp := protected.Group("", middleware.RequireRole("creador"))
	courses.RegisterRoutes(creatorGrp, coursesSvc)
	// Catalog + enrollment (C2.4) — alumno-facing, JWT-only (no RequireRole).
	courses.RegisterCatalogRoutes(protected, coursesSvc)
	// Categoria admin CRUD — administrador-only.
	courses.RegisterCategoriasAdminRoutes(adminGrp, coursesSvc)

	// Notifications module (notifications-inapp) — MUST be built BEFORE certsSvc and approvalsSvc
	// (both depend on it via WithNotifier). Pure leaf: imports nobody from other domain modules.
	notificationsRepo := notifications.NewRepository(db)
	notificationsSvc := notifications.NewService(notificationsRepo)

	// Certificates module (C5.1) — MUST be built BEFORE evaluationsSvc (evaluations depends on it).
	// userNameAdapter bridges users.Service.GetByID (*UserSummary) to certificates.UserNameReader.
	// coursesSvc satisfies certificates.CourseTituloReader structurally (GetCourseTitulo added in C5.1).
	certsRepo := certificates.NewRepository(db)
	certsSvc := certificates.NewService(
		certsRepo,
		storageClient,
		userNameAdapter{svc: usersSvc},
		coursesSvc,
		cfg.Storage.PresignTTL,
		certificates.WithNotifier(notificationsSvc),
	)

	// Evaluations module (C3.1 + C3.2 + C2.4) — coursesSvc satisfies evaluations.CoursesChecker structurally.
	// C2.4: WithEnrollmentCompleter wired — coursesSvc satisfies evaluations.EnrollmentCompleter structurally
	// (MarkEnrollmentCompleted on the public courses.Service interface).
	// C5.1: WithCertificateIssuer wired — certsSvc satisfies evaluations.CertificateIssuer structurally.
	evaluationsRepo := evaluations.NewRepository(db)
	evaluationsSvc := evaluations.NewService(evaluationsRepo, coursesSvc,
		evaluations.WithEnrollmentCompleter(coursesSvc),
		evaluations.WithCertificateIssuer(certsSvc),
	)
	evaluations.RegisterRoutes(creatorGrp, evaluationsSvc)
	// C3.2: student attempt lifecycle on the JWT-only protected group (no RequireRole restriction).
	evaluations.RegisterStudentRoutes(protected, evaluationsSvc)

	// Approvals module (C4.1) — coursesSvc satisfies CourseStateManager, evaluationsSvc satisfies EvaluationValidator (both structural).
	approvalsRepo := approvals.NewRepository(db)
	approvalsSvc := approvals.NewService(approvalsRepo, coursesSvc, evaluationsSvc, approvals.WithNotifier(notificationsSvc))
	approvals.RegisterCreatorRoutes(creatorGrp, approvalsSvc) // POST /courses/:courseId/submit
	approvals.RegisterAdminRoutes(adminGrp, approvalsSvc)     // GET /approvals/pending, POST approve/reject
	approvals.RegisterHistoryRoutes(protected, approvalsSvc)  // GET /courses/:id/approvals (owner-or-admin)

	// Certificates module routes (C5.1) — JWT-only, no RequireRole.
	certificates.RegisterRoutes(protected, certsSvc)
	// Public certificate verification (no JWT) — mounted on the unauthenticated /api group.
	certificates.RegisterPublicRoutes(api, certsSvc)

	// Notifications module routes (notifications-inapp) — JWT-only, no RequireRole.
	notifications.RegisterRoutes(protected, notificationsSvc)

	// Reporting module (C6.1) — pure-SQL read-only, no migration.
	reportingRepo := reporting.NewRepository(db)
	reportingSvc := reporting.NewService(reportingRepo)
	// First supervisor-gated group.
	supervisorGrp := protected.Group("", middleware.RequireRole("supervisor"))
	reporting.RegisterAdminRoutes(adminGrp, reportingSvc)           // GET /reports/global, /reports/courses
	reporting.RegisterSupervisorRoutes(supervisorGrp, reportingSvc) // GET /reports/team
	reporting.RegisterSelfRoutes(protected, reportingSvc)           // GET /reports/users/:id/progress (admin-or-self)

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

// ── userNameAdapter ────────────────────────────────────────────────────────────

// userNameAdapter bridges users.Service.GetByID (returns *UserSummary) to
// certificates.UserNameReader (expects GetUserNombre(ctx, id) (string, error)).
// users.Service.GetByID signature mismatch requires this thin adapter in main.go.
type userNameAdapter struct {
	svc usersService.Service
}

func (a userNameAdapter) GetUserNombre(ctx context.Context, id string) (string, error) {
	u, err := a.svc.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	return u.Nombre, nil
}
