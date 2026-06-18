# Organizational Harness System Design

> 基于 Harness Engineering × 组织管理架构自进化研究报告构建的混合劳动力组织运营系统

## 技术栈

- **Frontend**: Next.js (App Router, React, TypeScript)
- **Backend**: Go (模块化单体, Domain-Driven Design)
- **Database**: PostgreSQL (多 schema 按 Domain 隔离)
- **Communication**: REST + WebSocket + Domain Events

## 架构原则

- 模块化单体：内部按 Domain 分包，可平滑拆解为独立服务
- Domain Events 解耦：模块间通过事件异步通信
- API Gateway 统一入口：认证、鉴权、限流集中
- Next.js 作为 BFF：服务端渲染 + API 聚合

## 九大 Domain 总览

| Domain | 对标 ETCLOVG | 核心职责 |
|--------|-------------|---------|
| Identity | — (基础层) | 人类员工与 AI 员工的身份、角色、权限管理 |
| Organization | Execution | MVRU 沙箱化组织架构设计与配置 |
| Layer | — (新增) | 组织层级(战略/战术/执行)与业务运营匹配 |
| Capability | Tooling | 能力目录注册、发现与智能路由 |
| Workflow | Lifecycle + Context | P-E-R(Planner-Executor-Reviewer)流程编排与混合劳动力协作 |
| Observability | Observability | 全链路追踪与多维指标实时监控 |
| Verification | Verification | 三维验证(结果/路径/环境)与三级审核(L1机器/L2 AI/L3专家) |
| Governance | Governance | 原则导向治理与分级权限(L1~L4) |
| Evolution | Meta-Harness | 自进化引擎 + Decision Weight 核心算法 |

---

## Identity Domain

### 核心模型

```
User:       人类员工实体
AIAgent:    AI 员工实体（数字员工/智能员工）
Role:       Planner | Executor | Reviewer（可组合）
Permission: Level 1~4 + 资源级权限粒度
Session:    认证会话（人类 SSO / AI API Key）
```

### 关键接口

- `Authenticate(credentials) → Session`
- `Authorize(session, action, resource) → bool`
- `RegisterAgent(spec) → AIAgent`
  - AI Agent 注册时声明：模型类型、能力集合、权限范围、可用工具

### AI 员工一等公民设计

- 人类和 AI 共用同一套 Role 体系
- AI 的 Level 4 权限天然受限（红线不可触碰）
- AI 注册时能力声明进入 Capability Domain 的能力目录
- 所有操作记录统一进入 Observability Trace

---

## Organization Domain

ETCLOVG 映射：**Execution（执行环境）**

### 核心模型

```
Organization: 组织根节点
MVRU:         沙箱化执行单元（最小可行重构单元）
  - boundary: 沙箱边界（数据权限 / 资源配额 / 网络策略）
  - capabilities: 该 MVRU 具备的能力集合（从 Capability 目录挂载）
  - members: 成员列表（人类 + AI）
Team:         MVRU 下的子团队
RoleTemplate: 角色模板（Planner/Executor/Reviewer）
Relationship: 组织单元间关系（report / collaborate / depend）
```

### 核心操作

- `CreateMVRU(spec) → MVRU`
- `AssignCapability(mvru, cap)`
- `SetBoundary(mvru, rules)`
- `AddMember(mvru, userOrAgent, role)`
- `Visualize() → OrgChart`
- `HealthCheck(mvru) → Score`

### MVRU 生命周期

```
设计 → 启动（分配资源/人员到位）→ 运行（日常协作+Obs监控）
→ 评估（Verif层定期评估，权重系统更新）
→ 进化（重组/拆分/合并）→ (回到设计 或 解散)
```

### 前端设计器

- 画布组件（基于 React Flow），拖拽创建 MVRU/Team/Role
- 连线定义关系（report/collaborate/depend）
- 右侧配置面板：边界配置、能力挂载、成员管理、ETCLOVG 分层配置
- 底部健康度预览栏
- 版本管理：发布快照、对比差异、回滚

### ETCLOVG 映射到组织配置

每个 MVRU 在创建时对应七层配置：

| 层 | 配置内容 |
|----|---------|
| Execution | MVRU 定义 + 基础设施/资源配额 |
| Tooling | 能力目录挂载（指向 Capability Domain） |
| Context | 知识库绑定（指向 Workflow Domain 的上下文管理） |
| Lifecycle | 流程模板绑定（指向 Workflow Domain） |
| Observability | 监控配置（指标/告警规则） |
| Verification | 审核节点配置（审核链定义） |
| Governance | 权限规则 + 治理原则绑定 |

---

## Layer Domain ★

组织层级与业务运营的匹配模块，独立一等 Domain，与其他模块构成多维协同。

### 三层级运营模型

