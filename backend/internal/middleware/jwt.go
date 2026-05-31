package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/httperr"
)

const (
	contextKeyUserID = "userID"
	contextKeyRoles  = "roles"
)

// JWT returns a Gin middleware that validates an HS256 Bearer token from the
// Authorization header. On success it injects "userID" (string) and "roles"
// ([]string) into the Gin context so downstream handlers can read them via
// UserIDFrom and RolesFrom helpers. Returns 401 on any validation failure.
func JWT(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			httperr.Render(c, httperr.Unauthorized("MISSING_TOKEN", "authorization header is required"))
			return
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		}, jwt.WithValidMethods([]string{"HS256"}))
		if err != nil || !token.Valid {
			httperr.Render(c, httperr.Unauthorized("INVALID_TOKEN", "token is invalid or expired"))
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			httperr.Render(c, httperr.Unauthorized("INVALID_TOKEN", "malformed token claims"))
			return
		}

		userID, _ := claims["sub"].(string)
		roles := toStringSlice(claims["roles"])

		c.Set(contextKeyUserID, userID)
		c.Set(contextKeyRoles, roles)

		c.Next()
	}
}

// UserIDFrom extracts the authenticated user ID from the Gin context.
// Returns an empty string if the JWT middleware has not run or the key is absent.
func UserIDFrom(c *gin.Context) string {
	v, _ := c.Get(contextKeyUserID)
	id, _ := v.(string)
	return id
}

// RolesFrom extracts the authenticated user's roles from the Gin context.
// Returns nil if the JWT middleware has not run or the key is absent.
func RolesFrom(c *gin.Context) []string {
	v, _ := c.Get(contextKeyRoles)
	roles, _ := v.([]string)
	return roles
}

// toStringSlice converts a JWT claim value ([]any or []string) to []string.
func toStringSlice(v any) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}
