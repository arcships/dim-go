# 右栏与各面板：元素清单

作用：按 `docs/project-organization.md` §9.16 二，穷举参考产品 dim-electron 的右栏（文件 / 改动 / 产物）与终端抽屉的界面元素，单位固定为 **屏 → 区域 → 分组 → 元素**，每元素六项：元素 ID / 名称（文案原文）/ 控件 / 默认 / 条件 / 定位。

边界：只登记"有什么元素、长什么样、什么条件下在场"。不写行为需求（点击后做什么、失败如何重试、顺序如何），不写我方命名，不自造参考产品没有的元素。

## 0 来源与取证状态

- 事实源：参考产品源码，路径均相对 `packages/desktop/packages/renderer/src/`，记作 `<ref>/<相对路径>:<行号>`。
- 文案：取 `i18n/zh-CN.ts` 原文（键名随定位给出）；模板内硬编码文案照抄。
- 证据状态：本版为**源码抽取**，未附截图裁片。按 §9.16 七，清单只承担"元素在不在、文案是什么、由哪段模板产生"；`证据` 列待按屏跑通参考产品后补。
- 计数约定：同一元素的不同**文案**各占一行（如「展开面板」与「恢复面板宽度」）；同一文案在多处渲染只占一行，定位列出多处。
- 分批：第一批 = 右栏 + 时间线内「改动 / 产物」两块 + 终端抽屉（区域 A–J）；第二批 = 任务面板、侧聊、子代理、过程详情（区域 K–P）。两批均已落入本文件。

## 1 屏与区域划分

| 屏 | 区域 | 覆盖 | 分组数 |
|---|---|---|---|
| 右栏 | A 面板外壳与标签条 | 拖拽手柄、返回、标签页、新建标签、标签右键菜单、展开 | 1 |
| 右栏 | B 文件标签页 | 文件树、搜索态、加载更多、目录错误、行内标记、右键菜单 | 1 |
| 右栏 | C 改动（Review）标签页 | 范围选择、Git 动作、分组、行控件、内联 diff、还原确认、底部统计 | 1 |
| 右栏 | D 文件预览标签页 | 头部工具条、各内容形态、空/错态、选中气泡 | 1 |
| 右栏 | E 文档预览工具条 | 搜索、页码、缩放 | 1 |
| 时间线内块 | F 本轮改动块 | 折叠头、撤销/重新应用、文件行、右键菜单、确认弹窗 | 1 |
| 时间线内块 | G 本轮产物块 | 计数头、Git 动作、产物行、动作与下拉、展开收起 | 1 |
| 终端抽屉 | H 工具条 | 终端标签、新建终端、关闭终端 | 1 |
| 终端抽屉 | I 终端内容与提示 | 终端画布、提示行 | 1 |
| 终端抽屉 | J 右键菜单 | 复制 / 粘贴 / 清屏 / 新建 / 关闭进程 | 1 |
| 任务面板 | K 任务列表与空态 | 后台进程/唤醒共用的列表、状态图标、空态 | 1 |
| 任务面板 | L 停止按钮 | 停止进程 / 取消唤醒的两段确认按钮 | 1 |
| 任务面板 | M 后台进程详情 | 状态、复制命令、打开日志、元信息、只读输出终端 | 1 |
| 任务面板 | N 唤醒详情 | 触发时间与提示内容 | 1 |
| 侧边对话 | O 侧聊面板 | 错误条、结束条、空态、输入区、结束确认 | 1 |
| 子代理对话 | P 子代理面板 | 错误条、重试、加载态、只读时间线 | 1 |

---

# 屏：右栏

## A 面板外壳与标签条

外壳定位：`<ref>/components/right-panel/RightPanelBar.vue`（标签条）与 `<ref>/components/chat-workspace/ChatWorkspace.vue:6753`（`<aside data-testid="right-panel-shell">`）。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| RP-A-01 | 拖拽调整面板宽度 | 竖向拖拽条（4px，悬停变主色） | 显示 | 面板为并列布局（非 replacement 布局） | `<ref>/components/chat-workspace/ChatWorkspace.vue:6771`、`:2058`（`chatWorkspace.resizeChangesPanel`） |
| RP-A-02 | 返回 | 图标按钮（左箭头，`aria-label`=返回） | 隐藏 | 面板为 replacement 布局（窄窗/覆盖式） | `<ref>/components/right-panel/RightPanelBar.vue:289`（`common.back`） |
| RP-A-03 | （macOS 交通灯预留位，无文案） | 占位块（80px） | 隐藏 | replacement 布局且侧栏收起且 macOS 且非全屏 | `<ref>/components/right-panel/RightPanelBar.vue:275` |
| RP-A-04 | 展开侧边栏 ⌘B | 图标按钮 + 悬停提示 | 隐藏 | replacement 布局且侧栏收起 | `<ref>/components/right-panel/RightPanelBar.vue:280` → `<ref>/components/chat-workspace/WorkspaceHeaderNavigation.vue:35` |
| RP-A-05 | 上一个工作区会话 | 图标按钮 + 悬停提示 | 隐藏 | 标签条启用工作区导航且未隐藏会话导航（右栏内 `hide-thread-navigation` 为真，故隐藏） | `<ref>/components/chat-workspace/WorkspaceHeaderNavigation.vue:51` |
| RP-A-06 | 下一个工作区会话 | 图标按钮 + 悬停提示 | 隐藏 | 同上 | `<ref>/components/chat-workspace/WorkspaceHeaderNavigation.vue:70` |
| RP-A-07 | （标签页行，图标 + 文案 + 快捷键 + 关闭） | 标签页 | 显示 | 每个已打开标签页一行 | `<ref>/components/right-panel/RightPanelBar.vue:302` |
| RP-A-08 | （标签图标：文件标签 = 文件夹） | 图标 | 显示 | 标签页 `kind=files` | `<ref>/components/right-panel/RightPanelBar.vue:325` |
| RP-A-09 | Review | 标签页文案 | 显示 | 标签页 `kind=diff`（图标 = 差异） | `<ref>/components/right-panel/RightPanelBar.vue:324`、`:159`（`changeSummary.review`） |
| RP-A-10 | 文件 | 标签页文案 | 显示 | 标签页 `kind=files` | `<ref>/components/right-panel/RightPanelBar.vue:157`（`contextPanel.files`） |
| RP-A-11 | 浏览器 | 标签页文案（图标 = 地球） | 显示 | 标签页 `kind=browser` | `<ref>/components/right-panel/RightPanelBar.vue:159`（`toolDisplay.browser`） |
| RP-A-12 | （文件预览标签文案 = 文件名，缺路径时用「打开文件」） | 标签页文案（图标 = 文件类型图标） | 显示 | 标签页 `kind=file-preview` | `<ref>/components/right-panel/RightPanelBar.vue:160`、`:319`（`contextPanel.openFile`） |
| RP-A-13 | 终端 | 标签页文案（图标 = 方块终端） | 显示 | 标签页 `kind=terminal` 且对应终端标签无标题 | `<ref>/components/right-panel/RightPanelBar.vue:161`（`chatHeader.terminal`） |
| RP-A-14 | （终端标签标题 = shell 上报标题或工作目录末段） | 标签页文案 | 显示 | 标签页 `kind=terminal` 且终端标签有标题 | `<ref>/components/right-panel/RightPanelBar.vue:161`、`:120` |
| RP-A-15 | 工作流 | 标签页文案（图标 = 工作流） | 显示 | 标签页 `kind=workflow` 且未取到 run 名 | `<ref>/components/right-panel/RightPanelBar.vue:162`（`workflow.title`） |
| RP-A-16 | 侧边对话 | 标签页文案（图标 = 对话气泡） | 显示 | 标签页 `kind=session-branch` 且无自定义标题 | `<ref>/components/right-panel/RightPanelBar.vue:163`（`sideChat.title`）；运行中时图标换旋转圈 `:329` |
| RP-A-17 | 唤醒 `{count}` | 标签页文案（图标 = 时钟） | 显示 | 标签页 `kind=session-wakeup` | `<ref>/components/right-panel/RightPanelBar.vue:165` |
| RP-A-18 | 后台进程 `{count}` | 标签页文案（图标 = 方块终端） | 显示 | 标签页 `kind=background-processes` | `<ref>/components/right-panel/RightPanelBar.vue:166` |
| RP-A-19 | 扩展 | 标签页文案（图标 = 方块组） | 显示 | 标签页 `kind=ui-extension` | `<ref>/components/right-panel/RightPanelBar.vue:167`（`uiExtensions.title`） |
| RP-A-20 | （子代理会话标题） | 标签页文案（图标 = 对话气泡） | 显示 | 标签页 `kind=subagent-chat` | `<ref>/components/right-panel/RightPanelBar.vue:168`、`:339` |
| RP-A-21 | （MCP App 标题） | 标签页文案（图标 = 应用窗口） | 显示 | 标签页 `kind=mcp-app` | `<ref>/components/right-panel/RightPanelBar.vue:164`、`:335` |
| RP-A-22 | ⌘1…⌘9 / Ctrl+1…9 | 快捷键徽标 | 隐藏 | 按住命令键且面板展开且标签数 > 1，且该标签序号 < 9 | `<ref>/components/right-panel/RightPanelBar.vue:347`、`:1324` |
| RP-A-23 | 关闭（标签 ×） | 图标按钮（`aria-label`=关闭） | 隐藏（悬停显现） | 标签可关闭；显示快捷键徽标时该 × 不渲染 | `<ref>/components/right-panel/RightPanelBar.vue:353`（`common.close`） |
| RP-A-24 | （新建标签按钮 ＋，无文案） | 图标按钮 | 显示 | 可选新建项非空 | `<ref>/components/right-panel/RightPanelBar.vue:386` |
| RP-A-25 | 文件 | 菜单项（文件夹图标） | 显示 | 当前无 files 标签页 | `<ref>/components/right-panel/RightPanelBar.vue:136`、`:404` |
| RP-A-26 | Review | 菜单项（差异图标） | 显示 | 当前无 diff 标签页且改动功能可见 | `<ref>/components/right-panel/RightPanelBar.vue:139`、`:404` |
| RP-A-27 | 浏览器 | 菜单项（地球图标） | 显示 | 当前无 browser 标签页 | `<ref>/components/right-panel/RightPanelBar.vue:142`、`:404` |
| RP-A-28 | 终端 | 菜单项（方块终端图标） | 显示 | 终端功能可见且有工作目录 | `<ref>/components/right-panel/RightPanelBar.vue:145`、`:404` |
| RP-A-29 | 侧边对话 | 菜单项（对话气泡图标） | 显示 | 有活跃会话且非新会话 | `<ref>/components/right-panel/RightPanelBar.vue:148`、`:404` |
| RP-A-30 | 扩展 | 菜单项（方块组图标） | 显示 | 存在扩展面板且无 ui-extension 标签页 | `<ref>/components/right-panel/RightPanelBar.vue:151`、`:404` |
| RP-A-31 | 在右侧新建标签页 | 右键菜单项（面板向右图标） | 显示 | 终端功能可见 | `<ref>/components/right-panel/RightPanelBar.vue:233`、`:370`（`rightPanel.tabContext.newTabToRight`） |
| RP-A-32 | 重新加载 | 右键菜单项（刷新图标） | 显示 | 标签类型为 files / diff / browser / file-preview | `<ref>/components/right-panel/RightPanelBar.vue:240`（`rightPanel.tabContext.reload`） |
| RP-A-33 | 关闭 | 右键菜单项（× 图标） | 显示 | 该标签可关闭 | `<ref>/components/right-panel/RightPanelBar.vue:247`（`rightPanel.tabContext.close`） |
| RP-A-34 | 关闭其他标签页 | 右键菜单项（面板图标） | 显示 | 除自身外还有可关闭标签页 | `<ref>/components/right-panel/RightPanelBar.vue:254`（`rightPanel.tabContext.closeOtherTabs`） |
| RP-A-35 | 关闭右侧标签页 | 右键菜单项（面板收起图标） | 显示 | 该标签右侧还有可关闭标签页 | `<ref>/components/right-panel/RightPanelBar.vue:261`（`rightPanel.tabContext.closeTabsToRight`） |
| RP-A-36 | 展开面板 | 图标按钮（放大图标） | 显示 | 非紧凑布局且面板未展开 | `<ref>/components/right-panel/RightPanelBar.vue:417`（`rightPanel.expandPanel`） |
| RP-A-37 | 恢复面板宽度 | 图标按钮（缩小图标，按下态） | 隐藏 | 面板已展开 | `<ref>/components/right-panel/RightPanelBar.vue:417`、`:428`（`rightPanel.restoreWidth`） |

