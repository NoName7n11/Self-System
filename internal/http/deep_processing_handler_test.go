package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"selfsystems/internal/service"
)

func TestDeepProcessingHealthReturnsServiceUnavailableWhenMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/processing/deep/health", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusServiceUnavailable, "service_unavailable", "deep processing is not configured")
}

func TestDeepProcessingMetricsReturnsSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)

	processor := service.NewDeepProcessor(nil, nil, nil, nil, service.DeepProcessingSettings{
		Enabled:       true,
		QueueCapacity: 4,
		WorkerCount:   1,
	})
	handler := NewHandlerWithOptions(nil, nil, nil, nil, nil, nil, WithDeepProcessor(processor))
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/processing/deep/metrics", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response struct {
		Data service.DeepProcessingMetrics `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode deep metrics response: %v", err)
	}
	if !response.Data.Enabled {
		t.Fatalf("expected deep metrics enabled=true")
	}
	if response.Data.QueueCapacity != 4 {
		t.Fatalf("expected queue capacity 4, got %d", response.Data.QueueCapacity)
	}
}

func TestReprocessDeepResourceReturnsTooManyRequestsWhenQueueFull(t *testing.T) {
	gin.SetMode(gin.TestMode)

	processor := service.NewDeepProcessor(nil, nil, nil, nil, service.DeepProcessingSettings{
		Enabled:       true,
		QueueCapacity: 1,
		WorkerCount:   1,
	})
	handler := NewHandlerWithOptions(nil, nil, nil, nil, nil, nil, WithDeepProcessor(processor))
	router := gin.New()
	handler.RegisterRoutes(router)

	firstRequest := httptest.NewRequest(http.MethodPost, "/api/v1/processing/deep/reprocess/res-1", nil)
	firstRecorder := httptest.NewRecorder()
	router.ServeHTTP(firstRecorder, firstRequest)
	if firstRecorder.Code != http.StatusAccepted {
		t.Fatalf("expected first enqueue status %d, got %d", http.StatusAccepted, firstRecorder.Code)
	}

	secondRequest := httptest.NewRequest(http.MethodPost, "/api/v1/processing/deep/reprocess/res-2", nil)
	secondRecorder := httptest.NewRecorder()
	router.ServeHTTP(secondRecorder, secondRequest)

	assertErrorResponse(t, secondRecorder, http.StatusTooManyRequests, "rate_limited", service.ErrDeepProcessingQueueFull.Error())
}
