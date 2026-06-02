// Package http exposes the reservation domain over REST/JSON for the mini app.
package http

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/farid/reservation-service/internal/reservation/usecase"
	"github.com/farid/reservation-service/pkg/grpcclient"
	"github.com/farid/reservation-service/pkg/rate"
)

const ctxDriverID = "driver_id"

type reservationHandler struct {
	uc     usecase.ReservationUsecase
	users  grpcclient.UserClient
	jwtKey string // super-app RS256 PEM; "" skips signature verify (dev)
}

// RegisterReservationHandler mounts mini-app REST routes under rg.
// All routes except /availability require auth. Idempotency-Key header required on writes.
// lim may be nil; rate limiting is skipped when nil or on Redis errors (fail-open).
func RegisterReservationHandler(rg *gin.RouterGroup, uc usecase.ReservationUsecase, jwtPubKeyPEM string, lim rate.Limiter, users grpcclient.UserClient) {
	h := &reservationHandler{uc: uc, users: users, jwtKey: jwtPubKeyPEM}

	rg.GET("/availability", h.jwtMiddleware(), h.getAvailability)

	res := rg.Group("/reservations")
	res.Use(h.jwtMiddleware())

	res.POST("", rateLimitDriver(lim, "reservations:create", 10, time.Minute), h.create)
	res.GET("/:id", h.get)
	res.POST("/:id/confirm", h.confirm)
	res.POST("/:id/cancel", h.cancel)
	res.POST("/:id/check-in", h.checkIn)
	res.POST("/:id/check-out", h.checkOut)
}
