# 工具渲染与 Markdown 结构覆盖矩阵（参考产品运行时采集）

采集对象：参考产品桌面端聊天时间线（参考产品仓库根记作 `<ref>`，定位写作 `<ref>/<相对路径>:<行号>`，相对 `packages/desktop/packages/renderer/src`）。

本文件只记录**采到了什么、怎么触发的、证据在哪、缺什么与为什么缺**；不写需求条目、不写验收判定。

## 0. 口径与方法

- **触发**：在参考产品运行实例的同一个会话里，用输入框逐条发提示词，一轮只验证一组结构；等本轮结束（"停止回复"按钮消失、输入框恢复）再发下一条。
- **列表事实**：工具渲染器清单来自 `<ref>/components/chat-workspace/timeline/tool-renderers/index.ts:29`（13 个注册项，generic 兜底最后）与 `.../timeline-helpers.ts:342`（工具名 → 分类）；本轮会话实际可用的工具名由模型自述 + 实际调用双向核对（见 §1 注）。
- **两态**：对每个工具行/思考块，先收起截一张、点折叠头展开再截一张；折叠头与展开面板的元素清单来自 DOM（`data-testid`）。
- **动画参数**：全部用 `getComputedStyle` 实测（`transition-*` / `animation-*` / `grid-template-rows` / `transform`），不取源码估计值；源码里的 CSS 只用于解释实测值的来源（§5 给出源码行号）。
- **证据目录**：`docs/domains/requirements/runtime/session/`
  - `turns/`：整轮级 DOM 快照（`*.json`）+ 视口逐屏截图（`*.png`、`*-pN.png`）
  - `blocks/`：逐行两态截图（`*-collapsed.png` / `*-expanded.png` / `*-expanded-bottom.png`）与实测参数 JSON（`*.states.json`、`blk-*.meta.json`）
- **路径写法**：会话工作区记作 `<ws>`，临时夹具目录记作 `<tmp>`，夹具文件记作 `<tmp>/probe.md`、`<tmp>/artifact.md`。
- 计数：**变体**按"渲染器 × 可见形态"计（同一渲染器的不同标签/面板/状态各算一行）。

## 1. 工具渲染器覆盖（13 个注册项 / 40 个变体）

注：本会话工具集实测为 `read`、`write`、`edit`、`glob`、`grep`、`exec`、`exec_stop`、`agent`、`ask_user`、`branch`、`plugin`、`recall`、`settings`、`skill`、`wait`、`web_fetch`、`web_search`、`member_approval_allow/reject`，以及同一 MCP server 暴露的 22 个 `mcp_*` 工具（`MCP · dim/desktop_bro`）。`todo_write`/`plan_read`/`plan_write`/`browser`/`cua-*`/图像生成不在其中——这是 §4 未采的主因。

