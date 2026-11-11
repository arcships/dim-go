# D7 工作区与文件

Status: active（第一版实现 + 变更读取完成，待组织评审）
Owner: D7 领域 agent
范围: 工作区内本机文件与仓库的只读能力：目录读取、文件预览、关键词搜索、git 只读状态、本次运行变更读取、按平台外部打开、路径规范化与越界拒绝。
依据: `docs/domains/R.md` 块 11、`docs/domains/A.md` 工作区模块与四条总规矩、`AGENTS.md`。
Last verified: 2026-11-11

## 1. 边界

**做**

| 能力 | 说明 |
|---|---|
| 路径规范化 | 相对/绝对路径统一成工作区内绝对路径；符号链接解析后按新路径重新校验 |
| 目录读取 | 名称、类型、大小、修改时间；一次调用内附带 git 跟踪状态；含隐藏条目 |
| 文件预览 | 编码探测、二进制识别、按上限读取并显式标注截断 |
| 关键词搜索 | 一次性递归扫描，返回路径、行号、列号与片段；条数与字节双上限；不建索引 |
| git 只读 | 当前分支、是否分离头指针、是否有未提交改动、改动条目数 |
| 变更读取 | 从工具结果声明收集本次 run 的路径，按工作区内/外分类并补绝对路径与是否存在 |
| 外部打开 | 按平台调用系统默认处理程序，返回子进程退出码 |

**不做**

| 不做 | 归属/原因 |
|---|---|
| 界面、视图模型 | U1 |
| 连接、服务端索引搜索 | D1 |
| 投影适配、run 归属判定、工具声明提取 | D2/D12；本包只定义最小接口，见 §4.7 |
| 附件暂存与提交 | D8 |
| 任何写路径、缓存、索引、长期状态 | A 四条总规矩；本包只有一次性只读结果 |
| commit、checkout、stage、pull 等 git 写操作 | 本领域明确不提供 |

本包只使用 Go 标准库，不引第三方依赖，不改 `go.mod`。

## 2. 包结构

```text
internal/workspace/
├── workspace.go      路径规范化、工作区根、越界判定、错误值
├── dir.go            目录读取、条目类型、git 状态字段
├── preview.go        预览、编码探测、二进制识别、截断说明
├── search.go         一次性关键词扫描、上限与停止原因
├── changes.go        工具结果声明 → 路径清单（工作区内/外、去重、排序、存在性）
├── git.go            git 只读状态、porcelain 解析、目录状态聚合
├── open.go           平台命令选择、子进程退出码
└── *_test.go         临时目录与临时 git 仓库测试
```

包内不建共享 utils；重复的小工具（`joinNotes`、错误包装）留在各自文件。

## 3. 对外接口

```go
func NewWorkspace(root string) (*Workspace, error)
func (w *Workspace) Root() string
func (w *Workspace) Resolve(rel string) (string, error)
func (w *Workspace) Relative(p string) (string, error)

func (w *Workspace) ListDir(rel string, opts ListOptions) (DirListing, error)
func (w *Workspace) Preview(rel string, opts PreviewOptions) (Preview, error)
func (w *Workspace) Search(rel string, opts SearchOptions) (SearchResult, error)
func (w *Workspace) GitStatus(ctx context.Context) (GitStatus, error)
func (w *Workspace) ReadRunChanges(ctx context.Context, source ChangeSource, sessionID, runID string) (RunChanges, error)
func (w *Workspace) Open(rel string) (int, error)
```

变更读取的注入接口只有一条方法：

```go
type ChangeSource interface {
    ToolResultDeclarations(ctx context.Context, sessionID, runID string) ([]ToolResultDeclaration, error)
}
```

`ToolResultDeclaration` 带 `ToolCallID`、`ToolName` 与 `Paths`；适配器由集成层实现，本包不导入投影内部。

错误值：`ErrOutsideWorkspace`、`ErrPathNotExist`、`ErrNotDirectory`、`ErrNotRegularFile`、`ErrIsDirectory`、`ErrGitUnavailable`、`ErrUnsupportedPlatform`、`ErrOpenFailed`、`ErrEmptyQuery`、`ErrPreviewLimitTooLarge`。调用方按 `errors.Is` 分类，不用字符串匹配。

