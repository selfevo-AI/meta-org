# HarnessCompany - 自进化组织管理系统

[English](README_EN.md) | 简体中文

一个面向混合人力（人类员工 + AI Agent）的自进化组织管理平台，基于 **ETCLOVG** 框架（Execution, Tooling, Context, Lifecycle, Observability, Verification, Governance）构建。

## 核心思想

- **AI Agent 与人类一视同仁**：Agent 作为一等公民参与组织运作，拥有独立的身份、角色和权限体系
- **决策权重引擎**：六维算法动态评估每个参与者的信任/权威分数，驱动自动化决策
- **P-E-R 工作流**：Planner → Executor → Reviewer 三阶段编排，根据风险/复杂度动态简化
- **MVRU 沙箱执行**：最小可重组单元，在隔离环境中安全执行组织变更
- **自进化闭环**：感知 → 学习 → 实验 → 验证 → 知识沉淀，持续优化组织运行

## 技术栈

| 层 | 技术 |
|---|---|
| 前端 | Next.js 16 (App Router, React 19, TypeScript, Tailwind CSS) |
| 后端 | Go 1.22 (DDD 模块化单体, Chi Router v5) |
| 数据库 | PostgreSQL 16 (多 schema 域隔离) |
| 容器化 | Docker Compose |

## 领域架构（9 大域）

```
┌─────────────────────────────────────────────────────────────┐
│                       Evolution (自进化)                       │
│  决策权重引擎 · 感知引擎 · 学习引擎 · 知识引擎                   │
├──────────┬──────────┬──────────┬──────────┬──────────────────┤
│Identity  │Organizat │  Layer   │Capability│   Workflow       │
│(身份)    │ion(组织) │ (分层)   │ (能力)   │  (工作流)         │
├──────────┴──────────┴──────────┴──────────┴──────────────────┤
│   Observability (可观测) · Verification (验证) · Governance (治理) │
└──────────────────────────────────────────────────────────────┘
```

## 快速开始

```bash
docker compose up --build
```

启动后：
- PostgreSQL 16 → `:5432`
- Go 后端 → `:8080`
- Next.js 前端 → `:3000`

## 项目结构

```
backend/          # Go 后端 (9 域 handler/service/repository)
frontend/         # Next.js 前端
migrations/       # 10 个 SQL 迁移文件
docker-compose.yml
```

## 配置

通过环境变量配置，详见 `backend/internal/pkg/config/config.go`。

关键变量：`DATABASE_URL`, `JWT_SECRET`, `SERVER_PORT`, `CORS_ORIGINS`。

## 开发状态

全部 9 个领域已完成实现，包含完整的后端 API、SQL 迁移、Docker 编排和基础前端脚手架。
