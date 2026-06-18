# Phase 1: Foundation — Project Scaffolding + Identity Domain

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Scaffold the complete project monorepo and build Identity Domain (users, AI agents, roles, permissions) — the foundation all other Domains depend on.

**Architecture:** Modular monolith with Go backend (DDD packages) + Next.js frontend + PostgreSQL. Go API Gateway as unified entry point. Identity Domain provides auth middleware for all future Domains.

**Tech Stack:** Go 1.22+, Next.js 14 (App Router), PostgreSQL 16, Docker Compose

---

### Task 1: Project scaffolding — Go backend skeleton

**Files:**
- Create: `backend/go.mod`
- Create: `backend/cmd/server/main.go`
- Create: `backend/internal/pkg/config/config.go`
- Create: `backend/internal/pkg/database/postgres.go`
- Create: `backend/internal/pkg/server/server.go`
- Create: `backend/internal/pkg/middleware/auth.go` (stub)
- Create: `backend/internal/gateway/router.go`

- [ ] **Step 1: Initialize Go module**

```bash
mkdir -p /root/HarnessCompany/backend/cmd/server
mkdir -p /root/HarnessCompany/backend/internal/pkg/config
mkdir -p /root/HarnessCompany/backend/internal/pkg/database
mkdir -p /root/HarnessCompany/backend/internal/pkg/server
mkdir -p /root/HarnessCompany/backend/internal/pkg/middleware
mkdir -p /root/HarnessCompany/backend/internal/gateway
mkdir -p /root/HarnessCompany/backend/internal/domain/identity
```

- [ ] **Step 2: Create go.mod**

```
module github.com/harness-org/backend

go 1.22

require (
	github.com/go-chi/chi/v5 v5.1.0
	github.com/go-chi/cors v1.2.1
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.7.1
	github.com/golang-jwt/jwt/v5 v5.2.1
	golang.org/x/crypto v0.28.0
)
```

- [ ] **Step 3: Create config package**

```go
package config

import (
	"os"
	"strconv"
)

type Config struct {
	ServerPort    int
	DatabaseURL   string
	JWTSecret     string
	CorsOrigins   string
}

func Load() *Config {
	return &Config{
		ServerPort:    getEnvInt("SERVER_PORT", 8080),
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/harness_org?sslmode=disable"),
		JWTSecret:     getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		CorsOrigins:   getEnv("CORS_ORIGINS", "http://localhost:3000"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
```

- [ ] **Step 4: Create database package**

```go
package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}
	return pool, nil
}
```

- [ ] **Step 5: Create server package**

```go
package server

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func New(router *chi.Mux, port int) *http.Server {
	return &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}
}

func NewRouter(corsOrigins string) *chi.Mux {
	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{corsOrigins},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))
	return r
}
```

- [ ] **Step 6: Create gateway router with health check**

```go
package gateway

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r *chi.Mux) {
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", healthCheck)
	})
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
```

- [ ] **Step 7: Create main.go**

```go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/harness-org/backend/internal/gateway"
	"github.com/harness-org/backend/internal/pkg/config"
	"github.com/harness-org/backend/internal/pkg/database"
	"github.com/harness-org/backend/internal/pkg/server"
)

func main() {
	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()

	router := server.NewRouter(cfg.CorsOrigins)
	gateway.RegisterRoutes(router)

	srv := server.New(router, cfg.ServerPort)
	go func() {
		log.Printf("server starting on :%d", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	srv.Shutdown(shutdownCtx)
}
```

- [ ] **Step 8: Verify build**

Run: `cd /root/HarnessCompany/backend && go mod tidy && go build ./cmd/server/`
Expected: binary builds without errors

- [ ] **Step 9: Commit**

```bash
cd /root/HarnessCompany && git add backend/ && git commit -m "feat: scaffold Go backend with chi router, postgres, health check"
```

---

### Task 2: PostgreSQL schema — Identity tables

**Files:**
- Create: `migrations/001_identity.sql`

- [ ] **Step 1: Create identity migration**