元素数：37

## B 文件标签页（FilesView + FileTreeNode）

定位：`<ref>/components/context-panel/FilesView.vue`、`<ref>/components/context-panel/FileTreeNode.vue`；筛选框与权限提示在 `<ref>/components/context-panel/ContextPanel.vue`。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| RP-B-01 | 筛选文件... | 输入框（前置搜索图标） | 显示 | 当前标签为 files 或 diff | `<ref>/components/context-panel/ContextPanel.vue:541`（`contextPanel.filterFiles`） |
| RP-B-02 | （文件树骨架条 ×3） | 骨架 | 显示 | 非搜索态、加载中且树为空 | `<ref>/components/context-panel/FilesView.vue:329` |
| RP-B-03 | 正在搜索整个工作区… | 行内状态文本 | 隐藏 | 搜索态且状态为 searching | `<ref>/components/context-panel/FilesView.vue:334`（`contextPanel.searchingWorkspace`） |
| RP-B-04 | 文件搜索失败。 | 行内错误文本（危险色） | 隐藏 | 搜索态且状态为 error | `<ref>/components/context-panel/FilesView.vue:341`（`contextPanel.searchFailed`） |
| RP-B-05 | 重试 | 文字按钮 | 隐藏 | 搜索失败行内 | `<ref>/components/context-panel/FilesView.vue:347`（`common.retry`） |
| RP-B-06 | 部分文件夹无法搜索，结果可能不完整。 | 行内警告文本（琥珀色） | 隐藏 | 搜索态且状态为 partial | `<ref>/components/context-panel/FilesView.vue:351`（`contextPanel.searchIncomplete`） |
| RP-B-07 | 仅显示前 200 个匹配项 | 行内提示文本 | 隐藏 | 搜索命中还有更多 | `<ref>/components/context-panel/FilesView.vue:359`（`contextPanel.searchResultsLimited`，`{count}=200`） |
| RP-B-08 | 无匹配文件 | 空态标题（上方圆形文件图标） | 隐藏 | 搜索完成且无命中；或树上卷为空 | `<ref>/components/context-panel/FilesView.vue:367`、`:415`（`contextPanel.noMatchingFiles`） |
| RP-B-09 | 没有找到包含“{query}”的文件 | 空态说明文本 | 隐藏 | 搜索完成且无命中 | `<ref>/components/context-panel/FilesView.vue:375`（`contextPanel.noMatchingFilesDetail`） |
| RP-B-10 | （搜索命中文件行：图标 + 文件名 + 次级目录 + 改动统计） | 行 | 显示 | 搜索态有命中，每个命中一行 | `<ref>/components/context-panel/FilesView.vue:380` → `FileTreeNode.vue:194` |
| RP-B-11 | （目录行：展开箭头 + 文件夹图标 + 目录名） | 行（可拖拽） | 显示 | 非搜索态，树中每个目录 | `<ref>/components/context-panel/FileTreeNode.vue:112` |
| RP-B-12 | （目录行改动色：有改动后代时目录名转琥珀色） | 文本态 | — | 该目录后代含改动文件 | `<ref>/components/context-panel/FileTreeNode.vue:137` |
| RP-B-13 | （匹配高亮：文件名命中部分黄底） | 文本高亮 | 隐藏 | 搜索态且文件名含关键词 | `<ref>/components/context-panel/FileTreeNode.vue:139`、`:219` |
| RP-B-14 | 加载更多 | 目录内文字按钮 | 隐藏 | 该目录存在未加载分页 | `<ref>/components/context-panel/FileTreeNode.vue:165`（`contextPanel.loadMore`） |
| RP-B-15 | 正在加载... | 目录内文字按钮（禁用态） | 隐藏 | 该目录分页加载中 | `<ref>/components/context-panel/FileTreeNode.vue:174`（`common.loading`） |
| RP-B-16 | 无法加载此文件夹。 | 目录内错误行（危险色） | 隐藏 | 展开该目录后加载失败 | `<ref>/components/context-panel/FileTreeNode.vue:178`（`contextPanel.directoryLoadFailed`） |
| RP-B-17 | 重试 | 目录内错误行尾按钮 | 隐藏 | 同上 | `<ref>/components/context-panel/FileTreeNode.vue:186`（`common.retry`） |
| RP-B-18 | （文件行：文件类型图标 + 文件名） | 行（可拖拽） | 显示 | 非搜索态，树中每个文件 | `<ref>/components/context-panel/FileTreeNode.vue:194` |
| RP-B-19 | （文件行次级目录路径） | 次级文本（11px） | 隐藏 | 搜索命中行且有目录前缀 | `<ref>/components/context-panel/FileTreeNode.vue:222` |
| RP-B-20 | +`{added}`/-`{removed}` | 统计文本 | 隐藏 | 文件有改动记录 | `<ref>/components/context-panel/FileTreeNode.vue:72`、`:227` |
| RP-B-21 | ○ | 圆点徽标（已暂存） | 隐藏 | 文件已暂存 | `<ref>/components/context-panel/FileTreeNode.vue:231` |
| RP-B-22 | （已忽略/点开头文件：文件名灰化） | 文本态 | — | `gitTracking=ignored` 或以 `.` 开头 | `<ref>/components/context-panel/FileTreeNode.vue:216` |
| RP-B-23 | （有改动文件：文件名琥珀色） | 文本态 | — | 文件有改动记录 | `<ref>/components/context-panel/FileTreeNode.vue:217` |
| RP-B-24 | 加载更多 | 树上卷末尾文字按钮 | 隐藏 | 工作区根目录存在未加载分页 | `<ref>/components/context-panel/FilesView.vue:446`（`contextPanel.loadMore`） |
| RP-B-25 | 正在加载... | 同上（禁用态） | 隐藏 | 根目录分页加载中 | `<ref>/components/context-panel/FilesView.vue:455`（`common.loading`） |
| RP-B-26 | 无法加载此文件夹。 | 根级错误行（危险色） | 隐藏 | 根目录加载失败 | `<ref>/components/context-panel/FilesView.vue:398`（`contextPanel.directoryLoadFailed`） |
| RP-B-27 | 重试 | 根级错误行尾按钮 | 隐藏 | 同上 | `<ref>/components/context-panel/FilesView.vue:404`（`common.retry`） |
| RP-B-28 | 暂存 | 右键菜单项（＋ 图标） | 显示 | 文件有改动且未暂存，且改动功能可见 | `<ref>/components/context-panel/FileTreeNode.vue:240`（`contextPanel.stage`） |
| RP-B-29 | 取消暂存 | 右键菜单项（－ 图标） | 显示 | 文件有改动且已暂存 | `<ref>/components/context-panel/FileTreeNode.vue:248`（`contextPanel.unstage`） |
| RP-B-30 | 查看差异 | 右键菜单项（差异图标） | 显示 | 文件有改动 | `<ref>/components/context-panel/FileTreeNode.vue:257`（`contextPanel.viewDiff`） |
| RP-B-31 | 预览 | 右键菜单项（眼睛图标） | 显示 | — | `<ref>/components/context-panel/FileTreeNode.vue:263`（`common.preview`） |
| RP-B-32 | 用外部工具打开 | 右键菜单项（外链图标） | 显示 | — | `<ref>/components/context-panel/FileTreeNode.vue:267`（`contextPanel.openExternal`） |
| RP-B-33 | 在文件管理器中显示 | 右键菜单项（文件夹图标） | 显示 | 链接动作上下文存在 | `<ref>/components/context-panel/FileTreeNode.vue:271`（`linkMenu.revealInFolder`） |
| RP-B-34 | 另存为… | 右键菜单项（下载图标） | 显示 | 链接动作上下文存在 | `<ref>/components/context-panel/FileTreeNode.vue:279`（`linkMenu.saveAs`） |
| RP-B-35 | 复制路径 | 右键菜单项（复制图标） | 显示 | — | `<ref>/components/context-panel/FileTreeNode.vue:288`（`contextPanel.copyPath`） |
| RP-B-36 | （右键菜单分隔线 ×3） | 分隔线 | 显示 | 随相邻菜单项条件 | `<ref>/components/context-panel/FileTreeNode.vue:256`、`:261`、`:287` |

