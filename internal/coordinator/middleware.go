package coordinator

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/unicompute/unicompute/internal/config"
	"github.com/unicompute/unicompute/pkg/api"
)

type contextKey string

const (
	ContextKeyUserID contextKey = "user_id"
	ContextKeyRole   contextKey = "role" // "requester" or "worker"
)

// Middleware wraps HTTP handlers to provide cross-cutting concerns.
type Middleware struct {
	config config.SecurityConfig
}

// NewMiddleware initializes a new Middleware instance.
func NewMiddleware(cfg config.SecurityConfig) *Middleware {
	return &Middleware{config: cfg}
}

// RequireAuth is a middleware that verifies JWT tokens in the Authorization header.
func (m *Middleware) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			apiErr := api.NewError(api.ErrUnauthorized, "Missing Authorization header")
			http.Error(w, apiErr.Error(), apiErr.HTTPStatus)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			apiErr := api.NewError(api.ErrUnauthorized, "Invalid Authorization header format. Expected 'Bearer <token>'")
			http.Error(w, apiErr.Error(), apiErr.HTTPStatus)
			return
		}

		tokenString := parts[1]
		claims, err := m.VerifyToken(tokenString)
		if err != nil {
			apiErr := api.NewError(api.ErrInvalidToken, err.Error())
			http.Error(w, apiErr.Error(), apiErr.HTTPStatus)
			return
		}

		// Inject claims into context
		ctx := context.WithValue(r.Context(), ContextKeyUserID, claims.Subject)
		ctx = context.WithValue(ctx, ContextKeyRole, claims.Role)
		
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// GenerateToken creates a new JWT token for a given user and role.
func (m *Middleware) GenerateToken(userID, role string) (string, error) {
	if m.config.JWTSecret == "" {
		return "", fmt.Errorf("JWT secret is not configured")
	}

	claims := struct {
		Role string `json:"role"`
		jwt.RegisteredClaims
	}{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.config.JWTExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.config.JWTSecret))
}

// VerifyToken validates a token and returns its claims.
func (m *Middleware) VerifyToken(tokenString string) (*struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}, error) {
	if m.config.JWTSecret == "" {
		return nil, fmt.Errorf("JWT secret is not configured")
	}

	claims := &struct {
		Role string `json:"role"`
		jwt.RegisteredClaims
	}{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.config.JWTSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}
