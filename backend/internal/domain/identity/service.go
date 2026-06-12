package identity

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrAgentNotFound      = errors.New("agent not found")
)

type Service struct {
	repo      *Repository
	jwtSecret string
}

func NewService(repo *Repository, jwtSecret string) *Service {
	return &Service{repo: repo, jwtSecret: jwtSecret}
}

type AuthResponse struct {
	Token     string `json:"token"`
	UserID    string `json:"user_id"`
	UserType  string `json:"user_type"`
	ExpiresAt int64  `json:"expires_at"`
}

type RegisterAgentResponse struct {
	Agent  *AIAgent `json:"agent"`
	APIKey string   `json:"api_key"`
}

func (s *Service) AuthenticateUser(ctx context.Context, email, password string) (*AuthResponse, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, expiresAt, err := s.generateJWT(user.ID.String(), "human", user.Email)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &AuthResponse{
		Token:     token,
		UserID:    user.ID.String(),
		UserType:  "human",
		ExpiresAt: expiresAt,
	}, nil
}

func (s *Service) AuthenticateAgent(ctx context.Context, agentID uuid.UUID, apiKey string) (*AuthResponse, error) {
	agent, err := s.repo.GetAgentByID(ctx, agentID)
	if err != nil {
		return nil, ErrAgentNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(agent.APIKeyHash), []byte(apiKey)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, expiresAt, err := s.generateJWT(agent.ID.String(), "ai", agent.Name)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &AuthResponse{
		Token:     token,
		UserID:    agent.ID.String(),
		UserType:  "ai",
		ExpiresAt: expiresAt,
	}, nil
}

func (s *Service) RegisterUser(ctx context.Context, input CreateUserInput) (*User, error) {
	return s.repo.CreateUser(ctx, input)
}

func (s *Service) RegisterAgent(ctx context.Context, input CreateAgentInput) (*RegisterAgentResponse, error) {
	agent, apiKey, err := s.repo.CreateAgent(ctx, input)
	if err != nil {
		return nil, err
	}
	return &RegisterAgentResponse{Agent: agent, APIKey: apiKey}, nil
}

func (s *Service) ValidateToken(tokenString string) (string, string, error) {
	token, err := s.parseJWT(tokenString)
	if err != nil {
		return "", "", err
	}
	userID, ok := token["sub"].(string)
	if !ok {
		return "", "", fmt.Errorf("invalid token: missing subject")
	}
	userType, ok := token["type"].(string)
	if !ok {
		return "", "", fmt.Errorf("invalid token: missing type")
	}
	return userID, userType, nil
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

func (s *Service) generateJWT(subject, userType, identifier string) (string, int64, error) {
	expiresAt := time.Now().Add(24 * time.Hour)
	expiresAtUnix := expiresAt.Unix()

	header := jwtHeader{Alg: "HS256", Typ: "JWT"}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", 0, fmt.Errorf("marshal header: %w", err)
	}

	payload := map[string]any{
		"sub":  subject,
		"type": userType,
		"email": identifier,
		"exp":  expiresAtUnix,
		"iat":  time.Now().Unix(),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", 0, fmt.Errorf("marshal payload: %w", err)
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

	signingInput := headerB64 + "." + payloadB64
	mac := hmac.New(sha256.New, []byte(s.jwtSecret))
	mac.Write([]byte(signingInput))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signingInput + "." + signature, expiresAtUnix, nil
}

func (s *Service) parseJWT(tokenString string) (map[string]any, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	headerB64, payloadB64, sigB64 := parts[0], parts[1], parts[2]

	mac := hmac.New(sha256.New, []byte(s.jwtSecret))
	mac.Write([]byte(headerB64 + "." + payloadB64))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(sigB64), []byte(expectedSig)) {
		return nil, fmt.Errorf("invalid token signature")
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}

	var claims map[string]any
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", err)
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid token: missing exp")
	}
	if time.Now().Unix() > int64(exp) {
		return nil, fmt.Errorf("token expired")
	}

	return claims, nil
}
