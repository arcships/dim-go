# D8 附件暂存

Status: active（第一版实现完成：暂存、上限、去重、分类、失效识别与清理）
Owner: D8 附件暂存领域
依据: `docs/domains/R.md` 功能块 12 与 R7、R8；`docs/domains/A.md` 的「附件」对象与「附件暂存」状态机
范围: 只做附件选择之后的本地暂存。不做提交与上传（属写登记）、界面、会话、工作区读取与文件写入路径。

本领域只保存可由用户丢弃的本地状态：来源文件的一份副本与索引元数据。暂存区不落会话、轮次、草稿等业务事实，
不向服务端发任何请求。

## 1 结果包

**领域**：D8 附件暂存

**输入**

- `docs/domains/R.md` 功能块 12（选定后的暂存、文本落点、图片形态、上限与拒绝、重复挂载去重）与 R7（文本 10 MB、图片 8 MB、超限在提交前拒绝）、R8（附件暂存属于允许保留的本地状态）。
- `docs/domains/A.md` 的「附件」对象（本地暂存标识；路径、类别、字节数、去重键）与「附件暂存」状态机（空 → 暂存中 → 待提交 / 拒绝）。
- 本领域实现约束：只用标准库；暂存区为应用数据目录下的独立子目录；文件 0600、目录 0700；去重用内容摘要；上限在暂存时判定；不静默截断。

**产物**

| 路径 | 职责 |
|---|---|
| `internal/attachments/attachments.go` | 对外类型：类别、暂存项、上限、校验结果、清理报告、错误类型 |
| `internal/attachments/classify.go` | 按内容判定文本、图片与不受支持的二进制容器 |
| `internal/attachments/store.go` | 暂存区打开、暂存、读取、引用计数、校验、清理、索引与内容文件的原子写入 |
| `internal/attachments/store_test.go` | 暂存、超限、去重、失效识别与清理、权限、重启恢复等测试 |
| `internal/attachments/classify_test.go` | 类别判定的表驱动测试 |
| `docs/domains/D8-attachments.md` | 本文件 |

**验收**

- 命令：`go test ./internal/attachments -count=1`、`go vet ./...`，本领域全部通过。
- 本领域测试：`go test ./internal/attachments -v`，30 个顶层测试与 31 个子测试全部通过。
- 仓库整体 `go test ./...` 需等其他领域并行写入落地后复跑；本领域不修改他人包，也不为他域的失败背书。
- 证据档位：本机测试（固定样例）。不涉及服务端事实，因此不用于宣称附件提交链路可用。

**依赖**

- 上游：D3 会话目录与生命周期（草稿与待发附件的归属）、D7 工作区与文件（选择来源与路径规范化，本版只接收路径，不依赖其包）。
- 下游：写登记（提交与上传，尚未实现）。调用方必须按「每次成功 `Stage` 配对一次 `Release`」使用引用计数，并在启动与提交后调用 `Cleanup`。

**上下文需求**

- 需要 D3 明确：草稿删除、提交失败、提交成功三种情况下，待发附件何时释放引用。
- 需要用户确认暂存区总大小上限的具体数值（R 附 B 的 Q-3；本版默认 500 MB，见 §5）。
- 需要 A 确认：来源路径是否必须随附件保留。本版只保留文件名与摘要，不落来源路径（见 §11）。

## 2 与需求、架构的对应

| 需求 / 架构条目 | 本领域责任 | 不在本领域 |
|---|---|---|
| 选定后的暂存 | `Stage` 复制内容并返回标识、名称、类别、字节数、摘要 | 选择对话框、粘贴入口 |
| 文本附件落点 | 判为 `text`，供上层按服务端资源引用提交 | 资源写入与消息体拼装 |
| 图片附件形态 | 判为 `image`，供上层按图片内容提交 | 图片内容条目生成 |
| 上限与拒绝（R7） | 暂存时判定单文件与总量，超限拒绝并给出字节数与上限 | 展示文案与提交前拦截 |
| 重复挂载去重（R8） | 按内容摘要只存一份，重复挂载返回同一标识 | 资源写入去重（上层按标识去重） |
| A「附件暂存」状态机前三行 | 空 → 暂存中 → 待提交 / 拒绝 | 「提交被受理 → 已提交」属写登记 |
| A「附件暂存」失效标注 | `Validate` 逐项报告缺失与元数据不一致 | 界面上标注「不可用」 |
| R8 本地状态边界 | 暂存区只放内容副本与索引元数据，可整体丢弃 | 业务事实 |

