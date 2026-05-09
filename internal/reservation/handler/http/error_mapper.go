package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	apperror "github.com/farid/reservation-service/pkg/error"
)

// renderError converts a domain error into the appropriate HTTP status + body.
func renderError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	if ae, ok := err.(*apperror.AppError); ok {
		switch ae.Code {
		case "VALIDATION", "GEOFENCE_VIOLATION":
			c.JSON(http.StatusBadRequest, gin.H{"error": ae.Code, "message": ae.Message})
			return
		case "NOT_FOUND":
			c.JSON(http.StatusNotFound, gin.H{"error": ae.Code, "message": ae.Message})
			return
		case "CONFLICT", "DOUBLE_BOOK":
			c.JSON(http.StatusConflict, gin.H{"error": ae.Code, "message": ae.Message})
			return
		case "INVALID_STATE":
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": ae.Code, "message": ae.Message})
			return
		case "LOCK_UNAVAILABLE":
			c.JSON(http.StatusTooManyRequests, gin.H{"error": ae.Code, "message": ae.Message})
			return
		case "UNAUTHENTICATED":
			c.JSON(http.StatusUnauthorized, gin.H{"error": ae.Code, "message": ae.Message})
			return
		case "UPSTREAM_DOWN":
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": ae.Code, "message": ae.Message})
			return
		}
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "INTERNAL", "message": err.Error()})
}
