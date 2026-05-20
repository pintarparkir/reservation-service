package http

import (
	"github.com/gin-gonic/gin"

	"github.com/farid/reservation-service/pkg/utils"
)

// renderError is a convenience wrapper around utils.Error for backward compatibility.
func renderError(c *gin.Context, err error) {
	utils.Error(c, err)
}