## 3 对象与状态

### 3.1 暂存项

| 字段 | 业务含义 | 约束 |
|---|---|---|
| `ID` | 暂存标识，上层用它引用附件、去重上传 | 128 位随机十六进制，创建后不变 |
| `Digest` | 内容的 SHA-256 摘要，十六进制小写 | 去重键；内容文件的文件名 |
| `Name` | 首次暂存时的文件名（不含目录） | 只用于展示，不参与路径拼接 |
| `Category` | 按内容判定的类别 | `text` 或 `image` |
| `Size` | 内容字节数 | 与内容文件实际长度一致 |
| `Refs` | 挂载引用计数 | 每次成功 `Stage` 加一，`Release` 减一；不小于 0 |
| `StagedAt` | 该内容首次进入暂存区的时间 | UTC |

同一摘要只对应一个暂存项。重复挂载不新建项，只增加 `Refs` 并返回同一 `ID`。

### 3.2 状态与迁移

对应 A「附件暂存」状态机：

| 当前状态 | 事件 | 下一个状态 | 本领域的表现 |
|---|---|---|---|
| 空 | 选择文件 | 暂存中 | `Stage` 开始读取来源文件 |
| 暂存中 | 读取成功且在上限内 | 待提交 | `Stage` 返回 `Entry`，内容已落盘 |
| 暂存中 | 超限或不可读 | 拒绝（终态） | 返回 `*LimitError` / `*UnsupportedError` / 读取错误，不留下暂存项 |
| 待提交 | 提交被受理 | 已提交 | 由写登记实现；本领域只提供 `Release` 与 `Cleanup` |

补充约定：

- **引用归零不等于删除**：`Release` 把 `Refs` 减到 0 后保留记录与内容文件，由下一次 `Stage` 或 `Cleanup` 回收。这样上层可以在提交失败后重新挂载同一内容而不必重新选择文件。
- **失效项可以仍被引用**：`Cleanup` 对 `Refs > 0` 的失效项只报告、不删除，避免删掉草稿仍在使用的项。
- **重启**：`Open` 读取索引，`Validate` 逐个复算内容文件，能识别「内容文件缺失 / 不是普通文件 / 读不出来 / 字节数不一致 / 摘要不一致 / 类别不一致 / 记录不合法」七类失效。

## 4 暂存区格式与版本策略

### 4.1 目录与权限

```text
<应用数据目录>/attachment-staging/          0700
├── index.json                              0600
└── blobs/                                  0700
    ├── <sha256 十六进制>                    0600
    └── .tmp-*                              0600（写入中的临时文件）
```

- 内容文件按摘要命名，天然去重；同一摘要只有一个文件。
- 索引与内容文件都用「临时文件 + `fsync` + 改名」原子替换，目录项尽力同步。
- `Open` 会把暂存区目录、`blobs` 目录收紧到 0700，把索引收紧到 0600。
- 索引和内容文件都不写来源路径，只有文件名、类别、字节数、摘要、引用计数与暂存时间。

### 4.2 索引格式

```json
{
  "version": 1,
  "entries": [
    {
      "id": "0f3c9a5d2b7e4c18a6d1f8e3b0c7a925",
      "digest": "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
      "name": "notes.txt",
      "category": "text",
      "size": 4,
      "refs": 1,
      "staged_at": "2026-11-10T05:00:00Z"
    }
  ]
}
```

加载时校验：版本号、标识与摘要唯一、摘要为 64 位小写十六进制、类别在允许集合内、字节数与引用计数非负、
文件名与暂存时间非空。任一不成立即返回错误，不带病运行。

### 4.3 版本策略

