# 项目流程

Status: active
Last updated: 2026-11-09
适用范围：本仓库的手动开发与 subagent 编排。验收标准见 [acceptance.md](acceptance.md)，路线图见 [development-plan.md](development-plan.md)。

## 1. 原则

1. **单一事实源**：session、branch、timeline、approval、model、permission 的真相只在 App Server；Go 侧只保留可丢弃的视图状态。
2. **不读私有存储**：不读 App Server 的 SQLite/JSONL/session 文件/`daemon.log`，不调私有 IPC 或插件实例。
3. **不建第二份真相**：不写本地 session/run/approval 存储；崩溃或重连后从 App Server 重建。
4. **缺能力就 WIP**：入口保留、禁用、写清等待的上游方法；不伪造成功、不做本地兜底。
5. **证据分级**：测试通过 ≠ 产品验收；写结论时必须注明证据等级。

## 2. 角色与文件所有权

| 角色 | 职责 | 权限 |
|---|---|---|
| Leader（主 agent） | 拆任务、定边界、合并与提交、文档收口 | 可改任意文件，但优先让 owner 改 |
| Owner（subagent） | 在指定包内实现 + 测试 + 自验 | 只改被分配的路径 |
| Auditor | 只读审计，输出文档或报告 | 不改源码 |

规则：

- **一个包同时只有一个 owner**。跨包改动拆给多个 agent 或由 leader 合并。
- 每个任务必须写明：可改路径、禁改路径、必须运行的命令、完成判定。
- subagent 默认**不 commit**；由 leader 验证后按 scope 提交。授权提交时必须逐文件 `git add`。
- agent 遇工具/编译错误时，leader 用 `mode=steer` 让同一 agent 继续原任务，不缩小范围。
- 不允许用 `t.Skip` 掩盖失败；真实环境跑不了就如实报告 blocker。

## 3. 阶段与门禁

```text
S0 合同/需求 → S1 实现 → S2 集成 → S3 测试级验证 → S4 产品验收 → S5 提交与文档
```

| 阶段 | 进入条件 | 退出条件（门禁） |
|---|---|---|
| S0 合同 | 需求或缺陷有明确行为 | 有公开合同依据（wire DTO / 设计合同）；没有就停在 WIP |
| S1 实现 | S0 通过；文件所有权已分配 | 代码 + 单元测试；`gofmt` 干净 |
| S2 集成 | 依赖包已合并或并存无冲突 | 跨包调用点全部接线；fake 端到端通过 |
| S3 测试级验证 | S2 通过 | `go build ./...`、`go vet ./...`、`go test -count=1 ./...`；关键路径加 `-race` |
| S4 产品验收 | S3 通过 | 功能项 L4（真实窗口）；视觉项 L4 + L5（设计对照） |
| S5 提交与文档 | S4 或明确标注“仅测试级” | commit 按 scope 拆分；相关文档同步 |

门禁不通过时：保留工作树、写清 blocker、继续下一步可以并行的任务；不得用假证据跨过。

## 4. 测试与验证

按改动范围选择命令，全量验证在冻结工作树后执行：

```bash
gofmt -l .                 # 必须无输出
git diff --check           # 必须无输出
go build ./...
go vet ./...
go test -count=1 ./...
go test -race -count=1 ./...        # 涉及并发/生命周期时必须
go test -race -count=5 ./internal/session ./internal/appserver ./internal/integration
```

真实 daemon（opt-in，需先构建 `dim-app-server`）：

```bash
DIMAGENT_REAL_APP_SERVER_BIN=/tmp/dim-2-target/debug/dim-app-server \
  DIMAGENT_REAL_SESSION_FLOW=1 \
  go test -count=1 -run '^TestRealDaemon(Smoke|SessionFlow)$' ./internal/integration -v
```

- fake/headless 结果不能替代 real-daemon 证据；real-daemon 也不能替代真实窗口验收。
- 清理自己启动的 daemon 进程；不留临时 probe 文件。

## 5. 提交规范

- 一个 commit 只做一件事，按 scope 拆分：`internal/ui`、`cmd/dimagent`、`internal/integration`、docs 各成一条。
- 格式：`<type>(<scope>): <summary>`，如 `fix(integration): default input lookup to the session branch`。
- type 用 `feat`/`fix`/`test`/`docs`/`refactor`/`chore`；不用 `update`/`wip`。
- 提交前必须：目标包测试通过、`gofmt` 干净、`git diff --check` 干净；全量测试在合并波次时跑。
- 不提交生成物、临时 probe、未使用代码；不 `git add -A`。

## 6. 文档同步

| 改动 | 必须同步 |
|---|---|
| wire/方法/错误语义 | [protocol-contract.md](protocol-contract.md) |
| projection/transient 语义 | [projection-contract.md](projection-contract.md) |
| UI 组件/WIP 入口 | [native-plan.md](native-plan.md) |
| 验收状态变化 | [acceptance.md](acceptance.md) |
| token/组件对齐 | [ui-parity.md](ui-parity.md) |
| 里程碑/波次 | [development-plan.md](development-plan.md) |
| 对外状态与入口 | `README.md` |

规则：当前事实写进当前文档；逐日/逐波记录写进 `docs/archive/`，并保持链接可跳转。

## 7. Scheduler 与持续运行

- 使用 `mode: reuse`，不设置 `providerId`/`modelId`（复用当前执行上下文，避免每轮新建 session）。
- 默认 10 分钟触发、单轮上限 30 分钟；overlap 跳过而不是排队。
- 每轮先读：`git status`、最近 commit、agent 状态、上一个 run 结果，再决定下一步。
- 调度器只做“检查 + 推进”，不改变本流程的门禁。

## 8. 异常处理

| 情况 | 处理 |
|---|---|
| 上游方法未确认 | 停在 WIP，入口 visible-disabled，文档写明缺口 |
| mutation 结果未知（超时/断线） | 用 `session.input_lookup` 读回；**绝不自动重发** |
| `subscription_changed` | 只读路径最多重试一次；写操作 fail-closed，不重放 |
| transport 丢失 | 报错态；由 `Reattach` 在 transport 终止时重建，不用旧投影当真相 |
| daemon 不可用 | 显示连接错误；不回退 fixture，不用本地存储兜底 |
| 测试 flaky | 改成事件驱动等待，禁止用加长 sleep 掩盖 |

## 9. 当前波次

Wave 3（UI 1:1）进行中，顺序与验收见 [development-plan.md](development-plan.md) 和 [ui-parity.md](ui-parity.md)。