元素数：36

## C 改动（Review）标签页（DiffView）

定位：`<ref>/components/context-panel/DiffView.vue`；范围选择、Git 动作、权限提示、底部统计在 `<ref>/components/context-panel/ContextPanel.vue`。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| RP-C-01 | 未提交改动 | 下拉触发器文案（`data-scope=uncommitted`） | 显示 | 当前范围为未提交改动 | `<ref>/components/context-panel/ContextPanel.vue:443`（`contextPanel.scope.uncommitted`） |
| RP-C-02 | 当前分支改动 | 下拉触发器文案 | 显示 | 当前范围为分支改动 | 同上（`contextPanel.scope.branch`） |
| RP-C-03 | Last Run 改动 | 下拉触发器文案 | 显示 | 当前范围为最近一次 run | 同上（`contextPanel.scope.lastRun`） |
| RP-C-04 | 未提交改动 当前工作区改动 | 下拉选项（两段文本） | 显示 | 下拉展开 | `<ref>/components/context-panel/ContextPanel.vue:452`（`scope.uncommitted` + `scope.uncommittedDetail`） |
| RP-C-05 | 当前分支改动 分支与基线差异 | 下拉选项（两段文本） | 显示 | 下拉展开 | `<ref>/components/context-panel/ContextPanel.vue:456`（`scope.branch` + `scope.branchDetail`） |
| RP-C-06 | Last Run 改动 最近一次 agent run 触达的文件 | 下拉选项（两段文本） | 显示 | 下拉展开 | `<ref>/components/context-panel/ContextPanel.vue:460`（`scope.lastRun` + `scope.lastRunDetail`） |
| RP-C-07 | 提交或推送 | 按钮（上传云图标 + 下拉箭头） | 显示 | 有会话且范围非 lastRun | `<ref>/components/context-panel/ContextPanel.vue:468`（`gitDialogs.commitOrPush`） |
| RP-C-08 | （当前分支名） | 子菜单触发器（分支图标 + 分支名） | 显示 | 提交或推送菜单展开 | `<ref>/components/context-panel/ContextPanel.vue:483` |
| RP-C-09 | （本地分支名） | 子菜单项（分支图标；当前分支带勾且禁用） | 显示 | 同上，每个本地分支一行 | `<ref>/components/context-panel/ContextPanel.vue:489` |
| RP-C-10 | 提交 | 菜单项（提交图标） | 显示 | 有已暂存文件才可用 | `<ref>/components/context-panel/ContextPanel.vue:503`（`gitDialogs.commit`） |
| RP-C-11 | 提交并推送 | 菜单项（上传云图标） | 显示 | 有已暂存文件才可用 | `<ref>/components/context-panel/ContextPanel.vue:511`（`gitDialogs.commitAndPush`） |
| RP-C-12 | 推送 | 菜单项（上传云图标） | 显示 | — | `<ref>/components/context-panel/ContextPanel.vue:519`（`gitDialogs.push`） |
| RP-C-13 | （菜单分隔线） | 分隔线 | 显示 | 菜单展开 | `<ref>/components/context-panel/ContextPanel.vue:502` |
| RP-C-14 | （刷新按钮，循环箭头图标，无文案） | 图标按钮 | 显示 | — ；加载中图标旋转 | `<ref>/components/context-panel/ContextPanel.vue:528` |
| RP-C-15 | 没有写入权限，部分操作不可用 | 警告条（琥珀底 + 三角图标） | 隐藏 | 出现权限错误（EPERM/EACCES） | `<ref>/components/context-panel/ContextPanel.vue:554`（`contextPanel.permissionDenied`） |
| RP-C-16 | 重试 | 警告条尾部文字按钮 | 隐藏 | 同上 | `<ref>/components/context-panel/ContextPanel.vue:560`（`common.retry`） |
| RP-C-17 | （改动列表骨架条 ×3） | 骨架 | 显示 | 加载中 | `<ref>/components/context-panel/DiffView.vue:491` |
| RP-C-18 | 没有未提交的改动 | 空态标题（上方圆形对勾图标） | 隐藏 | 范围=未提交改动且无改动 | `<ref>/components/context-panel/DiffView.vue:505`（`contextPanel.noUncommittedChanges`） |
| RP-C-19 | 工作区是干净的 | 空态说明文本 | 隐藏 | 同上 | `<ref>/components/context-panel/DiffView.vue:506`（`contextPanel.cleanWorkspace`） |
| RP-C-20 | 没有 Last Run 改动 | 空态标题（覆盖默认） | 隐藏 | 范围=lastRun 且 runtime 可报告、无触达文件 | `<ref>/components/chat-workspace/ChatWorkspace.vue:3948`（`noLastRunChanges`） |
| RP-C-21 | 最近一次 agent run 没有记录改动文件。 | 空态说明（覆盖默认） | 隐藏 | 同上 | `<ref>/components/chat-workspace/ChatWorkspace.vue:3960`（`noLastRunChangesDetail`） |
| RP-C-22 | Last Run 改动不可用 | 空态标题（覆盖默认） | 隐藏 | 范围=lastRun 且 runtime 无法报告 | `<ref>/components/chat-workspace/ChatWorkspace.vue:3949`（`lastRunUnavailable`） |
| RP-C-23 | 当前 agent runtime 无法报告最近一次 run 的改动文件。 | 空态说明（覆盖默认） | 隐藏 | 同上 | `<ref>/components/chat-workspace/ChatWorkspace.vue:3962`（`lastRunUnavailableDetail`） |
| RP-C-24 | 当前分支没有改动 | 空态标题（覆盖默认） | 隐藏 | 范围=分支且无差异 | `<ref>/components/chat-workspace/ChatWorkspace.vue:3954`（`noBranchChanges`） |
| RP-C-25 | 当前分支与其基线一致。 | 空态说明（覆盖默认） | 隐藏 | 同上 | `<ref>/components/chat-workspace/ChatWorkspace.vue:3966`（`noBranchChangesDetail`） |
| RP-C-26 | Git 改动不可用 | 空态标题（覆盖默认） | 隐藏 | 当前范围读取 Git 改动失败 | `<ref>/components/chat-workspace/ChatWorkspace.vue:3952`（`gitChangesUnavailable`） |
| RP-C-27 | 无法加载 Git 改动，请刷新后重试。 | 空态说明（覆盖默认） | 隐藏 | 同上 | `<ref>/components/chat-workspace/ChatWorkspace.vue:3964`（`gitChangesUnavailableDetail`） |
| RP-C-28 | 已暂存（`{count}`） | 分组标题行（折叠箭头 + 文本，整行可点） | 显示 | 已暂存文件数 > 0 | `<ref>/components/context-panel/DiffView.vue:517`（`contextPanel.stagedFiles`） |
| RP-C-29 | 全部取消暂存 | 分组标题行尾小按钮（－ 图标；执行中转旋转） | 显示 | 分组展开 | `<ref>/components/context-panel/DiffView.vue:530`（`contextPanel.unstageAll`） |
| RP-C-30 | 未暂存（`{count}`） | 分组标题行（折叠箭头 + 文本） | 显示 | 未暂存文件数 > 0 | `<ref>/components/context-panel/DiffView.vue:658`（`contextPanel.unstagedFiles`） |
| RP-C-31 | 全部暂存 | 分组标题行尾小按钮（勾图标；执行中转旋转） | 显示 | 分组展开 | `<ref>/components/context-panel/DiffView.vue:671`（`contextPanel.stageAll`） |
| RP-C-32 | （改动文件行：文件图标 + 文件名 + 新增/删除统计） | 行（`title` 为完整路径） | 显示 | 每个改动文件一行 | `<ref>/components/context-panel/DiffView.vue:556`、`:697` |
| RP-C-33 | +`{added}` | 统计文本（绿色） | 显示 | 每个改动文件行 | `<ref>/components/context-panel/DiffView.vue:572`、`:713` |
| RP-C-34 | -`{removed}` | 统计文本（红色） | 显示 | 每个改动文件行 | `<ref>/components/context-panel/DiffView.vue:573`、`:714` |
| RP-C-35 | 取消暂存 | 行内图标按钮（－，`aria-label`/`title`=取消暂存） | 显示 | 该行属已暂存分组 | `<ref>/components/context-panel/DiffView.vue:578`（`contextPanel.unstage`） |
| RP-C-36 | 暂存 | 行内图标按钮（＋，`aria-label`/`title`=暂存） | 显示 | 该行属未暂存分组 | `<ref>/components/context-panel/DiffView.vue:719`（`contextPanel.stage`） |
| RP-C-37 | 还原 | 行内图标按钮（回转箭头，`aria-label`/`title`=还原） | 显示 | 该行属未暂存分组 | `<ref>/components/context-panel/DiffView.vue:730`（`contextPanel.revert`） |
| RP-C-38 | （行内联 diff 展开区） | 代码块（unified diff，隐藏头部/复制/展开按钮，行内嵌） | 隐藏 | 行已展开且 diff 内容已加载 | `<ref>/components/context-panel/DiffView.vue:618`、`:769` |
| RP-C-39 | 已截断 | 行内提示文本 | 隐藏 | 该 diff 被截断 | `<ref>/components/context-panel/DiffView.vue:625`、`:776`（`contextPanel.previewTruncated`） |
| RP-C-40 | 取消暂存 | 已暂存行右键菜单项（－ 图标） | 显示 | 该行属已暂存分组 | `<ref>/components/context-panel/DiffView.vue:592` |
| RP-C-41 | 还原 | 已暂存行右键菜单项（回转箭头） | 显示 | 同上 | `<ref>/components/context-panel/DiffView.vue:597` |
| RP-C-42 | 预览 | 行右键菜单项（眼睛图标） | 显示 | 任一分组 | `<ref>/components/context-panel/DiffView.vue:602`、`:754`（`common.preview`） |
| RP-C-43 | 用外部工具打开 | 行右键菜单项（外链图标） | 显示 | 任一分组 | `<ref>/components/context-panel/DiffView.vue:606`、`:758`（`contextPanel.openExternal`） |
| RP-C-44 | 复制路径 | 行右键菜单项（复制图标） | 显示 | 任一分组 | `<ref>/components/context-panel/DiffView.vue:611`、`:763`（`contextPanel.copyPath`） |
| RP-C-45 | 暂存 | 未暂存行右键菜单项（＋ 图标） | 显示 | 该行属未暂存分组 | `<ref>/components/context-panel/DiffView.vue:744` |
| RP-C-46 | 还原 | 未暂存行右键菜单项（回转箭头） | 显示 | 同上 | `<ref>/components/context-panel/DiffView.vue:749` |
| RP-C-47 | （行右键菜单分隔线 ×3 ×2 组） | 分隔线 | 显示 | 随相邻菜单项条件 | `<ref>/components/context-panel/DiffView.vue:596`、`:601`、`:610`、`:748`、`:753`、`:762` |
| RP-C-48 | 确认还原未暂存改动？ | 弹窗标题 | 隐藏 | 点击未暂存行「还原」 | `<ref>/components/context-panel/DiffView.vue:804`（`contextPanel.confirmRevertTitle`） |
| RP-C-49 | 这会永久丢弃“{path}”中的未暂存改动。 | 弹窗说明文本 | 隐藏 | 同上 | `<ref>/components/context-panel/DiffView.vue:808`（`contextPanel.confirmRevertDescription`） |
| RP-C-50 | 取消 | 弹窗次按钮 | 隐藏 | 同上 | `<ref>/components/context-panel/DiffView.vue:813`（`common.cancel`） |
| RP-C-51 | 还原 | 弹窗主按钮（危险色） | 隐藏 | 同上 | `<ref>/components/context-panel/DiffView.vue:816`（`contextPanel.confirmRevertAction`） |
| RP-C-52 | `{count}` 个文件改动 | 底部统计左侧文本 | 显示 | 当前标签为 diff 且改动数 > 0 | `<ref>/components/context-panel/ContextPanel.vue:664`（`contextPanel.fileChangesCount`） |
| RP-C-53 | +`{totalAdded}` | 底部统计文本（绿色） | 显示 | 同上 | `<ref>/components/context-panel/ContextPanel.vue:668` |
| RP-C-54 | -`{totalRemoved}` | 底部统计文本（红色） | 显示 | 同上 | `<ref>/components/context-panel/ContextPanel.vue:669` |