- 当前版本 `1`。写入固定写当前版本。
- 兼容变化（新增可选字段、调整字段注释）不升版本：旧代码忽略未知字段，新代码对缺省字段取默认值。
- 不兼容变化（字段语义、结构、类别判定规则、摘要算法、目录布局）必须升版本。
- 打开时版本不等于当前支持版本即返回 `ErrIndexVersion`，由上层决定是否 `Reset`。暂存区不承载业务事实，重建的代价只是重新选择文件。
- 摘要算法只使用 SHA-256。更换算法属于不兼容变化，需升版本并重建。

### 4.4 崩溃与异常恢复

- 内容文件已写、索引未更新：留下无索引记录的文件，`Cleanup` 按孤儿文件回收。
- 索引已更新、内容文件未删：同理由 `Cleanup` 回收。
- 索引写坏：`Open` 返回 `ErrIndexCorrupt`；上层可调用 `Reset`（或删除暂存区目录）重建为空。
- 暂存区目录被外部改动：`Validate` 报告失效项，`Cleanup` 按规则处理。

## 5 上限

| 上限 | 默认值 | 来源 | 判定 |
|---|---|---|---|
| 单文件文本 | 10,000,000 字节 | R7：文本 10 MB | 按内容判为文本后判定 |
| 单文件图片 | 8,000,000 字节 | R7：图片 8 MB | 按内容判为图片后判定 |
| 暂存区总大小 | 500,000,000 字节 | 需求未规定，本领域默认值，可按配置覆盖 | 新增内容时按去重后总量判定 |

判定顺序：

1. 先读文件头（512 字节）判定类别，据此选单文件上限；来源文件已超该上限时直接拒绝，不读全量内容。
2. 读取内容（最多读上限加一字节，识别读取期间文件变大），超限即拒绝。
3. 按完整内容复核类别；类别若与文件头判定不同，按新类别再判一次上限。
4. 计算 SHA-256。摘要已存在时不增加占用，只增加引用计数，不做总量判定。
5. 摘要不存在时先回收无引用项，再用「已占用 + 本次字节数」判定总量。

拒绝行为（R7 选 A）：

- 超限直接拒绝，不截断、不写暂存区、不返回部分内容。
- `*LimitError` 给出 `Kind`（单文件 / 总大小）、`Category`、`Size`（本次字节数）、`Limit`（被触发的上限），总大小超限时另给 `Current`（已占用字节数）。
- 错误文案同时包含实际字节数与上限，例如：`附件超过单文件上限：文本附件 33 字节，上限 32 字节`、`附件超过暂存区总大小上限：已占用 100 字节，本次文本附件 100 字节，合计 200 字节，上限 150 字节`。
- 恰好等于上限时接受，测试覆盖。

## 6 去重

- 去重键是内容摘要（SHA-256），不是路径、文件名或时间。
- 同一摘要只存一份内容文件、一条索引记录，重复挂载返回同一 `ID` 并增加 `Refs`。
- 重复挂载时若发现内容文件缺失或与本次内容不一致，以本次读到的内容修复内容文件并校正记录的字节数与类别。
- 并发调用 `Stage` 由进程内互斥串行化；测试覆盖 8 个并发挂载同一内容得到同一标识、`Refs = 8`、内容文件数为 1。
- 已知取舍：同一内容以不同文件名挂载时，展示名称保留首次暂存的名字。去重语义优先于名称差异（见 §11）。

## 7 类别判定

只按内容判断，不看扩展名。

**图片**（按文件头）：PNG、JPEG、GIF87a/89a、TIFF（II/MM）、WebP（RIFF + WEBP）、HEIF/AVIF（ISO 基础媒体格式的 `ftyp` 主品牌）。
命中即 `image`。

**已知二进制容器**（按文件头）：PDF、ZIP 及其变体、GZIP、XZ、BZIP2、7Z、RAR、ELF、PE、Mach-O、Java 类文件、OLE、SQLite。
命中即拒绝，原因写明识别到的格式，避免把二进制当作文本上传。

**其余按文本处理**，判定规则：

- 出现 NUL 字节 → 二进制，拒绝（UTF-16 文本因此被拒绝，见 §11）。
- 除制表、换行、竖向制表、换页、回车与 ANSI 转义外，控制字节占比超过 1% → 二进制，拒绝。
- 其他内容判为 `text`：UTF-8 与常见单字节/多字节文本编码都能通过，不要求扩展名或 MIME。

