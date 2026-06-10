package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const defaultMaxRequestBodyBytes int64 = 1 << 20 // 1 MiB

func MaxBodyBytesMiddleware(maxBytes int64) gin.HandlerFunc {
	if maxBytes <= 0 {
		maxBytes = defaultMaxRequestBodyBytes
	}

	return func(c *gin.Context) {
		if c.Request != nil && c.Request.ContentLength > maxBytes {
			respondErrorCode(c, http.StatusRequestEntityTooLarge, "payload_too_large", "request body too large")
			c.Abort()
			return
		}
		if c.Request != nil && c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
