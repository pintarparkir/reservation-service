// Package utils provides HTTP response helper functions aligned with gin framework.
package utils

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	apperror "github.com/farid/reservation-service/pkg/error"
	"github.com/farid/reservation-service/pkg/logger"
)

// ResponseWrapper encapsulates all response data in a consistent format.
type ResponseWrapper struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
	Message string      `json:"message"`
	Code    int         `json:"code"`
}

// ErrorInfo provides structured error details for responses.
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// AuditMeta holds metadata for audit logging.
type AuditMeta struct {
	Method      string    `json:"method"`
	Path        string    `json:"path"`
	StatusCode  int       `json:"status_code"`
	ClientIP    string    `json:"client_ip"`
	Timestamp   time.Time `json:"timestamp"`
	RequestSize int       `json:"request_size"`
}

// OK sends a success response with HTTP status 200.
func OK(c *gin.Context, data interface{}, message string) {
	Response(c, data, message, http.StatusOK)
}

// Created sends a success response with HTTP status 201.
func Created(c *gin.Context, data interface{}, message string) {
	Response(c, data, message, http.StatusCreated)
}

// Response sends a success response with the given HTTP status code.
func Response(c *gin.Context, data interface{}, message string, statusCode int) {
	success := statusCode < http.StatusBadRequest

	logAudit(c, statusCode, nil)

	result := ResponseWrapper{
		Success: success,
		Data:    data,
		Message: message,
		Code:    statusCode,
	}

	c.JSON(statusCode, result)
}

// Error sends an error response, mapping domain errors to HTTP status codes.
func Error(c *gin.Context, err error) {
	if err == nil {
		Response(c, nil, "unknown error", http.StatusInternalServerError)
		return
	}

	var statusCode int
	var errorCode string
	var errorMessage string

	// Map domain errors to HTTP status codes.
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		errorCode = appErr.Code
		errorMessage = appErr.Message

		switch appErr.Code {
		case "VALIDATION", "GEOFENCE_VIOLATION":
			statusCode = http.StatusBadRequest
		case "UNAUTHENTICATED":
			statusCode = http.StatusUnauthorized
		case "NOT_FOUND":
			statusCode = http.StatusNotFound
		case "CONFLICT", "DOUBLE_BOOK":
			statusCode = http.StatusConflict
		case "INVALID_STATE":
			statusCode = http.StatusUnprocessableEntity
		case "LOCK_UNAVAILABLE":
			statusCode = http.StatusTooManyRequests
		case "UPSTREAM_DOWN":
			statusCode = http.StatusServiceUnavailable
		default:
			statusCode = http.StatusInternalServerError
		}
	} else {
		// Fallback for non-AppError errors.
		statusCode = http.StatusInternalServerError
		errorCode = "INTERNAL"
		errorMessage = err.Error()
	}

	logAudit(c, statusCode, &ErrorInfo{Code: errorCode, Message: errorMessage})

	result := ResponseWrapper{
		Success: false,
		Error: &ErrorInfo{
			Code:    errorCode,
			Message: errorMessage,
		},
		Message: errorMessage,
		Code:    statusCode,
	}

	c.JSON(statusCode, result)
}

// logAudit logs request metadata for audit purposes.
func logAudit(c *gin.Context, statusCode int, errInfo *ErrorInfo) {
	meta := AuditMeta{
		Method:      c.Request.Method,
		Path:        c.Request.URL.Path,
		StatusCode:  statusCode,
		ClientIP:    c.ClientIP(),
		Timestamp:   time.Now(),
		RequestSize: int(c.Request.ContentLength),
	}

	metaJSON, _ := json.Marshal(meta)
	fields := map[string]interface{}{
		"meta": string(metaJSON),
	}

	if errInfo != nil {
		fields["error_code"] = errInfo.Code
		fields["error_message"] = errInfo.Message
		logger.Warn(c.Request.Context(), "http_response_error", fields)
	} else {
		logger.Info(c.Request.Context(), "http_response_ok", fields)
	}
}
