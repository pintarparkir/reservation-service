package http

import (
	"github.com/gin-gonic/gin"

	"github.com/farid/reservation-service/internal/reservation/model"
	apperror "github.com/farid/reservation-service/pkg/error"
	"github.com/farid/reservation-service/pkg/utils"
)

func (h *reservationHandler) getAvailability(c *gin.Context) {
	vt := model.VehicleType(c.Query("type"))
	if !model.IsValidVehicleType(vt) {
		renderError(c, &apperror.AppError{Code: "VALIDATION", Message: "type query must be CAR or MOTORCYCLE"})
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
	utils.OK(c, resp, "availability retrieved")
}

func (h *reservationHandler) create(c *gin.Context) {
	var body createReq
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.Error(c, err)
		return
	}
	idem := c.GetHeader("Idempotency-Key")
	if idem == "" {
		renderError(c, &apperror.AppError{Code: "VALIDATION", Message: "Idempotency-Key required"})
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
	utils.Created(c, toDTO(out), "reservation created")
}

func (h *reservationHandler) get(c *gin.Context) {
	out, err := h.uc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		renderError(c, err)
		return
	}
	utils.OK(c, toDTO(out), "reservation retrieved")
}

func (h *reservationHandler) confirm(c *gin.Context) {
	out, err := h.uc.Confirm(c.Request.Context(), c.Param("id"))
	if err != nil {
		renderError(c, err)
		return
	}
	utils.OK(c, toDTO(out), "reservation confirmed")
}

func (h *reservationHandler) cancel(c *gin.Context) {
	var body cancelReq
	_ = c.ShouldBindJSON(&body) // body is optional
	out, err := h.uc.Cancel(c.Request.Context(), model.CancelRequest{ID: c.Param("id"), Reason: body.Reason})
	if err != nil {
		renderError(c, err)
		return
	}
	utils.OK(c, toDTO(out), "reservation cancelled")
}

func (h *reservationHandler) checkIn(c *gin.Context) {
	var body checkInReq
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.Error(c, err)
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
	utils.OK(c, toDTO(out), "checked in")
}

func (h *reservationHandler) checkOut(c *gin.Context) {
	out, err := h.uc.CheckOut(c.Request.Context(), c.Param("id"))
	if err != nil {
		renderError(c, err)
		return
	}
	utils.OK(c, toDTO(out), "checked out")
}