元素数：54

## D 文件预览标签页（FilePreviewView）

定位：`<ref>/components/context-panel/FilePreviewView.vue`；Office/PDF 分支转 `<ref>/components/context-panel/OfficeFilePreview.vue`。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| RP-D-01 | 返回文件 | 图标按钮（左箭头，`title`=返回文件） | 显示 | 预览标签页处于激活态 | `<ref>/components/context-panel/FilePreviewView.vue:843`（`contextPanel.backToFiles`） |
| RP-D-02 | （文件完整路径，悬停提示为路径 + 换行 + 在文件管理器中显示） | 文本按钮（`title` 带头部工具条） | 显示 | 有预览路径 | `<ref>/components/context-panel/FilePreviewView.vue:852`（`contextPanel.revealInFolder`） |
| RP-D-03 | 缩小 | 图标按钮（`title`=缩小） | 显示 | 预览内容为图片 | `<ref>/components/context-panel/FilePreviewView.vue:867`（`common.zoomOut`） |
| RP-D-04 | 重置缩放 | 图标按钮（`title`=重置缩放） | 显示 | 同上 | `<ref>/components/context-panel/FilePreviewView.vue:878`（`common.resetZoom`） |
| RP-D-05 | `{zoom}%` | 缩放百分比文本 | 显示 | 同上 | `<ref>/components/context-panel/FilePreviewView.vue:888` |
| RP-D-06 | 放大 | 图标按钮（`title`=放大） | 显示 | 同上 | `<ref>/components/context-panel/FilePreviewView.vue:891`（`common.zoomIn`） |
| RP-D-07 | 用外部工具打开 | 图标按钮（外链图标，`title`=用外部工具打开） | 显示 | 有预览路径；无路径时禁用 | `<ref>/components/context-panel/FilePreviewView.vue:903`（`contextPanel.openExternal`） |
| RP-D-08 | （预览骨架：1 条标题条 + 8 条行） | 骨架 | 显示 | 预览加载中 | `<ref>/components/context-panel/FilePreviewView.vue:916` |
| RP-D-09 | 无法预览文件 | 错误态标题（上方警告图标） | 隐藏 | 预览请求失败 | `<ref>/components/context-panel/FilePreviewView.vue:922`（`contextPanel.previewUnavailable`） |
| RP-D-10 | （错误原文） | 错误态说明文本 | 隐藏 | 同上 | `<ref>/components/context-panel/FilePreviewView.vue:926` |
| RP-D-11 | （图片视口） | 内容区（可滚动、可手势缩放） | 隐藏 | 预览内容为图片 | `<ref>/components/context-panel/FilePreviewView.vue:931` |
| RP-D-12 | 无法预览文件 | 图片加载失败态 | 隐藏 | 图片加载出错 | `<ref>/components/context-panel/FilePreviewView.vue:950` |
| RP-D-13 | （视频播放器，含原生控件） | 内容区 | 隐藏 | 预览内容为视频且有 cwd | `<ref>/components/context-panel/FilePreviewView.vue:967` |
| RP-D-14 | 无法预览文件 | 视频加载失败态 | 隐藏 | 视频加载出错 | `<ref>/components/context-panel/FilePreviewView.vue:973` |
| RP-D-15 | （Office/PDF 预览面） | 内容区（转 OfficeFilePreview） | 隐藏 | 预览类型为 pdf 或 office、有格式且有 cwd | `<ref>/components/context-panel/FilePreviewView.vue:990` |
| RP-D-16 | 二进制文件 | 内容区标题（上方警告图标） | 隐藏 | 预览类型为 binary | `<ref>/components/context-panel/FilePreviewView.vue:1001`（`contextPanel.previewBinaryTitle`） |
| RP-D-17 | 此文件无法在 DimAgent 内预览，可使用外部工具打开。 | 内容区说明文本 | 隐藏 | 同上 | `<ref>/components/context-panel/FilePreviewView.vue:1002`（`contextPanel.previewBinaryDescription`） |
| RP-D-18 | （Markdown 渲染区） | 内容区 | 隐藏 | 预览类型为 markdown 且有内容 | `<ref>/components/context-panel/FilePreviewView.vue:1007` |
| RP-D-19 | （Mermaid 图卡片：含标题栏、模式切换、复制、导出、全屏、折叠、缩放控件） | 内容区 | 隐藏 | 预览内容语言为 mermaid | `<ref>/components/context-panel/FilePreviewView.vue:1032`、`:51` |
| RP-D-20 | （代码块：行号、语法高亮、滚动） | 内容区 | 隐藏 | 预览类型为 code/text 且非 mermaid | `<ref>/components/context-panel/FilePreviewView.vue:1046` |
| RP-D-21 | 无法预览文件 | 兜底态标题 | 隐藏 | 以上分支均不命中 | `<ref>/components/context-panel/FilePreviewView.vue:1068` |
| RP-D-22 | （选中气泡容器） | 浮层 | 隐藏 | 预览内有文本选中 | `<ref>/components/context-panel/FilePreviewView.vue:1075` → `<ref>/components/selection/SelectionPopover.vue:91` |

元素数：22

## E 文档预览工具条（OfficeToolbar）

