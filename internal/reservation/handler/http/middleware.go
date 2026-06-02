package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/farid/reservation-service/pkg/logger"

	pkgjwt "github.com/farid/reservation-service/pkg/jwt"
)

// jwtMiddleware parses + verifies the Bearer JWT, then resolves the internal
// driver UUID via user-service gRPC UpsertDriver (lazy registration).
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

		// Resolve internal driver UUID via user-service gRPC.
		if h.users != nil {
			driverID, upsertErr := h.users.UpsertDriver(c.Request.Context(), claims.Sub, claims.Phone, "")
			if upsertErr != nil {
				logger.Error(c.Request.Context(), "upsert driver failed", map[string]interface{}{logger.ErrorKey: upsertErr.Error()})
				c.AbortWithStatusJSON(http.StatusInternalServerError,
					gin.H{"error": "INTERNAL", "message": "driver resolution failed"})
				return
			}
			c.Set(ctxDriverID, driverID)
		} else {
			// Fallback: use sub directly (dev mode with no user-service)
			c.Set(ctxDriverID, claims.Sub)
		}
		c.Next()
	}
}
