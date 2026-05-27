package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Envelope is the standard JSON wrapper for all API responses.
type Envelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// APIError holds structured error information returned to the client.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Meta carries optional pagination or supplementary metadata.
type Meta struct {
	Page       int   `json:"page,omitempty"`
	PageSize   int   `json:"page_size,omitempty"`
	TotalItems int64 `json:"total_items,omitempty"`
	TotalPages int   `json:"total_pages,omitempty"`
}

// OK sends a 200 response with the provided data payload.
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Envelope{Success: true, Data: data})
}

// Created sends a 201 response with the provided data payload.
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Envelope{Success: true, Data: data})
}

// OKWithMeta sends a 200 response with data and pagination metadata.
func OKWithMeta(c *gin.Context, data interface{}, meta *Meta) {
	c.JSON(http.StatusOK, Envelope{Success: true, Data: data, Meta: meta})
}

// NoContent sends a 204 response with no body.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// BadRequest sends a 400 response with the given code and message.
func BadRequest(c *gin.Context, code, message string) {
	c.JSON(http.StatusBadRequest, Envelope{
		Success: false,
		Error:   &APIError{Code: code, Message: message},
	})
}

// Unauthorized sends a 401 response.
func Unauthorized(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, Envelope{
		Success: false,
		Error:   &APIError{Code: "UNAUTHORIZED", Message: message},
	})
}

// Forbidden sends a 403 response.
func Forbidden(c *gin.Context, message string) {
	c.JSON(http.StatusForbidden, Envelope{
		Success: false,
		Error:   &APIError{Code: "FORBIDDEN", Message: message},
	})
}

// NotFound sends a 404 response.
func NotFound(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, Envelope{
		Success: false,
		Error:   &APIError{Code: "NOT_FOUND", Message: message},
	})
}

// Conflict sends a 409 response.
func Conflict(c *gin.Context, code, message string) {
	c.JSON(http.StatusConflict, Envelope{
		Success: false,
		Error:   &APIError{Code: code, Message: message},
	})
}

// UnprocessableEntity sends a 422 response.
func UnprocessableEntity(c *gin.Context, code, message string) {
	c.JSON(http.StatusUnprocessableEntity, Envelope{
		Success: false,
		Error:   &APIError{Code: code, Message: message},
	})
}

// InternalServerError sends a 500 response. The raw error is not exposed.
func InternalServerError(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, Envelope{
		Success: false,
		Error:   &APIError{Code: "INTERNAL_ERROR", Message: "an unexpected error occurred"},
	})
}

// ServiceUnavailable sends a 503 response. Used during health checks.
func ServiceUnavailable(c *gin.Context, message string) {
	c.JSON(http.StatusServiceUnavailable, Envelope{
		Success: false,
		Error:   &APIError{Code: "SERVICE_UNAVAILABLE", Message: message},
	})
}