| # | 渲染器（`renderer.id`） | 变体 | 触发方式（提示词/操作） | 收起态证据 | 展开态证据 | 展开后追加（实测 `data-testid`） | 状态 |
|---|---|---|---|---|---|---|---|
| 1 | `file` | `read` 普通 | "用 read 工具读 `<tmp>/probe.md`" | `blocks/t-batch-b1-collapsed.png` | 无行内展开；点击打开右侧预览 `blocks/blk-read-right-panel.png` | 右侧预览面板（非行内） | 已采 |
| 2 | `file` | `read` offset+limit | "读 `<tmp>/probe.md`：`offset=2 limit=3`" | `blocks/t-read-b1-collapsed.png` | 同上（`t-read-b1-expanded.png`，行高不变） | 折叠头文案变 `「{file} 第2到4行」` | 已采 |
| 3 | `file` | `read` 仅 offset | "再只用 `offset=2`" | `blocks/t-read-b2-collapsed.png` | 同上 | 折叠头文案变 `「{file} 第2行起」` | 已采 |
| 4 | `file` | `read` 缓存 blob（`readCachedContent`） | 需读取超内联预算的大文件、结果落到 session blob | — | — | 折叠头文案 `「缓存内容」` | 未采：需构造超预算读取，且路径落到会话 blob 目录 |
| 5 | `file` | `write` | "新建 `<tmp>/artifact.md` 写入 H1 与三行列表" | `blocks/t-write-edit-b1-collapsed.png`、`blocks/t-batch-b8-collapsed.png` | `...-b1-expanded.png`、`...-b8-expanded.png` | `timeline-tool-file-panel` + `tool-code-block`（写入内容预览）+ 折叠头 `+N` 行数 | 已采 |
| 6 | `file` | `edit` | "把 H1 改成「覆盖演示」" | `blocks/t-write-edit-b2-collapsed.png` | `blocks/t-write-edit-b2-expanded.png` | `timeline-tool-file-panel` + `timeline-tool-file-path-open` + `tool-code-block`（+A/-D 计数在折叠头） | 已采 |
| 7 | `command` | `exec` 前台 | "执行 `ls -la`" | `blocks/t-batch-b4-collapsed.png` | `blocks/t-batch-b4-expanded.png` | `timeline-tool-exec-panel`（命令 + 输出） | 已采 |
| 8 | `command` | `exec` 后台（`runInBackground`） | "`exec` 后台跑 `sleep 15 && echo bg-done`" | `blocks/t-exec-bg-b1-collapsed.png` | `blocks/t-exec-bg-b1-expanded.png` | `timeline-tool-exec-panel`；折叠头文案 `「后台执行」` | 已采 |
| 9 | `search` | `grep` | "用 grep 搜 `TODO|FIXME|HACK` 列前 5 条" | `blocks/t-batch-b3-collapsed.png` | `blocks/t-batch-b3-expanded.png` | `timeline-tool-search-panel` | 已采 |
| 10 | `search` | `glob` 带 pattern | "`glob` `*.md`" | `blocks/t-glob-b3-collapsed.png` | `blocks/t-glob-b3-expanded.png` | `timeline-tool-search-panel` | 已采 |
| 11 | `search` | `glob` 列目录（`listFiles`） | "只给 path 不给 pattern" | `blocks/t-glob-b1-collapsed.png`（文案 `「列出文件 <tmp>」`） | `blocks/t-glob-b1-expanded.png` | `timeline-tool-search-panel`（本例为参数错误回显） | 已采 |
| 12 | `search` | 搜索错误态 | 同上（pattern 必填，工具返回错误） | `blocks/t-batch-b3-collapsed.png` 同族 | `blocks/t-glob-b1-expanded.png` | 错误文本走 `Plain Text` 代码块 | 已采 |
| 13 | `web` | `web_search` | "用 web_search 搜 OpenAI 官网" | `blocks/t-web-b2-collapsed.png` | 同图（无面板，行高不变） | 折叠头追加 `✗`；无面板 | 已采（错误态） |
| 14 | `web` | `web_search` 成功态 | 同上 | — | — | 结果列表 `web-results` + favicon 批次 | 未采：本机抓取被环境阻断，只回错误（见 §4） |
| 15 | `web` | `web_fetch` | "用 web_fetch 抓 `https://example.com`" | `blocks/t-web-b1-collapsed.png`、`blocks/t-batch-b9-collapsed.png` | `blocks/t-web-b1-expanded.png`、`blocks/t-batch-b9-expanded.png` | `timeline-tool-web-panel` + `favicon-img` + `timeline-tool-web-error` | 已采（错误态） |
| 16 | `web` | `web_fetch` 成功态 | 同上 | — | — | 正文预览 | 未采：同上 |
| 17 | `mcp` | MCP 工具（带主参数） | "用 MCP 浏览器工具打开 `https://example.com`" | `blocks/t-mcp-b1-collapsed.png` | `blocks/t-mcp-b1-expanded.png` | `timeline-tool-raw-panel`（输入/输出）；折叠头为 `MCP · dim/desktop_…` + 主参数 | 已采 |
| 18 | `mcp` | MCP 工具（无主参数） | 同轮后续 3 次 MCP 调用 | `blocks/t-mcp-b3-collapsed.png` | `blocks/t-mcp-b3-expanded.png` | 同上 | 已采 |
| 19 | `agent` | `action=create` | "创建两个子代理：统计 `<tmp>` 下 .log 数量 / 列顶层 5 个目录" | `blocks/t-agent-b1-collapsed.png` | `blocks/t-agent-b1-expanded.png` | `timeline-tool-raw-panel` | 已采 |
| 20 | `agent` | `action=list` | "再用 agent 列出所有代理状态" | `blocks/t-agent-b7-collapsed.png`（`「查询智能体状态」`） | `blocks/t-agent-b7-expanded.png` | `timeline-tool-agent-panel`（每行 provider 图标 + 名称/状态；本次 provider 图标渲染为 `?` 占位，行内文本为空） | 已采 |
| 21 | `agent` | `action=cancel` | "取消其中一个任务" | `blocks/t-agent-b9-collapsed.png`（`「取消智能体任务」`） | `blocks/t-agent-b9-expanded.png` | `timeline-tool-raw-panel` | 已采 |
| 22 | `agent` | `action=query_provider_model` | 需模型查询动作 | — | — | 折叠头 `「查询可用模型」`，面板列模型 | 未采：工具集不含该动作 |
| 23 | `agent` | `action=query_external_model` | 需外部智能体模型 | — | — | 折叠头 `「查询外部智能体模型」` | 未采：同上 |
| 24 | `wait` | 结束（`reason=notification`） | "拿到 taskId 后用 wait 等它结束" | `blocks/t-exec-bg-b2-collapsed.png`（`「结束等待」`） | 同图（无面板，`hasDetails=false`） | 无 | 已采 |
| 25 | `wait` | 等待中（具名） | 需在后台任务未结束时截取 | — | — | 折叠头 `「等待 · {name}」` | 未采：等待窗口 <1s，来不及截；无"等待中"稳定态可停 |
| 26 | `wait` | 等待中（任意） | 同上 | — | — | `「等待后台任务」` | 未采：同上 |
| 27 | `wait` | 超时 / 停止 | 需 `timeoutMs` 命中或用户 steer | — | — | `「等待超时（{duration}）」`/`「停止等待」` | 未采：需等满超时或中途 steer |
| 28 | `ask_user` | 待答 | "用 ask_user 问我一道单选题（A/B/C）" | `blocks/t-ask-user-b1-collapsed.png` | `blocks/t-ask-user-b1-expanded.png` + 面板 `turns/11-tool-ask-user-pending.png` | `timeline-tool-raw-panel`；行外另挂提权/提问面板（`Shell 选择` + 单选 + 拒绝/提交） | 已采 |
| 29 | `ask_user` | 已答 | 选 B 后提交 | `blocks/t-ask-user-b1-collapsed.png` | 同图 | 折叠头主参数换成答案摘要 | 已采 |
| 30 | `ask_user` | 超时（`toolAskUser.timedOut`） | 需放任不答直到超时 | `turns/10-tool-ask-user-timeout.png` | — | 折叠头 `「未作答（{duration}）」` | 未采：等待期被自动回答（≈7.5min 后返回答案） |
| 31 | `skill` | 带描述 | "用 skill 调用 pptx 技能" | `blocks/t-skill-b1-collapsed.png`（`「使用技能 pptx」`） | `blocks/t-skill-b1-expanded.png` | `timeline-tool-skill-panel`（技能描述） | 已采 |
| 32 | `skill` | 无描述（`hasDetails=false`） | 需返回无 description 的技能 | — | — | 无折叠箭头 | 未采：本机技能均返回描述 |
| 33 | `generic` | 已知名工具（`recall`/`settings`/`plugin`） | "依次用 recall、settings、plugin 各做一次最小调用" | `blocks/t-generic-b1-collapsed.png`…`-b3-collapsed.png` | `blocks/t-generic-b1-expanded.png`…`-b3-expanded.png` | `timeline-tool-raw-panel`（输入/输出） | 已采 |
| 34 | `generic` | 未分类工具（`exec_stop`、`member_approval_allow`） | "后台起 `sleep 60` 再用 exec_stop 停掉" | `blocks/t-exec-stop-b2-collapsed.png`、`blocks/t-agent-b4-collapsed.png` | `blocks/t-exec-stop-b2-expanded.png`、`blocks/t-agent-b4-expanded.png` | `timeline-tool-raw-panel` | 已采 |
| 35 | `todos` | 新建（`todosCreated`） | 需 `todo_write`/`todowrite` 工具 | — | — | `timeline-tool-todos-panel` | 未采：本会话无 todo 工具（模型自述"计划工具在当前会话中不存在"） |
| 36 | `todos` | 部分完成（`todosPending`） | 同上 | — | — | 同上 + 计数 | 未采：同上 |
| 37 | `todos` | 全部完成（`todosAllDone`） | 同上 | — | — | 同上 | 未采：同上 |
| 38 | `browser` | `browser` 动作族（`navigate`/`cdp`/`screenshot`…） | 工具名 `browser` | — | — | 动作专属主参数 + 结果预览 | 未采：本会话无 `browser` 工具；浏览器能力经 MCP 暴露（已按 §1 #17/#18 采） |
| 39 | `cua` | `cua-driver__*` | 需 computer-use 驱动 | — | — | `cua-summary` + 结果预览 + 截图卡片 | 未采：无 computer-use 工具 |
| 40 | `generic` | 图像生成（`toolDisplay.generateImage`） | 需图像生成工具/权益 | — | — | 生成结果走独立 `generated-image` 块（见 §2） | 未采：无该工具 |
| — | 折叠头结构（所有工具行共用） | — | — | 见 `blocks/*-collapsed.png` | — | 图标（`timeline-tool-icon`）+ 标签（`timeline-tool-summary-label`）+ 主参数（`timeline-tool-primary-parameter`，可空）+ 状态/计数徽标 + 箭头（`timeline-tool-chevron`） | 已采 |
| — | 折叠头结构（思考块） | — | — | `blocks/t-batch-b0-collapsed.png` | `blocks/t-batch-b0-expanded.png` | 灯泡图标 + 「思考了 Xs」+ 箭头；展开追加 `timeline-thinking-quote`（思考原文引用块） | 已采 |