```sql
-- 001_identity.sql

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TYPE role_type AS ENUM ('planner', 'executor', 'reviewer');
CREATE TYPE permission_level AS ENUM ('L1', 'L2', 'L3', 'L4');

CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            VARCHAR(255) NOT NULL,
    email           VARCHAR(255) UNIQUE NOT NULL,
    password_hash   VARCHAR(255) NOT NULL,
    avatar_url      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE ai_agents (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            VARCHAR(255) NOT NULL,
    model_type      VARCHAR(100) NOT NULL,
    api_key_hash    VARCHAR(255) NOT NULL,
    capabilities    JSONB NOT NULL DEFAULT '[]',
    permission_level permission_level NOT NULL DEFAULT 'L2',
    metadata        JSONB DEFAULT '{}',
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE roles (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            VARCHAR(100) NOT NULL UNIQUE,
    role_type       role_type NOT NULL,
    description     TEXT,
    permissions     JSONB NOT NULL DEFAULT '[]',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE user_roles (
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id         UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE agent_roles (
    agent_id        UUID NOT NULL REFERENCES ai_agents(id) ON DELETE CASCADE,
    role_id         UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (agent_id, role_id)
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_agents_model ON ai_agents(model_type);
```

- [ ] **Step 2: Create migration runner in Go**

Create: `backend/internal/pkg/database/migrate.go`

```go
package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RunMigrations(ctx context.Context, pool *pgxpool.Pool, migrationsDir string) error {
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var sqlFiles []string
	for _, f := range files {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".sql" {
			sqlFiles = append(sqlFiles, f.Name())
		}
	}
	sort.Strings(sqlFiles)

	for _, f := range sqlFiles {
		content, err := os.ReadFile(filepath.Join(migrationsDir, f))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", f, err)
		}
		if _, err := pool.Exec(ctx, string(content)); err != nil {
			return fmt.Errorf("execute migration %s: %w", f, err)
		}
		fmt.Printf("migration applied: %s\n", f)
	}
	return nil
}
```

- [ ] **Step 3: Call migrations from main.go**

Edit `backend/cmd/server/main.go` — add after db connection:

```go
if err := database.RunMigrations(ctx, db, "migrations"); err != nil {
	log.Fatalf("migrations failed: %v", err)
}
```

- [ ] **Step 4: Commit**

```bash
cd /root/HarnessCompany && git add migrations/ backend/internal/pkg/database/migrate.go backend/cmd/server/main.go && git commit -m "feat: add identity schema and migration runner"
```

---

### Task 3: Identity Domain — Go models and repository

**Files:**
- Create: `backend/internal/domain/identity/model.go`
- Create: `backend/internal/domain/identity/repository.go`

- [ ] **Step 1: Create identity models**

```go
package identity

import (
	"time"

	"github.com/google/uuid"
)

type RoleType string

const (
	RolePlanner  RoleType = "planner"
	RoleExecutor RoleType = "executor"
	RoleReviewer RoleType = "reviewer"
)

type PermissionLevel string

const (
	PermissionL1 PermissionLevel = "L1"
	PermissionL2 PermissionLevel = "L2"
	PermissionL3 PermissionLevel = "L3"
	PermissionL4 PermissionLevel = "L4"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	AvatarURL    string    `json:"avatar_url,omitempty"`
	Roles        []Role    `json:"roles,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type AIAgent struct {
	ID              uuid.UUID       `json:"id"`
	Name            string          `json:"name"`
	ModelType       string          `json:"model_type"`
	APIKeyHash      string          `json:"-"`
	Capabilities    []string        `json:"capabilities"`
	PermissionLevel PermissionLevel `json:"permission_level"`
	Metadata        map[string]any  `json:"metadata,omitempty"`
	IsActive        bool            `json:"is_active"`
	Roles           []Role          `json:"roles,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type Role struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	RoleType    RoleType        `json:"role_type"`
	Description string          `json:"description,omitempty"`
	Permissions []string        `json:"permissions"`
}

type CreateUserInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type CreateAgentInput struct {
	Name            string          `json:"name"`
	ModelType       string          `json:"model_type"`
	Capabilities    []string        `json:"capabilities"`
	PermissionLevel PermissionLevel `json:"permission_level"`
	Metadata        map[string]any  `json:"metadata,omitempty"`
}
```

- [ ] **Step 2: Create identity repository**

