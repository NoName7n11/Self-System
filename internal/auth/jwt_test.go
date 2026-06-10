package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"selfsystems/internal/config"
)

func TestTokenFromRequestUsesAuthorizationHeaderOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	request := httptest.NewRequest(http.MethodGet, "/api/v1/sync/events?access_token=query-token", nil)
	c.Request = request

	if token := tokenFromRequest(c); token != "" {
		t.Fatalf("expected empty token when only query token is present, got %q", token)
	}

	c.Request.Header.Set("Authorization", "Bearer header-token")
	if token := tokenFromRequest(c); token != "header-token" {
		t.Fatalf("expected header token, got %q", token)
	}
}

func TestMiddlewareRejectsQueryTokenWithoutHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := NewJWTService(config.AuthConfig{
		Enabled:         true,
		JWTSecret:       "test-secret",
		JWTIssuer:       "self-systems",
		JWTAudience:     "self-systems-clients",
		TokenTTLMinutes: 5,
	})
	validToken, err := service.IssueToken("user-1")
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	router := gin.New()
	router.GET("/protected", service.Middleware(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	queryRequest := httptest.NewRequest(http.MethodGet, "/protected?access_token="+validToken, nil)
	queryResponse := httptest.NewRecorder()
	router.ServeHTTP(queryResponse, queryRequest)
	if queryResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized for query token, got %d", queryResponse.Code)
	}

	headerRequest := httptest.NewRequest(http.MethodGet, "/protected", nil)
	headerRequest.Header.Set("Authorization", "Bearer "+validToken)
	headerResponse := httptest.NewRecorder()
	router.ServeHTTP(headerResponse, headerRequest)
	if headerResponse.Code != http.StatusOK {
		t.Fatalf("expected success for header token, got %d", headerResponse.Code)
	}
}
