# HarnessCompany - Self-Evolving Organization Management System

English | [简体中文](README.md)

HarnessCompany is a self-evolving organization management platform for hybrid teams of human employees and AI agents. It is built around the **ETCLOVG** framework: Execution, Tooling, Context, Lifecycle, Observability, Verification, and Governance.

## Core Ideas

- **AI agents as first-class participants**: agents share the same identity, role, and permission model as human users.
- **Decision weight engine**: a six-dimension scoring model evaluates trust and authority for automated decisions.
- **P-E-R workflow**: Planner -> Executor -> Reviewer orchestration, with risk-aware simplification.
- **MVRU execution boundary**: minimal viable reconfigurable units isolate organizational changes.
- **Self-evolution loop**: sense, learn, experiment, verify, and preserve knowledge for continuous improvement.

## Tech Stack

| Layer | Technology |
|---|---|
| Frontend | Next.js 16, React 19, TypeScript, Tailwind CSS |
| Backend | Go 1.22, modular DDD monolith, Chi Router v5 |
| Database | PostgreSQL 16 with domain-oriented schemas |
| Infrastructure | Docker Compose |

## Domain Architecture

The backend is organized into nine domains:

- `identity`: human users, AI agents, roles, and authentication
- `organization`: organizations, MVRUs, teams, members, and relationships
- `layer`: strategic, tactical, and operational layer configuration
- `capability`: capability catalog, binding, matching, and invocation metadata
- `workflow`: workflow templates, instances, tasks, decisions, and context
- `observability`: traces, spans, metrics, and execution telemetry
- `verification`: reports, review assignments, and result scoring
- `governance`: permissions, principles, and control rules
- `evolution`: decision weights, experiments, knowledge, and signals

## Quick Start

```bash
docker compose up --build
```

Services:

- PostgreSQL 16: `localhost:5432`
- Go API: `localhost:8080`
- Next.js frontend: `localhost:3000`

## Project Structure

```text
backend/          Go API, domain modules, shared packages
frontend/         Next.js App Router frontend
migrations/       SQL migrations applied by the backend
docker-compose.yml
```

## Local Development

```bash
cd backend && go test ./...
cd backend && go build ./cmd/server
cd frontend && npm install
cd frontend && npm run lint
cd frontend && npm run build
```

When running the backend outside Docker, provide PostgreSQL and set `MIGRATIONS_PATH=../migrations`.

## Configuration

Backend configuration is loaded from environment variables in `backend/internal/pkg/config/config.go`.

Key variables:

- `DATABASE_URL`
- `JWT_SECRET`
- `SERVER_PORT`
- `CORS_ORIGINS`
- `MIGRATIONS_PATH`

Frontend API calls use:

- `NEXT_PUBLIC_API_URL`

## Current Status

The repository includes the full backend API surface for all nine domains, SQL migrations, Docker Compose configuration, JWT-protected API routing, workflow transaction handling, and a basic frontend scaffold.