定位：`<ref>/components/context-panel/OfficeToolbar.vue`；挂载与状态在 `<ref>/components/context-panel/OfficeFilePreview.vue`。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| RP-E-01 | （文档骨架：1 条标题条 + 8 条行） | 骨架 | 显示 | 文档字节加载中（DOCX 含模型解析中） | `<ref>/components/context-panel/OfficeFilePreview.vue:422` |
| RP-E-02 | 无法预览文件 | 错误态标题（上方警告图标） | 隐藏 | 字节加载或解析失败 | `<ref>/components/context-panel/OfficeFilePreview.vue:428` |
| RP-E-03 | （错误原文） | 错误态说明文本 | 隐藏 | 同上 | `<ref>/components/context-panel/OfficeFilePreview.vue:432` |
| RP-E-04 | 重试 | 错误态按钮 | 隐藏 | 同上 | `<ref>/components/context-panel/OfficeFilePreview.vue:433`（`common.retry`） |
| RP-E-05 | 搜索 | 工具条按钮（放大镜图标 + 文本） | 显示 | 工具条搜索未展开 | `<ref>/components/context-panel/OfficeToolbar.vue:162`（`sidebar.search`） |
| RP-E-06 | 搜索 | 搜索输入框（前置放大镜图标，占位与 `aria-label` 均为搜索） | 隐藏 | 工具条搜索已展开 | `<ref>/components/context-panel/OfficeToolbar.vue:100`（`sidebar.search`） |
| RP-E-07 | 正在搜索… | 搜索状态文本 | 隐藏 | 搜索已展开且状态为 searching | `<ref>/components/context-panel/OfficeToolbar.vue:54`（`contextPanel.officeSearchSearching`） |
| RP-E-08 | 搜索失败 | 搜索状态文本 | 隐藏 | 搜索已展开且状态为 error（错误消息在悬停提示） | `<ref>/components/context-panel/OfficeToolbar.vue:55`（`contextPanel.officeSearchFailed`） |
| RP-E-09 | 0 个结果 | 搜索状态文本 | 隐藏 | 已输入关键词且命中数为 0 | `<ref>/components/context-panel/OfficeToolbar.vue:57`（`chatHeader.searchNoResults`） |
| RP-E-10 | `{current}` / `{total}` 个结果 | 搜索状态文本 | 隐藏 | 有命中 | `<ref>/components/context-panel/OfficeToolbar.vue:58`（`chatHeader.searchResults`） |
| RP-E-11 | 上一个搜索结果 | 图标按钮（`title`/`aria-label`=上一个搜索结果） | 隐藏 | 搜索已展开；搜索中或无命中时禁用 | `<ref>/components/context-panel/OfficeToolbar.vue:124`（`chatHeader.previousSearchResult`） |
| RP-E-12 | 下一个搜索结果 | 图标按钮（`title`/`aria-label`=下一个搜索结果） | 隐藏 | 同上 | `<ref>/components/context-panel/OfficeToolbar.vue:136`（`chatHeader.nextSearchResult`） |
| RP-E-13 | 关闭搜索 | 图标按钮（`title`/`aria-label`=关闭搜索） | 隐藏 | 搜索已展开 | `<ref>/components/context-panel/OfficeToolbar.vue:148`（`chatHeader.closeSearch`） |
| RP-E-14 | `{current}` / `{total}` | 页码文本 | 显示 | 工具条搜索未展开，格式非 xlsx，且已知当前页与总页数 | `<ref>/components/context-panel/OfficeToolbar.vue:176` |
| RP-E-15 | 缩小 | 图标按钮（`title`=缩小） | 显示 | 工具条搜索未展开且启用缩放 | `<ref>/components/context-panel/OfficeToolbar.vue:185`（`common.zoomOut`） |
| RP-E-16 | `{zoom}%` | 缩放百分比文本 | 显示 | 同上 | `<ref>/components/context-panel/OfficeToolbar.vue:194` |
| RP-E-17 | 放大 | 图标按钮（`title`=放大） | 显示 | 同上 | `<ref>/components/context-panel/OfficeToolbar.vue:197`（`common.zoomIn`） |
| RP-E-18 | 重置缩放 | 图标按钮（`title`=重置缩放） | 显示 | 同上 | `<ref>/components/context-panel/OfficeToolbar.vue:206`（`common.resetZoom`） |

元素数：18

右栏合计：37 + 36 + 54 + 22 + 18 = 167

---

# 屏：时间线内「本轮改动」与「本轮产物」

## F 本轮改动块（TimelineFileChanges / TimelineFileChangesView）

定位：`<ref>/components/chat-workspace/timeline/TimelineFileChanges.vue`、`TimelineFileChangesView.vue`、`<ref>/components/chat-workspace/git/UndoChangesDialog.vue`。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TL-F-01 | `{count}` 个文件改动 | 折叠头按钮（箭头 + 文本，整行可点） | 显示 | 最新一轮有改动文件且摘要数 > 0 | `<ref>/components/chat-workspace/timeline/TimelineFileChangesView.vue:54`（`changeSummary.fileChangesCount`） |
| TL-F-02 | `{count}` 个文件改动（已撤销） | 折叠头按钮文案（灰化） | 显示 | 本轮改动已撤销 | `<ref>/components/chat-workspace/timeline/TimelineFileChangesView.vue:14`（`fileChanges.filesChangedUndone`） |
| TL-F-03 | 撤销 | 链接按钮（回转箭头；读取预览中转旋转） | 显示 | 非运行中、未撤销、非 App Server 只读、预览可用 | `<ref>/components/chat-workspace/timeline/TimelineFileChangesView.vue:65`（`changeSummary.undo`） |
| TL-F-04 | 重新应用 | 链接按钮（重做箭头；执行中转旋转） | 显示 | 已撤销且有可恢复路径、非运行中、非只读 | `<ref>/components/chat-workspace/timeline/TimelineFileChangesView.vue:79`（`fileChanges.reapply`） |
| TL-F-05 | （文件列表容器，展开态） | 容器（圆角描边） | 隐藏 | 点击折叠头展开 | `<ref>/components/chat-workspace/timeline/TimelineFileChangesView.vue:95` |
| TL-F-06 | （改动文件行：状态字母 + 路径 + 统计） | 行 | 显示 | 展开后每个改动文件一行 | `<ref>/components/chat-workspace/timeline/TimelineFileChangesView.vue:103` |
| TL-F-07 | M / A / D / ? | 状态字母（等宽，按改动类型着色） | 显示 | 行有类型信息；无类型时为 `?` | `<ref>/components/chat-workspace/timeline/TimelineFileChangesView.vue:111`、`:20` |
| TL-F-08 | （文件路径，`title` 为完整路径） | 行内文本（超长截断） | 显示 | 每个改动文件行 | `<ref>/components/chat-workspace/timeline/TimelineFileChangesView.vue:116` |
| TL-F-09 | +`{insertions}`/-`{deletions}` | 行内统计（等宽） | 显示 | 行有增删行数 | `<ref>/components/chat-workspace/timeline/TimelineFileChangesView.vue:119`、`:32` |
| TL-F-10 | binary | 行内统计文本（等宽） | 隐藏 | 行为二进制文件 | `<ref>/components/chat-workspace/timeline/TimelineFileChangesView.vue:37`（`fileChanges.binary`） |
| TL-F-11 | 用默认应用打开 | 行右键菜单项（外链图标） | 显示 | 本地动作可用 | `<ref>/components/chat-workspace/timeline/TimelineFileChangesView.vue:123`（`linkMenu.openWithDefault`） |
| TL-F-12 | 在文件管理器中显示 | 行右键菜单项（文件夹图标） | 显示 | 本地动作可用 | `<ref>/components/chat-workspace/timeline/TimelineFileChangesView.vue:130`（`linkMenu.revealInFolder`） |
| TL-F-13 | 复制路径 | 行右键菜单项（复制图标） | 显示 | 行有绝对路径 | `<ref>/components/chat-workspace/timeline/TimelineFileChangesView.vue:138`（`contextPanel.copyPath`） |
| TL-F-14 | 复制相对路径 | 行右键菜单项（复制图标） | 显示 | 行属工作区范围 | `<ref>/components/chat-workspace/timeline/TimelineFileChangesView.vue:145`（`fileChanges.copyRelativePath`） |
| TL-F-15 | （右键菜单分隔线） | 分隔线 | 显示 | 本地动作可用 | `<ref>/components/chat-workspace/timeline/TimelineFileChangesView.vue:137` |
| TL-F-16 | 撤销改动 | 弹窗标题 | 隐藏 | 点击「撤销」且预览可用 | `<ref>/components/chat-workspace/git/UndoChangesDialog.vue:103`（`fileChanges.undoDialogTitle`） |
| TL-F-17 | 重新应用改动 | 弹窗标题 | 隐藏 | 点击「重新应用」 | `<ref>/components/chat-workspace/git/UndoChangesDialog.vue:103`（`fileChanges.reapplyDialogTitle`） |
| TL-F-18 | 撤销本轮的文件改动？ | 弹窗问句 | 隐藏 | 撤销模式 | `<ref>/components/chat-workspace/git/UndoChangesDialog.vue:106`（`fileChanges.undoDialogQuestion`） |
| TL-F-19 | 重新应用本轮的文件改动？ | 弹窗问句 | 隐藏 | 重新应用模式 | `<ref>/components/chat-workspace/git/UndoChangesDialog.vue:106`（`fileChanges.reapplyDialogQuestion`） |
| TL-F-20 | （弹窗文件行：状态字母 + 路径 + 统计，只读） | 行 | 隐藏 | 弹窗内每个文件一行 | `<ref>/components/chat-workspace/git/UndoChangesDialog.vue:110` |
| TL-F-21 | （删除） | 弹窗行统计文本 | 隐藏 | 撤销模式下该文件为新增 | `<ref>/components/chat-workspace/git/UndoChangesDialog.vue:73`（`fileChanges.willDelete`） |
| TL-F-22 | binary | 弹窗行统计文本 | 隐藏 | 该文件为二进制 | `<ref>/components/chat-workspace/git/UndoChangesDialog.vue:77`（`fileChanges.binary`） |
| TL-F-23 | +`{count}` 更多 | 弹窗底部提示文本 | 隐藏 | 弹窗省略了文件 | `<ref>/components/chat-workspace/git/UndoChangesDialog.vue:128`（`artifacts.moreCount`） |
| TL-F-24 | 取消 | 弹窗次按钮 | 隐藏 | 弹窗打开且非执行中 | `<ref>/components/chat-workspace/git/UndoChangesDialog.vue:131`（`common.cancel`） |
| TL-F-25 | 撤销改动 | 弹窗主按钮（危险色；执行中转旋转） | 隐藏 | 撤销模式 | `<ref>/components/chat-workspace/git/UndoChangesDialog.vue:134`（`fileChanges.undoConfirm`） |
| TL-F-26 | 重新应用 | 弹窗主按钮（主色；执行中转旋转） | 隐藏 | 重新应用模式 | `<ref>/components/chat-workspace/git/UndoChangesDialog.vue:134`（`fileChanges.reapplyConfirm`） |