## 4. 关键判定

### 4.1 路径规范化与越界

1. 相对路径先拼到工作区根，绝对路径直接 `Clean`。
2. 解析已存在前缀的符号链接，不存在的尾段保留；解析结果可能是新的绝对路径。
3. 在**解析后的路径**上判断是否等于根或位于根之下；越界返回 `ErrOutsideWorkspace`，不启动任何读取或外部程序。
4. 中间路径是普通文件时返回 `ErrNotDirectory`，不把 `file/child` 当成可用路径。
5. 工作区根在 `NewWorkspace` 时解析符号链接；系统目录自身的符号链接不会被误判为越界。

测试：`TestResolveInside`、`TestResolveRejectsEscape`、`TestResolveRejectsPrefixSibling`、`TestResolveSymlink`、`TestResolveIntermediateFile`。覆盖 `..`、绝对越界、同前缀兄弟目录、目录/文件符号链接越界。

### 4.2 目录读取

- 一次 `os.ReadDir`，按名称字节序返回，包含隐藏条目与 `.git` 条目，不返回 `.` 与 `..`；不跟随符号链接。
- 条目类型：`file` / `dir` / `symlink` / `other`；大小与修改时间取 `Lstat` 语义。
- git 状态在一次调用内读取一次并附在条目上；目录状态由子条目状态按优先级聚合（冲突 > 修改 > 新增 > 删除 > 重命名 > 未跟踪 > 忽略）。
- git 缺失或不在仓库内不让列表失败，只在 `Note` 说明原因。
- 条目数超过上限时显式返回 `Truncated=true` 与说明，不静默截断。

测试：`TestListDirEntries`、`TestListDirLimitsAndErrors`、`TestListDirSymlinkKind`、`TestListDirGitStatus`、`TestListDirOutsideGitRepo`。

### 4.3 预览与编码

- 判定顺序：空文件 → UTF-32 BOM（明确不支持）→ UTF-8 BOM → UTF-16LE/BE BOM → NUL 字节（二进制）→ UTF-8 合法（文本）→ 其他（未识别编码，按二进制）。
- 文本返回 `Content`；二进制只返回元信息与原因，不返回内容，不猜测解码。
- 截断按字节执行；UTF-8 末尾被切断的半个字符会回退，最多回退 3 字节；UTF-16 末尾半个码元丢弃。
- 发生截断时 `Truncated=true`，`Note` 写明文件总字节数与已读字节数。

测试：`TestPreviewUTF8`、`TestPreviewTruncated`、`TestPreviewTruncatedMultibyte`、`TestPreviewExactlyAtLimit`、`TestPreviewEncodings`、`TestPreviewBinary`、`TestPreviewEmptyAndErrors`。

### 4.4 搜索

- 每次调用重新 `WalkDir`，不读缓存、不写索引；测试 `TestSearchCreatesNoIndex` 在搜索前后对比目录树。
- 跳过 `.git` 目录、符号链接、非普通文件、二进制文件、超过单文件上限的文件。
- 大小写不敏感、按行匹配；同一行只返回第一条命中；`Column` 按原行字符序号计算。
- 匹配按字符做大小写折叠，不先整体小写再取位置；个别字符（如开尔文符号、长 s）折叠后字节数会变化，整体小写会让列号错位。
- 三类上限：匹配条数、检查文件数、累计读取字节数。任一触发即停止，返回 `StopReason`（`max-matches` / `max-files` / `max-scan-bytes`）与 `Truncated=true`。
- 返回路径统一相对工作区根、以 `/` 分隔；即使搜索起点是子目录，路径仍从工作区根算起。

测试：`TestSearchMatches`、`TestSearchCaseInsensitive`、`TestSearchLimits`、`TestSearchSnippetAndScope`、`TestSearchSkipsSymlinks`、`TestSearchCreatesNoIndex`。

### 4.5 git 只读

