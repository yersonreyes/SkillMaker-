package httpserver

import (
	"net/http"

	"github.com/gin-contrib/requestid"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"

	"github.com/yersonreyes/SkillMaker-/backend/internal/middleware"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/config"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/storage"

	"github.com/gin-gonic/gin"
)

// healthBody is the JSON shape returned by the liveness endpoint.
type healthBody struct {
	Status string `json:"status"`
}

// readyBody is the JSON shape returned by the readiness endpoint on failure.
type readyBody struct {
	Status string   `json:"status"`
	Issues []string `json:"issues,omitempty"`
}

// NewRouter constructs the Gin engine with all global middleware and built-in
// routes (health, readiness, swagger docs). Domain routes are registered
// separately by each module's RegisterRoutes function in cmd/api/main.go.
func NewRouter(cfg *config.Config, db *gorm.DB, store storage.Client) *gin.Engine {
	r := gin.New()

	// Global middleware stack
	r.Use(
		gin.Recovery(),
		requestid.New(),
		middleware.Logger(),
		middleware.CORS(cfg.AllowedOrigins),
	)

	// Swagger UI (available in all environments; lock down via ALLOWED_ORIGINS
	// or a reverse proxy in production if desired)
	r.GET("/api/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Liveness: always 200, no dependency checks
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, healthBody{Status: "ok"})
	})

	// Readiness: checks DB ping + storage ping; returns 503 with details on failure
	r.GET("/api/health/ready", func(c *gin.Context) {
		var issues []string

		sqlDB, err := db.DB()
		if err != nil || sqlDB.Ping() != nil {
			issues = append(issues, "db")
		}

		if store != nil {
			if err := store.Ping(c.Request.Context()); err != nil {
				issues = append(issues, "storage")
			}
		}

		if len(issues) > 0 {
			c.JSON(http.StatusServiceUnavailable, readyBody{Status: "degraded", Issues: issues})
			return
		}

		c.JSON(http.StatusOK, readyBody{Status: "ok"})
	})

	return r
}
