package http

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func pagination(c *gin.Context) (int, int) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(defaultPageLimit)))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit <= 0 {
		limit = defaultPageLimit
	}
	if limit > maxPageLimit {
		limit = maxPageLimit
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func parseBoundedInt(raw string, fallback, min, max int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	if parsed < min {
		return fallback
	}
	if parsed > max {
		return max
	}
	return parsed
}

func respondOperationError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	message := err.Error()
	if isValidationErrorMessage(message) {
		respondError(c, http.StatusBadRequest, message)
		return
	}

	respondInternalError(c, err)
}

func respondInternalError(c *gin.Context, err error) {
	slog.Error("operation failed", "error", err, "request_id", RequestIDFromContext(c), "path", c.Request.URL.Path)
	respondError(c, http.StatusInternalServerError, "internal server error")
}

func isValidationErrorMessage(message string) bool {
	lower := strings.ToLower(strings.TrimSpace(message))
	validationHints := []string{
		"required",
		"invalid",
		"not found",
		"must be",
		"empty",
	}

	for _, hint := range validationHints {
		if strings.Contains(lower, hint) {
			return true
		}
	}

	return false
}

func respondError(c *gin.Context, status int, message string) {
	code := errorCodeInternal
	switch status {
	case http.StatusBadRequest:
		code = errorCodeValidation
	case http.StatusNotFound:
		code = errorCodeNotFound
	case http.StatusServiceUnavailable:
		code = errorCodeServiceUnavailable
	}
	respondErrorCode(c, status, code, message)
}

func respondErrorCode(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": message, "code": code})
}