元素数：26

## G 本轮产物块（TimelineArtifacts / TimelineArtifactsView）

定位：`<ref>/components/chat-workspace/timeline/TimelineArtifacts.vue`、`TimelineArtifactsView.vue`。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TL-G-01 | `{count}` 个产物 | 计数文本（块头左） | 显示 | 本轮有可归类的产物文件 | `<ref>/components/chat-workspace/timeline/TimelineArtifactsView.vue:30`（`artifacts.count`） |
| TL-G-02 | 提交或推送 | 按钮（上传云图标 + 下拉箭头） | 显示 | 工作区在 Git 分支上 | `<ref>/components/chat-workspace/timeline/TimelineArtifactsView.vue:34`（`gitDialogs.commitOrPush`） |
| TL-G-03 | 提交 | 菜单项（提交图标） | 显示 | 菜单展开 | `<ref>/components/chat-workspace/timeline/TimelineArtifactsView.vue:47`（`gitDialogs.commit`） |
| TL-G-04 | 提交并推送 | 菜单项（上传云图标） | 显示 | 菜单展开 | `<ref>/components/chat-workspace/timeline/TimelineArtifactsView.vue:54`（`gitDialogs.commitAndPush`） |
| TL-G-05 | 推送 | 菜单项（上传云图标） | 显示 | 菜单展开 | `<ref>/components/chat-workspace/timeline/TimelineArtifactsView.vue:61`（`gitDialogs.push`） |
| TL-G-06 | （产物列表容器，圆角描边） | 容器 | 显示 | 产物数 > 0 | `<ref>/components/chat-workspace/timeline/TimelineArtifactsView.vue:73` |
| TL-G-07 | （产物行：文件类型图标 + 文件名） | 行（`title` 为完整路径） | 显示 | 每个产物一行；折叠时最多 3 行 | `<ref>/components/chat-workspace/timeline/TimelineArtifactsView.vue:78` |
| TL-G-08 | （已删除产物行半透明） | 行态 | — | 该产物文件已不存在 | `<ref>/components/chat-workspace/timeline/TimelineArtifactsView.vue:80` |
| TL-G-09 | 已删除 | 行内标签文本 | 隐藏 | 该产物文件已不存在 | `<ref>/components/chat-workspace/timeline/TimelineArtifactsView.vue:96`（`artifacts.deleted`） |
| TL-G-10 | 无法预览文件 | 行内替代文本 | 隐藏 | 产物存在但不可打开 | `<ref>/components/chat-workspace/timeline/TimelineArtifactsView.vue:103`（`contextPanel.previewUnavailable`） |
| TL-G-11 | 打开 | 行内按钮（外链图标 + 文本） | 显示 | 产物存在且可打开 | `<ref>/components/chat-workspace/timeline/TimelineArtifactsView.vue:105`（`artifacts.open`） |
| TL-G-12 | （行内下拉触发器，下箭头图标，无文案） | 图标按钮 | 显示 | 产物存在且可打开，且本地动作或下载可用 | `<ref>/components/chat-workspace/timeline/TimelineArtifactsView.vue:116` |
| TL-G-13 | 下载 | 下拉菜单项（下载图标） | 显示 | 下载可用 | `<ref>/components/chat-workspace/timeline/TimelineArtifactsView.vue:128`（`artifacts.download`） |
| TL-G-14 | 用默认应用打开 | 下拉菜单项（外链图标） | 显示 | 本地动作可用 | `<ref>/components/chat-workspace/timeline/TimelineArtifactsView.vue:131`（`linkMenu.openWithDefault`） |
| TL-G-15 | 在文件管理器中显示 | 下拉菜单项（文件夹图标） | 显示 | 本地动作可用 | `<ref>/components/chat-workspace/timeline/TimelineArtifactsView.vue:138`（`linkMenu.revealInFolder`） |
| TL-G-16 | 全部 `{count}` 个产物 | 列表底部展开按钮（右箭头 + 文本） | 显示 | 产物数 > 3 且未展开 | `<ref>/components/chat-workspace/timeline/TimelineArtifactsView.vue:152`、`:16`（`artifacts.allArtifacts`） |
| TL-G-17 | 收起 | 列表底部收起按钮（下箭头 + 文本） | 隐藏 | 已展开 | `<ref>/components/chat-workspace/timeline/TimelineArtifactsView.vue:152`、`:160`（`artifacts.showLess`） |

元素数：17

时间线内块合计：26 + 17 = 43

---

# 屏：终端抽屉（TerminalDrawer）

定位：`<ref>/components/chat-workspace/TerminalDrawer.vue`。

## H 工具条

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TM-H-01 | （终端标签：标题文本，激活态底色高亮） | 标签页（标题按钮） | 显示 | 每个已打开终端标签一行 | `<ref>/components/chat-workspace/TerminalDrawer.vue:796`、`:813` |
| TM-H-02 | （终端标签标题 = shell 上报标题的目录末段；无标题时用「终端」） | 标签文案 | 显示 | 每个终端标签 | `<ref>/components/chat-workspace/TerminalDrawer.vue:154`、`:382`（`chatHeader.terminal`） |
| TM-H-03 | 关闭进程 | 标签内圆形图标按钮（×，`aria-label`=关闭进程） | 显示 | 每个终端标签 | `<ref>/components/chat-workspace/TerminalDrawer.vue:805`（`terminal.closeProcess`） |
| TM-H-04 | 新建终端 | 图标按钮（＋，`aria-label`=新建终端）+ 悬停提示「新建终端」 | 显示 | 工具条显示；无工作目录时禁用 | `<ref>/components/chat-workspace/TerminalDrawer.vue:823`（`terminal.newSession`） |
| TM-H-05 | 关闭终端 | 图标按钮（×，`aria-label`=关闭终端）+ 悬停提示「关闭终端」 | 显示 | 工具条显示 | `<ref>/components/chat-workspace/TerminalDrawer.vue:845`（`chatHeader.closeTerminal`） |

元素数：5

## I 终端内容与提示

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TM-I-01 | 终端 | 终端画布容器（`role=application`，`aria-label`=终端） | 显示 | 抽屉打开 | `<ref>/components/chat-workspace/TerminalDrawer.vue:782`（`chatHeader.terminal`） |
| TM-I-02 | （终端面板：每标签一个，非激活态隐藏） | 画布层 | 显示 | 每个终端标签一层，仅激活层可见 | `<ref>/components/chat-workspace/TerminalDrawer.vue:867`、`:872` |
| TM-I-03 | ［进程已关闭］ | 提示行（无标签时占满内容区） | 隐藏 | 无终端标签且有提示文案（主动关闭进程） | `<ref>/components/chat-workspace/TerminalDrawer.vue:880`（`terminal.processClosed`=`[进程已关闭]`） |
| TM-I-04 | ［进程已退出，代码 `{code}`］ | 提示行 | 隐藏 | 无终端标签且有提示文案（进程退出） | `<ref>/components/chat-workspace/TerminalDrawer.vue:880`、`:302`（`terminal.processExited`=`[进程已退出，代码 {code}]`） |
| TM-I-05 | ［无法打开终端：需要 cwd］ | 提示行 | 隐藏 | 无终端标签且有提示文案（缺工作目录） | `<ref>/components/chat-workspace/TerminalDrawer.vue:880`、`:525`（`terminal.cwdRequired`=`[无法打开终端：需要 cwd]`） |
| TM-I-06 | [Failed to open terminal: `{message}`] | 提示行（硬编码英文） | 隐藏 | 无终端标签且有提示文案（pty 打开失败） | `<ref>/components/chat-workspace/TerminalDrawer.vue:596` |

元素数：6

## J 右键菜单

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TM-J-01 | 复制选中内容 | 右键菜单项（复制图标） | 显示 | 终端内有选中内容才可用 | `<ref>/components/chat-workspace/TerminalDrawer.vue:890`（`terminal.copySelection`） |
| TM-J-02 | 粘贴 | 右键菜单项（粘贴图标） | 显示 | 剪贴板可读、有活跃终端、bridge 就绪才可用 | `<ref>/components/chat-workspace/TerminalDrawer.vue:898`（`terminal.paste`） |
| TM-J-03 | （菜单分隔线） | 分隔线 | 显示 | 菜单展开 | `<ref>/components/chat-workspace/TerminalDrawer.vue:906` |
| TM-J-04 | 清屏 | 右键菜单项（橡皮图标） | 显示 | — | `<ref>/components/chat-workspace/TerminalDrawer.vue:907`（`terminal.clear`） |
| TM-J-05 | 新建终端 | 右键菜单项（＋ 图标） | 显示 | 无工作目录时禁用 | `<ref>/components/chat-workspace/TerminalDrawer.vue:914`（`terminal.newSession`） |
| TM-J-06 | 关闭进程 | 右键菜单项（方块图标，危险色） | 显示 | 无活跃终端或 bridge 未就绪时禁用 | `<ref>/components/chat-workspace/TerminalDrawer.vue:923`（`terminal.closeProcess`） |

