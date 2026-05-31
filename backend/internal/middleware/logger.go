package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
)

// Logger returns a Gin middleware that logs each request as a structured
// slog event after the handler chain completes.
// Fields logged: request_id, method, path, status, duration_ms, ip, user_id
// (user_id is included only when a JWT was validated upstream).
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start).Milliseconds()
		requestID := requestid.Get(c)
		userID := UserIDFrom(c)

		args := []any{
			"request_id", requestID,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", duration,
			"ip", c.ClientIP(),
		}
		if userID != "" {
			args = append(args, "user_id", userID)
		}

		slog.Info("request", args...)
	}
}