变体计数：40 行；**已采 24**，未采 16（原因见各行与 §4）。

## 2. 非工具时间线块（18 类）

| # | 块（`TimelineBlockKind` / 组件） | 触发方式 | 收起态证据 | 展开/悬停态证据 | 展开后追加 | 状态 |
|---|---|---|---|---|---|---|
| 1 | 用户消息气泡 | 任意一轮 | `turns/13-md-headings-p1.png` | 同图 | — | 已采 |
| 2 | 用户消息轨 + 操作条（时间/复制/编辑/分叉/分支） | 悬停用户气泡 | `blocks/blk-user-actions-hover.png` | 同图（悬停才显形） | 4 个图标按钮 | 已采 |
| 3 | 助手文本块 | 任意一轮 | `turns/13-md-headings-p1.png` | — | — | 已采 |
| 4 | 思考块（`thinking`） | 开启思考后任一轮 | `blocks/t-batch-b0-collapsed.png` | `blocks/t-batch-b0-expanded.png` | `timeline-thinking-quote` | 已采 |
| 5 | 工具行（`tool`） | 见 §1 | 见 §1 | 见 §1 | 见 §1 | 已采 |
| 6 | 工具组头（`timeline-block-group`，「思考 N 轮 · 读 N 次文件」） | 同一轮 ≥2 个块时自动分组 | `blocks/blk-block-group-collapsed.png` | `blocks/blk-block-group-expanded.png`（左侧竖轨 `timeline-block-group-rail`） | 组内块列表 + 竖轨 | 已采 |
| 7 | 工作摘要条（`timeline-work-summary`，「已经工作了 Xs · …」） | 旧轮次折叠后默认显示 | `blocks/blk-work-summary-collapsed.png`、`-hover.png` | `blocks/blk-work-summary-expanded.png`（展开本轮块） | 本轮全部块 | 已采 |
| 8 | 压缩分隔（`compression-divider`） | 输入 `/compact` 执行「压缩当前上下文」 | `blocks/blk-compression-collapsed.png` | `blocks/blk-compression-expanded.png` | `compression-summary-panel`（本次为「无法加载摘要。重试」失败态） | 已采（摘要失败态） |
| 9 | 产物列表（`timeline-artifacts`） | 在会话工作区内新建交付文件（`.html`/`.md`/…） | `blocks/blk-artifacts-collapsed.png` | `blocks/blk-artifacts-expanded.png`、下拉 `blk-artifacts-dropdown.png` | `timeline-artifacts-list` + 每卡 `打开` + 下拉（外部打开/在文件夹显示） | 已采 |
| 10 | 文件改动（`timeline-file-changes`） | 同一轮内 write/edit 后（仅最新轮） | `blocks/blk-file-changes-collapsed.png`（`1 个文件改动`） | `blocks/blk-file-changes-expanded.png` | `timeline-file-changes-list`（文件行、右键菜单、撤销/重做） | 已采 |
| 11 | 助手操作条（复制/分叉/重试 + 完成时间） | 悬停助手消息尾部 | `blocks/blk-assistant-actions-hover.png` | 同图 | 3 个图标 + `timeline-assistant-completed-time` | 已采 |
| 12 | 生成图片（`generated-image`） | 需图像生成工具 | — | — | `generated-image-result` + 操作条 | 未采：无图像生成工具/权益 |
| 13 | 计划草稿（`plan-draft`，`plan_read`/`plan_write`） | 需 plan 工具 | — | — | 草稿面板 | 未采：无可用的 plan 工具 |
| 14 | 重试提示（`retry`） | 需 provider 重试 | — | — | 重试原因/倒计时 | 未采：本机 provider 未触发重试 |
| 15 | 错误边界（`TimelineErrorBoundary`） | 需渲染异常 | — | — | 错误兜底 | 未采：无法在不注入故障的前提下触发 |
| 16 | 窗口截断分隔（`TimelineWindowTruncationDivider`） | 需时间线窗口截断 | — | — | 截断提示 | 未采：本会话消息量未触发 |
| 17 | 上下文引用列表（`timeline-context-references`） | 需消息带 @ 文件引用 | — | — | 引用 chip 列表 | 未采：未构造引用消息 |
| 18 | 工具排队态（`timeline-tool-queued`） | 工具已派发未执行 | — | — | 排队提示 | 未采：窗口 <1s，未捕获 |

