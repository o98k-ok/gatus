# Gatus Agent Guide

## Development Principles

1. **勤劳务实**  
   不做“先改再说”的提交式开发。任何改动都要结合当前实现、调用链和验证结果判断是否成立。

2. **全面思考**  
   变更前先评估影响面，尤其是以下路径：
   - Go 后端 API 与存储接口
   - `config` 下的配置模型与兼容性
   - `web/app` 前端展示与交互
   - `web/static` 嵌入产物与最终二进制行为

3. **经验丰富**  
   以资深 Go / Vue 工程师的标准要求实现质量。优先做结构清晰、兼容性明确、可测试的修改，避免临时性补丁。

4. **追求卓越**  
   不满足于 “能跑”。需要同时关注：
   - 配置兼容
   - API 语义清晰
   - 代码可维护性
   - 测试覆盖
   - 构建可复现

5. **善于合作**  
   遇到不确定性时，优先通过代码、测试、文档和实际构建收敛事实；必要时再向用户同步权衡，不凭空拍板。

6. **主人翁意识**  
   这是公开项目。新增字段、接口、文档和前端交互都应可被长期维护，不引入含糊命名、隐式约定或脆弱实现。

## Project Stack

- 后端：Go 1.24，Fiber v2
- 配置：YAML，核心模型位于 `config/`
- 存储：memory / SQLite / Postgres，核心实现位于 `storage/store/`
- 前端：Vue 3 + Vue Router + Vue CLI
- UI：Tailwind CSS + `lucide-vue-next`
- 图表：Chart.js + `vue-chartjs` + annotation plugin
- 发布形态：前端构建到 `web/static/`，再由 Go 二进制 embed

## Working Rules

### 1. 先看现状，再动手

- 改配置能力时，先检查：
  - `config/endpoint/`
  - `api/`
  - `storage/store/`
- 改前端图表或详情页时，先检查：
  - `web/app/src/views/`
  - `web/app/src/components/`
  - 后端对应 API 返回结构
- 改存储时，必须同时检查 memory 和 SQL 两套实现是否需要同步。

### 2. 优先兼容现有行为

- 现有公开配置、JSON 字段、路由和 badge 行为，默认视为兼容面。
- 如果必须调整返回结构或配置格式：
  - 优先保留旧字段
  - 提供平滑迁移路径
  - 补测试覆盖兼容场景

### 3. 变更必须贴合技术栈

- Go 代码应延续现有包结构，不跨层堆逻辑。
- Fiber handler 保持现有错误处理风格和状态码约定。
- Vue 组件遵循当前 Composition API 写法和现有依赖栈，不引入不必要的新库。
- 涉及图表时，优先复用当前 Chart.js / `vue-chartjs` 方案，不重造渲染层。

### 4. 测试与验证不能省

- 后端逻辑改动后，至少执行：
  - `go test ./...`
- 前端展示或嵌入链路改动后，优先执行：
  - `./build.sh`
- 仅前端静态改动、且不需要完整二进制验证时，可先执行：
  - `npm --prefix web/app run build`
- 如果改动覆盖前后端联动、embed、配置解析或发布路径，最终验证以：
  - `./build.sh`
  为准。

## Validation Rule

### Default

- 每次完成代码改动后，必须根据影响范围做验证。
- 不允许跳过验证直接交付“理论正确”的修改。

### Preferred Commands

1. 后端改动优先：
   - `go test ./...`

2. 前端改动优先：
   - `npm --prefix web/app run build`

3. 涉及最终产物、前后端联动、静态资源 embed、配置加载、发布行为时：
   - `./build.sh`

### Notes

- `./build.sh` 会执行前端构建、`go mod tidy` 和后端构建，并生成最终 `./gatus`。
- 若验证失败，需要记录失败原因并基于失败结果继续修正，而不是绕过。
- 如果本次只修改文档或纯说明性文件，可不执行构建，但应明确说明未运行验证。

## Change Checklist

在提交任何实现前，至少确认以下事项：

- 是否影响已有 YAML 配置兼容性
- 是否影响已有 API 返回结构
- 是否影响 memory / SQLite / Postgres 之间的一致性
- 是否影响前端已有页面或图表行为
- 是否补充了必要测试
- 是否完成了对应构建或测试验证

## Build Log

| Date (YYYY-MM-DD) | Time | Commit/Ref | Command | Result | Notes |
|---|---|---|---|---|---|
| 2025-11-25 | 11:05 | `84f8528d` | `n/a (historical backfill)` | unknown | 单 metric 能力首版：新增 `endpoint.metric` 配置、`/api/v1/endpoints/:key/metrics/:duration` 历史接口，以及详情页单指标图表展示；原始构建命令与结果未在仓库中保留。 |
| 2026-03-07 | 20:48 | `b01e3354` | `go test ./config/endpoint/... ./api/... ./alerting/provider/custom/...` | ✅ PASS | 多指标图表支持：新增 `metrics` 数组配置、API 返回 `series[]`（单指标保留旧字段兼容）、前端多 dataset 图表展示；新增 6 个配置校验测试用例。 |
| 2026-03-07 | 20:48 | `b01e3354` | `npm --prefix web/app run build` | ✅ PASS | 前端构建验证通过，产物输出到 `web/static/`。 |
| 2026-03-08 | 09:43 | `f1fce5c0` | `n/a (historical backfill)` | unknown | 新增 `config-test-multi-metric.yaml` 本地测试配置：包含一个 `metrics:` 多指标 endpoint 和一个 `metric:` 单指标兼容 endpoint；提交前未保留构建/运行验证记录。 |
