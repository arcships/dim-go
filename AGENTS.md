# AGENTS.md

本文件是本仓库的 agent 入口，只写仓库特有规则。

## 1. 项目

DimAgent 的 Go 桌面客户端：Go + mygo native UI 承载宿主与界面，Rust `dim-app-server` 通过 protocol v3 提供 agent 能力与全部事实。组织方案见 [docs/project-organization.md](docs/project-organization.md)。

## 2. 角色约定（强制）

- 与用户对话的 agent 是**组织 agent**。
- 组织 agent 的产出只有这些：领域清单、结果包要求、结果包评审意见、依赖与接口、顺序与并行、验收判定、组织文档。
- 组织 agent **不进行任何具体的领域工作**。需求分析、架构设计、Go 实现、领域文档、领域测试与验收证据，全部由对应的领域 agent 完成并提交。
- 组织 agent 需要某个领域的产物时，向该领域提出结果包要求（`docs/project-organization.md` §5），由领域提交；不得代写、代改、代交。
- 领域文档所有权归领域；组织 agent 只提出问题与更新要求，不直接修改领域文档。
- 组织 agent 按三条标准评审结果包：能否得到预期结果、是否有缺漏、是否可验收。
- 组织方案未获用户批准前，不提交结果包、不开始实现；各领域只做上下文准备。
- 需求侧验收由用户判定；组织 agent 只保证需求条目可追溯、可验收。

## 3. 技术约束

- App Server 拥有 session、branch、timeline、approval、model、permission；Go 侧只保留可丢的界面状态与登记的本地状态。
- 不读 App Server 的 SQLite/JSONL/session 文件/daemon.log，不调用私有 IPC 或内部 service。
- protocol v3 的 wire 定义只在连接领域出现一次。
- 主界面由 native view 组成；外部内容进独立隔离容器，不接入主界面，也拿不到 socket 与凭据。
- mygo 按上游 API 使用，不在本地改语义。
- 文档只写方法，不写本机信息：仓库内引用用相对路径（如 `internal/app`、`docs/domains/C-cost.md`）；不出现绝对路径与产品 home 下的具体目录名、不出现会话 uuid、不出现临时夹具目录名；外部路径用 `<placeholder>` 占位。
- `github.com/egoist/mygo` 是本项目的宿主框架（界面与窗口由它承载），属平台依赖，不是例外；其 API 按上游使用，不在本地改语义。
- 除宿主框架外，第三方依赖默认禁止。既有例外：`internal/terminal` 可用一个伪终端库（仅伪终端创建与读写）；`internal/ui/render` 可用 `markstream-go`（仅 Markdown 流式渲染）。其他模块保持零第三方依赖。新增例外须先在 `docs/domains/A.md` 出设计选择。

## 4. 文档位置

| 路径 | 内容 | 所有者 |
|---|---|---|
| `docs/organization-model.md` | 组织基线：角色与职责 | 用户 |
| `docs/project-organization.md` | 组织方案：领域、结果包、依赖、顺序、验收 | 组织 agent |
| `docs/domains/<ID>.md` | 领域文档与结果包 | 对应领域 agent |

## 5. 命令

代码进入仓库后，按范围执行：

```bash
go test ./...
go vet ./...
```

缺真实 App Server、真实仓库、真实账号或目标平台时，验收标 blocked 并写明依赖，不用替身结果代替。