块计数：18 类；已采 13，未采 5。

## 3. Markdown 结构（22 项）

触发：一条"直接输出 Markdown、不要用工具"的提示词（`turns/13-md-headings-*` 与 `turns/02-md-structures-*`）。

| # | 结构 | 证据 | 实测渲染 |
|---|---|---|---|
| 1 | H1 | `turns/13-md-headings-p1.png` | 粗体大字号，段前后间距 |
| 2 | H2 | 同上 | 次一级 |
| 3 | H3 | 同上 | 再次一级 |
| 4 | H4 | 同上 | 更小 |
| 5 | H5 | `turns/13-md-headings-p1.png`、`-p2.png` | 更小、加粗 |
| 6 | H6 | `turns/13-md-headings-p2.png` | 最小一级，与正文接近但加粗 |
| 7 | 无序列表（3 层嵌套） | `turns/13-md-headings-p2.png` | 圆点逐层缩进 |
| 8 | 有序列表（3 层嵌套） | 同上 | 数字逐层缩进 |
| 9 | 引用 + 嵌套引用 | `turns/13-md-headings-p3.png`、`turns/02-md-structures-p6.png` | 左侧竖条 + 缩进 |
| 10 | 表格（左/中/右对齐） | `turns/13-md-headings-p3.png` | 表头底色、边框、列对齐生效 |
| 11 | 行内代码 | `turns/13-md-headings-p3.png`、`turns/01-batch-tools-md.png` | 圆角底色 chip |
| 12 | 代码块 `ts` | `turns/14-md-codeblocks-p1.png` | Shiki 高亮 + 语言条（图标+名称）+ 复制/折叠/更多 |
| 13 | 代码块 `go` | `turns/14-md-codeblocks-p1.png` | 同上 |
| 14 | 代码块 `sh` | `turns/14-md-codeblocks-p1.png`、`turns/02-md-structures-p8.png` | 同上（Shell 图标） |
| 15 | 代码块 `python` | `turns/02-md-structures-p7.png`、`turns/14-md-codeblocks-p1.png`、`blocks/blk-read-right-panel.png` | 同上（Python 图标） |
| 16 | 代码块 `json` | `turns/14-md-codeblocks-p2.png`、`turns/02-md-structures-p7.png` | 同上（`{}` 图标 + `-N/+N` 差异计数在 diff 时出现） |
| 17 | 任务列表（勾选/未勾选） | `turns/02-md-structures-p6.png` | 方形复选框，选中为蓝底白勾 |
| 18 | 分隔线 | `turns/02-md-structures-p6.png` | 1px 细线 |
| 19 | 外部链接 | `turns/02-md-structures-p6.png` | 蓝色带下划线，点击走外链动作 |
| 20 | 文件链接（本地文件 chip） | `turns/02-md-structures-p6.png`、`turns/01-batch-tools-md.png` | 文件图标 + 名称 chip，点击打开右侧预览 |
| 21 | 远程图片 | `turns/02-md-structures-p5.png` | 圆角、限宽、懒加载占位 |
| 22 | 行内数学 / 块级数学 | `turns/02-md-structures-p4.png` | KaTeX：`E=mc²`、`∫₀¹x²dx=1/3`、求和式居中 |
| 23 | Mermaid 流程图 | `turns/02-md-structures-p3.png` | 独立卡片：`预览/源码` 切换、复制、下载、放大、缩放条（缩放/100%） |
| 24 | Mermaid 时序图 | `turns/02-md-structures-p2.png` | 同上 |
| 25 | diff 围栏块 | `turns/02-md-structures.png` | 双栏并排 diff：红/绿行底、行号、`-N +N` 计数 |
| 26 | 脚注引用 + 定义 | `turns/02-md-structures.png` | 上标 `[1]` + 文末定义 + 回跳箭头 |
| 27 | 删除线 / 粗体 / 斜体 / 粗斜体 | `turns/02-md-structures.png` | 逐项生效 |
| 28 | 斜杠命令菜单（输入区附属） | `turns/slash-menu.png`、`turns/slash-compact.png` | `/` 触发指令 + 技能两段列表，右对齐说明文案 |