元素数：6

终端抽屉合计：5 + 6 + 6 = 17

---

# 屏：任务面板（后台进程 / 唤醒）

面板由右栏承载：标签页文案「后台进程 `{count}`」「唤醒 `{count}`」（见 RP-A-17、RP-A-18），面板内容为 `SessionTasksPanel` → `SessionTasksPanelView`；详情区按行类型分发到 `ProcessDetail`（进程）或 `WakeupDetail`（唤醒）。

## K 任务列表与空态（SessionTasksPanelView）

定位：`<ref>/components/chat-workspace/SessionTasksPanelView.vue`；状态图标 `<ref>/components/chat-workspace/ProcessStatusIcon.vue`。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TP-K-01 | 后台进程 | 列表容器（`role=list` 的 `aria-label`） | 显示 | 面板类型为后台进程且列表非空 | `<ref>/components/chat-workspace/SessionTasksPanelView.vue:81`（`processPanel.title`） |
| TP-K-02 | 唤醒 | 列表容器（`role=list` 的 `aria-label`） | 显示 | 面板类型为唤醒且列表非空 | `<ref>/components/chat-workspace/SessionTasksPanelView.vue:81`（`sessionWakeup.title`） |
| TP-K-03 | （任务行，整行分组） | 行（`role=listitem`，底部分隔线） | 显示 | 每个任务一行 | `<ref>/components/chat-workspace/SessionTasksPanelView.vue:84` |
| TP-K-04 | （行主按钮：状态图标 + 标题 + 分支徽标 + 时间） | 行按钮（选中态有底色） | 显示 | 每个任务一行 | `<ref>/components/chat-workspace/SessionTasksPanelView.vue:88` |
| TP-K-05 | （进程状态图标：运行中 = 旋转圈） | 图标 | 显示 | 行类型为进程且状态运行中 | `<ref>/components/chat-workspace/ProcessStatusIcon.vue:9` |
| TP-K-06 | （进程状态图标：已完成 = 圈勾，成功色） | 图标 | 显示 | 行类型为进程且状态已完成 | `<ref>/components/chat-workspace/ProcessStatusIcon.vue:10` |
| TP-K-07 | （进程状态图标：失败 = 圈叉，危险色） | 图标 | 显示 | 行类型为进程且状态失败 | `<ref>/components/chat-workspace/ProcessStatusIcon.vue:11` |
| TP-K-08 | （进程状态图标：已停止 = 方块） | 图标 | 显示 | 行类型为进程且状态已停止 | `<ref>/components/chat-workspace/ProcessStatusIcon.vue:12` |
| TP-K-09 | （唤醒行图标：时钟） | 图标 | 显示 | 行类型为唤醒 | `<ref>/components/chat-workspace/SessionTasksPanelView.vue:96` |
| TP-K-10 | （任务标题，`title` 为完整标题） | 行内文本（超长截断） | 显示 | 每个任务行 | `<ref>/components/chat-workspace/SessionTasksPanelView.vue:97` |
| TP-K-11 | （分支标识，等宽小字） | 徽标 | 隐藏 | 该任务所属分支不是 `main` | `<ref>/components/chat-workspace/SessionTasksPanelView.vue:98` |
| TP-K-12 | （任务时间，等宽数字） | 行内文本 | 显示 | 每个任务行 | `<ref>/components/chat-workspace/SessionTasksPanelView.vue:104` |
| TP-K-13 | （行尾旋转圈，无文案） | 图标（动画） | 隐藏 | 行类型为唤醒且该行无停止按钮 | `<ref>/components/chat-workspace/SessionTasksPanelView.vue:107` |
| TP-K-14 | （详情区容器，与列表之间有分隔线） | 容器（可滚动） | 显示 | 列表中有选中行 | `<ref>/components/chat-workspace/SessionTasksPanelView.vue:110` |
| TP-K-15 | 暂无运行进程 | 空态文本 | 隐藏 | 面板类型为后台进程且列表为空 | `<ref>/components/chat-workspace/SessionTasksPanelView.vue:113`（`processPanel.empty`） |
| TP-K-16 | 暂无待唤醒事项 | 空态文本 | 隐藏 | 面板类型为唤醒且列表为空 | `<ref>/components/chat-workspace/SessionTasksPanelView.vue:113`（`sessionWakeup.empty`） |

元素数：16

## L 停止按钮（TaskStopButton）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TP-L-01 | 停止 | 图标按钮（方块图标，`aria-label`/`title`=停止） | 显示 | 行类型为进程且该行可停止 | `<ref>/components/chat-workspace/TaskStopButton.vue:50`（`subagentPanel.stopTask`） |
| TP-L-02 | 确认停止 | 图标按钮（对勾图标，再次点击态，危险色） | 隐藏 | 进程停止按钮已被点击过一次（待确认，超时后自动复位） | `<ref>/components/chat-workspace/TaskStopButton.vue:58`（`subagentPanel.confirmStop`） |
| TP-L-03 | 取消唤醒 | 图标按钮（垃圾桶图标，`aria-label`/`title`=取消唤醒） | 显示 | 行类型为唤醒且该行可取消 | `<ref>/components/chat-workspace/TaskStopButton.vue:59`（`sessionWakeup.cancel`） |
| TP-L-04 | 再次点击取消唤醒 | 图标按钮（对勾图标，再次点击态，危险色） | 隐藏 | 取消唤醒按钮已被点击过一次 | `<ref>/components/chat-workspace/TaskStopButton.vue:59`（`sessionWakeup.confirmCancel`） |
| TP-L-05 | （执行中旋转圈，无文案） | 图标（动画，禁用态） | 隐藏 | 该行停止/取消请求进行中 | `<ref>/components/chat-workspace/TaskStopButton.vue:57` |

元素数：5

## M 后台进程详情（ProcessDetail / ProcessDetailView）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TP-M-01 | 运行中 · `{time}` | 详情头状态文本（前置状态图标） | 显示 | 选中进程状态为运行中 | `<ref>/components/chat-workspace/ProcessDetailView.vue:30`、`:10`（`processPanel.status.running`） |
| TP-M-02 | 已完成 · `{time}` | 详情头状态文本 | 显示 | 选中进程状态为已完成 | `<ref>/components/chat-workspace/ProcessDetailView.vue:30`（`processPanel.status.completed`） |
| TP-M-03 | 失败 · `{time}` | 详情头状态文本 | 显示 | 选中进程状态为失败 | `<ref>/components/chat-workspace/ProcessDetailView.vue:30`（`processPanel.status.failed`） |
| TP-M-04 | 已停止 · `{time}` | 详情头状态文本 | 显示 | 选中进程状态为已停止 | `<ref>/components/chat-workspace/ProcessDetailView.vue:30`（`processPanel.status.cancelled`） |
| TP-M-05 | 复制命令 | 图标按钮（复制图标，`aria-label`/`title`=复制命令） | 显示 | 该进程有命令 | `<ref>/components/chat-workspace/ProcessDetailView.vue:33`（`processPanel.copyCommand`） |
| TP-M-06 | 已复制 | 按钮图文 / 读屏播报文本 | 隐藏 | 命令已复制（2 秒内） | `<ref>/components/chat-workspace/ProcessDetailView.vue:40`、`:42`（`common.copied`） |
| TP-M-07 | 打开日志 | 图标按钮（外链图标，`aria-label`/`title`=打开日志） | 显示 | 日志可用 | `<ref>/components/chat-workspace/ProcessDetailView.vue:43`（`processPanel.openLog`） |
| TP-M-08 | `{toolName}` · `{startedAt}` · pid `{pid}` · 结束于 `{time}` | 元信息行（各段按有无拼接） | 显示 | 任一段有值 | `<ref>/components/chat-workspace/ProcessDetailView.vue:53`（`processPanel.endedAt`） |
| TP-M-09 | $ `{command}` | 命令行文本（等宽、可换行） | 显示 | 该进程有命令 | `<ref>/components/chat-workspace/ProcessDetailView.vue:57` |
| TP-M-10 | （后台任务输出终端，只读） | 画布（等宽终端，可滚动） | 显示 | 详情激活、文档可见且终端未失败 | `<ref>/components/chat-workspace/ProcessDetail.vue:77` → `<ref>/components/chat-workspace/BackgroundTaskTerminal.vue:76` |
| TP-M-11 | 预览已截断 | 终端右下覆盖标签（半透明黑底） | 隐藏 | 结构化最终预览超界丢过输出 | `<ref>/components/chat-workspace/BackgroundTaskTerminal.vue:78`（`processPanel.previewTruncated`） |
| TP-M-12 | 终端操作失败 | 失败提示行（危险色，`role=alert`） | 隐藏 | 终端挂载或订阅失败 | `<ref>/components/chat-workspace/ProcessDetail.vue:65`（`terminal.actionFailed`） |
| TP-M-13 | 重新连接 | 失败提示行尾按钮（旋转箭头图标） | 隐藏 | 同上 | `<ref>/components/chat-workspace/ProcessDetail.vue:67`（`processPanel.reconnect`） |

元素数：13

