package identity

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrAgentNotFound      = errors.New("agent not found")
	ErrValidation         = errors.New("validation error")
)

type UserRepository interface {
	CreateUser(ctx context.Context, input CreateUserInput) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
	CreateAgent(ctx context.Context, input CreateAgentInput) (*AIAgent, string, error)
	GetAgentByID(ctx context.Context, id uuid.UUID) (*AIAgent, error)
	ListAgents(ctx context.Context, limit int) ([]AIAgent, error)
	ListRoles(ctx context.Context) ([]Role, error)
}

type Service struct {
	repo      UserRepository
	jwtSecret string
	tokenTTL  time.Duration
}

type ServiceOption func(*Service)

func WithTokenTTL(ttl time.Duration) ServiceOption {
	return func(s *Service) {
		s.tokenTTL = ttl
	}
}

func NewService(repo UserRepository, jwtSecret string, opts ...ServiceOption) *Service {
	s := &Service{repo: repo, jwtSecret: jwtSecret, tokenTTL: 24 * time.Hour}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

type AuthResponse struct {
	Token     string `json:"token"`
	UserID    string `json:"user_id"`
	UserType  string `json:"user_type"`
	ExpiresAt int64  `json:"expires_at"`
}

type RegisterAgentResponse struct {
	Agent  AIAgent `json:"agent"`
	APIKey string  `json:"api_key"`
}

func (s *Service) AuthenticateUser(ctx context.Context, email, password string) (*AuthResponse, error) {
	if email == "" || password == "" {
		return nil, fmt.Errorf("%w: email and password are required", ErrValidation)
	}

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		log.Printf("authenticate user: %v", err)
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		log.Printf("password mismatch for user %s: %v", email, err)
		return nil, ErrInvalidCredentials
	}

	token, expiresAt, err := s.generateJWT(user.ID.String(), "human", user.Name)
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
	if apiKey == "" {
		return nil, fmt.Errorf("%w: api key is required", ErrValidation)
	}

	agent, err := s.repo.GetAgentByID(ctx, agentID)
	if err != nil {
		log.Printf("authenticate agent: %v", err)
		return nil, ErrInvalidCredentials
	}

	if !agent.IsActive {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(agent.APIKeyHash), []byte(apiKey)); err != nil {
		log.Printf("api key mismatch for agent %s: %v", agentID, err)
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

func (s *Service) RegisterUser(ctx context.Context, input CreateUserInput) (*UserResponse, error) {
	if input.Name == "" || input.Email == "" || input.Password == "" {
		return nil, fmt.Errorf("%w: name, email, and password are required", ErrValidation)
	}
	user, err := s.repo.CreateUser(ctx, input)
	if err != nil {
		return nil, err
	}
	resp := NewUserResponse(user)
	return &resp, nil
}

func (s *Service) RegisterAgent(ctx context.Context, input CreateAgentInput) (*RegisterAgentResponse, error) {
	if input.Name == "" || input.ModelType == "" {
		return nil, fmt.Errorf("%w: name and model_type are required", ErrValidation)
	}
	if input.PermissionLevel == "" {
		input.PermissionLevel = PermissionL2
	}
	if input.AgentOrigin == "" {
		input.AgentOrigin = "internal"
	}
	if input.AgentOrigin != "internal" && input.AgentOrigin != "external" {
		return nil, fmt.Errorf("%w: invalid agent_origin", ErrValidation)
	}
	if input.ServiceClass == "" {
		input.ServiceClass = "model"
	}
	if input.RiskLevel == "" {
		input.RiskLevel = "medium"
	}
	if !isValidRiskLevel(input.RiskLevel) {
		return nil, fmt.Errorf("%w: invalid risk_level", ErrValidation)
	}
	agent, apiKey, err := s.repo.CreateAgent(ctx, input)
	if err != nil {
		return nil, err
	}
	return &RegisterAgentResponse{Agent: *agent, APIKey: apiKey}, nil
}

func (s *Service) ListAgents(ctx context.Context, limit int) ([]AIAgent, error) {
	return s.repo.ListAgents(ctx, limit)
}

func (s *Service) ListRoles(ctx context.Context) ([]Role, error) {
	return s.repo.ListRoles(ctx)
}

func (s *Service) ValidateToken(tokenString string) (string, string, error) {
	claims, err := s.parseJWT(tokenString)
	if err != nil {
		return "", "", err
	}
	userID, ok := claims["sub"].(string)
	if !ok {
		return "", "", fmt.Errorf("invalid token: missing subject")
	}
	userType, ok := claims["type"].(string)
	if !ok {
		return "", "", fmt.Errorf("invalid token: missing type")
	}
	return userID, userType, nil
}

func isValidRiskLevel(level string) bool {
	switch level {
	case "low", "medium", "high", "critical":
		return true
	default:
		return false
	}
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

func (s *Service) generateJWT(subject, userType, name string) (string, int64, error) {
	expiresAt := time.Now().Add(s.tokenTTL)
	expiresAtUnix := expiresAt.Unix()

	header := jwtHeader{Alg: "HS256", Typ: "JWT"}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", 0, fmt.Errorf("marshal header: %w", err)
	}

	payload := map[string]any{
		"sub":  subject,
		"type": userType,
		"name": name,
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