Markdown 计数：28 项，全部已采（数学/mermaid/diff 均按实际渲染记录）。

## 4. 未采清单与原因

| 未采项 | 原因 | 若要采需 |
|---|---|---|
| `todos` 渲染器（3 变体） | 本会话工具集无 `todo_write`/`todowrite`；模型明确回复"计划工具在当前会话中不存在" | 打开带 plan/todo 工具的配置（或换 provider 配置） |
| `browser` 渲染器 | 工具集无 `browser`；浏览器能力以 MCP 工具形式提供 | 启用内置 browser 工具 |
| `cua` 渲染器 | 无 computer-use 驱动工具 | 接入 cua-driver |
| `web_search` / `web_fetch` 成功态 | 本机网络策略/服务未配置，两个工具均返回错误（错误文案指向 `<ref>/docs/design/plugins/Web.md`） | 配置可用的抓取/搜索服务 |
| `read` 缓存 blob 变体 | 需超内联预算的读取并落到会话 blob | 读大文件并让结果走 blob 存储 |
| `wait` 等待中/超时/停止（4 变体） | 等待态窗口极短；超时需等满；停止需人工 steer | 构造长任务 + 目视抓取，或注入 steer |
| `ask_user` 超时变体 | 等待约 7.5 分钟后被自动回答（返回"已收到你的回答"） | 关闭自动回答 |
| `agent` 模型查询（2 变体） | 工具集动作不含 `query_provider_model`/`query_external_model` | 打开相应配置 |
| `skill` 无描述变体 | 本机技能均返回 description | 造一个无描述技能 |
| 生成图片块 | 无图像生成工具/权益 | 开通图像生成 |
| 计划草稿块 | 无 `plan_read`/`plan_write` | 打开计划模式 |
| 重试提示块 | provider 未重试 | 注入 5xx/限流 |
| 错误边界 | 需渲染期异常 | 注入故障（会破坏采集环境，未做） |
| 窗口截断分隔 | 消息量未达截断阈值 | 长会话推进到截断 |
| 上下文引用列表 | 未构造 @ 引用消息 | 用 @ 引用文件后发送 |
| 工具排队态 | 派发到执行的间隔 <1s | 高并发工具轮次连拍 |

