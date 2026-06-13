package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

type contextKey string

const UserContextKey contextKey = "user"

type AuthenticatedUser struct {
	ID   string
	Type string
	Name string
}

func UserFromContext(ctx context.Context) (AuthenticatedUser, bool) {
	user, ok := ctx.Value(UserContextKey).(AuthenticatedUser)
	return user, ok
}

func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, err := authenticateBearer(r.Header.Get("Authorization"), jwtSecret)
			if err != nil {
				writeUnauthorized(w)
				return
			}
			ctx := context.WithValue(r.Context(), UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func authenticateBearer(authHeader string, jwtSecret string) (AuthenticatedUser, error) {
	if jwtSecret == "" {
		return AuthenticatedUser{}, errors.New("jwt secret is required")
	}

	parts := strings.Fields(authHeader)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return AuthenticatedUser{}, errors.New("bearer token is required")
	}

	return validateJWT(parts[1], jwtSecret)
}

func validateJWT(tokenString string, jwtSecret string) (AuthenticatedUser, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return AuthenticatedUser{}, errors.New("invalid token format")
	}

	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return AuthenticatedUser{}, errors.New("invalid token header")
	}
	var header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return AuthenticatedUser{}, errors.New("invalid token header")
	}
	if header.Alg != "HS256" || header.Typ != "JWT" {
		return AuthenticatedUser{}, errors.New("unsupported token header")
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return AuthenticatedUser{}, errors.New("invalid token signature")
	}
	mac := hmac.New(sha256.New, []byte(jwtSecret))
	mac.Write([]byte(parts[0] + "." + parts[1]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return AuthenticatedUser{}, errors.New("invalid token signature")
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return AuthenticatedUser{}, errors.New("invalid token payload")
	}
	var claims map[string]any
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return AuthenticatedUser{}, errors.New("invalid token payload")
	}

	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return AuthenticatedUser{}, errors.New("missing token subject")
	}
	userType, ok := claims["type"].(string)
	if !ok || userType == "" {
		return AuthenticatedUser{}, errors.New("missing token type")
	}
	exp, ok := claims["exp"].(float64)
	if !ok {
		return AuthenticatedUser{}, errors.New("missing token expiry")
	}
	if time.Now().Unix() >= int64(exp) {
		return AuthenticatedUser{}, errors.New("token expired")
	}
	name, _ := claims["name"].(string)

	return AuthenticatedUser{ID: sub, Type: userType, Name: name}, nil
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
}