空文件判为 `text`。

## 8 失效识别与清理

`Validate` 逐项复算，返回 `Check{Entry, Problem, Detail}`：

| `Problem` | 含义 |
|---|---|
| `record_invalid` | 索引记录字段不合法 |
| `blob_missing` | 内容文件缺失 |
| `blob_not_regular` | 内容路径不是普通文件 |
| `blob_unreadable` | 内容文件读不出来 |
| `size_mismatch` | 内容文件字节数与记录不一致 |
| `digest_mismatch` | 复算摘要与记录不一致 |
| `category_mismatch` | 复算类别与记录不一致 |

`Cleanup` 的规则：

1. `Refs > 0` 的项一律保留；其中失效项写入 `KeptReferenced` 报告，不删除。
2. `Refs == 0` 的项回收：有效项记入 `RemovedUnreferenced`，失效项记入 `RemovedInvalid`。
3. 先保存索引（去掉记录），再删除内容文件；中途失败留下的文件在下一次清理中按孤儿处理。
4. `blobs` 目录中没有任何索引记录引用的文件（含遗留的 `.tmp-*`）作为孤儿删除，记入 `RemovedOrphans`。
5. `BytesFreed` 汇总本次释放的字节数。

`Reset` 清空全部记录与内容文件，保留目录与权限；用于索引损坏后的重建。

调用时机建议：启动时 `Validate` + `Cleanup`（标注失效、回收孤儿），提交受理或草稿删除后对相应项 `Release`，退出前不需要额外动作。

## 9 Go 接口

包路径 `internal/attachments`，只依赖标准库。

```go
func Open(dataDir string, limits Limits) (*Store, error) // dataDir 是应用数据根目录
func DefaultLimits() Limits

func (s *Store) Stage(path string) (Entry, error)
func (s *Store) Get(id string) (Entry, error)
func (s *Store) List() ([]Entry, error)
func (s *Store) Read(id string) ([]byte, error)
func (s *Store) Release(id string) error
func (s *Store) Validate() ([]Check, error)
func (s *Store) Cleanup() (CleanupReport, error)
func (s *Store) Reset() error
func (s *Store) Dir() string
func (s *Store) Limits() Limits
```

错误：

- `*LimitError`：超限，字段见 §5。
- `*UnsupportedError`：内容既不是文本也不是图片；`errors.Is(err, ErrUnsupported)` 成立。
- `ErrNotFound`、`ErrNotRegular`、`ErrIndexCorrupt`、`ErrIndexVersion`：哨兵错误，可 `errors.Is` 判断。

调用约定：

- 每次成功 `Stage` 视为一次挂载，调用方在附件被移除、草稿丢弃或提交被受理后调用一次 `Release`。
- 上层按 `ID` 去重上传；同一 `ID` 只应产生一次资源写入。
- 上传前用 `Validate` 或 `Read` 确认内容可用；失效项不应提交。
- `Read` 返回暂存区副本，来源文件之后的变化不影响结果。

## 10 验收证据

执行环境：本机 macOS / Go 1.24.3，仓库根目录。档位：本机测试（固定样例）。

```text
$ go test ./internal/attachments -count=1
ok      github.com/arcships/dimagent-mygo/internal/attachments   1.088s

$ go vet ./...
（无输出，退出码 0）

$ go test ./... -count=1
（仓库整体：本领域与 workspace 通过；他域仍在并行写入，其测试失败与本领域无关，
  最终以各域落地后的集成复跑为准）
```

逐项对照：