## 5. 折叠动画参数（`getComputedStyle` 实测）

采集于本轮工具行/思考块/组头/摘要条（逐块实测值见 `blocks/*.states.json`）。

### 5.0 折叠头与展开区元素（DOM 实测）

| 行类型 | 折叠头元素（左→右） | 展开区追加 |
|---|---|---|
| 工具行 | 图标 `timeline-tool-icon`（16px lucide）+ 标签 `timeline-tool-summary-label`（如「读取」「执行」「使用技能」）+ 主参数 `timeline-tool-primary-parameter`（可空，参数变更时重放 `param-swap-in`）+ 状态/计数徽标（`+N`/`-N`、`✗`、favicon 批次）+ 箭头 `timeline-tool-chevron`（16px） | `.collapse-grid > .collapse-grid-inner > .py-1`：渲染器专属面板（`timeline-tool-exec-panel` / `-file-panel` / `-search-panel` / `-web-panel` / `-skill-panel` / `-agent-panel` / `-todos-panel` / `-raw-panel`）与 `tool-code-block`、`tool-parameter-preview`、`timeline-tool-file-path-open`、`timeline-tool-web-error` |
| 思考行 | 灯泡图标 `timeline-thinking-icon` + 「思考了 Xs」+ 箭头 | `timeline-thinking-quote`（思考原文引用块） |
| 工具组头 | 组图标 `timeline-block-group-icon` + 摘要 `timeline-block-group-summary`（「思考 N 轮 · 读 N 次文件」）+ 箭头 `timeline-block-group-toggle` | 组内块列表 + 左侧竖轨 `timeline-block-group-rail`（`timeline-block-group-content`） |
| 工作摘要条 | 分隔线 + 文案「已经工作了 Xs · 动作统计」+ 箭头（`timeline-work-summary`） | 该轮次全部块 |
| 文件改动 | 计数「N 个文件改动」+ 箭头（`timeline-file-changes-toggle`） | `timeline-file-changes-list`（每行 `-file`/`-status`/`-undo`/`-reapply` + 右键菜单） |
| 产物 | 计数「N 个产物」 | `timeline-artifacts-list`（每卡 `timeline-artifact-card` + `-open` + `-download` + `-dropdown-trigger`） |
| 压缩分隔 | 文案「上下文已压缩」+ 摘要开关（`compression-summary-toggle`） | `compression-summary-panel` / `compression-summary-plaintext`；失败态给 `compression-failure-message/code/details` |