## N 唤醒详情（WakeupDetail）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TP-N-01 | `{time}` 触发 | 详情头文本 | 显示 | 选中行类型为唤醒 | `<ref>/components/chat-workspace/WakeupDetail.vue:11`（`sessionWakeup.fireAt`） |
| TP-N-02 | （唤醒提示内容） | 正文文本（可换行） | 显示 | 同上 | `<ref>/components/chat-workspace/WakeupDetail.vue:14` |

元素数：2

任务面板合计：16 + 5 + 13 + 2 = 36

---

# 屏：侧边对话（SideChatPanel）

右栏标签页文案「侧边对话」（RP-A-16），面板内容为 `SideChatPanel`。时间线、审批与提问卡按各自清单登记，此处只登记面板自有外壳与输入区。

## O 侧聊面板

定位：`<ref>/components/chat-workspace/SideChatPanel.vue`。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| SC-O-01 | （错误条，橙底） | 横幅（文本 + 重试按钮） | 隐藏 | 打开分支或发送失败 | `<ref>/components/chat-workspace/SideChatPanel.vue:565` |
| SC-O-02 | 重试 | 横幅尾部文字按钮 | 隐藏 | 有错误且尚未取到分支 payload 且分支已存在 | `<ref>/components/chat-workspace/SideChatPanel.vue:570`（`common.retry`） |
| SC-O-03 | 此分支已结束，不再接收消息。历史仍可在此查看；完成后可关闭此标签页。 | 状态条（灰底，顶部分隔线） | 隐藏 | 分支状态为 ending 或 ended | `<ref>/components/chat-workspace/SideChatPanel.vue:582`（`sideChat.ended`） |
| SC-O-04 | （侧聊时间线，只读） | 内容区（复用 PlainTimeline） | 显示 | 已取到会话且有消息 | `<ref>/components/chat-workspace/SideChatPanel.vue:590` |
| SC-O-05 | （空态旋转圈，无文案） | 图标（动画） | 隐藏 | 无消息且正在加载 / 发送 / 运行中 | `<ref>/components/chat-workspace/SideChatPanel.vue:607` |
| SC-O-06 | 此侧边对话会从所选消息之前开始。 | 空态文本 | 隐藏 | 无消息、分支尚未创建、且从某条消息处开启 | `<ref>/components/chat-workspace/SideChatPanel.vue:608`（`sideChat.fromHere`） |
| SC-O-07 | 就当前 thread 提问。侧边对话会继承启动时的上下文。 | 空态文本 | 隐藏 | 无消息且其余空态不命中 | `<ref>/components/chat-workspace/SideChatPanel.vue:614`（`sideChat.empty`） |
| SC-O-08 | （权限提示卡位置） | 卡位（复用 InlinePermissionPrompt） | 隐藏 | 分支未结束且有权限待批 | `<ref>/components/chat-workspace/SideChatPanel.vue:618` |
| SC-O-09 | （提问提示卡位置） | 卡位（复用 InlineElicitationPrompt） | 隐藏 | 分支未结束且有提问待答 | `<ref>/components/chat-workspace/SideChatPanel.vue:625` |
| SC-O-10 | 问一个旁支问题… | 输入框（占位） | 显示 | 输入区常驻；分支结束 / 发送中 / 分支未就绪时禁用 | `<ref>/components/chat-workspace/SideChatPanel.vue:636`（`sideChat.placeholder`） |
| SC-O-11 | 结束分支 | 图标按钮（圆圈停止图标，`title`=结束分支，悬停转危险色） | 显示 | 面板激活、分支已建立且非 main、状态为 open、未在结束中 | `<ref>/components/chat-workspace/SideChatPanel.vue:644`（`sideChat.endBranch`） |
| SC-O-12 | 停止 | 图标按钮（方块图标，`title`=停止） | 隐藏 | 分支运行中；停止请求进行中时禁用 | `<ref>/components/chat-workspace/SideChatPanel.vue:655`（`sideChat.stop`） |
| SC-O-13 | 发送 | 图标按钮（纸飞机图标，`title`=发送） | 显示 | 分支未运行 | `<ref>/components/chat-workspace/SideChatPanel.vue:666`（`sideChat.send`） |
| SC-O-14 | 引导 | 图标按钮（纸飞机图标，`title`=引导） | 显示 | 分支运行中（同一按钮，文案切换） | `<ref>/components/chat-workspace/SideChatPanel.vue:666`（`sideChat.steer`） |
| SC-O-15 | 结束此分支？ | 确认弹窗标题 | 隐藏 | 点击「结束分支」 | `<ref>/components/chat-workspace/SideChatPanel.vue:682`（`sideChat.endBranchTitle`） |
| SC-O-16 | 此操作会永久结束该分支并中断当前回合。分支将不再接收消息，但历史仍可在此标签页中查看。 | 确认弹窗说明文本 | 隐藏 | 同上 | `<ref>/components/chat-workspace/SideChatPanel.vue:683`（`sideChat.endBranchDescription`） |
| SC-O-17 | 取消 | 确认弹窗次按钮 | 隐藏 | 同上 | `<ref>/components/chat-workspace/SideChatPanel.vue:686`（`common.cancel`） |
| SC-O-18 | 确认 | 确认弹窗主按钮（危险色；结束请求中禁用） | 隐藏 | 同上 | `<ref>/components/chat-workspace/SideChatPanel.vue:687`（`common.confirm`） |

元素数：18

---

# 屏：子代理对话（SubagentChatPanel）

右栏标签页文案为子代理会话标题（RP-A-20），面板内容为 `SubagentChatPanel`；面板本身无输入区。

## P 子代理面板

定位：`<ref>/components/chat-workspace/SubagentChatPanel.vue`。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| SA-P-01 | （行内错误条，橙底，单行截断） | 横幅 | 隐藏 | 已有内容后再次连接失败 | `<ref>/components/chat-workspace/SubagentChatPanel.vue:127` |
| SA-P-02 | 重试 | 横幅尾部文字按钮（刷新图标 + 文本） | 隐藏 | 同上 | `<ref>/components/chat-workspace/SubagentChatPanel.vue:133`（`common.retry`） |
| SA-P-03 | （首屏错误文本，警告色） | 空态错误文本 | 隐藏 | 尚未取到内容即失败 | `<ref>/components/chat-workspace/SubagentChatPanel.vue:141` |
| SA-P-04 | 重试 | 首屏错误态按钮（刷新图标 + 文本） | 隐藏 | 同上 | `<ref>/components/chat-workspace/SubagentChatPanel.vue:142`（`common.retry`） |
| SA-P-05 | 正在加载... | 加载态文本（前置旋转圈） | 显示 | 尚未就绪且无错误 | `<ref>/components/chat-workspace/SubagentChatPanel.vue:148`（`common.loading`） |
| SA-P-06 | （子代理时间线，只读） | 内容区（复用 PlainTimeline） | 显示 | 已就绪 | `<ref>/components/chat-workspace/SubagentChatPanel.vue:154` |

元素数：6

---

## 2 合计

| 屏 | 区域 | 分组 | 元素数 |
|---|---|---|---|
| 右栏 | A 面板外壳与标签条 | 1 | 37 |
| 右栏 | B 文件标签页 | 1 | 36 |
| 右栏 | C 改动（Review）标签页 | 1 | 54 |
| 右栏 | D 文件预览标签页 | 1 | 22 |
| 右栏 | E 文档预览工具条 | 1 | 18 |
| 时间线内块 | F 本轮改动块 | 1 | 26 |
| 时间线内块 | G 本轮产物块 | 1 | 17 |
| 终端抽屉 | H 工具条 | 1 | 5 |
| 终端抽屉 | I 终端内容与提示 | 1 | 6 |
| 终端抽屉 | J 右键菜单 | 1 | 6 |
| 任务面板 | K 任务列表与空态 | 1 | 16 |
| 任务面板 | L 停止按钮 | 1 | 5 |
| 任务面板 | M 后台进程详情 | 1 | 13 |
| 任务面板 | N 唤醒详情 | 1 | 2 |
| 侧边对话 | O 侧聊面板 | 1 | 18 |
| 子代理对话 | P 子代理面板 | 1 | 6 |
| **合计** | 16 区域 | 16 | **287** |

## 3 未纳入本清单的相邻元素（避免重复登记）

| 元素 | 归属 | 说明 |
|---|---|---|
| 选中文本气泡内的「加入对话 / 标注问题 / 提交 / 评论框」 | 全局面板清单 G-I3 | 预览区与文档预览复用同一组件，本清单只在 RP-D-22 登记容器 |
| 右栏开关（开/关右侧面板）的标题栏按钮 | 主窗口清单（标题栏区域） | 由标题栏承担，不在右栏内 |
| 侧边栏展开按钮的悬停提示文案「展开侧边栏 ⌘B」 | 主窗口清单（左栏栏头） | 本清单只登记右栏 replacement 布局内的同一个按钮（RP-A-04） |
| 会话侧聊、子代理、任务面板、过程详情 | 已登记 | 见区域 K–P |
| 时间线本体（消息、工具卡、思考、Markdown、滚动/搜索） | 主窗口清单（时间线区域） | 侧聊与子代理面板内复用 `PlainTimeline`，这里只登记容器（SC-O-04、SA-P-06） |
| 审批卡与提问卡（`InlinePermissionPrompt`、`InlineElicitationPrompt` 及其展开后的审批/提问界面） | 全局面板清单（C 审批、D 提问） | 侧聊只在输入区上方留卡位（SC-O-08、SC-O-09） |
| 上下文面板内置标签条（`<ref>/components/context-panel/TabBar.vue`：标签行、左右滚动按钮、溢出菜单） | 不登记 | 右栏布局下 `show-tab-bar=false`，该标签条模板不渲染；右栏的标签页由 RightPanelBar 承担（见区域 A）。若日后恢复独立面板布局需要单独成组 |
