package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/self-docs/backend/internal/repository"
)

// respondBindError maps JSON binding failures: oversized bodies become 413,
// everything else is a malformed-body 400.
func respondBindError(c *gin.Context, err error) {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		respondError(c, http.StatusRequestEntityTooLarge, CodePayloadTooLarge,
			"request body exceeds the configured size limit")
		return
	}
	respondError(c, http.StatusBadRequest, CodeBadRequest, "malformed JSON body")
}

// Error codes from docs/api.md section 2.
const (
	CodeValidationError = "validation_error"
	CodeBadRequest      = "bad_request"
	CodeUnauthorized    = "unauthorized"
	CodeForbidden       = "forbidden"
	CodeNotFound        = "not_found"
	CodeConflict        = "conflict"
	CodePayloadTooLarge = "payload_too_large"
	CodeUnprocessable   = "unprocessable"
	CodeRateLimited     = "rate_limited"
	CodeInternalError   = "internal_error"
)

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Details []fieldError `json:"details,omitempty"`
}

// fieldError pinpoints a single invalid field.
type fieldError struct {
	Field string `json:"field"`
	Issue string `json:"issue"`
}

// respondError writes the standard error envelope.
func respondError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, errorBody{Error: errorDetail{Code: code, Message: message}})
}

// respondValidation writes a validation_error envelope with field details.
func respondValidation(c *gin.Context, message string, details ...fieldError) {
	c.AbortWithStatusJSON(http.StatusBadRequest, errorBody{Error: errorDetail{
		Code:    CodeValidationError,
		Message: message,
		Details: details,
	}})
}

// respondRepoError maps a repository sentinel error to its HTTP status. It
// returns true when the error was recognized and a response was written, and
// false when the caller should handle it as an internal error.
func respondRepoError(c *gin.Context, err error) bool {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		respondError(c, http.StatusNotFound, CodeNotFound, "resource not found")
	case errors.Is(err, repository.ErrConflict):
		respondError(c, http.StatusConflict, CodeConflict, "a document with this slug already exists")
	case errors.Is(err, repository.ErrInvalidHierarchy):
		respondError(c, http.StatusUnprocessableEntity, CodeUnprocessable, err.Error())
	default:
		return false
	}
	return true
}

// respondInternal logs the underlying error and returns a generic message so
// internal details are never leaked to clients.
func respondInternal(c *gin.Context, err error) {
	log.Printf("internal error: %v", err)
	respondError(c, http.StatusInternalServerError, CodeInternalError, "an unexpected error occurred")
}

// handleRepoError maps known repository errors or falls back to a 500.
func handleRepoError(c *gin.Context, err error) {
	if !respondRepoError(c, err) {
		respondInternal(c, err)
	}
}
