package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/farid/reservation-service/internal/reservation/model"
)

func (h *reservationHandler) getAvailability(c *gin.Context) {
	vt := model.VehicleType(c.Query("type"))
	if !model.IsValidVehicleType(vt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "VALIDATION", "message": "type query must be CAR or MOTORCYCLE"})
		return
	}
	floors, total, err := h.uc.Availability(c.Request.Context(), vt)
	if err != nil {
		renderError(c, err)
		return
	}
	resp := availabilityResp{AvailableCount: total, ByFloor: make([]byFloor, 0, len(floors))}
	for _, f := range floors {
		resp.ByFloor = append(resp.ByFloor, byFloor{Floor: f.Floor, Count: f.Count})
	}
	c.JSON(http.StatusOK, resp)
}

func (h *reservationHandler) create(c *gin.Context) {
	var body createReq
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "BAD_REQUEST", "message": err.Error()})
		return
	}
	idem := c.GetHeader("Idempotency-Key")
	if idem == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "VALIDATION", "message": "Idempotency-Key required"})
		return
	}
	out, err := h.uc.Create(c.Request.Context(), model.CreateReservationRequest{
		DriverID:        c.GetString(ctxDriverID),
		VehicleType:     model.VehicleType(body.VehicleType),
		Mode:            body.Mode,
		PreferredSpotID: body.PreferredSpotID,
		IdempotencyKey:  idem,
	})
	if err != nil {
		renderError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toDTO(out))
}

func (h *reservationHandler) get(c *gin.Context) {
	out, err := h.uc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		renderError(c, err)
		return
	}
	c.JSON(http.StatusOK, toDTO(out))
}

func (h *reservationHandler) confirm(c *gin.Context) {
	out, err := h.uc.Confirm(c.Request.Context(), c.Param("id"))
	if err != nil {
		renderError(c, err)
		return
	}
	c.JSON(http.StatusOK, toDTO(out))
}

func (h *reservationHandler) cancel(c *gin.Context) {
	var body cancelReq
	_ = c.ShouldBindJSON(&body) // body is optional
	out, err := h.uc.Cancel(c.Request.Context(), model.CancelRequest{ID: c.Param("id"), Reason: body.Reason})
	if err != nil {
		renderError(c, err)
		return
	}
	c.JSON(http.StatusOK, toDTO(out))
}

func (h *reservationHandler) checkIn(c *gin.Context) {
	var body checkInReq
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "BAD_REQUEST", "message": err.Error()})
		return
	}
	out, err := h.uc.CheckIn(c.Request.Context(), model.CheckInRequest{
		ID:             c.Param("id"),
		Latitude:       body.Latitude,
		Longitude:      body.Longitude,
		GPSUnavailable: body.GPSUnavailable,
	})
	if err != nil {
		renderError(c, err)
		return
	}
	c.JSON(http.StatusOK, toDTO(out))
}

func (h *reservationHandler) checkOut(c *gin.Context) {
	out, err := h.uc.CheckOut(c.Request.Context(), c.Param("id"))
	if err != nil {
		renderError(c, err)
		return
	}
	c.JSON(http.StatusOK, toDTO(out))
}