| 层级 | 角色 | 业务焦点 | 决策类型 | 时间视野 | AI 角色 |
|------|------|---------|---------|---------|--------|
| 战略层 | C-suite / 战略委员会 | 方向设定、资源配置、治理原则 | 高不确定性、原则导向 | 季度/年度 | 战略推演、情景分析、信号扫描 |
| 战术层 | MVRU Lead / 产品负责人 | 目标拆解、流程设计、能力建设 | 中等确定性、数据驱动 | 周/月 | 流程优化建议、资源调度、预测 |
| 执行层 | Executor(人/AI) / Reviewer | 任务执行、交付产出、质量保障 | 高确定性、规范内自主 | 天 | 主执行者、辅助审核、异常上报 |

### 层级匹配流程

每个业务运营活动到达时：

1. **Layer Classifier**：基于复杂度/风险/战略重要性自动分类
2. **层级匹配规则**：确定应由战略/战术/执行哪一层处理
3. **动态路由**：匹配的层级执行，同时信息向上下层同步

### 层级判定维度

```
LayerMatch(operation) → Layer
  计算三个维度：
  ├── ComplexityScore: 涉及 MVRU 数 / 依赖链长度 / 领域跨度
  ├── RiskScore:       财务影响 / 合规敏感度 / 声誉影响
  └── StrategicScore:  与战略目标对齐度 / 长期影响范围
  综合得分 → 映射到战略/战术/执行层
```

### 层级间信息流动协议

- 战略层 → 战术层：方向/原则/约束
- 战术层 → 执行层：目标/资源/流程
- 执行层 → 战术层：反馈/异常/数据
- 战术层 → 战略层：汇总/洞察/建议

关键规则：
- 信息不过度上浮：执行层能解决的不到战术层
- 信息不过度下沉：战略层方向讨论不干扰执行层
- 异常自动升级：遇到权限外问题自动升级
- 可越级但可追溯：跨级沟通需记录在 Trace

### 层级 × ETCLOVG 交叉矩阵

| ETCLOVG | 战略层 | 战术层 | 执行层 |
|---------|--------|--------|--------|
| Execution | 定义组织边界 | 配置 MVRU 沙箱 | 在沙箱内工作 |
| Tooling | 批准能力方向 | 挂载能力到团队 | 调用能力执行 |
| Context | 组织记忆(文化/战略) | 经验记忆(最佳实践) | 工作记忆(任务上下文) |
| Lifecycle | 高阶流程框架 | 编排具体工作流 | 执行流程步骤 |
| Observability | 全局健康度 | MVRU 级别指标 | 个人/AI 任务追踪 |
| Verification | 验证原则审定 | 验证标准设定 | 执行验证操作 |
| Governance | 核心原则设定 | 原则→规则解释 | 原则内自主决策 |

### 层级 × Decision Weight α 调节

在不同层级，决策权重的六维度 α 自动调整：

| α 维度 | 战略层 | 战术层 | 执行层 |
|--------|--------|--------|--------|
| Expertise | 0.15 | 0.25 | 0.35 |
| TrackRecord | 0.20 | 0.25 | 0.20 |
| Reliability | 0.15 | 0.15 | 0.20 |
| Recency | 0.10 | 0.15 | 0.10 |
| ContextFit | 0.10 | 0.10 | 0.10 |
| Principle | **0.30** | 0.10 | 0.05 |

### 层级 × Evolution 联动

| 层级 | 进化类型 | 触发条件 | 实验方式 |
|------|---------|---------|---------|
| 战略层 | 原则调整/架构重组/资源重配 | 环境剧变/长期指标逆转 | 一个 MVRU 先行试点 |
| 战术层 | 流程优化/能力调整/权限调整 | 指标异常/瓶颈识别 | A/B 测试新旧流程 |
| 执行层 | 工具推荐/上下文优化/自动化提升 | 效率下降/错误率上升 | 自动切换并验证 |

---

## Capability Domain

ETCLOVG 映射：**Tooling（工具接口）**

### 核心模型

```
Capability: 能力定义（原子能力单元）
  - id, name, version, description
  - inputSchema / outputSchema (JSON Schema)
  - preconditions, errorHandling
  - permissionLevel (L1~L4)
  - costEstimate (时间/算力)

CapabilityGroup: 能力分组（按领域/部门）

CapabilityBinding: 能力挂载（绑到 MVRU 或 Team）
  - mvruId / teamId, capabilityId, bindingConfig

CapabilityInvocation: 能力调用记录（全链路追踪用）
  - caller, input/output, duration/cost, outcome
```

### 智能路由引擎 (Router)