```go
package identity

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateUser(ctx context.Context, input CreateUserInput) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &User{}
	err = r.db.QueryRow(ctx,
		`INSERT INTO users (name, email, password_hash) VALUES ($1, $2, $3)
		 RETURNING id, name, email, password_hash, avatar_url, created_at, updated_at`,
		input.Name, input.Email, string(hash),
	).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}
	err := r.db.QueryRow(ctx,
		`SELECT id, name, email, password_hash, avatar_url, created_at, updated_at
		 FROM users WHERE email = $1`, email,
	).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return user, nil
}

func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	user := &User{}
	err := r.db.QueryRow(ctx,
		`SELECT id, name, email, password_hash, avatar_url, created_at, updated_at
		 FROM users WHERE id = $1`, id,
	).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return user, nil
}

func (r *Repository) CreateAgent(ctx context.Context, input CreateAgentInput) (*AIAgent, error) {
	keyHash, err := bcrypt.GenerateFromPassword([]byte(uuid.New().String()), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("generate api key: %w", err)
	}

	capJSON, _ := json.Marshal(input.Capabilities)
	metaJSON, _ := json.Marshal(input.Metadata)

	agent := &AIAgent{}
	err = r.db.QueryRow(ctx,
		`INSERT INTO ai_agents (name, model_type, api_key_hash, capabilities, permission_level, metadata)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, name, model_type, api_key_hash, capabilities, permission_level, metadata, is_active, created_at, updated_at`,
		input.Name, input.ModelType, string(keyHash), capJSON, input.PermissionLevel, metaJSON,
	).Scan(&agent.ID, &agent.Name, &agent.ModelType, &agent.APIKeyHash, &capJSON, &agent.PermissionLevel, &metaJSON, &agent.IsActive, &agent.CreatedAt, &agent.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create agent: %w", err)
	}
	return agent, nil
}

func (r *Repository) GetAgentByID(ctx context.Context, id uuid.UUID) (*AIAgent, error) {
	agent := &AIAgent{}
	var capJSON, metaJSON []byte
	err := r.db.QueryRow(ctx,
		`SELECT id, name, model_type, api_key_hash, capabilities, permission_level, metadata, is_active, created_at, updated_at
		 FROM ai_agents WHERE id = $1`, id,
	).Scan(&agent.ID, &agent.Name, &agent.ModelType, &agent.APIKeyHash, &capJSON, &agent.PermissionLevel, &metaJSON, &agent.IsActive, &agent.CreatedAt, &agent.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get agent by id: %w", err)
	}
	json.Unmarshal(capJSON, &agent.Capabilities)
	json.Unmarshal(metaJSON, &agent.Metadata)
	return agent, nil
}

func (r *Repository) ListRoles(ctx context.Context) ([]Role, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, role_type, description, permissions FROM roles ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var role Role
		var permJSON []byte
		if err := rows.Scan(&role.ID, &role.Name, &role.RoleType, &role.Description, &permJSON); err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		json.Unmarshal(permJSON, &role.Permissions)
		roles = append(roles, role)
	}
	return roles, nil
}
```

- [ ] **Step 3: Commit**

```bash
cd /root/HarnessCompany && git add backend/internal/domain/identity/model.go backend/internal/domain/identity/repository.go && git commit -m "feat: identity domain models and postgres repository"
```

---

### Task 4: Identity Domain — Service layer with auth logic

**Files:**
- Create: `backend/internal/domain/identity/service.go`

- [ ] **Step 1: Create identity service**

```go
package identity

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
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
	UserType  string `json:"user_type"` // "human" or "ai"
	ExpiresAt int64  `json:"expires_at"`
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

func (s *Service) RegisterAgent(ctx context.Context, input CreateAgentInput) (*AIAgent, error) {
	return s.repo.CreateAgent(ctx, input)
}

func (s *Service) ValidateToken(tokenString string) (string, string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil {
		return "", "", fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", "", fmt.Errorf("invalid token")
	}

	userID := claims["sub"].(string)
	userType := claims["type"].(string)
	return userID, userType, nil
}

func (s *Service) generateJWT(subject, userType, identifier string) (string, int64, error) {
	expiresAt := time.Now().Add(24 * time.Hour)
	claims := jwt.MapClaims{
		"sub":  subject,
		"type": userType,
		"email": identifier,
		"exp":  expiresAt.Unix(),
		"iat":  time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", 0, err
	}
	return tokenString, expiresAt.Unix(), nil
}
```

- [ ] **Step 2: Commit**

```bash
cd /root/HarnessCompany && git add backend/internal/domain/identity/service.go && git commit -m "feat: identity service with JWT auth and registration"
```

---

### Task 5: Identity Domain — HTTP handlers and API routes