| 对象 | 选择器 | 实测属性 | 实测值 |
|---|---|---|---|
| 折叠容器 | `.collapse-grid` | `transition-property` | `grid-template-rows` |
| | | `transition-duration` / `timing-function` / `delay` | `0.3s` / `cubic-bezier(0.16, 1, 0.3, 1)` / `0s` |
| | | `grid-template-rows`（收起 → 展开） | `0px` → 实测像素（例：`95.5px`、`250.5px`） |
| 面板内层 | `.collapse-grid-inner` | `transition-property` | `opacity` |
| | | `duration` / `timing-function` | `0.3s` / `ease-in` |
| | | `overflow` / `min-height` | `hidden` / `0px` |
| 折叠箭头 | `[data-testid=timeline-tool-chevron]` | `transition-property` | `color, transform` |
| | | `duration` / `timing-function` | `0.15s` / `cubic-bezier(0.4, 0, 0.2, 1)` |
| | | 展开态类 | `rotate-90`（收起无该类，`transform: none`） |
| 行根 | `[data-testid=timeline-tool-block]` | 内联 `transition-delay` | `40ms × data-enter-index`（实测 index=1 → `40ms`） |
| 主参数 | `.param-enter-anim` | `animation-name` / `duration` / `timing-function` | `param-swap-in` / `0.2s` / `ease-out`（`opacity: 0→1`） |
| 参数替换（离开） | `.param-swap-leave-active` | `transition` | `opacity 150ms ease-in` |
| 组内块入场 | `.group-item-enter-active` | `transition` | `opacity 240ms ease-out, transform 240ms cubic-bezier(0.16, 1, 0.3, 1)`（`translateY(6px)→0`） |
| 组内块离开 | `.group-item-leave-active` | `transition` | `opacity 150ms ease-in` |
| 展开面板挂载 | 组件逻辑（`panelMounted`） | 行为 | 展开当帧即挂载；收起后才卸载，保证收起动画有真实内容 |
| 减少动效 | `@media (prefers-reduced-motion: reduce)` | 行为 | 动画置 `none`、参数动画关闭 |

