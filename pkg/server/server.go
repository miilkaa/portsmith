// Package server provides a Gin-based HTTP server with sensible defaults:
// recovery, request ID, CORS, structured error handling, and health endpoint.
package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// BindAndValidate binds JSON body and runs validator tags.
// On error it writes a 400 response and returns the error —
// the handler should return immediately when err != nil.
func BindAndValidate(c *gin.Context, req any) error {
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return err
	}
	return nil
}
