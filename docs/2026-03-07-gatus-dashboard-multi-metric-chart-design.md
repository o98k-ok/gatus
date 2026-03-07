# Gatus Dashboard 单图表多指标展示方案

日期：2026-03-07

## 背景
当前 Gatus 详情页默认展示的是 response time 历史趋势。现有系统已经做过定制，支持在单个图表中展示一个自定义指标。

本次目标是进一步支持：在同一个图表中展示多个自定义指标，用于更集中地观察某个 endpoint 的核心监控指标变化。

当前期望范围：
- 单图表展示多个自定义指标
- 暂不要求与 response time 混合展示
- 后续开发将基于本文档推进

## 当前实现现状
代码现状已经具备一部分基础能力：

1. 前端图表组件已经从默认 response time 做过定制，当前组件位于：
   - `web/app/src/components/ResponseTimeChart.vue`

2. 后端已新增指标历史接口：
   - `/api/v1/endpoints/:key/metrics/:duration`

3. 当前指标配置仍是单指标模型：
   - `endpoint.metric`

4. 存储层已经会把每次检查中各 condition 的左值落库到：
   - `endpoint_result_conditions.value`

这意味着：
- 不需要新增采集链路
- 不需要新增数据库表
- 主要改动集中在配置模型、历史查询接口、前端图表多序列展示

## 难度评估
整体评估为：中等难度。

原因如下：

### 低成本部分
- condition 的 resolved value 已经持久化，具备历史追溯基础
- 当前已有单指标历史查询接口，可以复用查询逻辑
- Chart.js 本身支持多 dataset 折线图

### 中成本部分
- 配置层现在只支持 `metric` 单对象
- API 当前只返回单条序列
- 前端图表当前是“response time / 单指标”二选一模式，不是多序列模型

### 风险点
- 多个指标的时间戳可能不完全对齐
- 多个指标的单位可能不同
- 如果后续要把 response time 和业务指标画在一张图里，会引入双轴或归一化复杂度

结论：
如果本期目标限定为“仅支持多个自定义指标共图，不与 response time 混画”，则方案风险较低，可控性较好。

## 推荐方案
推荐采用以下方案：

- 展示范围：仅支持多个自定义指标共用一张图
- 配置方式：新增 `metrics` 数组
- 兼容策略：保留现有 `metric` 单指标配置
- 接口策略：后端统一返回 `series[]`
- 前端策略：一个图表渲染多个 dataset
- 性能策略：v1 先按每个 metric 分别查询实现，不做复杂 SQL 合并

这是当前性价比最高、兼容性最好、实施风险最低的方案。

## 详细设计

### 1. 配置层设计
当前配置为：

```yaml
metric:
  name: CPU
  value: "[BODY].cpu"
  unit: "%"
```

推荐扩展为：

```yaml
metrics:
  - name: CPU
    value: "[BODY].cpu"
    unit: "%"
  - name: Memory
    value: "[BODY].memory"
    unit: "MB"
```

同时保留旧配置 `metric`，用于兼容已有场景。

#### 配置规则
- `metric` 和 `metrics` 不能同时配置
- `metrics` 至少包含 1 项
- 每个 metric 都必须满足现有校验规则：
  - `name` 非空
  - `value` 非空
  - `value` 必须是 placeholder pattern，例如 `[BODY].cpu`
- 同一 endpoint 下建议校验 `metrics[].name` 唯一，避免图例冲突

#### 运行时兼容策略
- 如果配置的是 `metric`，运行时转换为长度为 1 的指标列表
- 后续内部逻辑统一按“指标列表”处理，减少分支复杂度

### 2. 后端接口设计
当前接口：

- `/api/v1/endpoints/:key/metrics/:duration`

建议保留路径不变，但返回结构从单序列升级为多序列：

```json
{
  "series": [
    {
      "name": "CPU",
      "unit": "%",
      "timestamps": [1710000000000, 1710000300000],
      "values": ["12.30", "14.80"]
    },
    {
      "name": "Memory",
      "unit": "MB",
      "timestamps": [1710000000000, 1710000300000],
      "values": ["256.00", "260.50"]
    }
  ]
}
```

#### 返回结构说明
- `series`：多条指标序列
- 每条序列包含：
  - `name`
  - `unit`
  - `timestamps`
  - `values`

#### 空数据策略
如果 endpoint 未配置任何指标，返回：

```json
{
  "series": []
}
```