```
MatchTask(task) → []RankedCapability
  基于语义匹配(embedding) + 历史成功率 + 当前负载

Route(task) → ExecutionPlan
  确定由谁执行(人 or AI) + 使用哪些能力
  考虑 DecisionWeight 影响路由优先级

Fallback(task, reason) → AlternativePlan
  首选方案失败时的降级/替代方案
```

### 能力发现协议

- OpenAPI 风格的结构化描述
- 自然语言查询能力目录
- AI 辅助能力推荐（任务意图 → 能力匹配）

---

## Workflow Domain

ETCLOVG 映射：**Lifecycle（生命周期和编排）** + **Context（上下文和记忆）**

### 核心模型

```
WorkflowTemplate: 流程模板
  - stages: []Stage (type: plan|exec|review)
  - assigneeType: human|ai|either
  - requiredWeight (最低决策权重)
  - tools: []CapabilityId
  - routingRules (动态路由规则)

WorkflowInstance: 流程实例
  - status: active|paused|completed|failed
  - currentStage, traceId

Task: 任务单元
  - assignee, input/output, decisionWeight (快照)
  - reviewChain

Decision: 决策记录
  - decisionMaker, weight, inputs/output, reasoning
  - outcome (事后验证 → 权重系统)

WorkflowContext: 上下文管理
  - workingMemory (工作记忆)
  - injectedExperience (注入历史经验)
  - principleNotes (原则提醒)
```

### Planner-Executor-Reviewer 分工

- **Planner**: 分析需求 → 设计方案 → 拆解任务 → 输出 TaskList
- **Executor**: 按方案执行 → 输出交付物 + 执行日志
- **Reviewer**: 独立审核(必须与 Executor 不同主体) → 三维检查(结果/路径/环境)

### 动态流程编排

- 低风险低复杂度 → 简化流程(AI 直接执行，事后审计)
- 高风险高复杂度 → 完整 P-E-R + 多轮审核
- 决策权重上升 → 自动精简该主体的审批链

### 上下文注入机制

- 关键决策节点自动检索相关历史经验
- 新人(新 AI)加入时自动注入相关上下文
- 复盘 → 结构化记录 → 知识库更新 → 下次自动注入

---

## Observability Domain

ETCLOVG 映射：**Observability（可观测性）**

### 追踪系统

- Trace: 端到端业务流程追踪
- Span: 每个节点的输入/输出/决策者/耗时/修改记录
- 结构化日志: 所有操作统一格式

### 多维指标体系

| 指标类型 | 具体指标 |
|---------|---------|
| 效率指标 | 决策周期、执行周期、等待时间、返工率 |
| 质量指标 | 错误类型分布、返工原因、验收通过率 |
| 成本指标 | 人力投入、AI 调用成本、资源消耗 |
| 健康度指标 | 团队士气(NPS)、协作效率、知识流动 |

### 核心接口

- `RecordDecision(decision)`
- `RecordInvocation(capInvocation)`
- `RecordTraceStep(traceSpan)`
- `QueryMetrics(filter) → Metrics`

---

## Verification Domain

ETCLOVG 映射：**Verification（验证和评估）**

### 三维验证

```
ResultVerification:      交付物满足验收标准？
PathVerification:        执行过程符合规范/原则？
EnvironmentVerification: 环境因素导致的假失败？
```

### 三级审核链

| 级别 | 执行者 | 方法 |
|------|-------|------|
| L1 | 机器检查 | 数据校验、合规扫描、自动化测试 |
| L2 | AI 辅助 | 模式识别、异常检测、一致性检查 |
| L3 | 专家审查 | 战略方向、关键决策、高风险判断 |

### Meta-Verification

- 验证者与被验证者必须是不同系统/主体
- 定期校准验证结果
- 跨 MVRU 互审 / 外部审计

### 核心接口

```
Verify(spec) → VerificationReport
  - 三维独立评分 + 综合结论 + 改进建议
  - 验证结果 → Decision Weight Engine (反向传播)
```

---

## Governance Domain

ETCLOVG 映射：**Governance（治理和安全）**

### 分级权限模型

| 级别 | 行为 | 适用场景 |
|------|------|---------|
| L1 | 自动执行 | 低风险，AI/数字员工直接操作 |
| L2 | 通知后执行 | 中等风险，执行后通知相关人员 |
| L3 | 审批后执行 | 高风险，必须审批 |
| L4 | 禁止 | 红线，不可触碰 |

### 原则系统 (Principle-Based)

- 核心原则定义（少量、清晰、可测试）
  - 如："决策必须可追溯"、"AI 不参与最终人事决策"
- 原则冲突解决机制 + 优先级规则
- 原则遵循度评估（→ 输入到 Decision Weight 的 PrincipleScore）

### "删控制"机制

- 定期评估每个控制点的 ROI
- 当团队/个人权重达标 → 自动精简控制
- 控制精简记录本身可审计