- 只执行 `git rev-parse --show-toplevel` 与 `git status --porcelain=v1 --branch`；命令带 `GIT_OPTIONAL_LOCKS=0`，避免刷新索引。
- 非仓库返回 `IsRepo=false` 且不报错；本机没有 git 返回 `ErrGitUnavailable`；上下文取消返回 `context.Canceled`。
- 分支头兼容 `main...origin/main [ahead 1]`、`No commits yet on main`、`HEAD (no branch)`。
- 目录列表内部的快照使用 `-z --untracked-files=all`，路径不受 git 引号转义影响。

测试：`TestGitStatusCleanAndDirty`、`TestGitStatusDetached`、`TestGitStatusEmptyRepo`、`TestGitStatusNotRepo`、`TestGitStatusCanceled`、`TestGitStatusUnavailable`、`TestParseBranchHeader`、`TestParsePorcelainZ`。

### 4.6 外部打开

- 平台命令：darwin `open <path>`；linux/freebsd/openbsd/netbsd `xdg-open <path>`；windows `cmd /c start "" <path>`；其他平台 `ErrUnsupportedPlatform`，不做终端兜底。
- 路径先规范化并确认存在；不存在返回 `-1` 与 `ErrPathNotExist`，越界返回 `-1` 与 `ErrOutsideWorkspace`，两种拒绝都不启动子进程。
- 返回打开器进程的退出码；非 0 时同时返回包装 `ErrOpenFailed` 的错误；启动失败没有退出码，返回 `-1`。
- 测试通过注入执行器验证参数与退出码传递，**不真的启动桌面程序**；真实命令分支只测 `sh -c exit N`。

测试：`TestOpenSuccess`、`TestOpenExitCode`、`TestOpenRejectsMissingAndOutside`、`TestOpenDirectory`、`TestOpenCommandFor`、`TestRunExternalCommand`。

### 4.7 变更读取（本次运行）

- 数据源只有注入的 `ChangeSource`：按会话与 run 返回每个工具结果的路径声明。适配器从投影的工具结果里取 `structuredContent.affectedPaths`、`write`/`edit` 声明的 `structuredContent.path` 与 `fileWritten` effect；本包不扫描工作区、不推断 diff、不读 git 历史、不执行工具。
- source 为 nil 或声明为空时返回空清单，不把“没有声明”当成“没有改动”，也不据此扫描任何目录。
- 每条声明：去首尾空白、展开 `~/` 与 `~\`、相对路径拼到工作区根、`Clean`，再解析已存在前缀的符号链接；空路径与解析后等于工作区根的路径丢弃。
- 解析后在工作区内：`scope=workspace`，`path` 为 `/` 分隔的相对路径；在工作区外：`scope=external`，`path` 为绝对路径。两者都带解析后的 `absPath` 与 `exists`（`Stat` 成功即存在，声明删除的文件为 false）。
- 按 `scope + path` 去重（同一路径的相对/绝对写法合并），工作区内的条目排在前，同类按路径字节序排序；超过 `MaxChangePaths=256` 时截断并显式返回 `Truncated=true` 与说明。
- `coverage` 恒为 `tool_reports`：这是工具声明级的部分覆盖，不声称覆盖 exec、外部修改、未声明的写入或未产生声明的改动。
- 不落盘、不缓存；一次调用一次结果。上下文取消在调用来源与遍历声明之间都会检查。

接口形状（集成层适配，不导入投影）：

```go
type ToolResultDeclaration struct {
    ToolCallID string
    ToolName   string
    Paths      []string
}