### 3. Store / 查询层设计
存储层当前能力已经足够支持 v1。

#### 现状
- `endpoint_result_conditions.value` 已经存储每次检查中 condition 左值
- `GetMetricHistory(key, pattern, from, to)` 已可按 pattern 查询历史数据
- 聚合策略已经存在：
  - `1h`：原始数据
  - `24h`：5 分钟聚合
  - `7d`：30 分钟聚合
  - `30d`：2 小时聚合

#### v1 实现建议
- 不修改数据库 schema
- 不强行引入批量查询接口
- 后端 API 层按 `metrics` 循环调用 `GetMetricHistory`
- 再组装为 `series[]` 返回

#### 性能判断
- 对 2 到 5 个指标的典型场景是可接受的
- 当前详情页在 metric 场景默认切到 `1h`，也能降低点位数量
- 如果未来单 endpoint 指标数继续增加，再考虑扩展批量 SQL 查询接口

### 4. 前端图表设计
当前图表组件是单序列思路，需改为多 dataset 模式。

#### 核心改造点
- `ResponseTimeChart.vue` 内部状态从：
  - 单个 `timestamps`
  - 单个 `values`
  - 单个 `metricName / metricUnit`
- 改为：
  - `series[]`
  - 每个 series 映射成一个 dataset

#### 展示策略
- 同一个折线图展示多个 dataset
- 开启 legend，显示各指标名称
- tooltip 展示：
  - 指标名称
  - 当前值
  - 单位
  - 时间点
- 每个指标使用不同颜色，采用固定颜色池，避免刷新后颜色漂移

#### 标题建议
图表标题保持通用，不再绑定某个单指标名称：
- `Core Monitoring Indicators`

#### 时间范围策略
保持现有 duration：
- `1h`
- `24h`
- `7d`
- `30d`

若 endpoint 配置了自定义指标，默认仍建议进入 `1h` 视图，避免高频数据点过多导致图表拥挤。

#### 事件标注
当前 UNHEALTHY 事件 annotation 可继续保留。

改造要求：
- 标注位置不要再只依赖单一序列
- 应改为基于所有 dataset 的全局最大值决定 annotation 的相对位置

### 5. 单位与展示约束
本期不建议在一个图中引入 response time 与自定义指标混画。

原因：
- response time 通常单位为 `ms`
- 业务指标可能是 `%`、`count`、`MB`、`qps`
- 混画后会出现量纲不同、尺度差异过大的问题
- 会引入双 Y 轴或归一化策略，复杂度明显上升

因此本期约束如下：
- 单图只展示多个自定义指标
- 不处理 response time 与指标叠加
- 不引入双 Y 轴
- 若多个指标单位不同，仍允许展示，但不做特殊缩放，仅在 tooltip / legend 中展示 unit

## 推荐实施步骤
建议按以下顺序开发：

1. 扩展配置模型
   - 新增 `metrics []*Metric`
   - 保留 `metric *Metric`
   - 完成校验与兼容逻辑

2. 扩展 endpoint status 返回结构
   - 返回 `metrics`
   - 保持旧 `metric` 兼容

3. 升级 metrics history API
   - 保留原路径
   - 返回结构改为 `series[]`

4. 改造前端图表
   - 支持多 dataset
   - 增加 legend / 多指标 tooltip / 颜色映射

5. 补齐测试
   - 配置校验
   - API 返回
   - 单指标兼容
   - 多指标渲染

## 测试建议

### 后端测试
- 仅 `metric` 配置时接口返回 1 条 series
- 仅 `metrics` 配置时接口返回多条 series
- `metric` 和 `metrics` 同时出现时报错
- 未配置指标时返回 `series: []`
- endpoint 不存在返回 404
- 非法 duration 返回 400

### 前端测试
- 单指标场景不回归
- 多指标图例正确显示
- tooltip 展示正确的 name / unit / value
- 切换 duration 时能重新加载多序列数据
- 某个指标无数据时不影响其他指标展示
- 事件 annotation 在多序列场景下仍正常显示

## 结论
本需求适合在当前定制基础上继续演进，推荐采用“新增 `metrics` 数组并兼容旧 `metric`”的方式实现。

该方案的优点是：
- 兼容现有配置与已有单指标能力
- 不需要改采集链路
- 不需要改数据库表结构
- 可以以较低风险完成“单图多指标”目标

后续若需要支持“指标 + response time 混合展示”，建议在本期完成后再单独设计双轴方案。
