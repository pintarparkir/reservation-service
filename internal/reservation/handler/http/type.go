// Package http exposes the reservation domain over REST/JSON for the mini app.
package http

import (
	"github.com/gin-gonic/gin"

	"github.com/farid/reservation-service/internal/reservation/usecase"
)

const ctxDriverID = "driver_id"

type reservationHandler struct {
	uc     usecase.ReservationUsecase
	jwtKey string // super-app RS256 PEM; "" skips signature verify (dev)
}

// RegisterReservationHandler mounts mini-app REST routes under rg.
// All routes except /availability require auth. Idempotency-Key header required on writes.
func RegisterReservationHandler(rg *gin.RouterGroup, uc usecase.ReservationUsecase, jwtPubKeyPEM string) {
	h := &reservationHandler{uc: uc, jwtKey: jwtPubKeyPEM}

	rg.GET("/availability", h.jwtMiddleware(), h.getAvailability)

	res := rg.Group("/reservations")
	res.Use(h.jwtMiddleware())

	res.POST("", h.create)
	res.GET("/:id", h.get)
	res.POST("/:id/confirm", h.confirm)
	res.POST("/:id/cancel", h.cancel)
	res.POST("/:id/check-in", h.checkIn)
	res.POST("/:id/check-out", h.checkOut)
}
