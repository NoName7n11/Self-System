package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
	originAllowList := normalizeOrigins(allowedOrigins)
	allowAll := len(originAllowList) == 0

	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.Request.Header.Get("Origin"))
		originLower := strings.ToLower(origin)

		if origin != "" && (allowAll || originAllowList[originLower]) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
			c.Header("Access-Control-Max-Age", "3600")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func normalizeOrigins(origins []string) map[string]bool {
	result := make(map[string]bool, len(origins))
	for _, origin := range origins {
		trimmed := strings.TrimSpace(strings.ToLower(origin))
		if trimmed == "" {
			continue
		}
		result[trimmed] = true
	}
	return result
}
