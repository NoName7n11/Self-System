package http

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMaxBodyBytesMiddlewareRejectsByContentLength(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/mutate", MaxBodyBytesMiddleware(64), func(c *gin.Context) {
		_, _ = io.Copy(io.Discard, c.Request.Body)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	body := strings.Repeat("x", 128)
	req := httptest.NewRequest(http.MethodPost, "/mutate", bytes.NewBufferString(body))
	req.ContentLength = int64(len(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rec.Code)
	}
}

func TestMaxBodyBytesMiddlewareAllowsSmallBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/mutate", MaxBodyBytesMiddleware(64), func(c *gin.Context) {
		_, _ = io.Copy(io.Discard, c.Request.Body)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	body := strings.Repeat("x", 32)
	req := httptest.NewRequest(http.MethodPost, "/mutate", bytes.NewBufferString(body))
	req.ContentLength = int64(len(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestMaxBodyBytesMiddlewareCapsStreamingBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/mutate", MaxBodyBytesMiddleware(64), func(c *gin.Context) {
		if _, err := io.Copy(io.Discard, c.Request.Body); err != nil {
			c.AbortWithStatus(http.StatusRequestEntityTooLarge)
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	body := strings.Repeat("x", 128)
	req := httptest.NewRequest(http.MethodPost, "/mutate", bytes.NewBufferString(body))
	req.ContentLength = -1
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 from streaming overflow, got %d", rec.Code)
	}
}