type ChangeSource interface {
    ToolResultDeclarations(ctx context.Context, sessionID, runID string) ([]ToolResultDeclaration, error)
}
```

测试：`TestReadRunChangesMergesAndClassifies`、`TestReadRunChangesEmptyAndNilSource`、`TestReadRunChangesSourceErrorAndCancel`、`TestReadRunChangesTruncates`、`TestReadRunChangesSkipsRootAndBlank`、`TestReadRunChangesHomeExpansion`、`TestReadRunChangesSymlinkEscape`。覆盖多工具结果合并、重复路径、工作区外路径、相对路径规范化、home 展开、符号链接越界、空声明不扫描与来源错误。

## 5. 默认上限

| 项 | 默认 | 可调 | 超限表现 |
|---|---|---|---|
| 目录条目 | 5000（`DefaultMaxDirEntries`） | `MaxEntries`，0 用默认，负值不限制 | `Truncated=true` + 说明 |
| 预览读取 | 256 KiB（`DefaultPreviewBytes`） | `MaxBytes`；硬上限 16 MiB（`MaxPreviewBytes`） | 截断说明；超硬上限报错 |
| 搜索命中 | 100（`DefaultMaxMatches`） | 同上 | `StopReason=max-matches` |
| 搜索文件数 | 20000（`DefaultMaxFiles`） | 同上 | `StopReason=max-files` |
| 单文件参与搜索 | 2 MiB（`DefaultMaxFileBytes`） | 同上 | 跳过并计入 `Skipped` |
| 累计读取 | 32 MiB（`DefaultMaxScanBytes`） | 同上 | `StopReason=max-scan-bytes` |
| 片段长度 | 200 字符（`DefaultSnippetRunes`） | 同上 | 以 `…` 结尾 |
| 变更路径 | 256（`MaxChangePaths`） | 固定 | `Truncated=true` + 说明 |

所有上限触发的“不完整”都在结果结构里显式标记，不静默截断。搜索的累计读取上限在达到后停止继续扫描，至少完成当前一份文件，因此累计值最多比上限多一份文件（且不超过单文件上限）。

## 6. 验收证据

执行命令与本机结果（2026-11-11）：

```text
$ go test ./internal/workspace -count=1
ok   github.com/arcships/dimagent-mygo/internal/workspace   coverage: 85.3% of statements

$ go test -race ./internal/workspace -count=1
ok   github.com/arcships/dimagent-mygo/internal/workspace

$ go test ./...
ok   github.com/arcships/dimagent-mygo/internal/workspace
（其余包全部 ok）

$ go vet ./...
（无输出）

$ GOOS=windows go build ./internal/workspace
$ GOOS=linux go build ./internal/workspace
（均无输出，跨平台编译通过）
```

仓库级 `go test ./...` 与 `go vet ./...` 在本领域完成后均为通过状态。

### 6.1 真实模型证据（写入声明与 exec 边界）

日期 2026-11-11。夹具只存在 `<fixture-dir>`（本机临时目录，不入库；真实 daemon `0.1.0-alpha.21`、真实 provider、已登录账号）；测试只把夹具的账号与配置目录复制到新的临时目录后启动，来源目录只读，不改夹具。权限用 `full-access`，避免审批打断工具调用。

复现命令：

```bash
DIM_REAL_MODEL_FIXTURE=<fixture-dir> \
  go test -run TestRealDaemonRunChangesEvidence -v -count=1 ./internal/workspace/
