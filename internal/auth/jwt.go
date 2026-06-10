package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"selfsystems/internal/config"
)

const (
	// ContextSubjectKey stores authenticated subject id in Gin context.
	ContextSubjectKey = "auth.subject"
)

type Claims struct {
	jwt.RegisteredClaims
}

type JWTService struct {
	enabled  bool
	secret   []byte
	issuer   string
	audience string
	tokenTTL time.Duration
}

func NewJWTService(cfg config.AuthConfig) *JWTService {
	ttl := time.Duration(cfg.TokenTTLMinutes) * time.Minute
	if ttl <= 0 {
		ttl = 60 * time.Minute
	}

	return &JWTService{
		enabled:  cfg.Enabled,
		secret:   []byte(strings.TrimSpace(cfg.JWTSecret)),
		issuer:   strings.TrimSpace(cfg.JWTIssuer),
		audience: strings.TrimSpace(cfg.JWTAudience),
		tokenTTL: ttl,
	}
}

func (s *JWTService) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !s.enabled {
			c.Next()
			return
		}

		tokenString := tokenFromRequest(c)
		if tokenString == "" {
			abortUnauthorized(c)
			return
		}

		claims := &Claims{}
		_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			if len(s.secret) == 0 {
				return nil, errors.New("missing jwt secret")
			}
			return s.secret, nil
		},
			jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
			jwt.WithIssuer(s.issuer),
			jwt.WithAudience(s.audience),
		)
		if err != nil {
			abortUnauthorized(c)
			return
		}
		if strings.TrimSpace(claims.Subject) == "" {
			abortUnauthorized(c)
			return
		}

		c.Set(ContextSubjectKey, claims.Subject)
		c.Next()
	}
}

func (s *JWTService) IssueToken(subject string) (string, error) {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return "", errors.New("subject is required")
	}
	if len(s.secret) == 0 {
		return "", errors.New("jwt secret is required")
	}

	now := time.Now().UTC()
	claims := Claims{RegisteredClaims: jwt.RegisteredClaims{
		Subject:   subject,
		Issuer:    s.issuer,
		Audience:  jwt.ClaimStrings{s.audience},
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.tokenTTL)),
	}}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func abortUnauthorized(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error": "unauthorized",
		"code":  "unauthorized",
	})
}

func tokenFromRequest(c *gin.Context) string {
	return bearerToken(c.GetHeader("Authorization"))
}

func bearerToken(authHeader string) string {
	authHeader = strings.TrimSpace(authHeader)
	if authHeader == "" {
		return ""
	}

	parts := strings.Fields(authHeader)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}

	return strings.TrimSpace(parts[1])
}