| 验收项 | 证据（测试） | 观察结果 |
|---|---|---|
| 正常暂存 | `TestStageTextAttachments`、`TestStageImageCategory`、`TestStageCopiesContent`、`TestStageClassifiesByFullContent` | 返回标识、名称、类别、字节数与 SHA-256 摘要；内容为副本；`List`、`Read`、`Validate` 一致；文件头之后出现二进制的混合内容按完整内容拒绝 |
| 超限拒绝 | `TestStageRejectsOversizeFile`、`TestStageRejectsOversizeTotal`、`TestStageOversizeImageUsesImageLimit`、`TestStageAcceptsExactlyAtLimit` | 单文件与总大小超限返回 `*LimitError`，含字节数与上限；不留下暂存项或内容文件；恰好达到上限可暂存且内容不被截断 |
| 重复文件去重 | `TestStageDeduplicatesByDigest`、`TestStageDeduplicatesConcurrently`、`TestTotalLimitIgnoresDuplicateContent`、`TestStageRepairsCorruptBlobOnDuplicate`、`TestStageRepairsMissingBlobOnDuplicate` | 同一内容返回同一标识，内容文件数为 1，引用计数累加；并发去重成立；重复挂载不增加总量占用；内容文件被改写或删除时在重复挂载中修复 |
| 失效识别与清理 | `TestValidateDetectsInvalidEntries`、`TestValidateDetectsInvalidRecord`、`TestCleanupRemovesInvalidUnreferenced`、`TestCleanupKeepsReferencedInvalid`、`TestCleanupRemovesUnreferencedAndOrphans`、`TestStageReclaimsReleasedEntries` | 缺失、字节数不一致、摘要不一致、路径不是普通文件、类别不一致、记录不合法均能识别；清理删除无引用失效项，保留仍被引用的失效项，回收孤儿文件 |
| 权限位 | `TestPermissions` | 暂存区目录与 `blobs` 目录 0700；`index.json` 与内容文件 0600；重开后索引权限被重新收紧 |
| 重启可识别 | `TestRestartKeepsEntriesAndReferences`、`TestOpenRejectsUnsupportedIndex` | 重启后暂存项、引用计数与有效性保持；不支持的索引版本与损坏索引被拒绝 |
| 复位 | `TestReset` | 记录与内容文件清空，目录保留，可继续暂存 |

所有测试通过，`go vet` 无报告；本领域未引入第三方依赖，未改 `go.mod`。

## 11 已知限制

1. **不做提交与上传**：本领域只提供暂存内容与元数据；资源写入、图片内容条目与消息体属于写登记，尚未实现。R 功能块 12 的四条完成标志依赖提交链路，本版只能提供「超限在暂存时拒绝并给出字节数」与「重复挂载只留一份内容、返回同一标识」两段证据，其余标为未实现。
2. **单进程**：用进程内互斥保证并发安全；多个进程同时写同一暂存区没有文件锁，不在本版范围（客户端是单实例）。
3. **同内容异名共享名称**：去重键是摘要，同一内容以不同文件名挂载时展示名保留首次暂存的名字。若上层需要按名区分，应在选择入口先做判断。
4. **来源路径不落盘**：索引只保留文件名；来源路径由上层自行持有。理由：路径不是附件身份的一部分，且暂存区不应积累无关元数据。
5. **类别只支持文本与图片**：PDF、压缩包、可执行文件等按「不受支持」拒绝；若需求扩展类别，需要新增类别并升索引版本。
6. **类别启发式的边界**：SVG 按文本处理；UTF-16 文本因含 NUL 被拒绝；BMP 未列入图片（避免「BM」文本误判）；以 `MZ` 开头的文本会被当作二进制拒绝（极罕见）。所有拒绝都带原因，不会静默改判。
7. **总大小上限默认值未经用户拍板**：默认 500 MB，属 Q-3 的待确认数值；单文件上限按 R7 的 10 MB / 8 MB 折算为十进制字节。
8. **引用计数依赖调用方配对**：只 `Stage` 不 `Release` 会持续占用总大小额度，直到调用 `Cleanup` 且引用归零。上层必须遵守 §9 的调用约定。
9. **索引损坏不自动丢弃**：`Open` 返回错误，由上层决定是否 `Reset`；本领域不替上层做丢弃决定。
10. **校验成本**：`Validate`/`Cleanup` 会对每个内容文件全量复算摘要，条目多时较慢；这是为了能发现「同长度但内容被改写」的失效项。
11. **没有过期时间**：暂存项不会按时间自动删除，回收只由引用计数与清理触发。

## 12 变更记录

| 日期 | 变更 | 原因 |
|---|---|---|
| 2026-11-10 | 建立 D8 附件暂存第一版：暂存、单文件与总量上限、摘要去重、文本/图片分类、失效识别与清理、权限位与重启恢复 | 按组织侧开工包实现第一版领域结果包 |
