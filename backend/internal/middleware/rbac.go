package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/httperr"
)

// RequireRole returns a Gin middleware that enforces role-based access control.
// The request is allowed if the authenticated user has ANY of the allowed roles.
// Returns 403 if the user holds none of them.
// This middleware must be used AFTER the JWT middleware (which populates roles).
func RequireRole(allowed ...string) gin.HandlerFunc {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, r := range allowed {
		allowedSet[r] = struct{}{}
	}

	return func(c *gin.Context) {
		roles := RolesFrom(c)
		for _, role := range roles {
			if _, ok := allowedSet[role]; ok {
				c.Next()
				return
			}
		}
		httperr.Render(c, httperr.Forbidden("FORBIDDEN", "you do not have permission to access this resource"))
	}
}