源码对照（用于解释实测值来源，非替代实测）：
- `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:1174`（`.collapse-grid`）、`:1186`（`.collapse-grid-inner`）、`:799`（箭头 `rotate-90`）
- `<ref>/components/chat-workspace/timeline/TimelineBlockGroup.vue:575`（同款折叠）、`:590`（组内块入场/离开）
- `<ref>/styles/timeline-animation.css:17`（`param-swap-in` 200ms）
- `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:589`（`read` 行点击走右侧预览，不展开行内面板）

## 6. 计数汇总

| 类别 | 总变体 | 已采 | 未采 |
|---|---|---|---|
| 工具渲染器（13 注册项） | 40 | 24 | 16 |
| 时间线块类型 | 18 | 13 | 5 |
| Markdown 结构 | 28 | 28 | 0 |
| 合计 | 86 | 65 | 21 |

## 7. 采集环境与限制

- 采集在参考产品运行实例（Electron + CDP 9222）上进行，会话工作区为 `<ws>`；文件类夹具写 `<tmp>`（产物、文件改动块需落在会话工作区内才会显示）。
- 工具授权逐次点击「仅这次 / 允许此会话期间访问」放行；`ask_user` 面板用手动选择 A/B/C 后「提交」。
- 会话内执行 `/compact` 后旧轮次被折叠（`上下文已压缩`），旧轮次的行内块不再挂载——§1/§2 的两态证据均在折叠前采集。
- 采集期间同一实例曾被他路提示词插入（自动审批脚本），逐轮已用"最新用户消息 == 本轮提示词"校验，被污染的轮次未计为证据。