```

观察输出（只列路径数量、分类、coverage 与布尔值；不含文件内容、命令正文、提示词、模型输出与凭据）：

```text
写入轮: 声明=1 路径=1 workspace=1 external=0 目标存在=true coverage=tool_reports
exec 轮: 声明=0 路径=0 workspace=0 external=0 目标在清单=false 目标存在=true coverage=tool_reports
```

结论：

- 写入轮：模型按要求调用 `write`，工具结果带写入路径声明（`fileWritten`/`affectedPaths`）；声明路径进入变更清单，分类为工作区内、存在性为真、`coverage=tool_reports`。
- exec 轮：模型按要求调用 `exec`，命令确实在磁盘上产生文件（目标存在=true）；但 exec 结果没有写入声明，变更清单为空，目标不在清单（目标在清单=false），边界成立。
- 模型不配合时测试用 `t.Skip` 标 blocked，并记录观察到的工具调用与事件类型，不伪造通过。
- 该测试在同一天连续运行三次，输出一致。

需求到测试的对应：

| R 块 11 完成标志 | 覆盖测试 | 本版结论 |
|---|---|---|
| ① 目录条目与 `ls -a` 一致 | `TestListDirEntries`（含隐藏文件、名称序、大小、类型） | 通过（集合一致；排序按字节序，见 §8） |
| ② 内容读取字节与磁盘文件一致 | `TestPreviewUTF8`、`TestPreviewExactlyAtLimit`、`TestPreviewEncodings` | 通过 |
| ③ 搜索不超上限、单次扫描、不生成索引 | `TestSearchLimits`、`TestSearchCreatesNoIndex` | 通过（服务端索引分支未接，见 §8） |
| ④ 变更只含本次运行工具声明路径 | `TestReadRunChangesMergesAndClassifies`、`TestReadRunChangesEmptyAndNilSource`、`TestReadRunChangesSkipsRootAndBlank`、`TestRealDaemonRunChangesEvidence` | 包级与真实模型证据均通过（写入声明进清单、exec 改动不进）；产品路径的投影适配与 run 归属仍由 D12 接线 |
| ⑤ 外部打开退出码 0、路径不存在非 0 | `TestOpenSuccess`、`TestOpenExitCode`、`TestOpenRejectsMissingAndOutside` | 通过；不存在时返回 `-1` + 明确错误，不启动子进程 |

测试全部使用 `t.TempDir()` 与临时 git 仓库；本机有 git 时执行 git 相关用例，无 git 时相关用例 `t.Skip`，不把跳过当通过。

## 7. git 依赖的处理

- 调用本机 `git` 子进程，不解析 `.git` 内部文件，不读 object 数据库。
- 仓库根由 `rev-parse --show-toplevel` 给出；工作区可以是仓库根或仓库内子目录。
- git 不可用：列表仍返回，`GitAvailable=false` 且 `Note` 说明；`GitStatus` 返回 `ErrGitUnavailable`。
- 工作区不在仓库内：`GitStatus.IsRepo=false`，不报错。
- 所有命令只读，且带 `GIT_OPTIONAL_LOCKS=0`；没有任何写 git 的代码路径。
- 测试在临时目录 `git init -b main` 并本地配置身份，不依赖全局 git 配置。

## 8. 已知限制与未决

| 项 | 现状 | 处理 |
|---|---|---|
| 服务端索引搜索 | 未接；本版只做本机一次性扫描 | 服务端提供索引能力后由调用方优先走服务端；本包保持“用完即弃” |
| 变更读取覆盖 | 只覆盖工具结果声明的路径，不覆盖 exec、外部修改与未声明写入；`coverage=tool_reports` | 产品基线即如此；不自行 diff、扫描或读 git 历史 |
| 变更读取接线 | 真实模型证据直接从事件流构造声明；投影适配器与 run 归属仍由集成层（D12）实现 | 本包只定义最小接口；产品路径的端到端证据待 D12 适配后补 |

## 9. 与架构的对应

- A「工作区」一行：拥有“本机文件与仓库只读结果（用完即弃）”，无长期状态；本包只返回一次性结构，不持有跨调用状态；不调用其他领域。
- A 四条总规矩：本领域不写事实、不提交意图、不显示未经确认的内容，也不参与界面临时状态。
- 依赖方向：调用方（界面、附件暂存、集成层）调用本包；本包不反向调用界面、连接、会话或附件包，也不导入投影类型。
- 变更读取只用注入的 `ChangeSource` 拿工具结果声明，不把“声明的路径”提升为服务端事实，也不在本地推导改动类型。

## 10. 变更记录

| 日期 | 变更 | 原因 |
|---|---|---|
| 2026-11-10 | 第一版：路径规范化、目录读取 + git 状态、预览、搜索、git 只读、外部打开与测试 | 执行组织 agent 的 D7 实现任务 |
| 2026-11-11 | 新增变更读取：工具结果声明 → 路径清单；补 `ChangeSource` 接口、上限与测试 | 补齐 R 块 11 的本次运行变更读取 |
| 2026-11-11 | 补真实模型证据：写入声明进入清单、exec 改动不计入；记录复现命令与观察输出 | 原变更读取只有 fixture 证据 |