**Files:**
- Create: `backend/internal/domain/identity/handler.go`
- Modify: `backend/internal/gateway/router.go`

- [ ] **Step 1: Create identity HTTP handlers**

```go
package identity

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/auth/login", h.login)
	r.Post("/auth/register", h.register)
	r.Post("/agents/register", h.registerAgent)
	r.Post("/agents/auth", h.authenticateAgent)
	r.Get("/roles", h.listRoles)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	resp, err := h.service.AuthenticateUser(r.Context(), req.Email, req.Password)
	if err != nil {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	user, err := h.service.RegisterUser(r.Context(), CreateUserInput{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		http.Error(w, `{"error":"registration failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (h *Handler) registerAgent(w http.ResponseWriter, r *http.Request) {
	var input CreateAgentInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	agent, err := h.service.RegisterAgent(r.Context(), input)
	if err != nil {
		http.Error(w, `{"error":"agent registration failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(agent)
}

func (h *Handler) authenticateAgent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AgentID string `json:"agent_id"`
		APIKey  string `json:"api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	agentID, err := uuid.Parse(req.AgentID)
	if err != nil {
		http.Error(w, `{"error":"invalid agent id"}`, http.StatusBadRequest)
		return
	}

	resp, err := h.service.AuthenticateAgent(r.Context(), agentID, req.APIKey)
	if err != nil {
		http.Error(w, `{"error":"authentication failed"}`, http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) listRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.service.repo.ListRoles(r.Context())
	if err != nil {
		http.Error(w, `{"error":"failed to list roles"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(roles)
}
```

- [ ] **Step 2: Update gateway router to register identity routes**

Edit `backend/internal/gateway/router.go`:

```go
package gateway

import (
	"github.com/go-chi/chi/v5"
	"github.com/harness-org/backend/internal/domain/identity"
)

type Dependencies struct {
	IdentityHandler *identity.Handler
}

func RegisterRoutes(r *chi.Mux, deps *Dependencies) {
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", healthCheck)
		if deps != nil && deps.IdentityHandler != nil {
			deps.IdentityHandler.RegisterRoutes(r)
		}
	})
}
```

- [ ] **Step 3: Update main.go to wire dependencies**

Edit `backend/cmd/server/main.go`:

```go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/harness-org/backend/internal/domain/identity"
	"github.com/harness-org/backend/internal/gateway"
	"github.com/harness-org/backend/internal/pkg/config"
	"github.com/harness-org/backend/internal/pkg/database"
	"github.com/harness-org/backend/internal/pkg/server"
)

func main() {
	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()

	if err := database.RunMigrations(ctx, db, "migrations"); err != nil {
		log.Fatalf("migrations failed: %v", err)
	}

	identRepo := identity.NewRepository(db)
	identSvc := identity.NewService(identRepo, cfg.JWTSecret)
	identHandler := identity.NewHandler(identSvc)

	router := server.NewRouter(cfg.CorsOrigins)
	gateway.RegisterRoutes(router, &gateway.Dependencies{
		IdentityHandler: identHandler,
	})

	srv := server.New(router, cfg.ServerPort)
	go func() {
		log.Printf("server starting on :%d", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	srv.Shutdown(shutdownCtx)
}
```

- [ ] **Step 4: Verify build**

Run: `cd /root/HarnessCompany/backend && go build ./cmd/server/`
Expected: binary builds without errors

- [ ] **Step 5: Commit**

```bash
cd /root/HarnessCompany && git add backend/internal/domain/identity/handler.go backend/internal/gateway/router.go backend/cmd/server/main.go && git commit -m "feat: identity API routes with login, register, agent auth"
```

---

### Task 6: Next.js frontend scaffolding

**Files:**
- Create: `frontend/package.json`
- Create: `frontend/tsconfig.json`
- Create: `frontend/next.config.js`
- Create: `frontend/tailwind.config.ts`
- Create: `frontend/postcss.config.js`
- Create: `frontend/src/app/layout.tsx`
- Create: `frontend/src/app/page.tsx`
- Create: `frontend/src/app/globals.css`
- Create: `frontend/src/lib/api.ts`
- Create: `frontend/src/lib/auth.ts`

- [ ] **Step 1: Create project files**

```bash
mkdir -p /root/HarnessCompany/frontend/src/app
mkdir -p /root/HarnessCompany/frontend/src/lib
mkdir -p /root/HarnessCompany/frontend/src/components
mkdir -p /root/HarnessCompany/frontend/public
```

- [ ] **Step 2: Create package.json**

```json
{
  "name": "harness-org-frontend",
  "version": "0.1.0",
  "private": true,
  "scripts": {
    "dev": "next dev",
    "build": "next build",
    "start": "next start",
    "lint": "next lint"
  },
  "dependencies": {
    "next": "^14.2.0",
    "react": "^18.3.0",
    "react-dom": "^18.3.0",
    "lucide-react": "^0.400.0"
  },
  "devDependencies": {
    "@types/node": "^20.0.0",
    "@types/react": "^18.3.0",
    "@types/react-dom": "^18.3.0",
    "typescript": "^5.4.0",
    "tailwindcss": "^3.4.0",
    "postcss": "^8.4.0",
    "autoprefixer": "^10.4.0"
  }
}
```

- [ ] **Step 3: Create tsconfig.json**

```json
{
  "compilerOptions": {
    "target": "ES2017",
    "lib": ["dom", "dom.iterable", "esnext"],
    "allowJs": true,
    "skipLibCheck": true,
    "strict": true,
    "noEmit": true,
    "esModuleInterop": true,
    "module": "esnext",
    "moduleResolution": "bundler",
    "resolveJsonModule": true,
    "isolatedModules": true,
    "jsx": "preserve",
    "incremental": true,
    "plugins": [{ "name": "next" }],
    "paths": { "@/*": ["./src/*"] }
  },
  "include": ["next-env.d.ts", "**/*.ts", "**/*.tsx", ".next/types/**/*.ts"],
  "exclude": ["node_modules"]
}
```

- [ ] **Step 4: Create next.config.js**

```js
/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'standalone',
}

module.exports = nextConfig
```

- [ ] **Step 5: Create tailwind.config.ts**

```ts
import type { Config } from 'tailwindcss'

const config: Config = {
  content: ['./src/**/*.{js,ts,jsx,tsx,mdx}'],
  theme: {
    extend: {
      colors: {
        primary: { 50: '#eff6ff', 100: '#dbeafe', 200: '#bfdbfe', 300: '#93c5fd', 400: '#60a5fa', 500: '#3b82f6', 600: '#2563eb', 700: '#1d4ed8', 800: '#1e40af', 900: '#1e3a8a' },
      },
    },
  },
  plugins: [],
}
export default config
```

- [ ] **Step 6: Create postcss.config.js**

```js
module.exports = {
  plugins: {
    tailwindcss: {},
    autoprefixer: {},
  },
}
```

- [ ] **Step 7: Create globals.css**

```css
@tailwind base;
@tailwind components;
@tailwind utilities;

:root {
  --foreground: #0f172a;
  --background: #f8fafc;
}

body {
  color: var(--foreground);
  background: var(--background);
}
```

- [ ] **Step 8: Create API client**

```typescript
// src/lib/api.ts
const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1'

interface RequestOptions {
  method?: string
  body?: unknown
  token?: string
}

export async function apiRequest<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }

  if (options.token) {
    headers['Authorization'] = `Bearer ${options.token}`
  }

  const response = await fetch(`${API_BASE}${path}`, {
    method: options.method || 'GET',
    headers,
    body: options.body ? JSON.stringify(options.body) : undefined,
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Unknown error' }))
    throw new Error(error.error || `HTTP ${response.status}`)
  }

  return response.json()
}

export interface AuthResponse {
  token: string
  user_id: string
  user_type: 'human' | 'ai'
  expires_at: number
}

export interface User {
  id: string
  name: string
  email: string
  avatar_url?: string
  created_at: string
  updated_at: string
}

export interface AIAgent {
  id: string
  name: string
  model_type: string
  capabilities: string[]
  permission_level: string
  metadata: Record<string, unknown>
  is_active: boolean
  created_at: string
  updated_at: string
}
```

- [ ] **Step 9: Create auth lib**

```typescript
// src/lib/auth.ts
const TOKEN_KEY = 'harness_token'
const USER_KEY = 'harness_user'

export function setSession(token: string, userId: string, userType: string): void {
  if (typeof window === 'undefined') return
  localStorage.setItem(TOKEN_KEY, token)
  localStorage.setItem(USER_KEY, JSON.stringify({ id: userId, type: userType }))
}

export function getToken(): string | null {
  if (typeof window === 'undefined') return null
  return localStorage.getItem(TOKEN_KEY)
}

export function clearSession(): void {
  if (typeof window === 'undefined') return
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(USER_KEY)
}

export function isAuthenticated(): boolean {
  return !!getToken()
}
```

- [ ] **Step 10: Create root layout**

```tsx
// src/app/layout.tsx
import type { Metadata } from 'next'
import './globals.css'

export const metadata: Metadata = {
  title: 'Harness Organization System',
  description: 'Self-evolving organizational management platform',
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="zh-CN">
      <body className="min-h-screen bg-slate-50 text-slate-900">
        {children}
      </body>
    </html>
  )
}
```

- [ ] **Step 11: Create home page**

```tsx
// src/app/page.tsx
export default function Home() {
  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="text-center space-y-4">
        <h1 className="text-4xl font-bold text-slate-900">Harness Organization</h1>
        <p className="text-lg text-slate-500">Self-evolving organizational management platform</p>
        <div className="flex gap-4 justify-center pt-4">
          <a href="/login" className="px-6 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 transition">
            Sign In
          </a>
          <a href="/register" className="px-6 py-2 border border-slate-300 rounded-lg hover:bg-slate-100 transition">
            Register
          </a>
        </div>
      </div>
    </div>
  )
}
```

- [ ] **Step 12: Verify install**

Run: `cd /root/HarnessCompany/frontend && npm install && npm run build`
Expected: Next.js builds successfully

- [ ] **Step 13: Commit**

```bash
cd /root/HarnessCompany && git add frontend/ && git commit -m "feat: scaffold Next.js frontend with API client and auth"
```

---

### Task 7: Docker Compose and seed data

**Files:**
- Create: `docker-compose.yml`
- Create: `migrations/002_seed_roles.sql`

- [ ] **Step 1: Create docker-compose.yml**

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: harness_org
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data

  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      SERVER_PORT: "8080"
      DATABASE_URL: "postgres://postgres:postgres@postgres:5432/harness_org?sslmode=disable"
      JWT_SECRET: "dev-secret-change-in-production"
      CORS_ORIGINS: "http://localhost:3000"
    depends_on:
      - postgres

  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
    ports:
      - "3000:3000"
    environment:
      NEXT_PUBLIC_API_URL: "http://localhost:8080/api/v1"
    depends_on:
      - backend

volumes:
  pgdata:
```

- [ ] **Step 2: Create seed roles migration**

```sql
-- 002_seed_roles.sql

INSERT INTO roles (name, role_type, description, permissions) VALUES
  ('Strategic Planner', 'planner', 'C-suite and strategic decision makers', '["org:read","org:write","strategy:full","governance:full"]'),
  ('Tactical Planner', 'planner', 'MVRU leads and product managers', '["mvru:read","mvru:write","workflow:full","capability:read"]'),
  ('AI Planner', 'planner', 'AI agents responsible for planning', '["mvru:read","workflow:read","capability:read"]'),
  ('Human Executor', 'executor', 'Human team members executing tasks', '["task:read","task:write","capability:use"]'),
  ('AI Executor', 'executor', 'AI agents executing defined tasks', '["task:read","task:write","capability:use"]'),
  ('Independent Reviewer', 'reviewer', 'Independent reviewers (human)', '["review:full","verification:read"]'),
  ('AI Reviewer', 'reviewer', 'AI agents performing automated review', '["review:limited","verification:read"]')
ON CONFLICT (name) DO NOTHING;
```

- [ ] **Step 3: Commit**

```bash
cd /root/HarnessCompany && git add docker-compose.yml migrations/002_seed_roles.sql && git commit -m "feat: docker-compose setup and seed roles migration"
```

---

### Task 8: Create Go Dockerfile

- [ ] **Step 1: Create backend Dockerfile**

Create: `backend/Dockerfile`

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server/

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /server .
COPY migrations/ ./migrations/
EXPOSE 8080
CMD ["./server"]
```

- [ ] **Step 2: Create frontend Dockerfile**

Create: `frontend/Dockerfile`

```dockerfile
FROM node:20-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:20-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
COPY --from=builder /app/public ./public
EXPOSE 3000
CMD ["node", "server.js"]
```

- [ ] **Step 3: Commit**

```bash
cd /root/HarnessCompany && git add backend/Dockerfile frontend/Dockerfile && git commit -m "feat: Dockerfiles for backend and frontend"
```

