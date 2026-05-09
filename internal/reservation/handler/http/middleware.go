package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	pkgjwt "github.com/farid/reservation-service/pkg/jwt"
)

// jwtMiddleware parses + verifies the Bearer JWT, sets driver_id in context.
// Note: driver_id here is the super-app's `sub` (external_user_id). When we
// need our internal user_profile.id we'd resolve via gRPC user-service.
// For MVP we trust `sub` as a logical driver identifier — the user-service
// already lazy-registered the driver on first /v1/me call.
func (h *reservationHandler) jwtMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("Authorization")
		if !strings.HasPrefix(raw, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized,
				gin.H{"error": "UNAUTHENTICATED", "message": "missing bearer token"})
			return
		}
		token := strings.TrimPrefix(raw, "Bearer ")

		claims, err := pkgjwt.Parse(token, h.jwtKey)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized,
				gin.H{"error": "UNAUTHENTICATED", "message": err.Error()})
			return
		}

		// Use the JWT sub as the driver identifier for now. A future iteration
		// adds a Redis-cached lookup against user-service.GetUserByExternalID.
		c.Set(ctxDriverID, claims.Sub)
		c.Next()
	}
}
