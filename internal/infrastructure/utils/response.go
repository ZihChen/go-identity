package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

func Unauthorized(c *gin.Context, error, message string) *gin.Context {
	c.JSON(http.StatusUnauthorized, ErrorResponse{
		Error:   error,
		Message: message,
	})
	return c
}

func BadRequest(c *gin.Context, error, message string) *gin.Context {
	c.JSON(http.StatusBadRequest, ErrorResponse{
		Error:   error,
		Message: message,
	})
	return c
}

func InternalError(c *gin.Context, error, message string) *gin.Context {
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Error:   error,
		Message: message,
	})
	return c
}

func Forbidden(c *gin.Context, error, message string) *gin.Context {
	c.JSON(http.StatusForbidden, ErrorResponse{
		Error:   error,
		Message: message,
	})
	return c
}

func NotFound(c *gin.Context, error, message string) *gin.Context {
	c.JSON(http.StatusNotFound, ErrorResponse{
		Error:   error,
		Message: message,
	})
	return c
}