---

## Evolution Domain

ETCLOVG 映射：**Meta-Harness（自进化）**

### 决策权重引擎 (Decision Weight) — 系统核心

#### 六维度算法

```
DecisionWeight(User/AIAgent, Context) =
    α₁ × ExpertiseScore      (能力匹配度)
  + α₂ × TrackRecordScore    (历史表现分)
  + α₃ × ReliabilityScore    (可靠性/一致性)
  + α₄ × RecencyScore        (时效性衰减)
  + α₅ × ContextFitScore     (上下文契合度)
  + α₆ × PrincipleScore      (原则遵循度)

α₁~α₆ 为动态参数，由 Meta-Learning 层调节
```

#### 各维度计算

- **ExpertiseScore**: 基于 Capability Registry 声明 + 历史成功率 + AI模型基准性能
- **TrackRecordScore**: 最近 N 次决策的验证结果（正确+Δ，错误-Δ，严重错误大幅衰减）
- **ReliabilityScore**: 决策一致性 + 可解释性 + 稳定性（时间方差）
- **RecencyScore**: 指数衰减 e^(-λt)，λ 根据领域变化速度动态调整
- **ContextFitScore**: 当前上下文与历史决策上下文的 embedding 相似度
- **PrincipleScore**: Governance 层合规模块产出的合规评分

#### 自进化机制 (Meta-Learning)

每次决策完成后：

1. Verification 层产出 OutcomeScore (0~1)
2. 反向传播调整：
   - 提升/降低该主体的各维度分数
   - 同时调整 α₁~α₆（某维度贡献大则下次它的 α 上调）
3. 类似 SkillOpt"编辑-验证"循环：权重更新 → 验证集整体改善 → 决定是否接受
4. 跨主体调节：Human A 权重下降时自动向 AI B 转移

#### 决策权重影响范围

| 场景 | 影响方式 |
|------|---------|
| 任务分配 | 权重高者优先匹配高复杂度/高风险任务 |
| 审批流程 | 权重高者可触发 L2(通知后执行)，低者需 L3(审批后执行) |
| 集体决策 | 按权重加权投票，非一人一票 |
| 资源分配 | 权重高者获更多算力/工具/数据访问 |
| 流程精简 | 权重累积到阈值可精简审批链 |

### 感知引擎 (Sensing)

- **内感知**: Observability 数据扫描（异常检测、流程瓶颈识别）
- **外感知**: 市场变化、竞品动态、政策法规（API/手动输入）
- **输出**: 带优先级的时间序列 Signal[]

### 学习引擎 (Learning)

- 假设生成: "如果调整 X，可能改善 Y"
- 实验设计: 参数范围、对照组、成功标准
- 实验执行: 单个 MVRU 范围，可随时回滚
- 结果验证: 实验组 vs 基线 → 决策是否全量推广

### 知识沉淀引擎 (Knowledge)

- 复盘 → 结构化记录 → 自动提取关键经验
- 更新到 Workflow Context 层
- 类比: Harness Coding 的"偏航记录 → 规则沉淀 → 下次注入"

---

## 全局数据流

```
Operation/需求
    │
    ▼
Layer Classifier ── 匹配层级(战略/战术/执行)
    │
    ▼
Capability Router ── 匹配能力 + 调用者(基于DecisionWeight)
    │
    ▼
Workflow Engine ── P-E-R 流程编排 + 上下文注入
    │
    ▼
Execution(人/AI) ── 任务执行 + 工具调用
    │
    ▼
Verification ── 三维验证 + 三级审核
    │
    ├──→ Decision Weight Engine (更新权重)
    ├──→ Evolution Engine (自进化感知输入)
    └──→ Observability (全链路记录)
```

---

## 项目结构

```
/harness-org/
├── frontend/                    # Next.js
│   ├── src/
│   │   ├── app/                 # App Router pages
│   │   ├── components/          # 共享 UI 组件
│   │   ├── lib/                 # API client, utilities
│   │   └── stores/              # 状态管理
│   ├── package.json
│   └── tsconfig.json
│
├── backend/                     # Go
│   ├── cmd/
│   │   └── server/              # 入口
│   ├── internal/
│   │   ├── domain/              # 九大 Domain
│   │   │   ├── identity/
│   │   │   ├── organization/
│   │   │   ├── layer/
│   │   │   ├── capability/
│   │   │   ├── workflow/
│   │   │   ├── observability/
│   │   │   ├── verification/
│   │   │   ├── governance/
│   │   │   └── evolution/
│   │   ├── pkg/                 # 公共库
│   │   └── gateway/             # API Gateway
│   ├── go.mod
│   └── go.sum
│
├── migrations/                  # PostgreSQL 迁移
│   └── *.sql
│
└── docker-compose.yml
```
