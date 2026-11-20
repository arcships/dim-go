# 全局面板八类的行为需求

Status: draft v2（八类全量：276 元素 / 2,348 条；§1 审批 587、§2 搜索 382、§3 命令 235、§4 提问 182、§5 确认 174、§6 重命名 132、§7 导出与 Fork 183、§8 导入与更新 473；§9 为计数与门禁自查）
Input: [`inventory-global-rt.md`](inventory-global-rt.md)（八类 276 元素：可见 40、带条件 236）＋ `runtime/main/*.json` 与同名 `.png`（1030×725 CSS 视口，DPR=2）
Owner: 需求分析与行为需求（全局块）
Last updated: 2026-11-19
Baseline: 验收列按 [`design-baseline.md`](design-baseline.md) §11「本产品定值」回填（`判据 = <常量名> <值>`）；四份需求里没有待定判据。

## 0. 口径

### 0.1 一条需求一行

| 字段 | 写法 |
|---|---|
| 需求 ID | `R-<类缩写>-<序号>`：审批 `AP`、搜索 `GS`、命令 `GM`、提问 `GQ`、确认 `GC`、重命名 `GR`、导出与 Fork `GF`、导入与更新 `GI`；类内三位起连续，不复用 |
| 元素 ID | 只取 [`inventory-global-rt.md`](inventory-global-rt.md) 各表第一列（`G-C1-01` 形式）；一条需求挂一个元素 |
| 维度 | 状态 / 内容 / 交互 / 动画 / 样式 / 关联 / 异常 / 记忆，再加参考产品里实际出现的其它表现（键盘、窄窗、高 DPI、暗色、无障碍、性能、多语言）；**示例，包括但不限于** |
| 场景 | 用户此刻在做什么、处在哪个采集态；未出现过的元素行首标「未观察（条件）」，其后写触发条件（数据 / 交互） |
| 期望 | 屏上可观察的结果：文案逐字、包围盒、数量、顺序、形变、去向；坐标一律 CSS px（png 像素 ÷ 2） |
| 异常 | 失败、超时、无权限、断线、资源缺失、并发时的表现；不适用写「—」 |
| 验收 | 三形态之一：①具体值；②`判据 = <基线常量名> <值>`；③「与期望列一致」 |

### 0.2 证据

| 代号 | 文件 | 屏上内容 | 本版用途 |
|---|---|---|---|
| A | `runtime/main/session-approval.json`（左栏展开，x≈386） | 一次 `exec` 权限请求：标题「DimAgent 想要：执行」＋输入区（`ls -la` + `cwd`）＋三枚按钮 | C1 卡片骨架、C4 命令渲染器 |
| C38 | `runtime/main/session-streaming.json`（左栏收起，x≈30） | 同一次请求的第二证据，x 平移 −356 | C1 坐标随左栏平移 |
| C39 | `runtime/main/session-approval-timeout.json` | 同族请求，参数行换成 `timeoutMs: 60 s` | C4-04 的参数行第二分支、C1 三按钮第二处坐标 |

三份都是 `exec` 的普通权限请求。**没有一份态命中沙箱越权卡（C2）、子代理模型卡（C3）、成员审批区块（C1-04–C1-12）与 C4 的另外九类渲染器**——这些按清单的「需数据 / 需交互」逐条写条件，验收走形态②。

### 0.3 「未观察（条件）」的比例与写法

八类 276 项里，60 份采集态直接验证到的只有 **40 项**（清单 §9.2）：搜索 16（A1 9 + A2 5 + A5 2）、命令 13、审批 11；其余 **236 项（85.5%）** 在采集态里没有出现，逐条在场景列写触发条件（数据 / 交互），验收一律形态②或「与期望列一致」。逐类数字见 §9.2。

**审批域的覆盖率最高**（11/63）：A/C38/C39 三份 `exec` 权限请求验到了 C1 的容器、标题、输入区块位、动作按钮组与三枚按钮，以及 C4 的容器、「输入」标签、命令行、参数行；C2（沙箱越权）、C3（子代理模型）、成员审批区块与 C4 另外九类渲染器全部按条件写。

文案占位：请求里的真实工作目录写作 `{工作区路径}`，真实命令保留形态但换 `{命令}`（A 态 `ls -la` 在这批证据里可核，写作原文）。

---

## 1. 审批（63 元素）

审批卡贴在输入区上沿、不遮时间线；三张卡（工具权限 / 沙箱越权 / 子代理模型）共用一个外层骨架（边框 + 16px 内边距 + 与页面同底色），差异在标题、说明与按钮语义。A 态卡片 ≈370,503 647×173；标题 386,519 615×21；输入区 387,552 613×27；按钮行 y=632，按钮高 28。

### 1.1 工具权限卡（G-C1-01–G-C1-21）

| 需求 ID | 元素 ID | 维度 | 场景 | 期望 | 异常 | 验收 | 状态 |
|---|---|---|---|---|---|---|---|
| R-AP-001 | G-C1-01 | 状态 | A 态：会话收到 `tool_call` 权限请求、左栏展开 | 卡片在场：贴输入区上沿，横跨主区内容宽（≈647），上缘 503、下缘 660；边框 1px + 内边距 16；底色与页面同色 | — | 卡片包围盒 ≈370,503 647×173 | 已实现（映射 = G-C1-01→internal/ui/approval.go:App.approvalCard；标记 = State@App.currentApproval） |
| R-AP-002 | G-C1-02 | 状态 | C38 态：左栏收起 | 同一张卡整体左移 356（x≈14 → 30），宽度随主区变宽（615 → 659）；卡片结构不变 | — | 标题 x=30、宽 659（A 态 386、宽 615，差 356） | 已实现（映射 = G-C1-02→internal/ui/approval.go:App.toolApprovalBody；标记 = State） |
| R-AP-003 | G-C1-01 | 状态 | 请求被答复后 | 卡片整体消失，输入区回到原位；不留空白占位 | — | 屏上 卡片 计数 = 0 | 已实现（映射 = G-C1-01→internal/ui/approval.go:App.approvalCard；标记 = State@App.currentApproval） |
| R-AP-004 | G-C1-01 | 状态 | 同一会话连续两个请求 | 屏上只显示当前一个请求的卡片；前一个随答复消失，不纵向堆叠两张 | — | 卡片计数恒为 1 | 已实现（映射 = G-C1-01→internal/ui/approval.go:App.approvalCard；标记 = State@App.currentApproval） |
| R-AP-005 | G-C1-01 | 内容 | 请求带长命令 | 卡片高度增长（输入区最高 240 后内部滚动），标题与按钮行位置不重叠 | — | 判据 = 控件尺寸 按钮高 行内 21 / 小 25 / 次 28 / 常规 32 / 大 35；其余 = 滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C1-01→internal/ui/approval.go:App.approvalCard；标记 = 文案「审批卡」） |
| R-AP-006 | G-C1-01 | 交互 | 鼠标滚轮在卡片上 | 卡片本身不滚动；只有输入区（C4-01）内部滚动 | — | 判据 = 滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-007 | G-C1-01 | 交互 | 鼠标点击卡片空白处时 | 无动作（不关闭、不答复）；只有按钮触发答复 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-C1-01→internal/ui/approval.go:App.approvalCard |
| R-AP-008 | G-C1-01 | 动画 | 请求到达 | 卡片自输入区上沿展开出现（高度增长），≤150ms；时间线不被推挤跳动 | 减少动效下直接出现 | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；时长 = 期望列给的毫秒值（±0ms），减少动效时 none（§6 动效表） | 未实现（映射到位但没有该维度的标记）：G-C1-01→internal/ui/approval.go:App.approvalCard |
| R-AP-009 | G-C1-01 | 动画 | 答复后 | 卡片收起消失，输入区上移复位；无残留遮罩 | 减少动效下瞬切 | 屏上 卡片 计数 = 0 | 未实现（映射到位但没有该维度的标记）：G-C1-01→internal/ui/approval.go:App.approvalCard |
| R-AP-010 | G-C1-01 | 样式 | 浅色主题 | 卡片边框中性色、底色与页面一致；只用描边区分，不加阴影 | — | 判据 = 色值 边框 明 #e5e5e5 / 暗 #2e2e2e；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px；内边距 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28）/ 卡片外框 4.5；档位 3.5·7·10.5·14·21·28px | 已实现（映射 = G-C1-01→internal/ui/approval.go:App.approvalCard；标记 = Width） |
| R-AP-011 | G-C1-01 | 样式 | 深色主题 | 描边在深底上可见（比底色亮一档），不用浅色主题的线 | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d；其余 = 文字与底色对比度 ≥ 4.5:1（正文）、≥ 3:1（大字与图标）（CL-CONTRAST） | 已实现（映射 = G-C1-01→internal/ui/approval.go:App.approvalCard；标记 = Width） |
| R-AP-012 | G-C1-01 | 关联 | 卡片在场、输入区被卡片取代时 | 卡片存在时输入框不可见或被卡片取代（A 态屏上无输入框），卡片底缘与输入区上沿贴合 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C1-01→internal/ui/approval.go:App.approvalCard |
| R-AP-013 | G-C1-01 | 关联 | 卡片出现、时间线仍在其下方时 | 卡片不遮挡最后一条消息；时间线可视高度收缩但不滚动跳位 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-014 | G-C1-01 | 关联 | 展开或收起左栏时 | 左栏展开/收起时卡片随主区平移与变宽，不固定在窗口坐标 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C1-01→internal/ui/approval.go:App.approvalCard |
| R-AP-015 | G-C1-01 | 无障碍 | 读屏进入该元素时 | 卡片是一个对话框/区域，读出名由标题承担；焦点默认落第一个按钮 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-C1-01→internal/ui/approval.go:App.approvalCard；标记 = Role） |
| R-AP-016 | G-C1-01 | 无障碍 | 读屏遍历该元素时 | 顺序为：标题→输入区内容→按钮组（拒绝、仅这次、始终允许…） | — | 播报顺序 = 标题 → 输入区内容 → 按钮组（拒绝 → 仅这次 → 始终允许） | 已实现（映射 = G-C1-01→internal/ui/approval.go:App.approvalCard；标记 = Role） |
| R-AP-017 | G-C1-01 | 键盘 | 在卡片内按 Tab 时 | 焦点在按钮之间循环，不逃到背景；Esc 不答复（审批不提供快捷键误触） | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 未实现（映射到位但没有该维度的标记）：G-C1-01→internal/ui/approval.go:App.approvalCard |
| R-AP-018 | G-C1-01 | 异常 | 提交答复时连接断开 | 卡片保留并显示提交错误（C1-16）；不假装成功、不消失 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 已实现（映射 = G-C1-01→internal/ui/approval.go:App.approvalCard；标记 = Err@App.sandboxApprovalBody） |
| R-AP-019 | G-C1-01 | 异常 | 答复超时（服务端已判超时） | 卡片转为不可答复态或显示失败原因，按钮禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-C1-01→internal/ui/approval.go:App.approvalCard；标记 = Err@App.sandboxApprovalBody） |
| R-AP-020 | G-C1-01 | 异常 | 请求在渲染前被撤销 | 不出现空卡片（无标题、无按钮的残框） | — | 屏上 空卡片 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-021 | G-C1-01 | 记忆 | 切到另一会话再回来 | 未答复的请求仍在原会话的输入区上方；不跨会话串卡 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C1-01→internal/ui/approval.go:App.approvalCard |
| R-AP-022 | G-C1-01 | 记忆 | 关掉右栏/切标签 | 卡片不受右栏影响，位置与内容不变 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C1-01→internal/ui/approval.go:App.approvalCard |
| R-AP-023 | G-C1-01 | 窄窗 | 窗口收窄到 replacement 布局 | 卡片宽度随主区收缩，标题与按钮仍完整；按钮行可折行但不被裁 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 未实现（映射到位但没有该维度的标记）：G-C1-01→internal/ui/approval.go:App.approvalCard |
| R-AP-024 | G-C1-01 | 高DPI | 在 DPR=2 的屏幕上时 | 1px 描边在 2× 下为 2 物理像素，圆角平滑 | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C1-01→internal/ui/approval.go:App.approvalCard |
| R-AP-025 | G-C1-01 | 多语言 | 系统语言切到英文 | 标题模板「{agent} 想要：{action}」整体按语言包切换，动作标签跟着换 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 已实现（映射 = G-C1-01→internal/ui/approval.go:App.approvalCard；标记 = 文案「审批卡」） |
| R-AP-026 | G-C1-01 | 性能 | 长命令（200 行） | 卡片出现 ≤200ms，输入区滚动流畅 | — | 判据 = 动效 布局 250ms / 颜色 150ms / 折叠 300ms；缓动 cubic-bezier(0.16, 1, 0.3, 1) / cubic-bezier(0.4, 0, 0.2, 1)；其余 = 滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C1-01→internal/ui/approval.go:App.approvalCard |
| R-AP-027 | G-C1-01 | 内容 | 请求带成员审批上下文（成员审批区块在场）时 | 卡片高度含该区块；标题仍在外层顶部，区块在标题与输入区之间 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C1-01→internal/ui/approval.go:App.approvalCard；标记 = 文案「审批卡」） |
| R-AP-028 | G-C1-01 | 内容 | 请求带自动审批提示（提示块在场）时 | 提示块出现在输入区之后、按钮行之前 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM） | 已实现（映射 = G-C1-01→internal/ui/approval.go:App.approvalCard；标记 = 文案「审批卡」） |
| R-AP-029 | G-C1-01 | 状态 | 三张卡互斥 | 同一时刻只出现一张卡（工具权限 / 沙箱越权 / 子代理模型），不并列 | — | 卡片计数恒为 1 | 已实现（映射 = G-C1-01→internal/ui/approval.go:App.approvalCard；标记 = State@App.currentApproval） |
| R-AP-030 | G-C1-01 | 关联 | 同一会话先后到达审批卡与提问卡时 | 审批卡与提问卡不同时占同一位置；若两者都在，按到达顺序排队，一次只显示一张 | — | 屏上 卡片 计数 = 1 | 未实现（映射到位但没有该维度的标记）：G-C1-01→internal/ui/approval.go:App.approvalCard |
| R-AP-031 | G-C1-01 | 关联 | 答复后回到会话时间线时 | 答复后卡片消失，审批记录进入该会话的审批历史（由时间线/审批域承担），不在输入区留痕 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 未实现（映射到位但没有该维度的标记）：G-C1-01→internal/ui/approval.go:App.approvalCard |
| R-AP-032 | G-C1-02 | 内容 | A 态：`exec` 工具的权限请求 | 标题逐字「DimAgent 想要：执行」（386,519 615×21），动作标签是「执行」而不是工具名 `exec` | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalCopySandboxExecute「执行」 |
| R-AP-033 | G-C1-02 | 内容 | C38/C39 态 | 同文案，x≈30、宽 659；文案本身不随左栏改变 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C1-02→internal/ui/approval.go:App.toolApprovalBody；标记 = 文案「正在加载成员操作...」等 4 条@approvalTitle） |
| R-AP-034 | G-C1-02 | 内容 | 工具动作不同 | 动作标签按工具切换（读取文件 / 写入文件 / 运行命令等），不直接显示英文工具名 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-C1-02→internal/ui/approval.go:App.toolApprovalBody；标记 = 文案「正在加载成员操作...」等 4 条@approvalTitle） |
| R-AP-035 | G-C1-02 | 关联 | 请求来自没有专用标签的工具（MCP 等）时 | 只有工具无专用标签（MCP 等）时才退成「使用工具」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalUseToolAction「使用工具」 |
| R-AP-036 | G-C1-02 | 样式 | 分别切到浅色与深色主题时 | 16px 级半粗主前景色；左对齐、单行；深色主题不改变字重 | — | 判据 = 字号 对话框标题 15.75px；字重 半粗 600；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-037 | G-C1-02 | 无障碍 | 读屏进入该元素时 | 作为卡片标题被读出；`{action}` 部分读中文而非英文枚举 | — | 读屏文本 = 「DimAgent 想要：{action}」 | 已实现（映射 = G-C1-02→internal/ui/approval.go:App.toolApprovalBody；标记 = Label@App.memberApprovalBlock） |
| R-AP-038 | G-C1-02 | 内容 | 动作标签超长（长工具名映射） | 标题单行显示，超宽被容器裁剪，不换行把按钮行推下去 | — | 标题恒 1 行 | 已实现（映射 = G-C1-02→internal/ui/approval.go:App.toolApprovalBody；标记 = 文案「正在加载成员操作...」等 4 条@approvalTitle） |
| R-AP-039 | G-C1-02 | 关联 | 普通请求与成员请求交替到达时 | 两者互斥：普通请求显示本标题，成员请求显示成员标题 | — | 屏上 标题 计数 = 1 | 未实现（映射到位但没有该维度的标记）：G-C1-02→internal/ui/approval.go:App.toolApprovalBody |
| R-AP-040 | G-C1-02 | 异常 | 动作映射表缺该工具 | 回退为「使用工具」，不显示空冒号或空标题 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-041 | G-C1-02 | 记忆 | 同一工具再次请求 | 标题文案稳定，不出现「再次」「重试」前缀 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-042 | G-C1-02 | 窄窗 | 窗口收窄到最窄宽度时 | 标题单行裁剪，不折行挤掉按钮 | — | 窄窗下标题 1 行 | 未实现（映射到位但没有该维度的标记）：G-C1-02→internal/ui/approval.go:App.toolApprovalBody |
| R-AP-043 | G-C1-02 | 多语言 | 把界面语言切到英文时 | 「DimAgent wants to: {action}」形态，冒号与空格按语言包 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 已实现（映射 = G-C1-02→internal/ui/approval.go:App.toolApprovalBody；标记 = 文案「正在加载成员操作...」等 4 条@approvalTitle） |
| R-AP-044 | G-C1-02 | 高DPI | 在 DPR=2 的屏幕上时 | 半粗笔画在 2× 下不糊 | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C1-02→internal/ui/approval.go:App.toolApprovalBody |
| R-AP-045 | G-C1-02 | 内容 | 权限请求来自子代理 | 标题仍为「DimAgent 想要：{action}」，不加子代理前缀（子代理信息在输入区） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-C1-02→internal/ui/approval.go:App.toolApprovalBody；标记 = 文案「正在加载成员操作...」等 4 条@approvalTitle） |
| R-AP-046 | G-C1-03 | 状态 | 未观察（条件）：工具无专用动作标签，或工具来自 MCP | 标题回退为「使用工具」，`{action}` 位置不再出现工具名 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（逐字）：internal/ui/approval.go:approvalUseToolAction「使用工具」 |
| R-AP-047 | G-C1-03 | 关联 | 标题位回退到「使用工具」时 | 同一标题位二选一；回退时标题仍占一行、字重不变 | — | 判据 = 字号 页面标题 17.5px；字重 常规 400 / 中粗 500 / 半粗 600 / 粗 700 | 未实现（映射到位但没有该维度的标记）：G-C1-03→internal/ui/approval.go:approvalTitle |
| R-AP-048 | G-C1-03 | 内容 | MCP 工具名很长（`server__tool`） | 不把原始工具名拼进标题（避免把 `__` 暴露到屏上） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 已实现（映射 = G-C1-03→internal/ui/approval.go:approvalTitle；标记 = 文案「正在加载成员操作...」等 4 条） |
| R-AP-049 | G-C1-03 | 样式 | 回退态标题与普通态标题先后出现时 | 同字号同字重同色 | — | 判据 = 字号 页面标题 17.5px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 未实现（映射到位但没有该维度的标记）：G-C1-03→internal/ui/approval.go:approvalTitle |
| R-AP-050 | G-C1-03 | 无障碍 | 读屏进入该元素时 | 读作「使用工具」，并在输入区读工具名（若可） | — | 读屏文本 = 「使用工具」 | 已实现（逐字）：internal/ui/approval.go:approvalUseToolAction「使用工具」 |
| R-AP-051 | G-C1-03 | 异常 | 工具名未知且动作映射为空 | 仍显示「使用工具」，不出现空标题 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-052 | G-C1-03 | 记忆 | 同一 MCP 工具再次请求 | 仍为「使用工具」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/ui/approval.go:approvalUseToolAction「使用工具」 |
| R-AP-053 | G-C1-03 | 多语言 | 把界面语言切到英文时 | 「Use tool」，词序按语言包 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 已实现（映射 = G-C1-03→internal/ui/approval.go:approvalTitle；标记 = 文案「正在加载成员操作...」等 4 条） |
| R-AP-054 | G-C1-03 | 窄窗 | 窗口收窄到最窄宽度时 | 单行不折行 | — | 窄窗下 1 行 | 未实现（映射到位但没有该维度的标记）：G-C1-03→internal/ui/approval.go:approvalTitle |
| R-AP-055 | G-C1-04 | 状态 | 未观察（条件）：请求来自团队成员分支（`memberApproval`） | 标题与输入区之间出现成员区块：浅底 30%、边框 60%、内边距 12，宽度撑满卡片内宽 | — | 判据 = 底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d；描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；内边距 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28）/ 卡片外框 4.5；档位 3.5·7·10.5·14·21·28px；栏宽·布局 输入区最大宽 672 | 已实现（映射 = G-C1-04→internal/ui/approval.go:App.memberApprovalBlock；标记 = State） |
| R-AP-056 | G-C1-04 | 关联 | 成员审批区块在场、区块内各段依次渲染时 | 区块内自上而下：区块标题、成员标识、主分支、错误/不可用说明、审批 ID、原因 | — | 排列顺序 = 区块标题 → 成员标识 → 主分支 → 错误/不可用说明 → 审批 ID → 原因 | 未实现（映射到位但没有该维度的标记）：G-C1-04→internal/ui/approval.go:App.memberApprovalBlock |
| R-AP-057 | G-C1-04 | 内容 | 成员动作加载中 | 区块只显示标题（加载中文案），下方信息行不出现空标签 | — | 屏上 空标签 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-058 | G-C1-04 | 样式 | 分别切到浅色与深色主题时 | 底色为前景色 30% 透明叠底、描边 60%；深色主题下同样可见 | — | 判据 = 底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d；色值 边框 明 #e5e5e5 / 暗 #2e2e2e；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px | 已实现（映射 = G-C1-04→internal/ui/approval.go:App.memberApprovalBlock；标记 = Width） |
| R-AP-059 | G-C1-04 | 交互 | 鼠标点击区块空白处时 | 无动作（不是按钮）；区块内「重试」才可点 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/ui/approval.go:approvalCopyMemberError「重试」 |
| R-AP-060 | G-C1-04 | 无障碍 | 读屏进入该元素时 | 作为分组读出，名由区块标题承担；内部信息逐行读出 | — | 播报顺序 = 区块标题 → 成员标识 → 主分支 → 错误/不可用说明 → 审批 ID → 原因 | 已实现（映射 = G-C1-04→internal/ui/approval.go:App.memberApprovalBlock；标记 = Label） |
| R-AP-061 | G-C1-04 | 异常 | 成员信息全部缺失 | 区块仍出现（说明这是成员请求），内部只留标题，不留空框 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C1-04→internal/ui/approval.go:App.memberApprovalBlock；标记 = Err） |
| R-AP-062 | G-C1-04 | 记忆 | 切到其它会话再切回时 | 区块按请求重建，不残留上一次成员信息 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；上一次内容出现次数 0（ST-VISIBLE） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-063 | G-C1-04 | 窄窗 | 窗口收窄到最窄宽度时 | 区块内文本折行，审批 ID 可断行 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 未实现（映射到位但没有该维度的标记）：G-C1-04→internal/ui/approval.go:App.memberApprovalBlock |
| R-AP-064 | G-C1-05 | 内容 | 未观察（条件）：成员动作就绪 | 区块标题逐字「该成员想要：{action}」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-C1-05→internal/ui/approval.go:approvalTitle；标记 = 文案「正在加载成员操作...」等 4 条） |
| R-AP-065 | G-C1-05 | 内容 | 成员动作加载中 | 标题逐字「正在加载成员操作...」，三点为英文句点 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalCopyMemberLoading「正在加载成员操作...」 |
| R-AP-066 | G-C1-05 | 内容 | 加载失败 | 标题逐字「无法加载成员操作」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalCopyMemberErrorTitle「无法加载成员操作」 |
| R-AP-067 | G-C1-05 | 内容 | 客户端与服务端都无法给原始动作 | 标题逐字「当前客户端无法审查成员操作」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalCopyMemberUnavailableTitle「当前客户端无法审查成员操作」 |
| R-AP-068 | G-C1-05 | 状态 | 四个分支 | 四段文案互斥，同一时刻只出现一段 | — | 四段计数之和为 1 | 已实现（映射 = G-C1-05→internal/ui/approval.go:approvalTitle；标记 = State） |
| R-AP-069 | G-C1-05 | 样式 | 分别切到浅色与深色主题时 | 14px 级半粗；比外层标题小一档 | — | 判据 = 字号 正文 14px；字重 半粗 600 | 未实现（映射到位但没有该维度的标记）：G-C1-05→internal/ui/approval.go:approvalTitle |
| R-AP-070 | G-C1-05 | 关联 | 成员操作加载失败或不可审查时 | 加载失败时下方跟错误说明（C1-08）；不可审查时跟说明（C1-09） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C1-05→internal/ui/approval.go:approvalTitle |
| R-AP-071 | G-C1-05 | 无障碍 | 读屏进入该元素时 | 作为区块标题读出；加载态播报状态 | — | 读屏文本 = 「该成员想要：{action}」 | 未实现（映射到位但没有该维度的标记）：G-C1-05→internal/ui/approval.go:approvalTitle |
| R-AP-072 | G-C1-05 | 异常 | 加载中无限停留 | 保持加载文案，不自动跳到失败 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-C1-05→internal/ui/approval.go:approvalTitle；标记 = Err） |
| R-AP-073 | G-C1-05 | 记忆 | 重试后 | 按重试结果切换到对应文案 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-C1-05→internal/ui/approval.go:approvalTitle |
| R-AP-074 | G-C1-06 | 内容 | 未观察（条件）：成员请求带 `agentId` 与 `branchId` | 文字行逐字「成员 {agentId} · 分支 {branchId}」，中间用中点分隔 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-C1-06→internal/ui/approval.go:App.memberApprovalBlock；标记 = 文案「成员审批」等 3 条） |
| R-AP-075 | G-C1-06 | 内容 | 只有 agentId | 该行只显示成员段，不留「· 分支」空段 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:memberLabel「· 分支」 |
| R-AP-076 | G-C1-06 | 样式 | 成员标识行与主分支行同时在场时 | 14px 级次要色；两行左对齐、行距紧 | — | 判据 = 字号 正文 14px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a；行距 3.5 / 7 / 10.5 / 14px（3.5px 网格） | 已实现（映射 = G-C1-06→internal/ui/approval.go:App.memberApprovalBlock；标记 = Width） |
| R-AP-077 | G-C1-06 | 无障碍 | 读屏进入该元素时 | 读出成员与分支标识 | — | 读屏文本 = 「成员 {agentId} · 分支 {branc」 | 已实现（映射 = G-C1-06→internal/ui/approval.go:App.memberApprovalBlock；标记 = Label） |
| R-AP-078 | G-C1-06 | 异常 | 标识超长 | 单行 truncate，`title` 给完整值 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C1-06→internal/ui/approval.go:App.memberApprovalBlock；标记 = Err） |
| R-AP-079 | G-C1-06 | 交互 | 选中标识文本后复制时 | 可选中复制（便于排查） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-C1-06→internal/ui/approval.go:App.memberApprovalBlock；标记 = Clicked） |
| R-AP-080 | G-C1-06 | 记忆 | 切到其它会话再切回时 | 行内标识按请求重建 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-C1-06→internal/ui/approval.go:App.memberApprovalBlock |
| R-AP-081 | G-C1-06 | 状态 | 成员请求但两个标识都缺 | 该行不出现（不留「成员 · 分支」空壳） | — | 屏上 文字行 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-082 | G-C1-06 | 窄窗 | 窗口收窄到最窄宽度时 | 两段标识在窄卡片上折行，中点不落行首 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 未实现（映射到位但没有该维度的标记）：G-C1-06→internal/ui/approval.go:App.memberApprovalBlock |
| R-AP-083 | G-C1-07 | 内容 | 未观察（条件）：存在 leader 分支 | 文字行逐字「主分支 {branchId}」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-C1-07→internal/ui/approval.go:App.memberApprovalBlock；标记 = 文案「成员审批」等 3 条） |
| R-AP-084 | G-C1-07 | 状态 | 无 leader 分支 | 该行不出现，不留空行 | — | 屏上 文字行 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-085 | G-C1-07 | 样式 | 主分支行紧跟成员标识行渲染时 | 同字号同色，紧跟成员行之下 | — | 判据 = 行距 3.5 / 7 / 10.5 / 14px（3.5px 网格）；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-C1-07→internal/ui/approval.go:App.memberApprovalBlock；标记 = Width） |
| R-AP-086 | G-C1-07 | 无障碍 | 读屏进入该元素时 | 读出主分支标识 | — | 读屏文本 = 「主分支 {branchId}」 | 已实现（映射 = G-C1-07→internal/ui/approval.go:App.memberApprovalBlock；标记 = Label） |
| R-AP-087 | G-C1-07 | 异常 | 分支 ID 超长 | truncate，悬停出完整值 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C1-07→internal/ui/approval.go:App.memberApprovalBlock；标记 = Err） |
| R-AP-088 | G-C1-07 | 交互 | 选中屏上文本后复制时 | 可复制 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-C1-07→internal/ui/approval.go:App.memberApprovalBlock；标记 = Clicked） |
| R-AP-089 | G-C1-07 | 记忆 | 切到其它会话再切回时 | 主分支 ID 逐字不变，且仍紧跟成员标识行之后（两行相邻） | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM） | 未实现（映射到位但没有该维度的标记）：G-C1-07→internal/ui/approval.go:App.memberApprovalBlock |
| R-AP-090 | G-C1-07 | 窄窗 | 窗口收窄到最窄宽度时 | 分支 ID 折行或 truncate，不与「主分支」标签拆开 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（逐字）：internal/ui/approval.go:App.memberApprovalBlock「主分支」 |
| R-AP-091 | G-C1-08 | 内容 | 未观察（条件）：成员动作 RPC 失败 | 错误文字逐字「App Server 未能返回原始操作。请重试或拒绝。」，危险色 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalCopyMemberError「App Server 未能返回原始操作。请重试或拒绝。」 |
| R-AP-092 | G-C1-08 | 关联 | 失败说明在场并出现「重试」时 | 失败行下方/右侧出现「重试」，两者成组 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（逐字）：internal/ui/approval.go:approvalCopyMemberError「重试」 |
| R-AP-093 | G-C1-08 | 样式 | 分别切到浅色与深色主题时 | 14px 级危险色；不加底色 | — | 判据 = 色值 破坏 明 #e7000b / 暗 #ff6467；字号 正文 14px | 已实现（映射 = G-C1-08→internal/ui/approval.go:App.toolApprovalBody；标记 = Font） |
| R-AP-094 | G-C1-08 | 无障碍 | 读屏进入该元素时 | `role=alert` 播报全文 | — | 读屏文本 = 「App Server 未能返回原始操作。请重试或」 | 已实现（映射 = G-C1-08→internal/ui/approval.go:App.toolApprovalBody；标记 = Label@App.memberApprovalBlock） |
| R-AP-095 | G-C1-08 | 异常 | 连续两次失败 | 仍只有一段错误文字 | — | 错误段计数为 1 | 已实现（映射 = G-C1-08→internal/ui/approval.go:App.toolApprovalBody；标记 = erR） |
| R-AP-096 | G-C1-08 | 记忆 | 重试成功后 | 错误文字消失，标题切到就绪文案 | — | 屏上 错误文字 计数 = 0 | 已实现（映射 = G-C1-08→internal/ui/approval.go:App.toolApprovalBody；标记 = persist@approvalInputPanel） |
| R-AP-097 | G-C1-08 | 交互 | 鼠标点击错误文字时 | 无动作；只有「重试」（C1-12）可点 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/ui/approval.go:approvalCopyMemberError「重试」 |
| R-AP-098 | G-C1-08 | 窄窗 | 窗口收窄到最窄宽度时 | 整句折为 2–3 行，区块内不横向溢出 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-C1-08→internal/ui/approval.go:App.toolApprovalBody |
| R-AP-099 | G-C1-09 | 内容 | 未观察（条件）：协议缺原始动作字段（双方都不可审查） | 说明逐字「当前客户端与 App Server 无法显示原始操作，只能拒绝。」，次要色 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalCopyMemberUnavailable「当前客户端与 App Server 无法显示原始操作，只能拒绝。」 |
| R-AP-100 | G-C1-09 | 关联 | 不可审查说明下方跟出审批 ID 行时 | 本行之下跟「审批 {approvalId}」，便于上报 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C1-09→internal/ui/approval.go:App.toolApprovalBody |
| R-AP-101 | G-C1-09 | 关联 | 按钮可用性 | 出现本行时只保留「拒绝」类动作，「仅这次」「始终允许」不出现 | — | 屏上 说明文字 计数 = 1 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-102 | G-C1-09 | 样式 | 分别切到浅色与深色主题时 | 14px 级次要色；不用危险色（不是报错，是能力缺失） | — | 判据 = 色值 次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-C1-09→internal/ui/approval.go:App.toolApprovalBody；标记 = Font） |
| R-AP-103 | G-C1-09 | 无障碍 | 读屏进入该元素时 | 读出整句，并说明只能拒绝 | — | 读屏文本 = 「当前客户端与 App Server 无法显示原始」 | 已实现（映射 = G-C1-09→internal/ui/approval.go:App.toolApprovalBody；标记 = Label@App.memberApprovalBlock） |
| R-AP-104 | G-C1-09 | 异常 | 之后协议补齐原始动作 | 本行消失，改显示正常说明与完整按钮组 | — | 屏上 说明文字 计数 = 0 | 已实现（映射 = G-C1-09→internal/ui/approval.go:App.toolApprovalBody；标记 = erR） |
| R-AP-105 | G-C1-09 | 记忆 | 切到其它会话再切回时 | 整句逐字不变，仍位于成员区块标题之下 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C1-09→internal/ui/approval.go:App.toolApprovalBody；标记 = persist@approvalInputPanel） |
| R-AP-106 | G-C1-09 | 窄窗 | 窗口收窄到最窄宽度时 | 长句折行，「只能拒绝」语义不被裁掉 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（逐字）：internal/ui/approval.go:approvalCopyMemberUnavailable「只能拒绝」 |
| R-AP-107 | G-C1-10 | 内容 | 未观察（条件）：成员动作不可审查 | 等宽文字逐字「审批 {approvalId}」，12px 次要色，可断行 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C1-10→internal/ui/approval.go:App.memberApprovalBlock；标记 = 文案「成员审批」等 3 条） |
| R-AP-108 | G-C1-10 | 内容 | approvalId 超长 | 按字符断行（不横向溢出区块） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C1-10→internal/ui/approval.go:App.memberApprovalBlock；标记 = 文案「成员审批」等 3 条） |
| R-AP-109 | G-C1-10 | 交互 | 选中屏上文本后复制时 | 可复制完整 ID（便于上报） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C1-10→internal/ui/approval.go:App.memberApprovalBlock；标记 = Clicked） |
| R-AP-110 | G-C1-10 | 样式 | 等宽文本渲染后 | 12px 级等宽字体、次要色 | — | 判据 = 字号 UI 正文 12.25px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a；其余 = 字号 12.25 / 行高 18 / 字重 400，说明文字 10.5 / 14（TX-TYPE-BODY / TX-TYPE-MICRO）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C1-10→internal/ui/approval.go:App.memberApprovalBlock；标记 = Width） |
| R-AP-111 | G-C1-10 | 无障碍 | 读屏进入该元素时 | 读出「审批」与 ID | — | 读屏文本 = 「审批」 | 已实现（逐字）：internal/ui/approval.go:approvalCopyAutoReviewFailed「审批」 |
| R-AP-112 | G-C1-10 | 异常 | ID 缺失 | 该行不出现（不显示「审批 」尾巴） | — | 屏上 等宽文字 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-113 | G-C1-10 | 记忆 | 切到其它会话再切回时 | approvalId 逐字不变，仍等宽、位于说明行之后 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM） | 未实现（映射到位但没有该维度的标记）：G-C1-10→internal/ui/approval.go:App.memberApprovalBlock |
| R-AP-114 | G-C1-10 | 高DPI | 在 DPR=2 的屏幕上时 | 等宽 ID 在 2× 下字符宽度一致，断行位置稳定 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C1-10→internal/ui/approval.go:App.memberApprovalBlock |
| R-AP-115 | G-C1-11 | 内容 | 未观察（条件）：成员动作就绪且带 `reason` | 说明文字（14px 次要色）显示原因原文，不截断成省略号 | — | 判据 = 不截断：屏上以「…」收尾的出现次数 = 0，超长按期望列的校验错误行处理（TX-ELLIPSIS 出现次数 0 / TX-TIP）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-116 | G-C1-11 | 内容 | 原因为空串 | 该行不出现 | — | 屏上 说明文字 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-117 | G-C1-11 | 内容 | 原因超长 | 折行为多行，区块高度增长，不横向溢出 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-C1-11→internal/ui/approval.go:App.memberApprovalBlock；标记 = 文案「成员审批」等 3 条） |
| R-AP-118 | G-C1-11 | 关联 | 审批原因行渲染在区块内时 | 位于成员标识与审批 ID 之间（按区块顺序），不与外层标题混排 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM） | 未实现（映射到位但没有该维度的标记）：G-C1-11→internal/ui/approval.go:App.memberApprovalBlock |
| R-AP-119 | G-C1-11 | 样式 | 分别切到浅色与深色主题时 | 14px 级次要色；不用引号包裹 | — | 判据 = 字号 正文 14px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-C1-11→internal/ui/approval.go:App.memberApprovalBlock；标记 = Width） |
| R-AP-120 | G-C1-11 | 无障碍 | 读屏进入该元素时 | 读出原因全文 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-C1-11→internal/ui/approval.go:App.memberApprovalBlock；标记 = Label） |
| R-AP-121 | G-C1-11 | 异常 | 原因含换行 | 保留换行（按原文分行），不压成一行 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-C1-11→internal/ui/approval.go:App.memberApprovalBlock；标记 = Err） |
| R-AP-122 | G-C1-11 | 交互 | 选中原因文本后复制时 | 可选中复制原因原文，不触发任何动作 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-123 | G-C1-12 | 状态 | 未观察（条件）：成员动作加载失败且可重试 | 次级小型描边按钮「重试」，位于成员区块内 | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px） | 已实现（逐字）：internal/ui/approval.go:approvalCopyMemberError「重试」 |
| R-AP-124 | G-C1-12 | 交互 | 鼠标点击该元素时 | 重新拉取成员动作；标题切回「正在加载成员操作...」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalCopyMemberLoading「正在加载成员操作...」 |
| R-AP-125 | G-C1-12 | 状态 | 加载进行中时 | 按钮禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-C1-12→internal/ui/approval.go:App.memberApprovalBlock；标记 = State） |
| R-AP-126 | G-C1-12 | 关联 | 失败说明在场时 | 失败文字在场时才出现 | — | 无失败时按钮计数为 0 | 未实现（映射到位但没有该维度的标记）：G-C1-12→internal/ui/approval.go:App.memberApprovalBlock |
| R-AP-127 | G-C1-12 | 样式 | 「重试」与主按钮同屏出现时 | 同为小号按钮但为 `outline` 次级样式，比主按钮轻 | — | 判据 = 控件尺寸 按钮高 次 28；描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d | 已实现（映射 = G-C1-12→internal/ui/approval.go:App.memberApprovalBlock；标记 = Width） |
| R-AP-128 | G-C1-12 | 无障碍 | 读屏进入该元素时 | 读作按钮「重试」，说明重试的是成员操作 | — | 读屏文本 = 「重试」 | 已实现（逐字）：internal/ui/approval.go:approvalCopyMemberError「重试」 |
| R-AP-129 | G-C1-12 | 键盘 | 按 Enter 时 | 重新拉取成员操作，标题切回「正在加载成员操作...」 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（逐字）：internal/ui/approval.go:approvalCopyMemberLoading「正在加载成员操作...」 |
| R-AP-130 | G-C1-12 | 异常 | 重试再失败 | 仍可重试，不进入死态 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-C1-12→internal/ui/approval.go:App.memberApprovalBlock；标记 = Err） |
| R-AP-131 | G-C1-13 | 状态 | A 态：工具展示区解析成功 | 输入区块在场：387,552 613×27 起，含「输入」标签、命令行（`$` + `ls -la`）、参数行（`cwd: {工作区路径}`）；最大高 240，超出滚动 | — | 输入区块起点 387,552；标签行高 27 | 已实现（逐字）：internal/app/contextusage.go:App.ReadContextUsage「输入」 |
| R-AP-132 | G-C1-13 | 状态 | 工具展示区解析失败（无匹配渲染器） | 输入区块不出现，标题与按钮行之间留空或直接相接；不显示空框 | — | 屏上 空框 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-133 | G-C1-13 | 关联 | 同一请求换用不同工具、输入区内容随之更换时 | 区块内容由 C4 的渲染器决定（命令 / 文件 / 搜索 / 网页 / 技能 / 子代理 / 浏览器 / CUA / 等待 / 通用） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-C1-13→internal/ui/approval.go:App.toolApprovalBody |
| R-AP-134 | G-C1-13 | 样式 | 分别切到浅色与深色主题时 | 边框 1px + 卡片底色（比页面底色略深或略浅一档）；圆角小 | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px | 已实现（映射 = G-C1-13→internal/ui/approval.go:App.toolApprovalBody；标记 = Font） |
| R-AP-135 | G-C1-13 | 交互 | 内容超过 240 高 | 区块内部滚动，滚动条贴区块右缘；卡片整体高度不无限增长 | — | 输入区块高 = 240 封顶，超出内部滚动 | 已实现（映射 = G-C1-13→internal/ui/approval.go:App.toolApprovalBody；标记 = Clicked@App.memberApprovalBlock） |
| R-AP-136 | G-C1-13 | 无障碍 | 读屏进入该元素时 | 区块可读，但整块不进入 Tab 顺序（只读展示） | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-C1-13→internal/ui/approval.go:App.toolApprovalBody；标记 = Label@App.memberApprovalBlock） |
| R-AP-137 | G-C1-13 | 异常 | 内容解码失败 | 区块内给失败提示，不显示半截乱码 | — | 屏上 半截乱码 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-138 | G-C1-13 | 窄窗 | 窗口收窄到最窄宽度时 | 区块宽度随卡片收缩，命令行横向滚动 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C1-13→internal/ui/approval.go:App.toolApprovalBody |
| R-AP-139 | G-C1-13 | 高DPI | 在 DPR=2 的屏幕上时 | 描边与等宽字符在 2× 下清晰 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C1-13→internal/ui/approval.go:App.toolApprovalBody |
| R-AP-140 | G-C1-13 | 记忆 | 切到其它会话再切回时 | 区块内容按请求重建，滚动位置回顶 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C1-13→internal/ui/approval.go:App.toolApprovalBody；标记 = persist@approvalInputPanel） |
| R-AP-141 | G-C1-14 | 状态 | 未观察（条件）：自动审批返回 `reviewer_uncertain` | 出现提示块：标题「需要你判断」+ 交回原因；浅底 50%、内边距 12 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalCopyNeedsDecision「需要你判断」 |
| R-AP-142 | G-C1-14 | 内容 | 无交回原因 | 只显示标题，不留空行 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-C1-14→internal/ui/approval.go:App.handoffBlock；标记 = 文案「自动审批提示」） |
| R-AP-143 | G-C1-14 | 关联 | 自动审批未完成、按钮组仍在下方时 | 提示块之下仍是完整三枚按钮（由人决定） | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C1-14→internal/ui/approval.go:App.handoffBlock |
| R-AP-144 | G-C1-14 | 样式 | 分别切到浅色与深色主题时 | 标题 14px 半粗；块底色为前景色 50% 透明叠底 | — | 判据 = 底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d；字号 正文 14px | 已实现（映射 = G-C1-14→internal/ui/approval.go:App.handoffBlock；标记 = Width） |
| R-AP-145 | G-C1-14 | 无障碍 | 读屏进入该元素时 | 作为提示块读出，标题在前、原因在后 | — | 播报顺序 = 作为提示块读出，标题 → 原因 | 已实现（映射 = G-C1-14→internal/ui/approval.go:App.handoffBlock；标记 = Label） |
| R-AP-146 | G-C1-14 | 异常 | 原因超长 | 折行，块高增长，不溢出卡片 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-C1-14→internal/ui/approval.go:App.handoffBlock；标记 = Reason） |
| R-AP-147 | G-C1-14 | 记忆 | 切到其它会话再切回时 | 标题与交回原因逐字不变，位置仍在输入区块之下、按钮行之上 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C1-14→internal/ui/approval.go:App.handoffBlock |
| R-AP-148 | G-C1-14 | 窄窗 | 窗口收窄到最窄宽度时 | 标题不折行、原因折行，两者不重叠 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 未实现（映射到位但没有该维度的标记）：G-C1-14→internal/ui/approval.go:App.handoffBlock |
| R-AP-149 | G-C1-15 | 状态 | 未观察（条件）：自动审批返回 `reviewer_failed` | 提示块逐字「自动审批没有完成，请由你决定。」，14px 次要色 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalCopyAutoReviewFailed「自动审批没有完成，请由你决定。」 |
| R-AP-150 | G-C1-15 | 关联 | 自动审批以不同返回码结束时 | 两块互斥（不同返回码），不与「需要你判断」同时出现 | — | 两块计数之和为 1 | 已实现（逐字）：internal/ui/approval.go:approvalCopyNeedsDecision「需要你判断」 |
| R-AP-151 | G-C1-15 | 样式 | 分别切到浅色与深色主题时 | 14px 级次要色；块底色 50% 叠底、内边距 12 | — | 判据 = 底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d；字号 正文 14px | 已实现（映射 = G-C1-15→internal/ui/approval.go:App.handoffBlock；标记 = Width） |
| R-AP-152 | G-C1-15 | 无障碍 | 读屏进入该元素时 | 读出整句 | — | 读屏文本 = 「自动审批没有完成，请由你决定。」 | 已实现（映射 = G-C1-15→internal/ui/approval.go:App.handoffBlock；标记 = Label） |
| R-AP-153 | G-C1-15 | 异常 | 自动审批随后恢复 | 刷新后提示块消失，回到普通卡 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 已实现（映射 = G-C1-15→internal/ui/approval.go:App.handoffBlock；标记 = Reason） |
| R-AP-154 | G-C1-15 | 记忆 | 切到其它会话再切回时 | 整句逐字不变，块底色与内边距不变 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-C1-15→internal/ui/approval.go:App.handoffBlock |
| R-AP-155 | G-C1-15 | 交互 | 鼠标点击提示块时 | 无动作（不是「手动审批」按钮） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-C1-15→internal/ui/approval.go:App.handoffBlock |
| R-AP-156 | G-C1-15 | 窄窗 | 窗口收窄到最窄宽度时 | 整句折行，按钮行仍完整可见 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 未实现（映射到位但没有该维度的标记）：G-C1-15→internal/ui/approval.go:App.handoffBlock |
| R-AP-157 | G-C1-16 | 状态 | 未观察（条件）：点任一动作后提交失败 | 按钮行上方出现错误文字（14px 危险色），卡片保留，按钮恢复可点 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C1-16→internal/ui/approval.go:App.approvalErrorLine |
| R-AP-158 | G-C1-16 | 内容 | 失败原因很长 | 折行显示完整原因，不截断成一行省略 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 未实现（映射到位但没有该维度的标记）：G-C1-16→internal/ui/approval.go:App.approvalErrorLine |
| R-AP-159 | G-C1-16 | 交互 | 失败后重试同一按钮 | 错误文字先清掉再提交（不叠加多条） | — | 重试后错误段计数为 1 | 未实现（映射到位但没有该维度的标记）：G-C1-16→internal/ui/approval.go:App.approvalErrorLine |
| R-AP-160 | G-C1-16 | 样式 | 分别切到浅色与深色主题时 | 14px 级危险色，无底色 | — | 判据 = 色值 破坏 明 #e7000b / 暗 #ff6467；字号 正文 14px | 已实现（映射 = G-C1-16→internal/ui/approval.go:App.approvalErrorLine；标记 = TextColor） |
| R-AP-161 | G-C1-16 | 无障碍 | 读屏进入该元素时 | `role=alert` 播报一次原因 | — | 播报次数 = 1 | 已实现（映射 = G-C1-16→internal/ui/approval.go:App.approvalErrorLine；标记 = Role） |
| R-AP-162 | G-C1-16 | 异常 | 连续失败 | 同一位置更新文案，不堆叠 | — | 错误段恒 1 | 已实现（映射 = G-C1-16→internal/ui/approval.go:App.approvalErrorLine；标记 = Err） |
| R-AP-163 | G-C1-16 | 记忆 | 切到其它会话再切回时 | 请求仍在时失败信息随之保留（若服务端仍视为未答复） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-C1-16→internal/ui/approval.go:App.approvalErrorLine |
| R-AP-164 | G-C1-16 | 窄窗 | 窗口收窄到最窄宽度时 | 失败原因折行，不把按钮行挤出卡片 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 未实现（映射到位但没有该维度的标记）：G-C1-16→internal/ui/approval.go:App.approvalErrorLine |
| R-AP-165 | G-C1-17 | 状态 | A 态：卡片在场 | 按钮组右对齐，三枚按钮同一行（y=632），间距 8；拒绝 48 宽、仅这次 58、始终允许 109 | — | 三枚按钮同排、右对齐、间距 8：拒绝 772,632 48×28、仅这次 827,632 58×28、始终允许 891,632 109×28 | 已实现（映射 = G-C1-17→internal/ui/approval.go:App.approvalActions；标记 = Disabled） |
| R-AP-166 | G-C1-17 | 样式 | 分别切到浅色与深色主题时 | 无底色容器；按钮高 28、圆角小 | — | 判据 = 圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px；间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C1-17→internal/ui/approval.go:App.approvalActions；标记 = Width） |
| R-AP-167 | G-C1-17 | 关联 | 输入区块内容高度增长后 | 按钮行与区块之间留 12–16 间隙，不紧贴 | — | 按钮行上缘 − 区块下缘 = 12–16 | 未实现（映射到位但没有该维度的标记）：G-C1-17→internal/ui/approval.go:App.approvalActions |
| R-AP-168 | G-C1-17 | 交互 | 鼠标悬停在任一按钮时 | 只该按钮出悬停态，其它按钮不动 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-C1-17→internal/ui/approval.go:App.approvalActions；标记 = Clicked） |
| R-AP-169 | G-C1-17 | 无障碍 | 读屏进入该元素时 | 作为按钮组读出，组内按钮按视觉顺序 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-C1-17→internal/ui/approval.go:App.approvalActions；标记 = Label） |
| R-AP-170 | G-C1-17 | 键盘 | 按 Tab 时 | 焦点在组内按视觉顺序移动，循环不逃出 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 未实现（映射到位但没有该维度的标记）：G-C1-17→internal/ui/approval.go:App.approvalActions |
| R-AP-171 | G-C1-17 | 窄窗 | 窗口收窄到最窄宽度时 | 按钮折行时右对齐保持、间距不变、不裁字 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 未实现（映射到位但没有该维度的标记）：G-C1-17→internal/ui/approval.go:App.approvalActions |
| R-AP-172 | G-C1-17 | 异常 | 只提供一种动作（如只能拒绝） | 组内只剩一枚按钮，仍右对齐，不居中 | — | 屏上 按钮组 计数 = 1 | 已实现（映射 = G-C1-17→internal/ui/approval.go:App.approvalActions；标记 = Err@App.respondApproval） |
| R-AP-173 | G-C1-17 | 高DPI | 在 DPR=2 的屏幕上时 | 按钮描边与圆角在 2× 下平滑 | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C1-17→internal/ui/approval.go:App.approvalActions |
| R-AP-174 | G-C1-18 | 状态 | A 态：请求提供一次性允许 | 主按钮「仅这次」在场：827,632 58×28，实心强调色、白字 | — | 屏上文案逐字「仅这次」，包围盒 827,632 58×28 | 已实现（逐字）：internal/ui/approval.go:approvalCopyAllowOnce「仅这次」 |
| R-AP-175 | G-C1-18 | 交互 | 鼠标点击该元素时 | 以「仅这次」答复该请求；卡片消失；工具调用继续 | — | 屏上 卡片 计数 = 0 | 已实现（逐字）：internal/ui/approval.go:approvalCopyAllowOnce「仅这次」 |
| R-AP-176 | G-C1-18 | 关联 | 一次性允许与长期允许按钮同屏时 | 一次性允许与「始终允许」是两个不同答复值，不合并 | — | 屏上 主按钮 计数 = 1 | 已实现（逐字）：internal/ui/approval.go:approvalCopyAlwaysCommand「始终允许」 |
| R-AP-177 | G-C1-18 | 状态 | 成员审批请求到达时 | 文案仍为「仅这次」（成员允许全部也显示此文案，不复用「始终允许」） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalCopyAllowOnce「仅这次」 |
| R-AP-178 | G-C1-18 | 状态 | 提交进行中时 | 按钮禁用（50% 不透明），文案不变 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED） | 未实现（映射到位但没有该维度的标记）：G-C1-18→internal/ui/approval.go:approvalActionsFor |
| R-AP-179 | G-C1-18 | 样式 | 分别切到浅色与深色主题时 | 强调色实心（浅色主题为绿/品牌色），白字，28 高、圆角小 | — | 判据 = 底色 主色 明 #339751 / 暗 #63cb7b；字色 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C1-18→internal/ui/approval.go:approvalActionsFor；标记 = Type@approvalSandboxLongTermCopy） |
| R-AP-180 | G-C1-18 | 样式 | 鼠标悬停在行上时 | 底色加深一档，白字不变 | — | 判据 = 底色 悬停底 明 #f5f5f5 / 暗 #262626（侧栏 明 #e2e9ed / 暗 #282c2b） | 已实现（映射 = G-C1-18→internal/ui/approval.go:approvalActionsFor；标记 = Type@approvalSandboxLongTermCopy） |
| R-AP-181 | G-C1-18 | 关联 | 三枚按钮同屏排列时 | 「拒绝」在左、本按钮居中，两按钮间距 8 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（逐字）：internal/ui/approval.go:approvalCopyReject「拒绝」 |
| R-AP-182 | G-C1-18 | 无障碍 | 读屏进入该元素时 | 读作按钮「仅这次」，并说明是「只允许这一次」 | — | 读屏文本 = 「仅这次」 / 「只允许这一次」 | 已实现（逐字）：internal/ui/approval.go:approvalCopyAllowOnce「仅这次」 |
| R-AP-183 | G-C1-18 | 键盘 | Enter / Space | 以「仅这次」答复该请求，卡片消失 | — | 屏上 卡片 计数 = 0 | 已实现（逐字）：internal/ui/approval.go:approvalCopyAllowOnce「仅这次」 |
| R-AP-184 | G-C1-18 | 异常 | 提交失败 | 卡片保留、按钮回可点、显示 C1-16 错误 | — | 卡片 计数 = 1；主按钮 可点状态 = 可点 | 未实现（映射到位但没有该维度的标记）：G-C1-18→internal/ui/approval.go:approvalActionsFor |
| R-AP-185 | G-C1-18 | 异常 | 快速连续点击时 | 只提交一次答复；第二次点击被吞掉 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-C1-18→internal/ui/approval.go:approvalActionsFor |
| R-AP-186 | G-C1-18 | 记忆 | 同一会话再出现同类请求 | 按钮默认可点（不复用上次的禁用态） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-C1-18→internal/ui/approval.go:approvalActionsFor |
| R-AP-187 | G-C1-18 | 窄窗 | 窗口收窄到最窄宽度时 | 主按钮不被裁，文案完整 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-C1-18→internal/ui/approval.go:approvalActionsFor |
| R-AP-188 | G-C1-18 | 多语言 | 把界面语言切到英文时 | 「Just this once」，宽度自适应，不裁字 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C1-18→internal/ui/approval.go:approvalActionsFor；标记 = 文案「拒绝」等 5 条） |
| R-AP-189 | G-C1-18 | 内容 | 无一次性允许选项 | 按钮不出现（组内只剩其他动作），不留空位 | — | 屏上 主按钮 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-190 | G-C1-19 | 状态 | A 态：普通工具请求 | 次按钮「拒绝」在场：772,632 48×28，描边次按钮 | — | 屏上文案逐字「拒绝」，包围盒 772,632 48×28 | 已实现（逐字）：internal/ui/approval.go:approvalCopyReject「拒绝」 |
| R-AP-191 | G-C1-19 | 内容 | 沙箱越权卡（C2） | 同一位置文案换成「不允许」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（逐字）：internal/ui/approval.go:approvalCopyDontAllow「不允许」 |
| R-AP-192 | G-C1-19 | 交互 | 鼠标点击该元素时 | 以拒绝答复；卡片消失；工具调用不执行 | — | 屏上 卡片 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-193 | G-C1-19 | 状态 | 提交进行中时 | 禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-C1-19→internal/ui/approval.go:approvalActionsFor |
| R-AP-194 | G-C1-19 | 关联 | 三枚按钮同屏排列时 | 拒绝在最左（最先读到的位置），主按钮在其右 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C1-19→internal/ui/approval.go:approvalActionsFor |
| R-AP-195 | G-C1-19 | 样式 | 分别切到浅色与深色主题时 | 无底色、1px 描边、主前景色文字；高 28 | — | 判据 = 色值 边框 明 #e5e5e5 / 暗 #2e2e2e；底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d；字色 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-C1-19→internal/ui/approval.go:approvalActionsFor；标记 = Type@approvalSandboxLongTermCopy） |
| R-AP-196 | G-C1-19 | 无障碍 | 读屏进入该元素时 | 读作按钮「拒绝」 | — | 读屏文本 = 「拒绝」 | 已实现（逐字）：internal/ui/approval.go:approvalCopyReject「拒绝」 |
| R-AP-197 | G-C1-19 | 键盘 | 按 Enter 时 | 以拒绝答复该请求，卡片消失 | — | 屏上 卡片 计数 = 0 | 未实现（映射到位但没有该维度的标记）：G-C1-19→internal/ui/approval.go:approvalActionsFor |
| R-AP-198 | G-C1-19 | 异常 | 提交失败 | 卡片保留、按钮回可点、显示 C1-16 | — | 卡片 计数 = 1；次按钮 可点状态 = 可点 | 未实现（映射到位但没有该维度的标记）：G-C1-19→internal/ui/approval.go:approvalActionsFor |
| R-AP-199 | G-C1-19 | 异常 | 不可审查的成员请求 | 「拒绝」仍在，其他按钮不出现 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-200 | G-C1-19 | 记忆 | 新请求 | 按钮可用 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-C1-19→internal/ui/approval.go:approvalActionsFor |
| R-AP-201 | G-C1-19 | 窄窗 | 窗口收窄到最窄宽度时 | 文案完整不裁 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-C1-19→internal/ui/approval.go:approvalActionsFor |
| R-AP-202 | G-C1-19 | 多语言 | 把界面语言切到英文时 | 「Reject」/「Don't allow」按场景 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 已实现（映射 = G-C1-19→internal/ui/approval.go:approvalActionsFor；标记 = 文案「拒绝」等 5 条） |
| R-AP-203 | G-C1-20 | 状态 | A 态：`allow_all` 且工具为 `exec` | 次按钮「始终允许此命令」在场：891,632 109×28 | — | 屏上文案逐字「始终允许此命令」，包围盒 891,632 109×28 | 已实现（逐字）：internal/ui/approval.go:approvalCopyAlwaysCommand「始终允许此命令」 |
| R-AP-204 | G-C1-20 | 内容 | `read`/`write`/`edit` 的 `allow_all` | 同一位置文案换「允许此会话期间访问」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（逐字）：internal/ui/approval.go:approvalCopyAllowForSession「允许此会话期间访问」 |
| R-AP-205 | G-C1-20 | 内容 | 其他工具的 `allow_all` | 文案为「始终允许文件访问」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalCopyAlwaysFile「始终允许文件访问」 |
| R-AP-206 | G-C1-20 | 状态 | 三种文案互斥 | 同一按钮只显示一种，由工具类别决定 | — | 屏上 次按钮 计数 = 1 | 未实现（映射到位但没有该维度的标记）：G-C1-20→internal/ui/approval.go:approvalLongTermCopy；测试提及 internal/ui/negative_evidence_test.go:包级 |
| R-AP-207 | G-C1-20 | 交互 | 鼠标点击该元素时 | 以「始终允许」答复；卡片消失；后续同类请求不再询问 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 已实现（逐字）：internal/ui/approval.go:approvalCopyAlwaysCommand「始终允许」 |
| R-AP-208 | G-C1-20 | 关联 | 两种放行按钮先后出现时 | 长期允许与一次性允许答复值不同 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-C1-20→internal/ui/approval.go:approvalLongTermCopy；测试提及 internal/ui/negative_evidence_test.go:包级 |
| R-AP-209 | G-C1-20 | 状态 | 提交进行中时 | 禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-C1-20→internal/ui/approval.go:approvalLongTermCopy；测试提及 internal/ui/negative_evidence_test.go:包级 |
| R-AP-210 | G-C1-20 | 样式 | 拒绝与长期允许按钮同屏时 | 同尺寸（高 28）、描边次按钮；只因文案长短宽度不同（109 vs 48） | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；内边距 左缘 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28）；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C1-20→internal/ui/approval.go:approvalLongTermCopy；标记 = Type@approvalSandboxLongTermCopy） |
| R-AP-211 | G-C1-20 | 无障碍 | 读屏进入该元素时 | 读作按钮，名称与当前文案一致，并说明作用域（命令 / 会话 / 文件） | — | 播报顺序 = 始终允许此命令 → 允许此会话期间访问 → 始终允许文件访问 | 未实现（映射到位但没有该维度的标记）：G-C1-20→internal/ui/approval.go:approvalLongTermCopy；测试提及 internal/ui/negative_evidence_test.go:包级 |
| R-AP-212 | G-C1-20 | 键盘 | 按 Enter 时 | 以长期允许答复该请求，卡片消失 | — | 屏上 卡片 计数 = 0 | 未实现（映射到位但没有该维度的标记）：G-C1-20→internal/ui/approval.go:approvalLongTermCopy；测试提及 internal/ui/negative_evidence_test.go:包级 |
| R-AP-213 | G-C1-20 | 异常 | 提交失败 | 卡片保留、按钮回可点 | — | 卡片 计数 = 1；次按钮 可点状态 = 可点 | 未实现（映射到位但没有该维度的标记）：G-C1-20→internal/ui/approval.go:approvalLongTermCopy；测试提及 internal/ui/negative_evidence_test.go:包级 |
| R-AP-214 | G-C1-20 | 异常 | 请求不提供 `allow_all` | 按钮不出现（不显示为禁用） | — | 屏上 为禁用 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-215 | G-C1-20 | 记忆 | 已允许后同一工具再请求 | 不再出卡（长期允许生效），日志里可查 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-C1-20→internal/ui/approval.go:approvalLongTermCopy；测试提及 internal/ui/negative_evidence_test.go:包级 |
| R-AP-216 | G-C1-20 | 窄窗 | 窗口收窄到最窄宽度时 | 长文案仍完整（可缩字号或折行），不裁成「始终允许…」 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS）；字号 12.25 / 行高 18 / 字重 400，说明文字 10.5 / 14（TX-TYPE-BODY / TX-TYPE-MICRO） | 未实现（映射到位但没有该维度的标记）：G-C1-20→internal/ui/approval.go:approvalLongTermCopy；测试提及 internal/ui/negative_evidence_test.go:包级 |
| R-AP-217 | G-C1-20 | 多语言 | 把界面语言切到英文时 | 三种作用域文案各自本地化 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-C1-20→internal/ui/approval.go:approvalLongTermCopy；标记 = 文案「始终允许此命令」等 3 条） |
| R-AP-218 | G-C1-20 | 内容 | 沙箱越权请求到达时 | 同一位置换成「始终允许网络访问 / 始终允许文件访问」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C1-20→internal/ui/approval.go:approvalLongTermCopy；标记 = 文案「始终允许此命令」等 3 条） |
| R-AP-219 | G-C1-21 | 状态 | 未观察（条件）：点动作后提交中 | 整组按钮降到 50% 不透明并禁用，文案不变 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED） | 已实现（映射 = G-C1-21→internal/ui/approval.go:App.approvalActions；标记 = Disabled） |
| R-AP-220 | G-C1-21 | 状态 | 整体禁用（如请求已失效） | 同样 50% 不透明，点击无副作用 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED） | 已实现（映射 = G-C1-21→internal/ui/approval.go:App.approvalActions；标记 = Disabled） |
| R-AP-221 | G-C1-21 | 关联 | 提交失败后按钮恢复可点时 | 失败后恢复 100% 不透明与可点 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED） | 未实现（映射到位但没有该维度的标记）：G-C1-21→internal/ui/approval.go:App.approvalActions |
| R-AP-222 | G-C1-21 | 样式 | 分别切到浅色与深色主题时 | 0.5 不透明叠在各自底色上，不改变按钮尺寸与位置 | — | 判据 = 色值取 token：主色 #349651/#63cb7b、正文 #0a0a0a/#ececec、次文 #737373/#8a8a8a、边框 #e5e5e5/#2e2e2e、警示 #e17100/#ffb900，偏差 0（CL-ACCENT / CL-TEXT / CL-TEXT-2 / CL-BORDER / CL-WARN）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-223 | G-C1-21 | 无障碍 | 读屏进入该元素时 | 禁用态播报「不可用」 | — | 读屏文本 = 「不可用」 | 已实现（逐字）：internal/account/account.go:ErrActionsUnavailable「不可用」 |
| R-AP-224 | G-C1-21 | 交互 | 按钮禁用时鼠标悬停 | 不出悬停底色变化 | — | 判据 = 底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d；描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px；其余 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 已实现（映射 = G-C1-21→internal/ui/approval.go:App.approvalActions；标记 = Clicked） |
| R-AP-225 | G-C1-21 | 异常 | 提交中长时间无响应 | 保持禁用直到超时并转 C1-16 失败 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-C1-21→internal/ui/approval.go:App.approvalActions；标记 = Err@App.respondApproval） |
| R-AP-226 | G-C1-21 | 记忆 | 卡片重渲染重建时 | 不继承上一张卡的禁用态 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-C1-21→internal/ui/approval.go:App.approvalActions |

### 1.2 沙箱越权卡（G-C2-01–G-C2-08）

未观察（条件）：C2-01–C2-08 全部。三份审批请求都是普通 `exec` 权限，不是沙箱拦下后的放宽请求；本组每条的条件列写「权限请求 `variant=sandbox` + 四类资源之一」。

| 需求 ID | 元素 ID | 维度 | 场景 | 期望 | 异常 | 验收 | 状态 |
|---|---|---|---|---|---|---|---|
| R-AP-227 | G-C2-01 | 状态 | 未观察（条件）：权限请求 `variant=sandbox` | 卡片在场，位置与骨架同 C1（贴输入区上沿），边框 + 内边距 16，底色与页面一致 | — | 卡片包围盒与 C1 同高宽量级 | 已实现（映射 = G-C2-01→internal/ui/approval.go:App.approvalCard；标记 = State@App.currentApproval） |
| R-AP-228 | G-C2-01 | 关联 | 沙箱越权请求到达、工具权限卡不在屏上时 | 与工具权限卡互斥，同一时刻只有一张 | — | 卡片计数恒为 1 | 未实现（映射到位但没有该维度的标记）：G-C2-01→internal/ui/approval.go:App.approvalCard |
| R-AP-229 | G-C2-01 | 内容 | 被拦资源为网络 / 读 / 写 / 执行 | 卡片高度随标题与说明行数变化；结构固定为「标题→资源值→说明→按钮」 | — | 排列顺序 = 标题 → 资源值 → 说明 → 按钮 | 已实现（映射 = G-C2-01→internal/ui/approval.go:App.approvalCard；标记 = 文案「审批卡」） |
| R-AP-230 | G-C2-01 | 样式 | 分别切到浅色与深色主题时 | 与 C1 卡同一套描边、圆角、底色 | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px | 已实现（映射 = G-C2-01→internal/ui/approval.go:App.approvalCard；标记 = Width） |
| R-AP-231 | G-C2-01 | 交互 | 鼠标点击卡片空白处时 | 无动作；只有按钮答复 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-C2-01→internal/ui/approval.go:App.approvalCard |
| R-AP-232 | G-C2-01 | 无障碍 | 读屏进入该元素时 | 作为对话框区域，读出名由 C2-02 标题承担 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-C2-01→internal/ui/approval.go:App.approvalCard；标记 = Role） |
| R-AP-233 | G-C2-01 | 异常 | 答复时断线 | 卡片保留并显示 C2-05 错误 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-C2-01→internal/ui/approval.go:App.approvalCard；标记 = Err@App.sandboxApprovalBody） |
| R-AP-234 | G-C2-01 | 记忆 | 切到其它会话再切回时 | 卡片按请求重建 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-C2-01→internal/ui/approval.go:App.approvalCard |
| R-AP-235 | G-C2-01 | 窄窗 | 窗口收窄到最窄宽度时 | 卡片随主区收缩，域名/路径断行 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C2-01→internal/ui/approval.go:App.approvalCard |
| R-AP-236 | G-C2-02 | 内容 | 网络访问被拦 | 标题逐字「网络访问已拦截」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalCopySandboxNetwork「网络访问已拦截」 |
| R-AP-237 | G-C2-02 | 内容 | 文件读取被拦 | 标题逐字「文件读取已拦截」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalCopySandboxRead「文件读取已拦截」 |
| R-AP-238 | G-C2-02 | 内容 | 文件写入被拦 | 标题逐字「文件写入已拦截」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalCopySandboxWrite「文件写入已拦截」 |
| R-AP-239 | G-C2-02 | 内容 | 文件执行被拦 | 标题逐字「文件执行已拦截」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalCopySandboxExecute「文件执行已拦截」 |
| R-AP-240 | G-C2-02 | 状态 | 四类互斥 | 同一时刻只显示一种标题 | — | 屏上 标题 计数 = 1 | 已实现（映射 = G-C2-02→internal/ui/approval.go:App.sandboxApprovalBody；标记 = Disabled@App.approvalActions） |
| R-AP-241 | G-C2-02 | 样式 | 两种卡的标题先后出现时 | 同为 16px 级半粗主前景色 | — | 判据 = 字号 对话框标题 15.75px；字重 半粗 600 | 已实现（映射 = G-C2-02→internal/ui/approval.go:App.sandboxApprovalBody；标记 = Font） |
| R-AP-242 | G-C2-02 | 无障碍 | 读屏进入该元素时 | 读出标题与「被拦截」语义 | — | 读屏文本 = 「被拦截」 | 已实现（映射 = G-C2-02→internal/ui/approval.go:App.sandboxApprovalBody；标记 = Role@App.approvalErrorLine） |
| R-AP-243 | G-C2-02 | 异常 | 资源类型未知 | 回退为「网络访问已拦截」或不显示卡（不出现空标题） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-244 | G-C2-02 | 记忆 | 同类被拦再次发生 | 标题一致 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-C2-02→internal/ui/approval.go:App.sandboxApprovalBody |
| R-AP-245 | G-C2-02 | 多语言 | 把界面语言切到英文时 | 四类标题各自本地化 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 已实现（映射 = G-C2-02→internal/ui/approval.go:App.sandboxApprovalBody；标记 = 文案「网络访问已拦截」等 4 条@sandboxTitle） |
| R-AP-246 | G-C2-03 | 内容 | 未观察（条件）：沙箱卡在场 | 等宽文字显示被拦域名或路径原文，位于标题之下 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C2-03→internal/ui/approval.go:App.sandboxApprovalBody；标记 = 文案「网络访问已拦截」等 4 条@sandboxTitle） |
| R-AP-247 | G-C2-03 | 内容 | 路径很长 | 可换行（等宽 14px），不横向溢出卡片 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-C2-03→internal/ui/approval.go:App.sandboxApprovalBody；标记 = 文案「网络访问已拦截」等 4 条@sandboxTitle） |
| R-AP-248 | G-C2-03 | 内容 | 域名（无路径） | 只显示域名，不带 `https://` 前缀以外的多余字符串（按请求原文） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C2-03→internal/ui/approval.go:App.sandboxApprovalBody；标记 = 文案「网络访问已拦截」等 4 条@sandboxTitle） |
| R-AP-249 | G-C2-03 | 交互 | 选中屏上文本后复制时 | 可复制域名/路径 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C2-03→internal/ui/approval.go:App.sandboxApprovalBody；标记 = Clicked@App.approvalActions） |
| R-AP-250 | G-C2-03 | 样式 | 等宽文本渲染后 | 14px 级等宽、主前景色；不与说明行同色 | — | 判据 = 字号 正文 14px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a；其余 = 字号 12.25 / 行高 18 / 字重 400，说明文字 10.5 / 14（TX-TYPE-BODY / TX-TYPE-MICRO）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C2-03→internal/ui/approval.go:App.sandboxApprovalBody；标记 = Font） |
| R-AP-251 | G-C2-03 | 无障碍 | 读屏进入该元素时 | 读出域名/路径原文 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-C2-03→internal/ui/approval.go:App.sandboxApprovalBody；标记 = Role@App.approvalErrorLine） |
| R-AP-252 | G-C2-03 | 异常 | 值为空 | 该行不出现，不留空行 | — | 屏上 等宽文字 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-253 | G-C2-03 | 记忆 | 切到其它会话再切回时 | 域名/路径逐字不变，仍位于标题之下、说明之上 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C2-03→internal/ui/approval.go:App.sandboxApprovalBody |
| R-AP-254 | G-C2-04 | 内容 | 网络访问被拦 | 说明逐字「{tool} 需要访问该域名。」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-C2-04→internal/ui/approval.go:sandboxMessage；标记 = 文案「需要访问该路径。」等 5 条） |
| R-AP-255 | G-C2-04 | 内容 | 文件读取被拦 | 说明逐字「需要读取该路径。」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalCopySandboxReadMessage「需要读取该路径。」 |
| R-AP-256 | G-C2-04 | 内容 | 文件写入被拦 | 说明逐字「需要写入该路径。」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalCopySandboxWriteMessage「需要写入该路径。」 |
| R-AP-257 | G-C2-04 | 内容 | 文件执行被拦 | 说明逐字「需要执行该路径。」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalCopySandboxExecuteMessage「需要执行该路径。」 |
| R-AP-258 | G-C2-04 | 关联 | 拦截标题与对应说明同时在场时 | 说明与标题成对：读标题→读对应说明 | — | 排列顺序 = 读标题 → 读对应说明 | 未实现（映射到位但没有该维度的标记）：G-C2-04→internal/ui/approval.go:sandboxMessage |
| R-AP-259 | G-C2-04 | 样式 | 资源值行与说明行同时在场时 | 14px 级次要色，位于资源值之下 | — | 判据 = 字号 正文 14px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a；行距 3.5 / 7 / 10.5 / 14px（3.5px 网格） | 已实现（映射 = G-C2-04→internal/ui/approval.go:sandboxMessage；标记 = Type） |
| R-AP-260 | G-C2-04 | 无障碍 | 读屏进入该元素时 | 读出整句（含工具名） | — | 读屏文本 = 「{tool} 需要访问该域名。」 | 已实现（映射 = G-C2-04→internal/ui/approval.go:App.sandboxApprovalBody；标记 = Role@App.approvalErrorLine） |
| R-AP-261 | G-C2-04 | 异常 | 工具名为空 | 说明省去工具名，改为「需要访问该域名。」，不留空主语 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalCopySandboxNetworkMessage「需要访问该域名。」 |
| R-AP-262 | G-C2-04 | 记忆 | 切到其它会话再切回时 | 说明逐字不变，且与当前标题仍然配对（不串到另一类资源） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-C2-04→internal/ui/approval.go:sandboxMessage |
| R-AP-263 | G-C2-05 | 状态 | 未观察（条件）：沙箱卡上点任一动作后提交失败 | 按钮行上方出现错误文字（14px 危险色） | — | 判据 = 字号 正文 14px；字重 常规 400 / 中粗 500 / 半粗 600 / 粗 700 | 未实现（映射到位但没有该维度的标记）：G-C2-05→internal/ui/approval.go:App.approvalErrorLine |
| R-AP-264 | G-C2-05 | 交互 | 失败后再点 | 错误先清除再提交，不叠加 | — | 错误段恒 1 | 未实现（映射到位但没有该维度的标记）：G-C2-05→internal/ui/approval.go:App.approvalErrorLine |
| R-AP-265 | G-C2-05 | 样式 | 沙箱卡提交出错时 | 同字号同色同位置关系 | — | 判据 = 字号 说明文字 10.5px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-C2-05→internal/ui/approval.go:App.approvalErrorLine；标记 = TextColor） |
| R-AP-266 | G-C2-05 | 无障碍 | 读屏进入该元素时 | `role=alert` 播报一次 | — | 播报次数 = 1 | 已实现（映射 = G-C2-05→internal/ui/approval.go:App.approvalErrorLine；标记 = Role） |
| R-AP-267 | G-C2-05 | 异常 | 连续失败 | 同一位置更新文案 | — | 错误段恒 1 | 已实现（映射 = G-C2-05→internal/ui/approval.go:App.approvalErrorLine；标记 = Err） |
| R-AP-268 | G-C2-05 | 记忆 | 切到其它会话再切回时 | 请求仍在则错误保留 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-C2-05→internal/ui/approval.go:App.approvalErrorLine |
| R-AP-269 | G-C2-05 | 关联 | 沙箱卡提交出错、按钮组仍留在屏上时 | 失败后按钮恢复可点，两者同屏且不重叠 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-C2-05→internal/ui/approval.go:App.approvalErrorLine |
| R-AP-270 | G-C2-05 | 窄窗 | 窗口收窄到最窄宽度时 | 失败原因折行，卡片高度增长 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 未实现（映射到位但没有该维度的标记）：G-C2-05→internal/ui/approval.go:App.approvalErrorLine |
| R-AP-271 | G-C2-06 | 状态 | 未观察（条件）：沙箱卡在场 | 次按钮「不允许」，位于按钮组最左 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalCopyDontAllow「不允许」 |
| R-AP-272 | G-C2-06 | 交互 | 鼠标点击该元素时 | 以拒绝答复；工具不执行；卡片消失 | — | 屏上 卡片 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-273 | G-C2-06 | 关联 | 沙箱卡按钮组出现时 | 同一按钮位的另一文案（沙箱场景用「不允许」） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalCopyDontAllow「不允许」 |
| R-AP-274 | G-C2-06 | 样式 | 次按钮在屏上时 | 无底色、1px 描边、高 28 | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C2-06→internal/ui/approval.go:approvalActionsFor；标记 = Type@approvalSandboxLongTermCopy） |
| R-AP-275 | G-C2-06 | 无障碍 | 读屏进入该元素时 | 读作「不允许」 | — | 读屏文本 = 「不允许」 | 已实现（逐字）：internal/ui/approval.go:approvalCopyDontAllow「不允许」 |
| R-AP-276 | G-C2-06 | 键盘 | 按 Enter 时 | 以「不允许」答复，卡片消失且工具不执行 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-277 | G-C2-06 | 异常 | 提交失败 | 卡片保留并显示 C2-05 | — | 卡片 计数 = 1；错误文字 计数 = 1 | 未实现（映射到位但没有该维度的标记）：G-C2-06→internal/ui/approval.go:approvalActionsFor |
| R-AP-278 | G-C2-06 | 记忆 | 新沙箱请求到达 | 按钮回可点（不复用上一卡的禁用态） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-C2-06→internal/ui/approval.go:approvalActionsFor |
| R-AP-279 | G-C2-07 | 状态 | 未观察（条件）：沙箱卡在场 | 主按钮「仅这次」，实心强调色 | — | 判据 = 底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d；其余 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalCopyAllowOnce「仅这次」 |
| R-AP-280 | G-C2-07 | 交互 | 鼠标点击该元素时 | 本次放行该访问；卡片消失；工具重试 | — | 屏上 卡片 计数 = 0 | 未实现（映射到位但没有该维度的标记）：G-C2-07→internal/ui/approval.go:approvalActionsFor |
| R-AP-281 | G-C2-07 | 关联 | 沙箱卡两个放行按钮同屏时 | 一次性放行与长期放行是两个答复值 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-C2-07→internal/ui/approval.go:approvalActionsFor |
| R-AP-282 | G-C2-07 | 样式 | 沙箱卡主按钮与工具卡主按钮先后出现时 | 同尺寸、同底色、白字 | — | 判据 = 底色 主色 明 #339751 / 暗 #63cb7b；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C2-07→internal/ui/approval.go:approvalActionsFor；标记 = Type@approvalSandboxLongTermCopy） |
| R-AP-283 | G-C2-07 | 无障碍 | 读屏进入该元素时 | 读作「仅这次」 | — | 读屏文本 = 「仅这次」 | 已实现（逐字）：internal/ui/approval.go:approvalCopyAllowOnce「仅这次」 |
| R-AP-284 | G-C2-07 | 键盘 | 按 Enter 时 | 本次放行该访问，卡片消失 | — | 屏上 卡片 计数 = 0 | 未实现（映射到位但没有该维度的标记）：G-C2-07→internal/ui/approval.go:approvalActionsFor |
| R-AP-285 | G-C2-07 | 异常 | 提交失败 | 卡片保留并显示 C2-05 | — | 卡片 计数 = 1；错误文字 计数 = 1 | 未实现（映射到位但没有该维度的标记）：G-C2-07→internal/ui/approval.go:approvalActionsFor |
| R-AP-286 | G-C2-07 | 记忆 | 新沙箱请求到达 | 主按钮可用 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-C2-07→internal/ui/approval.go:approvalActionsFor |
| R-AP-287 | G-C2-08 | 内容 | 未观察（条件）：沙箱卡在场且资源为网络 | 次按钮「始终允许网络访问」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalCopyAlwaysNetwork「始终允许网络访问」 |
| R-AP-288 | G-C2-08 | 内容 | 资源为文件（读/写/执行） | 文案「始终允许文件访问」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalCopyAlwaysFile「始终允许文件访问」 |
| R-AP-289 | G-C2-08 | 状态 | 两类互斥 | 同一按钮只显示一种文案 | — | 屏上 次按钮 计数 = 1 | 未实现（映射到位但没有该维度的标记）：G-C2-08→internal/ui/approval.go:approvalSandboxLongTermCopy |
| R-AP-290 | G-C2-08 | 交互 | 鼠标点击该元素时 | 长期放行该类访问；后续同类被拦不再出卡 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-C2-08→internal/ui/approval.go:approvalSandboxLongTermCopy |
| R-AP-291 | G-C2-08 | 样式 | 「不允许」与「始终允许」同屏时 | 同尺寸描边次按钮；宽度随文案变 | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C2-08→internal/ui/approval.go:approvalSandboxLongTermCopy；标记 = Type） |
| R-AP-292 | G-C2-08 | 无障碍 | 读屏进入该元素时 | 读出作用域（网络/文件） | — | 读屏文本 = 「始终允许网络访问」 | 未实现（映射到位但没有该维度的标记）：G-C2-08→internal/ui/approval.go:approvalSandboxLongTermCopy |
| R-AP-293 | G-C2-08 | 键盘 | 按 Enter 时 | 长期放行该类访问，后续同类被拦不再出卡 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 未实现（映射到位但没有该维度的标记）：G-C2-08→internal/ui/approval.go:approvalSandboxLongTermCopy |
| R-AP-294 | G-C2-08 | 异常 | 提交失败 | 卡片保留并显示 C2-05 | — | 卡片 计数 = 1；错误文字 计数 = 1 | 未实现（映射到位但没有该维度的标记）：G-C2-08→internal/ui/approval.go:approvalSandboxLongTermCopy |
| R-AP-295 | G-C2-08 | 记忆 | 已长期放行后 | 不再出卡，策略在设置/权限域可见 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-C2-08→internal/ui/approval.go:approvalSandboxLongTermCopy |

### 1.3 子代理模型卡（G-C3-01–G-C3-09）

未观察（条件）：C3-01–C3-09 全部（条件：权限请求 `variant=subagent_model`，子任务要换更贵的模型）。

| 需求 ID | 元素 ID | 维度 | 场景 | 期望 | 异常 | 验收 | 状态 |
|---|---|---|---|---|---|---|---|
| R-AP-296 | G-C3-01 | 状态 | 未观察（条件）：`variant=subagent_model` | 卡片在场，与 C1 同骨架（边框 + 内边距 16），位置贴输入区上沿 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C3-01→internal/ui/approval.go:App.subagentModelBody；标记 = Hidden@iconGlyph） |
| R-AP-297 | G-C3-01 | 关联 | 子任务请求需要更贵模型时 | 三卡互斥，同一时刻一张 | — | 卡片计数恒为 1 | 未实现（映射到位但没有该维度的标记）：G-C3-01→internal/ui/approval.go:App.subagentModelBody |
| R-AP-298 | G-C3-01 | 内容 | 带任务标题 | 卡片高度含任务行与模型区块；结构「标题→任务→模型→价格标记→回退说明→按钮」 | — | 排列顺序 = 标题 → 任务 → 模型 → 价格标记 → 回退说明 → 按钮 | 已实现（映射 = G-C3-01→internal/ui/approval.go:App.subagentModelBody；标记 = 文案「模型」） |
| R-AP-299 | G-C3-01 | 样式 | 分别切到浅色与深色主题时 | 同 C1 描边与圆角 | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px | 已实现（映射 = G-C3-01→internal/ui/approval.go:App.subagentModelBody；标记 = Font） |
| R-AP-300 | G-C3-01 | 交互 | 点空白 | 无动作 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-C3-01→internal/ui/approval.go:App.subagentModelBody；标记 = Clicked@App.approvalActions） |
| R-AP-301 | G-C3-01 | 无障碍 | 读屏进入该元素时 | 读出名由 C3-02 承担 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-C3-01→internal/ui/approval.go:App.subagentModelBody；标记 = Label） |
| R-AP-302 | G-C3-01 | 异常 | 答复时断线 | 卡片保留并显示 C3-07 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-C3-01→internal/ui/approval.go:App.subagentModelBody；标记 = Err） |
| R-AP-303 | G-C3-01 | 记忆 | 切到其它会话再切回时 | 按请求重建 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-C3-01→internal/ui/approval.go:App.subagentModelBody |
| R-AP-304 | G-C3-01 | 窄窗 | 窗口收窄到最窄宽度时 | 模型名与供方名折行，不溢出 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 未实现（映射到位但没有该维度的标记）：G-C3-01→internal/ui/approval.go:App.subagentModelBody |
| R-AP-305 | G-C3-02 | 内容 | 未观察（条件）：子代理模型卡在场 | 标题逐字「DimAgent 想要为子任务使用更贵的模型」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalCopySubagentTitle「DimAgent 想要为子任务使用更贵的模型」 |
| R-AP-306 | G-C3-02 | 样式 | 模型卡标题超长时 | 16px 级半粗主前景色，单行（超长裁剪） | — | 判据 = 字号 对话框标题 15.75px；字重 半粗 600 | 已实现（映射 = G-C3-02→internal/ui/approval.go:App.subagentModelBody；标记 = Font） |
| R-AP-307 | G-C3-02 | 关联 | 模型卡标题与价格标记同时在场时 | 标题只说「更贵」，价格判断由 C3-05 标记承担 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalCopySubagentTitle「更贵」 |
| R-AP-308 | G-C3-02 | 无障碍 | 读屏进入该元素时 | 读出整句 | — | 读屏文本 = 「DimAgent 想要为子任务使用更贵的模型」 | 已实现（映射 = G-C3-02→internal/ui/approval.go:App.subagentModelBody；标记 = Label） |
| R-AP-309 | G-C3-02 | 异常 | 文案超宽 | 单行裁剪，不折行挤按钮 | — | 标题 1 行 | 已实现（映射 = G-C3-02→internal/ui/approval.go:App.subagentModelBody；标记 = Err） |
| R-AP-310 | G-C3-02 | 记忆 | 同类请求再次出现 | 文案一致 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-C3-02→internal/ui/approval.go:App.subagentModelBody |
| R-AP-311 | G-C3-02 | 多语言 | 把界面语言切到英文时 | 按语言包切换 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 已实现（映射 = G-C3-02→internal/ui/approval.go:App.subagentModelBody；标记 = 文案「模型」） |
| R-AP-312 | G-C3-02 | 交互 | 点击标题 | 无动作、不可聚焦 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-C3-02→internal/ui/approval.go:App.subagentModelBody；标记 = Clicked@App.approvalActions） |
| R-AP-313 | G-C3-03 | 内容 | 未观察（条件）：请求带 `taskTitle` | 说明行逐字「子任务 · {taskTitle}」，中点分隔 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-C3-03→internal/ui/approval.go:App.subagentModelBody；标记 = 文案「模型」） |
| R-AP-314 | G-C3-03 | 状态 | 无任务标题 | 该行不出现，不留「子任务 ·」空段 | — | 屏上 说明行 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-315 | G-C3-03 | 内容 | 任务标题超长 | 单行 truncate，`title` 给完整值 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C3-03→internal/ui/approval.go:App.subagentModelBody；标记 = 文案「模型」） |
| R-AP-316 | G-C3-03 | 样式 | 子任务行紧贴标题下方出现时 | 14px 级次要色，紧贴标题下方 | — | 判据 = 字号 正文 14px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a；行距 3.5 / 7 / 10.5 / 14px（3.5px 网格） | 已实现（映射 = G-C3-03→internal/ui/approval.go:App.subagentModelBody；标记 = Font） |
| R-AP-317 | G-C3-03 | 无障碍 | 读屏进入该元素时 | 读出任务标题 | — | 读屏文本 = 「子任务 · {taskTitle}」 | 已实现（映射 = G-C3-03→internal/ui/approval.go:App.subagentModelBody；标记 = Label） |
| R-AP-318 | G-C3-03 | 异常 | 标题含换行 | 压成单行显示 | — | 行数恒 1 | 已实现（映射 = G-C3-03→internal/ui/approval.go:App.subagentModelBody；标记 = Err） |
| R-AP-319 | G-C3-03 | 记忆 | 切到其它会话再切回时 | 任务标题逐字不变，仍紧跟卡片标题之后 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM） | 未实现（映射到位但没有该维度的标记）：G-C3-03→internal/ui/approval.go:App.subagentModelBody |
| R-AP-320 | G-C3-03 | 交互 | 点击任务行 | 无动作（不跳转到子代理面板） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-321 | G-C3-04 | 状态 | 未观察（条件）：模型有 provider 元数据 | 区块内一行：20px 提供方图标 + 模型显示名 + 提供方名称；区块为边框 + 浅底 30% + 内边距 12 | — | 判据 = 字号 UI 正文 12.25px；字重 常规 400 / 中粗 500 / 半粗 600 / 粗 700 | 已实现（映射 = G-C3-04→internal/ui/approval.go:App.subagentModelBody；标记 = Hidden@iconGlyph） |
| R-AP-322 | G-C3-04 | 关联 | 模型行与价格标记同时在场时 | 价格标记在区块内该行之下（或行尾），不与模型名混排 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C3-04→internal/ui/approval.go:App.subagentModelBody |
| R-AP-323 | G-C3-04 | 内容 | 无 provider 元数据 | 不显示图标与供方名，只显示模型名，不留空图标位 | — | 屏上 图标与供方名 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-324 | G-C3-04 | 内容 | 模型名超长 | 单行 truncate，悬停出完整名 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C3-04→internal/ui/approval.go:App.subagentModelBody；标记 = 文案「模型」） |
| R-AP-325 | G-C3-04 | 样式 | 分别切到浅色与深色主题时 | 图标 20px；模型名 14px 级主前景；供方名 12px 次要色 | — | 判据 = 控件尺寸 图标 16 / 14 / 12px；字号 正文 14px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C3-04→internal/ui/approval.go:App.subagentModelBody；标记 = Font） |
| R-AP-326 | G-C3-04 | 无障碍 | 读屏进入该元素时 | 读出「模型名，来自 {提供方}」 | — | 读屏文本 = 「模型名，来自 {提供方}」 | 已实现（映射 = G-C3-04→internal/ui/approval.go:App.subagentModelBody；标记 = Label） |
| R-AP-327 | G-C3-04 | 异常 | 图标资源缺失 | 用通用模型图标兜底，无破图 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-C3-04→internal/ui/approval.go:App.subagentModelBody；标记 = Err） |
| R-AP-328 | G-C3-04 | 记忆 | 切到其它会话再切回时 | 模型名与供方名逐字不变，图标仍为同一提供方 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-C3-04→internal/ui/approval.go:App.subagentModelBody |
| R-AP-329 | G-C3-05 | 内容 | 未观察（条件）：模型价格高于当前 | 标记逐字「价格高于当前模型」，12px 琥珀色 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalCopySubagentPricier「价格高于当前模型」 |
| R-AP-330 | G-C3-05 | 状态 | 价格不高于当前（理论上不出现该卡） | 标记不出现；不显示「价格更低」等反义文案 | — | 屏上 标记 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-331 | G-C3-05 | 关联 | 价格标记与回退说明同时在场时 | 一个说贵、一个说拒绝后的回退，两句不重复 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-332 | G-C3-05 | 样式 | 分别切到浅色与深色主题时 | 琥珀色 12px，无底色、无图标 | — | 判据 = 色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a；字号 UI 正文 12.25px | 已实现（映射 = G-C3-05→internal/ui/approval.go:App.subagentModelBody；标记 = Font） |
| R-AP-333 | G-C3-05 | 无障碍 | 读屏进入该元素时 | 读出「价格高于当前模型」 | — | 读屏文本 = 「价格高于当前模型」 | 已实现（逐字）：internal/ui/approval.go:approvalCopySubagentPricier「价格高于当前模型」 |
| R-AP-334 | G-C3-05 | 异常 | 价格未知 | 不显示该标记（不显示「价格未知」） | — | 屏上 标记 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-335 | G-C3-05 | 记忆 | 切到其它会话再切回时 | 标记仍在，颜色与字号不变 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；字号 12.25 / 行高 18 / 字重 400，说明文字 10.5 / 14（TX-TYPE-BODY / TX-TYPE-MICRO） | 未实现（映射到位但没有该维度的标记）：G-C3-05→internal/ui/approval.go:App.subagentModelBody |
| R-AP-336 | G-C3-05 | 交互 | 点击「价格高于当前模型」 | 无动作（不给价格明细浮层） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-C3-05→internal/ui/approval.go:App.subagentModelBody；标记 = Clicked@App.approvalActions） |
| R-AP-337 | G-C3-06 | 内容 | 未观察（条件）：子代理模型卡在场 | 说明逐字「拒绝时子任务将使用当前模型 {model} 继续」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-C3-06→internal/ui/approval.go:subagentFallbackCopy；标记 = 文案「拒绝时子任务将继续使用当前模型」等 3 条） |
| R-AP-338 | G-C3-06 | 内容 | 当前模型名很长 | 折行显示，不截断模型名 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-C3-06→internal/ui/approval.go:subagentFallbackCopy；标记 = 文案「拒绝时子任务将继续使用当前模型」等 3 条） |
| R-AP-339 | G-C3-06 | 关联 | 回退说明与「使用当前模型」按钮同屏时 | 说明与「使用当前模型」按钮说的是同一件事，语义一致 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalCopySubagentReject「使用当前模型」 |
| R-AP-340 | G-C3-06 | 样式 | 价格标记下方渲染回退说明时 | 12px 级次要色，紧随价格标记 | — | 判据 = 字号 UI 正文 12.25px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 未实现（映射到位但没有该维度的标记）：G-C3-06→internal/ui/approval.go:subagentFallbackCopy |
| R-AP-341 | G-C3-06 | 无障碍 | 读屏进入该元素时 | 读出整句 | — | 读屏文本 = 「拒绝时子任务将使用当前模型 {model} 继续」 | 未实现（映射到位但没有该维度的标记）：G-C3-06→internal/ui/approval.go:subagentFallbackCopy |
| R-AP-342 | G-C3-06 | 异常 | 当前模型名未知 | 退为「拒绝时子任务将继续使用当前模型」，不留空占位 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:subagentFallbackCopy「拒绝时子任务将继续使用当前模型」 |
| R-AP-343 | G-C3-06 | 记忆 | 切到其它会话再切回时 | 说明里的当前模型名不变，行位置仍在价格标记之下 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C3-06→internal/ui/approval.go:subagentFallbackCopy |
| R-AP-344 | G-C3-06 | 交互 | 点击回退说明 | 无动作 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-C3-06→internal/ui/approval.go:subagentFallbackCopy |
| R-AP-345 | G-C3-07 | 状态 | 未观察（条件）：点选项后提交失败 | 按钮行上方出现错误文字（14px 危险色） | — | 判据 = 字号 正文 14px；字重 常规 400 / 中粗 500 / 半粗 600 / 粗 700 | 未实现（映射到位但没有该维度的标记）：G-C3-07→internal/ui/approval.go:App.approvalErrorLine |
| R-AP-346 | G-C3-07 | 交互 | 失败后再点 | 先清错误再提交 | — | 错误段恒 1 | 未实现（映射到位但没有该维度的标记）：G-C3-07→internal/ui/approval.go:App.approvalErrorLine |
| R-AP-347 | G-C3-07 | 样式 | 模型卡提交出错时 | 同字号同色 | — | 判据 = 字号 说明文字 10.5px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-C3-07→internal/ui/approval.go:App.approvalErrorLine；标记 = TextColor） |
| R-AP-348 | G-C3-07 | 无障碍 | 读屏进入该元素时 | `role=alert` 播报一次 | — | 播报次数 = 1 | 已实现（映射 = G-C3-07→internal/ui/approval.go:App.approvalErrorLine；标记 = Role） |
| R-AP-349 | G-C3-07 | 异常 | 连续失败 | 同一位置更新 | — | 错误段恒 1 | 已实现（映射 = G-C3-07→internal/ui/approval.go:App.approvalErrorLine；标记 = Err） |
| R-AP-350 | G-C3-07 | 记忆 | 切到其它会话再切回时 | 请求仍在则错误保留 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-C3-07→internal/ui/approval.go:App.approvalErrorLine |
| R-AP-351 | G-C3-07 | 关联 | 模型卡提交出错、按钮组仍可点时 | 失败后按钮恢复可点，与错误同屏 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-C3-07→internal/ui/approval.go:App.approvalErrorLine |
| R-AP-352 | G-C3-07 | 窄窗 | 窗口收窄到最窄宽度时 | 错误折行，按钮行不被挤出 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 未实现（映射到位但没有该维度的标记）：G-C3-07→internal/ui/approval.go:App.approvalErrorLine |
| R-AP-353 | G-C3-08 | 状态 | 未观察（条件）：拒绝类选项（`reject_once`/`reject_always`） | 次按钮「使用当前模型」，位于按钮组左 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalCopySubagentReject「使用当前模型」 |
| R-AP-354 | G-C3-08 | 交互 | 鼠标点击该元素时 | 子任务改用当前模型继续；卡片消失 | — | 屏上 卡片 计数 = 0 | 未实现（映射到位但没有该维度的标记）：G-C3-08→internal/ui/approval.go:approvalActionsFor |
| R-AP-355 | G-C3-08 | 关联 | 回退说明在场时 | 文案与说明行一致 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-C3-08→internal/ui/approval.go:approvalActionsFor |
| R-AP-356 | G-C3-08 | 样式 | 次按钮在屏上时 | 无底色、1px 描边、高 28 | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C3-08→internal/ui/approval.go:approvalActionsFor；标记 = Type@approvalSandboxLongTermCopy） |
| R-AP-357 | G-C3-08 | 无障碍 | 读屏进入该元素时 | 读作「使用当前模型」 | — | 读屏文本 = 「使用当前模型」 | 已实现（逐字）：internal/ui/approval.go:approvalCopySubagentReject「使用当前模型」 |
| R-AP-358 | G-C3-08 | 键盘 | 按 Enter 时 | 子任务改用当前模型继续，卡片消失 | — | 屏上 卡片 计数 = 0 | 未实现（映射到位但没有该维度的标记）：G-C3-08→internal/ui/approval.go:approvalActionsFor |
| R-AP-359 | G-C3-08 | 异常 | 提交失败 | 卡片保留并显示 C3-07 | — | 卡片 计数 = 1；错误文字 计数 = 1 | 未实现（映射到位但没有该维度的标记）：G-C3-08→internal/ui/approval.go:approvalActionsFor |
| R-AP-360 | G-C3-08 | 记忆 | 同类请求再次出现 | 按钮默认可用 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-C3-08→internal/ui/approval.go:approvalActionsFor |
| R-AP-361 | G-C3-09 | 状态 | 未观察（条件）：允许类选项 | 主按钮「本会话允许」，实心强调色 | — | 判据 = 底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d；其余 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalCopySubagentAllow「本会话允许」 |
| R-AP-362 | G-C3-09 | 交互 | 鼠标点击该元素时 | 本次会话内该子任务用贵模型；卡片消失 | — | 屏上 卡片 计数 = 0 | 未实现（映射到位但没有该维度的标记）：G-C3-09→internal/ui/approval.go:approvalActionsFor |
| R-AP-363 | G-C3-09 | 关联 | 允许与拒绝按钮同屏时 | 一个允许一个拒绝，位置与配色不同 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C3-09→internal/ui/approval.go:approvalActionsFor |
| R-AP-364 | G-C3-09 | 样式 | 模型卡主按钮与工具卡主按钮先后出现时 | 同尺寸、同底色、白字 | — | 判据 = 底色 主色 明 #339751 / 暗 #63cb7b；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C3-09→internal/ui/approval.go:approvalActionsFor；标记 = Type@approvalSandboxLongTermCopy） |
| R-AP-365 | G-C3-09 | 无障碍 | 读屏进入该元素时 | 读出作用域「本会话」 | — | 读屏文本 = 「本会话」 | 已实现（逐字）：internal/ui/approval.go:approvalCopySubagentAllow「本会话」 |
| R-AP-366 | G-C3-09 | 键盘 | 按 Enter 时 | 本会话允许该贵模型，卡片消失 | — | 屏上 卡片 计数 = 0 | 未实现（映射到位但没有该维度的标记）：G-C3-09→internal/ui/approval.go:approvalActionsFor |
| R-AP-367 | G-C3-09 | 异常 | 提交失败 | 卡片保留并显示 C3-07 | — | 卡片 计数 = 1；错误文字 计数 = 1 | 未实现（映射到位但没有该维度的标记）：G-C3-09→internal/ui/approval.go:approvalActionsFor |
| R-AP-368 | G-C3-09 | 记忆 | 本会话内同类请求 | 不再出卡（会话内已允许） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-C3-09→internal/ui/approval.go:approvalActionsFor |

### 1.4 审批输入区（G-C4-01–G-C4-25）

已观察：C4-01、C4-02、C4-03、C4-04 的命令渲染器分支（A 与 C39 两份）。其余九类渲染器未观察（条件：对应工具的权限请求）。

| 需求 ID | 元素 ID | 维度 | 场景 | 期望 | 异常 | 验收 | 状态 |
|---|---|---|---|---|---|---|---|
| R-AP-369 | G-C4-01 | 状态 | A 态：按工具名与入参命中命令渲染器 | 输入区块在场：起点 387,552，宽 613，最大高 240；边框 1px + 卡片底色、圆角小 | — | 区块 起点 387,552、宽 613、最大高 240 | 未实现（映射到位但没有该维度的标记）：G-C4-01→internal/ui/approval_panels.go:approvalInputPanel；测试提及 internal/ui/reconcile_scope_test.go:包级 |
| R-AP-370 | G-C4-01 | 状态 | 带超时的请求到达时 | 同区块左移 356、变宽（657）；内部三段（标签/命令行/参数行）顺序不变 | — | 区块 x=31、宽 657（A 态 387/613，差 356） | 未实现（映射到位但没有该维度的标记）：G-C4-01→internal/ui/approval_panels.go:approvalInputPanel；测试提及 internal/ui/reconcile_scope_test.go:包级 |
| R-AP-371 | G-C4-01 | 内容 | 命令很长（多行） | 区块高度增长到 240 封顶后内部滚动；标签行随内容滚动或不滚动（按实现）但始终可达 | — | 区块高 ≤240 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-372 | G-C4-01 | 交互 | 鼠标滚轮在元素上滚动时 | 区块内部滚动，卡片位置不动 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-01→internal/ui/approval_panels.go:approvalInputPanel；标记 = Key） |
| R-AP-373 | G-C4-01 | 交互 | 选中屏上文本后 | 命令行与参数行可选中复制 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-C4-01→internal/ui/approval_panels.go:approvalInputPanel；标记 = Key） |
| R-AP-374 | G-C4-01 | 关联 | 同一请求换用不同工具、输入区整块切换时 | 同一区块按工具切换内容（命令 / 文件 / 搜索 / 网页 / 技能 / 子代理 / 浏览器 / CUA / 等待 / 通用），区块本身不变形 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-01→internal/ui/approval_panels.go:approvalInputPanel；测试提及 internal/ui/reconcile_scope_test.go:包级 |
| R-AP-375 | G-C4-01 | 样式 | 分别切到浅色与深色主题时 | 底色比卡片底略深或略浅一档，形成「输入框」感；描边 1px | — | 判据 = 底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d；描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px；内边距 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28）/ 卡片外框 4.5；档位 3.5·7·10.5·14·21·28px | 已实现（逐字）：internal/ui/search_chat.go:chatSearchLabel「输入框」 |
| R-AP-376 | G-C4-01 | 无障碍 | 读屏进入该元素时 | 区块只读展示，不进 Tab 顺序；内容可整体读取 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-C4-01→internal/ui/approval_panels.go:approvalInputPanel；标记 = Label） |
| R-AP-377 | G-C4-01 | 异常 | 入参无法解析 | 退回通用渲染器（C4-25），不出现空区块 | — | 屏上 空区块 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-378 | G-C4-01 | 记忆 | 切到其它会话再切回时 | 区块按请求重建，滚动回顶 | — | 判据 = 滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-01→internal/ui/approval_panels.go:approvalInputPanel；标记 = persist） |
| R-AP-379 | G-C4-01 | 窄窗 | 窗口收窄到最窄宽度时 | 区块宽随卡片收缩，命令行横向滚动 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-01→internal/ui/approval_panels.go:approvalInputPanel；测试提及 internal/ui/reconcile_scope_test.go:包级 |
| R-AP-380 | G-C4-01 | 高DPI | 在 DPR=2 的屏幕上时 | 等宽字符与描边在 2× 下清晰 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-01→internal/ui/approval_panels.go:approvalInputPanel；测试提及 internal/ui/reconcile_scope_test.go:包级 |
| R-AP-381 | G-C4-01 | 性能 | 200 行命令 / 补丁 | 区块渲染 ≤200ms，滚动流畅 | — | 判据 = 动效 布局 250ms / 颜色 150ms / 折叠 300ms；缓动 cubic-bezier(0.16, 1, 0.3, 1) / cubic-bezier(0.4, 0, 0.2, 1)；其余 = 滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-01→internal/ui/approval_panels.go:approvalInputPanel；测试提及 internal/ui/reconcile_scope_test.go:包级 |
| R-AP-382 | G-C4-01 | 内容 | 旧版纯文本入参 | 以 `{input: 文本}` 形态进通用渲染器，不显示原始 JSON 转义 | — | 屏上 原始 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-383 | G-C4-01 | 状态 | 区块在场但无任何字段 | 标签行仍显示渲染器名，下方给通用渲染的原始入参 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-01→internal/ui/approval_panels.go:approvalInputPanel；测试提及 internal/ui/reconcile_scope_test.go:包级 |
| R-AP-384 | G-C4-01 | 关联 | 输入区内容高度增长时 | 区块与按钮行不重叠；区块高增长时按钮行下移而非覆盖 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-C4-01→internal/ui/approval_panels.go:approvalInputPanel；测试提及 internal/ui/reconcile_scope_test.go:包级 |
| R-AP-385 | G-C4-02 | 内容 | A 态：命令渲染器 | 顶部标签逐字「输入」（387,552 613×27），11px 次要色 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/app/contextusage.go:App.ReadContextUsage「输入」 |
| R-AP-386 | G-C4-02 | 内容 | 搜索 / 网页搜索 / 行列表 / 通用渲染器 | 同一位置也用「输入」；不给每类造新标签 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/app/contextusage.go:App.ReadContextUsage「输入」 |
| R-AP-387 | G-C4-02 | 状态 | 区块内部滚动 | 标签行随内容滚出（不吸顶）；再次滚回可见 | — | 判据 = 动效 布局 250ms / 颜色 150ms / 折叠 300ms；缓动 cubic-bezier(0.16, 1, 0.3, 1) / cubic-bezier(0.4, 0, 0.2, 1)；其余 = 滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0 | 未实现（映射到位但没有该维度的标记）：G-C4-02→internal/ui/approval_panels.go:approvalPanelLabel |
| R-AP-388 | G-C4-02 | 样式 | 分别切到浅色与深色主题时 | 11px 级、次要色、左对齐；与命令行之间距 2–4 | — | 判据 = 字号 键帽 11px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a；间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px | 已实现（映射 = G-C4-02→internal/ui/approval_panels.go:approvalPanelLabel；标记 = TextColor） |
| R-AP-389 | G-C4-02 | 无障碍 | 读屏进入该元素时 | 作为区块的标签读出，先于命令行 | — | 读屏文本 = 「输入」 | 已实现（映射 = G-C4-02→internal/ui/approval_panels.go:approvalPanelLabel；标记 = Label） |
| R-AP-390 | G-C4-02 | 异常 | 渲染器无标签（如文件渲染器用文件名顶替） | 该渲染器用文件名或工具名顶替，不留空标签行 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-C4-02→internal/ui/approval_panels.go:approvalPanelLabel |
| R-AP-391 | G-C4-02 | 记忆 | 切到其它会话再切回时 | 标签仍为「输入」，仍在区块顶端同一位置 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（逐字）：internal/app/contextusage.go:App.ReadContextUsage「输入」 |
| R-AP-392 | G-C4-02 | 高DPI | 在 DPR=2 的屏幕上时 | 11px 在 2× 下仍可读 | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-02→internal/ui/approval_panels.go:approvalPanelLabel |
| R-AP-393 | G-C4-03 | 内容 | A 态：命令字段存在 | 等宽代码行「$ ls -la」：`$` 为灰色前缀（397,581 6×14），命令体 397,579 592×18 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-03→internal/ui/approval_panels.go:approvalCommandPanel；标记 = 文案「输入」） |
| R-AP-394 | G-C4-03 | 内容 | 带超时的请求到达时 | 「$ sleep 25」，x=42、宽 636 | — | 屏上文案为 `sleep 25` | 已实现（映射 = G-C4-03→internal/ui/approval_panels.go:approvalCommandPanel；标记 = 文案「输入」） |
| R-AP-395 | G-C4-03 | 内容 | 命令含多行 | 按行显示，保留换行，不压成一行 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-C4-03→internal/ui/approval_panels.go:approvalCommandPanel；标记 = 文案「输入」） |
| R-AP-396 | G-C4-03 | 内容 | 命令含超长单行 | 折行或横向滚动（按实现），不裁掉参数 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-C4-03→internal/ui/approval_panels.go:approvalCommandPanel；标记 = 文案「输入」） |
| R-AP-397 | G-C4-03 | 内容 | 命令含中文路径 | 等宽显示中文，字符不错位 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-03→internal/ui/approval_panels.go:approvalCommandPanel；标记 = 文案「输入」） |
| R-AP-398 | G-C4-03 | 交互 | 选中屏上文本后复制时 | 复制内容不含 `$` 前缀（只复制命令体） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-03→internal/ui/approval_panels.go:approvalCommandPanel；标记 = Key） |
| R-AP-399 | G-C4-03 | 样式 | 分别切到浅色与深色主题时 | 等宽 14px 级；`$` 灰色、命令体主前景色 | — | 判据 = 字号 正文 14px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a；其余 = 字号 12.25 / 行高 18 / 字重 400，说明文字 10.5 / 14（TX-TYPE-BODY / TX-TYPE-MICRO）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-03→internal/ui/approval_panels.go:approvalPanelBody；标记 = TextColor） |
| R-AP-400 | G-C4-03 | 无障碍 | 读屏进入该元素时 | 读出命令文本，念「美元符号」或跳过前缀 | — | 读屏文本 = 「美元符号」 | 已实现（映射 = G-C4-03→internal/ui/approval_panels.go:approvalCommandPanel；标记 = Label） |
| R-AP-401 | G-C4-03 | 异常 | 命令为空 | 该行不出现（不显示孤立的 `$`） | — | 屏上 孤立的 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-402 | G-C4-03 | 记忆 | 切到其它会话再切回时 | 命令体逐字不变，`$` 前缀与行首缩进不变 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-03→internal/ui/approval_panels.go:approvalCommandPanel |
| R-AP-403 | G-C4-03 | 高DPI | 在 DPR=2 的屏幕上时 | 等宽字符列宽一致、不错位 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-03→internal/ui/approval_panels.go:approvalCommandPanel |
| R-AP-404 | G-C4-03 | 多语言 | 英文 / 中文界面 | 命令原文不翻译（属用户数据） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-03→internal/ui/approval_panels.go:approvalCommandPanel；标记 = 文案「输入」） |
| R-AP-405 | G-C4-03 | 性能 | 200 行命令 | 渲染 ≤150ms | — | 判据 = 动效 布局 250ms / 颜色 150ms / 折叠 300ms；缓动 cubic-bezier(0.16, 1, 0.3, 1) / cubic-bezier(0.4, 0, 0.2, 1) | 未实现（映射到位但没有该维度的标记）：G-C4-03→internal/ui/approval_panels.go:approvalCommandPanel |
| R-AP-406 | G-C4-03 | 窄窗 | 窗口收窄到最窄宽度时 | 长命令横向滚动或折行，`$` 不单独成行 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-C4-03→internal/ui/approval_panels.go:approvalPanelBody；标记 = narrow@approvalCodeBlock） |
| R-AP-407 | G-C4-04 | 内容 | A 态：`cwd` 字段存在 | 参数行「cwd: {工作区路径}」（397,596 592×18），缩进两格、70% 不透明 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED） | 已实现（映射 = G-C4-04→internal/ui/approval_panels.go:approvalCommandPanel；标记 = 文案「输入」） |
| R-AP-408 | G-C4-04 | 内容 | C39 态：`timeoutMs` 存在 | 参数行「timeoutMs: 60 s」（42,596 636×18） | — | 屏上文案逐字含 `60 s` | 已实现（映射 = G-C4-04→internal/ui/approval_panels.go:approvalCommandPanel；标记 = 文案「输入」） |
| R-AP-409 | G-C4-04 | 内容 | `runInBackground` 存在 | 参数行「runInBackground: true / false」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-04→internal/ui/approval_panels.go:approvalCommandPanel；标记 = 文案「输入」） |
| R-AP-410 | G-C4-04 | 状态 | 多个字段同时存在 | 每个字段一行，按固定顺序（cwd → timeoutMs → runInBackground） | — | 排列顺序 = cwd → timeoutMs → runInBackground | 未实现（映射到位但没有该维度的标记）：G-C4-04→internal/ui/approval_panels.go:approvalCommandPanel |
| R-AP-411 | G-C4-04 | 内容 | 无任何参数字段 | 参数行组整体不出现，不留空行 | — | 屏上 等宽参数行组 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-412 | G-C4-04 | 样式 | 命令行与参数行同屏时 | 同为等宽、字号略小或同号；70% 不透明使其比命令弱一档 | — | 判据 = 字号 代码 13px；其余 = 色值取 token：主色 #349651/#63cb7b、正文 #0a0a0a/#ececec、次文 #737373/#8a8a8a、边框 #e5e5e5/#2e2e2e、警示 #e17100/#ffb900，偏差 0（CL-ACCENT / CL-TEXT / CL-TEXT-2 / CL-BORDER / CL-WARN）；字号 12.25 / 行高 18 / 字重 400，说明文字 10.5 / 14（TX-TYPE-BODY / TX-TYPE-MICRO） | 未实现（映射到位但没有该维度的标记）：G-C4-04→internal/ui/approval_panels.go:approvalCommandPanel |
| R-AP-413 | G-C4-04 | 无障碍 | 读屏进入该元素时 | 逐行读出键与值 | — | 播报顺序 = cwd: {cwd} → timeoutMs: {值} → runInBackground: {；每行先键后值 | 已实现（映射 = G-C4-04→internal/ui/approval_panels.go:approvalCommandPanel；标记 = Label） |
| R-AP-414 | G-C4-04 | 内容 | 超时值单位 | 值为秒时带单位 `s`（如 `60 s`），不带毫秒原值 | — | 屏上无 `60000` | 已实现（映射 = G-C4-04→internal/ui/approval_panels.go:approvalCommandPanel；标记 = 文案「输入」） |
| R-AP-415 | G-C4-04 | 关联 | `cwd` 为当前工作区 | 仍显示该行（不省略），路径用完整值 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-04→internal/ui/approval_panels.go:approvalCommandPanel |
| R-AP-416 | G-C4-04 | 异常 | 字段值类型异常（对象/数组） | 序列化为单行文本，不显示 `[object Object]` | — | 屏上 等宽参数行组 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-417 | G-C4-04 | 记忆 | 切到其它会话再切回时 | 各键值逐字不变，行序不变（cwd → timeoutMs → runInBackground） | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-04→internal/ui/approval_panels.go:approvalCommandPanel |
| R-AP-418 | G-C4-04 | 高DPI | 在 DPR=2 的屏幕上时 | 等宽键值对齐 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-04→internal/ui/approval_panels.go:approvalCommandPanel |
| R-AP-419 | G-C4-05 | 状态 | 未观察（条件）：`read`/`write`/`edit`/`apply_patch` 的权限请求 | 顶部标签为文件名（超长时截断），无文件名时退为工具名 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-05→internal/ui/approval_panels.go:approvalFilePanel |
| R-AP-420 | G-C4-05 | 内容 | `patch.diff` 场景 | 标签显示 `patch.diff` 或工具名，不显示空标签 | — | 屏上 空标签 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-421 | G-C4-05 | 样式 | 「输入」标签与文件名行同屏时 | 同 11px 次要色标签位 | — | 判据 = 字号 键帽 11px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-C4-05→internal/ui/approval_panels.go:approvalPanelLabel；标记 = TextColor） |
| R-AP-422 | G-C4-05 | 无障碍 | 读屏进入该元素时 | 读出文件名或工具名 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-C4-05→internal/ui/approval_panels.go:approvalFilePanel；标记 = Label） |
| R-AP-423 | G-C4-05 | 异常 | 文件名超长 | truncate，悬停出完整路径 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-05→internal/ui/approval_panels.go:approvalFilePanel |
| R-AP-424 | G-C4-05 | 记忆 | 切到其它会话再切回时 | 标签仍为同一文件名/工具名，截断位置相同 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-05→internal/ui/approval_panels.go:approvalFilePanel |
| R-AP-425 | G-C4-05 | 窄窗 | 窗口收窄到最窄宽度时 | 标签截断但不为空 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-05→internal/ui/approval_panels.go:approvalFilePanel |
| R-AP-426 | G-C4-05 | 交互 | 点击标签 | 无动作（不是打开文件的入口） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-C4-05→internal/ui/approval_panels.go:approvalFilePanel；标记 = Key） |
| R-AP-427 | G-C4-06 | 状态 | 未观察（条件）：`edit` 请求带 `oldText` 与 `newText` | 双栏代码块：左旧右新，无行号 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-06→internal/ui/approval_panels.go:approvalDiffBlock；标记 = Hidden@approvalDiffColumn） |
| R-AP-428 | G-C4-06 | 内容 | 新旧文本长度差异大 | 两栏等高（按较高一侧），短侧留空不拉伸 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 已实现（映射 = G-C4-06→internal/ui/approval_panels.go:approvalDiffBlock；标记 = 文案「已截断 %d 行，完整内容见会话记录」等 2 条@approvalDiffColumn） |
| R-AP-429 | G-C4-06 | 内容 | 只有一侧有值 | 只显示有值的一栏，不留空栏 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-C4-06→internal/ui/approval_panels.go:approvalDiffBlock；标记 = 文案「已截断 %d 行，完整内容见会话记录」等 2 条@approvalDiffColumn） |
| R-AP-430 | G-C4-06 | 交互 | 选中屏上文本后复制时 | 可分别选中两栏文本 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-06→internal/ui/approval_panels.go:approvalDiffBlock；测试提及 internal/ui/reconcile_scope_test.go:包级 |
| R-AP-431 | G-C4-06 | 样式 | 分别切到浅色与深色主题时 | 等宽 12px；旧栏偏红、新栏偏绿底或左侧色条 | — | 判据 = 底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d；字号 UI 正文 12.25px | 已实现（映射 = G-C4-06→internal/ui/approval_panels.go:approvalDiffBlock；标记 = Width） |
| R-AP-432 | G-C4-06 | 无障碍 | 读屏进入该元素时 | 先读旧文本再读新文本，并说明两栏关系 | — | 播报顺序 = 旧文本 → 新文本 | 已实现（映射 = G-C4-06→internal/ui/approval_panels.go:approvalDiffBlock；标记 = Label@approvalDiffColumn） |
| R-AP-433 | G-C4-06 | 异常 | 文本超长 | 各栏内部纵向滚动，不撑破卡片 | — | 判据 = 滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-06→internal/ui/approval_panels.go:approvalDiffBlock；标记 = reason@approvalDiffColumn） |
| R-AP-434 | G-C4-06 | 记忆 | 切到其它会话再切回时 | 两栏文本逐字不变，左右栏顺序不变 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM） | 未实现（映射到位但没有该维度的标记）：G-C4-06→internal/ui/approval_panels.go:approvalDiffBlock；测试提及 internal/ui/reconcile_scope_test.go:包级 |
| R-AP-435 | G-C4-07 | 状态 | 未观察（条件）：`write` 请求带 `content` | 单栏代码块，带行号 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-07→internal/ui/approval_panels.go:approvalCodeBlock |
| R-AP-436 | G-C4-07 | 内容 | 文件为空（新建空文件） | 显示空块而非隐藏块（表示将写入空内容） | — | 屏上 代码块 计数 = 0 | 已实现（映射 = G-C4-07→internal/ui/approval_panels.go:approvalFilePanel；标记 = 文案「输入」） |
| R-AP-437 | G-C4-07 | 内容 | 内容超长 | 块内滚动，最大高受 C4-01 约束 | — | 判据 = 滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-07→internal/ui/approval_panels.go:approvalFilePanel；标记 = 文案「输入」） |
| R-AP-438 | G-C4-07 | 交互 | 选中屏上文本后复制时 | 可复制，行号不进剪贴板 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-C4-07→internal/ui/approval_panels.go:approvalCodeBlock；标记 = Pressed） |
| R-AP-439 | G-C4-07 | 样式 | 分别切到浅色与深色主题时 | 等宽 12px；行号列次要色、右对齐 | — | 判据 = 字号 代码 13px；其余 = 色值取 token：主色 #349651/#63cb7b、正文 #0a0a0a/#ececec、次文 #737373/#8a8a8a、边框 #e5e5e5/#2e2e2e、警示 #e17100/#ffb900，偏差 0（CL-ACCENT / CL-TEXT / CL-TEXT-2 / CL-BORDER / CL-WARN） | 已实现（映射 = G-C4-07→internal/ui/approval_panels.go:approvalCodeBlock；标记 = Height） |
| R-AP-440 | G-C4-07 | 无障碍 | 读屏进入该元素时 | 读出内容，不读行号 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-C4-07→internal/ui/approval_panels.go:approvalFilePanel；标记 = Label） |
| R-AP-441 | G-C4-07 | 异常 | 内容含二进制 | 给不可展示提示，不渲染乱码 | — | 屏上 乱码 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-442 | G-C4-07 | 记忆 | 切到其它会话再切回时 | 内容逐字不变，行号列仍在左侧且行号重新从 1 开始 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-07→internal/ui/approval_panels.go:approvalCodeBlock |
| R-AP-443 | G-C4-08 | 状态 | 未观察（条件）：`apply_patch`/`patch` 请求带 `patch` | 单栏代码块（无行号）显示补丁文本 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-C4-08→internal/ui/approval_panels.go:approvalCodeBlock |
| R-AP-444 | G-C4-08 | 内容 | 补丁含多文件段 | 按段落照常显示，不合并 | — | 屏上 代码块 计数 = 1 | 已实现（映射 = G-C4-08→internal/ui/approval_panels.go:approvalFilePanel；标记 = 文案「输入」） |
| R-AP-445 | G-C4-08 | 样式 | 写入内容块与补丁文本块先后出现时 | 同为无底色代码块，但无行号列 | — | 判据 = 底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d | 已实现（映射 = G-C4-08→internal/ui/approval_panels.go:approvalCodeBlock；标记 = Height） |
| R-AP-446 | G-C4-08 | 无障碍 | 读屏进入该元素时 | 读出补丁文本 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-C4-08→internal/ui/approval_panels.go:approvalFilePanel；标记 = Label） |
| R-AP-447 | G-C4-08 | 异常 | 补丁格式非法 | 原文照显（不解析失败就丢弃），可复制 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-C4-08→internal/ui/approval_panels.go:approvalCodeBlock |
| R-AP-448 | G-C4-08 | 记忆 | 切到其它会话再切回时 | 补丁文本逐字不变，仍无行号列 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-C4-08→internal/ui/approval_panels.go:approvalCodeBlock |
| R-AP-449 | G-C4-08 | 交互 | 选中补丁文本 | 可选中复制，不改变区块布局 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-450 | G-C4-08 | 窄窗 | 窗口收窄到最窄宽度时 | 补丁块内横向滚动，不撑破卡片 | — | 判据 = 滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-08→internal/ui/approval_panels.go:approvalCodeBlock；标记 = narrow） |
| R-AP-451 | G-C4-09 | 状态 | 未观察（条件）：文件类请求带 `replaceAll` / `offset` / `limit` / `encoding` | 等宽参数行组，逐字段一行 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-09→internal/ui/approval_panels.go:approvalPanelBody |
| R-AP-452 | G-C4-09 | 内容 | `replaceAll: true` | 屏上文案为「replaceAll: true」，布尔用英文小写 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-09→internal/ui/approval_panels.go:approvalPanelBody；标记 = 文案「图片预览」等 2 条@approvalImageBlock） |
| R-AP-453 | G-C4-09 | 内容 | 无这些字段 | 参数行组不出现 | — | 屏上 等宽参数行组 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-454 | G-C4-09 | 样式 | 命令行与参数行同屏时 | 同 70% 不透明等宽参数行样式 | — | 判据 = 字号 代码 13px；其余 = 色值取 token：主色 #349651/#63cb7b、正文 #0a0a0a/#ececec、次文 #737373/#8a8a8a、边框 #e5e5e5/#2e2e2e、警示 #e17100/#ffb900，偏差 0（CL-ACCENT / CL-TEXT / CL-TEXT-2 / CL-BORDER / CL-WARN） | 已实现（映射 = G-C4-09→internal/ui/approval_panels.go:approvalPanelBody；标记 = TextColor） |
| R-AP-455 | G-C4-09 | 无障碍 | 读屏进入该元素时 | 逐行读出键值 | — | 播报顺序 = replaceAll: {值} → offset/limit/encod；每行先键后值 | 已实现（映射 = G-C4-09→internal/ui/approval_panels.go:approvalPanelBody；标记 = Label@approvalImageBlock） |
| R-AP-456 | G-C4-09 | 异常 | 值为空串 | 显示空值（`encoding: `），不删行 | — | 空值行个数 = 1，行内文案为「—」 | 已实现（映射 = G-C4-09→internal/ui/approval_panels.go:approvalPanelBody；标记 = err@approvalImageBlock） |
| R-AP-457 | G-C4-09 | 记忆 | 切到其它会话再切回时 | 各字段逐字不变，行序不变 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-09→internal/ui/approval_panels.go:approvalPanelBody |
| R-AP-458 | G-C4-09 | 交互 | 鼠标点击参数行时 | 无动作、不可编辑（值来自请求原文） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-09→internal/ui/approval_panels.go:approvalPanelBody；标记 = Pressed@approvalCodeBlock） |
| R-AP-459 | G-C4-10 | 状态 | 未观察（条件）：`grep`/`glob` 的权限请求 | 等宽主值行按渲染器显示「pattern: {值}」「query: {值}」或「path: {值}」，90% 不透明 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-C4-10→internal/ui/approval_panels.go:approvalPanelBody |
| R-AP-460 | G-C4-10 | 内容 | 键名选择 | `grep` 用 pattern、`glob` 用 pattern、列举用 path；不并列多键 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-10→internal/ui/approval_panels.go:approvalPanelBody；标记 = 文案「图片预览」等 2 条@approvalImageBlock） |
| R-AP-461 | G-C4-10 | 样式 | 主值行与参数行同屏时 | 主值行比参数行更亮（90% vs 70%） | — | 判据 = 色值取 token：主色 #349651/#63cb7b、正文 #0a0a0a/#ececec、次文 #737373/#8a8a8a、边框 #e5e5e5/#2e2e2e、警示 #e17100/#ffb900，偏差 0（CL-ACCENT / CL-TEXT / CL-TEXT-2 / CL-BORDER / CL-WARN）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-C4-10→internal/ui/approval_panels.go:approvalPanelBody；标记 = TextColor） |
| R-AP-462 | G-C4-10 | 无障碍 | 读屏进入该元素时 | 读出键与值 | — | 播报顺序 = pattern: {值} → query: {值} → path: {值}；每行先键后值 | 已实现（映射 = G-C4-10→internal/ui/approval_panels.go:approvalPanelBody；标记 = Label@approvalImageBlock） |
| R-AP-463 | G-C4-10 | 异常 | 值为空 | 行仍显示键名与空值 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-10→internal/ui/approval_panels.go:approvalPanelBody；标记 = err@approvalImageBlock） |
| R-AP-464 | G-C4-10 | 记忆 | 切到其它会话再切回时 | 键名与值逐字不变（不因重开换成另一键名） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-10→internal/ui/approval_panels.go:approvalPanelBody |
| R-AP-465 | G-C4-10 | 交互 | 点击主值行 | 无动作；不提供「按此条件重跑」入口 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-C4-10→internal/ui/approval_panels.go:approvalPanelBody；标记 = Pressed@approvalCodeBlock） |
| R-AP-466 | G-C4-10 | 窄窗 | 窗口收窄到最窄宽度时 | 长 pattern 折行，键名不被拆开 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-C4-10→internal/ui/approval_panels.go:approvalPanelBody；标记 = narrow@approvalCodeBlock） |
| R-AP-467 | G-C4-11 | 状态 | 未观察（条件）：检索请求带 `path` 且主值不是路径 | 次要行「path: {值}」，70% 不透明，紧跟主值行 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED） | 未实现（映射到位但没有该维度的标记）：G-C4-11→internal/ui/approval_panels.go:approvalPanelBody |
| R-AP-468 | G-C4-11 | 状态 | 主值本身就是路径（列举类） | 不重复出现本行（避免两行同值） | — | 屏上 等宽次要行 计数 = 1（不与主值行重复） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-469 | G-C4-11 | 样式 | 路径行与主值行同屏时 | 同尺寸等宽，亮度低一档 | — | 判据 = 色值取 token：主色 #349651/#63cb7b、正文 #0a0a0a/#ececec、次文 #737373/#8a8a8a、边框 #e5e5e5/#2e2e2e、警示 #e17100/#ffb900，偏差 0（CL-ACCENT / CL-TEXT / CL-TEXT-2 / CL-BORDER / CL-WARN）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-11→internal/ui/approval_panels.go:approvalPanelBody；标记 = TextColor） |
| R-AP-470 | G-C4-11 | 无障碍 | 读屏进入该元素时 | 读出路径 | — | 读屏文本 = 「path: {值}」 | 已实现（映射 = G-C4-11→internal/ui/approval_panels.go:approvalPanelBody；标记 = Label@approvalImageBlock） |
| R-AP-471 | G-C4-11 | 异常 | 路径超长 | 折行或横向滚动，不截断参数 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-C4-11→internal/ui/approval_panels.go:approvalPanelBody；标记 = err@approvalImageBlock） |
| R-AP-472 | G-C4-11 | 记忆 | 切到其它会话再切回时 | path 值逐字不变，仍在主值行之下 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-11→internal/ui/approval_panels.go:approvalPanelBody |
| R-AP-473 | G-C4-11 | 交互 | 点击 path 行 | 无动作 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-11→internal/ui/approval_panels.go:approvalPanelBody；标记 = Pressed@approvalCodeBlock） |
| R-AP-474 | G-C4-11 | 窄窗 | 窗口收窄到最窄宽度时 | 路径折行，缩进保持 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-C4-11→internal/ui/approval_panels.go:approvalPanelBody；标记 = narrow@approvalCodeBlock） |
| R-AP-475 | G-C4-12 | 状态 | 未观察（条件）：检索请求带附加参数 | 等宽参数行组逐字段列出（include/glob/outputMode/before/after/context/-B/-A/-C/lineNumbers/-n/literal/limit） | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-12→internal/ui/approval_panels.go:approvalPanelBody |
| R-AP-476 | G-C4-12 | 内容 | 同一语义有两种键（context 与 -C） | 按请求实际键显示，不合并、不改名 | — | 屏上 等宽参数行组 计数 = 1 | 已实现（映射 = G-C4-12→internal/ui/approval_panels.go:approvalPanelBody；标记 = 文案「图片预览」等 2 条@approvalImageBlock） |
| R-AP-477 | G-C4-12 | 内容 | 布尔参数 | 显示 `true` / `false` 小写 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-12→internal/ui/approval_panels.go:approvalPanelBody；标记 = 文案「图片预览」等 2 条@approvalImageBlock） |
| R-AP-478 | G-C4-12 | 样式 | 命令行与参数行组同屏时 | 同 70% 不透明参数行 | — | 判据 = 字号 代码 13px；其余 = 色值取 token：主色 #349651/#63cb7b、正文 #0a0a0a/#ececec、次文 #737373/#8a8a8a、边框 #e5e5e5/#2e2e2e、警示 #e17100/#ffb900，偏差 0（CL-ACCENT / CL-TEXT / CL-TEXT-2 / CL-BORDER / CL-WARN） | 已实现（映射 = G-C4-12→internal/ui/approval_panels.go:approvalPanelBody；标记 = TextColor） |
| R-AP-479 | G-C4-12 | 无障碍 | 读屏进入该元素时 | 逐行读出 | — | 播报顺序 = include → glob → outputMode → beforeContext → afterContext → context → -B/-A/-C → lineNumbers/-n → literal → limit；每行先键后值 | 已实现（映射 = G-C4-12→internal/ui/approval_panels.go:approvalPanelBody；标记 = Label@approvalImageBlock） |
| R-AP-480 | G-C4-12 | 异常 | 参数很多（10+） | 区块内滚动，不撑破卡片 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0 | 已实现（映射 = G-C4-12→internal/ui/approval_panels.go:approvalPanelBody；标记 = err@approvalImageBlock） |
| R-AP-481 | G-C4-12 | 记忆 | 切到其它会话再切回时 | 各字段逐字不变，键名大小写不变 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-12→internal/ui/approval_panels.go:approvalPanelBody |
| R-AP-482 | G-C4-12 | 交互 | 鼠标点击参数行时 | 无动作 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-12→internal/ui/approval_panels.go:approvalPanelBody；标记 = Pressed@approvalCodeBlock） |
| R-AP-483 | G-C4-13 | 状态 | 未观察（条件）：`web_search` 的权限请求 | 等宽主值行「query: {值}」 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-C4-13→internal/ui/approval_panels.go:approvalWebPanel |
| R-AP-484 | G-C4-13 | 内容 | 查询为空 | 行仍显示（`query: `），不删行 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-13→internal/ui/approval_panels.go:approvalWebPanel；标记 = 文案「输入」） |
| R-AP-485 | G-C4-13 | 样式 | 主值行与参数行同屏时 | 同 90% 不透明主值行 | — | 判据 = 色值取 token：主色 #349651/#63cb7b、正文 #0a0a0a/#ececec、次文 #737373/#8a8a8a、边框 #e5e5e5/#2e2e2e、警示 #e17100/#ffb900，偏差 0（CL-ACCENT / CL-TEXT / CL-TEXT-2 / CL-BORDER / CL-WARN）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-C4-13→internal/ui/approval_panels.go:approvalWebPanel |
| R-AP-486 | G-C4-13 | 无障碍 | 读屏进入该元素时 | 读出查询词 | — | 读屏文本 = 「query: {值}」 | 已实现（映射 = G-C4-13→internal/ui/approval_panels.go:approvalWebPanel；标记 = Label） |
| R-AP-487 | G-C4-13 | 异常 | 查询超长 | 折行，不截断 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 未实现（映射到位但没有该维度的标记）：G-C4-13→internal/ui/approval_panels.go:approvalWebPanel |
| R-AP-488 | G-C4-13 | 记忆 | 切到其它会话再切回时 | 查询词逐字不变，`query: ` 前缀保留 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-13→internal/ui/approval_panels.go:approvalWebPanel |
| R-AP-489 | G-C4-13 | 交互 | 选中查询词 | 可选中复制（不含「query: 」前缀） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（逐字）：internal/ui/approval_panels.go:approvalSearchPanel「query:」 |
| R-AP-490 | G-C4-13 | 窄窗 | 窗口收窄到最窄宽度时 | 查询折行，键名完整 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 未实现（映射到位但没有该维度的标记）：G-C4-13→internal/ui/approval_panels.go:approvalWebPanel |
| R-AP-491 | G-C4-14 | 状态 | 未观察（条件）：网页搜索带分页 | 参数行「page/pageSize: {值}」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-14→internal/ui/approval_panels.go:approvalWebPanel |
| R-AP-492 | G-C4-14 | 内容 | 只有 page | 只显示 `page: {值}`，不补默认 pageSize | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-14→internal/ui/approval_panels.go:approvalWebPanel；标记 = 文案「输入」） |
| R-AP-493 | G-C4-14 | 样式 | 分页行与参数行同屏时 | 70% 不透明等宽 | — | 判据 = 色值取 token：主色 #349651/#63cb7b、正文 #0a0a0a/#ececec、次文 #737373/#8a8a8a、边框 #e5e5e5/#2e2e2e、警示 #e17100/#ffb900，偏差 0（CL-ACCENT / CL-TEXT / CL-TEXT-2 / CL-BORDER / CL-WARN）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-14→internal/ui/approval_panels.go:approvalWebPanel |
| R-AP-494 | G-C4-14 | 无障碍 | 读屏进入该元素时 | 读出页码与页大小 | — | 读屏文本 = 「page/pageSize: {值}」 | 已实现（映射 = G-C4-14→internal/ui/approval_panels.go:approvalWebPanel；标记 = Label） |
| R-AP-495 | G-C4-14 | 异常 | 值为 0 或负 | 原样显示（改由执行侧校验），不在卡片里改写 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-14→internal/ui/approval_panels.go:approvalWebPanel |
| R-AP-496 | G-C4-14 | 记忆 | 切到其它会话再切回时 | 页码与页大小逐字不变 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-14→internal/ui/approval_panels.go:approvalWebPanel |
| R-AP-497 | G-C4-14 | 交互 | 点击页码行 | 无动作（不是翻页控件） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-14→internal/ui/approval_panels.go:approvalWebPanel；标记 = Key） |
| R-AP-498 | G-C4-14 | 窄窗 | 窗口收窄到最窄宽度时 | 两个数值同行或折行，键名完整 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 未实现（映射到位但没有该维度的标记）：G-C4-14→internal/ui/approval_panels.go:approvalWebPanel |
| R-AP-499 | G-C4-15 | 状态 | 未观察（条件）：`web_fetch` 的权限请求 | 顶部标签显示 URL（超长截断） | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-15→internal/ui/approval_panels.go:approvalPanelLabel |
| R-AP-500 | G-C4-15 | 内容 | URL 很长带查询串 | truncate 显示，悬停出完整 URL | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-15→internal/ui/approval_panels.go:approvalPanelLabel；标记 = 文案「站点：」@siteBadge） |
| R-AP-501 | G-C4-15 | 样式 | URL 行与「输入」标签同屏时 | 同 11px 次要色标签位，等宽或常规按实现 | — | 判据 = 字号 键帽 11px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-C4-15→internal/ui/approval_panels.go:approvalPanelLabel；标记 = TextColor） |
| R-AP-502 | G-C4-15 | 无障碍 | 读屏进入该元素时 | 读出 URL | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-C4-15→internal/ui/approval_panels.go:approvalPanelLabel；标记 = Label） |
| R-AP-503 | G-C4-15 | 异常 | URL 非法 | 原文照显，不显示「无效 URL」替代（审批对象是原文） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-504 | G-C4-15 | 记忆 | 切到其它会话再切回时 | URL 逐字不变，截断位置相同 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-15→internal/ui/approval_panels.go:approvalPanelLabel |
| R-AP-505 | G-C4-15 | 交互 | 选中 URL | 可复制完整 URL | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-C4-15→internal/ui/approval_panels.go:approvalWebPanel；标记 = Key） |
| R-AP-506 | G-C4-15 | 窄窗 | 窗口收窄到最窄宽度时 | URL 截断但悬停可看全 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-15→internal/ui/approval_panels.go:approvalPanelLabel |
| R-AP-507 | G-C4-16 | 状态 | 未观察（条件）：网页抓取带附加参数 | 参数行组「startIndex/contextMaxCharacters/mode: {值}」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-16→internal/ui/approval_panels.go:approvalWebPanel |
| R-AP-508 | G-C4-16 | 内容 | 无附加参数 | 参数行组不出现 | — | 屏上 等宽参数行组 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-509 | G-C4-16 | 样式 | 参数行与主值行同屏时 | 70% 不透明等宽 | — | 判据 = 色值取 token：主色 #349651/#63cb7b、正文 #0a0a0a/#ececec、次文 #737373/#8a8a8a、边框 #e5e5e5/#2e2e2e、警示 #e17100/#ffb900，偏差 0（CL-ACCENT / CL-TEXT / CL-TEXT-2 / CL-BORDER / CL-WARN）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-16→internal/ui/approval_panels.go:approvalWebPanel |
| R-AP-510 | G-C4-16 | 无障碍 | 读屏进入该元素时 | 逐行读出 | — | 播报顺序 = startIndex → contextMaxCharacters → mode；每行先键后值 | 已实现（映射 = G-C4-16→internal/ui/approval_panels.go:approvalWebPanel；标记 = Label） |
| R-AP-511 | G-C4-16 | 异常 | 数值超界 | 原样显示，不在卡片里夹带校验错误 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-16→internal/ui/approval_panels.go:approvalWebPanel |
| R-AP-512 | G-C4-16 | 记忆 | 切到其它会话再切回时 | 各字段逐字不变，行序不变 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-16→internal/ui/approval_panels.go:approvalWebPanel |
| R-AP-513 | G-C4-16 | 交互 | 鼠标点击参数行时 | 无动作 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-16→internal/ui/approval_panels.go:approvalWebPanel；标记 = Key） |
| R-AP-514 | G-C4-16 | 窄窗 | 窗口收窄到最窄宽度时 | 参数行折行，缩进保持 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 未实现（映射到位但没有该维度的标记）：G-C4-16→internal/ui/approval_panels.go:approvalWebPanel |
| R-AP-515 | G-C4-17 | 状态 | 未观察（条件）：`skill` 的权限请求 | 顶部标签 + 等宽行「skill_name: {名称}」 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-17→internal/ui/approval_panels.go:approvalPanelFor |
| R-AP-516 | G-C4-17 | 内容 | 技能名带命名空间（`plugin:name`） | 原样显示，不裁掉前缀 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-C4-17→internal/ui/approval_panels.go:approvalPanelFor；标记 = 文案「输入」@approvalCommandPanel） |
| R-AP-517 | G-C4-17 | 样式 | 技能标签与值行同时在场时 | 标签同 11px 次要色；值行 90% 不透明等宽 | — | 判据 = 字号 键帽 11px；其余 = 色值取 token：主色 #349651/#63cb7b、正文 #0a0a0a/#ececec、次文 #737373/#8a8a8a、边框 #e5e5e5/#2e2e2e、警示 #e17100/#ffb900，偏差 0（CL-ACCENT / CL-TEXT / CL-TEXT-2 / CL-BORDER / CL-WARN） | 未实现（映射到位但没有该维度的标记）：G-C4-17→internal/ui/approval_panels.go:approvalPanelFor |
| R-AP-518 | G-C4-17 | 无障碍 | 读屏进入该元素时 | 读出技能名 | — | 读屏文本 = 「skill_name: {名称}」 | 已实现（映射 = G-C4-17→internal/ui/approval_panels.go:approvalPanelFor；标记 = Label@approvalCommandPanel） |
| R-AP-519 | G-C4-17 | 异常 | 技能名为空 | 行仍显示键名 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-C4-17→internal/ui/approval_panels.go:approvalPanelFor；标记 = err@approvalGenericPanel） |
| R-AP-520 | G-C4-17 | 记忆 | 切到其它会话再切回时 | 技能名逐字不变，命名空间前缀（`plugin:`）保留 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-C4-17→internal/ui/approval_panels.go:approvalPanelFor |
| R-AP-521 | G-C4-17 | 交互 | 选中技能名 | 可复制技能名 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-C4-17→internal/ui/approval_panels.go:approvalPanelFor；标记 = Key） |
| R-AP-522 | G-C4-17 | 窄窗 | 窗口收窄到最窄宽度时 | 命名空间不折断成两行（整段换行） | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 未实现（映射到位但没有该维度的标记）：G-C4-17→internal/ui/approval_panels.go:approvalPanelFor |
| R-AP-523 | G-C4-18 | 状态 | 未观察（条件）：`agent` 的 `create`/`create_external` 动作 | 头行：14px 图标 + 名称 + 任务标题 + 模型 id，四件一行 | — | 判据 = 字号 UI 正文 12.25px；控件尺寸 图标 16 / 14 / 12px | 已实现（映射 = G-C4-18→internal/ui/approval_panels.go:approvalHeadRow；标记 = Hidden@iconGlyph） |
| R-AP-524 | G-C4-18 | 内容 | 任务标题缺失 | 该段不出现，其余三段照常，不留双空格 | — | 屏上 行 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-525 | G-C4-18 | 内容 | 名称/模型 id 超长 | 各段 truncate，行高不变 | — | 行数恒 1 | 已实现（映射 = G-C4-18→internal/ui/approval_panels.go:approvalHeadRow；标记 = 文案「图标：」@iconGlyph） |
| R-AP-526 | G-C4-18 | 样式 | 等宽与非等宽混排 | 名称与任务标题常规字体，模型 id 等宽小号 | — | 判据 = 字号 UI 正文 12.25px；其余 = 字号 12.25 / 行高 18 / 字重 400，说明文字 10.5 / 14（TX-TYPE-BODY / TX-TYPE-MICRO）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-18→internal/ui/approval_panels.go:approvalHeadRow；标记 = TextColor） |
| R-AP-527 | G-C4-18 | 无障碍 | 读屏进入该元素时 | 依次读出四段 | — | 播报顺序 = 图标 → 名称 → 任务标题 → 模型 id | 已实现（映射 = G-C4-18→internal/ui/approval_panels.go:approvalHeadRow；标记 = Label@iconGlyph） |
| R-AP-528 | G-C4-18 | 异常 | 只有图标无文本 | 不显示空行（至少需一段文本） | — | 屏上 空行 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-529 | G-C4-18 | 记忆 | 切到其它会话再切回时 | 四段逐字不变（图标 + 名称 + 任务标题 + 模型 id） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-C4-18→internal/ui/approval_panels.go:approvalHeadRow |
| R-AP-530 | G-C4-18 | 交互 | 点击头行 | 无动作（不打开该子代理） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-C4-18→internal/ui/approval_panels.go:approvalHeadRow |
| R-AP-531 | G-C4-19 | 状态 | 未观察（条件）：子代理请求带 `prompt` | 等宽文本块，可断行，位于头行之下 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-19→internal/ui/approval_panels.go:approvalAgentPanel |
| R-AP-532 | G-C4-19 | 内容 | prompt 多行 | 保留换行 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-C4-19→internal/ui/approval_panels.go:approvalAgentPanel；标记 = 文案「输入」） |
| R-AP-533 | G-C4-19 | 内容 | prompt 超长 | 块内滚动（受 240 限制），可复制全文 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-19→internal/ui/approval_panels.go:approvalAgentPanel；标记 = 文案「输入」） |
| R-AP-534 | G-C4-19 | 样式 | prompt 块与命令行同屏时 | 等宽、无 `$` 前缀、主前景色 | — | 判据 = 字号 UI 正文 12.25px；其余 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；字号 12.25 / 行高 18 / 字重 400，说明文字 10.5 / 14（TX-TYPE-BODY / TX-TYPE-MICRO）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-19→internal/ui/approval_panels.go:approvalAgentPanel |
| R-AP-535 | G-C4-19 | 无障碍 | 读屏进入该元素时 | 读出 prompt 全文 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-C4-19→internal/ui/approval_panels.go:approvalAgentPanel；标记 = Label） |
| R-AP-536 | G-C4-19 | 异常 | prompt 为空 | 块不出现 | — | 屏上 等宽文本块 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-537 | G-C4-19 | 记忆 | 切到其它会话再切回时 | prompt 全文逐字不变，换行位置相同 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 未实现（映射到位但没有该维度的标记）：G-C4-19→internal/ui/approval_panels.go:approvalAgentPanel |
| R-AP-538 | G-C4-19 | 交互 | 选中 prompt | 可复制 prompt 全文 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-19→internal/ui/approval_panels.go:approvalAgentPanel |
| R-AP-539 | G-C4-20 | 状态 | 未观察（条件）：子代理请求带 `effort` / `forkContext` | 等宽参数行组逐字段一行 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-20→internal/ui/approval_panels.go:approvalPanelBody |
| R-AP-540 | G-C4-20 | 内容 | 无这些字段 | 参数行组不出现 | — | 屏上 等宽参数行组 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-541 | G-C4-20 | 样式 | 参数行与主值行同屏时 | 70% 不透明等宽 | — | 判据 = 色值取 token：主色 #349651/#63cb7b、正文 #0a0a0a/#ececec、次文 #737373/#8a8a8a、边框 #e5e5e5/#2e2e2e、警示 #e17100/#ffb900，偏差 0（CL-ACCENT / CL-TEXT / CL-TEXT-2 / CL-BORDER / CL-WARN）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-20→internal/ui/approval_panels.go:approvalPanelBody；标记 = TextColor） |
| R-AP-542 | G-C4-20 | 无障碍 | 读屏进入该元素时 | 逐行读出 | — | 播报顺序 = effort → forkContext；每行先键后值 | 已实现（映射 = G-C4-20→internal/ui/approval_panels.go:approvalPanelBody；标记 = Label@approvalImageBlock） |
| R-AP-543 | G-C4-20 | 异常 | `forkContext: true` | 显示 `true`，不翻译成「是」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-20→internal/ui/approval_panels.go:approvalPanelBody；标记 = err@approvalImageBlock） |
| R-AP-544 | G-C4-20 | 记忆 | 切到其它会话再切回时 | effort 与 forkContext 的值逐字不变 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-20→internal/ui/approval_panels.go:approvalPanelBody |
| R-AP-545 | G-C4-20 | 交互 | 鼠标点击参数行时 | 无动作 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-20→internal/ui/approval_panels.go:approvalPanelBody；标记 = Pressed@approvalCodeBlock） |
| R-AP-546 | G-C4-20 | 窄窗 | 窗口收窄到最窄宽度时 | 参数行折行，缩进保持 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-C4-20→internal/ui/approval_panels.go:approvalPanelBody；标记 = narrow@approvalCodeBlock） |
| R-AP-547 | G-C4-21 | 状态 | 未观察（条件）：`agent` 的 list/cancel/query 等非创建动作 | 顶部标签 + 行列表，逐行「键: 值」（action/taskId/agentName/taskTitle/agentType/providerId/modelId/effort/reason） | — | 排列顺序 = action → taskId → agentName → taskTitle → agentType → providerId → modelId → effort → reason | 未实现（映射到位但没有该维度的标记）：G-C4-21→internal/ui/approval_panels.go:approvalPanelBody |
| R-AP-548 | G-C4-21 | 内容 | 字段缺值 | 缺的字段不出行，不留「键: 」空行 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 已实现（逐字）：internal/account/actions.go:randomActionKey「键:」 |
| R-AP-549 | G-C4-21 | 内容 | 字段很多 | 区块内滚动 | — | 判据 = 滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-21→internal/ui/approval_panels.go:approvalPanelBody；标记 = 文案「图片预览」等 2 条@approvalImageBlock） |
| R-AP-550 | G-C4-21 | 样式 | 同类行列表在不同渲染器下先后出现时 | 同一套行列表样式（键次要色、值主色、等宽） | — | 判据 = 行高 UI 正文 17.5px；间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px | 已实现（映射 = G-C4-21→internal/ui/approval_panels.go:approvalPanelBody；标记 = TextColor） |
| R-AP-551 | G-C4-21 | 无障碍 | 读屏进入该元素时 | 逐行读出键值 | — | 播报顺序 = action → taskId → agentName → taskTitle → agentType → providerId → modelId → effort → reason；每行先键后值 | 已实现（映射 = G-C4-21→internal/ui/approval_panels.go:approvalPanelBody；标记 = Label@approvalImageBlock） |
| R-AP-552 | G-C4-21 | 异常 | 值含嵌套对象 | 序列化为单行文本 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-21→internal/ui/approval_panels.go:approvalPanelBody；标记 = err@approvalImageBlock） |
| R-AP-553 | G-C4-21 | 记忆 | 切到其它会话再切回时 | 每个键值逐字不变，行序不变 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-C4-21→internal/ui/approval_panels.go:approvalPanelBody |
| R-AP-554 | G-C4-21 | 交互 | 点击行列表任一行 | 无动作 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-C4-21→internal/ui/approval_panels.go:approvalPanelBody；标记 = Pressed@approvalCodeBlock） |
| R-AP-555 | G-C4-22 | 状态 | 未观察（条件）：`browser` 工具的权限请求 | 顶部标签 + 行列表（action/url/ref/value/key/direction/amount/type/timeout/expression/method/params/selector/interactiveOnly/compact/fullPage/clear） | — | 判据 = 动效 布局 250ms / 颜色 150ms / 折叠 300ms；缓动 cubic-bezier(0.16, 1, 0.3, 1) / cubic-bezier(0.4, 0, 0.2, 1) | 未实现（映射到位但没有该维度的标记）：G-C4-22→internal/ui/approval_panels.go:approvalPanelBody |
| R-AP-556 | G-C4-22 | 内容 | 字段极多（16+） | 区块内滚动，最大高 240 | — | 判据 = 滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-22→internal/ui/approval_panels.go:approvalPanelBody；标记 = 文案「图片预览」等 2 条@approvalImageBlock） |
| R-AP-557 | G-C4-22 | 内容 | 只有 action | 只有一行，不留空行 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-C4-22→internal/ui/approval_panels.go:approvalPanelBody；标记 = 文案「图片预览」等 2 条@approvalImageBlock） |
| R-AP-558 | G-C4-22 | 样式 | 浏览器行列表与子代理行列表先后出现时 | 同一套行列表样式 | — | 判据 = 行高 UI 正文 17.5px；间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px | 已实现（映射 = G-C4-22→internal/ui/approval_panels.go:approvalPanelBody；标记 = TextColor） |
| R-AP-559 | G-C4-22 | 无障碍 | 读屏进入该元素时 | 逐行读出 | — | 播报顺序 = action → url → ref → value → key → direction → amount → type → timeout → expression → method → params → selector → interactive；每行先键后值 | 已实现（映射 = G-C4-22→internal/ui/approval_panels.go:approvalPanelBody；标记 = Label@approvalImageBlock） |
| R-AP-560 | G-C4-22 | 异常 | 本机无浏览器能力（能力经 MCP 提供） | 该渲染器不出现；若出现则按通用渲染器兜底 | — | 屏上 顶部标签 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-561 | G-C4-22 | 记忆 | 切到其它会话再切回时 | 每个键值逐字不变，行序不变 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-C4-22→internal/ui/approval_panels.go:approvalPanelBody |
| R-AP-562 | G-C4-22 | 交互 | 点击行列表任一行 | 无动作 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-C4-22→internal/ui/approval_panels.go:approvalPanelBody；标记 = Pressed@approvalCodeBlock） |
| R-AP-563 | G-C4-23 | 状态 | 未观察（条件）：接入 cua-driver 工具 | 顶部标签 + 行列表，首行「tool: {tool}」，工具名的 `cua-driver__` 前缀被去掉 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-C4-23→internal/ui/approval_panels.go:approvalPanelBody |
| R-AP-564 | G-C4-23 | 内容 | 入参很多 | 区块内滚动 | — | 判据 = 滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-23→internal/ui/approval_panels.go:approvalPanelBody；标记 = 文案「图片预览」等 2 条@approvalImageBlock） |
| R-AP-565 | G-C4-23 | 样式 | 工具行列表与子代理行列表先后出现时 | 同一套行列表样式 | — | 判据 = 行高 UI 正文 17.5px；间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px | 已实现（映射 = G-C4-23→internal/ui/approval_panels.go:approvalPanelBody；标记 = TextColor） |
| R-AP-566 | G-C4-23 | 无障碍 | 读屏进入该元素时 | 读出工具名与各参数 | — | 读屏文本 = 「tool: {tool}」 | 已实现（映射 = G-C4-23→internal/ui/approval_panels.go:approvalPanelBody；标记 = Label@approvalImageBlock） |
| R-AP-567 | G-C4-23 | 异常 | 前缀不存在 | 工具名原样显示 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-C4-23→internal/ui/approval_panels.go:approvalPanelBody；标记 = err@approvalImageBlock） |
| R-AP-568 | G-C4-23 | 记忆 | 切到其它会话再切回时 | 工具名（去掉 `cua-driver__` 前缀后）与各参数逐字不变 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-C4-23→internal/ui/approval_panels.go:approvalPanelBody |
| R-AP-569 | G-C4-23 | 交互 | 点击行列表任一行 | 无动作 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-C4-23→internal/ui/approval_panels.go:approvalPanelBody；标记 = Pressed@approvalCodeBlock） |
| R-AP-570 | G-C4-23 | 窄窗 | 窗口收窄到最窄宽度时 | 行列表折行，值不与键拆开 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-C4-23→internal/ui/approval_panels.go:approvalPanelBody；标记 = narrow@approvalCodeBlock） |
| R-AP-571 | G-C4-24 | 状态 | 未观察（条件）：`wait` 工具的权限请求 | 顶部标签 + 行列表「taskId: {id}」「timeoutMs: {值}」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-C4-24→internal/ui/approval_panels.go:approvalPanelBody |
| R-AP-572 | G-C4-24 | 内容 | 只有 taskId | 只一行，不补 timeoutMs 默认值 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 已实现（映射 = G-C4-24→internal/ui/approval_panels.go:approvalPanelBody；标记 = 文案「图片预览」等 2 条@approvalImageBlock） |
| R-AP-573 | G-C4-24 | 样式 | 子代理等待行与子代理行列表先后出现时 | 同一套行列表样式 | — | 判据 = 行高 UI 正文 17.5px；间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px | 已实现（映射 = G-C4-24→internal/ui/approval_panels.go:approvalPanelBody；标记 = TextColor） |
| R-AP-574 | G-C4-24 | 无障碍 | 读屏进入该元素时 | 读出任务与超时 | — | 播报顺序 = taskId: {id} → timeoutMs: {值} | 已实现（映射 = G-C4-24→internal/ui/approval_panels.go:approvalPanelBody；标记 = Label@approvalImageBlock） |
| R-AP-575 | G-C4-24 | 异常 | timeoutMs 为 0 | 原样显示（`0`），不改写 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-C4-24→internal/ui/approval_panels.go:approvalPanelBody；标记 = err@approvalImageBlock） |
| R-AP-576 | G-C4-24 | 记忆 | 切到其它会话再切回时 | taskId 与 timeoutMs 逐字不变 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-C4-24→internal/ui/approval_panels.go:approvalPanelBody |
| R-AP-577 | G-C4-24 | 交互 | 点击 taskId 行 | 无动作（不跳到该任务详情） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-C4-24→internal/ui/approval_panels.go:approvalPanelBody；标记 = Pressed@approvalCodeBlock） |
| R-AP-578 | G-C4-24 | 窄窗 | 窗口收窄到最窄宽度时 | 两行折行，缩进保持 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-C4-24→internal/ui/approval_panels.go:approvalPanelBody；标记 = narrow@approvalCodeBlock） |
| R-AP-579 | G-C4-25 | 状态 | 未观察（条件）：`recall`/`settings`/`plugin`/`exec_stop`/MCP 等未登记工具的权限请求 | 顶部标签（等宽 11px 工具名）+ 等宽代码块，两空格缩进的 JSON 入参 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-C4-25→internal/ui/approval_panels.go:approvalGenericPanel |
| R-AP-580 | G-C4-25 | 内容 | 旧版纯文本入参 `{input: 文本}` | 同一块内以文本呈现，不显示 `\n` 等转义字面量 | — | 屏上 顶部标签 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-AP-581 | G-C4-25 | 内容 | 入参很大 | 块内滚动，最大高 240 | — | 判据 = 滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-25→internal/ui/approval_panels.go:approvalGenericPanel；标记 = 文案「输入」） |
| R-AP-582 | G-C4-25 | 交互 | 选中屏上文本后复制时 | 可复制 JSON 原文 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-C4-25→internal/ui/approval_panels.go:approvalCodeBlock；标记 = Pressed） |
| R-AP-583 | G-C4-25 | 样式 | 分别切到浅色与深色主题时 | 等宽 12px、两空格缩进、键值同色（不做语法高亮） | — | 判据 = 字号 UI 正文 12.25px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-25→internal/ui/approval_panels.go:approvalCodeBlock；标记 = Height） |
| R-AP-584 | G-C4-25 | 无障碍 | 读屏进入该元素时 | 读出工具名与 JSON 文本 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-C4-25→internal/ui/approval_panels.go:approvalGenericPanel；标记 = Label） |
| R-AP-585 | G-C4-25 | 异常 | 入参不是合法 JSON | 原文照显（不折叠成「无效 JSON」） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-C4-25→internal/ui/approval_panels.go:approvalGenericPanel；标记 = err） |
| R-AP-586 | G-C4-25 | 记忆 | 切到其它会话再切回时 | 工具名与 JSON 逐字不变，缩进仍为两空格 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-C4-25→internal/ui/approval_panels.go:approvalGenericPanel |
| R-AP-587 | G-C4-25 | 窄窗 | 窗口收窄到最窄宽度时 | 块内横向滚动，不撑破卡片 | — | 判据 = 滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-C4-25→internal/ui/approval_panels.go:approvalCodeBlock；标记 = narrow） |

### 1.5 审批域维度覆盖说明

63 项元素维度覆盖全部 ≥6，无需例外。三件只到 6 维：

| 元素 | 维度数 | 缺的两维与理由 |
|---|---:|---|
| G-C1-03 | 6 | 无「动画 / 交互」：标题回退是文案替换，无自身过渡、不可点（交互由外层标题位承担）。 |
| G-C1-10 | 6 | 无「动画 / 窄窗」：等宽 ID 行不做过场；断行行为已在「内容」列写尽。 |
| G-C2-03 | 6 | 无「动画 / 窄窗」：域名/路径是静态等宽文本，窄窗行为与 C2-01 卡片一并承担。 |

### 1.6 实现现状（2026-11-19）

状态列：**已实现 302 条、未实现 285 条**（共 587）。本轮落地的是审批卡与输入渲染器：

- **输入区 13 个渲染器**：命令（`$` + cwd / timeoutMs / runInBackground）、文件（路径标签 + 编辑双栏差异 / 写入带行号 / 补丁无行号）、检索（pattern / query / path + 参数）、网页（搜索 query + 分页；抓取 URL 标签 + 站点徽记 + 附加参数）、技能（`skill_name`）、子代理（创建类头行 + prompt + effort / forkContext；其余动作画行列表）、浏览器、电脑操作（`tool:` 去前缀）、等待（taskId / timeoutMs）、MCP（server / tool / 参数）、图片（缩略图 + 尺寸）、待办（清单）、通用（工具名 + 两空格缩进 JSON；旧版纯文本入参按 `{input: 文本}`）。每条一个界面用例，见 `internal/ui/approval_test.go`。
- **三个决定与长期允许**：文案由工具类别决定（`始终允许此命令` / `允许此会话期间访问` / `始终允许文件访问`），一次性允许与长期允许是两个答复值；提交中与已答未结算整组禁用（50% 不透明），长期允许在服务端没给长期规则参数时置灰。键位照参考产品：Enter / y 一次性允许，Esc / n 拒绝（与 R-AP-017、R-AP-197 的期望列口径不同，见那两行的状态）。
- **失败回执与重试**：提交失败写进卡片自己的错误行（危险色、`Role(Alert)`、按钮行之上），卡片保留、按钮恢复；再点同一枚按钮先清错误再提交。失败原因不写标题下方的全局错误行，避免同一句出现两遍。
- **成员审批**：按请求工具名 `member_approval_allow` 认出，明细通道（`session.collaboration.approval.get`）本端没有，按参考产品的「当前客户端无法审查成员操作」只画审批 ID 与「拒绝」。
- **沙箱越权卡 / 子代理模型卡 / 自动审批提示块**：渲染器与文案已就位（`internal/ui/approval.go` 的 `sandboxApprovalBody` / `subagentModelBody` / `handoffBlock`），protocol v3 的 `approval.request` 没有这三类字段（只有 tool / host 两种来源），连接层不填，所以生产环境不会出现这三张卡。
- **本轮仍缺**：屏上文本的选中复制、差异各栏独立滚动、2× 走查、i18n、入参“解码失败”提示、切换会话后的滚动位置回顶；长期允许在服务端不支持时保留但置灰（R-AP-214 期望隐藏）。

---



---

## 2. 搜索（46 元素）

五组：会话内搜索浮层（A1，10）、搜索聊天对话框（A2，18）、`@` 提及（A3，11）、`$` 技能提及（A4，5）、时间线命中高亮（A5，2）。

### 2.1 会话内搜索浮层（G-A1-01–G-A1-09）

条件：主区有活跃会话时 `Cmd/Ctrl+F` 打开；浮层绝对定位在标题栏右侧，点浮层外关闭。已采三态：C24（空）/ C25（无命中）/ C26（有命中，35 条）。

本端实现（`internal/ui/search.go`）：命中来自**本帧时间线已经拿到的文本**（`RowSnapshot` 与样例轮次里的用户消息、助手正文、工具参数与输出、思考块），不新增协议方法——协议 v3 没有会话内检索（`gaps.md` G-02），所以检索是本机的、同步的：输入即重建命中，没有「进行中」态，也就没有旋转圈与失败分支。浮层是 `Overlay` 里的一层：整窗一层透明点击区负责「点浮层外关闭」，浮层本体绝对定位在标题栏右侧（宽 `min(420, 主区 − 2×内边距)`），不推挤标题栏与时间线。状态全部是可丢的视图状态：查询、命中、当前项、滚动还原位置、搜索范围开关（思考块可关）、跳转时展开过的折叠位（关闭即复原）。跳转用 `ScrollIntoView` 把当前命中的行滚进视口，关闭后写回打开前的滚动位置。

| 需求 ID | 元素 ID | 维度 | 场景 | 期望 | 异常 | 验收 | 状态 |
|---|---|---|---|---|---|---|---|
| R-GS-001 | G-A1-01 | 状态 | C24：会话中按 `Cmd/Ctrl+F` | 浮层出现：圆角 + 1px 描边 + 阴影 + 背景模糊，两行网格（44px 输入行 + 结果行）；实测 ≈310,10 406×80（模板声明 420，实测窄 14） | — | 屏上出现 1 个浮层，包围盒 ≈310,10 406×80 | 已实现（映射 = G-A1-01→internal/ui/search.go:App.sessionSearchOverlay；标记 = Hidden） |
| R-GS-002 | G-A1-01 | 交互 | 点击浮层外 | 浮层关闭，搜索状态清除，时间线高亮一并消失 | — | 屏上 浮层 计数 = 0 | 已实现（映射 = G-A1-01→internal/ui/search.go:App.sessionSearchKeys；标记 = Key） |
| R-GS-003 | G-A1-01 | 状态 | 已有浮层时再按 `Cmd/Ctrl+F` | 不新开第二个：聚焦已有输入框并全选 | — | 浮层计数恒为 1 | 已实现（映射 = G-A1-01→internal/ui/search.go:App.sessionSearchOverlay；标记 = Hidden） |
| R-GS-004 | G-A1-01 | 关联 | 浮层打开、标题栏与时间线仍在屏上时 | 浮层浮在标题栏之上，不推挤标题栏与时间线（`absolute`） | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-A1-01→internal/ui/search.go:App.sessionSearchKeys；标记 = Screen） |
| R-GS-005 | G-A1-01 | 动画 | 打开 / 关闭 | 出现与消失 ≤120ms，无位移；减少动效下瞬切 | 减少动效下瞬切 | 屏上 浮层 计数 = 0；时长 = 期望列给的毫秒值（±0ms），减少动效时 none（§6 动效表） | 未实现（映射到位但没有该维度的标记）：G-A1-01→internal/ui/search.go:App.sessionSearchKeys；测试提及 internal/ui/lint_test.go:lintDividerAllowlist |
| R-GS-006 | G-A1-01 | 样式 | 分别切到浅色与深色主题时 | 背景为上层色 + 模糊、阴影柔和；深色主题下阴影更弱、描边更明显 | — | 判据 = 底色 浮层 明 #ffffff / 暗 #222222；描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；其余 = 色值取 token：主色 #349651/#63cb7b、正文 #0a0a0a/#ececec、次文 #737373/#8a8a8a、边框 #e5e5e5/#2e2e2e、警示 #e17100/#ffb900，偏差 0（CL-ACCENT / CL-TEXT / CL-TEXT-2 / CL-BORDER / CL-WARN） | 已实现（映射 = G-A1-01→internal/ui/search.go:App.sessionSearchOverlay；标记 = Width） |
| R-GS-007 | G-A1-01 | 无障碍 | 读屏进入该元素时 | 浮层读作搜索区域，进入后焦点在输入框 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-A1-01→internal/ui/search.go:App.sessionSearchOverlay；标记 = AccessHidden） |
| R-GS-008 | G-A1-01 | 异常 | 时间线为空（空会话） | 浮层仍可打开，结果恒「0 个结果」，不报错 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-009 | G-A1-01 | 记忆 | 关闭再打开 | 不记忆上次查询（打开即为空，除选中文本预填） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-A1-01→internal/ui/search.go:App.sessionSearchKeys；测试提及 internal/ui/lint_test.go:lintDividerAllowlist |
| R-GS-010 | G-A1-01 | 窄窗 | 窄窗（replacement 布局） | 浮层右缘贴窗口内边距，宽度按 `min(420, 100vw-… )` 收缩，不溢出窗口 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-A1-01→internal/ui/search.go:App.sessionSearchOverlay；标记 = narrow） |
| R-GS-011 | G-A1-02a | 状态 | C24/C25/C26：不在搜索中 | 输入框左端内侧显示放大镜（20px），与占位文字同一行 | — | 判据 = 字号 UI 正文 12.25px；字重 常规 400 / 中粗 500 / 半粗 600 / 粗 700 | 已实现（映射 = G-A1-02a→internal/ui/search.go:App.sessionSearchInputRow；标记 = Hidden@iconGlyphSize） |
| R-GS-012 | G-A1-02a | 关联 | 检索进行中与返回后交替时 | 二者互斥：瞬时检索期间换旋转圈，返回后换回放大镜 | — | 屏上 图标 计数 = 1 | 已实现（映射 = G-A1-02a→internal/ui/search.go:App.sessionSearchInputRow；标记 = Select） |
| R-GS-013 | G-A1-02a | 样式 | 分别切到浅色与深色主题时 | 次要色描边、20px；与占位文字同色 | — | 判据 = 控件尺寸 图标 16 / 14 / 12px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-A1-02a→internal/ui/search.go:App.sessionSearchInputRow；标记 = Height） |
| R-GS-014 | G-A1-02a | 交互 | 鼠标点击该图标时 | 穿透到输入框并聚焦（图标不拦截） | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-015 | G-A1-02a | 无障碍 | 读屏进入该元素时 | 不单独播报（可读名由输入框承担） | — | 播报次数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-016 | G-A1-02a | 高DPI | 在 DPR=2 的屏幕上时 | 图标 2× 下清晰 | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-A1-02a→internal/ui/search.go:App.sessionSearchInputRow；标记 = Scale@fontStep.sizeIn） |
| R-GS-017 | G-A1-02a | 交互 | 搜索中切换图标时点击原位置 | 点击落在输入框上（图标位置不吞点击），焦点不丢 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-A1-02a→internal/ui/search.go:App.sessionSearchInputRow；标记 = Key） |
| R-GS-018 | G-A1-02a | 内容 | 查询为空 | 放大镜仍显示（不因空查询隐藏） | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 已实现（映射 = G-A1-02a→internal/ui/search.go:App.sessionSearchInputRow；标记 = 文案「关闭搜索」等 2 条） |
| R-GS-019 | G-A1-02b | 状态 | 未观察（条件）：检索进行中（`pending`，本机检索瞬时返回，三态均未捕获） | 放大镜换成旋转圈（Loader2），同样 20px、同位 | — | 判据 = 控件尺寸 图标 16 / 14 / 12px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（找不到元素渲染点） |
| R-GS-020 | G-A1-02b | 动画 | 进行中 | 旋转连续；结果返回后与结果行同帧换回放大镜 | 减少动效下静态 | 判据 = 动效 布局 250ms / 颜色 150ms / 折叠 300ms；缓动 cubic-bezier(0.16, 1, 0.3, 1) / cubic-bezier(0.4, 0, 0.2, 1) | 未实现（找不到元素渲染点） |
| R-GS-021 | G-A1-02b | 关联 | 上一次有结果、本次检索进行中时 | 进行中结果文本保持上一次值或空，不显示「0 个结果」假值 | — | 计数文本 ≠「0 个结果」（进行中保持上次值） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-022 | G-A1-02b | 异常 | 检索挂住 | 旋转持续，界面仍可改查询、可关闭 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（找不到元素渲染点） |
| R-GS-023 | G-A1-02b | 高DPI | 在 DPR=2 的屏幕上时 | 旋转圈在 2× 下边缘平滑 | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（找不到元素渲染点） |
| R-GS-024 | G-A1-02b | 内容 | 检索瞬时返回 | 旋转圈可能只在 1 帧出现；不得残留成常驻图标 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 未实现（找不到元素渲染点） |
| R-GS-025 | G-A1-02b | 样式 | 分别切到浅色与深色主题时 | 旋转圈与放大镜同尺寸同色，切换时位置不跳 | — | 判据 = 控件尺寸 图标 16 / 14 / 12px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 未实现（找不到元素渲染点） |
| R-GS-026 | G-A1-02b | 交互 | 检索中点击图标 | 不触发重搜；点击穿透到输入框 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-027 | G-A1-03 | 内容 | C24 打开即用 | 占位文案逐字「搜索聊天...」（三点为英文句点），无边框、无底色 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/search.go:sessionSearchPlaceholder「搜索聊天...」 |
| R-GS-028 | G-A1-03 | 状态 | 打开时时间线有选中文本 | 输入框预填该选中文本并全选（可直接覆盖输入） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-A1-03→internal/ui/app.go:App.viewKeys；标记 = Hidden@App.sidebarWorkspaces） |
| R-GS-029 | G-A1-03 | 状态 | C25 / C26 | 框内为查询词，占位消失；框宽不变（338×18 文本区） | — | 屏上 输入框 计数 = 0 | 已实现（映射 = G-A1-03→internal/ui/app.go:App.viewKeys；标记 = Hidden@App.sidebarWorkspaces） |
| R-GS-030 | G-A1-03 | 交互 | 在输入框键入时 | 逐字触发检索；时间线命中实时更新 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-A1-03→internal/ui/search.go:App.sessionSearchTextKey；标记 = Key） |
| R-GS-031 | G-A1-03 | 交互 | 全选后按 Delete | 查询清空，退回 C24（无结果行），时间线高亮清除 | — | 判据 = 底色 选中底 明 #f5f5f5 / 暗 #262626；字色 主文字 明 #0a0a0a / 暗 #ececec；其余 = 选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED） | 已实现（映射 = G-A1-03→internal/ui/search.go:App.sessionSearchTextKey；标记 = Key） |
| R-GS-032 | G-A1-03 | 键盘 | 输入框内按 Enter | 跳到下一个命中（等同 G-A1-08） | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-A1-03→internal/ui/search.go:App.sessionSearchTextKey；标记 = Key） |
| R-GS-033 | G-A1-03 | 键盘 | 输入框内按 Shift+Enter | 跳到上一个命中（等同 G-A1-07） | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-A1-03→internal/ui/search.go:App.sessionSearchTextKey；标记 = Key） |
| R-GS-034 | G-A1-03 | 键盘 | 输入框内按 Esc | 关闭浮层并清除高亮 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-A1-03→internal/ui/search.go:App.sessionSearchTextKey；标记 = Key） |
| R-GS-035 | G-A1-03 | 内容 | 查询超长 | 单行横向滚动，框高不变 | — | 框高恒 44 行内 | 已实现（映射 = G-A1-03→internal/ui/app.go:App.viewKeys；标记 = 文案「工作区」@App.sidebarWorkspaces） |
| R-GS-036 | G-A1-03 | 无障碍 | 读屏进入该元素时 | 输入框可读名为「搜索聊天消息」（见 G-A1-04），值为当前查询 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/search.go:sessionSearchLabel「搜索聊天消息」 |
| R-GS-037 | G-A1-03 | 异常 | 检索 RPC 失败 | 保持查询词，结果区给 0 或上次值，不弹错误框 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A1-03→internal/ui/app.go:App.viewKeys；标记 = err@App.createSessionAt） |
| R-GS-038 | G-A1-04 | 无障碍 | 读屏进入浮层 | 输入框可读名逐字「搜索聊天消息」 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/search.go:sessionSearchLabel「搜索聊天消息」 |
| R-GS-039 | G-A1-04 | 状态 | C24/C25/C26 | 该可读名恒定，不随查询或结果数变化 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE） | 已实现（映射 = G-A1-04→internal/ui/search.go:App.sessionSearchInputRow；标记 = Hidden@iconGlyphSize） |
| R-GS-040 | G-A1-04 | 关联 | 读屏读取可读名与占位文案时 | 可读名（「搜索聊天消息」）与占位（「搜索聊天...」）是两段不同文本，不互相顶替 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/ui/search.go:sessionSearchLabel「搜索聊天消息」 |
| R-GS-041 | G-A1-04 | 内容 | 读屏取输入框名称 | 名称逐字「搜索聊天消息」，与占位「搜索聊天...」并存且不同 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/search.go:sessionSearchLabel「搜索聊天消息」 |
| R-GS-042 | G-A1-04 | 无障碍 | Chrome/Electron 辅助技术 | 输入框以 `textbox` 语义暴露，名称不为 placeholder | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-A1-04→internal/ui/search.go:App.sessionSearchInputRow；标记 = Label） |
| R-GS-043 | G-A1-04 | 关联 | 两处输入框先后被读屏访问时 | 两处输入框各自有可读名，不共用一个 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A1-04→internal/ui/search.go:App.sessionSearchInputRow；标记 = Select） |
| R-GS-044 | G-A1-04 | 样式 | 读屏与 JSON 的 `label` 列比对时 | 该字段属 `aria-label`，不进可见文本；JSON 的 `label` 列携带它 | — | 判据 = 字号 UI 正文 12.25px；字重 常规 400 / 中粗 500 / 半粗 600 / 粗 700；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a；间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px | 已实现（映射 = G-A1-04→internal/ui/search.go:App.sessionSearchInputRow；标记 = Height） |
| R-GS-045 | G-A1-04 | 状态 | 浮层关闭时 | 该可读名随浮层卸载而消失，不残留在时间线 | — | 屏上 输入框的无障碍名 计数 = 0；上一次内容出现次数 0（ST-VISIBLE） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-046 | G-A1-05 | 状态 | C24/C25/C26 | 浮层右侧关闭键：X 图标，28×28（实测 688,18），可读名「关闭搜索」 | — | 屏上按钮 28×28，可读名一致 | 已实现（逐字）：internal/ui/search.go:App.sessionSearchInputRow「关闭搜索」 |
| R-GS-047 | G-A1-05 | 交互 | 鼠标点击该元素时 | 关闭浮层并清除高亮，焦点回到时间线 | — | 判据 = 底色 选中底 明 #f5f5f5 / 暗 #262626；字色 主文字 明 #0a0a0a / 暗 #ececec；其余 = 选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED） | 未实现（映射到位但没有该维度的标记）：G-A1-05→internal/ui/timeline.go:App.timelineFeed |
| R-GS-048 | G-A1-05 | 样式 | 鼠标悬停在行上时 | 悬停出浅底圆角；图标转主前景色 | — | 判据 = 底色 悬停底 明 #f5f5f5 / 暗 #262626（侧栏 明 #e2e9ed / 暗 #282c2b）；字色 次文字 明 #737373 / 暗 #8a8a8a | 未实现（映射到位但没有该维度的标记）：G-A1-05→internal/ui/timeline.go:App.timelineFeed |
| R-GS-049 | G-A1-05 | 无障碍 | 读屏进入该元素时 | 读作按钮「关闭搜索」 | — | 读屏文本 = 「关闭搜索」 | 已实现（逐字）：internal/ui/search.go:App.sessionSearchInputRow「关闭搜索」 |
| R-GS-050 | G-A1-05 | 键盘 | 按 Enter 时 | 等同点击 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-A1-05→internal/ui/timeline.go:App.timelineFeed；标记 = Tab） |
| R-GS-051 | G-A1-05 | 状态 | 查询为空 | 关闭键仍可用（不因无查询而禁用） | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-A1-05→internal/ui/timeline.go:App.timelineFeed |
| R-GS-052 | G-A1-05 | 关联 | 点关闭键收起浮层后 | 关闭后输入框随浮层一起卸载，焦点回到时间线正文 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-A1-05→internal/ui/timeline.go:App.timelineFeed；标记 = Tab） |
| R-GS-053 | G-A1-05 | 样式 | 关闭键与前后箭头按钮同屏时 | 同为 28px 热区的图标按钮，只有图标不同 | — | 判据 = 控件尺寸 图标 16 / 14 / 12px | 未实现（映射到位但没有该维度的标记）：G-A1-05→internal/ui/timeline.go:App.timelineFeed |
| R-GS-054 | G-A1-06 | 状态 | 查询词非空时 | 输入行下方出现结果行（36px，上边框，左右分布：左按钮组、右结果计数） | — | 结果行高 36，顶上边框 1px | 已实现（映射 = G-A1-06→internal/ui/search.go:App.sessionSearchResultRow；标记 = Visible） |
| R-GS-055 | G-A1-06 | 状态 | 查询为空（C24） | 结果行不出现，浮层只有 44px 一行（实测浮层 80 → 44+36） | — | 屏上 结果行 计数 = 0；浮层高 = 44（从 80 回落 36） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-056 | G-A1-06 | 关联 | 结果行在场、行内三段同屏时 | 三件同在该行内：左两个箭头、右计数文本 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-A1-06→internal/ui/search.go:App.sessionSearchResultRow |
| R-GS-057 | G-A1-06 | 样式 | 上边框 | 1px 中性色，横跨浮层内宽 | — | 判据 = 色值 边框 明 #e5e5e5 / 暗 #2e2e2e | 已实现（映射 = G-A1-06→internal/ui/search.go:App.sessionSearchResultRow；标记 = Height） |
| R-GS-058 | G-A1-06 | 无障碍 | 读屏进入该元素时 | 作为工具行读出，含按钮与计数 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-A1-06→internal/ui/search.go:App.sessionSearchResultRow；标记 = Label） |
| R-GS-059 | G-A1-06 | 异常 | 结果从有到无 | 结果行随查询清空一起消失，不留空行 | — | 屏上 行 计数 = 0 | 未实现（映射到位但没有该维度的标记）：G-A1-06→internal/ui/search.go:App.sessionSearchResultRow |
| R-GS-060 | G-A1-06 | 状态 | 查询从有到空 | 结果行消失后浮层高度回落 36，浮层底边随之上移 | — | 高度差恰为 36 | 已实现（映射 = G-A1-06→internal/ui/search.go:App.sessionSearchResultRow；标记 = Visible） |
| R-GS-061 | G-A1-06 | 内容 | 结果数为 0 | 结果行仍显示（左右分布不变，右侧为「0 个结果」） | — | 屏上 行 计数 = 1（0 命中时） | 已实现（逐字）：internal/ui/search.go:App.sessionSearchResultRow「0 个结果」 |
| R-GS-062 | G-A1-07 | 状态 | 查询词非空时 | 「上一个搜索结果」按钮（↑，25×25，实测 322,61），无命中时禁用 | — | 按钮 25×25；0 命中时置灰 | 已实现（逐字）：internal/ui/search.go:App.sessionSearchResultRow「上一个搜索结果」 |
| R-GS-063 | G-A1-07 | 交互 | 点击（有命中） | 当前项前移一位；时间线滚动到该命中并高亮（G-A5-02 迁移） | — | 判据 = 底色 选中底 明 #f5f5f5 / 暗 #262626；字色 主文字 明 #0a0a0a / 暗 #ececec；其余 = 选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED） | 未实现（映射到位但没有该维度的标记）：G-A1-07→internal/ui/search.go:App.moveSessionSearchHit；测试提及 internal/ui/reconcile_scope_test.go:包级 |
| R-GS-064 | G-A1-07 | 交互 | 在第 1 条点上一个 | 循环到末条（或停在首条，按实现一致），不越界成空 | — | 计数文本 ≠「0 / 35 个结果」（循环到末条时） | 未实现（映射到位但没有该维度的标记）：G-A1-07→internal/ui/search.go:App.moveSessionSearchHit；测试提及 internal/ui/reconcile_scope_test.go:包级 |
| R-GS-065 | G-A1-07 | 状态 | 0 命中（C25） | 按钮禁用（图标色变淡，JSON 无此维度，须读 PNG） | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-A1-07→internal/ui/search.go:App.moveSessionSearchHit；测试提及 internal/ui/reconcile_scope_test.go:包级 |
| R-GS-066 | G-A1-07 | 样式 | 图标按钮在屏上时 | 28px 热区、无底色；禁用态仅降对比 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；文字与底色对比度 ≥ 4.5:1（正文）、≥ 3:1（大字与图标）（CL-CONTRAST） | 未实现（映射到位但没有该维度的标记）：G-A1-07→internal/ui/search.go:App.moveSessionSearchHit；测试提及 internal/ui/reconcile_scope_test.go:包级 |
| R-GS-067 | G-A1-07 | 无障碍 | 读屏进入该元素时 | 读作按钮「上一个搜索结果」，禁用时播报不可用 | — | 读屏文本 = 「上一个搜索结果」 | 已实现（逐字）：internal/ui/search.go:App.sessionSearchResultRow「上一个搜索结果」 |
| R-GS-068 | G-A1-07 | 键盘 | Enter（焦点在此） | 等同点击 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 未实现（映射到位但没有该维度的标记）：G-A1-07→internal/ui/search.go:App.moveSessionSearchHit；测试提及 internal/ui/reconcile_scope_test.go:包级 |
| R-GS-069 | G-A1-07 | 状态 | 只有 1 条命中 | 按钮可用（可循环到自身），不置灰 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-A1-07→internal/ui/search.go:App.moveSessionSearchHit；测试提及 internal/ui/reconcile_scope_test.go:包级 |
| R-GS-070 | G-A1-08 | 状态 | 查询词非空时 | 「下一个搜索结果」按钮（↓，25×25，实测 357,61） | — | 按钮 25×25，与上一个间距 10 | 已实现（逐字）：internal/ui/search.go:App.sessionSearchResultRow「下一个搜索结果」 |
| R-GS-071 | G-A1-08 | 交互 | 鼠标点击该元素时 | 当前项后移一位；时间线滚动跟随 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-A1-08→internal/ui/search.go:App.moveSessionSearchHit |
| R-GS-072 | G-A1-08 | 交互 | 在最后一条点下一个 | 循环到第 1 条 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-A1-08→internal/ui/search.go:App.moveSessionSearchHit |
| R-GS-073 | G-A1-08 | 状态 | 0 命中 | 禁用 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-A1-08→internal/ui/search.go:App.moveSessionSearchHit |
| R-GS-074 | G-A1-08 | 样式 | 前后箭头按钮同屏时 | 同尺寸同层级，仅图标朝向不同 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0） | 未实现（映射到位但没有该维度的标记）：G-A1-08→internal/ui/search.go:App.moveSessionSearchHit |
| R-GS-075 | G-A1-08 | 无障碍 | 读屏进入该元素时 | 读作按钮「下一个搜索结果」 | — | 读屏文本 = 「下一个搜索结果」 | 已实现（逐字）：internal/ui/search.go:App.sessionSearchResultRow「下一个搜索结果」 |
| R-GS-076 | G-A1-08 | 键盘 | 按 Enter 时 | 等同点击 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 未实现（映射到位但没有该维度的标记）：G-A1-08→internal/ui/search.go:App.moveSessionSearchHit |
| R-GS-077 | G-A1-08 | 状态 | 只有 1 条命中 | 同上：可用且点击后仍停在该条 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-A1-08→internal/ui/search.go:App.moveSessionSearchHit |
| R-GS-078 | G-A1-09 | 内容 | C25 无命中 | 结果行右侧文本逐字「0 个结果」（实测 664,64 48×18） | — | 屏上文案逐字为「0 个结果」 | 已实现（逐字）：internal/ui/search.go:App.sessionSearchResultRow「0 个结果」 |
| R-GS-079 | G-A1-09 | 内容 | C26 有命中 | 逐字「1 / 35 个结果」（实测 641,64 71×18），格式为「{当前} / {总数} 个结果」 | — | 屏上文案逐字为「1 / 35 个结果」 | 已实现（映射 = G-A1-09→internal/ui/search.go:App.sessionSearchResultRow；标记 = 文案「搜索结果」等 11 条） |
| R-GS-080 | G-A1-09 | 内容 | 总数很大（如 253） | 数字原样显示，不加缩写；文本左扩、右缘贴浮层内边距 | — | 屏上为「1 / 253 个结果」 | 已实现（映射 = G-A1-09→internal/ui/search.go:App.sessionSearchResultRow；标记 = 文案「搜索结果」等 11 条） |
| R-GS-081 | G-A1-09 | 状态 | 检索进行中 | 文本保持上次值，不闪「0 个结果」 | — | 计数文本 ≠「0 个结果」（进行中保持上次值） | 已实现（逐字）：internal/ui/search.go:App.sessionSearchResultRow「0 个结果」 |
| R-GS-082 | G-A1-09 | 关联 | 用箭头切换当前结果时 | 斜杠前的数字等于当前定位序号（1 基），随 G-A1-07/08 同步 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-A1-09→internal/ui/search.go:App.sessionSearchResultRow |
| R-GS-083 | G-A1-09 | 样式 | 结果计数文本与结果行同屏时 | 12–13px 次要色、等宽数字；容器标 `aria-live=polite` | — | 判据 = 字号 代码 13px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-A1-09→internal/ui/search.go:App.sessionSearchResultRow；标记 = Height） |
| R-GS-084 | G-A1-09 | 无障碍 | 读屏进入该元素时 | 结果变化时朗读一次计数（polite，不打断输入） | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-A1-09→internal/ui/search.go:App.sessionSearchResultRow；标记 = Label） |
| R-GS-085 | G-A1-09 | 异常 | 命中数超过上限 | 显示「{当前} / {上限} 个结果」，不显示「更多」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |

### 2.2 搜索聊天对话框（G-A2-01–G-A2-14）

条件：全局入口唤起（左栏「搜索」行 / `⌘G`），模态，无右上关闭按钮；最多 8 条结果，按置顶/最近分组。已采 Q（空结果）；S/K 同族（背景不同）。

| 需求 ID | 元素 ID | 维度 | 场景 | 期望 | 异常 | 验收 | 状态 |
|---|---|---|---|---|---|---|---|
| R-GS-086 | G-A2-01 | 状态 | Q：按 `⌘G` 或点左栏「搜索」 | 模态对话框出现：实测 291,293 448×139，`max-w-xl`、无内边距、无右上关闭按钮 | — | 屏上出现 1 个 `role=dialog`，包围盒 291,293 448×139 | 已实现（映射 = G-A2-01→internal/ui/search_chat.go:App.chatSearchCatalog；标记 = Loading） |
| R-GS-087 | G-A2-01 | 关联 | 搜索聊天对话框打开、背景时间线仍在时 | 背景被遮罩压暗，遮罩内不可交互 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A2-01→internal/ui/search_chat.go:App.chatSearchDialog；标记 = Screen） |
| R-GS-088 | G-A2-01 | 交互 | 按 Esc 时 | 关闭对话框，焦点回到唤起前的位置 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-A2-01→internal/ui/search_chat.go:App.chatSearchFailedRow；标记 = Clicked） |
| R-GS-089 | G-A2-01 | 状态 | 已有对话框时再唤起 | 不新开第二个；聚焦已有输入框 | — | 对话框计数恒为 1 | 已实现（映射 = G-A2-01→internal/ui/search_chat.go:App.chatSearchCatalog；标记 = Loading） |
| R-GS-090 | G-A2-01 | 样式 | 分别切到浅色与深色主题时 | 圆角 + 阴影 + 上层底色；深色主题下描边更明显 | — | 判据 = 圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px；底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d；其余 = 色值取 token：主色 #349651/#63cb7b、正文 #0a0a0a/#ececec、次文 #737373/#8a8a8a、边框 #e5e5e5/#2e2e2e、警示 #e17100/#ffb900，偏差 0（CL-ACCENT / CL-TEXT / CL-TEXT-2 / CL-BORDER / CL-WARN） | 已实现（映射 = G-A2-01→internal/ui/search_chat.go:App.chatSearchFailedRow；标记 = Height） |
| R-GS-091 | G-A2-01 | 无障碍 | 读屏进入该元素时 | 作为模态对话框，焦点进入框内并困住 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-A2-01→internal/ui/search_chat.go:App.chatSearchFailedRow；标记 = Role） |
| R-GS-092 | G-A2-01 | 异常 | 会话列表接口失败 | 对话框保留并显示空态或错误提示，不空白无反馈 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-A2-01→internal/ui/search_chat.go:App.closeChatSearch；标记 = Reason） |
| R-GS-093 | G-A2-01 | 记忆 | 关闭后再次打开时 | 不记忆上次查询（每次打开为空、聚焦输入框） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-A2-01→internal/ui/search_chat.go:App.closeChatSearch；测试提及 internal/ui/a11y_screens_test.go:a11yScreens |
| R-GS-094 | G-A2-01 | 窄窗 | 窗口收窄到最窄宽度时 | 宽按 `min(max-w-xl, 100vw-…)` 收缩，不溢出 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-A2-01→internal/ui/search_chat.go:App.closeChatSearch；测试提及 internal/ui/a11y_screens_test.go:a11yScreens |
| R-GS-095 | G-A2-02 | 内容 | 打开「搜索聊天」对话框时 | 标题「搜索聊天」与描述「搜索并打开已有聊天」为 `sr-only`：屏上不可见，仅供读屏 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（逐字）：internal/ui/search.go:sessionSearchPlaceholder「搜索聊天」 |
| R-GS-096 | G-A2-02 | 无障碍 | 读屏进入该元素时 | 标题作为对话框名、描述作为说明被读出 | — | 读屏文本 = 「搜索聊天」 | 已实现（映射 = G-A2-02→internal/ui/search_chat.go:App.chatSearchDialog；标记 = Role） |
| R-GS-097 | G-A2-02 | 状态 | 任何状态 | 两者恒定 `sr-only`，不在任何状态下显形 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-A2-02→internal/ui/search_chat.go:App.chatSearchDialog；标记 = Loading@App.chatSearchResults） |
| R-GS-098 | G-A2-02 | 样式 | 读屏与 JSON 的 `label` 列比对时 | JSON 里 `w=1`（`sr-only`），宽度不可用于布局判断 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-A2-02→internal/ui/search_chat.go:App.chatSearchDialog；标记 = Width） |
| R-GS-099 | G-A2-02 | 关联 | 标题描述与输入框占位同时存在时 | 标题描述与输入框占位是三段不同文本，不互相顶替 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-A2-02→internal/ui/search_chat.go:App.chatSearchDialog；标记 = Screen） |
| R-GS-100 | G-A2-02 | 内容 | 采集态 Q | JSON 里 h2 与 p 的 `w=1`、`h` 分别为 16/18，属 `sr-only` | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 已实现（映射 = G-A2-02→internal/ui/search_chat.go:App.chatSearchDialog；标记 = 文案「搜索聊天」等 2 条） |
| R-GS-101 | G-A2-02 | 状态 | 结果区加载中 | 标题与描述不随状态变化（始终仅读屏） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-A2-02→internal/ui/search_chat.go:App.chatSearchDialog；标记 = Loading@App.chatSearchResults） |
| R-GS-102 | G-A2-02 | 关联 | 无命中、对话框内只剩空态文案时 | 空态文案在屏上，标题描述不在屏上，二者不冲突 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-A2-02→internal/ui/search_chat.go:App.chatSearchDialog；标记 = Screen） |
| R-GS-103 | G-A2-03 | 状态 | 打开「搜索聊天」对话框时 | 输入框左端显示搜索图标（16px），位于占位文字左侧 ≈300,312 | — | 判据 = 字号 UI 正文 12.25px；字重 常规 400 / 中粗 500 / 半粗 600 / 粗 700 | 已实现（映射 = G-A2-03→internal/ui/search_chat.go:App.chatSearchInputRow；标记 = Hidden@iconGlyph） |
| R-GS-104 | G-A2-03 | 样式 | 分别切到浅色与深色主题时 | 次要色 16px 描边图标 | — | 判据 = 控件尺寸 图标 16 / 14 / 12px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-A2-03→internal/ui/search_chat.go:App.chatSearchInputRow；标记 = Height） |
| R-GS-105 | G-A2-03 | 交互 | 鼠标点击该图标时 | 穿透到输入框并聚焦 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A2-03→internal/ui/search_chat.go:App.chatSearchInputRow；标记 = Pressed） |
| R-GS-106 | G-A2-03 | 无障碍 | 读屏进入该元素时 | 不单独播报 | — | 播报次数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-107 | G-A2-03 | 高DPI | 在 DPR=2 的屏幕上时 | 2× 下清晰 | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-A2-03→internal/ui/search_chat.go:App.chatSearchInputRow；标记 = Scale@fontStep.sizeIn） |
| R-GS-108 | G-A2-03 | 异常 | 图标资源缺失 | 用通用放大镜兜底，无破图 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-A2-03→internal/ui/search_chat.go:App.chatSearchInputRow |
| R-GS-109 | G-A2-03 | 内容 | 输入框有内容 | 图标不随文字滚动，贴在框左端内侧 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-A2-03→internal/ui/search_chat.go:App.chatSearchInputRow；标记 = 文案「搜索聊天」等 2 条） |
| R-GS-110 | G-A2-03 | 状态 | 结果为空 | 图标仍显示（不换成告警图标） | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-A2-03→internal/ui/search_chat.go:App.chatSearchInputRow；标记 = Hidden@iconGlyph） |
| R-GS-111 | G-A2-04 | 内容 | 打开「搜索聊天」对话框时 | 输入框占位逐字「搜索聊天」（不入 JSON，PNG 可见）；实测 331,294 394×48（模板声明 56，实测 48） | — | 屏上占位逐字为「搜索聊天」；包围盒 331,294 394×48 | 已实现（逐字）：internal/ui/search.go:sessionSearchPlaceholder「搜索聊天」 |
| R-GS-112 | G-A2-04 | 状态 | 打开即聚焦 | 光标在框内，无边框高亮（无边框设计） | — | 判据 = 底色 选中底 明 #f5f5f5 / 暗 #262626；字色 主文字 明 #0a0a0a / 暗 #ececec；其余 = 选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED） | 已实现（映射 = G-A2-04→internal/ui/search_chat.go:App.chatSearchInputRow；标记 = Hidden@iconGlyph） |
| R-GS-113 | G-A2-04 | 交互 | 在输入框键入时 | 逐字过滤会话；结果最多 8 条 | — | 结果数 ≤8 | 已实现（映射 = G-A2-04→internal/ui/search_chat.go:App.chatSearchInputRow；标记 = Pressed） |
| R-GS-114 | G-A2-04 | 键盘 | 按 ↑ 或 ↓ 时 | 在结果间移动选中项（G-A2-13），输入框内容不变 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-A2-04→internal/ui/search_chat.go:App.chatSearchInputRow；标记 = Key） |
| R-GS-115 | G-A2-04 | 键盘 | 按 Enter 时 | 打开当前选中会话并关闭对话框 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-A2-04→internal/ui/search_chat.go:App.chatSearchInputRow；标记 = Key） |
| R-GS-116 | G-A2-04 | 键盘 | ⌘1–⌘8 / Ctrl+1–8 | 直接打开对应序号结果（见 G-A2-12） | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-A2-04→internal/ui/search_chat.go:App.chatSearchInputRow；标记 = Key） |
| R-GS-117 | G-A2-04 | 内容 | 查询超长 | 单行横向滚动，框高恒 48 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-A2-04→internal/ui/search_chat.go:App.chatSearchInputRow；标记 = 文案「搜索聊天」等 2 条） |
| R-GS-118 | G-A2-04 | 无障碍 | 读屏进入该元素时 | 读作搜索输入框，值为当前查询 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-A2-04→internal/ui/search_chat.go:App.chatSearchInputRow；标记 = Label） |
| R-GS-119 | G-A2-04 | 异常 | 查询无命中 | 结果区显示「未找到聊天」（G-A2-14），输入框保留查询 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/search_chat.go:chatSearchEmptyText「未找到聊天」 |
| R-GS-120 | G-A2-05 | 状态 | 未观察（条件）：当前页结果未就绪（`role=status`，旋转图标 + 文字） | 结果区显示「正在加载...」，三点为英文句点 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/ui/search_chat.go:chatSearchLoadingText「正在加载...」 |
| R-GS-121 | G-A2-05 | 动画 | 加载进行中时 | 旋转图标连续旋转；结果返回后整行消失 | 减少动效下静态 | 判据 = 动效 布局 250ms / 颜色 150ms / 折叠 300ms；缓动 cubic-bezier(0.16, 1, 0.3, 1) / cubic-bezier(0.4, 0, 0.2, 1)；其余 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 未实现（映射到位但没有该维度的标记）：G-A2-05→internal/ui/search_chat.go:App.chatSearchResults；测试提及 internal/ui/search_chat_test.go:包级 |
| R-GS-122 | G-A2-05 | 关联 | 检索进行中、尚无命中时 | 加载中不显示「未找到聊天」（避免先空后有的闪烁） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-123 | G-A2-05 | 样式 | 状态行在屏上时 | 居中、12–13px 次要色、上下留白 8 | — | 判据 = 字号 代码 13px；内边距 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28）/ 卡片外框 4.5；档位 3.5·7·10.5·14·21·28px；其余 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A2-05→internal/ui/search_chat.go:App.chatSearchResults；标记 = Height） |
| R-GS-124 | G-A2-05 | 无障碍 | 读屏进入该元素时 | `role=status` 播报一次 | — | 播报次数 = 1 | 已实现（映射 = G-A2-05→internal/ui/search_chat.go:App.chatSearchResults；标记 = Label） |
| R-GS-125 | G-A2-05 | 异常 | 加载失败 | 换成空态或错误提示，不停在加载中 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A2-05→internal/ui/search_chat.go:App.chatSearchResults；标记 = Reason） |
| R-GS-126 | G-A2-05 | 内容 | 首屏结果瞬时返回 | 该行可能只出现 1 帧；不得残留成常驻文案 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A2-05→internal/ui/search_chat.go:App.chatSearchResults；标记 = 文案「正在加载搜索聊天」等 3 条） |
| R-GS-127 | G-A2-05 | 样式 | 加载行与空态文案先后出现时 | 同为居中文案，但一行带旋转图标、一行不带 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A2-05→internal/ui/search_chat.go:App.chatSearchResults；标记 = Height） |
| R-GS-128 | G-A2-06 | 状态 | 未观察（条件）：已翻到第 2 页之后 | 结果列表顶部出现整行「上一页」文字按钮（32px 高） | — | 判据 = 字号 UI 正文 12.25px；字重 常规 400 / 中粗 500 / 半粗 600 / 粗 700 | 已实现（逐字）：internal/ui/search_chat.go:chatSearchPrevText「上一页」 |
| R-GS-129 | G-A2-06 | 交互 | 鼠标点击该元素时 | 回到上一页；选中项重置为第 1 条 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A2-06→internal/ui/search_chat.go:App.chatSearchTextRow；标记 = Clicked） |
| R-GS-130 | G-A2-06 | 状态 | 已在第 1 页 | 按钮不出现 | — | 屏上 文字按钮 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-131 | G-A2-06 | 样式 | 整行文字按钮在屏上时 | 左对齐、12–13px 次要色、悬停浅底 | — | 判据 = 行高 UI 正文 17.5px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-A2-06→internal/ui/search_chat.go:App.chatSearchTextRow；标记 = Height） |
| R-GS-132 | G-A2-06 | 无障碍 | 读屏进入该元素时 | 读作按钮「上一页」 | — | 读屏文本 = 「上一页」 | 已实现（逐字）：internal/ui/search_chat.go:chatSearchPrevText「上一页」 |
| R-GS-133 | G-A2-06 | 键盘 | 按 Enter 时 | 等同点击 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 未实现（映射到位但没有该维度的标记）：G-A2-06→internal/ui/search_chat.go:App.chatSearchTextRow；测试提及 internal/ui/search_chat_test.go:包级 |
| R-GS-134 | G-A2-06 | 内容 | 上一页进行中 | 按钮短暂禁用，列表不闪空 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A2-06→internal/ui/search_chat.go:App.chatSearchResults；标记 = 文案「正在加载搜索聊天」等 3 条） |
| R-GS-135 | G-A2-06 | 样式 | 行首与行尾边界处 | 为列表顶部第一行（在分组标题之上） | — | 判据 = 该行在分组标题之上：该行 y + 行高 ≤ 分组标题 y + 1px，出现次数 1（AL-STACK / AL-TOL-1） | 已实现（映射 = G-A2-06→internal/ui/search_chat.go:App.chatSearchTextRow；标记 = Height） |
| R-GS-136 | G-A2-07 | 状态 | 未观察（条件）：还有下一页（结果 >8 条） | 结果列表底部出现整行「显示更多」文字按钮（32px 高） | — | 判据 = 字号 UI 正文 12.25px；字重 常规 400 / 中粗 500 / 半粗 600 / 粗 700 | 已实现（逐字）：internal/ui/search_chat.go:chatSearchMoreText「显示更多」 |
| R-GS-137 | G-A2-07 | 交互 | 鼠标点击该元素时 | 追加下一页；已显示项不重排 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-A2-07→internal/ui/search_chat.go:App.chatSearchTextRow；标记 = Clicked） |
| R-GS-138 | G-A2-07 | 状态 | 已到末页 | 按钮消失 | — | 屏上 文字按钮 计数 = 0 | 已实现（映射 = G-A2-07→internal/ui/search_chat.go:App.chatSearchResults；标记 = Loading） |
| R-GS-139 | G-A2-07 | 关联 | 结果超过一页、上一页按钮同时在屏上时 | 两者可同屏（上方上一页、下方显示更多） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-A2-07→internal/ui/search_chat.go:App.chatSearchResults；标记 = Select@App.chatSearchHitRow） |
| R-GS-140 | G-A2-07 | 样式 | 整行文字按钮在屏上时 | 与「上一页」同款，居中或左对齐一致 | — | 判据 = 行高 UI 正文 17.5px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（逐字）：internal/ui/search_chat.go:chatSearchPrevText「上一页」 |
| R-GS-141 | G-A2-07 | 无障碍 | 读屏进入该元素时 | 读作按钮「显示更多」 | — | 读屏文本 = 「显示更多」 | 已实现（逐字）：internal/ui/search_chat.go:chatSearchMoreText「显示更多」 |
| R-GS-142 | G-A2-07 | 内容 | 加载下一页 | 按钮短暂禁用，已显示项不重排 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A2-07→internal/ui/search_chat.go:App.chatSearchResults；标记 = 文案「正在加载搜索聊天」等 3 条） |
| R-GS-143 | G-A2-07 | 样式 | 行首与行尾边界处 | 为列表底部最后一行（在结果行之下） | — | 判据 = 该行在结果行之下，y 差 = 相邻块档（3.5 / 7 / 10.5 / 14 / 21 之一），偏差 ≤ 1px（AL-STACK / AL-TOL-1） | 已实现（映射 = G-A2-07→internal/ui/search_chat.go:App.chatSearchTextRow；标记 = Height） |
| R-GS-144 | G-A2-08 | 内容 | 未观察（条件）：该分组有命中 | 分组标题「置顶聊天」在上、「最近聊天」在下，12px 灰字 | — | 判据 = 字号 UI 正文 12.25px；字重 常规 400 / 中粗 500 / 半粗 600 / 粗 700 | 已实现（逐字）：internal/ui/search_chat.go:chatSearchGroupLabels「置顶聊天」 |
| R-GS-145 | G-A2-08 | 状态 | 某组无命中 | 该组标题不渲染（不显示空组） | — | 屏上 空组 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-146 | G-A2-08 | 关联 | 两个分组标题同时在场时 | 置顶组排在最近组之前，顺序固定 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM） | 已实现（映射 = G-A2-08→internal/ui/search_chat.go:App.chatSearchResults；标记 = Select@App.chatSearchHitRow） |
| R-GS-147 | G-A2-08 | 样式 | 分组标题在屏上时 | 12px、次要色、左侧内边距与结果行文字对齐 | — | 判据 = 字号 说明文字 10.5px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a；内边距 左缘 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28） | 已实现（映射 = G-A2-08→internal/ui/search_chat.go:App.chatSearchResults；标记 = Height） |
| R-GS-148 | G-A2-08 | 无障碍 | 读屏进入该元素时 | 作为分组标题读出，行归入组 | — | 读屏文本 = 「置顶聊天」 | 已实现（映射 = G-A2-08→internal/ui/search_chat.go:App.chatSearchResults；标记 = Label） |
| R-GS-149 | G-A2-08 | 异常 | 两组都无命中 | 两个标题都不出现，只留空态 | — | 屏上 分组标题 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-150 | G-A2-08 | 状态 | 两组都有命中 | 两个标题都显示：置顶在上、最近在下 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM） | 已实现（映射 = G-A2-08→internal/ui/search_chat.go:App.chatSearchResults；标记 = Loading） |
| R-GS-151 | G-A2-08 | 异常 | 某组命中被排除后为空 | 该组标题消失，另一组不受影响 | — | 屏上 分组标题 计数 = 0 | 已实现（映射 = G-A2-08→internal/ui/search_chat.go:App.chatSearchResults；标记 = Reason） |
| R-GS-152 | G-A2-09 | 内容 | 未观察（条件）：有结果 | 结果行主行显示会话标题（14px 半粗），超长 truncate | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-A2-09→internal/ui/search_chat.go:App.chatSearchHitRow；标记 = 文案「已置顶」等 3 条@App.chatSearchRowIcon） |
| R-GS-153 | G-A2-09 | 内容 | 标题为空 | 显示「未命名会话」，不留空行 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/search_chat.go:chatSearchUnnamedText「未命名会话」 |
| R-GS-154 | G-A2-09 | 状态 | 选中 | 主行随整行进入强调底 + 强调前景 | — | 判据 = 底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d；字色 主文字 明 #0a0a0a / 暗 #ececec | 已实现（映射 = G-A2-09→internal/ui/search_chat.go:App.chatSearchHitRow；标记 = Hidden@App.chatSearchRowIcon） |
| R-GS-155 | G-A2-09 | 样式 | 主行与副行同屏排列时 | 主行 14px 半粗、主前景；副行 12px 次要 | — | 判据 = 字号 正文 14px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-A2-09→internal/ui/search_chat.go:App.chatSearchHitRow；标记 = Height） |
| R-GS-156 | G-A2-09 | 无障碍 | 读屏进入该元素时 | 播报会话标题 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-A2-09→internal/ui/search_chat.go:App.chatSearchHitRow；标记 = Label） |
| R-GS-157 | G-A2-09 | 异常 | 标题含不可见字符 | 按可见字符显示 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-A2-09→internal/ui/search_chat.go:App.chatSearchHitRow；测试提及 internal/ui/search_chat_test.go:包级 |
| R-GS-158 | G-A2-09 | 状态 | 该行数据加载中 | 主行显示占位（骨架或空白），不留 literal「加载中」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/composer_extra.go:modelEmptyState「加载中」 |
| R-GS-159 | G-A2-09 | 状态 | 该行加载失败 | 主行显示会话 id 或「无法加载」，不空白 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalCopyMemberErrorTitle「无法加载」 |
| R-GS-160 | G-A2-10 | 内容 | 未观察（条件）：结果所属工作区有名 | 副行显示项目显示名（12px 灰字），超长 truncate | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-A2-10→internal/ui/search_chat.go:App.chatSearchHitRow；标记 = 文案「已置顶」等 3 条@App.chatSearchRowIcon） |
| R-GS-161 | G-A2-10 | 状态 | 工作区无名 | 副行不出现（或留空），不显示原始路径 | — | 屏上 原始路径 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-162 | G-A2-10 | 样式 | 副行紧跟主行渲染时 | 12px 次要色、与主行左缘对齐 | — | 判据 = 字号 UI 正文 12.25px；内边距 左缘 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28） | 已实现（映射 = G-A2-10→internal/ui/search_chat.go:App.chatSearchHitRow；标记 = Height） |
| R-GS-163 | G-A2-10 | 无障碍 | 读屏进入该元素时 | 播报项目名 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-A2-10→internal/ui/search_chat.go:App.chatSearchHitRow；标记 = Label） |
| R-GS-164 | G-A2-10 | 异常 | 路径含本机 home | 只显示显示名，不泄露绝对路径 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-A2-10→internal/ui/search_chat.go:App.chatSearchHitRow；测试提及 internal/ui/search_chat_test.go:包级 |
| R-GS-165 | G-A2-10 | 状态 | 该行加载中 | 副行不出现（避免半截项目名） | — | 屏上 结果行副行 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-166 | G-A2-10 | 状态 | 行加载失败 | 副行不出现，主行承担错误表达 | — | 屏上 结果行副行 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-167 | G-A2-10 | 内容 | 项目名与标题相同 | 仍两行都显示（不合并） | — | 屏上 结果行副行 计数 = 1 | 已实现（映射 = G-A2-10→internal/ui/search_chat.go:App.chatSearchHitRow；标记 = 文案「已置顶」等 3 条@App.chatSearchRowIcon） |
| R-GS-168 | G-A2-11a | 状态 | 未观察（条件）：该行 RPC 失败 | 行首 32px 底托内显示红色告警圈图标 | — | 判据 = 字号 UI 正文 12.25px；字重 常规 400 / 中粗 500 / 半粗 600 / 粗 700 | 已实现（映射 = G-A2-11a→internal/ui/search_chat.go:App.chatSearchRowIcon；标记 = Hidden） |
| R-GS-169 | G-A2-11a | 关联 | 五种行状态分布在不同会话行上时 | 五分支互斥（失败/加载/未读/置顶/默认） | — | 屏上 图标 计数 = 1 | 未实现（映射到位但没有该维度的标记）：G-A2-11a→internal/ui/search_chat.go:App.chatSearchRowIcon |
| R-GS-170 | G-A2-11a | 样式 | 底托在屏上时 | 32px 圆角底托 + 图标；告警为危险色 | — | 判据 = 控件尺寸 图标 16 / 14 / 12px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-A2-11a→internal/ui/search_chat.go:App.chatSearchRowIcon；标记 = Width） |
| R-GS-171 | G-A2-11a | 无障碍 | 读屏进入该元素时 | 播报该行「加载失败」 | — | 读屏文本 = 「加载失败」 | 已实现（逐字）：internal/ui/composer_extra.go:modelEmptyState「加载失败」 |
| R-GS-172 | G-A2-11a | 异常 | 该行稍后恢复 | 图标换成对应状态（未读/置顶/默认） | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0） | 未实现（映射到位但没有该维度的标记）：G-A2-11a→internal/ui/search_chat.go:App.chatSearchRowIcon |
| R-GS-173 | G-A2-11a | 内容 | 该行数据完全失败 | 底托仍是 32px（不留空位导致行错位） | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-A2-11a→internal/ui/search_chat.go:App.chatSearchRowIcon；标记 = 文案「已置顶」等 3 条） |
| R-GS-174 | G-A2-11a | 状态 | 失败行被选中 | 选中态覆盖底托（强调底），图标仍可辨 | — | 判据 = 底色 选中底 明 #f5f5f5 / 暗 #262626；字色 主文字 明 #0a0a0a / 暗 #ececec；其余 = 选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED） | 已实现（映射 = G-A2-11a→internal/ui/search_chat.go:App.chatSearchRowIcon；标记 = Hidden） |
| R-GS-175 | G-A2-11a | 交互 | 点击失败行 | 打开失败会话或给提示，不静默无反应 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-176 | G-A2-11b | 状态 | 未观察（条件）：该行数据加载中 | 底托内显示旋转圈 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GS-177 | G-A2-11b | 动画 | 加载进行中时 | 旋转连续；返回后切到对应状态图标 | 减少动效下静态 | 判据 = 动效 布局 250ms / 颜色 150ms / 折叠 300ms；缓动 cubic-bezier(0.16, 1, 0.3, 1) / cubic-bezier(0.4, 0, 0.2, 1) | 未实现（找不到元素渲染点） |
| R-GS-178 | G-A2-11b | 样式 | 底托在屏上时 | 同 32px 底托，图标次要色 | — | 判据 = 控件尺寸 图标 16 / 14 / 12px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 未实现（找不到元素渲染点） |
| R-GS-179 | G-A2-11b | 无障碍 | 读屏进入该元素时 | 播报「加载中」 | — | 读屏文本 = 「加载中」 | 已实现（逐字）：internal/ui/composer_extra.go:modelEmptyState「加载中」 |
| R-GS-180 | G-A2-11b | 异常 | 长时间不返回 | 停在加载态，不擅自判失败 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（找不到元素渲染点） |
| R-GS-181 | G-A2-11b | 内容 | 加载中行被选中 | 底托与图标同时进入强调底，图标仍可辨 | — | 判据 = 底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d；字色 主文字 明 #0a0a0a / 暗 #ececec | 未实现（找不到元素渲染点） |
| R-GS-182 | G-A2-11b | 状态 | 加载完成但为未读 | 切到未读圆点（不是默认图标） | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0） | 未实现（找不到元素渲染点） |
| R-GS-183 | G-A2-11b | 交互 | 点击加载中行 | 按规则先打开或提示仍在加载，不卡死 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（找不到元素渲染点） |
| R-GS-184 | G-A2-11c | 状态 | 未观察（条件）：该会话未读 | 底托内显示实心圆点 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GS-185 | G-A2-11c | 关联 | 同一行同时满足未读与置顶时 | 未读优先于置顶（同一行只显示一个） | — | 屏上 图标 计数 = 1 | 未实现（找不到元素渲染点） |
| R-GS-186 | G-A2-11c | 样式 | 圆点 | 6–8px 实心圆、强调色 | — | 判据 = 色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a；其余 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（找不到元素渲染点） |
| R-GS-187 | G-A2-11c | 无障碍 | 读屏进入该元素时 | 播报「未读」 | — | 读屏文本 = 「未读」 | 已实现（逐字）：internal/ui/settings_git_screen.go:gitCopyUnread「未读」 |
| R-GS-188 | G-A2-11c | 异常 | 打开后已读 | 会话打开后该行不再标未读 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（找不到元素渲染点） |
| R-GS-189 | G-A2-11c | 状态 | 未读且置顶 | 只显示一种（未读优先），不叠两个图标 | — | 屏上 图标 计数 = 1 | 未实现（找不到元素渲染点） |
| R-GS-190 | G-A2-11c | 内容 | 多条未读 | 每条各自显示圆点，不汇总到一行 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GS-191 | G-A2-11c | 交互 | 点击未读行 | 打开会话后该行圆点消失 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 未实现（找不到元素渲染点） |
| R-GS-192 | G-A2-11d | 状态 | 未观察（条件）：该会话已置顶 | 底托内显示琥珀星标 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-A2-11d→internal/ui/search_chat.go:App.chatSearchRowIcon；标记 = Hidden） |
| R-GS-193 | G-A2-11d | 关联 | 置顶行渲染在分组内时 | 星标行必落在「置顶聊天」组内 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（逐字）：internal/ui/search_chat.go:chatSearchGroupLabels「置顶聊天」 |
| R-GS-194 | G-A2-11d | 样式 | 星标 | 琥珀色、描边或实心一致 | — | 判据 = 色值 次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-A2-11d→internal/ui/search_chat.go:App.chatSearchRowIcon；标记 = Width） |
| R-GS-195 | G-A2-11d | 无障碍 | 读屏进入该元素时 | 播报「已置顶」 | — | 读屏文本 = 「已置顶」 | 已实现（逐字）：internal/ui/search_chat.go:App.chatSearchRowIcon「已置顶」 |
| R-GS-196 | G-A2-11d | 异常 | 取消置顶后 | 星标消失，行可能移出置顶组 | — | 屏上 图标 计数 = 0 | 未实现（映射到位但没有该维度的标记）：G-A2-11d→internal/ui/search_chat.go:App.chatSearchRowIcon；测试提及 internal/ui/search_chat_test.go:包级 |
| R-GS-197 | G-A2-11d | 状态 | 置顶且加载失败 | 显示告警图标（失败优先），不显示星标 | — | 屏上 星标 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-198 | G-A2-11d | 内容 | 置顶组内多行 | 每行都有星标 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 已实现（映射 = G-A2-11d→internal/ui/search_chat.go:App.chatSearchRowIcon；标记 = 文案「已置顶」等 3 条） |
| R-GS-199 | G-A2-11d | 交互 | 点击置顶行 | 打开会话，星标与分组不变 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A2-11d→internal/ui/search_chat.go:App.chatSearchRowIcon；标记 = Pressed） |
| R-GS-200 | G-A2-11e | 状态 | 未观察（条件）：以上皆否 | 底托内显示默认消息图标 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0） | 已实现（映射 = G-A2-11e→internal/ui/search_chat.go:App.chatSearchRowIcon；标记 = Hidden） |
| R-GS-201 | G-A2-11e | 关联 | 兜底 | 任何状态取值异常都落到本分支，不显示空底托 | — | 屏上 空底托 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-202 | G-A2-11e | 样式 | 底托在屏上时 | 32px、中性底、线稿图标 | — | 判据 = 控件尺寸 图标 16 / 14 / 12px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-A2-11e→internal/ui/search_chat.go:App.chatSearchRowIcon；标记 = Width） |
| R-GS-203 | G-A2-11e | 无障碍 | 读屏进入该元素时 | 不播报额外状态（只有标题与项目名） | — | 播报次数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-204 | G-A2-11e | 内容 | 默认图标行的标题为空 | 仍显示默认图标（不空底托） | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-A2-11e→internal/ui/search_chat.go:App.chatSearchRowIcon；标记 = 文案「已置顶」等 3 条） |
| R-GS-205 | G-A2-11e | 状态 | 该行从默认变未读 | 图标切到未读圆点 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0） | 已实现（映射 = G-A2-11e→internal/ui/search_chat.go:App.chatSearchRowIcon；标记 = Hidden） |
| R-GS-206 | G-A2-11e | 交互 | 点击默认图标行 | 打开会话 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A2-11e→internal/ui/search_chat.go:App.chatSearchRowIcon；标记 = Pressed） |
| R-GS-207 | G-A2-11e | 样式 | 底托色 | 中性底色 + 线稿图标；选中行转强调底 | — | 判据 = 底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d | 已实现（映射 = G-A2-11e→internal/ui/search_chat.go:App.chatSearchRowIcon；标记 = Width） |
| R-GS-208 | G-A2-12 | 内容 | 未观察（条件）：有结果行 | 行尾快捷键提示：macOS 为「⌘{n}」、其它平台「Ctrl+{n}」，n 为 1–8 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-A2-12→internal/ui/search_chat.go:App.chatSearchHitRow；标记 = 文案「已置顶」等 3 条@App.chatSearchRowIcon） |
| R-GS-209 | G-A2-12 | 内容 | 第 9 条之后 | 超过 8 条不显示序号提示（本对话框最多 8 条） | — | 最多 8 个提示 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-210 | G-A2-12 | 关联 | 按下对话框内搜索输入框的快捷键时 | 提示的键位与实际可用的快捷键一致（同一修饰键） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A2-12→internal/ui/search_chat.go:App.chatSearchHitRow；标记 = Select） |
| R-GS-211 | G-A2-12 | 样式 | 快捷键提示与主副行同屏时 | 12px 灰字、字距略松、右对齐 | — | 判据 = 字号 UI 正文 12.25px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；字号 12.25 / 行高 18 / 字重 400，说明文字 10.5 / 14（TX-TYPE-BODY / TX-TYPE-MICRO） | 已实现（映射 = G-A2-12→internal/ui/search_chat.go:App.chatSearchHitRow；标记 = Height） |
| R-GS-212 | G-A2-12 | 无障碍 | 读屏进入该元素时 | 播报「快捷键 ⌘1」类信息 | — | 读屏文本 = 「快捷键 ⌘1」 | 已实现（映射 = G-A2-12→internal/ui/search_chat.go:App.chatSearchHitRow；标记 = Label） |
| R-GS-213 | G-A2-12 | 异常 | 自定义快捷键冲突 | 提示随实际绑定变化，不写死 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-A2-12→internal/ui/search_chat.go:App.chatSearchHitRow；测试提及 internal/ui/search_chat_test.go:包级 |
| R-GS-214 | G-A2-12 | 内容 | 平台为 Linux | 显示「Ctrl+{n}」，不显示 ⌘ | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-215 | G-A2-12 | 状态 | 结果被排除到 8 条以内 | 序号连续（1..n），不留空号 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-A2-12→internal/ui/search_chat.go:App.chatSearchHitRow；标记 = Hidden@App.chatSearchRowIcon） |
| R-GS-216 | G-A2-13 | 状态 | 未观察（条件）：有结果后按 ↑/↓ 或悬停 | 当前行显示强调底 + 强调前景；默认第 1 行为选中 | — | 有且只有 1 行高亮 | 已实现（映射 = G-A2-13→internal/ui/search_chat.go:App.chatSearchHitRow；标记 = Hidden@App.chatSearchRowIcon） |
| R-GS-217 | G-A2-13 | 交互 | 鼠标悬停到另一行时 | 高亮移到悬停行，键盘选中同步 | — | 判据 = 底色 选中底 明 #f5f5f5 / 暗 #262626；字色 主文字 明 #0a0a0a / 暗 #ececec；其余 = 选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A2-13→internal/ui/search_chat.go:App.chatSearchHitRow；标记 = Pressed@App.chatSearchRowIcon） |
| R-GS-218 | G-A2-13 | 交互 | 点击高亮行 | 与 Enter 同：打开该会话并关闭对话框 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A2-13→internal/ui/search_chat.go:App.chatSearchHitRow；标记 = Pressed@App.chatSearchRowIcon） |
| R-GS-219 | G-A2-13 | 动画 | 反复按 ↑ / ↓ 时 | 高亮瞬时迁移（≤80ms），不滚动页面（列表内跟随滚动） | 减少动效下瞬切 | 判据 = 底色 选中底 明 #f5f5f5 / 暗 #262626；字色 主文字 明 #0a0a0a / 暗 #ececec；其余 = 选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；时长 = 期望列给的毫秒值（±0ms），减少动效时 none（§6 动效表） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-220 | G-A2-13 | 样式 | 高亮在屏上时 | 圆角小、强调底 10–15%、文字转强调前景 | — | 判据 = 底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px；其余 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A2-13→internal/ui/search_chat.go:App.chatSearchHitRow；标记 = Height） |
| R-GS-221 | G-A2-13 | 无障碍 | 读屏进入该元素时 | 高亮行作为当前项播报 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-A2-13→internal/ui/search_chat.go:App.chatSearchHitRow；标记 = Label） |
| R-GS-222 | G-A2-13 | 异常 | 结果刷新后原选中项消失 | 高亮回到第 1 行 | — | 判据 = 底色 选中底 明 #f5f5f5 / 暗 #262626；字色 主文字 明 #0a0a0a / 暗 #ececec；其余 = 选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A2-13→internal/ui/search_chat.go:App.chatSearchFrame；标记 = Reason） |
| R-GS-223 | G-A2-13 | 内容 | 高亮行的主副行与图标 | 全部转为强调前景（含副行与图标），不只主行 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A2-13→internal/ui/search_chat.go:App.chatSearchHitRow；标记 = 文案「已置顶」等 3 条@App.chatSearchRowIcon） |
| R-GS-224 | G-A2-13 | 状态 | 结果列表只有 1 行 | 该行默认高亮，↑/↓ 不改变 | — | 判据 = 底色 选中底 明 #f5f5f5 / 暗 #262626；字色 主文字 明 #0a0a0a / 暗 #ececec；其余 = 选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-225 | G-A2-14 | 内容 | Q 态：无任何命中 | 空态文字逐字「未找到聊天」（实测 299,350 432×74），居中、上下留白 32 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（逐字）：internal/ui/search_chat.go:chatSearchEmptyText「未找到聊天」 |
| R-GS-226 | G-A2-14 | 关联 | 检索进行中时 | 加载中不显示本文案 | — | 屏上 本文案 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-227 | G-A2-14 | 样式 | 空态在屏上时 | 14px 级次要色、居中 | — | 判据 = 字号 正文 14px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-A2-14→internal/ui/search_chat.go:App.chatSearchEmptyRow；标记 = Height） |
| R-GS-228 | G-A2-14 | 无障碍 | 读屏进入该元素时 | 读出该句 | — | 读屏文本 = 「未找到聊天」 | 已实现（映射 = G-A2-14→internal/ui/search_chat.go:App.chatSearchEmptyRow；标记 = Label） |
| R-GS-229 | G-A2-14 | 异常 | 输入查询后仍空 | 文案不变（不把查询词写进空态） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-A2-14→internal/ui/search_chat.go:App.chatSearchEmptyRow；测试提及 internal/ui/search_chat_test.go:包级 |
| R-GS-230 | G-A2-14 | 交互 | 点击空态 | 无动作 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-A2-14→internal/ui/search_chat.go:App.chatSearchEmptyRow；测试提及 internal/ui/search_chat_test.go:包级 |
| R-GS-231 | G-A2-14 | 状态 | 查询为空白字符 | 按无结果处理并显示该句（不显示「未找到聊天」以外的词） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-232 | G-A2-14 | 内容 | 查询非空且无命中 | 空态句不嵌查询词（与 A1-11 不同） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-A2-14→internal/ui/search_chat.go:App.chatSearchEmptyRow；标记 = 文案「未找到聊天」） |

### 2.3 `@` 提及弹层（G-A3-01–G-A3-11）

条件：输入框可用且有工作区目录，键入 `@` 后出现；查询非空时合并文件、提交、技能三类候选。已采态里无一份捕获，全部为操作型。

| 需求 ID | 元素 ID | 维度 | 场景 | 期望 | 异常 | 验收 | 状态 |
|---|---|---|---|---|---|---|---|
| R-GS-233 | G-A3-01 | 状态 | 未观察（条件）：输入框可用 ∧ 有 cwd ∧ 键入 `@` | 弹层出现：`max-h-72`、宽 `min(35rem, 100vw-2rem)`，锚在光标上方或下方（空间不足时翻转） | — | 屏上出现 1 个弹层，不溢出窗口 | 已实现（映射 = G-A3-01→internal/ui/mention.go:App.pickPopover；标记 = Loading） |
| R-GS-234 | G-A3-01 | 关联 | 输入框先键入 `@` 再键入 `$` 时 | 二者互斥：`@` 与 `$` 不同时触发两个弹层 | — | 弹层计数为 1 | 已实现（映射 = G-A3-01→internal/ui/mention.go:App.pickPopover；标记 = Open@App.pickRow） |
| R-GS-235 | G-A3-01 | 交互 | 输入空格 / Esc | 弹层关闭，`@` 与已输入前缀保留或按规则清掉（不留半截候选） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A3-01→internal/ui/mention.go:App.pickPopover；标记 = Pressed@App.pickSkeletons） |
| R-GS-236 | G-A3-01 | 动画 | 该元素出现时 | 出现 ≤120ms，锚点不跳 | 减少动效下瞬切 | 判据 = 动效 布局 250ms / 颜色 150ms / 折叠 300ms；缓动 cubic-bezier(0.16, 1, 0.3, 1) / cubic-bezier(0.4, 0, 0.2, 1) | 未实现（映射到位但没有该维度的标记）：G-A3-01→internal/ui/mention.go:pickTriggerOf；测试提及 internal/ui/lint_test.go:lintDividerAllowlist |
| R-GS-237 | G-A3-01 | 样式 | 浮层在屏上时 | 上层底色 + 描边 + 阴影；圆角与主输入区一致 | — | 判据 = 底色 浮层 明 #ffffff / 暗 #222222；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px；其余 = 色值取 token：主色 #349651/#63cb7b、正文 #0a0a0a/#ececec、次文 #737373/#8a8a8a、边框 #e5e5e5/#2e2e2e、警示 #e17100/#ffb900，偏差 0（CL-ACCENT / CL-TEXT / CL-TEXT-2 / CL-BORDER / CL-WARN） | 已实现（映射 = G-A3-01→internal/ui/mention.go:App.pickPopover；标记 = Height） |
| R-GS-238 | G-A3-01 | 无障碍 | 读屏进入该元素时 | 读作候选列表（listbox），可逐项遍历 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-A3-01→internal/ui/mention.go:App.pickPopover；标记 = Label） |
| R-GS-239 | G-A3-01 | 异常 | 无 cwd | 弹层不出现（改由 G-A3-03 的空态文案说明），不留空弹层 | — | 屏上 浮层 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-240 | G-A3-01 | 内容 | `@` 在行首 / 行中 | 两种情况都可触发（不限于行首） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 已实现（映射 = G-A3-01→internal/ui/mention.go:App.pickPopover；标记 = 文案「候选列表：」） |
| R-GS-241 | G-A3-02 | 状态 | 未观察（条件）：候选未返回 | 弹层内显示两条脉冲骨架（32px 高） | — | 骨架 2 条 | 已实现（映射 = G-A3-02→internal/ui/mention.go:App.pickSkeletons；标记 = Hidden） |
| R-GS-242 | G-A3-02 | 动画 | 骨架在屏上时 | 同步脉冲；返回后与候选行同帧替换 | 减少动效下静态 | 判据 = 动效 布局 250ms / 颜色 150ms / 折叠 300ms；缓动 cubic-bezier(0.16, 1, 0.3, 1) / cubic-bezier(0.4, 0, 0.2, 1) | 未实现（映射到位但没有该维度的标记）：G-A3-02→internal/ui/mention.go:App.pickSkeletons |
| R-GS-243 | G-A3-02 | 样式 | 骨架在屏上时 | 中性浅块、与候选行同高 | — | 判据 = 色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-A3-02→internal/ui/mention.go:App.pickSkeletons；标记 = Height） |
| R-GS-244 | G-A3-02 | 无障碍 | 读屏进入该元素时 | 不播报骨架，只播报「正在搜索」一次 | — | 播报次数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-245 | G-A3-02 | 异常 | 搜索失败 | 换成 G-A3-03 的失败文案 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-A3-02→internal/ui/mention.go:App.pickSkeletons |
| R-GS-246 | G-A3-02 | 内容 | 候选很少（1 条） | 骨架仍先出现（按拉取时序），随后替换为 1 行 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 已实现（映射 = G-A3-02→internal/ui/mention.go:App.pickSkeletons；标记 = 文案「正在搜索」） |
| R-GS-247 | G-A3-02 | 状态 | 候选为空 | 骨架之后给空态（G-A3-03），不留在骨架 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM） | 已实现（映射 = G-A3-02→internal/ui/mention.go:App.pickSkeletons；标记 = Hidden） |
| R-GS-248 | G-A3-02 | 交互 | 骨架期间按 Enter | 无候选可选，Enter 不插入任何内容 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A3-02→internal/ui/mention.go:App.pickSkeletons；标记 = Pressed） |
| R-GS-249 | G-A3-03 | 内容 | 未观察（条件）：查询无命中 | 空态文字逐字「未找到文件或目录」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/mention.go:pickNoFileText「未找到文件或目录」 |
| R-GS-250 | G-A3-03 | 内容 | 会话无 cwd | 空态文字逐字「当前无工作区目录」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/mention.go:pickNoWorkspaceText「当前无工作区目录」 |
| R-GS-251 | G-A3-03 | 内容 | 该次搜索 RPC 失败 | 空态文字逐字「文件搜索失败。」（含句号） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/mention.go:pickFileFailedText「文件搜索失败。」 |
| R-GS-252 | G-A3-03 | 状态 | 三分支互斥 | 同一时刻只有一条（按条件优先级：无 cwd > 失败 > 无命中） | — | 文案计数为 1 | 已实现（映射 = G-A3-03→internal/ui/mention.go:App.pickFailedRow；标记 = Disabled） |
| R-GS-253 | G-A3-03 | 样式 | 空态在屏上时 | 14px 次要色、左内边距与候选行文字对齐 | — | 判据 = 字号 正文 14px；内边距 左缘 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28） | 已实现（映射 = G-A3-03→internal/ui/mention.go:App.pickFailedRow；标记 = Height） |
| R-GS-254 | G-A3-03 | 无障碍 | 读屏进入该元素时 | 读出当前空态文案 | — | 读屏文本 = 「未找到文件或目录」 | 已实现（映射 = G-A3-03→internal/ui/mention.go:App.pickFailedRow；标记 = Role） |
| R-GS-255 | G-A3-03 | 状态 | 无 cwd 且查询也失败 | 按优先级显示「当前无工作区目录」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/mention.go:pickNoWorkspaceText「当前无工作区目录」 |
| R-GS-256 | G-A3-03 | 关联 | 无命中、`@` 弹层仍展开时 | 空态出现时弹层仍在（不关闭） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A3-03→internal/ui/mention.go:App.pickFileState；标记 = Open@App.rp） |
| R-GS-257 | G-A3-04 | 内容 | 未观察（条件）：该组有候选 | 分组标题「文件」「技能」，12px 灰字 | — | 判据 = 字号 UI 正文 12.25px；字重 常规 400 / 中粗 500 / 半粗 600 / 粗 700 | 已实现（逐字）：internal/appserver/daemon/process.go:spawnDaemon「文件」 |
| R-GS-258 | G-A3-04 | 状态 | 某组无候选 | 该组标题不渲染 | — | 屏上 分组标题 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-259 | G-A3-04 | 关联 | 文件组与技能组同时在场时 | 文件组在技能组之前（提交类并入文件组所在区段） | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-A3-04→internal/ui/mention.go:pickGroupHeader |
| R-GS-260 | G-A3-04 | 样式 | 分组标题在屏上时 | 12px 次要色，与候选行文字左缘对齐 | — | 判据 = 字号 UI 正文 12.25px；内边距 左缘 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28） | 已实现（映射 = G-A3-04→internal/ui/mention.go:pickGroupHeader；标记 = Height） |
| R-GS-261 | G-A3-04 | 无障碍 | 读屏进入该元素时 | 分组标题先于组内候选 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-A3-04→internal/ui/mention.go:pickGroupHeader；标记 = Label） |
| R-GS-262 | G-A3-04 | 内容 | 提交类候选 | 并入「文件」组显示（不另立第三组） | — | 组数为 2 | 已实现（逐字）：internal/appserver/daemon/process.go:spawnDaemon「文件」 |
| R-GS-263 | G-A3-04 | 状态 | 只有技能候选 | 只显示「技能」组标题 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalActionWord「技能」 |
| R-GS-264 | G-A3-04 | 状态 | 只有文件候选 | 只显示「文件」组标题（C37 无技能时的同规则） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/appserver/daemon/process.go:spawnDaemon「文件」 |
| R-GS-265 | G-A3-05 | 内容 | 未观察（条件）：文件候选 | 候选行主字为文件名（加粗）；目录名追加 `/` | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 已实现（映射 = G-A3-05→internal/ui/mention.go:App.pickRow；标记 = 文案「文件」等 3 条） |
| R-GS-266 | G-A3-05 | 内容 | 文件名超长 | 主字 truncate，`title` 给完整路径 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-A3-05→internal/ui/mention.go:App.pickRow；标记 = 文案「文件」等 3 条） |
| R-GS-267 | G-A3-05 | 交互 | 鼠标点击候选行时 | 把该路径作为提及插入输入框（替换当前 `@前缀`） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A3-05→internal/ui/mention.go:App.pickRow；标记 = Pressed） |
| R-GS-268 | G-A3-05 | 状态 | 选中项 | 行显示强调底 + 强调前景 | — | 判据 = 底色 选中底 明 #f5f5f5 / 暗 #262626；字色 主文字 明 #0a0a0a / 暗 #ececec；其余 = 选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED） | 已实现（映射 = G-A3-05→internal/ui/mention.go:App.pickRow；标记 = Hidden@iconGlyph） |
| R-GS-269 | G-A3-05 | 样式 | 主字 | 14px 加粗主前景；与次级目录形成两级 | — | 判据 = 字号 正文 14px；字重 粗 700 | 已实现（映射 = G-A3-05→internal/ui/mention.go:App.pickRow；标记 = Height） |
| R-GS-270 | G-A3-05 | 无障碍 | 读屏进入该元素时 | 播报名称与「目录/文件」 | — | 读屏文本 = 「目录/文件」 | 已实现（映射 = G-A3-05→internal/ui/mention.go:App.pickRow；标记 = Label） |
| R-GS-271 | G-A3-05 | 状态 | 候选被过滤掉 | 该行整体移除，行高不留空 | — | 屏上 候选行主字 计数 = 0 | 已实现（映射 = G-A3-05→internal/ui/mention.go:App.pickRow；标记 = Hidden@iconGlyph） |
| R-GS-272 | G-A3-05 | 内容 | 目录候选 | 主字为目录名 + 尾部 `/`，可继续追问子目录 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 已实现（映射 = G-A3-05→internal/ui/mention.go:App.pickRow；标记 = 文案「文件」等 3 条） |
| R-GS-273 | G-A3-06 | 状态 | 未观察（条件）：候选位于子目录 | 次级行显示所在目录（12px 灰字，截断） | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-A3-06→internal/ui/mention.go:App.pickRow；标记 = Hidden@iconGlyph） |
| R-GS-274 | G-A3-06 | 状态 | 候选在根目录 | 次级行不出现（不留空行） | — | 屏上 候选行次字 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-275 | G-A3-06 | 样式 | 次级行 | 12px 次要色、与主字左缘对齐 | — | 判据 = 字号 UI 正文 12.25px；内边距 左缘 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28） | 已实现（映射 = G-A3-06→internal/ui/mention.go:App.pickRow；标记 = Height） |
| R-GS-276 | G-A3-06 | 无障碍 | 读屏进入该元素时 | 播报所在目录 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-A3-06→internal/ui/mention.go:App.pickRow；标记 = Label） |
| R-GS-277 | G-A3-06 | 内容 | 候选位于根目录 | 次级行不出现，主字占据整行 | — | 屏上 候选行次字 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-278 | G-A3-06 | 状态 | 次级目录超长 | truncate 加省略号，悬停出完整路径 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-A3-06→internal/ui/mention.go:App.pickRow；标记 = Hidden@iconGlyph） |
| R-GS-279 | G-A3-06 | 内容 | 次级路径为 `.` | 不显示「.」，直接省略该行 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-280 | G-A3-06 | 样式 | 次级行颜色 | 次要色（浅一档），不与主字同色 | — | 判据 = 字色 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-A3-06→internal/ui/mention.go:App.pickRow；标记 = Height） |
| R-GS-281 | G-A3-06 | 记忆 | 关闭后重新打开弹层时 | 根级候选仍无次级行（不记忆上次的次级路径） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-A3-06→internal/ui/mention.go:App.pickRow |
| R-GS-282 | G-A3-07 | 内容 | 未观察（条件）：查询命中 git 提交 | 候选行主字为 7 位短 hash，半粗 | — | 屏上出现 7 位 hash | 已实现（映射 = G-A3-07→internal/ui/mention.go:App.pickEmptyRow；标记 = 文案「未找到技能」等 3 条） |
| R-GS-283 | G-A3-07 | 内容 | hash 冲突（同前缀） | 仍显示短 hash，靠次级标题区分 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-A3-07→internal/ui/mention.go:App.pickEmptyRow；标记 = 文案「未找到技能」等 3 条） |
| R-GS-284 | G-A3-07 | 样式 | 候选行主字与提交类条目同屏时 | 同字号，字重用半粗而非加粗；等宽显示 hash | — | 判据 = 字重 半粗 600；其余 = 字号 12.25 / 行高 18 / 字重 400，说明文字 10.5 / 14（TX-TYPE-BODY / TX-TYPE-MICRO） | 已实现（映射 = G-A3-07→internal/ui/mention.go:App.pickEmptyRow；标记 = Height） |
| R-GS-285 | G-A3-07 | 无障碍 | 读屏进入该元素时 | 逐字读出 hash（不被念成单词） | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-A3-07→internal/ui/mention.go:App.pickEmptyRow；标记 = Label） |
| R-GS-286 | G-A3-07 | 状态 | 提交候选被过滤掉 | 该行整体移除 | — | 屏上 候选行主字 计数 = 0 | 未实现（映射到位但没有该维度的标记）：G-A3-07→internal/ui/mention.go:App.pickEmptyRow |
| R-GS-287 | G-A3-07 | 内容 | 仓库很大（提交很多） | 候选数量受列表上限约束，超出部分需继续键入缩小 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 已实现（映射 = G-A3-07→internal/ui/mention.go:App.pickEmptyRow；标记 = 文案「未找到技能」等 3 条） |
| R-GS-288 | G-A3-07 | 样式 | 等宽文本渲染后 | hash 用等宽字体，7 位宽度一致 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；字号 12.25 / 行高 18 / 字重 400，说明文字 10.5 / 14（TX-TYPE-BODY / TX-TYPE-MICRO） | 已实现（映射 = G-A3-07→internal/ui/mention.go:App.pickEmptyRow；标记 = Height） |
| R-GS-289 | G-A3-07 | 内容 | 短 hash 与另一提交同前缀 | 两者都可出现，靠标题区分 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 已实现（映射 = G-A3-07→internal/ui/mention.go:App.pickEmptyRow；标记 = 文案「未找到技能」等 3 条） |
| R-GS-290 | G-A3-07 | 记忆 | 关闭后重新打开弹层时 | 提交候选重新拉取（不缓存旧 hash） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-A3-07→internal/ui/mention.go:App.pickEmptyRow |
| R-GS-291 | G-A3-08 | 内容 | 未观察（条件）：提交类候选行渲染时 | 次级行显示提交标题（subject），12px 灰字、截断 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（找不到元素渲染点） |
| R-GS-292 | G-A3-08 | 样式 | 次级行紧随主字渲染时 | 12px 次要色、与主字左缘对齐 | — | 判据 = 字号 UI 正文 12.25px；内边距 左缘 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28） | 未实现（找不到元素渲染点） |
| R-GS-293 | G-A3-08 | 无障碍 | 读屏进入该元素时 | 读出提交标题 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 未实现（找不到元素渲染点） |
| R-GS-294 | G-A3-08 | 内容 | 提交标题为空 | 次字不出现（不留空行） | — | 屏上 候选行次字 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-295 | G-A3-08 | 状态 | 被过滤 | 行整体移除 | — | 屏上 候选行次字 计数 = 0 | 未实现（找不到元素渲染点） |
| R-GS-296 | G-A3-08 | 内容 | 标题超长 | 截断加省略号，不折行 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 未实现（找不到元素渲染点） |
| R-GS-297 | G-A3-08 | 样式 | 次级行与主字同屏时 | 12px 次要色、左缘与主字对齐 | — | 判据 = 字号 UI 正文 12.25px；内边距 左缘 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28） | 未实现（找不到元素渲染点） |
| R-GS-298 | G-A3-08 | 异常 | 标题含换行 | 压成单行显示 | — | 行数恒 1 | 未实现（找不到元素渲染点） |
| R-GS-299 | G-A3-09 | 内容 | 未观察（条件）：提交候选 | 候选行右尾小字「{作者} · {日期}」，12px、70% 不透明 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED） | 未实现（找不到元素渲染点） |
| R-GS-300 | G-A3-09 | 内容 | 日期格式 | 相对或短日期（按实现），不显示 ISO 串 | — | 屏上 候选行右尾小字 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-301 | G-A3-09 | 样式 | 右尾 | 12px、70% 不透明、右对齐；不与次级行重叠 | — | 判据 = 字号 UI 正文 12.25px；其余 = 色值取 token：主色 #349651/#63cb7b、正文 #0a0a0a/#ececec、次文 #737373/#8a8a8a、边框 #e5e5e5/#2e2e2e、警示 #e17100/#ffb900，偏差 0（CL-ACCENT / CL-TEXT / CL-TEXT-2 / CL-BORDER / CL-WARN） | 未实现（找不到元素渲染点） |
| R-GS-302 | G-A3-09 | 无障碍 | 读屏进入该元素时 | 读出作者与日期 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 未实现（找不到元素渲染点） |
| R-GS-303 | G-A3-09 | 内容 | 作者缺失 | 只显示日期 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GS-304 | G-A3-09 | 内容 | 日期缺失 | 只显示作者 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GS-305 | G-A3-09 | 状态 | 行很窄（面板 311） | 右尾小字被挤掉时优先保留主字与次字 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（找不到元素渲染点） |
| R-GS-306 | G-A3-09 | 关联 | 作者与日期同处行右侧时 | 两者同处一行右侧，不重叠 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（找不到元素渲染点） |
| R-GS-307 | G-A3-10 | 内容 | 未观察（条件）：已安装技能进入候选 | 主字为技能名、次字为描述；描述缺省时只显示名称 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-A3-10→internal/ui/mention.go:App.pickRow；标记 = 文案「文件」等 3 条） |
| R-GS-308 | G-A3-10 | 内容 | 技能名含命名空间 | 原样显示（不裁前缀） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-A3-10→internal/ui/mention.go:App.pickRow；标记 = 文案「文件」等 3 条） |
| R-GS-309 | G-A3-10 | 样式 | 技能候选与文件候选先后出现时 | 同一套两级样式（主字 + 次字） | — | 判据 = 字号 UI 正文 12.25px | 已实现（映射 = G-A3-10→internal/ui/mention.go:App.pickRow；标记 = Height） |
| R-GS-310 | G-A3-10 | 无障碍 | 读屏进入该元素时 | 播报名称与描述 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-A3-10→internal/ui/mention.go:App.pickRow；标记 = Label） |
| R-GS-311 | G-A3-10 | 内容 | 技能描述为空 | 只显示名称，右尾留空 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-A3-10→internal/ui/mention.go:App.pickRow；标记 = 文案「文件」等 3 条） |
| R-GS-312 | G-A3-10 | 状态 | 技能被过滤 | 行整体移除 | — | 屏上 候选行主字 计数 = 0 | 已实现（映射 = G-A3-10→internal/ui/mention.go:App.pickRow；标记 = Hidden@iconGlyph） |
| R-GS-313 | G-A3-10 | 内容 | 技能名与文件重名 | 分属不同组，各自成行，不合并 | — | 屏上 候选行主字 计数 = 1 | 已实现（映射 = G-A3-10→internal/ui/mention.go:App.pickRow；标记 = 文案「文件」等 3 条） |
| R-GS-314 | G-A3-10 | 样式 | 主字与右尾同屏时 | 名称为主字、描述为右尾小字（12px） | — | 判据 = 字号 UI 正文 12.25px | 已实现（映射 = G-A3-10→internal/ui/mention.go:App.pickRow；标记 = Height） |
| R-GS-315 | G-A3-10 | 记忆 | 关闭后重新打开弹层时 | 技能候选重新拉取 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-A3-10→internal/ui/mention.go:App.pickRow；测试提及 internal/ui/mention_test.go:包级 |
| R-GS-316 | G-A3-11 | 状态 | 未观察（条件）：候选出现后按 ↑/↓ | 选中行显示强调底 + 强调前景，默认第 0 行 | — | 有且只有 1 行高亮 | 已实现（映射 = G-A3-11→internal/ui/mention.go:App.syncPick；标记 = State@pickState.reset） |
| R-GS-317 | G-A3-11 | 交互 | 鼠标悬停到另一行时 | 高亮迁到悬停行 | — | 判据 = 底色 选中底 明 #f5f5f5 / 暗 #262626；字色 主文字 明 #0a0a0a / 暗 #ececec；其余 = 选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A3-11→internal/ui/mention.go:App.syncPick；标记 = Menu） |
| R-GS-318 | G-A3-11 | 交互 | 按 Enter 时 | 插入选中候选并关闭弹层 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A3-11→internal/ui/mention.go:App.syncPick；标记 = Menu） |
| R-GS-319 | G-A3-11 | 样式 | 高亮在屏上时 | 强调底 10–15%、圆角小、文字转强调前景 | — | 判据 = 底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px；其余 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A3-11→internal/ui/composer.go:App.composerCard；标记 = Width） |
| R-GS-320 | G-A3-11 | 无障碍 | 读屏进入该元素时 | 高亮项作为当前项读出 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-A3-11→internal/ui/composer.go:App.composerCard；标记 = Label） |
| R-GS-321 | G-A3-11 | 异常 | 候选刷新后原选中项消失 | 高亮回到第 0 行 | — | 判据 = 底色 选中底 明 #f5f5f5 / 暗 #262626；字色 主文字 明 #0a0a0a / 暗 #ececec；其余 = 选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A3-11→internal/ui/composer.go:App.composerCard；标记 = Reason） |
| R-GS-322 | G-A3-11 | 内容 | 按 End / Home（若支持） | 跳到首/末候选（不支持则不动） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A3-11→internal/ui/composer.go:App.composerCard；标记 = 文案「输入卡」等 5 条） |
| R-GS-323 | G-A3-11 | 状态 | 结果为空 | 无高亮行（不是把空态高亮） | — | 屏上 高亮行 计数 = 0（不是把空态高亮） | 已实现（映射 = G-A3-11→internal/ui/mention.go:App.syncPick；标记 = State@pickState.reset） |

### 2.4 `$` 技能提及弹层（G-A4-01–G-A4-05）

条件：输入框可用且有 suggestion provider，键入 `$` 后出现；只列技能（同名按工作区/用户/插件优先级去重，禁用的不列）。注意：C9 `popover-skills`（输入区技能按钮的浮层）不属于本组。

| 需求 ID | 元素 ID | 维度 | 场景 | 期望 | 异常 | 验收 | 状态 |
|---|---|---|---|---|---|---|---|
| R-GS-324 | G-A4-01 | 状态 | 未观察（条件）：键入 `$` | 弹层出现：`max-h-72`、宽 `min(24rem, 100vw-2rem)` | — | 屏上出现 1 个弹层 | 已实现（映射 = G-A4-01→internal/ui/mention.go:App.pickPopover；标记 = Loading） |
| R-GS-325 | G-A4-01 | 关联 | 先键入 `@` 再键入 `$` 时 | 与 `@` 弹层互斥；同时键入时只有一个生效 | — | 弹层计数为 1 | 已实现（映射 = G-A4-01→internal/ui/mention.go:App.pickPopover；标记 = Open@App.pickRow） |
| R-GS-326 | G-A4-01 | 交互 | Esc / 空格 | 关闭弹层，不留半截候选 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A4-01→internal/ui/mention.go:App.pickPopover；标记 = Pressed@App.pickSkeletons） |
| R-GS-327 | G-A4-01 | 动画 | 该元素出现时 | ≤120ms，锚点不跳 | 减少动效下瞬切 | 判据 = 动效 布局 250ms / 颜色 150ms / 折叠 300ms；缓动 cubic-bezier(0.16, 1, 0.3, 1) / cubic-bezier(0.4, 0, 0.2, 1) | 未实现（映射到位但没有该维度的标记）：G-A4-01→internal/ui/mention.go:pickTriggerOf；测试提及 internal/ui/mention_test.go:包级 |
| R-GS-328 | G-A4-01 | 样式 | 浮层在屏上时 | 与 `@` 弹层同一套底色/描边/阴影，只宽度不同 | — | 判据 = 底色 浮层 明 #ffffff / 暗 #222222；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px | 已实现（映射 = G-A4-01→internal/ui/mention.go:App.pickPopover；标记 = Height） |
| R-GS-329 | G-A4-01 | 无障碍 | 读屏进入该元素时 | 读作候选列表，可遍历 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-A4-01→internal/ui/mention.go:App.pickPopover；标记 = Label） |
| R-GS-330 | G-A4-01 | 异常 | provider 不可用 | 弹层不出现，键入 `$` 当普通字符 | — | 屏上 浮层 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-331 | G-A4-01 | 内容 | `$` 在行中 | 同样触发（不限行首） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 已实现（映射 = G-A4-01→internal/ui/mention.go:App.pickPopover；标记 = 文案「候选列表：」） |
| R-GS-332 | G-A4-02 | 状态 | 未观察（条件）：技能列表未返回 | 弹层内两条脉冲骨架 | — | 骨架 2 条 | 已实现（映射 = G-A4-02→internal/ui/mention.go:App.pickSkeletons；标记 = Hidden） |
| R-GS-333 | G-A4-02 | 动画 | 骨架在屏上时 | 同步脉冲；返回后同帧替换 | 减少动效下静态 | 判据 = 动效 布局 250ms / 颜色 150ms / 折叠 300ms；缓动 cubic-bezier(0.16, 1, 0.3, 1) / cubic-bezier(0.4, 0, 0.2, 1) | 未实现（映射到位但没有该维度的标记）：G-A4-02→internal/ui/mention.go:App.pickSkeletons |
| R-GS-334 | G-A4-02 | 样式 | 骨架在屏上时 | 中性浅块、与候选行同高 | — | 判据 = 色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-A4-02→internal/ui/mention.go:App.pickSkeletons；标记 = Height） |
| R-GS-335 | G-A4-02 | 无障碍 | 读屏进入该元素时 | 不逐条播报骨架 | — | 播报次数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-336 | G-A4-02 | 内容 | 技能很少（1 条） | 骨架先出，随后替换 1 行 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM） | 已实现（映射 = G-A4-02→internal/ui/mention.go:App.pickSkeletons；标记 = 文案「正在搜索」） |
| R-GS-337 | G-A4-02 | 状态 | 技能列表为空 | 骨架后给空态（G-A4-03） | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-A4-02→internal/ui/mention.go:App.pickSkeletons；标记 = Hidden） |
| R-GS-338 | G-A4-02 | 交互 | 骨架期间按 Enter | 不插入任何内容 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A4-02→internal/ui/mention.go:App.pickSkeletons；标记 = Pressed） |
| R-GS-339 | G-A4-02 | 样式 | 骨架在屏上时 | 两条 32px 脉冲块，与候选行同高 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-A4-02→internal/ui/mention.go:App.pickSkeletons；标记 = Height） |
| R-GS-340 | G-A4-03 | 内容 | 未观察（条件）：`$` 后打无命中前缀 | 空态文字逐字「未找到技能」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/mention.go:pickNoSkillText「未找到技能」 |
| R-GS-341 | G-A4-03 | 状态 | 有候选 | 空态不出现 | — | 屏上 空态文字 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-342 | G-A4-03 | 样式 | 空态在屏上时 | 14px 次要色、左内边距与候选对齐 | — | 判据 = 字号 正文 14px；内边距 左缘 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28） | 已实现（映射 = G-A4-03→internal/ui/mention.go:App.pickEmptyRow；标记 = Height） |
| R-GS-343 | G-A4-03 | 无障碍 | 读屏进入该元素时 | 读出该句 | — | 读屏文本 = 「未找到技能」 | 已实现（映射 = G-A4-03→internal/ui/mention.go:App.pickEmptyRow；标记 = Label） |
| R-GS-344 | G-A4-03 | 状态 | 查询非空且有候选 | 空态不出现 | — | 屏上 空态文字 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-345 | G-A4-03 | 内容 | 空态句不嵌查询词 | 固定为「未找到技能」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/mention.go:pickNoSkillText「未找到技能」 |
| R-GS-346 | G-A4-03 | 样式 | 空态在屏上时 | 14px 次要色、左内边距与候选对齐 | — | 判据 = 字号 正文 14px；内边距 左缘 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28） | 已实现（映射 = G-A4-03→internal/ui/mention.go:App.pickEmptyRow；标记 = Height） |
| R-GS-347 | G-A4-03 | 关联 | 无匹配技能、`$` 弹层仍展开时 | 空态在场时弹层不关闭 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-A4-03→internal/ui/mention.go:App.pickEmptyRow；测试提及 internal/ui/mention_test.go:包级 |
| R-GS-348 | G-A4-04 | 内容 | 未观察（条件）：有技能候选 | 主字为技能名、右尾小字为描述；描述缺省时只显示名称 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-A4-04→internal/ui/mention.go:App.pickRow；标记 = 文案「文件」等 3 条） |
| R-GS-349 | G-A4-04 | 内容 | 同名技能多处安装 | 按工作区 > 用户 > 插件优先级去重，只出现一条 | — | 同名只 1 行 | 已实现（映射 = G-A4-04→internal/ui/mention.go:App.pickRow；标记 = 文案「文件」等 3 条） |
| R-GS-350 | G-A4-04 | 内容 | 技能被禁用 | 不进入候选 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 已实现（映射 = G-A4-04→internal/ui/mention.go:App.pickRow；标记 = 文案「文件」等 3 条） |
| R-GS-351 | G-A4-04 | 交互 | 鼠标点击候选行时 | 插入该技能名（替换 `$前缀`）并关闭弹层 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A4-04→internal/ui/mention.go:App.pickRow；标记 = Pressed） |
| R-GS-352 | G-A4-04 | 样式 | 两级 | 主字 14px、右尾 12px 次要色 | — | 判据 = 字号 UI 正文 12.25px | 已实现（映射 = G-A4-04→internal/ui/mention.go:App.pickRow；标记 = Height） |
| R-GS-353 | G-A4-04 | 无障碍 | 读屏进入该元素时 | 播报名称与描述 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-A4-04→internal/ui/mention.go:App.pickRow；标记 = Label） |
| R-GS-354 | G-A4-04 | 状态 | 候选被过滤 | 行整体移除 | — | 屏上 候选行主字 计数 = 0 | 已实现（映射 = G-A4-04→internal/ui/mention.go:App.pickRow；标记 = Hidden@iconGlyph） |
| R-GS-355 | G-A4-04 | 内容 | 无描述 | 只显示名称，右尾留空 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-A4-04→internal/ui/mention.go:App.pickRow；标记 = 文案「文件」等 3 条） |
| R-GS-356 | G-A4-04 | 样式 | 主字与右尾同屏时 | 名称 14px、描述 12px 次要色 | — | 判据 = 字号 UI 正文 12.25px | 已实现（映射 = G-A4-04→internal/ui/mention.go:App.pickRow；标记 = Height） |
| R-GS-357 | G-A4-05 | 状态 | 未观察（条件）：候选出现后按 ↑/↓ | 选中行强调底 + 强调前景，默认第 0 行 | — | 判据 = 底色 选中底 明 #f5f5f5 / 暗 #262626；字色 主文字 明 #0a0a0a / 暗 #ececec；其余 = 选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A4-05→internal/ui/mention.go:App.syncPick；标记 = State@pickState.reset） |
| R-GS-358 | G-A4-05 | 交互 | 按 Enter 时 | 插入选中技能并关闭弹层 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A4-05→internal/ui/mention.go:App.syncPick；标记 = Menu） |
| R-GS-359 | G-A4-05 | 样式 | 高亮在屏上时 | 强调底 10–15%、文字转强调前景 | — | 判据 = 底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d；其余 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A4-05→internal/ui/composer.go:App.composerCard；标记 = Width） |
| R-GS-360 | G-A4-05 | 无障碍 | 读屏进入该元素时 | 当前项被读出 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-A4-05→internal/ui/composer.go:App.composerCard；标记 = Label） |
| R-GS-361 | G-A4-05 | 异常 | 候选列表刷新 | 高亮保持或回第 0 行，不消失成空 | — | 高亮恒有 1 行 | 已实现（映射 = G-A4-05→internal/ui/composer.go:App.composerCard；标记 = Reason） |
| R-GS-362 | G-A4-05 | 状态 | 无候选 | 无高亮行 | — | 屏上 高亮行 计数 = 0 | 已实现（映射 = G-A4-05→internal/ui/mention.go:App.syncPick；标记 = State@pickState.reset） |
| R-GS-363 | G-A4-05 | 内容 | 按 Home / End（若支持） | 跳首/末候选 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A4-05→internal/ui/composer.go:App.composerCard；标记 = 文案「输入卡」等 5 条） |
| R-GS-364 | G-A4-05 | 键盘 | ↑ 在首项 | 停在首项或循环到末项（行为固定即可） | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-A4-05→internal/ui/mention.go:App.syncPick；标记 = Key） |

### 2.5 时间线命中高亮（G-A5-01–G-A5-02）

条件：会话内搜索有查询时，时间线正文内的命中片段被包一层行内标记。已采 C26：同屏两个命中（当前项 + 非当前项）。

本端实现：普通文本用 `RichText` 的 span 底色（其余命中 `colorSearchHit`，当前项 `colorSearchHitCurrent`，圆角取小标签档 4.75）；Markdown 正文把命中包进 `<mark>`（当前项再包一层 `<u>`），包之前跳过围栏代码块、行内代码、链接目标与 HTML 标签里的命中，不破坏原有的块级与行内结构。行内底色没有 1px 描边的能力，当前项靠更深的底色（Markdown 里再加下划线）与其余命中区分。

| 需求 ID | 元素 ID | 维度 | 场景 | 期望 | 异常 | 验收 | 状态 |
|---|---|---|---|---|---|---|---|
| R-GS-365 | G-A5-01 | 状态 | C26：命中但不是当前项 | 命中片段包一层行内高亮：圆角 3px、黄色 45% 底（实测 538,217 28×17，文字「文件」） | — | 屏上出现黄底片段，包围盒 28×17 | 已实现（逐字）：internal/appserver/daemon/process.go:spawnDaemon「文件」 |
| R-GS-366 | G-A5-01 | 关联 | 同一条消息内多个命中、其一被定位时 | 二者互斥：当前定位项用橙底 + 描边，其余用黄底无描边 | — | 屏上 行内高亮 计数 = 1 | 已实现（映射 = G-A5-01→internal/ui/markdown.go:markdownTheme；标记 = Tab） |
| R-GS-367 | G-A5-01 | 状态 | 查询改变 | 旧命中高亮全部清除，按新查询重建 | — | 判据 = 底色 选中底 明 #f5f5f5 / 暗 #262626；字色 主文字 明 #0a0a0a / 暗 #ececec；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED） | 未实现（映射到位但没有该维度的标记）：G-A5-01→internal/ui/search.go:App.searchText |
| R-GS-368 | G-A5-01 | 内容 | 命中片段跨样式（粗体内） | 高亮包在片段外层，不破坏原有字重与行内结构 | — | 判据 = 底色 选中底 明 #f5f5f5 / 暗 #262626；字色 主文字 明 #0a0a0a / 暗 #ececec；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED） | 未实现（映射到位但没有该维度的标记）：G-A5-01→internal/ui/search.go:App.searchText |
| R-GS-369 | G-A5-01 | 样式 | 分别切到浅色与深色主题时 | 黄色半透明底在深色主题下仍可辨（不刺眼、不盖住文字） | — | 判据 = 底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；文字与底色对比度 ≥ 4.5:1（正文）、≥ 3:1（大字与图标）（CL-CONTRAST） | 已实现（映射 = G-A5-01→internal/ui/search.go:App.searchText；标记 = TextColor） |
| R-GS-370 | G-A5-01 | 交互 | 点击高亮片段 | 无动作（不跳转、不切换当前项） | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GS-371 | G-A5-01 | 无障碍 | 读屏进入该元素时 | 命中片段按原文读出，不额外念「高亮」 | — | 读屏文本 = 「高亮」 | 未实现（映射到位但没有该维度的标记）：G-A5-01→internal/ui/search.go:App.searchText |
| R-GS-372 | G-A5-01 | 异常 | 结果数超过上限 | 只高亮已渲染的命中，不为了高亮强制渲染全部 | — | 判据 = 底色 选中底 明 #f5f5f5 / 暗 #262626；字色 主文字 明 #0a0a0a / 暗 #ececec；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED） | 未实现（映射到位但没有该维度的标记）：G-A5-01→internal/ui/search.go:App.searchText |
| R-GS-373 | G-A5-02 | 状态 | C26：当前定位项 | 当前命中片段：橙色 70% 底 + 1px 描边（实测 675,83 28×17）；该段文本被切分成两段（549,83 + 675,83） | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-A5-02→internal/ui/search.go:App.searchReveal；标记 = State@sessionSearchState.currentHit） |
| R-GS-374 | G-A5-02 | 交互 | 点击「下一个」 | 高亮从当前项迁到下一条命中；旧的降为 G-A5-01 | — | 同屏只有 1 个当前项 | 未实现（映射到位但没有该维度的标记）：G-A5-02→internal/ui/search.go:App.searchReveal |
| R-GS-375 | G-A5-02 | 关联 | 用箭头或点击切换当前命中时 | 当前项切换时时间线滚动到该项可见（居中或贴边一致） | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0 | 未实现（映射到位但没有该维度的标记）：G-A5-02→internal/ui/search.go:App.searchReveal |
| R-GS-376 | G-A5-02 | 样式 | 当前项与其余命中同屏时 | 同圆角 3px；多 1px 描边，底色更深 | — | 判据 = 底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d；描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-A5-02→internal/ui/search.go:App.searchText；标记 = TextColor） |
| R-GS-377 | G-A5-02 | 无障碍 | 读屏进入该元素时 | 当前项可被读屏定位（滚动后焦点/朗读跟随） | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-A5-02→internal/ui/search.go:App.searchReveal |
| R-GS-378 | G-A5-02 | 异常 | 当前项所在消息被折叠 | 自动展开该消息后再定位；失败则跳到下一条 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-A5-02→internal/ui/search.go:App.searchReveal |
| R-GS-379 | G-A5-02 | 高DPI | 在 DPR=2 的屏幕上时 | 3px 圆角与 1px 描边在 2× 下平滑 | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-A5-02→internal/ui/search.go:App.searchReveal |
| R-GS-380 | G-A5-02 | 内容 | 同一条消息内多个命中 | 只有当前定位那一个用当前项样式，其余用普通样式 | — | 每条消息至多 1 个当前项 | 未实现（映射到位但没有该维度的标记）：G-A5-02→internal/ui/search.go:App.searchReveal |
| R-GS-381 | G-A5-02 | 状态 | 离开该消息（滚动走） | 当前项样式保留（仍是定位点），不随可见性撤销 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-A5-02→internal/ui/search.go:App.searchReveal；标记 = State@sessionSearchState.currentHit） |
| R-GS-382 | G-A5-02 | 交互 | 点击「下一个」到末尾 | 当前项停在最后一条（不回到第一条，除非按钮循环策略如此） | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-A5-02→internal/ui/search.go:App.searchReveal |

### 2.6 搜索域维度覆盖说明

46 项全部 ≥6 维。G-A2-02（sr-only 标题/描述）到 6 维：无「交互 / 动画」——它不可见、不可点，也没有过渡；其价值在于读屏与验收口径（`w=1` 不能当布局依据）。


---

## 3. 命令（29 元素）

两组：斜杠命令弹层（B1，20）与命令面板基元（B2，9）。B1 已采 C37（10 行，弹层滚动停在 `/compact`）；B2 在 60 份态里没有任何一份打开（仅设置 → 添加模型调色板使用该套基元）。

### 3.1 斜杠命令弹层（G-B1-01–G-B1-20）

条件：输入框可用，行首键入 `/` 后出现；`new`、`login` 两个命令被显式排除，不在应用内出现。C37 实测：弹层 ≈390,320 327×221，候选行 390,y 327×28（y=352/380/408/436/464/492/520/548/576/604），命令名 x=401，描述右对齐。

本端实现：命令清单是**客户端静态表**（`internal/ui/slash_commands.go` 的 `slashCommands`），不依赖服务端（协议 v3 没有命令清单方法）。四组按本端口径排：常用 / 会话 / 设置 / 调试，共 13 条命令；每行是图标 + `/命令名` + 右端说明（不可用时换成原因）+ 真实键位（只有确有绑定的才有，例如 `/settings` 的 ⌘,）。`↑`/`↓` 在可执行项之间移动，`↵` 执行并清空命令文本，`Esc` 收起但保留 `/` 与前缀，继续输入按前缀（大小写不敏感）与模糊匹配过滤；候选超过 8 行在弹层内滚动。涉及写入的两条（`/compact`、`/reload-skills`）走既有 `Intents`，其余命令只改可丢的界面状态（设置屏路由、模式位、右栏标签）。参考产品的「指令 / 技能」两分组与技能候选没有照搬：本端没有技能候选清单进弹层（见 G-B1-06 各行）。

| 需求 ID | 元素 ID | 维度 | 场景 | 期望 | 异常 | 验收 | 状态 |
|---|---|---|---|---|---|---|---|
| R-GM-001 | G-B1-01 | 状态 | C37：输入框行首键入 `/` | 弹层出现：宽 327（`min(24rem,100vw-2rem)`），高受输入框上方空间限制 ≈221（未达 `max-h-72` 的 288） | — | 屏上出现 1 个弹层，宽 327 | 已实现（映射 = G-B1-01→internal/ui/slash_commands.go:App.slashTextKey；标记 = State@pickState.dismiss） |
| R-GM-002 | G-B1-01 | 状态 | 输入框位于窗口顶部（上方空间不足） | 弹层向下展开或高度压缩，不遮挡输入行本身 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-B1-01→internal/ui/slash_commands.go:App.slashTextKey；标记 = State@pickState.dismiss） |
| R-GM-003 | G-B1-01 | 交互 | 继续键入字符 | 弹层按前缀过滤，行数与高度同步变化，锚点不跳 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-B1-01→internal/ui/slash_commands.go:App.slashTextKey；标记 = Key） |
| R-GM-004 | G-B1-01 | 交互 | 按 Esc 时 | 弹层关闭，`/` 与前缀保留在输入框（不吞字） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B1-01→internal/ui/slash_commands.go:App.slashTextKey；标记 = Key） |
| R-GM-005 | G-B1-01 | 交互 | 点击弹层外 | 关闭弹层，不发送消息 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GM-006 | G-B1-01 | 内容 | 候选项多于可视高度 | 弹层内滚动，最大高受 `max-h-72` 约束；滚动不清空选中项 | — | 判据 = 选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED）；滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-B1-01→internal/ui/slash_commands.go:App.slashTextKey；标记 = 文案「不可用：」@App.slashExecuteSelected） |
| R-GM-007 | G-B1-01 | 动画 | 出现 / 关闭 | ≤120ms，无位移；减少动效下瞬切 | 减少动效下瞬切 | 判据 = 动效 布局 250ms / 颜色 150ms / 折叠 300ms；缓动 cubic-bezier(0.16, 1, 0.3, 1) / cubic-bezier(0.4, 0, 0.2, 1) | 未实现（映射到位但没有该维度的标记）：G-B1-01→internal/ui/slash_commands.go:App.slashTextKey；测试提及 internal/ui/lint_test.go:lintDividerAllowlist |
| R-GM-008 | G-B1-01 | 样式 | 浮层在屏上时 | 上层底色 + 描边 + 阴影，圆角与候选行一致 | — | 判据 = 底色 浮层 明 #ffffff / 暗 #222222；描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px | 已实现（映射 = G-B1-01→internal/ui/composer.go:App.composerCard；标记 = Width） |
| R-GM-009 | G-B1-01 | 无障碍 | 读屏进入该元素时 | 读作候选列表（listbox），可逐项遍历 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-B1-01→internal/ui/composer.go:App.composerCard；标记 = Label） |
| R-GM-010 | G-B1-01 | 异常 | 命令与技能列表都拉取失败 | 弹层显示空态（G-B1-03），不留空白浮层 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-B1-01→internal/ui/slash_commands.go:App.slashTextKey；标记 = reason@App.slashExecuteSelected） |
| R-GM-011 | G-B1-02 | 状态 | 未观察（条件）：命令/技能列表首次拉取未返回（窗口 <1s） | 弹层内两条脉冲骨架（32px） | — | 骨架 2 条 | 未实现（找不到元素渲染点） |
| R-GM-012 | G-B1-02 | 动画 | 骨架在屏上时 | 同步脉冲；返回后同帧替换为候选行 | 减少动效下静态 | 判据 = 动效 布局 250ms / 颜色 150ms / 折叠 300ms；缓动 cubic-bezier(0.16, 1, 0.3, 1) / cubic-bezier(0.4, 0, 0.2, 1) | 未实现（找不到元素渲染点） |
| R-GM-013 | G-B1-02 | 样式 | 骨架在屏上时 | 中性浅块、与候选行同高（28） | — | 判据 = 色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（找不到元素渲染点） |
| R-GM-014 | G-B1-02 | 无障碍 | 读屏进入该元素时 | 不逐条播报骨架 | — | 播报次数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GM-015 | G-B1-02 | 关联 | 加载中与返回后交替时 | 骨架之后要么出行，要么出空态，不同时出现 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM） | 未实现（找不到元素渲染点） |
| R-GM-016 | G-B1-02 | 异常 | 拉取失败 | 换成空态文案 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GM-017 | G-B1-02 | 内容 | 命令列表命中缓存 | 骨架不出现（直接出行） | — | 屏上 两条脉冲占位 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GM-018 | G-B1-02 | 状态 | 拉取失败 | 骨架换成空态，不留骨架 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（找不到元素渲染点） |
| R-GM-019 | G-B1-03 | 内容 | 未观察（条件）：`/` 后键入无匹配前缀（如 `/zzzz`） | 空态文字逐字「未找到命令或技能」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/slash_commands.go:App.slashMenu「未找到命令或技能」 |
| R-GM-020 | G-B1-03 | 状态 | 有匹配项时 | 空态不出现 | — | 屏上 空态文字 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GM-021 | G-B1-03 | 样式 | 空态在屏上时 | 14px 次要色、左内边距与候选行文字对齐 | — | 判据 = 字号 正文 14px；内边距 左缘 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28） | 已实现（映射 = G-B1-03→internal/ui/slash_commands.go:App.slashMenu；标记 = Height） |
| R-GM-022 | G-B1-03 | 无障碍 | 读屏进入该元素时 | 读出该句 | — | 读屏文本 = 「未找到命令或技能」 | 已实现（映射 = G-B1-03→internal/ui/slash_commands.go:App.slashMenu；标记 = Label） |
| R-GM-023 | G-B1-03 | 异常 | 无匹配但列表仍在拉取 | 先给骨架，再给空态（不提前判空） | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B1-03→internal/ui/slash_commands.go:App.slashMenu；标记 = reason@App.slashCommandRow） |
| R-GM-024 | G-B1-03 | 内容 | 前缀大小写不符（`/AG`） | 按不区分大小写匹配；仍无命中才给空态 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 已实现（映射 = G-B1-03→internal/ui/slash_commands.go:App.slashMenu；标记 = 文案「斜杠命令」等 2 条） |
| R-GM-025 | G-B1-03 | 状态 | `/` 后为空 | 不显示空态（显示全量命令） | — | 屏上 空态 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GM-026 | G-B1-03 | 关联 | 无匹配、两个分组标题都不在场时 | 空态时两个分组标题都不显示 | — | 屏上 空态文字 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GM-027 | G-B1-04 | 内容 | C37：命令组有项 | 分组标题「指令」在场（实测 390,328 327×25），12px 灰字 | — | 判据 = 字号 UI 正文 12.25px；字重 常规 400 / 中粗 500 / 半粗 600 / 粗 700 | 已实现（逐字）：internal/ui/settings_git_screen.go:gitCopyCommitNotes「指令」 |
| R-GM-028 | G-B1-04 | 内容 | 有技能候选 | 第二组标题「技能」；无候选时不渲染（C37 无该组） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GM-029 | G-B1-04 | 关联 | 指令组与技能组同时在场时 | 「指令」在「技能」之前，顺序固定 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM） | 已实现（逐字）：internal/ui/settings_git_screen.go:gitCopyCommitNotes「指令」 |
| R-GM-030 | G-B1-04 | 样式 | 分组标题在屏上时 | 12px 次要色、与命令名左缘对齐（x=390 vs 401） | — | 判据 = 字号 UI 正文 12.25px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a；内边距 左缘 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28） | 已实现（映射 = G-B1-04→internal/ui/slash_commands.go:slashGroupHeader；标记 = Height） |
| R-GM-031 | G-B1-04 | 无障碍 | 读屏进入该元素时 | 分组标题先于组内候选 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-B1-04→internal/ui/slash_commands.go:slashGroupHeader；标记 = Label） |
| R-GM-032 | G-B1-04 | 异常 | 组内项全部被过滤掉 | 该组标题一起消失 | — | 屏上 分组标题 计数 = 0 | 已实现（映射 = G-B1-04→internal/ui/slash_commands.go:App.slashMenuGroups；标记 = Reason@App.slashCommands） |
| R-GM-033 | G-B1-04 | 内容 | `/` 后为空 | 「指令」组显示全部命令（C37 为 10 行） | — | 组内行数 ≥10 | 已实现（逐字）：internal/ui/settings_git_screen.go:gitCopyCommitNotes「指令」 |
| R-GM-034 | G-B1-04 | 状态 | 技能组存在时 | 「技能」标题排在「指令」组之后 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM） | 已实现（逐字）：internal/ui/approval.go:approvalActionWord「技能」 |
| R-GM-035 | G-B1-05 | 内容 | C37：命令候选行 | 行内主字「/{命令名}」（x=401）+ 右尾描述（12px 灰字，右对齐）；行 327×28 | — | 行高 28；主字以 `/` 开头 | 已实现（映射 = G-B1-05→internal/ui/slash_commands.go:App.slashCommandRow；标记 = 文案「不可用：」） |
| R-GM-036 | G-B1-05 | 内容 | 命令无描述 | 只显示名称，右尾留空（不显示占位符） | — | 屏上 占位符 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GM-037 | G-B1-05 | 内容 | 描述超长 | 描述截断（右尾），命令名完整 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-B1-05→internal/ui/slash_commands.go:App.slashCommandRow；标记 = 文案「不可用：」） |
| R-GM-038 | G-B1-05 | 交互 | 鼠标点击该行时 | 把该命令插入输入框（替换 `/前缀`）并关闭弹层 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B1-05→internal/ui/slash_commands.go:App.slashCommandRow；标记 = Menu） |
| R-GM-039 | G-B1-05 | 状态 | 行选中 | 强调底 + 强调前景（同一时刻只 1 行） | — | 判据 = 底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d；字色 主文字 明 #0a0a0a / 暗 #ececec | 已实现（映射 = G-B1-05→internal/ui/slash_commands.go:App.slashCommandRow；标记 = Disabled） |
| R-GM-040 | G-B1-05 | 样式 | 主字与描述 | 主字 14px 主前景、描述 12px 次要色；两者垂直居中 | — | 判据 = 字号 UI 正文 12.25px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-B1-05→internal/ui/slash_commands.go:App.slashCommandRow；标记 = Height） |
| R-GM-041 | G-B1-05 | 无障碍 | 读屏进入该元素时 | 播报命令名与描述 | — | 读屏文本 = 「/{命令名}」 | 已实现（映射 = G-B1-05→internal/ui/slash_commands.go:App.slashCommandRow；标记 = Label） |
| R-GM-042 | G-B1-05 | 高DPI | 在 DPR=2 的屏幕上时 | 行高 28 与文字基线在 2× 下对齐 | — | 判据 = 行高 空态主标题 28px；间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px | 未实现（映射到位但没有该维度的标记）：G-B1-05→internal/ui/slash_commands.go:App.slashCommandRow；测试提及 internal/ui/slash_commands_test.go:包级 |
| R-GM-043 | G-B1-06 | 内容 | 未观察（条件）：`/` 过滤后含技能候选 | 技能候选行：主字为技能名、右尾描述；无描述只显示名称 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-B1-06→internal/ui/slash_commands.go:App.slashMenu；标记 = 文案「斜杠命令」等 2 条） |
| R-GM-044 | G-B1-06 | 关联 | 技能候选行落在技能组下时 | 同一套行样式与行高，落在「技能」组下 | — | 判据 = 行高 UI 正文 17.5px；间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px | 已实现（逐字）：internal/ui/approval.go:approvalActionWord「技能」 |
| R-GM-045 | G-B1-06 | 交互 | 点击技能行 | 插入技能名并关闭弹层 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B1-06→internal/ui/slash_commands.go:App.slashMenu；标记 = Menu） |
| R-GM-046 | G-B1-06 | 样式 | 指令行与技能行同屏时 | 同字号同高，仅组不同 | — | 判据 = 行高 UI 正文 17.5px；字号 UI 正文 12.25px | 已实现（映射 = G-B1-06→internal/ui/slash_commands.go:App.slashMenu；标记 = Height） |
| R-GM-047 | G-B1-06 | 无障碍 | 读屏进入该元素时 | 播报技能名与描述 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-B1-06→internal/ui/slash_commands.go:App.slashMenu；标记 = Label） |
| R-GM-048 | G-B1-06 | 异常 | 技能列表为空 | 该组不渲染，不留空标题 | — | 屏上 候选行 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GM-049 | G-B1-06 | 状态 | 技能候选被过滤 | 行整体移除，不留空组 | — | 屏上 候选行 计数 = 0 | 已实现（映射 = G-B1-06→internal/ui/slash_commands.go:App.slashMenu；标记 = Disabled@App.slashCommandRow） |
| R-GM-050 | G-B1-06 | 内容 | 技能名带命名空间 | 原样显示（不裁前缀） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-B1-06→internal/ui/slash_commands.go:App.slashMenu；标记 = 文案「斜杠命令」等 2 条） |
| R-GM-051 | G-B1-06 | 键盘 | Enter 选中技能行 | 插入技能名并关闭弹层 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 未实现（映射到位但没有该维度的标记）：G-B1-06→internal/ui/slash_commands.go:App.slashMenu；测试提及 internal/ui/slash_commands_test.go:包级 |
| R-GM-052 | G-B1-07 | 状态 | 未观察（条件）：按 ↑/↓ 或悬停 | 选中行显示强调底 + 强调前景；C37 截图里 10 行全为普通底，说明首行并非默认选中 | — | 未操作时无高亮；操作后 1 行高亮 | 已实现（映射 = G-B1-07→internal/ui/slash_commands.go:App.slashTextKey；标记 = State@pickState.dismiss） |
| R-GM-053 | G-B1-07 | 交互 | 反复按 ↑ / ↓ 时 | 高亮逐行迁移，列表跟随滚动保证可见 | — | 判据 = 底色 选中底 明 #f5f5f5 / 暗 #262626；字色 主文字 明 #0a0a0a / 暗 #ececec；其余 = 选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B1-07→internal/ui/slash_commands.go:App.slashMoveSelection；标记 = Menu@App.slashSelectable） |
| R-GM-054 | G-B1-07 | 交互 | 按 Enter 时 | 执行/插入当前高亮项并关闭弹层 | — | 判据 = 底色 选中底 明 #f5f5f5 / 暗 #262626；字色 主文字 明 #0a0a0a / 暗 #ececec；其余 = 选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B1-07→internal/ui/slash_commands.go:App.slashMoveSelection；标记 = Menu@App.slashSelectable） |
| R-GM-055 | G-B1-07 | 样式 | 高亮在屏上时 | 强调底 10–15%、圆角小、文字转强调前景 | — | 判据 = 底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px；其余 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B1-07→internal/ui/composer.go:App.composerCard；标记 = Width） |
| R-GM-056 | G-B1-07 | 无障碍 | 读屏进入该元素时 | 当前项被读出 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-B1-07→internal/ui/composer.go:App.composerCard；标记 = Label） |
| R-GM-057 | G-B1-07 | 异常 | 过滤后高亮项消失 | 高亮落到新的首项，不停在不存在的位置 | — | 判据 = 底色 选中底 明 #f5f5f5 / 暗 #262626；字色 主文字 明 #0a0a0a / 暗 #ececec；其余 = 选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B1-07→internal/ui/slash_commands.go:App.slashTextKey；标记 = reason@App.slashExecuteSelected） |
| R-GM-058 | G-B1-07 | 内容 | 无操作时 | 10 行全为普通底（C37 实测），说明首行不默认高亮 | — | 判据 = 底色 选中底 明 #f5f5f5 / 暗 #262626；字色 主文字 明 #0a0a0a / 暗 #ececec；其余 = 选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B1-07→internal/ui/slash_commands.go:App.slashTextKey；标记 = 文案「不可用：」@App.slashExecuteSelected） |
| R-GM-059 | G-B1-07 | 状态 | 鼠标移出弹层 | 高亮保留在最后悬停行（不清空） | — | 判据 = 底色 选中底 明 #f5f5f5 / 暗 #262626；字色 主文字 明 #0a0a0a / 暗 #ececec；其余 = 选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B1-07→internal/ui/slash_commands.go:App.slashTextKey；标记 = State@pickState.dismiss） |
| R-GM-060 | G-B1-08 | 内容 | 在输入框键入 `/` 打开指令菜单时 | 行内主字「/agent」（401,358）、右尾描述「切换至 Agent 模式」（619,359） | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（逐字）：internal/ui/slash_commands.go:App.slashCommands「切换至 Agent 模式」 |
| R-GM-061 | G-B1-08 | 交互 | 选中一条会话后执行时 | 输入区切换到 Agent 模式；弹层关闭；输入框清空该命令文本 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-B1-08→internal/ui/slash_commands.go:App.slashExecuteSelected；标记 = Menu） |
| R-GM-062 | G-B1-08 | 状态 | 已是 Agent 模式 | 候选仍出现（命令幂等），执行后模式不变 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-B1-08→internal/ui/slash_commands.go:App.slashExecuteSelected；标记 = State@pickState.reset） |
| R-GM-063 | G-B1-08 | 关联 | 执行 /agent 后回到主界面时 | 标题栏模式标签随之为 Agent，不出现两处不一致 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GM-064 | G-B1-08 | 样式 | 行样式渲染后 | 与 G-B1-05 同款（主字 + 右尾描述） | — | 判据 = 行高 UI 正文 17.5px；字号 UI 正文 12.25px | 未实现（映射到位但没有该维度的标记）：G-B1-08→internal/ui/slash_commands.go:App.slashExecuteSelected；测试提及 internal/ui/slash_commands_test.go:包级 |
| R-GM-065 | G-B1-08 | 无障碍 | 读屏进入该元素时 | 播报「/agent，切换至 Agent 模式」 | — | 读屏文本 = 「/agent，切换至 Agent 模式」 | 未实现（映射到位但没有该维度的标记）：G-B1-08→internal/ui/slash_commands.go:App.slashExecuteSelected；测试提及 internal/ui/slash_commands_test.go:包级 |
| R-GM-066 | G-B1-08 | 异常 | 模式切换失败 | 弹层关闭并给失败提示，模式保持原值 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B1-08→internal/ui/slash_commands.go:App.slashExecuteSelected；标记 = reason） |
| R-GM-067 | G-B1-08 | 记忆 | 关闭后重新打开弹层时 | 该项仍在原位（常驻候选，不因用过而消失） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 未实现（映射到位但没有该维度的标记）：G-B1-08→internal/ui/slash_commands.go:App.slashExecuteSelected；测试提及 internal/ui/slash_commands_test.go:包级 |
| R-GM-068 | G-B1-09 | 内容 | 在输入框键入 `/` 打开指令菜单时 | 主字「/goal」（401,386）、描述「切换至 Goal 模式」（625,387） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/slash_commands.go:App.slashCommands「切换至 Goal 模式」 |
| R-GM-069 | G-B1-09 | 交互 | 选中一条会话后执行时 | 切到 Goal 模式；弹层关闭 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-B1-09→internal/ui/slash_commands.go:App.slashCommands |
| R-GM-070 | G-B1-09 | 关联 | 先后执行 /agent 与 /goal 时 | 两者作用于同一模式位，互斥 | — | 屏上 固定命令行 计数 = 1 | 已实现（映射 = G-B1-09→internal/ui/slash_commands.go:App.slashCommands；标记 = Open） |
| R-GM-071 | G-B1-09 | 状态 | 已是 Goal 模式 | 仍可执行，模式不变 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-B1-09→internal/ui/slash_commands.go:App.slashCommands；标记 = State@App.switchMode） |
| R-GM-072 | G-B1-09 | 样式 | 行样式渲染后 | 同 G-B1-05 | — | 判据 = 行高 UI 正文 17.5px；字号 UI 正文 12.25px | 未实现（映射到位但没有该维度的标记）：G-B1-09→internal/ui/slash_commands.go:App.slashCommands |
| R-GM-073 | G-B1-09 | 无障碍 | 读屏进入该元素时 | 播报「/goal，切换至 Goal 模式」 | — | 读屏文本 = 「/goal，切换至 Goal 模式」 | 未实现（映射到位但没有该维度的标记）：G-B1-09→internal/ui/slash_commands.go:App.slashCommands |
| R-GM-074 | G-B1-09 | 异常 | 切换失败 | 给失败提示，模式不变 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B1-09→internal/ui/slash_commands.go:App.slashCommands；标记 = Reason） |
| R-GM-075 | G-B1-09 | 记忆 | 关闭后重新打开弹层时 | 仍在原位 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-B1-09→internal/ui/slash_commands.go:App.slashCommands |
| R-GM-076 | G-B1-10 | 内容 | 在输入框键入 `/` 打开指令菜单时 | 主字「/pet」（401,414）、描述「唤醒或收起 Pet」（635,415） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/slash_commands.go:App.slashCommands「唤醒或收起 Pet」 |
| R-GM-077 | G-B1-10 | 交互 | 选中一条会话后执行时 | Pet 显隐切换一次；弹层关闭 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-B1-10→internal/ui/slash_commands.go:App.slashCommands |
| R-GM-078 | G-B1-10 | 状态 | Pet 已关闭 | 执行后唤醒；已唤醒则收起（同一命令两向） | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B1-10→internal/ui/slash_commands.go:App.slashCommands；标记 = State@App.switchMode） |
| R-GM-079 | G-B1-10 | 关联 | 执行 /pet 后打开设置的宠物节时 | 状态与设置项同源，不出现「命令说开了、设置说关」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GM-080 | G-B1-10 | 样式 | 行样式渲染后 | 同 G-B1-05 | — | 判据 = 行高 UI 正文 17.5px | 未实现（映射到位但没有该维度的标记）：G-B1-10→internal/ui/slash_commands.go:App.slashCommands |
| R-GM-081 | G-B1-10 | 无障碍 | 读屏进入该元素时 | 播报「/pet，唤醒或收起 Pet」 | — | 读屏文本 = 「/pet，唤醒或收起 Pet」 | 未实现（映射到位但没有该维度的标记）：G-B1-10→internal/ui/slash_commands.go:App.slashCommands |
| R-GM-082 | G-B1-10 | 异常 | Pet 资源缺失 | 提示失败，不静默 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GM-083 | G-B1-10 | 记忆 | 关闭后重新打开弹层时 | 仍在原位 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-B1-10→internal/ui/slash_commands.go:App.slashCommands |
| R-GM-084 | G-B1-11 | 内容 | C37：常驻候选（非独立会话窗口） | 主字「/settings」（401,442）、描述「打开设置」（665,443） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/command_palette.go:App.commandPaletteSettings「打开设置」 |
| R-GM-085 | G-B1-11 | 交互 | 选中一条会话后执行时 | 打开设置窗口/路由；输入框清空命令文本 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-B1-11→internal/ui/slash_commands.go:App.slashCommands；测试提及 internal/ui/slash_commands_test.go:包级 |
| R-GM-086 | G-B1-11 | 状态 | 独立会话窗口 | 该命令不出现（`/settings` 依赖主窗口路由） | — | 屏上 固定命令行 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GM-087 | G-B1-11 | 关联 | 先后执行 /models、/mcp、/skills 时 | 同属「打开设置」类，执行后落在不同设置节 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/command_palette.go:App.commandPaletteSettings「打开设置」 |
| R-GM-088 | G-B1-11 | 样式 | 行样式渲染后 | 同 G-B1-05 | — | 判据 = 行高 UI 正文 17.5px | 未实现（映射到位但没有该维度的标记）：G-B1-11→internal/ui/slash_commands.go:App.slashCommands；测试提及 internal/ui/slash_commands_test.go:包级 |
| R-GM-089 | G-B1-11 | 无障碍 | 读屏进入该元素时 | 播报「/settings，打开设置」 | — | 读屏文本 = 「/settings，打开设置」 | 未实现（映射到位但没有该维度的标记）：G-B1-11→internal/ui/slash_commands.go:App.slashCommands；测试提及 internal/ui/slash_commands_test.go:包级 |
| R-GM-090 | G-B1-11 | 异常 | 设置窗口打开失败 | 提示失败，输入框内容保留 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B1-11→internal/ui/slash_commands.go:App.slashCommands；标记 = Reason） |
| R-GM-091 | G-B1-11 | 内容 | 文案超长（窄窗） | 描述 truncate，命令名完整 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-B1-11→internal/ui/slash_commands.go:App.slashCommands；标记 = 文案「还没有打开会话」等 17 条） |
| R-GM-092 | G-B1-12 | 内容 | 在输入框键入 `/` 打开指令菜单时 | 主字「/models」（401,470）、描述「打开模型设置」（644,471） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/slash_commands.go:App.slashCommands「打开模型设置」 |
| R-GM-093 | G-B1-12 | 交互 | 选中一条会话后执行时 | 打开设置并定位到模型供应节 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-B1-12→internal/ui/slash_commands.go:App.slashCommands；测试提及 internal/ui/slash_commands_test.go:包级 |
| R-GM-094 | G-B1-12 | 状态 | 独立会话窗口 | 该命令不出现 | — | 屏上 固定命令行 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GM-095 | G-B1-12 | 关联 | 先后执行 /settings 与 /models 时 | 一个总入口、一个直达子节 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B1-12→internal/ui/slash_commands.go:App.slashCommands；标记 = Open） |
| R-GM-096 | G-B1-12 | 样式 | 行样式渲染后 | 同 G-B1-05 | — | 判据 = 行高 UI 正文 17.5px | 未实现（映射到位但没有该维度的标记）：G-B1-12→internal/ui/slash_commands.go:App.slashCommands；测试提及 internal/ui/slash_commands_test.go:包级 |
| R-GM-097 | G-B1-12 | 无障碍 | 读屏进入该元素时 | 播报「/models，打开模型设置」 | — | 读屏文本 = 「/models，打开模型设置」 | 未实现（映射到位但没有该维度的标记）：G-B1-12→internal/ui/slash_commands.go:App.slashCommands；测试提及 internal/ui/slash_commands_test.go:包级 |
| R-GM-098 | G-B1-12 | 异常 | 模型节不可用 | 提示并停在设置主页 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B1-12→internal/ui/slash_commands.go:App.slashCommands；标记 = Reason） |
| R-GM-099 | G-B1-12 | 内容 | 命令名与描述同行渲染时 | 描述 truncate，命令名完整 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-B1-12→internal/ui/slash_commands.go:App.slashCommands；标记 = 文案「还没有打开会话」等 17 条） |
| R-GM-100 | G-B1-13 | 内容 | 在输入框键入 `/` 打开指令菜单时 | 主字「/mcp」（401,498）、描述「打开 MCP 设置」（635,499） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/slash_commands.go:App.slashCommands「打开 MCP 设置」 |
| R-GM-101 | G-B1-13 | 交互 | 选中一条会话后执行时 | 打开设置并定位到 MCP 节 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-B1-13→internal/ui/slash_commands.go:App.slashCommands；测试提及 internal/ui/slash_commands_test.go:包级 |
| R-GM-102 | G-B1-13 | 状态 | 独立会话窗口 | 该命令不出现 | — | 屏上 固定命令行 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GM-103 | G-B1-13 | 关联 | 先后执行 /models 与 /mcp 时 | 同为直达子节，互不影响 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B1-13→internal/ui/slash_commands.go:App.slashCommands；标记 = Open） |
| R-GM-104 | G-B1-13 | 样式 | 行样式渲染后 | 同 G-B1-05 | — | 判据 = 行高 UI 正文 17.5px | 未实现（映射到位但没有该维度的标记）：G-B1-13→internal/ui/slash_commands.go:App.slashCommands；测试提及 internal/ui/slash_commands_test.go:包级 |
| R-GM-105 | G-B1-13 | 无障碍 | 读屏进入该元素时 | 播报「/mcp，打开 MCP 设置」 | — | 读屏文本 = 「/mcp，打开 MCP 设置」 | 未实现（映射到位但没有该维度的标记）：G-B1-13→internal/ui/slash_commands.go:App.slashCommands；测试提及 internal/ui/slash_commands_test.go:包级 |
| R-GM-106 | G-B1-13 | 异常 | MCP 节不可用 | 提示并停在设置主页 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B1-13→internal/ui/slash_commands.go:App.slashCommands；标记 = Reason） |
| R-GM-107 | G-B1-13 | 内容 | 命令名与描述同行渲染时 | 描述 truncate，命令名完整 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-B1-13→internal/ui/slash_commands.go:App.slashCommands；标记 = 文案「还没有打开会话」等 17 条） |
| R-GM-108 | G-B1-14 | 内容 | 在输入框键入 `/` 打开指令菜单时 | 主字「/skills」（401,526）、描述「打开技能」（665,527） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/slash_commands.go:App.slashCommands「打开技能」 |
| R-GM-109 | G-B1-14 | 交互 | 选中一条会话后执行时 | 打开设置/技能页并定位到技能列表 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-B1-14→internal/ui/slash_commands.go:App.slashCommands；测试提及 internal/ui/slash_commands_test.go:包级 |
| R-GM-110 | G-B1-14 | 状态 | 独立会话窗口 | 该命令不出现 | — | 屏上 固定命令行 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GM-111 | G-B1-14 | 关联 | 先后执行 /skills 与 /reload-skills 时 | 一个打开页面、一个重扫磁盘 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B1-14→internal/ui/slash_commands.go:App.slashCommands；标记 = Open） |
| R-GM-112 | G-B1-14 | 样式 | 行样式渲染后 | 同 G-B1-05 | — | 判据 = 行高 UI 正文 17.5px | 未实现（映射到位但没有该维度的标记）：G-B1-14→internal/ui/slash_commands.go:App.slashCommands；测试提及 internal/ui/slash_commands_test.go:包级 |
| R-GM-113 | G-B1-14 | 无障碍 | 读屏进入该元素时 | 播报「/skills，打开技能」 | — | 读屏文本 = 「/skills，打开技能」 | 未实现（映射到位但没有该维度的标记）：G-B1-14→internal/ui/slash_commands.go:App.slashCommands；测试提及 internal/ui/slash_commands_test.go:包级 |
| R-GM-114 | G-B1-14 | 异常 | 技能页不可用 | 提示并停在设置主页 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B1-14→internal/ui/slash_commands.go:App.slashCommands；标记 = Reason） |
| R-GM-115 | G-B1-14 | 内容 | 命令名与描述同行渲染时 | 描述 truncate，命令名完整 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-B1-14→internal/ui/slash_commands.go:App.slashCommands；标记 = 文案「还没有打开会话」等 17 条） |
| R-GM-116 | G-B1-15 | 内容 | 在输入框键入 `/` 打开指令菜单时 | 主字「/reload-skills」（401,554）、描述「重新扫描磁盘上的技能」（602,555） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/slash_commands.go:App.slashCommands「重新扫描磁盘上的技能」 |
| R-GM-117 | G-B1-15 | 交互 | 选中一条会话后执行时 | 重扫磁盘技能并刷新技能列表；给执行反馈 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-B1-15→internal/ui/slash_commands.go:App.slashCommands；测试提及 internal/ui/slash_commands_test.go:包级 |
| R-GM-118 | G-B1-15 | 状态 | 执行进行中时 | 命令仍可再次触发，但同一时刻只跑一次重扫 | — | 判据 = 可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE）；可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B1-15→internal/ui/slash_commands.go:App.slashCommands；标记 = State@App.switchMode） |
| R-GM-119 | G-B1-15 | 异常 | 扫描失败 | 提示失败并保留旧列表 | — | 判据 = 计数 = 屏上可见项数、差 0（TX-EXACT 的计数形态） | 已实现（映射 = G-B1-15→internal/ui/slash_commands.go:App.slashCommands；标记 = Reason） |
| R-GM-120 | G-B1-15 | 样式 | 行样式渲染后 | 同 G-B1-05 | — | 判据 = 行高 UI 正文 17.5px | 未实现（映射到位但没有该维度的标记）：G-B1-15→internal/ui/slash_commands.go:App.slashCommands；测试提及 internal/ui/slash_commands_test.go:包级 |
| R-GM-121 | G-B1-15 | 无障碍 | 读屏进入该元素时 | 播报「/reload-skills，重新扫描磁盘上的技能」 | — | 读屏文本 = 「/reload-skills，重新扫描磁盘上的技能」 | 未实现（映射到位但没有该维度的标记）：G-B1-15→internal/ui/slash_commands.go:App.slashCommands；测试提及 internal/ui/slash_commands_test.go:包级 |
| R-GM-122 | G-B1-15 | 记忆 | 关闭后重新打开弹层时 | 仍在原位 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-B1-15→internal/ui/slash_commands.go:App.slashCommands；测试提及 internal/ui/slash_commands_test.go:包级 |
| R-GM-123 | G-B1-15 | 内容 | 描述与命令名同行时 | 描述「重新扫描磁盘上的技能」右对齐，命令名左对齐 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（逐字）：internal/ui/slash_commands.go:App.slashCommands「重新扫描磁盘上的技能」 |
| R-GM-124 | G-B1-16 | 内容 | 在输入框键入 `/` 打开指令菜单时 | 主字「/reload-plugins」（401,582）、描述「重新扫描已安装插件」（612,583）；PNG 中该行在弹层裁剪区内，需滚动露出 | — | 判据 = 滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（逐字）：internal/ui/slash_commands.go:App.slashCommands「重新扫描已安装插件」 |
| R-GM-125 | G-B1-16 | 交互 | 选中一条会话后执行时 | 重扫插件并刷新插件列表 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-B1-16→internal/ui/slash_commands.go:App.slashCommands |
| R-GM-126 | G-B1-16 | 状态 | 执行进行中时 | 同一时刻只跑一次重扫 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B1-16→internal/ui/slash_commands.go:App.slashCommands；标记 = State@App.switchMode） |
| R-GM-127 | G-B1-16 | 异常 | 扫描失败 | 提示并保留旧列表 | — | 判据 = 计数 = 屏上可见项数、差 0（TX-EXACT 的计数形态） | 已实现（映射 = G-B1-16→internal/ui/slash_commands.go:App.slashCommands；标记 = Reason） |
| R-GM-128 | G-B1-16 | 样式 | 行样式渲染后 | 同 G-B1-05 | — | 判据 = 行高 UI 正文 17.5px | 未实现（映射到位但没有该维度的标记）：G-B1-16→internal/ui/slash_commands.go:App.slashCommands |
| R-GM-129 | G-B1-16 | 无障碍 | 读屏进入该元素时 | 播报命令名与描述 | — | 播报顺序 = /reload-plugins → 重新扫描已安装插件 | 未实现（映射到位但没有该维度的标记）：G-B1-16→internal/ui/slash_commands.go:App.slashCommands |
| R-GM-130 | G-B1-16 | 记忆 | 关闭后重新打开弹层时 | 仍在原位 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-B1-16→internal/ui/slash_commands.go:App.slashCommands |
| R-GM-131 | G-B1-16 | 内容 | 命令名与描述同行渲染时 | 描述右对齐，命令名左对齐 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-B1-16→internal/ui/slash_commands.go:App.slashCommands；标记 = 文案「还没有打开会话」等 17 条） |
| R-GM-132 | G-B1-17 | 内容 | 在输入框键入 `/` 打开指令菜单时 | 主字「/compact」（401,610）、描述「压缩当前上下文」（633,611）；PNG 中在裁剪区内，需滚动露出 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（逐字）：internal/ui/slash_commands.go:App.slashCommands「压缩当前上下文」 |
| R-GM-133 | G-B1-17 | 交互 | 选中一条会话后执行时 | 触发一次上下文压缩；时间线出现压缩记录 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-B1-17→internal/ui/slash_commands.go:App.slashCommands；测试提及 internal/ui/slash_commands_test.go:包级 |
| R-GM-134 | G-B1-17 | 状态 | 回合运行中 | 命令仍可用，压缩在当前回合结束后生效（按实现一致） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B1-17→internal/ui/slash_commands.go:App.slashCommands；标记 = State@App.switchMode） |
| R-GM-135 | G-B1-17 | 异常 | 压缩失败 | 提示失败，上下文不变 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-B1-17→internal/ui/slash_commands.go:App.slashCommands；标记 = Reason） |
| R-GM-136 | G-B1-17 | 样式 | 行样式渲染后 | 同 G-B1-05 | — | 判据 = 行高 UI 正文 17.5px | 未实现（映射到位但没有该维度的标记）：G-B1-17→internal/ui/slash_commands.go:App.slashCommands；测试提及 internal/ui/slash_commands_test.go:包级 |
| R-GM-137 | G-B1-17 | 无障碍 | 读屏进入该元素时 | 播报「/compact，压缩当前上下文」 | — | 读屏文本 = 「/compact，压缩当前上下文」 | 未实现（映射到位但没有该维度的标记）：G-B1-17→internal/ui/slash_commands.go:App.slashCommands；测试提及 internal/ui/slash_commands_test.go:包级 |
| R-GM-138 | G-B1-17 | 记忆 | 关闭后重新打开弹层时 | 仍在原位 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-B1-17→internal/ui/slash_commands.go:App.slashCommands；测试提及 internal/ui/slash_commands_test.go:包级 |
| R-GM-139 | G-B1-17 | 内容 | 命令名与描述同行渲染时 | 描述右对齐，命令名左对齐 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-B1-17→internal/ui/slash_commands.go:App.slashCommands；标记 = 文案「还没有打开会话」等 17 条） |
| R-GM-140 | G-B1-18 | 内容 | 未观察（条件）：快速模式可用，且弹层滚动到 `/compact` 之后 | 主字「/fast」，描述为**未本地化的英文原文**「Toggle fast mode」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/slash_commands.go:App.slashCommands「Toggle fast mode」 |
| R-GM-141 | G-B1-18 | 状态 | 快速模式不可用 | 该命令不出现 | — | 屏上 固定命令行 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GM-142 | G-B1-18 | 交互 | 选中一条会话后执行时 | 切换快速模式（开↔关） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B1-18→internal/ui/slash_commands.go:App.slashCommandRow；标记 = Menu） |
| R-GM-143 | G-B1-18 | 关联 | 执行 /fast 后回到主界面时 | 快速模式状态在主界面有可见标识（状态条/标签） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B1-18→internal/ui/slash_commands.go:App.slashCommands；标记 = Open） |
| R-GM-144 | G-B1-18 | 样式 | 行样式渲染后 | 同 G-B1-05；英文描述不折行 | — | 判据 = 行高 UI 正文 17.5px；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-B1-18→internal/ui/slash_commands.go:App.slashCommandRow；标记 = Height） |
| R-GM-145 | G-B1-18 | 无障碍 | 读屏进入该元素时 | 播报英文描述原文（不翻译） | — | 读屏文本 = 「/fast」 | 已实现（映射 = G-B1-18→internal/ui/slash_commands.go:App.slashCommandRow；标记 = Label） |
| R-GM-146 | G-B1-18 | 异常 | 切换失败 | 提示失败，模式不变 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B1-18→internal/ui/slash_commands.go:App.slashCommands；标记 = Reason） |
| R-GM-147 | G-B1-18 | 内容 | 英文描述宽度 | 英文描述比中文长时仍单行 truncate | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-B1-18→internal/ui/slash_commands.go:App.slashCommands；标记 = 文案「还没有打开会话」等 17 条） |
| R-GM-148 | G-B1-19 | 内容 | 未观察（条件）：弹层滚动到 `/compact` 之后 | 主字「/branch」，描述「侧边对话」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/rightpanel_parts.go:rpNewTabItems「侧边对话」 |
| R-GM-149 | G-B1-19 | 交互 | 选中一条会话后执行时 | 为当前回合开启侧边对话（右栏「侧聊」标签） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-B1-19→internal/ui/slash_commands.go:App.slashCommands |
| R-GM-150 | G-B1-19 | 状态 | 已在侧聊中 | 命令仍可用，执行后聚焦已有侧聊而非新开 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B1-19→internal/ui/slash_commands.go:App.slashCommands；标记 = State@App.switchMode） |
| R-GM-151 | G-B1-19 | 关联 | 打开侧聊面板时 | 与「从消息处开侧聊」同一能力，两个入口 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B1-19→internal/ui/slash_commands.go:App.slashCommands；标记 = Open） |
| R-GM-152 | G-B1-19 | 样式 | 行样式渲染后 | 同 G-B1-05 | — | 判据 = 行高 UI 正文 17.5px | 未实现（映射到位但没有该维度的标记）：G-B1-19→internal/ui/slash_commands.go:App.slashCommands |
| R-GM-153 | G-B1-19 | 无障碍 | 读屏进入该元素时 | 播报「/branch，侧边对话」 | — | 读屏文本 = 「/branch，侧边对话」 | 未实现（映射到位但没有该维度的标记）：G-B1-19→internal/ui/slash_commands.go:App.slashCommands |
| R-GM-154 | G-B1-19 | 异常 | 当前无回合可侧聊 | 提示不可用，不打开空面板 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B1-19→internal/ui/slash_commands.go:App.slashCommands；标记 = Reason） |
| R-GM-155 | G-B1-19 | 内容 | 描述与命令名同行时 | 「侧边对话」右对齐 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（逐字）：internal/ui/rightpanel_parts.go:rpNewTabItems「侧边对话」 |
| R-GM-156 | G-B1-20 | 内容 | 未观察（条件）：弹层滚动到末项 | 主字「/workflow」，描述为未本地化英文「Browse workflows」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/slash_commands.go:App.slashCommands「Browse workflows」 |
| R-GM-157 | G-B1-20 | 交互 | 选中一条会话后执行时 | 打开工作流浏览入口 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-B1-20→internal/ui/slash_commands.go:App.slashCommands |
| R-GM-158 | G-B1-20 | 状态 | 无可用工作流 | 命令仍出现（入口常在），执行后给空态 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-B1-20→internal/ui/slash_commands.go:App.slashCommands；标记 = State@App.switchMode） |
| R-GM-159 | G-B1-20 | 样式 | 行样式渲染后 | 同 G-B1-05；为列表最后一项 | — | 判据 = 行高 UI 正文 17.5px；内边距 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28）/ 卡片外框 4.5；档位 3.5·7·10.5·14·21·28px | 未实现（映射到位但没有该维度的标记）：G-B1-20→internal/ui/slash_commands.go:App.slashCommands |
| R-GM-160 | G-B1-20 | 无障碍 | 读屏进入该元素时 | 播报英文描述原文 | — | 读屏文本 = 「/workflow」 | 未实现（映射到位但没有该维度的标记）：G-B1-20→internal/ui/slash_commands.go:App.slashCommands |
| R-GM-161 | G-B1-20 | 异常 | 入口不可用 | 提示失败，不静默 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GM-162 | G-B1-20 | 内容 | 英文描述宽度 | 「Browse workflows」单行 truncate | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（逐字）：internal/ui/slash_commands.go:App.slashCommands「Browse workflows」 |
| R-GM-163 | G-B1-20 | 状态 | 末项位置 | 为列表最后一行，下方无更多行 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-B1-20→internal/ui/slash_commands.go:App.slashCommands；标记 = State@App.switchMode） |

### 3.2 命令面板基元（G-B2-01–G-B2-09）

条件：参考产品内只有「设置 → 添加模型」调色板使用这套基元；斜杠命令与会话搜索各自手写浮层，不复用本组。60 份态里没有任何一份打开该调色板。

| 需求 ID | 元素 ID | 维度 | 场景 | 期望 | 异常 | 验收 | 状态 |
|---|---|---|---|---|---|---|---|
| R-GM-164 | G-B2-01 | 内容 | 未观察（条件）：打开设置 → 添加模型 | 调色板标题与描述：实际使用处为「添加模型」/「搜索模型...」；组件默认值是英文「Command Palette」/「Search for a command to run...」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/settings_configuration_screen.go:App.configModelsToolbar「添加模型」 |
| R-GM-165 | G-B2-01 | 内容 | 调用方未覆盖标题 | 回落到英文默认值，不显示空标题 | — | 屏上 空标题 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GM-166 | G-B2-01 | 状态 | 两者 `sr-only` | 标题与描述屏上不可见，仅供读屏 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-B2-01→internal/ui/command_palette.go:App.commandPaletteLayer；标记 = Hidden） |
| R-GM-167 | G-B2-01 | 无障碍 | 读屏进入该元素时 | 标题作对话框名、描述作说明 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-B2-01→internal/ui/command_palette.go:App.commandPaletteLayer；标记 = AccessHidden） |
| R-GM-168 | G-B2-01 | 关联 | 两个对话框标题先后被读屏访问时 | 同属 `sr-only` 标题模式，口径一致 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B2-01→internal/ui/command_palette.go:App.commandPaletteLayer；标记 = Tab@App.commandPaletteFooter） |
| R-GM-169 | G-B2-01 | 异常 | 描述未翻译 | 显示英文原文，不显示空串 | — | 屏上 空串 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GM-170 | G-B2-01 | 内容 | 实际使用处的标题 | 「添加模型」；描述「搜索模型...」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/settings_configuration_screen.go:App.configModelsToolbar「添加模型」 |
| R-GM-171 | G-B2-01 | 状态 | 调色板被调用方覆盖 | 显示调用方文案；未覆盖才回退英文默认值 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-B2-01→internal/ui/command_palette.go:App.commandPaletteLayer；标记 = Hidden） |
| R-GM-172 | G-B2-02 | 状态 | 未观察（条件）：调色板打开 | 面板容器：弹出底、圆角、`h-full w-full`（在对话框内铺满） | — | 屏上出现 1 个面板 | 已实现（映射 = G-B2-02→internal/ui/command_palette.go:App.commandPaletteLayer；标记 = Hidden） |
| R-GM-173 | G-B2-02 | 样式 | 面板与遮罩同时在场时 | 面板底色比遮罩亮一档，圆角与对话框一致 | — | 判据 = 底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d；圆角 lg 8.75px；其余 = 文字与底色对比度 ≥ 4.5:1（正文）、≥ 3:1（大字与图标）（CL-CONTRAST） | 已实现（映射 = G-B2-02→internal/ui/command_palette.go:App.commandPaletteLayer；标记 = Width） |
| R-GM-174 | G-B2-02 | 交互 | 按 Esc 时 | 关闭调色板，回到设置页 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B2-02→internal/ui/command_palette.go:App.commandPaletteLayer；标记 = Key） |
| R-GM-175 | G-B2-02 | 关联 | 面板内列表出现、内容超长时 | 面板内含列表容器，列表可滚动 | — | 判据 = 滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-B2-02→internal/ui/command_palette.go:App.commandPaletteLayer；标记 = Tab@App.commandPaletteFooter） |
| R-GM-176 | G-B2-02 | 无障碍 | 读屏进入该元素时 | 读作对话框内容区 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-B2-02→internal/ui/command_palette.go:App.commandPaletteLayer；标记 = AccessHidden） |
| R-GM-177 | G-B2-02 | 异常 | 无选项可显示 | 面板仍渲染并给空态（G-B2-08） | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B2-02→internal/ui/command_palette.go:App.commandPaletteLayer；标记 = err@dialogWidth） |
| R-GM-178 | G-B2-02 | 内容 | 面板内只有搜索框与列表 | 无额外头部（标题 sr-only） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 已实现（映射 = G-B2-02→internal/ui/command_palette.go:App.commandPaletteLayer；标记 = 文案「命令面板」） |
| R-GM-179 | G-B2-02 | 状态 | 面板关闭 | 容器整体卸载，不留空壳 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-B2-02→internal/ui/command_palette.go:App.commandPaletteLayer；标记 = Hidden） |
| R-GM-180 | G-B2-03 | 状态 | 未观察（条件）：面板打开，输入框自动聚焦 | 搜索输入框：左置放大镜图标、无边框、占位为调用方文案 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-B2-03→internal/ui/command_palette.go:App.commandPaletteInput；标记 = Hidden@iconGlyph） |
| R-GM-181 | G-B2-03 | 内容 | 在输入框键入时 | 逐字过滤选项；匹配为 0 时显示空态 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-B2-03→internal/ui/command_palette.go:App.commandPaletteInput；标记 = 文案「搜索命令、会话、文件或设置...」等 2 条） |
| R-GM-182 | G-B2-03 | 交互 | 清空 | 恢复完整列表 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B2-03→internal/ui/command_palette.go:App.commandPaletteInput；标记 = Key） |
| R-GM-183 | G-B2-03 | 样式 | 输入框在屏上时 | 与命令面板同款：无边框、字号 14、图标 16 | — | 判据 = 字号 UI 正文 12.25px；其余 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；字号 12.25 / 行高 18 / 字重 400，说明文字 10.5 / 14（TX-TYPE-BODY / TX-TYPE-MICRO） | 已实现（映射 = G-B2-03→internal/ui/command_palette.go:App.commandPaletteInput；标记 = Height） |
| R-GM-184 | G-B2-03 | 无障碍 | 读屏进入该元素时 | 读作搜索框，可读名非空 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-B2-03→internal/ui/command_palette.go:App.commandPaletteInput；标记 = Label） |
| R-GM-185 | G-B2-03 | 键盘 | 按 ↑ 或 ↓ 时 | 在选项间移动高亮 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-B2-03→internal/ui/command_palette.go:App.commandPaletteInput；标记 = Key） |
| R-GM-186 | G-B2-03 | 内容 | 占位文案在屏上时 | 「搜索模型...」（调用方传入），未覆盖时用组件默认 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/composer.go:App.modelSelector「搜索模型...」 |
| R-GM-187 | G-B2-03 | 状态 | 输入为空 | 列表显示全部项 | — | 判据 = 计数 = 屏上可见项数、差 0（TX-EXACT 的计数形态） | 已实现（映射 = G-B2-03→internal/ui/command_palette.go:App.commandPaletteInput；标记 = Hidden@iconGlyph） |
| R-GM-188 | G-B2-04 | 状态 | 未观察（条件）：调用方传入 `heading` | 分组标题（12px 灰字）出现在该组第一项之前 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-B2-04→internal/ui/command_palette.go:App.commandPaletteGroupHeader；标记 = Hidden@iconGlyph） |
| R-GM-189 | G-B2-04 | 状态 | 未传 `heading` | 不渲染标题（不留空行） | — | 屏上 标题 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GM-190 | G-B2-04 | 样式 | 分组标题在屏上时 | 12px 次要色、左内边距与项文字对齐 | — | 判据 = 字号 UI 正文 12.25px；内边距 左缘 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28） | 已实现（映射 = G-B2-04→internal/ui/command_palette.go:App.commandPaletteGroupHeader；标记 = Height） |
| R-GM-191 | G-B2-04 | 无障碍 | 读屏进入该元素时 | 标题先于组内项 | — | 播报顺序 = 分组标题 → 组内项 | 已实现（映射 = G-B2-04→internal/ui/command_palette.go:App.commandPaletteGroupHeader；标记 = Label） |
| R-GM-192 | G-B2-04 | 异常 | 该组被过滤空 | 标题一起消失 | — | 屏上 分组标题 计数 = 0 | 未实现（映射到位但没有该维度的标记）：G-B2-04→internal/ui/command_palette.go:App.commandPaletteGroupHeader |
| R-GM-193 | G-B2-04 | 内容 | 未传 heading | 不渲染标题行（不留 12px 空行） | — | 屏上 标题行 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GM-194 | G-B2-04 | 状态 | 标题随组显隐 | 组内项被过滤空时标题一起消失 | — | 屏上 分组标题 计数 = 0 | 已实现（映射 = G-B2-04→internal/ui/command_palette.go:App.commandPaletteGroupHeader；标记 = Hidden@iconGlyph） |
| R-GM-195 | G-B2-04 | 异常 | heading 为空串 | 按未传处理，不渲染 | — | 屏上 分组标题 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GM-196 | G-B2-05 | 状态 | 未观察（条件）：有项且匹配筛选 | 选项行：可选图标 + 文字；选中态强调底 | — | 判据 = 底色 选中底 明 #f5f5f5 / 暗 #262626；字色 主文字 明 #0a0a0a / 暗 #ececec；其余 = 选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED） | 已实现（映射 = G-B2-05→internal/ui/command_palette.go:App.commandPaletteRow；标记 = Disabled） |
| R-GM-197 | G-B2-05 | 交互 | 点击 / Enter | 执行该项动作并关闭调色板 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-B2-05→internal/ui/command_palette.go:App.commandPaletteRow |
| R-GM-198 | G-B2-05 | 内容 | 项文字超长 | 单行 truncate，图标不裁 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-B2-05→internal/ui/command_palette.go:App.commandPaletteRow；标记 = 文案「，快捷键」等 3 条） |
| R-GM-199 | G-B2-05 | 样式 | 行在屏上时 | 高 32、圆角小、左内边距 8 | — | 判据 = 行高 UI 正文 17.5px；内边距 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28）/ 卡片外框 4.5；档位 3.5·7·10.5·14·21·28px | 已实现（映射 = G-B2-05→internal/ui/command_palette.go:App.commandPaletteRow；标记 = Height） |
| R-GM-200 | G-B2-05 | 无障碍 | 读屏进入该元素时 | 读作选项，占 1 行 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-B2-05→internal/ui/command_palette.go:App.commandPaletteRow；标记 = Label） |
| R-GM-201 | G-B2-05 | 异常 | 项被禁用 | 置灰且不可选（不被选中高亮） | — | 判据 = 底色 选中底 明 #f5f5f5 / 暗 #262626；字色 主文字 明 #0a0a0a / 暗 #ececec；其余 = 选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED） | 已实现（映射 = G-B2-05→internal/ui/command_palette.go:App.commandPaletteRow；标记 = reason） |
| R-GM-202 | G-B2-05 | 样式 | 选项行与分组标题同时在场时 | 行首缩进与分组标题文字对齐；图标 16px 在文字左 | — | 判据 = 内边距 左缘 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28）；其余 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0） | 已实现（映射 = G-B2-05→internal/ui/command_palette.go:App.commandPaletteRow；标记 = Height） |
| R-GM-203 | G-B2-05 | 状态 | 选中项被过滤掉 | 高亮落到新的首项 | — | 判据 = 底色 选中底 明 #f5f5f5 / 暗 #262626；字色 主文字 明 #0a0a0a / 暗 #ececec；其余 = 选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED） | 已实现（映射 = G-B2-05→internal/ui/command_palette.go:App.commandPaletteRow；标记 = Disabled） |
| R-GM-204 | G-B2-06 | 状态 | 未观察（条件）：调用方传 `shortcut` | 行尾显示快捷键小字（字距加大） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-B2-06→internal/ui/command_palette.go:App.commandPaletteActions；标记 = State） |
| R-GM-205 | G-B2-06 | 状态 | 未传 `shortcut` | 行尾不显示（不留空白占位） | — | 屏上 行尾小字 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GM-206 | G-B2-06 | 样式 | 快捷键 | 12px、字距加大、次要色 | — | 判据 = 字号 UI 正文 12.25px；其余 = 字号 12.25 / 行高 18 / 字重 400，说明文字 10.5 / 14（TX-TYPE-BODY / TX-TYPE-MICRO） | 已实现（映射 = G-B2-06→internal/ui/command_palette.go:App.commandPaletteRow；标记 = Height） |
| R-GM-207 | G-B2-06 | 无障碍 | 读屏进入该元素时 | 播报快捷键 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-B2-06→internal/ui/keymap.go:App.keybindingLabel；标记 = Label） |
| R-GM-208 | G-B2-06 | 异常 | 快捷键与系统冲突 | 仍显示实际绑定，不显示失效键位 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GM-209 | G-B2-06 | 内容 | 未传 shortcut | 行尾留空（不显示占位符） | — | 屏上 占位符 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GM-210 | G-B2-06 | 状态 | 项被禁用 | 快捷键仍显示（不因禁用隐藏） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 已实现（映射 = G-B2-06→internal/ui/command_palette.go:App.commandPaletteActions；标记 = State） |
| R-GM-211 | G-B2-06 | 异常 | shortcut 为空串 | 按未传处理 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-B2-06→internal/ui/command_palette.go:App.commandPaletteActions；标记 = reason） |
| R-GM-212 | G-B2-07 | 状态 | 未观察（条件）：调用方使用分隔线 | 组与组之间出现 1px 分隔线 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；色值取 token：主色 #349651/#63cb7b、正文 #0a0a0a/#ececec、次文 #737373/#8a8a8a、边框 #e5e5e5/#2e2e2e、警示 #e17100/#ffb900，偏差 0（CL-ACCENT / CL-TEXT / CL-TEXT-2 / CL-BORDER / CL-WARN） | 已实现（映射 = G-B2-07→internal/ui/command_palette.go:App.commandPaletteInput；标记 = Hidden@iconGlyph） |
| R-GM-213 | G-B2-07 | 关联 | 分组标题与分隔线同时在场时 | 可同时存在（标题 + 分隔线），不互相替代 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；色值取 token：主色 #349651/#63cb7b、正文 #0a0a0a/#ececec、次文 #737373/#8a8a8a、边框 #e5e5e5/#2e2e2e、警示 #e17100/#ffb900，偏差 0（CL-ACCENT / CL-TEXT / CL-TEXT-2 / CL-BORDER / CL-WARN） | 未实现（映射到位但没有该维度的标记）：G-B2-07→internal/ui/command_palette.go:App.commandPaletteInput |
| R-GM-214 | G-B2-07 | 样式 | 分隔线 | 1px 中性色、左右各留 4–8 | — | 判据 = 色值 边框 明 #e5e5e5 / 暗 #2e2e2e；内边距 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28）/ 卡片外框 4.5；档位 3.5·7·10.5·14·21·28px | 已实现（映射 = G-B2-07→internal/ui/command_palette.go:App.commandPaletteInput；标记 = Height） |
| R-GM-215 | G-B2-07 | 无障碍 | 读屏进入该元素时 | 语义为分隔符 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-B2-07→internal/ui/command_palette.go:App.commandPaletteInput；标记 = Label） |
| R-GM-216 | G-B2-07 | 异常 | 相邻组都空 | 分隔线随之消失 | — | 屏上 分隔线 计数 = 0 | 未实现（映射到位但没有该维度的标记）：G-B2-07→internal/ui/command_palette.go:App.commandPaletteInput |
| R-GM-217 | G-B2-07 | 内容 | 只有一组（无分隔线） | 不渲染任何分隔线 | — | 屏上 任何分隔线 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GM-218 | G-B2-07 | 状态 | 相邻组都空 | 线随之消失 | — | 屏上 分隔线 计数 = 0 | 已实现（映射 = G-B2-07→internal/ui/command_palette.go:App.commandPaletteInput；标记 = Hidden@iconGlyph） |
| R-GM-219 | G-B2-07 | 异常 | 分隔线被过滤到列表首尾 | 首尾不渲染线（不出现顶边/底边孤线） | — | 屏上 顶边 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GM-220 | G-B2-08 | 状态 | 未观察（条件）：有搜索词且匹配为 0 | 空态容器居中，上下 24px 留白 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-B2-08→internal/ui/command_palette.go:App.commandPaletteResults；标记 = Disabled@App.commandPaletteRow） |
| R-GM-221 | G-B2-08 | 内容 | 空态文案 | 用调用方文案；未传时给通用「无匹配项」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-B2-08→internal/ui/command_palette.go:App.commandPaletteResults；标记 = 文案「没有匹配项」等 3 条） |
| R-GM-222 | G-B2-08 | 样式 | 空态在屏上时 | 14px 次要色、居中 | — | 判据 = 字号 正文 14px；内边距 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28）/ 卡片外框 4.5；档位 3.5·7·10.5·14·21·28px | 已实现（映射 = G-B2-08→internal/ui/command_palette.go:App.commandPaletteResults；标记 = TextColor） |
| R-GM-223 | G-B2-08 | 无障碍 | 读屏进入该元素时 | 读出空态文案 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-B2-08→internal/ui/command_palette.go:App.commandPaletteResults；标记 = Role） |
| R-GM-224 | G-B2-08 | 异常 | 仍有项但全部禁用 | 不显示空态（空态只表示匹配为 0） | — | 屏上 空态 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GM-225 | G-B2-08 | 内容 | 未传空态文案 | 显示通用文案（如「无匹配项」） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-B2-08→internal/ui/command_palette.go:App.commandPaletteResults；标记 = 文案「没有匹配项」等 3 条） |
| R-GM-226 | G-B2-08 | 状态 | 搜索词为空 | 不显示空态（列表全量） | — | 屏上 空态 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GM-227 | G-B2-08 | 样式 | 空态位置 | 居中于列表区，上下 24px 留白 | — | 判据 = 内边距 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28）/ 卡片外框 4.5；档位 3.5·7·10.5·14·21·28px | 已实现（映射 = G-B2-08→internal/ui/command_palette.go:App.commandPaletteResults；标记 = TextColor） |
| R-GM-228 | G-B2-09 | 状态 | 未观察（条件）：面板打开 | 滚动列表：最大高 300，超出滚动 | — | 列表高 ≤300 | 已实现（映射 = G-B2-09→internal/ui/command_palette.go:App.commandPaletteResults；标记 = Disabled@App.commandPaletteRow） |
| R-GM-229 | G-B2-09 | 内容 | 项超过可视数 | 滚动；高亮项移动时列表跟随 | — | 判据 = 底色 选中底 明 #f5f5f5 / 暗 #262626；字色 主文字 明 #0a0a0a / 暗 #ececec；其余 = 选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED）；滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0 | 已实现（映射 = G-B2-09→internal/ui/command_palette.go:App.commandPaletteResults；标记 = 文案「没有匹配项」等 3 条） |
| R-GM-230 | G-B2-09 | 样式 | 列表 | 无边框、无底色，滚动条细 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0 | 已实现（映射 = G-B2-09→internal/ui/command_palette.go:App.commandPaletteResults；标记 = TextColor） |
| R-GM-231 | G-B2-09 | 无障碍 | 读屏进入该元素时 | 读作列表，项为列表项 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-B2-09→internal/ui/command_palette.go:App.commandPaletteResults；标记 = Role） |
| R-GM-232 | G-B2-09 | 异常 | 列表为空 | 列表不出现，由空态占位 | — | 屏上 滚动列表 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GM-233 | G-B2-09 | 内容 | 列表项很多 | 滚动到底部不跳回顶部 | — | 判据 = 滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-B2-09→internal/ui/command_palette.go:App.commandPaletteResults；标记 = 文案「没有匹配项」等 3 条） |
| R-GM-234 | G-B2-09 | 状态 | 列表为空 | 列表容器不渲染，由空态占位 | — | 屏上 滚动列表 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GM-235 | G-B2-09 | 样式 | 滚动条 | 细滚动条、仅在滚动时显现 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0 | 已实现（映射 = G-B2-09→internal/ui/command_palette.go:App.commandPaletteResults；标记 = TextColor） |

### 3.3 命令域维度覆盖说明

29 项全部 ≥6 维。G-B2-04/G-B2-07（分组标题与分隔线）到 6 维：不可点、无过渡、无自身状态机；其「交互 / 动画」写在各自行内的「无动作 / 无反馈」。

---

## 4. 提问（22 元素）

提问卡（elicitation）内联在输入区上沿，可收起，多字段时逐题展示，带截止时间时显示倒计时。**采集现状：60 份态里没有一份停在待答态**；唯一旁证是 `runtime/session/turns/11-tool-ask-user-pending.png`（只有截图、无 DOM 导出），`10-tool-ask-user-timeout.png` 是超时后的时间线行、不是本卡。故 22 项全部按条件写。

| 需求 ID | 元素 ID | 维度 | 场景 | 期望 | 异常 | 验收 | 状态 |
|---|---|---|---|---|---|---|---|
| R-GQ-001 | G-D1-01 | 状态 | 未观察（条件）：会话收到 `elicitation` 且停在待答态 | 卡片在场：圆角 10px、1px 描边，纵向三段（头 / 可滚动体 / 脚），贴输入区上沿 | — | 屏上出现 1 张卡，三段结构可辨 | 已实现（映射 = G-D1-01→internal/ui/question.go:App.questionCard；标记 = State@App.syncQuestionCard） |
| R-GQ-002 | G-D1-01 | 状态 | 审批卡在场时又到达提问卡 | 两者不同时占同一位置：按到达顺序排队，一次只显示一张 | — | 卡片计数为 1 | 已实现（映射 = G-D1-01→internal/ui/question.go:App.questionCard；标记 = State@App.syncQuestionCard） |
| R-GQ-003 | G-D1-01 | 内容 | 多字段请求 | 卡片高度固定（体部滚动），切题时卡片不跳动 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-D1-01→internal/ui/question.go:App.questionCard；标记 = 文案「提问卡」等 2 条） |
| R-GQ-004 | G-D1-01 | 交互 | 滚动体部 | 头（字段标题/进度）与脚（按钮）不随之滚动 | — | 判据 = 滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-D1-01→internal/ui/question.go:App.questionCard；标记 = Pressed@App.questionHead） |
| R-GQ-005 | G-D1-01 | 动画 | 出现 / 答复后消失 | ≤150ms；不推挤时间线内容跳动 | 减少动效下瞬切 | 判据 = 动效 布局 250ms / 颜色 150ms / 折叠 300ms；缓动 cubic-bezier(0.16, 1, 0.3, 1) / cubic-bezier(0.4, 0, 0.2, 1) | 未实现（映射到位但没有该维度的标记）：G-D1-01→internal/ui/question.go:App.questionCard |
| R-GQ-006 | G-D1-01 | 样式 | 分别切到浅色与深色主题时 | 白底/上层底 + 1px 描边 + 圆角 10；深色主题下描边更明显 | — | 判据 = 底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d；描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px | 已实现（映射 = G-D1-01→internal/ui/question.go:App.questionCard；标记 = Width） |
| R-GQ-007 | G-D1-01 | 无障碍 | 读屏进入该元素时 | 作为对话框/区域，读出名由字段标题承担；焦点进入卡内 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-D1-01→internal/ui/question.go:App.questionCard；标记 = Role） |
| R-GQ-008 | G-D1-01 | 异常 | 答复时断线 | 卡片保留并显示错误文字（G-D1-17） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-D1-01→internal/ui/question.go:App.questionCard；标记 = err@App.questionFoot） |
| R-GQ-009 | G-D1-01 | 记忆 | 切到其它会话再切回时 | 未答复的请求仍在；已答的不重放 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-D1-01→internal/ui/question.go:App.questionCard |
| R-GQ-010 | G-D1-01 | 窄窗 | 窗口收窄到最窄宽度时 | 卡片宽随主区收缩，选项行折行，按钮不被裁 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 未实现（映射到位但没有该维度的标记）：G-D1-01→internal/ui/question.go:App.questionCard |
| R-GQ-011 | G-D1-02 | 内容 | 未观察（条件）：字段或 schema 带 title | 头部标题取 title（14px 半粗），超长 truncate | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-D1-02→internal/ui/question.go:App.questionHead；标记 = 文案「第 %d 题，共 %d 题」） |
| R-GQ-012 | G-D1-02 | 内容 | 标题优先级 | 取字段 `title` → schema `title` → 回退（G-D1-03） | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-D1-02→internal/ui/question.go:App.questionHead；标记 = 文案「第 %d 题，共 %d 题」） |
| R-GQ-013 | G-D1-02 | 状态 | 多字段切题 | 标题随当前题切换，不回退到第一题 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-D1-02→internal/ui/question.go:App.questionHead；标记 = Disabled@questionCollapseButton） |
| R-GQ-014 | G-D1-02 | 样式 | 当前字段无标题、显示回退文案时 | 14px 半粗主前景；与进度徽标同一行 | — | 判据 = 字号 正文 14px；字重 半粗 600；行高 UI 正文 17.5px | 已实现（映射 = G-D1-02→internal/ui/question.go:App.questionHead；标记 = TextColor） |
| R-GQ-015 | G-D1-02 | 无障碍 | 读屏进入该元素时 | 作为卡片标题/当前题名读出 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-D1-02→internal/ui/question.go:App.questionHead；标记 = Label） |
| R-GQ-016 | G-D1-02 | 异常 | 标题含换行 | 压成单行 truncate | — | 标题 1 行 | 未实现（映射到位但没有该维度的标记）：G-D1-02→internal/ui/question.go:App.questionHead |
| R-GQ-017 | G-D1-02 | 内容 | 标题超长（本地化后） | truncate 加省略号，悬停出完整标题 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-D1-02→internal/ui/question.go:App.questionHead；标记 = 文案「第 %d 题，共 %d 题」） |
| R-GQ-018 | G-D1-02 | 样式 | 标题与进度徽标同处头部一行时 | 同处头部一行：标题左、徽标右，互不重叠 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px | 已实现（映射 = G-D1-02→internal/ui/question.go:App.questionHead；标记 = TextColor） |
| R-GQ-019 | G-D1-03 | 内容 | 未观察（条件）：字段与 schema 都无标题 | 标题回退逐字「需要你的回答」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/question.go:questionCopyTitleFallback「需要你的回答」 |
| R-GQ-020 | G-D1-03 | 状态 | 有标题 | 回退文案不出现 | — | 屏上 标题回退文案 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GQ-021 | G-D1-03 | 样式 | 标题与回退文案先后出现时 | 同字号同字重，只是文案不同 | — | 判据 = 字号 页面标题 17.5px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a；其余 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；字号 12.25 / 行高 18 / 字重 400，说明文字 10.5 / 14（TX-TYPE-BODY / TX-TYPE-MICRO） | 未实现（映射到位但没有该维度的标记）：G-D1-03→internal/ui/question.go:questionTitle |
| R-GQ-022 | G-D1-03 | 无障碍 | 读屏进入该元素时 | 读出该句 | — | 读屏文本 = 「需要你的回答」 | 未实现（映射到位但没有该维度的标记）：G-D1-03→internal/ui/question.go:questionTitle |
| R-GQ-023 | G-D1-03 | 异常 | 标题为空串 | 视为无标题，走回退 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-D1-03→internal/ui/question.go:questionTitle |
| R-GQ-024 | G-D1-03 | 内容 | 字段 title 为空串 | 视为无标题 → 走回退「需要你的回答」 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/question.go:questionCopyTitleFallback「需要你的回答」 |
| R-GQ-025 | G-D1-03 | 状态 | 多字段中某题无标题 | 仅该题显示回退文案，其它题按各自标题 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-D1-03→internal/ui/question.go:questionTitle |
| R-GQ-026 | G-D1-03 | 记忆 | 切到下一题再回来 | 回退标题仍在（按该字段重新判定） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-D1-03→internal/ui/question.go:questionTitle |
| R-GQ-027 | G-D1-04 | 内容 | 未观察（条件）：一次 elicitation 带 ≥2 字段 | 进度徽标「{当前}/{总数}」，灰底 10px 圆角小字 | — | 屏上出现形如「2/5」的徽标 | 已实现（映射 = G-D1-04→internal/ui/question.go:App.questionBody；标记 = 文案「筛选选项…」等 6 条） |
| R-GQ-028 | G-D1-04 | 内容 | 单字段 | 徽标不出现 | — | 屏上 进度徽标 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GQ-029 | G-D1-04 | 关联 | 前后切题时 | 前进/后退时数字同步变化 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-D1-04→internal/ui/question.go:App.questionBody；标记 = Select） |
| R-GQ-030 | G-D1-04 | 样式 | 徽标 | 灰底、圆角 4、10px；与标题同行右端或紧随标题 | — | 判据 = 字号 小徽标 10px；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px；底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d | 已实现（映射 = G-D1-04→internal/ui/question.go:App.questionBody；标记 = Height） |
| R-GQ-031 | G-D1-04 | 无障碍 | 读屏进入该元素时 | 播报「第 2 题，共 5 题」 | — | 读屏文本 = 「第 2 题，共 5 题」 | 已实现（映射 = G-D1-04→internal/ui/question.go:App.questionBody；标记 = Label） |
| R-GQ-032 | G-D1-04 | 异常 | 字段数统计不准（流式追加） | 以当前字段列表长度为准，不显示「2/0」 | — | 分母 ≥1 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GQ-033 | G-D1-04 | 内容 | 字段数为 10 | 徽标显示「{当前}/10」，不缩写 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-D1-04→internal/ui/question.go:App.questionBody；标记 = 文案「筛选选项…」等 6 条） |
| R-GQ-034 | G-D1-04 | 样式 | 进度徽标紧贴标题右侧时 | 10px 小字、灰底圆角 4，紧贴标题右侧 | — | 判据 = 字号 小徽标 10px；底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d | 已实现（映射 = G-D1-04→internal/ui/question.go:App.questionBody；标记 = Height） |
| R-GQ-035 | G-D1-05 | 内容 | 未观察（条件）：卡片收起且未答 | 收起态状态行逐字「待选择」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/ui/question.go:questionCopyPending「待选择」 |
| R-GQ-036 | G-D1-05 | 内容 | 收起且选项多 | 逐字「待选择 · {count} 个选项」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-D1-05→internal/ui/question.go:App.questionCollapsedStatus；标记 = 文案「待选择 · %d 个选项」等 3 条） |
| R-GQ-037 | G-D1-05 | 内容 | 收起且已答单值 | 逐字「已选：{answer}」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-D1-05→internal/ui/question.go:App.questionCollapsedStatus；标记 = 文案「待选择 · %d 个选项」等 3 条） |
| R-GQ-038 | G-D1-05 | 内容 | 收起且已答多值 | 逐字「已选 {count} 项」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-D1-05→internal/ui/question.go:App.questionCollapsedStatus；标记 = 文案「待选择 · %d 个选项」等 3 条） |
| R-GQ-039 | G-D1-05 | 状态 | 四分支互斥 | 同一时刻只显示一句（按已答情况） | — | 文案计数为 1 | 已实现（映射 = G-D1-05→internal/ui/question.go:App.questionCollapsedStatus；标记 = State@questionCardState.answerAt） |
| R-GQ-040 | G-D1-05 | 样式 | 状态行在屏上时 | 12px 次要色、单行 truncate、位于收起态卡片头部 | — | 判据 = 字号 UI 正文 12.25px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a；其余 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-D1-05→internal/ui/question.go:App.questionCollapsedStatus；标记 = TextColor） |
| R-GQ-041 | G-D1-05 | 无障碍 | 读屏进入该元素时 | 读出当前状态句 | — | 读屏文本 = 「待选择」 | 已实现（映射 = G-D1-05→internal/ui/question.go:App.questionCollapsedStatus；标记 = Label） |
| R-GQ-042 | G-D1-05 | 异常 | 答案超长 | 「已选：{answer}」截断，悬停出完整值 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-D1-05→internal/ui/question.go:App.questionCollapsedStatus；测试提及 internal/ui/reconcile_scope_test.go:包级 |
| R-GQ-043 | G-D1-06 | 内容 | 未观察（条件）：请求带 `deadlineAt` | 倒计时逐字「剩余 {m:ss}」，12px 等宽数字，`role=timer` | — | 屏上出现形如「剩余 4:58」 | 已实现（映射 = G-D1-06→internal/ui/question.go:App.questionTimer；标记 = 文案「图标：」@iconGlyph） |
| R-GQ-044 | G-D1-06 | 内容 | 剩余不足 1 分钟 | 仍为 `m:ss` 形态（如「剩余 0:42」），不改成秒 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-D1-06→internal/ui/question.go:App.questionTimer；标记 = 文案「图标：」@iconGlyph） |
| R-GQ-045 | G-D1-06 | 内容 | 到点 | 倒计时归零后卡片转入超时处理（服务端自动回答或提示） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 已实现（映射 = G-D1-06→internal/ui/question.go:App.questionTimer；标记 = 文案「图标：」@iconGlyph） |
| R-GQ-046 | G-D1-06 | 状态 | 已作答 | 倒计时消失（不再显示剩余） | — | 屏上 倒计时 计数 = 0 | 已实现（映射 = G-D1-06→internal/ui/question.go:App.questionTimer；标记 = Hidden@iconGlyph） |
| R-GQ-047 | G-D1-06 | 动画 | 每秒跳变 | 数字逐秒变化，不整行重排；不闪 | 减少动效下仍逐秒更新 | 连续观察 10s 无抖动 | 已实现（映射 = G-D1-06→internal/ui/question.go:App.questionTimer；标记 = Duration@questionClock） |
| R-GQ-048 | G-D1-06 | 样式 | 等宽文本渲染后 | 12px 等宽、次要色或警示色（临近截止转警示） | — | 判据 = 字号 UI 正文 12.25px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-D1-06→internal/ui/question.go:App.questionTimer；标记 = Font） |
| R-GQ-049 | G-D1-06 | 无障碍 | 读屏进入该元素时 | `role=timer` 只播报关键时点（如最后 10 秒），不逐秒刷屏 | — | 读屏文本 = 「剩余 {m:ss}」 | 已实现（映射 = G-D1-06→internal/ui/question.go:App.questionTimer；标记 = Label） |
| R-GQ-050 | G-D1-06 | 异常 | 客户端时钟偏移 | 以服务端返回的时间差计算，不显示负数 | — | 屏上 负数 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GQ-051 | G-D1-07 | 内容 | 未观察（条件）：卡片在场 | 头部右侧文字小按钮「收起」，带下箭头 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（逐字）：internal/ui/question.go:questionCopyCollapse「收起」 |
| R-GQ-052 | G-D1-07 | 状态 | 已收起 | 同一按钮文案变「展开」，箭头旋转 180° | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/question.go:questionCopyExpand「展开」 |
| R-GQ-053 | G-D1-07 | 交互 | 点击收起 | 卡片折叠为单行（保留状态行 G-D1-05 与按钮），体部与脚部隐藏 | — | 判据 = 控件尺寸 按钮高 行内 21 / 小 25 / 次 28 / 常规 32 / 大 35；其余 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 已实现（映射 = G-D1-07→internal/ui/question.go:App.questionHead；标记 = Pressed） |
| R-GQ-054 | G-D1-07 | 交互 | 点击展开 | 恢复完整卡片，滚动位置回到顶部 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-D1-07→internal/ui/question.go:App.questionHead；标记 = Pressed） |
| R-GQ-055 | G-D1-07 | 动画 | 收 / 展 | 高度过渡 ≤150ms；减少动效下瞬切 | 减少动效下瞬切 | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；时长 = 期望列给的毫秒值（±0ms），减少动效时 none（§6 动效表） | 未实现（映射到位但没有该维度的标记）：G-D1-07→internal/ui/question.go:App.questionHead |
| R-GQ-056 | G-D1-07 | 样式 | 文字按钮在屏上时 | 12px 次要色、无边框、箭头 12px | — | 判据 = 字号 UI 正文 12.25px；其余 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0） | 已实现（映射 = G-D1-07→internal/ui/question.go:App.questionHead；标记 = TextColor） |
| R-GQ-057 | G-D1-07 | 无障碍 | 读屏进入该元素时 | 读作按钮「收起」/「展开」并带展开态 | — | 读屏文本 = 「收起」 / 「展开」 | 已实现（逐字）：internal/ui/question.go:questionCopyCollapse「收起」 |
| R-GQ-058 | G-D1-07 | 内容 | 收起态下点「展开」 | 恢复完整卡片，答案与滚动位置按收起前恢复 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0 | 已实现（映射 = G-D1-07→internal/ui/question.go:App.questionHead；标记 = 文案「第 %d 题，共 %d 题」） |
| R-GQ-059 | G-D1-07 | 状态 | 提交进行中时 | 「收起」禁用（避免收起后看不到结果） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/ui/question.go:questionCopyCollapse「收起」 |
| R-GQ-060 | G-D1-08 | 内容 | 未观察（条件）：请求带 message | 提问正文多行文字（13px），保留换行 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-D1-08→internal/ui/question.go:App.questionBody；标记 = 文案「筛选选项…」等 6 条） |
| R-GQ-061 | G-D1-08 | 内容 | message 为空 | 正文区不出现（不留空块） | — | 屏上 多行文字 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GQ-062 | G-D1-08 | 样式 | 提问正文与字段描述同屏时 | 13px 主前景、行高较松；与描述（14px 次要色）区分开 | — | 判据 = 字号 正文 14px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a；行高 UI 正文 17.5px | 已实现（映射 = G-D1-08→internal/ui/question.go:App.questionBody；标记 = Height） |
| R-GQ-063 | G-D1-08 | 交互 | 选中屏上文本后复制时 | 可复制正文 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-D1-08→internal/ui/question.go:App.questionBody；标记 = Key@App.questionOptions） |
| R-GQ-064 | G-D1-08 | 无障碍 | 读屏进入该元素时 | 读出正文全文 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-D1-08→internal/ui/question.go:App.questionBody；标记 = Label） |
| R-GQ-065 | G-D1-08 | 异常 | 正文超长 | 体部内滚动，不撑破卡片 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0 | 未实现（映射到位但没有该维度的标记）：G-D1-08→internal/ui/question.go:App.questionBody |
| R-GQ-066 | G-D1-08 | 内容 | message 含 Markdown | 按纯文本显示（不渲染 Markdown）或按实现一致，二者固定 | — | 屏上 多行文字 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GQ-067 | G-D1-08 | 样式 | 提问正文与字段描述同屏时 | 正文 13px 主前景；描述 14px 次要色，两者不同级 | — | 判据 = 字号 正文 14px | 已实现（映射 = G-D1-08→internal/ui/question.go:App.questionBody；标记 = Height） |
| R-GQ-068 | G-D1-09 | 内容 | 未观察（条件）：字段带 description | 说明文字（14px 灰）保留换行，位于当前题标题之后 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM） | 已实现（映射 = G-D1-09→internal/ui/question.go:App.questionBody；标记 = 文案「筛选选项…」等 6 条） |
| R-GQ-069 | G-D1-09 | 内容 | description 为空 | 不出现空行 | — | 屏上 空行 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GQ-070 | G-D1-09 | 关联 | 提问正文与字段描述上下排列时 | 正文（message）在上、字段描述在下，两者不合并 | — | 屏上 说明文字 计数 = 1 | 已实现（映射 = G-D1-09→internal/ui/question.go:App.questionBody；标记 = Select） |
| R-GQ-071 | G-D1-09 | 样式 | 说明段在屏上时 | 14px 次要色、行高 1.5 | — | 判据 = 字号 正文 14px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-D1-09→internal/ui/question.go:App.questionBody；标记 = Height） |
| R-GQ-072 | G-D1-09 | 无障碍 | 读屏进入该元素时 | 读出说明全文 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-D1-09→internal/ui/question.go:App.questionBody；标记 = Label） |
| R-GQ-073 | G-D1-09 | 异常 | 说明超长 | 折行，不截断 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 未实现（映射到位但没有该维度的标记）：G-D1-09→internal/ui/question.go:App.questionBody |
| R-GQ-074 | G-D1-09 | 内容 | 描述含换行 | 保留换行显示 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-D1-09→internal/ui/question.go:App.questionBody；标记 = 文案「筛选选项…」等 6 条） |
| R-GQ-075 | G-D1-09 | 内容 | 描述为空串 | 不渲染该行（不留空行） | — | 屏上 该行 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GQ-076 | G-D1-10 | 状态 | 未观察（条件）：当前字段选项数 >8 | 选项列表上方出现筛选输入框，占位逐字「筛选选项…」（中文省略号为单个字符） | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-D1-10→internal/ui/question.go:App.questionBody；标记 = State@questionCardState.questionSlot） |
| R-GQ-077 | G-D1-10 | 状态 | 选项 ≤8 | 筛选框不出现 | — | 屏上 输入框 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GQ-078 | G-D1-10 | 交互 | 在输入框键入时 | 选项实时过滤；高亮不丢 | — | 判据 = 底色 选中底 明 #f5f5f5 / 暗 #262626；字色 主文字 明 #0a0a0a / 暗 #ececec；其余 = 选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED） | 已实现（映射 = G-D1-10→internal/ui/question.go:App.questionBody；标记 = Key@App.questionOptions） |
| R-GQ-079 | G-D1-10 | 样式 | 输入框在屏上时 | 无边框、浅底、12–13px；左内边距 8 | — | 判据 = 字号 UI 正文 12.25px；底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d | 已实现（映射 = G-D1-10→internal/ui/question.go:App.questionBody；标记 = Height） |
| R-GQ-080 | G-D1-10 | 无障碍 | 读屏进入该元素时 | 读作搜索框，可读名为占位文案 | — | 读屏文本 = 「筛选选项…」 | 已实现（映射 = G-D1-10→internal/ui/question.go:App.questionBody；标记 = Label） |
| R-GQ-081 | G-D1-10 | 异常 | 过滤后无匹配 | 显示「没有匹配的选项」（G-D1-13） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/question.go:questionCopyNoMatch「没有匹配的选项」 |
| R-GQ-082 | G-D1-10 | 内容 | 筛选后剩 1 项 | 只显示 1 行选项，筛选框保留 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-D1-10→internal/ui/question.go:App.questionBody；标记 = 文案「筛选选项…」等 6 条） |
| R-GQ-083 | G-D1-10 | 状态 | 选项数从 9 降到 8 | 筛选框随之消失（阈值判定） | — | 屏上 输入框 计数 = 0 | 已实现（映射 = G-D1-10→internal/ui/question.go:App.questionBody；标记 = State@questionCardState.questionSlot） |
| R-GQ-084 | G-D1-11 | 状态 | 未观察（条件）：字段为 string 且带 `oneOf` | 每个选项一行：单选圈 + 文案；选中为主色边框 + 浅主色底 | — | 判据 = 底色 选中底 明 #f5f5f5 / 暗 #262626；字色 主文字 明 #0a0a0a / 暗 #ececec；其余 = 选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED） | 已实现（映射 = G-D1-11→internal/ui/question.go:App.questionOptions；标记 = State@questionCardState.questionSlot） |
| R-GQ-085 | G-D1-11 | 交互 | 鼠标点击选项行时 | 选中该项，单选互斥（旧选中取消） | — | 同时只有 1 项选中 | 已实现（映射 = G-D1-11→internal/ui/question.go:App.questionOptions；标记 = Key） |
| R-GQ-086 | G-D1-11 | 键盘 | ↑/↓ + Space | 可移动并选中 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-D1-11→internal/ui/question.go:App.questionOptions；标记 = Key） |
| R-GQ-087 | G-D1-11 | 内容 | 选项文案超长 | 折行或 truncate，行高随之调整 | — | 判据 = 行高 UI 正文 17.5px；间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px | 已实现（映射 = G-D1-11→internal/ui/question.go:App.questionOptions；标记 = 文案「没有匹配的选项」） |
| R-GQ-088 | G-D1-11 | 样式 | 选中态 | 主色 1px 边框 + 主色 8–10% 底；未选中为中性描边 | — | 判据 = 色值 边框 明 #e5e5e5 / 暗 #2e2e2e；底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d | 已实现（映射 = G-D1-11→internal/ui/question.go:App.questionOptions；标记 = TextColor） |
| R-GQ-089 | G-D1-11 | 无障碍 | 读屏进入该元素时 | 读作单选项，含选中态 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-D1-11→internal/ui/question.go:App.questionOptions；标记 = Label） |
| R-GQ-090 | G-D1-11 | 异常 | 选项值为空串 | 仍成行（合法选项），不跳过 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-D1-11→internal/ui/question.go:App.questionOptions |
| R-GQ-091 | G-D1-11 | 内容 | 选项无描述 | 只显示文案一行 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-D1-11→internal/ui/question.go:App.questionOptions；标记 = 文案「没有匹配的选项」） |
| R-GQ-092 | G-D1-12 | 状态 | 未观察（条件）：字段为 array 且带 `anyOf` | 每个选项一行：方形勾选框（选中显示对勾） | — | 判据 = 选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED） | 已实现（映射 = G-D1-12→internal/ui/question.go:App.questionOptions；标记 = State@questionCardState.questionSlot） |
| R-GQ-093 | G-D1-12 | 交互 | 鼠标点击选项行时 | 勾选/取消，多选可同时勾选 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-D1-12→internal/ui/question.go:App.questionOptions；标记 = Key） |
| R-GQ-094 | G-D1-12 | 键盘 | 按 Space 时 | 切换勾选 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-D1-12→internal/ui/question.go:App.questionOptions；标记 = Key） |
| R-GQ-095 | G-D1-12 | 内容 | 全选 / 全不选 | 允许多选为空（是否必填由 G-D1-17 判定） | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-D1-12→internal/ui/question.go:App.questionOptions；标记 = 文案「没有匹配的选项」） |
| R-GQ-096 | G-D1-12 | 样式 | 勾选框 | 14px 方形、圆角 2、选中填主色 + 白勾 | — | 判据 = 圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-D1-12→internal/ui/question.go:App.questionOptions；标记 = TextColor） |
| R-GQ-097 | G-D1-12 | 无障碍 | 读屏进入该元素时 | 读作复选项，含勾选态 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-D1-12→internal/ui/question.go:App.questionOptions；标记 = Label） |
| R-GQ-098 | G-D1-12 | 异常 | 选项重复 | 各自成行、各自可勾 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-D1-12→internal/ui/question.go:App.questionOptions；测试提及 internal/ui/reconcile_scope_main_test.go:包级 |
| R-GQ-099 | G-D1-12 | 内容 | 勾选数 = 选项数 | 无「全选」按钮（勾选状态各自表达） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/plugin_manage.go:pluginMarketInstallDesc「全选」 |
| R-GQ-100 | G-D1-12 | 内容 | 选项无描述 | 只显示文案一行 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-D1-12→internal/ui/question.go:App.questionOptions；标记 = 文案「没有匹配的选项」） |
| R-GQ-101 | G-D1-13 | 内容 | 未观察（条件）：筛选后无匹配 | 空态文字逐字「没有匹配的选项」（14px 灰） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/question.go:questionCopyNoMatch「没有匹配的选项」 |
| R-GQ-102 | G-D1-13 | 状态 | 有匹配项时 | 空态不出现 | — | 屏上 空态文字 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GQ-103 | G-D1-13 | 关联 | 筛选框展开后仍无匹配选项时 | 只在筛选框存在时可能出现 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-D1-13→internal/ui/question.go:App.questionOptions；标记 = Select） |
| R-GQ-104 | G-D1-13 | 样式 | 空态在屏上时 | 14px 次要色、左内边距与选项文字对齐 | — | 判据 = 字号 正文 14px；内边距 左缘 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28） | 已实现（映射 = G-D1-13→internal/ui/question.go:App.questionOptions；标记 = TextColor） |
| R-GQ-105 | G-D1-13 | 无障碍 | 读屏进入该元素时 | 读出该句 | — | 读屏文本 = 「没有匹配的选项」 | 已实现（映射 = G-D1-13→internal/ui/question.go:App.questionOptions；标记 = Label） |
| R-GQ-106 | G-D1-13 | 内容 | 筛选框为空 | 空态不出现（未筛选时不判空） | — | 屏上 空态文字 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GQ-107 | G-D1-13 | 异常 | 筛选后选项全部隐藏 | 行数与空态同时正确（0 行 + 空态句） | — | 行 计数 = 0；空态文字 计数 = 1 | 未实现（映射到位但没有该维度的标记）：G-D1-13→internal/ui/question.go:App.questionOptions |
| R-GQ-108 | G-D1-13 | 样式 | 多条空态句先后出现时 | 与其它空态句同字号同色，仅文案不同 | — | 判据 = 字号 空态主标题 21px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-D1-13→internal/ui/question.go:App.questionOptions；标记 = TextColor） |
| R-GQ-109 | G-D1-14 | 状态 | 未观察（条件）：字段为无选项数组 | 多行输入框，占位逐字「每行输入一个值」，1 行起、最多 4 行后内部滚动 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（逐字）：internal/ui/question.go:questionCopyFreeform「每行输入一个值」 |
| R-GQ-110 | G-D1-14 | 内容 | 输入 5 行以上 | 框高停在 4 行，内部滚动；换行保留 | — | 框高 ≤4 行 | 已实现（映射 = G-D1-14→internal/ui/question.go:App.questionBody；标记 = 文案「筛选选项…」等 6 条） |
| R-GQ-111 | G-D1-14 | 交互 | 按 Enter 时 | 换行（不提交） | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GQ-112 | G-D1-14 | 内容 | 每行一个值 | 提交时按行切分为数组，丢弃空行 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-D1-14→internal/ui/question.go:App.questionBody；标记 = 文案「筛选选项…」等 6 条） |
| R-GQ-113 | G-D1-14 | 样式 | 输入框在屏上时 | 浅底、圆角 6、12–13px、行高 1.5 | — | 判据 = 底色 输入框 明 #ffffff / 暗 #0c0c0c；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px；字号 UI 正文 12.25px | 已实现（映射 = G-D1-14→internal/ui/question.go:App.questionBody；标记 = Height） |
| R-GQ-114 | G-D1-14 | 无障碍 | 读屏进入该元素时 | 读作多行文本输入框 | — | 读屏文本 = 「每行输入一个值」 | 已实现（映射 = G-D1-14→internal/ui/question.go:App.questionBody；标记 = Label） |
| R-GQ-115 | G-D1-14 | 内容 | 输入为空 | 提交时该字段按未答处理（是否必填见 G-D1-17） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 已实现（映射 = G-D1-14→internal/ui/question.go:App.questionBody；标记 = 文案「筛选选项…」等 6 条） |
| R-GQ-116 | G-D1-14 | 状态 | 字段为必填且为空 | 「提交」禁用（G-D1-22） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/ui/command_palette.go:App.commandPaletteActions「提交」 |
| R-GQ-117 | G-D1-15 | 状态 | 未观察（条件）：字段 `allowCustom` 且有选项 | 选项列表下方出现标签「自定义答案」（12px 灰）+ 多行输入框（占位「输入自定义答案」） | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（逐字）：internal/ui/question.go:questionCopyCustom「自定义答案」 |
| R-GQ-118 | G-D1-15 | 交互 | 在自定义框输入 | 该输入接管答案（覆盖选项选择），选项选中态清除 | — | 判据 = 底色 选中底 明 #f5f5f5 / 暗 #262626；字色 主文字 明 #0a0a0a / 暗 #ececec；其余 = 选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED） | 未实现（映射到位但没有该维度的标记）：G-D1-15→internal/ui/question.go:App.questionCustomField；测试提及 internal/ui/reconcile_scope_main_test.go:包级 |
| R-GQ-119 | G-D1-15 | 内容 | 自定义为空 | 回到选项选择（自定义框空即不生效） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 未实现（映射到位但没有该维度的标记）：G-D1-15→internal/ui/question.go:App.questionCustomField；测试提及 internal/ui/reconcile_scope_main_test.go:包级 |
| R-GQ-120 | G-D1-15 | 样式 | 标签与输入框同时在场时 | 标签 12px 次要色；输入框同 G-D1-14 样式 | — | 判据 = 字号 UI 正文 12.25px；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px | 已实现（映射 = G-D1-15→internal/ui/question.go:App.questionCustomField；标记 = TextColor） |
| R-GQ-121 | G-D1-15 | 无障碍 | 读屏进入该元素时 | 标签与输入框成对读出 | — | 播报顺序 = 自定义答案 → 输入自定义答案 | 已实现（映射 = G-D1-15→internal/ui/question.go:App.questionCustomField；标记 = Label） |
| R-GQ-122 | G-D1-15 | 内容 | 自定义输入多行 | 按整段作为答案（不按行切分） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 未实现（映射到位但没有该维度的标记）：G-D1-15→internal/ui/question.go:App.questionCustomField；测试提及 internal/ui/reconcile_scope_main_test.go:包级 |
| R-GQ-123 | G-D1-15 | 状态 | 选项已选中后再输入自定义 | 自定义生效、选项取消选中 | — | 判据 = 选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED） | 未实现（映射到位但没有该维度的标记）：G-D1-15→internal/ui/question.go:App.questionCustomField；测试提及 internal/ui/reconcile_scope_main_test.go:包级 |
| R-GQ-124 | G-D1-15 | 样式 | 标签与输入框同时在场时 | 标签 12px 次要色在输入框之上，左缘对齐 | — | 判据 = 字号 UI 正文 12.25px；内边距 左缘 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28） | 已实现（映射 = G-D1-15→internal/ui/question.go:App.questionCustomField；标记 = TextColor） |
| R-GQ-125 | G-D1-15 | 记忆 | 切到其它回合再切回时 | 自定义输入保留 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-D1-15→internal/ui/question.go:App.questionCustomField；测试提及 internal/ui/reconcile_scope_main_test.go:包级 |
| R-GQ-126 | G-D1-16 | 状态 | 未观察（条件）：数组字段 `allowCustom` | 标签「自定义选项」（12px 灰）+ 多行输入框（占位「每行输入一个自定义值」） | — | 判据 = 字号 UI 正文 12.25px；字重 常规 400 / 中粗 500 / 半粗 600 / 粗 700 | 已实现（逐字）：internal/ui/question.go:questionCopyCustomOption「自定义选项」 |
| R-GQ-127 | G-D1-16 | 交互 | 输入多行 | 与勾选项合并提交（自定义项追加为答案） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-D1-16→internal/ui/question.go:questionCardState.answerAt；测试提及 internal/ui/reconcile_scope_main_test.go:包级 |
| R-GQ-128 | G-D1-16 | 内容 | 自定义与勾选重复 | 去重后提交（同一值只出现一次） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 未实现（映射到位但没有该维度的标记）：G-D1-16→internal/ui/question.go:questionCardState.answerAt；测试提及 internal/ui/reconcile_scope_main_test.go:包级 |
| R-GQ-129 | G-D1-16 | 样式 | 自定义选项与自定义答案两块同屏时 | 同款标签与输入框样式 | — | 判据 = 字号 UI 正文 12.25px；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px | 已实现（映射 = G-D1-16→internal/ui/question.go:App.questionCustomField；标记 = TextColor） |
| R-GQ-130 | G-D1-16 | 无障碍 | 读屏进入该元素时 | 标签与输入框成对读出 | — | 播报顺序 = 自定义选项 → 每行输入一个自定义值 | 已实现（映射 = G-D1-16→internal/ui/question.go:App.questionCustomField；标记 = Label） |
| R-GQ-131 | G-D1-16 | 内容 | 自定义行与勾选项都在 | 合并提交，顺序为勾选在前、自定义在后 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM） | 未实现（映射到位但没有该维度的标记）：G-D1-16→internal/ui/question.go:questionCardState.answerAt；测试提及 internal/ui/reconcile_scope_main_test.go:包级 |
| R-GQ-132 | G-D1-16 | 状态 | 只填自定义、未勾选 | 允许提交（自定义项即答案） | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-D1-16→internal/ui/question.go:questionCardState.answerAt；标记 = State） |
| R-GQ-133 | G-D1-16 | 样式 | 自定义选项与自定义答案两块同屏时 | 同款标签与输入框样式 | — | 判据 = 字号 UI 正文 12.25px；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px | 已实现（映射 = G-D1-16→internal/ui/question.go:App.questionCustomField；标记 = TextColor） |
| R-GQ-134 | G-D1-16 | 记忆 | 切到其它回合再切回时 | 自定义行保留 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-D1-16→internal/ui/question.go:questionCardState.answerAt；测试提及 internal/ui/reconcile_scope_main_test.go:包级 |
| R-GQ-135 | G-D1-17 | 内容 | 未观察（条件）：必填未答时点提交 | 错误文字逐字「请先回答所有必填问题」，14px 危险色 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/question.go:questionCopyRequired「请先回答所有必填问题」 |
| R-GQ-136 | G-D1-17 | 内容 | 提交失败 | 显示服务端返回的错误原文（不翻译、不改写） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-D1-17→internal/ui/question.go:App.submitQuestion；标记 = 文案「这条提问没有可作答的内容」等 4 条） |
| R-GQ-137 | G-D1-17 | 状态 | 两分支互斥 | 本地校验错误与服务端错误不同时显示 | — | 文案计数为 1 | 已实现（映射 = G-D1-17→internal/ui/question.go:App.submitQuestion；标记 = State） |
| R-GQ-138 | G-D1-17 | 交互 | 补答必填项 | 错误文字消失，「提交」恢复可用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 已实现（逐字）：internal/ui/command_palette.go:App.commandPaletteActions「提交」 |
| R-GQ-139 | G-D1-17 | 样式 | 错误文字 | 14px 危险色、脚部按钮行之上 | — | 判据 = 字号 正文 14px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-D1-17→internal/ui/question.go:App.questionFoot；标记 = TextColor） |
| R-GQ-140 | G-D1-17 | 无障碍 | 读屏进入该元素时 | `role=alert` 播报一次 | — | 播报次数 = 1 | 已实现（映射 = G-D1-17→internal/ui/question.go:App.questionFoot；标记 = Label） |
| R-GQ-141 | G-D1-17 | 异常 | 连续两次失败 | 同一位置更新文案，不堆叠 | — | 错误段恒 1 | 已实现（映射 = G-D1-17→internal/ui/question.go:App.submitQuestion；标记 = err） |
| R-GQ-142 | G-D1-17 | 内容 | 错误原因为多行 | 折行显示完整 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-D1-17→internal/ui/question.go:App.submitQuestion；标记 = 文案「这条提问没有可作答的内容」等 4 条） |
| R-GQ-143 | G-D1-18 | 状态 | 未观察（条件）：卡片在场 | 次按钮「拒绝」，`outline` 小号，位于脚部最左 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalCopyReject「拒绝」 |
| R-GQ-144 | G-D1-18 | 交互 | 鼠标点击该元素时 | 拒绝该请求（等同「不回答」）；卡片消失；时间线记录拒绝 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 已实现（映射 = G-D1-18→internal/ui/question.go:App.questionFoot；标记 = Clicked） |
| R-GQ-145 | G-D1-18 | 状态 | 提交进行中时 | 禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-D1-18→internal/ui/question.go:App.questionFoot；标记 = State@questionCardState.complete） |
| R-GQ-146 | G-D1-18 | 样式 | 次按钮在屏上时 | 描边、无底色、高 28 | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-D1-18→internal/ui/question.go:App.questionFoot；标记 = TextColor） |
| R-GQ-147 | G-D1-18 | 无障碍 | 读屏进入该元素时 | 读作按钮「拒绝」 | — | 读屏文本 = 「拒绝」 | 已实现（逐字）：internal/ui/approval.go:approvalCopyReject「拒绝」 |
| R-GQ-148 | G-D1-18 | 键盘 | 按 Enter 时 | 等同点击 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 未实现（映射到位但没有该维度的标记）：G-D1-18→internal/ui/question.go:App.questionFoot |
| R-GQ-149 | G-D1-18 | 异常 | 拒绝失败 | 卡片保留并给失败提示 | — | 卡片 计数 = 1；错误文字 计数 = 1 | 已实现（映射 = G-D1-18→internal/ui/question.go:App.questionFoot；标记 = err） |
| R-GQ-150 | G-D1-18 | 内容 | 文案在屏上时 | 恒为「拒绝」，不随题号变 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalCopyReject「拒绝」 |
| R-GQ-151 | G-D1-19 | 状态 | 未观察（条件）：字段数 ≥2 且已走到第 2 题之后 | 次按钮「上一题」出现（首题不出现） | — | 屏上 次按钮 计数 = 0（首题时） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GQ-152 | G-D1-19 | 交互 | 鼠标点击该元素时 | 回到上一题；已答内容保留；进度徽标数字减 1 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-D1-19→internal/ui/question.go:App.questionFoot；标记 = Clicked） |
| R-GQ-153 | G-D1-19 | 状态 | 已在首题 | 按钮隐藏（不置灰） | — | 屏上 次按钮 计数 = 0 | 已实现（映射 = G-D1-19→internal/ui/question.go:App.questionFoot；标记 = State@questionCardState.complete） |
| R-GQ-154 | G-D1-19 | 关联 | 中间题、上一题与下一题同屏时 | 两者可同屏（中间题），首题只有下一题、末题只有提交 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-D1-19→internal/ui/question.go:App.questionFoot；标记 = Select） |
| R-GQ-155 | G-D1-19 | 样式 | 次按钮在屏上时 | 小号（高 28）、无底色或描边 | — | 判据 = 控件尺寸 按钮高 次 28 | 已实现（映射 = G-D1-19→internal/ui/question.go:App.questionFoot；标记 = TextColor） |
| R-GQ-156 | G-D1-19 | 无障碍 | 读屏进入该元素时 | 读作按钮「上一题」 | — | 读屏文本 = 「上一题」 | 已实现（逐字）：internal/ui/question.go:questionCopyPrevious「上一题」 |
| R-GQ-157 | G-D1-19 | 键盘 | 按 Enter 时 | 等同点击 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 未实现（映射到位但没有该维度的标记）：G-D1-19→internal/ui/question.go:App.questionFoot |
| R-GQ-158 | G-D1-19 | 内容 | 文案在屏上时 | 恒为「上一题」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/question.go:questionCopyPrevious「上一题」 |
| R-GQ-159 | G-D1-20 | 状态 | 未观察（条件）：字段数 ≥2 且未到末题 | 主按钮「下一题」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/question.go:questionCopyNext「下一题」 |
| R-GQ-160 | G-D1-20 | 交互 | 鼠标点击该元素时 | 前进一题；当前题答案暂存 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-D1-20→internal/ui/question.go:App.questionFoot；标记 = Clicked） |
| R-GQ-161 | G-D1-20 | 状态 | 当前题必填未答 | 允许前进（不阻断），提交时才校验（按实现一致） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-D1-20→internal/ui/question.go:App.questionFoot；标记 = State@questionCardState.complete） |
| R-GQ-162 | G-D1-20 | 关联 | 切到末题时 | 末题时换成「提交」，两者不同时出现 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/command_palette.go:App.commandPaletteActions「提交」 |
| R-GQ-163 | G-D1-20 | 样式 | 主按钮在屏上时 | 实心强调色、高 28、白字 | — | 判据 = 底色 主色 明 #339751 / 暗 #63cb7b；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-D1-20→internal/ui/question.go:App.questionFoot；标记 = TextColor） |
| R-GQ-164 | G-D1-20 | 无障碍 | 读屏进入该元素时 | 读作按钮「下一题」 | — | 读屏文本 = 「下一题」 | 已实现（逐字）：internal/ui/question.go:questionCopyNext「下一题」 |
| R-GQ-165 | G-D1-20 | 键盘 | 按 Enter 时 | 等同点击 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 未实现（映射到位但没有该维度的标记）：G-D1-20→internal/ui/question.go:App.questionFoot |
| R-GQ-166 | G-D1-20 | 内容 | 文案在屏上时 | 恒为「下一题」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/question.go:questionCopyNext「下一题」 |
| R-GQ-167 | G-D1-21 | 状态 | 未观察（条件）：单字段或末题 | 主按钮「提交」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/command_palette.go:App.commandPaletteActions「提交」 |
| R-GQ-168 | G-D1-21 | 交互 | 鼠标点击该元素时 | 提交答案；卡片消失；时间线记录该次问答 | — | 屏上 卡片 计数 = 0 | 已实现（映射 = G-D1-21→internal/ui/question.go:App.questionFoot；标记 = Clicked） |
| R-GQ-169 | G-D1-21 | 状态 | 必填未满足（单字段） | 按钮禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-D1-21→internal/ui/question.go:App.submitQuestion；标记 = State） |
| R-GQ-170 | G-D1-21 | 状态 | 提交进行中时 | 禁用并显示执行态 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-D1-21→internal/ui/question.go:App.submitQuestion；标记 = State） |
| R-GQ-171 | G-D1-21 | 样式 | 主按钮在屏上时 | 实心强调色、高 28；为脚部最右按钮 | — | 判据 = 底色 主色 明 #339751 / 暗 #63cb7b；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-D1-21→internal/ui/question.go:App.questionFoot；标记 = TextColor） |
| R-GQ-172 | G-D1-21 | 无障碍 | 读屏进入该元素时 | 读作按钮「提交」 | — | 读屏文本 = 「提交」 | 已实现（逐字）：internal/ui/command_palette.go:App.commandPaletteActions「提交」 |
| R-GQ-173 | G-D1-21 | 键盘 | 按 Enter 时 | 等同点击 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 未实现（映射到位但没有该维度的标记）：G-D1-21→internal/ui/question.go:App.submitQuestion |
| R-GQ-174 | G-D1-21 | 异常 | 提交失败 | 卡片保留、错误显示（G-D1-17）、按钮恢复 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-D1-21→internal/ui/question.go:App.submitQuestion；标记 = err） |
| R-GQ-175 | G-D1-22 | 状态 | 未观察（条件）：提交中，或单字段且必填未满足 | 按钮降到 50% 不透明并禁用 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED） | 已实现（映射 = G-D1-22→internal/ui/question.go:App.questionFoot；标记 = State@questionCardState.complete） |
| R-GQ-176 | G-D1-22 | 状态 | 条件解除 | 恢复 100% 不透明与可点 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED） | 已实现（映射 = G-D1-22→internal/ui/question.go:App.questionFoot；标记 = State@questionCardState.complete） |
| R-GQ-177 | G-D1-22 | 关联 | 必填未满足、点过提交后 | 必填未满足时「提交」禁用，同时（若用户点过）显示校验错误 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/ui/command_palette.go:App.commandPaletteActions「提交」 |
| R-GQ-178 | G-D1-22 | 样式 | 禁用 | 0.5 不透明，尺寸与位置不变 | — | 判据 = 色值取 token：主色 #349651/#63cb7b、正文 #0a0a0a/#ececec、次文 #737373/#8a8a8a、边框 #e5e5e5/#2e2e2e、警示 #e17100/#ffb900，偏差 0（CL-ACCENT / CL-TEXT / CL-TEXT-2 / CL-BORDER / CL-WARN）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-D1-22→internal/ui/question.go:App.questionFoot；标记 = TextColor） |
| R-GQ-179 | G-D1-22 | 无障碍 | 读屏进入该元素时 | 禁用态播报「不可用」 | — | 读屏文本 = 「不可用」 | 已实现（逐字）：internal/account/account.go:ErrActionsUnavailable「不可用」 |
| R-GQ-180 | G-D1-22 | 交互 | 禁用时点击 / Enter | 无动作 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-D1-22→internal/ui/question.go:App.questionFoot；标记 = Clicked） |
| R-GQ-181 | G-D1-22 | 异常 | 提交超时 | 恢复可点并显示失败 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-D1-22→internal/ui/question.go:App.questionFoot；标记 = err） |
| R-GQ-182 | G-D1-22 | 内容 | 按钮禁用的两种原因 | 文案不变（仍为「提交」），只用不透明度表达不可用 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED） | 已实现（逐字）：internal/ui/command_palette.go:App.commandPaletteActions「提交」 |

### 4.1 提问域维度覆盖说明

22 项全部 ≥6 维。G-D1-03（回退标题）到 6 维：纯文案回退，无自身交互与过渡（由 G-D1-02 的行承担）。


---

## 5. 确认（21 元素）

四类确认框：编辑运行中消息（E1，4）、删除会话（E2，5）、归档会话（E3，6）、删除工作区会话（E4，6）。**五类零覆盖**：60 份态里没有任何一份打开过这四类确认框（入口都采到了），故 21 项全部按条件写。

### 5.1 编辑运行中消息（G-E1-01–G-E1-04）

条件：会话仍在运行时提交对已发消息的编辑。入口已采（C28 行内编辑器）。

| 需求 ID | 元素 ID | 维度 | 场景 | 期望 | 异常 | 验收 | 状态 |
|---|---|---|---|---|---|---|---|
| R-GC-001 | G-E1-01 | 内容 | 未观察（条件）：会话运行中 → 编辑已发消息 → 提交 | 弹窗标题逐字「会话正在运行」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/confirm.go:confirmCopyEditTitle「会话正在运行」 |
| R-GC-002 | G-E1-01 | 状态 | 会话空闲时编辑 | 不弹此框（直接发送编辑后的消息） | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-E1-01→internal/ui/confirm.go:App.confirmDialog；标记 = State） |
| R-GC-003 | G-E1-01 | 关联 | 编辑运行中确认框打开时 | 标题 + 描述 + 两按钮，四件齐全，无第三按钮 | — | 框内按钮恰 2 个 | 已实现（映射 = G-E1-01→internal/ui/confirm.go:App.confirmDialog；标记 = Select） |
| R-GC-004 | G-E1-01 | 样式 | 弹窗 | 16px 级半粗主前景；圆角 + 阴影 + 遮罩 | — | 判据 = 字号 对话框标题 15.75px；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px；其余 = 色值取 token：主色 #349651/#63cb7b、正文 #0a0a0a/#ececec、次文 #737373/#8a8a8a、边框 #e5e5e5/#2e2e2e、警示 #e17100/#ffb900，偏差 0（CL-ACCENT / CL-TEXT / CL-TEXT-2 / CL-BORDER / CL-WARN） | 已实现（映射 = G-E1-01→internal/ui/confirm.go:App.confirmDialog；标记 = TextColor） |
| R-GC-005 | G-E1-01 | 无障碍 | 读屏进入该元素时 | 作为对话框名读出，焦点默认落在「取消」 | — | 读屏文本 = 「取消」 | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GC-006 | G-E1-01 | 异常 | 弹窗期间该轮已自行结束 | 确认后不再弹错，直接按「无运行」路径发送 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/rightpanel_tasks.go:App.tasksPanel「无运行」 |
| R-GC-007 | G-E1-01 | 记忆 | 关闭后再编辑同一消息 | 弹窗重建，文案不变 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-E1-01→internal/ui/confirm.go:App.confirmTitle；测试提及 internal/ui/reconcile_scope_test.go:包级 |
| R-GC-008 | G-E1-01 | 内容 | 标题超长（本地化） | 单行 truncate | — | 标题 1 行 | 已实现（映射 = G-E1-01→internal/ui/confirm.go:App.confirmTitle；标记 = 文案「会话正在运行」等 5 条） |
| R-GC-009 | G-E1-02 | 内容 | 未观察（条件）：编辑运行中确认框打开时 | 描述逐字「当前会话仍在运行中。是否停止本次运行并发送编辑后的消息？」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/confirm.go:confirmCopyEditDesc「当前会话仍在运行中。是否停止本次运行并发送编辑后的消息？」 |
| R-GC-010 | G-E1-02 | 内容 | 描述过长（窄窗） | 折行，按钮行仍在最下方可见 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-E1-02→internal/ui/confirm.go:App.confirmDescription；标记 = 文案「当前会话仍在运行中。是否停止本次运行并发送编辑后的消息？」等 6 条） |
| R-GC-011 | G-E1-02 | 样式 | 描述在屏上时 | 14px 级次要色、行高 1.5 | — | 判据 = 字号 正文 14px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-E1-02→internal/ui/confirm.go:App.confirmDialog；标记 = TextColor） |
| R-GC-012 | G-E1-02 | 无障碍 | 读屏进入该元素时 | 作为描述读出，先于按钮 | — | 读屏文本 = 「当前会话仍在运行中。是否停止本次运行并发送编辑后」 | 已实现（映射 = G-E1-02→internal/ui/confirm.go:App.confirmDialog；标记 = Role） |
| R-GC-013 | G-E1-02 | 异常 | 描述未本地化 | 按语言包显示中文，不留英文原文 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-E1-02→internal/ui/confirm.go:App.confirmDialog；标记 = err） |
| R-GC-014 | G-E1-02 | 内容 | 描述在屏上时 | 两句话：先说状态、再说动作（停止并发送） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-E1-02→internal/ui/confirm.go:App.confirmDescription；标记 = 文案「当前会话仍在运行中。是否停止本次运行并发送编辑后的消息？」等 6 条） |
| R-GC-015 | G-E1-02 | 关联 | 确认框打开、描述与主按钮同屏时 | 描述里的动作与主按钮文案一致（都含「停止」「发送」） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/app/app.go:App.StopIntents「停止」 |
| R-GC-016 | G-E1-02 | 异常 | 描述为空 | 不允许：描述是这段确认的唯一解释 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-E1-02→internal/ui/confirm.go:App.confirmDialog；标记 = err） |
| R-GC-017 | G-E1-03 | 状态 | 未观察（条件）：弹窗在场 | 次按钮「取消」，位于左侧 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GC-018 | G-E1-03 | 交互 | 鼠标点击该元素时 | 关闭弹窗；编辑内容保留在行内编辑器里（不丢用户输入） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-E1-03→internal/ui/confirm.go:App.confirmDialog；标记 = Key） |
| R-GC-019 | G-E1-03 | 键盘 | Esc / Enter（焦点在此） | 等同点击 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-E1-03→internal/ui/confirm.go:App.confirmDialog；标记 = Shortcut） |
| R-GC-020 | G-E1-03 | 样式 | 次按钮在屏上时 | 描边、无底色、高 28 | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-E1-03→internal/ui/confirm.go:App.confirmDialog；标记 = TextColor） |
| R-GC-021 | G-E1-03 | 无障碍 | 读屏进入该元素时 | 读作按钮「取消」，为默认焦点 | — | 读屏文本 = 「取消」 | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GC-022 | G-E1-03 | 内容 | 文案在屏上时 | 恒为「取消」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GC-023 | G-E1-03 | 交互 | 点击时编辑内容为空 | 仍可取消（不因空草稿而禁用） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-E1-03→internal/ui/confirm.go:App.confirmDialog；标记 = Key） |
| R-GC-024 | G-E1-03 | 样式 | 取消与确认按钮同屏时 | 同为高 28，次按钮为描边 | — | 判据 = 控件尺寸 按钮高 次 28；描边 常规 1px（卡片外框 0.5px / 焦点环 3px） | 已实现（映射 = G-E1-03→internal/ui/confirm.go:App.confirmDialog；标记 = TextColor） |
| R-GC-025 | G-E1-04 | 状态 | 未观察（条件）：弹窗在场 | 主按钮「停止并发送」，危险色或主色、位于右侧 | — | 判据 = 底色 主色 明 #339751 / 暗 #63cb7b；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（逐字）：internal/ui/confirm.go:confirmCopyStopSend「停止并发送」 |
| R-GC-026 | G-E1-04 | 交互 | 鼠标点击该元素时 | 先停止当前回合，再以编辑后内容发送；弹窗关闭 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-E1-04→internal/ui/confirm.go:App.confirmDialog；标记 = Key） |
| R-GC-027 | G-E1-04 | 状态 | 停止中 / 发送中 | 按钮禁用并显示执行态 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-E1-04→internal/ui/confirm.go:App.confirmDialog；标记 = State） |
| R-GC-028 | G-E1-04 | 异常 | 停止失败 | 不发送；弹窗保留并给失败原因 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GC-029 | G-E1-04 | 样式 | 主按钮在屏上时 | 实心（危险色或主色）、白字、高 28 | — | 判据 = 底色 主色 明 #339751 / 暗 #63cb7b；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-E1-04→internal/ui/confirm.go:App.confirmDialog；标记 = TextColor） |
| R-GC-030 | G-E1-04 | 无障碍 | 读屏进入该元素时 | 读作按钮「停止并发送」，并说明会中断当前回合 | — | 读屏文本 = 「停止并发送」 | 已实现（逐字）：internal/ui/confirm.go:confirmCopyStopSend「停止并发送」 |
| R-GC-031 | G-E1-04 | 键盘 | 按 Enter 时 | 等同点击 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-E1-04→internal/ui/confirm.go:App.confirmDialog；标记 = Shortcut） |
| R-GC-032 | G-E1-04 | 内容 | 文案在屏上时 | 恒为「停止并发送」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/confirm.go:confirmCopyStopSend「停止并发送」 |

### 5.2 删除会话（G-E2-01–G-E2-05）

条件：单条删除（任意时刻）或批量删除（多选后）；标题与描述随条数切换。入口已采（C13 会话行右键「删除」157,416）。

| 需求 ID | 元素 ID | 维度 | 场景 | 期望 | 异常 | 验收 | 状态 |
|---|---|---|---|---|---|---|---|
| R-GC-033 | G-E2-01 | 内容 | 未观察（条件）：单条删除 | 标题逐字「删除会话」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/confirm.go:confirmCopyDeleteOne「删除会话」 |
| R-GC-034 | G-E2-01 | 内容 | 多条删除（多选 >1） | 标题逐字「删除 {count} 个会话」，count 为选中数 | — | 屏上形如「删除 3 个会话」 | 已实现（映射 = G-E2-01→internal/ui/confirm.go:App.confirmTitle；标记 = 文案「会话正在运行」等 5 条） |
| R-GC-035 | G-E2-01 | 状态 | 两条分支互斥 | 同一时刻只有一种标题 | — | 标题计数为 1 | 已实现（映射 = G-E2-01→internal/ui/confirm.go:App.confirmDialog；标记 = State） |
| R-GC-036 | G-E2-01 | 样式 | 标题在屏上时 | 16px 级半粗；弹窗圆角 + 阴影 + 遮罩 | — | 判据 = 字号 对话框标题 15.75px；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px | 已实现（映射 = G-E2-01→internal/ui/confirm.go:App.confirmDialog；标记 = TextColor） |
| R-GC-037 | G-E2-01 | 无障碍 | 读屏进入该元素时 | 作为对话框名读出 | — | 读屏文本 = 「删除会话」 | 已实现（映射 = G-E2-01→internal/ui/confirm.go:App.confirmDialog；标记 = Role） |
| R-GC-038 | G-E2-01 | 异常 | 选中集合在弹窗期间变化 | 标题按打开时的条数定格，不实时变；确认时以实际集合为准 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-E2-01→internal/ui/confirm.go:App.confirmDialog；标记 = err） |
| R-GC-039 | G-E2-01 | 窄窗 | 窗口收窄到最窄宽度时 | 标题单行不折行（过长截断），按钮行完整 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 未实现（映射到位但没有该维度的标记）：G-E2-01→internal/ui/confirm.go:App.confirmTitle；测试提及 internal/ui/reconcile_scope_test.go:包级 |
| R-GC-040 | G-E2-01 | 内容 | 批量删除 1 条（只选一条） | 走单条文案「删除会话」，不走「删除 1 个会话」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/confirm.go:confirmCopyDeleteOne「删除会话」 |
| R-GC-041 | G-E2-02 | 内容 | 单条删除 | 描述逐字「确定要永久删除“{title}”吗？删除后会话将立即从侧边栏移除，且无法恢复。」；会话名用中文直角引号「“”」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 已实现（映射 = G-E2-02→internal/ui/confirm.go:App.confirmDescription；标记 = 文案「当前会话仍在运行中。是否停止本次运行并发送编辑后的消息？」等 6 条） |
| R-GC-042 | G-E2-02 | 内容 | 多条删除 | 描述逐字「确定要永久删除以下会话吗？{names}」；`{names}` 最多三名，超出以「等 N 个」收尾 | — | 屏上最多 3 个名字 + 「等 N 个」 | 已实现（映射 = G-E2-02→internal/ui/confirm.go:App.confirmDescription；标记 = 文案「当前会话仍在运行中。是否停止本次运行并发送编辑后的消息？」等 6 条） |
| R-GC-043 | G-E2-02 | 内容 | 名字组合很长 | 折行显示，不横向溢出；「等 N 个」保持在末位 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-E2-02→internal/ui/confirm.go:App.confirmDescription；标记 = 文案「当前会话仍在运行中。是否停止本次运行并发送编辑后的消息？」等 6 条） |
| R-GC-044 | G-E2-02 | 关联 | 多条会话被选中、确认框打开时 | 单条标题配单条描述、多条标题配多条描述，不交叉 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-E2-02→internal/ui/confirm.go:App.confirmDescription；标记 = Open@confirmNames） |
| R-GC-045 | G-E2-02 | 样式 | 描述在屏上时 | 14px 级次要色、行高 1.5 | — | 判据 = 字号 正文 14px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-E2-02→internal/ui/confirm.go:App.confirmDialog；标记 = TextColor） |
| R-GC-046 | G-E2-02 | 无障碍 | 读屏进入该元素时 | 作为描述读出，会话名逐个可听 | — | 读屏文本 = 「确定要永久删除“{title}”吗？删除后会话将」 | 已实现（映射 = G-E2-02→internal/ui/confirm.go:App.confirmDialog；标记 = Role） |
| R-GC-047 | G-E2-02 | 异常 | 会话标题含引号 | 原文照显，不转义成 `&quot;` | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-E2-02→internal/ui/confirm.go:App.confirmDialog；标记 = err） |
| R-GC-048 | G-E2-02 | 内容 | 名字超过三名 | 显示前三名 + 「等 {N} 个」，N 为剩余数 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-E2-02→internal/ui/confirm.go:App.confirmDescription；标记 = 文案「当前会话仍在运行中。是否停止本次运行并发送编辑后的消息？」等 6 条） |
| R-GC-049 | G-E2-02 | 内容 | 单条会话名超长 | 名字在句中 truncate，句子仍完整 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-E2-02→internal/ui/confirm.go:App.confirmDescription；标记 = 文案「当前会话仍在运行中。是否停止本次运行并发送编辑后的消息？」等 6 条） |
| R-GC-050 | G-E2-03 | 内容 | 未观察（条件）：删除失败且为单条 | 错误文字显示失败原因（14px 危险色） | — | 判据 = 字号 正文 14px；字重 常规 400 / 中粗 500 / 半粗 600 / 粗 700 | 已实现（映射 = G-E2-03→internal/ui/confirm.go:App.confirmDialog；标记 = 文案「正在运行的会话会被中断，其队列中的后续消息也会被丢弃。」等 2 条） |
| R-GC-051 | G-E2-03 | 状态 | 多条删除失败 | 该行不出现（多条只有汇总提示，按实现一致） | — | 屏上 错误文字 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GC-052 | G-E2-03 | 交互 | 「重试」在屏上时 | 原因先清除再提交，不叠加 | — | 错误段恒 1 | 已实现（映射 = G-E2-03→internal/ui/confirm.go:App.confirmDialog；标记 = Key） |
| R-GC-053 | G-E2-03 | 样式 | 错误段在屏上时 | 14px 危险色、位于按钮行之上 | — | 判据 = 字号 正文 14px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-E2-03→internal/ui/confirm.go:App.confirmDialog；标记 = TextColor） |
| R-GC-054 | G-E2-03 | 无障碍 | 读屏进入该元素时 | `role=alert` 播报一次 | — | 播报次数 = 1 | 已实现（映射 = G-E2-03→internal/ui/confirm.go:App.confirmDialog；标记 = Role） |
| R-GC-055 | G-E2-03 | 异常 | 原因超长 | 折行显示完整，不截断 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-E2-03→internal/ui/confirm.go:App.confirmDialog；标记 = err） |
| R-GC-056 | G-E2-03 | 内容 | 失败原因为空 | 不渲染错误行（不留空行） | — | 屏上 错误行 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GC-057 | G-E2-03 | 交互 | 鼠标点击错误文字时 | 无动作 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-E2-03→internal/ui/confirm.go:App.confirmDialog；标记 = Key） |
| R-GC-058 | G-E2-04 | 状态 | 未观察（条件）：弹窗在场 | 次按钮「取消」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GC-059 | G-E2-04 | 交互 | 鼠标点击该元素时 | 关闭，不删除任何会话；多选状态保留 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-E2-04→internal/ui/confirm.go:App.confirmDialog；标记 = Key） |
| R-GC-060 | G-E2-04 | 键盘 | 按 Esc 时 | 等同点击 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-E2-04→internal/ui/confirm.go:App.confirmDialog；标记 = Shortcut） |
| R-GC-061 | G-E2-04 | 样式 | 次按钮在屏上时 | `outline`、高 28 | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-E2-04→internal/ui/confirm.go:App.confirmDialog；标记 = TextColor） |
| R-GC-062 | G-E2-04 | 无障碍 | 读屏进入该元素时 | 读作按钮「取消」 | — | 读屏文本 = 「取消」 | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GC-063 | G-E2-04 | 内容 | 文案在屏上时 | 恒为「取消」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GC-064 | G-E2-04 | 异常 | 删除进行中按 Esc | 不关闭（执行中 Esc 无效） | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-E2-04→internal/ui/confirm.go:App.confirmDialog；标记 = err） |
| R-GC-065 | G-E2-04 | 样式 | 取消与删除按钮同屏时 | 同高 28，描边次按钮 | — | 判据 = 控件尺寸 按钮高 次 28；描边 常规 1px（卡片外框 0.5px / 焦点环 3px） | 已实现（映射 = G-E2-04→internal/ui/confirm.go:App.confirmDialog；标记 = TextColor） |
| R-GC-066 | G-E2-05 | 内容 | 未观察（条件）：单条 | 危险按钮「删除」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/confirm.go:confirmCopyDeleteOne「删除」 |
| R-GC-067 | G-E2-05 | 内容 | 多条 | 文案「删除 ({count})」，含括号与数量 | — | 屏上形如「删除 (3)」 | 已实现（映射 = G-E2-05→internal/ui/confirm.go:App.confirmPrimaryText；标记 = 文案「停止中...」等 6 条） |
| R-GC-068 | G-E2-05 | 内容 | 提交进行中时 | 文案「删除中...」，按钮禁用 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/ui/confirm.go:confirmCopyDeleteBusy「删除中...」 |
| R-GC-069 | G-E2-05 | 交互 | 鼠标点击该元素时 | 删除目标会话；成功后关闭弹窗、列表少行 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-E2-05→internal/ui/confirm.go:App.confirmDialog；标记 = Key） |
| R-GC-070 | G-E2-05 | 异常 | 部分删除失败（多条） | 成功的已删、失败的保留并给原因；弹窗不静默关闭 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GC-071 | G-E2-05 | 样式 | 危险按钮在屏上时 | 危险色底 + 白字、高 28 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；色值取 token：主色 #349651/#63cb7b、正文 #0a0a0a/#ececec、次文 #737373/#8a8a8a、边框 #e5e5e5/#2e2e2e、警示 #e17100/#ffb900，偏差 0（CL-ACCENT / CL-TEXT / CL-TEXT-2 / CL-BORDER / CL-WARN） | 已实现（映射 = G-E2-05→internal/ui/confirm.go:App.confirmDialog；标记 = TextColor） |
| R-GC-072 | G-E2-05 | 无障碍 | 读屏进入该元素时 | 读作按钮「删除」（多条时含数量） | — | 读屏文本 = 「删除」 | 已实现（逐字）：internal/ui/confirm.go:confirmCopyDeleteOne「删除」 |
| R-GC-073 | G-E2-05 | 键盘 | 按 Enter 时 | 等同点击 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-E2-05→internal/ui/confirm.go:App.confirmDialog；标记 = Shortcut） |

### 5.3 归档会话（G-E3-01–G-E3-06）

条件：批量归档，或归档单条仍在运行的会话（空闲单条不弹此框，直接归档）。

| 需求 ID | 元素 ID | 维度 | 场景 | 期望 | 异常 | 验收 | 状态 |
|---|---|---|---|---|---|---|---|
| R-GC-074 | G-E3-01 | 内容 | 未观察（条件）：批量归档 | 标题逐字「归档 {count} 个会话」 | — | 屏上形如「归档 3 个会话」 | 已实现（映射 = G-E3-01→internal/ui/confirm.go:App.confirmTitle；标记 = 文案「会话正在运行」等 5 条） |
| R-GC-075 | G-E3-01 | 状态 | 归档空闲单条 | 不弹此框（直接归档，无确认） | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-E3-01→internal/ui/confirm.go:App.confirmDialog；标记 = State） |
| R-GC-076 | G-E3-01 | 关联 | 归档确认框打开时 | 标题 + 描述 + 可选警告 + 两按钮 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-E3-01→internal/ui/confirm.go:App.confirmDialog；标记 = Select） |
| R-GC-077 | G-E3-01 | 样式 | 标题在屏上时 | 16px 级半粗；圆角 + 阴影 + 遮罩 | — | 判据 = 字号 对话框标题 15.75px；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px | 已实现（映射 = G-E3-01→internal/ui/confirm.go:App.confirmDialog；标记 = TextColor） |
| R-GC-078 | G-E3-01 | 无障碍 | 读屏进入该元素时 | 作为对话框名读出 | — | 读屏文本 = 「归档 {count} 个会话」 | 已实现（映射 = G-E3-01→internal/ui/confirm.go:App.confirmDialog；标记 = Role） |
| R-GC-079 | G-E3-01 | 窄窗 | 窗口收窄到最窄宽度时 | 标题不折行，按钮行完整 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 未实现（映射到位但没有该维度的标记）：G-E3-01→internal/ui/confirm.go:App.confirmTitle；测试提及 internal/ui/reconcile_scope_test.go:包级 |
| R-GC-080 | G-E3-01 | 内容 | 批量 1 条 | 标题「归档 1 个会话」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-E3-01→internal/ui/confirm.go:App.confirmTitle；标记 = 文案「会话正在运行」等 5 条） |
| R-GC-081 | G-E3-01 | 内容 | 标题超长 | 单行不折行 | — | 标题 1 行 | 已实现（映射 = G-E3-01→internal/ui/confirm.go:App.confirmTitle；标记 = 文案「会话正在运行」等 5 条） |
| R-GC-082 | G-E3-02 | 内容 | 未观察（条件）：弹窗在场 | 描述逐字「确定要归档以下会话吗？{names}」；`{names}` 最多三名 + 「等 N 个」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-E3-02→internal/ui/confirm.go:App.confirmDescription；标记 = 文案「当前会话仍在运行中。是否停止本次运行并发送编辑后的消息？」等 6 条） |
| R-GC-083 | G-E3-02 | 内容 | 名字很长 | 折行，末尾仍为「等 N 个」 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-E3-02→internal/ui/confirm.go:App.confirmDescription；标记 = 文案「当前会话仍在运行中。是否停止本次运行并发送编辑后的消息？」等 6 条） |
| R-GC-084 | G-E3-02 | 样式 | 描述在屏上时 | 14px 级次要色 | — | 判据 = 字号 正文 14px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-E3-02→internal/ui/confirm.go:App.confirmDialog；标记 = TextColor） |
| R-GC-085 | G-E3-02 | 无障碍 | 读屏进入该元素时 | 读出会话名 | — | 读屏文本 = 「确定要归档以下会话吗？{names}」 | 已实现（映射 = G-E3-02→internal/ui/confirm.go:App.confirmDialog；标记 = Role） |
| R-GC-086 | G-E3-02 | 异常 | 会话名含引号 | 原文照显，不转义 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-E3-02→internal/ui/confirm.go:App.confirmDialog；标记 = err） |
| R-GC-087 | G-E3-02 | 内容 | 名字超过三名 | 前三名 + 「等 {N} 个」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-E3-02→internal/ui/confirm.go:App.confirmDescription；标记 = 文案「当前会话仍在运行中。是否停止本次运行并发送编辑后的消息？」等 6 条） |
| R-GC-088 | G-E3-02 | 内容 | 单条运行中会话 | 描述只含该会话名 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 已实现（映射 = G-E3-02→internal/ui/confirm.go:App.confirmDescription；标记 = 文案「当前会话仍在运行中。是否停止本次运行并发送编辑后的消息？」等 6 条） |
| R-GC-089 | G-E3-02 | 异常 | 名字为空 | 显示「未命名会话」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/confirm.go:confirmCopyUnnamed「未命名会话」 |
| R-GC-090 | G-E3-02 | 样式 | 描述位于标题之下时 | 14px 次要色，位于标题之下 | — | 判据 = 字号 正文 14px | 已实现（映射 = G-E3-02→internal/ui/confirm.go:App.confirmDialog；标记 = TextColor） |
| R-GC-091 | G-E3-02 | 窄窗 | 窗口收窄到最窄宽度时 | 名字仍完整显示，句子折行，按钮行完整 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 未实现（映射到位但没有该维度的标记）：G-E3-02→internal/ui/confirm.go:App.confirmDescription |
| R-GC-092 | G-E3-03 | 内容 | 未观察（条件）：目标中有运行中的会话 | 警告文字逐字「正在运行的会话会被中断，其队列中的后续消息也会被丢弃。」，14px 警示色 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/confirm.go:confirmCopyArchiveRun「正在运行的会话会被中断，其队列中的后续消息也会被丢弃。」 |
| R-GC-093 | G-E3-03 | 状态 | 目标全部空闲 | 警告不出现（不留空行） | — | 屏上 警告文字 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GC-094 | G-E3-03 | 关联 | 警告段出现在描述与按钮之间时 | 警告在描述之后、按钮之前，独立成段 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM） | 已实现（映射 = G-E3-03→internal/ui/confirm.go:App.confirmDialog；标记 = Select） |
| R-GC-095 | G-E3-03 | 样式 | 警告 | 14px 警示色（琥珀），无底色 | — | 判据 = 色值 警告 明 #e17100 / 暗 #ffb900；字号 正文 14px | 已实现（映射 = G-E3-03→internal/ui/confirm.go:App.confirmDialog；标记 = TextColor） |
| R-GC-096 | G-E3-03 | 无障碍 | 读屏进入该元素时 | 读出整句，作为风险提示 | — | 读屏文本 = 「正在运行的会话会被中断，其队列中的后续消息也会被」 | 已实现（映射 = G-E3-03→internal/ui/confirm.go:App.confirmDialog；标记 = Role） |
| R-GC-097 | G-E3-03 | 异常 | 运行中会话在弹窗期间结束 | 警告保留（以打开时状态为准），不中途消失 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 已实现（映射 = G-E3-03→internal/ui/confirm.go:App.confirmDialog；标记 = err） |
| R-GC-098 | G-E3-03 | 内容 | 警告句 | 两句：运行中会被中断 + 队列消息会被丢弃 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-E3-03→internal/ui/confirm.go:App.confirmDialog；标记 = 文案「正在运行的会话会被中断，其队列中的后续消息也会被丢弃。」等 2 条） |
| R-GC-099 | G-E3-03 | 状态 | 批量里只有部分在运行 | 警告仍显示一次（不分条列） | — | 警告恒 1 行 | 已实现（映射 = G-E3-03→internal/ui/confirm.go:App.confirmDialog；标记 = State） |
| R-GC-100 | G-E3-04 | 内容 | 未观察（条件）：归档失败 | 错误文字显示失败原因 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-E3-04→internal/ui/confirm.go:App.confirmDialog；标记 = 文案「正在运行的会话会被中断，其队列中的后续消息也会被丢弃。」等 2 条） |
| R-GC-101 | G-E3-04 | 交互 | 「重试」在屏上时 | 先清错误再提交 | — | 错误段恒 1 | 已实现（映射 = G-E3-04→internal/ui/confirm.go:App.confirmDialog；标记 = Key） |
| R-GC-102 | G-E3-04 | 样式 | 错误段在屏上时 | 14px 危险色、按钮行之上 | — | 判据 = 字号 正文 14px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-E3-04→internal/ui/confirm.go:App.confirmDialog；标记 = TextColor） |
| R-GC-103 | G-E3-04 | 无障碍 | 读屏进入该元素时 | `role=alert` 播报一次 | — | 播报次数 = 1 | 已实现（映射 = G-E3-04→internal/ui/confirm.go:App.confirmDialog；标记 = Role） |
| R-GC-104 | G-E3-04 | 异常 | 原因超长 | 折行完整显示 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-E3-04→internal/ui/confirm.go:App.confirmDialog；标记 = err） |
| R-GC-105 | G-E3-04 | 内容 | 失败原因为空 | 不渲染错误行 | — | 屏上 错误行 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GC-106 | G-E3-04 | 交互 | 鼠标点击错误文字时 | 无动作 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-E3-04→internal/ui/confirm.go:App.confirmDialog；标记 = Key） |
| R-GC-107 | G-E3-04 | 样式 | 错误段出现时 | 按钮行之上 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-E3-04→internal/ui/confirm.go:App.confirmDialog；标记 = TextColor） |
| R-GC-108 | G-E3-05 | 状态 | 未观察（条件）：弹窗在场 | 次按钮「取消」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GC-109 | G-E3-05 | 交互 | 鼠标点击该元素时 | 关闭；不归档；运行中的会话继续运行 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-E3-05→internal/ui/confirm.go:App.confirmDialog；标记 = Key） |
| R-GC-110 | G-E3-05 | 键盘 | 按 Esc 时 | 等同点击 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-E3-05→internal/ui/confirm.go:App.confirmDialog；标记 = Shortcut） |
| R-GC-111 | G-E3-05 | 样式 | 次按钮在屏上时 | 描边、高 28 | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-E3-05→internal/ui/confirm.go:App.confirmDialog；标记 = TextColor） |
| R-GC-112 | G-E3-05 | 无障碍 | 读屏进入该元素时 | 读作按钮「取消」 | — | 读屏文本 = 「取消」 | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GC-113 | G-E3-05 | 内容 | 文案在屏上时 | 恒为「取消」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GC-114 | G-E3-05 | 样式 | 取消与确认按钮同屏时 | 同高 28，描边次按钮 | — | 判据 = 控件尺寸 按钮高 次 28；描边 常规 1px（卡片外框 0.5px / 焦点环 3px） | 已实现（映射 = G-E3-05→internal/ui/confirm.go:App.confirmDialog；标记 = TextColor） |
| R-GC-115 | G-E3-05 | 无障碍 | 读屏进入该元素时 | 读作按钮「取消」，默认焦点 | — | 读屏文本 = 「取消」 | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GC-116 | G-E3-06 | 内容 | 未观察（条件）：弹窗在场 | 主按钮「归档 ({count})」，含括号与数量 | — | 屏上形如「归档 (3)」 | 已实现（映射 = G-E3-06→internal/ui/confirm.go:App.confirmPrimaryText；标记 = 文案「停止中...」等 6 条） |
| R-GC-117 | G-E3-06 | 内容 | 提交进行中时 | 文案「归档中...」，按钮禁用 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/ui/confirm.go:confirmCopyArchiveBusy「归档中...」 |
| R-GC-118 | G-E3-06 | 交互 | 鼠标点击该元素时 | 归档目标会话；运行中的先中断；成功后关闭弹窗、列表移出这些会话 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM） | 已实现（映射 = G-E3-06→internal/ui/confirm.go:App.confirmDialog；标记 = Key） |
| R-GC-119 | G-E3-06 | 异常 | 部分失败 | 成功的已归档、失败的保留并给原因 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-E3-06→internal/ui/confirm.go:App.confirmDialog；标记 = err） |
| R-GC-120 | G-E3-06 | 样式 | 主按钮在屏上时 | 实心主色（非危险色：归档可恢复）、白字、高 28 | — | 判据 = 底色 主色 明 #339751 / 暗 #63cb7b；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-E3-06→internal/ui/confirm.go:App.confirmDialog；标记 = TextColor） |
| R-GC-121 | G-E3-06 | 无障碍 | 读屏进入该元素时 | 读作按钮「归档」，含数量，并说明会中断运行中的会话 | — | 读屏文本 = 「归档」 | 已实现（逐字）：internal/ui/command_palette.go:App.commandPaletteActions「归档」 |
| R-GC-122 | G-E3-06 | 键盘 | 按 Enter 时 | 等同点击 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-E3-06→internal/ui/confirm.go:App.confirmDialog；标记 = Shortcut） |
| R-GC-123 | G-E3-06 | 内容 | 数量为 1 | 「归档 (1)」 | — | 主按钮 文案逐字 =「归档 (1)」 | 已实现（映射 = G-E3-06→internal/ui/confirm.go:App.confirmPrimaryText；标记 = 文案「停止中...」等 6 条） |
| R-GC-124 | G-E3-06 | 样式 | 归档与删除按钮同屏时 | 归档用主色（可恢复），不用危险色 | — | 判据 = 底色 主色 明 #339751 / 暗 #63cb7b | 已实现（映射 = G-E3-06→internal/ui/confirm.go:App.confirmDialog；标记 = TextColor） |

### 5.4 删除工作区会话（G-E4-01–G-E4-06）

条件：在侧栏工作区分组上执行删除；展示该工作区下将被删除的会话条数。入口已采（C16 工作区行 `⋯` →「移除」169,478）。

| 需求 ID | 元素 ID | 维度 | 场景 | 期望 | 异常 | 验收 | 状态 |
|---|---|---|---|---|---|---|---|
| R-GC-125 | G-E4-01 | 内容 | 未观察（条件）：工作区分组菜单 → 移除 | 标题逐字「删除工作区会话」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/confirm.go:confirmCopyWorkspace「删除工作区会话」 |
| R-GC-126 | G-E4-01 | 状态 | 该工作区无会话 | 仍弹此框（描述里计数为 0），或直接不显示入口（按实现一致） | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GC-127 | G-E4-01 | 样式 | 标题在屏上时 | 16px 级半粗；圆角 + 阴影 + 遮罩 | — | 判据 = 字号 对话框标题 15.75px；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px | 已实现（映射 = G-E4-01→internal/ui/confirm.go:App.confirmDialog；标记 = TextColor） |
| R-GC-128 | G-E4-01 | 无障碍 | 读屏进入该元素时 | 作为对话框名读出 | — | 读屏文本 = 「删除工作区会话」 | 已实现（映射 = G-E4-01→internal/ui/confirm.go:App.confirmDialog；标记 = Role） |
| R-GC-129 | G-E4-01 | 异常 | 弹窗期间会话数变化 | 描述里的计数按打开时定格；确认时以实际为准 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-E4-01→internal/ui/confirm.go:App.confirmDialog；标记 = err） |
| R-GC-130 | G-E4-01 | 内容 | 标题超长 | 单行不折行 | — | 标题 1 行 | 已实现（映射 = G-E4-01→internal/ui/confirm.go:App.confirmTitle；标记 = 文案「会话正在运行」等 5 条） |
| R-GC-131 | G-E4-01 | 状态 | 工作区已空 | 仍可打开（描述计数为 0） | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-E4-01→internal/ui/confirm.go:App.confirmDialog；标记 = State） |
| R-GC-132 | G-E4-01 | 异常 | 工作区在弹窗期间消失 | 确认时给失败原因 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-E4-01→internal/ui/confirm.go:App.confirmDialog；标记 = err） |
| R-GC-133 | G-E4-02 | 内容 | 未观察（条件）：弹窗在场 | 描述逐字「将删除“{name}”下的 {count} 条会话。」；`{name}` 为工作区显示名 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-E4-02→internal/ui/confirm.go:App.confirmDescription；标记 = 文案「当前会话仍在运行中。是否停止本次运行并发送编辑后的消息？」等 6 条） |
| R-GC-134 | G-E4-02 | 内容 | 名称为空 | 描述里不留空引号对，退为「该工作区」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/confirm.go:confirmCopyWorkspaceF「该工作区」 |
| R-GC-135 | G-E4-02 | 关联 | 描述与路径行上下排列时 | 描述在上、路径行在下，两句不合并 | — | 屏上 描述 计数 = 1 | 已实现（映射 = G-E4-02→internal/ui/confirm.go:App.confirmDescription；标记 = Open@confirmNames） |
| R-GC-136 | G-E4-02 | 样式 | 描述在屏上时 | 14px 级次要色 | — | 判据 = 字号 正文 14px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-E4-02→internal/ui/confirm.go:App.confirmDialog；标记 = TextColor） |
| R-GC-137 | G-E4-02 | 无障碍 | 读屏进入该元素时 | 读出名称与条数 | — | 读屏文本 = 「将删除“{name}”下的 {count} 条会」 | 已实现（映射 = G-E4-02→internal/ui/confirm.go:App.confirmDialog；标记 = Role） |
| R-GC-138 | G-E4-02 | 内容 | 名称为超长显示名 | 名称 truncate，句子完整 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-E4-02→internal/ui/confirm.go:App.confirmDescription；标记 = 文案「当前会话仍在运行中。是否停止本次运行并发送编辑后的消息？」等 6 条） |
| R-GC-139 | G-E4-02 | 内容 | 计数为 0 | 显示「将删除“{name}”下的 0 条会话。」 | — | 屏上 描述 计数 = 1（计数为 0 时） | 已实现（映射 = G-E4-02→internal/ui/confirm.go:App.confirmDescription；标记 = 文案「当前会话仍在运行中。是否停止本次运行并发送编辑后的消息？」等 6 条） |
| R-GC-140 | G-E4-02 | 关联 | 标题与描述同屏时 | 标题不含名称、描述含名称，不重复 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GC-141 | G-E4-02 | 样式 | 描述位于标题之下时 | 14px 次要色，位于标题之下 | — | 判据 = 字号 正文 14px | 已实现（映射 = G-E4-02→internal/ui/confirm.go:App.confirmDialog；标记 = TextColor） |
| R-GC-142 | G-E4-02 | 状态 | 工作区无会话 | 描述计数为 0 时仍显示该句 | — | 屏上 描述 计数 = 1（计数为 0 时） | 已实现（映射 = G-E4-02→internal/ui/confirm.go:App.confirmDialog；标记 = State） |
| R-GC-143 | G-E4-03 | 内容 | 未观察（条件）：弹窗在场 | 工作区路径（14px 灰），等宽或常规按实现，可折行 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-E4-03→internal/ui/confirm.go:App.confirmPathRow；标记 = 文案「工作区路径：」） |
| R-GC-144 | G-E4-03 | 内容 | 路径超长 | 折行或中段省略，完整值可悬停查看 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-E4-03→internal/ui/confirm.go:App.confirmPathRow；标记 = 文案「工作区路径：」） |
| R-GC-145 | G-E4-03 | 样式 | 路径行与描述同屏时 | 同字号次要色，行距紧 | — | 判据 = 字号 UI 正文 12.25px；行距 3.5 / 7 / 10.5 / 14px（3.5px 网格） | 已实现（映射 = G-E4-03→internal/ui/confirm.go:App.confirmPathRow；标记 = TextColor） |
| R-GC-146 | G-E4-03 | 无障碍 | 读屏进入该元素时 | 读出路径 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-E4-03→internal/ui/confirm.go:App.confirmPathRow；标记 = Label） |
| R-GC-147 | G-E4-03 | 内容 | 路径为空 | 不渲染该行 | — | 屏上 该行 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GC-148 | G-E4-03 | 样式 | 路径行渲染后 | 14px 次要色、可用等宽、单行或折行 | — | 判据 = 字号 正文 14px；其余 = 字号 12.25 / 行高 18 / 字重 400，说明文字 10.5 / 14（TX-TYPE-BODY / TX-TYPE-MICRO）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-E4-03→internal/ui/confirm.go:App.confirmPathRow；标记 = TextColor） |
| R-GC-149 | G-E4-03 | 异常 | 路径含不可见字符 | 按可见字符显示 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（映射到位但没有该维度的标记）：G-E4-03→internal/ui/confirm.go:App.confirmPathRow |
| R-GC-150 | G-E4-03 | 关联 | 描述与路径行上下排列时 | 路径行紧贴描述下缘，两句不合并 | — | 屏上 文字行 计数 = 1 | 已实现（映射 = G-E4-03→internal/ui/confirm.go:App.confirmPathRow；标记 = Select） |
| R-GC-151 | G-E4-04 | 内容 | 未观察（条件）：删除失败 | 错误文字显示失败原因 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-E4-04→internal/ui/confirm.go:App.confirmDialog；标记 = 文案「正在运行的会话会被中断，其队列中的后续消息也会被丢弃。」等 2 条） |
| R-GC-152 | G-E4-04 | 交互 | 「重试」在屏上时 | 先清错误再提交 | — | 错误段恒 1 | 已实现（映射 = G-E4-04→internal/ui/confirm.go:App.confirmDialog；标记 = Key） |
| R-GC-153 | G-E4-04 | 样式 | 错误段在屏上时 | 14px 危险色、按钮行之上 | — | 判据 = 字号 正文 14px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-E4-04→internal/ui/confirm.go:App.confirmDialog；标记 = TextColor） |
| R-GC-154 | G-E4-04 | 无障碍 | 读屏进入该元素时 | `role=alert` 播报一次 | — | 播报次数 = 1 | 已实现（映射 = G-E4-04→internal/ui/confirm.go:App.confirmDialog；标记 = Role） |
| R-GC-155 | G-E4-04 | 内容 | 失败原因为空 | 不渲染错误行 | — | 屏上 错误行 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GC-156 | G-E4-04 | 交互 | 鼠标点击错误文字时 | 无动作 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-E4-04→internal/ui/confirm.go:App.confirmDialog；标记 = Key） |
| R-GC-157 | G-E4-04 | 样式 | 错误段出现时 | 按钮行之上、路径行之下 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-E4-04→internal/ui/confirm.go:App.confirmDialog；标记 = TextColor） |
| R-GC-158 | G-E4-04 | 关联 | 错误段出现在路径行与按钮之间时 | 错误在路径行之下、按钮之上 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-E4-04→internal/ui/confirm.go:App.confirmDialog；标记 = Select） |
| R-GC-159 | G-E4-05 | 状态 | 未观察（条件）：弹窗在场 | 次按钮「取消」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GC-160 | G-E4-05 | 交互 | 鼠标点击该元素时 | 关闭；不删除任何会话；工作区仍在侧栏 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-E4-05→internal/ui/confirm.go:App.confirmDialog；标记 = Key） |
| R-GC-161 | G-E4-05 | 键盘 | 按 Esc 时 | 等同点击 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-E4-05→internal/ui/confirm.go:App.confirmDialog；标记 = Shortcut） |
| R-GC-162 | G-E4-05 | 样式 | 次按钮在屏上时 | 描边、高 28 | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-E4-05→internal/ui/confirm.go:App.confirmDialog；标记 = TextColor） |
| R-GC-163 | G-E4-05 | 无障碍 | 读屏进入该元素时 | 读作按钮「取消」 | — | 读屏文本 = 「取消」 | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GC-164 | G-E4-05 | 内容 | 文案在屏上时 | 恒为「取消」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GC-165 | G-E4-05 | 无障碍 | 读屏进入该元素时 | 读作按钮「取消」，默认焦点 | — | 读屏文本 = 「取消」 | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GC-166 | G-E4-05 | 样式 | 取消与确认按钮同屏时 | 同高 28，描边次按钮 | — | 判据 = 控件尺寸 按钮高 次 28；描边 常规 1px（卡片外框 0.5px / 焦点环 3px） | 已实现（映射 = G-E4-05→internal/ui/confirm.go:App.confirmDialog；标记 = TextColor） |
| R-GC-167 | G-E4-06 | 状态 | 未观察（条件）：弹窗在场 | 危险按钮「删除」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/confirm.go:confirmCopyDeleteOne「删除」 |
| R-GC-168 | G-E4-06 | 交互 | 鼠标点击该元素时 | 删除该工作区下全部会话；工作区本身保留或移除（按实现一致）；关闭弹窗 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 已实现（映射 = G-E4-06→internal/ui/confirm.go:App.confirmDialog；标记 = Key） |
| R-GC-169 | G-E4-06 | 异常 | 部分失败 | 成功的已删、失败的保留并给原因 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-E4-06→internal/ui/confirm.go:App.confirmDialog；标记 = err） |
| R-GC-170 | G-E4-06 | 样式 | 危险按钮在屏上时 | 危险色底 + 白字、高 28 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；色值取 token：主色 #349651/#63cb7b、正文 #0a0a0a/#ececec、次文 #737373/#8a8a8a、边框 #e5e5e5/#2e2e2e、警示 #e17100/#ffb900，偏差 0（CL-ACCENT / CL-TEXT / CL-TEXT-2 / CL-BORDER / CL-WARN） | 已实现（映射 = G-E4-06→internal/ui/confirm.go:App.confirmDialog；标记 = TextColor） |
| R-GC-171 | G-E4-06 | 无障碍 | 读屏进入该元素时 | 读作按钮「删除」，并说明会删除该工作区下的会话 | — | 读屏文本 = 「删除」 | 已实现（逐字）：internal/ui/confirm.go:confirmCopyDeleteOne「删除」 |
| R-GC-172 | G-E4-06 | 键盘 | 按 Enter 时 | 等同点击 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-E4-06→internal/ui/confirm.go:App.confirmDialog；标记 = Shortcut） |
| R-GC-173 | G-E4-06 | 内容 | 无工作区会话 | 「删除」仍可点（会移除工作区本身，按实现一致） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 已实现（逐字）：internal/ui/confirm.go:confirmCopyDeleteOne「删除」 |
| R-GC-174 | G-E4-06 | 异常 | 删除后工作区为空 | 工作区保留或移除，但侧栏不同时出现两份 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 已实现（映射 = G-E4-06→internal/ui/confirm.go:App.confirmDialog；标记 = err） |

### 5.5 确认域维度覆盖说明

21 项全部 ≥6 维。四类确认框的「取消」按钮（G-E1-03、G-E2-04、G-E3-05、G-E4-05）到 6 维：无「内容 / 动画」——文案恒定、无自身过渡，动画由弹窗整体承担。

---

## 6. 重命名（16 元素）

两组：会话重命名（F1，9）、工作区重命名（F2，7）。F1 在本机没有 `SessionTitle` 能力探测：改名直接走 `Intents.RenameSession`（协议已有 `session.rename`），弹窗可通达。

> 实现状态（2026-11-20）：弹窗已建（`internal/ui/rename_dialog.go`），入口为标题栏会话名双击、标题栏 `⋯` 菜单、左栏会话行 / 工作区行菜单；行内快捷改名保留在 F2（`app.go:viewKeys`）与 `sidebar.go:renameRow` / `groupRenameRow`。仍未实现：弹窗动效、描述两行上限、错误行「重试」、输入法组字抑制、默认焦点在「取消」。测试见 `internal/ui/rename_dialog_test.go`、`p0_facts_test.go:TestHeaderRenameIsAvailable`、`interaction_test.go:TestWorkspaceMenuPinsRenamesAndRemoves`、`uiwire/intents_test.go:TestIntentRename`。

### 6.1 会话重命名（G-F1-01–G-F1-09）

| 需求 ID | 元素 ID | 维度 | 场景 | 期望 | 异常 | 验收 | 状态 |
|---|---|---|---|---|---|---|---|
| R-GR-001 | G-F1-01 | 内容 | 未观察（条件）：`SessionTitle` 就绪后从会话行菜单点「重命名」 | 弹窗标题逐字「重命名会话」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/rename_dialog.go:renameCopySessionTitle「重命名会话」 |
| R-GR-002 | G-F1-01 | 状态 | `SessionTitle` 不可用 | 对话框不出现（菜单项本身置灰且文案标注不可用） | — | 屏上 标题 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GR-003 | G-F1-01 | 样式 | 标题在屏上时 | 16px 级半粗；圆角 + 阴影 + 遮罩 | — | 判据 = 字号 对话框标题 15.75px；字重 半粗 600；圆角 xl 12.75px；其余 = 色值取 token：主色 #349651/#63cb7b、正文 #0a0a0a/#ececec、次文 #737373/#8a8a8a、边框 #e5e5e5/#2e2e2e、警示 #e17100/#ffb900，偏差 0（CL-ACCENT / CL-TEXT / CL-TEXT-2 / CL-BORDER / CL-WARN） | 已实现（映射 = G-F1-01→internal/ui/rename_dialog.go:App.renameDialogView；标记 = TextColor） |
| R-GR-004 | G-F1-01 | 无障碍 | 读屏进入该元素时 | 作为对话框名读出 | — | 读屏文本 = 「重命名会话」 | 已实现（映射 = G-F1-01→internal/ui/rename_dialog.go:App.renameDialogView；标记 = Role） |
| R-GR-005 | G-F1-01 | 异常 | 弹窗打开后能力掉线 | 就绪提示转「SessionTitle 不可用」，「确认」禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/ui/composer.go:App.goalRow「确认」 |
| R-GR-006 | G-F1-01 | 窄窗 | 窗口收窄到最窄宽度时 | 标题单行，输入框与按钮行完整 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-F1-01→internal/ui/rename_dialog.go:App.renameDialogView |
| R-GR-007 | G-F1-01 | 内容 | 标题超长 | 单行不折行 | — | 标题 1 行 | 已实现（映射 = G-F1-01→internal/ui/rename_dialog.go:App.renameDialogView；标记 = 文案「工作区路径：」等 2 条） |
| R-GR-008 | G-F1-01 | 内容 | 菜单项「重命名」被点击后 | 菜单项为「重命名」，弹窗标题为「重命名会话」，不同级不冲突 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/rename_dialog.go:renameCopySessionTitle「重命名」 |
| R-GR-009 | G-F1-02 | 内容 | 未观察（条件）：重命名会话弹窗打开时 | 描述逐字「输入新的会话名称」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/rename_dialog.go:renameCopySessionDesc「输入新的会话名称」 |
| R-GR-010 | G-F1-02 | 样式 | 描述在屏上时 | 14px 级次要色 | — | 判据 = 字号 正文 14px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-F1-02→internal/ui/rename_dialog.go:App.renameDialogView；标记 = TextColor） |
| R-GR-011 | G-F1-02 | 无障碍 | 读屏进入该元素时 | 作为描述读出 | — | 读屏文本 = 「输入新的会话名称」 | 已实现（映射 = G-F1-02→internal/ui/rename_dialog.go:App.renameDialogView；标记 = Role） |
| R-GR-012 | G-F1-02 | 异常 | 描述折行 | 两行时不挤压输入框位置 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-F1-02→internal/ui/rename_dialog.go:App.renameDialogView；标记 = err） |
| R-GR-013 | G-F1-02 | 内容 | 描述在屏上时 | 单句「输入新的会话名称」，无句号或按语言包 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/rename_dialog.go:renameCopySessionDesc「输入新的会话名称」 |
| R-GR-014 | G-F1-02 | 内容 | 描述折行 | 两行时不挤压输入框 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-F1-02→internal/ui/rename_dialog.go:App.renameDialogView；标记 = 文案「工作区路径：」等 2 条） |
| R-GR-015 | G-F1-02 | 样式 | 描述在屏上时 | 14px 次要色 | — | 判据 = 字号 正文 14px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-F1-02→internal/ui/rename_dialog.go:App.renameDialogView；标记 = TextColor） |
| R-GR-016 | G-F1-02 | 异常 | 描述为空 | 不允许：描述说明该字段用途 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-F1-02→internal/ui/rename_dialog.go:App.renameDialogView；标记 = err） |
| R-GR-017 | G-F1-02 | 记忆 | 取消后再次打开时 | 描述不变 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-F1-02→internal/ui/rename_dialog.go:App.renameDialogView；标记 = View.@App.confirmRenameDialog） |
| R-GR-018 | G-F1-03 | 内容 | 未观察（条件）：弹窗在场 | 字段标签逐字「会话名称」（14px 中等） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/rename_dialog.go:renameCopySessionDesc「会话名称」 |
| R-GR-019 | G-F1-03 | 关联 | 字段标签与输入框上下排列时 | 标签在上、输入框在下，左缘对齐 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM） | 已实现（映射 = G-F1-03→internal/ui/rename_dialog.go:App.renameDialogView；标记 = Select） |
| R-GR-020 | G-F1-03 | 样式 | 字段标签在屏上时 | 14px、中等字重、主前景色 | — | 判据 = 字号 UI 正文 12.25px；字重 常规 400 / 中粗 500 / 半粗 600 / 粗 700 | 已实现（映射 = G-F1-03→internal/ui/rename_dialog.go:App.renameDialogView；标记 = TextColor） |
| R-GR-021 | G-F1-03 | 无障碍 | 读屏进入该元素时 | 与输入框关联（label for），读屏念「会话名称，编辑框」 | — | 读屏文本 = 「会话名称，编辑框」 | 已实现（映射 = G-F1-03→internal/ui/rename_dialog.go:App.renameDialogView；标记 = Role） |
| R-GR-022 | G-F1-03 | 异常 | 标签与占位重复 | 两者文案不同（标签「会话名称」、占位「输入会话名称」） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/rename_dialog.go:renameCopySessionDesc「会话名称」 |
| R-GR-023 | G-F1-03 | 内容 | 标签在屏上时 | 恒为「会话名称」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/rename_dialog.go:renameCopySessionDesc「会话名称」 |
| R-GR-024 | G-F1-03 | 状态 | 能力不可用 | 标签仍显示（字段仍在） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-F1-03→internal/ui/rename_dialog.go:App.renameDialogView；标记 = Disabled） |
| R-GR-025 | G-F1-03 | 异常 | 标签与占位重复 | 标签「会话名称」≠ 占位「输入会话名称」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/rename_dialog.go:renameCopySessionDesc「会话名称」 |
| R-GR-026 | G-F1-04 | 状态 | 未观察（条件）：弹窗打开 | 输入框预填当前标题，聚焦即全选（可直接覆盖） | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-F1-04→internal/ui/rename_dialog.go:App.renameDialogView；标记 = Disabled） |
| R-GR-027 | G-F1-04 | 内容 | 占位文案在屏上时 | 逐字「输入会话名称」；有预填时占位不可见 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/rename_dialog.go:renameCopySessionPlace「输入会话名称」 |
| R-GR-028 | G-F1-04 | 交互 | 在输入框键入时 | 逐字更新校验提示与「确认」可用性 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/composer.go:App.goalRow「确认」 |
| R-GR-029 | G-F1-04 | 交互 | 输入法组字 | 组字期间不触发校验抖动 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GR-030 | G-F1-04 | 样式 | 输入框在屏上时 | 描边 + 圆角 6、高 32、14px；聚焦时描边转强调色 | — | 判据 = 控件尺寸 输入框高 32（行内 25）；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px；色值 边框 明 #e5e5e5 / 暗 #2e2e2e | 已实现（映射 = G-F1-04→internal/ui/rename_dialog.go:App.renameDialogView；标记 = TextColor） |
| R-GR-031 | G-F1-04 | 无障碍 | 读屏进入该元素时 | 读作文本框，值为当前标题 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-F1-04→internal/ui/rename_dialog.go:App.renameDialogView；标记 = Role） |
| R-GR-032 | G-F1-04 | 异常 | 粘贴超长文本 | 超长部分被拒或折叠为校验错误（见 G-F1-05），不无声截断 | — | 判据 = 不截断：屏上以「…」收尾的出现次数 = 0，超长按期望列的校验错误行处理（TX-ELLIPSIS 出现次数 0 / TX-TIP）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GR-033 | G-F1-04 | 内容 | 当前标题超长（>80） | 预填完整值，并立即显示超限校验提示 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-F1-04→internal/ui/rename_dialog.go:App.renameDialogView；标记 = 文案「工作区路径：」等 2 条） |
| R-GR-034 | G-F1-05 | 内容 | 未观察（条件）：键入 81 字符 | 校验提示逐字「最多使用 {max} 个字符」，12px 灰 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-F1-05→internal/ui/rename_dialog.go:App.renameDialogView；标记 = 文案「工作区路径：」等 2 条） |
| R-GR-035 | G-F1-05 | 内容 | 输入名称与当前名称相同时 | 提示逐字「请输入不同的会话名称」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/rename_dialog.go:renameCopySameSession「请输入不同的会话名称」 |
| R-GR-036 | G-F1-05 | 内容 | 含控制字符或行分隔符 | 提示逐字「会话名称中不能包含控制字符」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/rename_dialog.go:renameCopyCtrlSession「会话名称中不能包含控制字符」 |
| R-GR-037 | G-F1-05 | 内容 | 去空白后为空 | 提示逐字「会话名称不能为空」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/rename_dialog.go:renameCopyEmptySession「会话名称不能为空」 |
| R-GR-038 | G-F1-05 | 状态 | 四分支互斥 | 同一时刻只显示一条（按检查顺序取第一条） | — | 提示计数为 1 | 已实现（映射 = G-F1-05→internal/ui/rename_dialog.go:App.renameDialogView；标记 = Disabled） |
| R-GR-039 | G-F1-05 | 关联 | 任一校验提示在场时 | 任一提示在场时「确认」禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/ui/composer.go:App.goalRow「确认」 |
| R-GR-040 | G-F1-05 | 样式 | 校验提示 | 12px 次要色（不是危险色：属预校验） | — | 判据 = 字号 UI 正文 12.25px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-F1-05→internal/ui/rename_dialog.go:App.renameDialogView；标记 = TextColor） |
| R-GR-041 | G-F1-05 | 无障碍 | 读屏进入该元素时 | 读出当前提示（不刷屏，改动时更新） | — | 读屏文本 = 「最多使用 {max} 个字符」 | 已实现（映射 = G-F1-05→internal/ui/rename_dialog.go:App.renameDialogView；标记 = Role） |
| R-GR-042 | G-F1-05 | 异常 | 输入 80 个全角字符 | 按字符数而非字节数判定，提示不误报 | — | 屏上 校验提示 计数 = 0（80 个全角字符时） | 已实现（映射 = G-F1-05→internal/ui/rename_dialog.go:App.renameDialogView；标记 = err） |
| R-GR-043 | G-F1-06 | 内容 | 未观察（条件）：能力探测中 | 就绪提示逐字「正在检查 SessionTitle…」，12px 灰 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-F1-06→internal/ui/rename_dialog.go:App.renameDialogView；标记 = 文案「工作区路径：」等 2 条） |
| R-GR-044 | G-F1-06 | 内容 | 能力不可用 | 提示逐字「SessionTitle 不可用」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-F1-06→internal/ui/rename_dialog.go:App.renameDialogView；标记 = 文案「工作区路径：」等 2 条） |
| R-GR-045 | G-F1-06 | 状态 | 探测中 / 不可用 / 就绪 | 三态互斥；就绪时该行不显示任何文案 | — | 屏上 就绪提示 计数 = 1 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GR-046 | G-F1-06 | 关联 | 路径探测中或不可用时 | 探测中或不可用时「确认」禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/ui/composer.go:App.goalRow「确认」 |
| R-GR-047 | G-F1-06 | 样式 | 就绪提示 | 12px 次要色、位于输入框下方 | — | 判据 = 字号 UI 正文 12.25px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-F1-06→internal/ui/rename_dialog.go:App.renameDialogView；标记 = TextColor） |
| R-GR-048 | G-F1-06 | 无障碍 | 读屏进入该元素时 | 读出当前提示 | — | 读屏文本 = 「正在检查 SessionTitle…」 | 已实现（映射 = G-F1-06→internal/ui/rename_dialog.go:App.renameDialogView；标记 = Role） |
| R-GR-049 | G-F1-06 | 异常 | 探测超时 | 停在该提示或转「不可用」，不无限停留 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/account.go:ErrActionsUnavailable「不可用」 |
| R-GR-050 | G-F1-06 | 内容 | 探测中时长 | 通常 <1s；期间「确认」禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/ui/composer.go:App.goalRow「确认」 |
| R-GR-051 | G-F1-07 | 内容 | 未观察（条件）：提交后 RPC 失败 | 错误文字显示失败原因（12px 危险色，`role=alert`） | — | 判据 = 字号 UI 正文 12.25px；字重 常规 400 / 中粗 500 / 半粗 600 / 粗 700 | 已实现（映射 = G-F1-07→internal/ui/rename_dialog.go:App.renameDialogView；标记 = 文案「工作区路径：」等 2 条） |
| R-GR-052 | G-F1-07 | 交互 | 「重试」在屏上时 | 先清错误再提交，不叠加 | — | 错误段恒 1 | 已实现（映射 = G-F1-07→internal/ui/rename_dialog.go:App.renameDialogView；标记 = OnSubmit） |
| R-GR-053 | G-F1-07 | 样式 | 错误段在屏上时 | 12px 危险色、按钮行之上 | — | 判据 = 字号 UI 正文 12.25px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-F1-07→internal/ui/rename_dialog.go:App.renameDialogView；标记 = TextColor） |
| R-GR-054 | G-F1-07 | 无障碍 | 读屏进入该元素时 | `role=alert` 播报一次 | — | 播报次数 = 1 | 已实现（映射 = G-F1-07→internal/ui/rename_dialog.go:App.renameDialogView；标记 = Role） |
| R-GR-055 | G-F1-07 | 异常 | 原因超长 | 折行完整 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-F1-07→internal/ui/rename_dialog.go:App.renameDialogView；标记 = err） |
| R-GR-056 | G-F1-07 | 内容 | 失败原因为空 | 不渲染错误行 | — | 屏上 错误行 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GR-057 | G-F1-07 | 内容 | 失败原因含英文 | 原文照显 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-F1-07→internal/ui/rename_dialog.go:App.renameDialogView；标记 = 文案「工作区路径：」等 2 条） |
| R-GR-058 | G-F1-07 | 交互 | 鼠标点击错误文字时 | 无动作 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-F1-07→internal/ui/rename_dialog.go:App.renameDialogView；标记 = OnSubmit） |
| R-GR-059 | G-F1-08 | 状态 | 未观察（条件）：弹窗在场 | 次按钮「取消」（`outline`） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GR-060 | G-F1-08 | 交互 | 鼠标点击该元素时 | 关闭；会话标题不变 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-F1-08→internal/ui/rename_dialog.go:App.renameDialogView；标记 = OnSubmit） |
| R-GR-061 | G-F1-08 | 键盘 | 按 Esc 时 | 等同点击 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-F1-08→internal/ui/rename_dialog.go:App.renameDialogView；标记 = Ctrl@App.renameDialogCopy） |
| R-GR-062 | G-F1-08 | 样式 | 次按钮在屏上时 | 描边、高 28 | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-F1-08→internal/ui/rename_dialog.go:App.renameDialogView；标记 = TextColor） |
| R-GR-063 | G-F1-08 | 无障碍 | 读屏进入该元素时 | 读作按钮「取消」 | — | 读屏文本 = 「取消」 | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GR-064 | G-F1-08 | 内容 | 文案在屏上时 | 恒为「取消」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GR-065 | G-F1-08 | 状态 | 提交进行中时 | 取消禁用（重命名已发出） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-F1-08→internal/ui/rename_dialog.go:App.renameDialogView；标记 = Disabled） |
| R-GR-066 | G-F1-08 | 无障碍 | 读屏进入该元素时 | 读作按钮「取消」，默认焦点 | — | 读屏文本 = 「取消」 | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GR-067 | G-F1-09 | 状态 | 未观察（条件）：校验通过 ∧ 能力就绪 ∧ 非提交中 | 「确认」改为可用态（默认禁用） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/ui/composer.go:App.goalRow「确认」 |
| R-GR-068 | G-F1-09 | 交互 | 鼠标点击该元素时 | 重命名该会话；成功后关闭弹窗、侧栏行标题即时更新 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-F1-09→internal/ui/rename_dialog.go:App.renameDialogView；标记 = OnSubmit） |
| R-GR-069 | G-F1-09 | 状态 | 提交进行中时 | 禁用并显示执行态 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-F1-09→internal/ui/rename_dialog.go:App.renameDialogView；标记 = Disabled） |
| R-GR-070 | G-F1-09 | 样式 | 主按钮在屏上时 | 实心主色、白字、高 28；禁用时降透明 | — | 判据 = 底色 主色 明 #339751 / 暗 #63cb7b；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；色值取 token：主色 #349651/#63cb7b、正文 #0a0a0a/#ececec、次文 #737373/#8a8a8a、边框 #e5e5e5/#2e2e2e、警示 #e17100/#ffb900，偏差 0（CL-ACCENT / CL-TEXT / CL-TEXT-2 / CL-BORDER / CL-WARN） | 已实现（映射 = G-F1-09→internal/ui/rename_dialog.go:App.renameDialogView；标记 = TextColor） |
| R-GR-071 | G-F1-09 | 无障碍 | 读屏进入该元素时 | 读作按钮「确认」，禁用时播报原因（校验未通过/能力不可用） | — | 读屏文本 = 「确认」 | 已实现（逐字）：internal/ui/composer.go:App.goalRow「确认」 |
| R-GR-072 | G-F1-09 | 键盘 | 按 Enter 时 | 可用时等同点击；禁用时不触发 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GR-073 | G-F1-09 | 内容 | 可用态文案渲染后 | 恒为「确认」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/composer.go:App.goalRow「确认」 |
| R-GR-074 | G-F1-09 | 异常 | 提交时能力掉线 | 给失败原因，标题不变 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-F1-09→internal/ui/rename_dialog.go:App.renameDialogView；标记 = err） |

### 6.2 工作区重命名（G-F2-01–G-F2-07）

条件：侧栏工作区分组菜单；改的是本地项目别名（不依赖 SessionTitle）。入口已采（C16 169,450、C45 169,442）。

| 需求 ID | 元素 ID | 维度 | 场景 | 期望 | 异常 | 验收 | 状态 |
|---|---|---|---|---|---|---|---|
| R-GR-075 | G-F2-01 | 内容 | 未观察（条件）：工作区行 `⋯` →「重命名」 | 标题逐字「重命名工作区」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/rename_dialog.go:renameCopyWorkspaceT「重命名工作区」 |
| R-GR-076 | G-F2-01 | 状态 | 无工作区（空态） | 入口不出现，自然无弹窗 | — | 屏上 标题 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GR-077 | G-F2-01 | 样式 | 标题在屏上时 | 16px 级半粗；圆角 + 阴影 + 遮罩 | — | 判据 = 字号 对话框标题 15.75px；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px | 已实现（映射 = G-F2-01→internal/ui/rename_dialog.go:App.renameDialogView；标记 = TextColor） |
| R-GR-078 | G-F2-01 | 无障碍 | 读屏进入该元素时 | 作为对话框名读出 | — | 读屏文本 = 「重命名工作区」 | 已实现（映射 = G-F2-01→internal/ui/rename_dialog.go:App.renameDialogView；标记 = Role） |
| R-GR-079 | G-F2-01 | 异常 | 工作区在弹窗期间被移除 | 确认时给失败原因，不写入别名 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GR-080 | G-F2-01 | 内容 | 标题超长 | 单行不折行 | — | 标题 1 行 | 已实现（映射 = G-F2-01→internal/ui/rename_dialog.go:App.renameDialogView；标记 = 文案「工作区路径：」等 2 条） |
| R-GR-081 | G-F2-01 | 状态 | 工作区无别名（用路径名） | 标题仍为「重命名工作区」，输入框预填当前显示名 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/rename_dialog.go:renameCopyWorkspaceT「重命名工作区」 |
| R-GR-082 | G-F2-01 | 异常 | 工作区已移除 | 入口不出现，自然无弹窗 | — | 屏上 标题 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GR-083 | G-F2-02 | 内容 | 未观察（条件）：重命名工作区弹窗打开时 | 描述逐字「输入新的工作区名称」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/rename_dialog.go:renameCopyWorkspaceDesc「输入新的工作区名称」 |
| R-GR-084 | G-F2-02 | 样式 | 描述在屏上时 | 14px 级次要色 | — | 判据 = 字号 正文 14px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-F2-02→internal/ui/rename_dialog.go:App.renameDialogView；标记 = TextColor） |
| R-GR-085 | G-F2-02 | 无障碍 | 读屏进入该元素时 | 作为描述读出 | — | 读屏文本 = 「输入新的工作区名称」 | 已实现（映射 = G-F2-02→internal/ui/rename_dialog.go:App.renameDialogView；标记 = Role） |
| R-GR-086 | G-F2-02 | 异常 | 描述折行 | 不挤压输入框位置 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-F2-02→internal/ui/rename_dialog.go:App.renameDialogView；标记 = err） |
| R-GR-087 | G-F2-02 | 内容 | 描述在屏上时 | 单句「输入新的工作区名称」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/rename_dialog.go:renameCopyWorkspaceDesc「输入新的工作区名称」 |
| R-GR-088 | G-F2-02 | 样式 | 描述在屏上时 | 14px 次要色 | — | 判据 = 字号 正文 14px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-F2-02→internal/ui/rename_dialog.go:App.renameDialogView；标记 = TextColor） |
| R-GR-089 | G-F2-02 | 异常 | 描述为空 | 不允许 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-F2-02→internal/ui/rename_dialog.go:App.renameDialogView；标记 = err） |
| R-GR-090 | G-F2-02 | 内容 | 描述折行 | 不挤压输入框 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 已实现（映射 = G-F2-02→internal/ui/rename_dialog.go:App.renameDialogView；标记 = 文案「工作区路径：」等 2 条） |
| R-GR-091 | G-F2-02 | 记忆 | 取消后再次打开时 | 描述不变 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-F2-02→internal/ui/rename_dialog.go:App.renameDialogView；标记 = View.@App.confirmRenameDialog） |
| R-GR-092 | G-F2-03 | 内容 | 未观察（条件）：弹窗在场 | 字段标签逐字「工作区名称」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/rename_dialog.go:renameCopyWorkspaceDesc「工作区名称」 |
| R-GR-093 | G-F2-03 | 关联 | 字段标签与输入框上下排列时 | 标签在上、输入框在下，左缘对齐 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM） | 已实现（映射 = G-F2-03→internal/ui/rename_dialog.go:App.renameDialogView；标记 = Select） |
| R-GR-094 | G-F2-03 | 样式 | 字段标签在屏上时 | 14px 中等字重主前景 | — | 判据 = 字号 UI 正文 12.25px；字重 常规 400 / 中粗 500 / 半粗 600 / 粗 700 | 已实现（映射 = G-F2-03→internal/ui/rename_dialog.go:App.renameDialogView；标记 = TextColor） |
| R-GR-095 | G-F2-03 | 无障碍 | 读屏进入该元素时 | 与输入框关联，读屏念「工作区名称，编辑框」 | — | 读屏文本 = 「工作区名称，编辑框」 | 已实现（映射 = G-F2-03→internal/ui/rename_dialog.go:App.renameDialogView；标记 = Role） |
| R-GR-096 | G-F2-03 | 异常 | 标签与占位重复 | 标签「工作区名称」≠ 占位「输入工作区名称」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/rename_dialog.go:renameCopyWorkspaceDesc「工作区名称」 |
| R-GR-097 | G-F2-03 | 内容 | 标签在屏上时 | 恒为「工作区名称」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/rename_dialog.go:renameCopyWorkspaceDesc「工作区名称」 |
| R-GR-098 | G-F2-03 | 异常 | 标签与占位重复 | 标签「工作区名称」≠ 占位「输入工作区名称」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/rename_dialog.go:renameCopyWorkspaceDesc「工作区名称」 |
| R-GR-099 | G-F2-03 | 关联 | 标签、输入框与路径行三段同屏时 | 标签/输入框在上、路径行在下，三段左缘对齐 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM） | 已实现（映射 = G-F2-03→internal/ui/rename_dialog.go:App.renameDialogView；标记 = Select） |
| R-GR-100 | G-F2-04 | 状态 | 未观察（条件）：弹窗打开 | 输入框预填当前别名，聚焦即全选 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-F2-04→internal/ui/rename_dialog.go:App.renameDialogView；标记 = Disabled） |
| R-GR-101 | G-F2-04 | 内容 | 输入框清空后 | 占位文案逐字「输入工作区名称」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/rename_dialog.go:renameCopyWorkspacePlc「输入工作区名称」 |
| R-GR-102 | G-F2-04 | 交互 | 在输入框键入时 | 实时更新「确认」可用性（非空且与当前不同） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/ui/composer.go:App.goalRow「确认」 |
| R-GR-103 | G-F2-04 | 样式 | 输入框在屏上时 | 描边 + 圆角 6、高 32；聚焦转强调色 | — | 判据 = 控件尺寸 输入框高 32（行内 25）；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px | 已实现（映射 = G-F2-04→internal/ui/rename_dialog.go:App.renameDialogView；标记 = TextColor） |
| R-GR-104 | G-F2-04 | 无障碍 | 读屏进入该元素时 | 读作文本框，值为当前别名 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-F2-04→internal/ui/rename_dialog.go:App.renameDialogView；标记 = Role） |
| R-GR-105 | G-F2-04 | 异常 | 输入含路径分隔符 | 允许（别名可含 `/`？按实现一致），但不得含控制字符 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-F2-04→internal/ui/rename_dialog.go:App.renameDialogView；标记 = err） |
| R-GR-106 | G-F2-04 | 内容 | 当前别名超长 | 预填完整值，输入框可横向滚动 | — | 判据 = 滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-F2-04→internal/ui/rename_dialog.go:App.renameDialogView；标记 = 文案「工作区路径：」等 2 条） |
| R-GR-107 | G-F2-04 | 异常 | 粘贴含控制字符 | 显示校验或拒绝，不写入 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GR-108 | G-F2-05 | 内容 | 未观察（条件）：弹窗在场 | 工作区路径（14px 灰）显示在输入框下方 | — | 判据 = 字号 正文 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-F2-05→internal/ui/rename_dialog.go:App.renameDialogView；标记 = 文案「工作区路径：」等 2 条） |
| R-GR-109 | G-F2-05 | 内容 | 路径超长 | 折行或中段省略，悬停可见完整 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-F2-05→internal/ui/rename_dialog.go:App.renameDialogView；标记 = 文案「工作区路径：」等 2 条） |
| R-GR-110 | G-F2-05 | 样式 | 路径行与描述同屏时 | 同字号次要色 | — | 判据 = 字号 UI 正文 12.25px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-F2-05→internal/ui/rename_dialog.go:App.renameDialogView；标记 = TextColor） |
| R-GR-111 | G-F2-05 | 无障碍 | 读屏进入该元素时 | 读出路径 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-F2-05→internal/ui/rename_dialog.go:App.renameDialogView；标记 = Role） |
| R-GR-112 | G-F2-05 | 内容 | 路径为空 | 不渲染该行 | — | 屏上 该行 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GR-113 | G-F2-05 | 样式 | 路径行渲染后 | 14px 次要色、单行或折行 | — | 判据 = 字号 正文 14px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-F2-05→internal/ui/rename_dialog.go:App.renameDialogView；标记 = TextColor） |
| R-GR-114 | G-F2-05 | 异常 | 路径含不可见字符 | 按可见字符显示 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-F2-05→internal/ui/rename_dialog.go:App.renameDialogView；标记 = err） |
| R-GR-115 | G-F2-05 | 关联 | 路径行渲染在标签与输入框之下时 | 路径行在标签与输入框之下 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-F2-05→internal/ui/rename_dialog.go:App.renameDialogView；标记 = Select） |
| R-GR-116 | G-F2-06 | 状态 | 未观察（条件）：弹窗在场 | 次按钮「取消」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GR-117 | G-F2-06 | 交互 | 鼠标点击该元素时 | 关闭；别名不变 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-F2-06→internal/ui/rename_dialog.go:App.renameDialogView；标记 = OnSubmit） |
| R-GR-118 | G-F2-06 | 键盘 | 按 Esc 时 | 等同点击 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-F2-06→internal/ui/rename_dialog.go:App.renameDialogView；标记 = Ctrl@App.renameDialogCopy） |
| R-GR-119 | G-F2-06 | 样式 | 次按钮在屏上时 | 描边、高 28 | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-F2-06→internal/ui/rename_dialog.go:App.renameDialogView；标记 = TextColor） |
| R-GR-120 | G-F2-06 | 无障碍 | 读屏进入该元素时 | 读作按钮「取消」 | — | 读屏文本 = 「取消」 | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GR-121 | G-F2-06 | 内容 | 文案在屏上时 | 恒为「取消」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GR-122 | G-F2-06 | 状态 | 提交进行中时 | 禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-F2-06→internal/ui/rename_dialog.go:App.renameDialogView；标记 = Disabled） |
| R-GR-123 | G-F2-06 | 无障碍 | 读屏进入该元素时 | 读作按钮「取消」，默认焦点 | — | 读屏文本 = 「取消」 | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GR-124 | G-F2-06 | 样式 | 取消与确认按钮同屏时 | 同高 28，描边次按钮 | — | 判据 = 控件尺寸 按钮高 次 28；描边 常规 1px（卡片外框 0.5px / 焦点环 3px） | 已实现（映射 = G-F2-06→internal/ui/rename_dialog.go:App.renameDialogView；标记 = TextColor） |
| R-GR-125 | G-F2-07 | 状态 | 未观察（条件）：输入非空且与当前不同 | 「确认」可用（默认禁用） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/ui/composer.go:App.goalRow「确认」 |
| R-GR-126 | G-F2-07 | 交互 | 鼠标点击该元素时 | 更新别名；关闭弹窗；侧栏工作区名即时更新 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-F2-07→internal/ui/rename_dialog.go:App.renameDialogView；标记 = OnSubmit） |
| R-GR-127 | G-F2-07 | 状态 | 提交进行中时 | 禁用并显示执行态 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-F2-07→internal/ui/rename_dialog.go:App.renameDialogView；标记 = Disabled） |
| R-GR-128 | G-F2-07 | 样式 | 主按钮在屏上时 | 实心主色、白字、高 28 | — | 判据 = 底色 主色 明 #339751 / 暗 #63cb7b；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-F2-07→internal/ui/rename_dialog.go:App.renameDialogView；标记 = TextColor） |
| R-GR-129 | G-F2-07 | 无障碍 | 读屏进入该元素时 | 读作按钮「确认」，禁用时播报原因 | — | 读屏文本 = 「确认」 | 已实现（逐字）：internal/ui/composer.go:App.goalRow「确认」 |
| R-GR-130 | G-F2-07 | 键盘 | 按 Enter 时 | 可用时等同点击 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-F2-07→internal/ui/rename_dialog.go:App.renameDialogView；标记 = Ctrl@App.renameDialogCopy） |
| R-GR-131 | G-F2-07 | 内容 | 可用态文案渲染后 | 恒为「确认」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/composer.go:App.goalRow「确认」 |
| R-GR-132 | G-F2-07 | 异常 | 提交失败 | 给原因，别名不变 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-F2-07→internal/ui/rename_dialog.go:App.renameDialogView；标记 = err） |

### 6.3 重命名域维度覆盖说明

16 项全部 ≥6 维。两组的「取消」（G-F1-08、G-F2-06）到 6 维：文案恒定、无过渡，动画随弹窗整体。


---

## 7. 导出与 Fork（22 元素）

两组：导出对话（G1，17）、Fork 会话（G2，5）。入口都采到了（C13 会话行右键「导出」157,352、「在新窗口中打开」157,324；C2 362,167/362,139），对话框本身未采。

### 7.1 导出对话（G-G1-01–G-G1-17）

| 需求 ID | 元素 ID | 维度 | 场景 | 期望 | 异常 | 验收 | 状态 |
|---|---|---|---|---|---|---|---|
| R-GF-001 | G-G1-01 | 内容 | 未观察（条件）：会话菜单「导出」 | 弹窗标题逐字「导出对话」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/export.go:exportTitle「导出对话」 |
| R-GF-002 | G-G1-01 | 样式 | 标题在屏上时 | 16px 级半粗；圆角 + 阴影 + 遮罩 | — | 判据 = 字号 对话框标题 15.75px；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px | 已实现（映射 = G-G1-01→internal/ui/export.go:App.exportPanel；标记 = Width） |
| R-GF-003 | G-G1-01 | 无障碍 | 读屏进入该元素时 | 作为对话框名读出 | — | 读屏文本 = 「导出对话」 | 已实现（映射 = G-G1-01→internal/ui/export.go:App.exportPanel；标记 = Role） |
| R-GF-004 | G-G1-01 | 交互 | 按 Esc 时 | 关闭弹窗，不导出 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-G1-01→internal/ui/export.go:App.exportPanel；标记 = Key） |
| R-GF-005 | G-G1-01 | 异常 | 会话已归档 | 仍可导出（归档不阻止导出），标题不变 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-G1-01→internal/ui/export.go:App.exportPanel；标记 = Reason） |
| R-GF-006 | G-G1-01 | 窄窗 | 窗口收窄到最窄宽度时 | 标题单行；弹窗宽随窗口收缩，内容区可滚动 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-G1-01→internal/ui/export.go:App.exportPanel |
| R-GF-007 | G-G1-01 | 内容 | 标题在屏上时 | 恒为「导出对话」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/export.go:exportTitle「导出对话」 |
| R-GF-008 | G-G1-01 | 异常 | 会话无回合 | 弹窗仍可打开，列表给「没有回合」类状态 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/export.go:exportEmptyTurns「没有回合」 |
| R-GF-009 | G-G1-02 | 内容 | 未观察（条件）：弹窗打开 | 描述逐字「导出「{name}」的会话对话。不包含原始 Provider 请求。」；`{name}` 为会话名，用「」包裹 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-G1-02→internal/ui/export.go:App.exportPanel；标记 = 文案「导出面板内容」等 9 条） |
| R-GF-010 | G-G1-02 | 内容 | 描述超 3 行 | 截断（最多 3 行），不留滚动条 | — | 描述行数 ≤3 | 已实现（映射 = G-G1-02→internal/ui/export.go:App.exportPanel；标记 = 文案「导出面板内容」等 9 条） |
| R-GF-011 | G-G1-02 | 内容 | 会话名为空 | 显示「未命名会话」，不留空「」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/confirm.go:confirmCopyUnnamed「未命名会话」 |
| R-GF-012 | G-G1-02 | 样式 | 描述在屏上时 | 14px 级次要色、行高 1.5 | — | 判据 = 字号 正文 14px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-G1-02→internal/ui/export.go:App.exportPanel；标记 = Width） |
| R-GF-013 | G-G1-02 | 无障碍 | 读屏进入该元素时 | 读出整句（含会话名） | — | 读屏文本 = 「导出「{name}」的会话对话。不包含原始 Provider 请求。」 | 已实现（映射 = G-G1-02→internal/ui/export.go:App.exportPanel；标记 = Role） |
| R-GF-014 | G-G1-02 | 内容 | 会话名超长 | 名字 truncate，句子完整 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-G1-02→internal/ui/export.go:App.exportPanel；标记 = 文案「导出面板内容」等 9 条） |
| R-GF-015 | G-G1-02 | 异常 | 描述为空 | 不允许：需说明导出范围 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-G1-02→internal/ui/export.go:App.exportPanel；标记 = Reason） |
| R-GF-016 | G-G1-02 | 内容 | 会话名含引号 | 名字原样嵌入「」内，引号不嵌套出错 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-G1-02→internal/ui/export.go:App.exportPanel；标记 = 文案「导出面板内容」等 9 条） |
| R-GF-017 | G-G1-02 | 窄窗 | 窗口收窄到最窄宽度时 | 描述最多 3 行后截断，弹窗不横向溢出 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-G1-02→internal/ui/export.go:App.exportPanel |
| R-GF-018 | G-G1-03 | 内容 | 未观察（条件）：弹窗打开 | 分组标签逐字「导出内容」（14px 中等） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/export.go:exportDescContent「导出内容」 |
| R-GF-019 | G-G1-03 | 关联 | 标签与三项勾选上下排列时 | 标签在上、三项勾选在下，左缘对齐 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM） | 已实现（映射 = G-G1-03→internal/ui/export.go:App.exportPanel；标记 = Screen） |
| R-GF-020 | G-G1-03 | 样式 | 分组标签在屏上时 | 14px 中等字重主前景 | — | 判据 = 字号 说明文字 10.5px；字重 常规 400 / 中粗 500 / 半粗 600 / 粗 700 | 已实现（映射 = G-G1-03→internal/ui/export.go:App.exportContentOptions；标记 = Height） |
| R-GF-021 | G-G1-03 | 无障碍 | 读屏进入该元素时 | 作为分组读出，其后三项归入该组 | — | 读屏文本 = 「导出内容」 | 已实现（映射 = G-G1-03→internal/ui/export.go:App.exportContent；标记 = Label） |
| R-GF-022 | G-G1-03 | 异常 | 三项全部关闭 | 允许导出（导出为纯消息），不强制勾选 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-G1-03→internal/ui/export.go:App.exportContent；标记 = err） |
| R-GF-023 | G-G1-03 | 内容 | 标签在屏上时 | 恒为「导出内容」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/export.go:exportDescContent「导出内容」 |
| R-GF-024 | G-G1-03 | 状态 | 三项全不勾 | 标签仍在（分组不因全空消失） | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 已实现（映射 = G-G1-03→internal/ui/export.go:App.exportPanel；标记 = Disabled） |
| R-GF-025 | G-G1-03 | 无障碍 | 读屏进入该元素时 | 作为分组标题读出 | — | 读屏文本 = 「导出内容」 | 已实现（映射 = G-G1-03→internal/ui/export.go:App.exportContent；标记 = Label） |
| R-GF-026 | G-G1-04 | 内容 | 未观察（条件）：弹窗打开 | 三个勾选项行：逐字「压缩记录」「工具详情」「思考」，方框 + 14px 文字 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/export.go:exportOptCompress「压缩记录」 |
| R-GF-027 | G-G1-04 | 状态 | 默认值 | 首屏读代码为未勾选；默认值须与实现一致并在此固化 | — | 判据 = 选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED） | 已实现（映射 = G-G1-04→internal/ui/export.go:App.exportPanel；标记 = Disabled） |
| R-GF-028 | G-G1-04 | 交互 | 鼠标点击该行时 | 切换勾选；点方框或文字都生效 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-G1-04→internal/ui/export.go:App.exportPanel；标记 = Key） |
| R-GF-029 | G-G1-04 | 键盘 | 按 Space 时 | 切换勾选 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-G1-04→internal/ui/export.go:App.exportPanel；标记 = Shortcut） |
| R-GF-030 | G-G1-04 | 关联 | 勾选「压缩记录」后 | 勾选「压缩记录」后出现说明行 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/export.go:exportOptCompress「压缩记录」 |
| R-GF-031 | G-G1-04 | 样式 | 勾选项 | 方框 14px 圆角 2；选中填主色 + 白勾；行高 28 | — | 判据 = 底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d；行高 空态主标题 28px；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-G1-04→internal/ui/export.go:App.exportContentOptions；标记 = Height） |
| R-GF-032 | G-G1-04 | 无障碍 | 读屏进入该元素时 | 逐项读作复选项，含勾选态 | — | 读屏文本 = 「压缩记录」 | 已实现（映射 = G-G1-04→internal/ui/export.go:App.exportContentOptions；标记 = Label） |
| R-GF-033 | G-G1-04 | 记忆 | 关闭后再次打开时 | 勾选状态按上次保留（同一次应用运行内）或回默认（按实现一致，须固定） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-G1-04→internal/ui/export.go:App.exportContentOptions |
| R-GF-034 | G-G1-05 | 内容 | 未观察（条件）：弹窗打开 | 第二分组标签逐字「消息」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/composer.go:App.composerCard「消息」 |
| R-GF-035 | G-G1-05 | 关联 | 两组标签纵向排列时 | 两组标签同字号，纵向排列（导出内容在上、消息在下） | — | 判据 = 字号 说明文字 10.5px；字重 常规 400 / 中粗 500 / 半粗 600 / 粗 700 | 已实现（映射 = G-G1-05→internal/ui/export.go:App.exportPanel；标记 = Screen） |
| R-GF-036 | G-G1-05 | 样式 | 两组标签同屏时 | 同款 14px 中等字重 | — | 判据 = 字号 说明文字 10.5px；字重 常规 400 / 中粗 500 / 半粗 600 / 粗 700 | 已实现（映射 = G-G1-05→internal/ui/export.go:App.exportPanel；标记 = Width） |
| R-GF-037 | G-G1-05 | 无障碍 | 读屏进入该元素时 | 作为分组读出 | — | 读屏文本 = 「消息」 | 已实现（映射 = G-G1-05→internal/ui/export.go:App.exportPanel；标记 = Role） |
| R-GF-038 | G-G1-05 | 异常 | 无回合（空会话） | 仍显示该标签，其下给不可用状态（G-G1-09） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-G1-05→internal/ui/export.go:App.exportPanel；标记 = Reason） |
| R-GF-039 | G-G1-05 | 内容 | 标签在屏上时 | 恒为「消息」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/composer.go:App.composerCard「消息」 |
| R-GF-040 | G-G1-05 | 状态 | 无回合 | 标签仍在，其下给状态文字 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-G1-05→internal/ui/export.go:App.exportPanel；标记 = Disabled） |
| R-GF-041 | G-G1-05 | 无障碍 | 读屏进入该元素时 | 作为分组标题读出 | — | 读屏文本 = 「消息」 | 已实现（映射 = G-G1-05→internal/ui/export.go:App.exportPanel；标记 = Role） |
| R-GF-042 | G-G1-06 | 内容 | 未观察（条件）：回合大纲已载入 | 计数文字逐字「保留 {kept} / {total} 个回合」 | — | 屏上形如「保留 12 / 20 个回合」 | 已实现（映射 = G-G1-06→internal/ui/export.go:App.exportTurnPicker；标记 = 文案「保留 %d / %d 个回合」等 6 条） |
| R-GF-043 | G-G1-06 | 状态 | 大纲未载入 | 该行不出现（不留 0/0 假值） | — | 屏上 计数文字 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GF-044 | G-G1-06 | 关联 | 勾选或排除回合后 | 勾选/排除后 kept 实时变化，total 不变 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-G1-06→internal/ui/export.go:App.exportTurnPicker；标记 = Select@screenHint） |
| R-GF-045 | G-G1-06 | 样式 | 计数 | 12px 次要色、等宽数字 | — | 判据 = 字号 UI 正文 12.25px；其余 = 字号 12.25 / 行高 18 / 字重 400，说明文字 10.5 / 14（TX-TYPE-BODY / TX-TYPE-MICRO）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-G1-06→internal/ui/export.go:App.exportTurnPicker；标记 = TextColor） |
| R-GF-046 | G-G1-06 | 无障碍 | 读屏进入该元素时 | 读出「保留 12 个，共 20 个回合」 | — | 读屏文本 = 「保留 12 个，共 20 个回合」 | 已实现（映射 = G-G1-06→internal/ui/export.go:App.exportTurnPicker；标记 = Label） |
| R-GF-047 | G-G1-06 | 异常 | kept 为 0 | 显示「保留 0 / 20 个回合」，且「导出」禁用 | — | 计数文字 可点状态 = 禁用（保留 0 时） | 已实现（逐字）：internal/ui/command_palette.go:App.commandPaletteActions「导出」 |
| R-GF-048 | G-G1-06 | 内容 | kept 变化 | 数字随勾选实时更新，total 不变 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-G1-06→internal/ui/export.go:App.exportTurnPicker；标记 = 文案「保留 %d / %d 个回合」等 6 条） |
| R-GF-049 | G-G1-06 | 无障碍 | 读屏进入该元素时 | 读出「保留 N 个，共 M 个回合」 | — | 读屏文本 = 「保留 N 个，共 M 个回合」 | 已实现（映射 = G-G1-06→internal/ui/export.go:App.exportTurnPicker；标记 = Label） |
| R-GF-050 | G-G1-07 | 状态 | 未观察（条件）：大纲已载入 | 搜索输入框在场，左置放大镜图标，占位逐字「搜索回合」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/export.go:exportSearchTurns「搜索回合」 |
| R-GF-051 | G-G1-07 | 交互 | 在输入框键入时 | 实时过滤回合条目；匹配数为 0 时给空态（G-G1-09 末支） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-G1-07→internal/ui/export.go:App.exportTurnPicker；标记 = Clicked@exportTextButton） |
| R-GF-052 | G-G1-07 | 关联 | 搜索结果非空与为空时分别打开 | 有匹配时「排除 N 个匹配」可用；0 匹配时禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-G1-07→internal/ui/export.go:App.exportTurnPicker；标记 = Select@screenHint） |
| R-GF-053 | G-G1-07 | 样式 | 输入框在屏上时 | 无边框或浅描边、圆角 6、高 32、带图标 | — | 判据 = 控件尺寸 输入框高 32（行内 25）；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px；其余 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0） | 已实现（映射 = G-G1-07→internal/ui/export.go:App.exportTurnPicker；标记 = TextColor） |
| R-GF-054 | G-G1-07 | 无障碍 | 读屏进入该元素时 | 读作搜索框，可读名「搜索回合」 | — | 读屏文本 = 「搜索回合」 | 已实现（逐字）：internal/ui/export.go:exportSearchTurns「搜索回合」 |
| R-GF-055 | G-G1-07 | 内容 | 大纲已载入、搜索输入框为空时 | 恒为「搜索回合」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/export.go:exportSearchTurns「搜索回合」 |
| R-GF-056 | G-G1-07 | 状态 | 输入为空 | 列表全量显示（不隐藏任何回合） | — | 屏上 任何回合 计数 = 0 | 已实现（映射 = G-G1-07→internal/ui/export.go:App.exportTurnPicker；标记 = Hidden@iconGlyph） |
| R-GF-057 | G-G1-07 | 无障碍 | 读屏进入该元素时 | 读作搜索框，可读名「搜索回合」 | — | 读屏文本 = 「搜索回合」 | 已实现（逐字）：internal/ui/export.go:exportSearchTurns「搜索回合」 |
| R-GF-058 | G-G1-08 | 状态 | 未观察（条件）：搜索有匹配 | 次级小按钮逐字「排除 {count} 个匹配」 | — | 屏上形如「排除 3 个匹配」 | 已实现（映射 = G-G1-08→internal/ui/export.go:App.exportTurnPicker；标记 = Hidden@iconGlyph） |
| R-GF-059 | G-G1-08 | 状态 | 匹配为 0 | 按钮禁用（文案保留或显示「排除 0 个匹配」） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-G1-08→internal/ui/export.go:App.exportTurnPicker；标记 = Hidden@iconGlyph） |
| R-GF-060 | G-G1-08 | 交互 | 鼠标点击该元素时 | 把匹配回合全部取消勾选；kept 同步减少 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-G1-08→internal/ui/export.go:App.exportTurnPicker；标记 = Clicked@exportTextButton） |
| R-GF-061 | G-G1-08 | 关联 | 按搜索批量排除与全部排除两个入口同屏时 | 一个是按搜索批量排除、一个是全部排除，两个入口不合并 | — | 屏上 次级小按钮 计数 = 1 | 已实现（映射 = G-G1-08→internal/ui/export.go:App.exportTurnPicker；标记 = Select@screenHint） |
| R-GF-062 | G-G1-08 | 样式 | 次级按钮在屏上时 | `outline`、高 28、12–13px | — | 判据 = 控件尺寸 按钮高 次 28；描边 常规 1px（卡片外框 0.5px / 焦点环 3px） | 已实现（映射 = G-G1-08→internal/ui/export.go:App.exportTurnPicker；标记 = TextColor） |
| R-GF-063 | G-G1-08 | 无障碍 | 读屏进入该元素时 | 读作按钮「排除 N 个匹配」 | — | 读屏文本 = 「排除 N 个匹配」 | 已实现（映射 = G-G1-08→internal/ui/export.go:App.exportTurnPicker；标记 = Label） |
| R-GF-064 | G-G1-08 | 异常 | 已排除后又搜索 | 按钮按当前匹配重新计数，不回退已排除项 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-G1-08→internal/ui/export.go:App.exportTurnPicker |
| R-GF-065 | G-G1-08 | 内容 | count 为 0 | 显示「排除 0 个匹配」并禁用，或隐藏（二选一固定） | — | 屏上 次级小按钮 计数 = 0 | 已实现（映射 = G-G1-08→internal/ui/export.go:App.exportTurnPicker；标记 = 文案「保留 %d / %d 个回合」等 6 条） |
| R-GF-066 | G-G1-09 | 内容 | 未观察（条件）：大纲加载中 | 状态文字逐字「正在读取对话…」，居中、上下 24px | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-G1-09→internal/ui/export.go:App.exportTurnPicker；标记 = 文案「保留 %d / %d 个回合」等 6 条） |
| R-GF-067 | G-G1-09 | 内容 | 加载失败 | 文字逐字「读取会话对话失败」，危险色 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-G1-09→internal/ui/export.go:App.exportTurnPicker；标记 = 文案「保留 %d / %d 个回合」等 6 条） |
| R-GF-068 | G-G1-09 | 内容 | 对话不可按回合拆分 | 文字逐字「该对话无法按回合拆分，因此无法过滤。导出文件仅包含对话头信息。」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-G1-09→internal/ui/export.go:App.exportTurnPicker；标记 = 文案「保留 %d / %d 个回合」等 6 条） |
| R-GF-069 | G-G1-09 | 内容 | 搜索无匹配 | 文字逐字「没有回合匹配此搜索。」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/ui/export.go:exportNoMatchText「没有回合匹配此搜索。」 |
| R-GF-070 | G-G1-09 | 状态 | 四分支互斥 | 同一时刻只有一句，按条件优先级 | — | 文案计数为 1 | 已实现（映射 = G-G1-09→internal/ui/export.go:App.exportTurnPicker；标记 = Hidden@iconGlyph） |
| R-GF-071 | G-G1-09 | 关联 | 加载中与不可拆分时分别打开 | 加载中与不可拆分时「导出」的可用性不同（前者禁用、后者可导出头信息） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/ui/command_palette.go:App.commandPaletteActions「导出」 |
| R-GF-072 | G-G1-09 | 无障碍 | 读屏进入该元素时 | 读出当前状态句 | — | 读屏文本 = 「正在读取对话…」 | 已实现（映射 = G-G1-09→internal/ui/export.go:App.exportTurnPicker；标记 = Label） |
| R-GF-073 | G-G1-09 | 内容 | 不可拆分时的按钮状态 | 「导出」可用（导出头信息），搜索框隐藏 | — | 屏上 状态文字 计数 = 0 | 已实现（逐字）：internal/ui/command_palette.go:App.commandPaletteActions「导出」 |
| R-GF-074 | G-G1-09 | 样式 | 状态文字 | 居中、14px 次要色、上下 24px | — | 判据 = 字号 正文 14px；内边距 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28）/ 卡片外框 4.5；档位 3.5·7·10.5·14·21·28px；其余 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-G1-09→internal/ui/export.go:App.exportTurnPicker；标记 = TextColor） |
| R-GF-075 | G-G1-10 | 状态 | 未观察（条件）：大纲返回且含回合 | 每个回合一行：方框 + 展开箭头 + 序号 + 预览 + 回复摘要「回复 · {tools} 个工具 · {chars}k」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-G1-10→internal/ui/export.go:App.exportTurnRow；标记 = Disabled@accIconButton） |
| R-GF-076 | G-G1-10 | 状态 | 用默认配置打开时 | 全部回合默认勾选 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-G1-10→internal/ui/export.go:App.exportTurnRow；标记 = Disabled@accIconButton） |
| R-GF-077 | G-G1-10 | 交互 | 点方框 | 切换该回合勾选；kept 同步 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-G1-10→internal/ui/export.go:App.exportTurnRow；标记 = Pressed@accIconButton） |
| R-GF-078 | G-G1-10 | 交互 | 点箭头 | 展开/收起该回合正文（G-G1-11）；勾选状态不变 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-G1-10→internal/ui/export.go:App.exportTurnRow；标记 = Pressed@accIconButton） |
| R-GF-079 | G-G1-10 | 内容 | 预览超长 | 预览单行 truncate；序号与摘要不被裁 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-G1-10→internal/ui/export.go:App.exportTurnRow；标记 = 文案「展开回合正文」等 5 条） |
| R-GF-080 | G-G1-10 | 内容 | 回复摘要缺数据 | 摘要段整体省略（不显示「回复 · 0 个工具」假值） | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GF-081 | G-G1-10 | 样式 | 行在屏上时 | 高约 32；方框 14px；摘要 12px 次要色右对齐 | — | 判据 = 行高 UI 正文 17.5px；字号 UI 正文 12.25px；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-G1-10→internal/ui/export.go:App.exportTurnRow；标记 = Height） |
| R-GF-082 | G-G1-10 | 无障碍 | 读屏进入该元素时 | 读作「第 N 回合，预览…，回复摘要…，已勾选」 | — | 读屏文本 = 「第 N 回合，预览…，回复摘要…，已勾选」 | 已实现（映射 = G-G1-10→internal/ui/export.go:App.exportTurnRow；标记 = Label@accIconButton） |
| R-GF-083 | G-G1-11 | 状态 | 未观察（条件）：该回合已展开 | 回合正文以等宽预格式块显示，最大高 256，超出滚动 | — | 块高 ≤256 | 已实现（映射 = G-G1-11→internal/ui/export.go:App.exportTurnRow；标记 = Disabled@accIconButton） |
| R-GF-084 | G-G1-11 | 内容 | 正在取文 | 块内显示「加载中…」（中文省略号） | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-G1-11→internal/ui/export.go:App.exportTurnBodyBlock；标记 = 文案「回合正文」） |
| R-GF-085 | G-G1-11 | 内容 | 正文含超长行 | 横向滚动，不折行（预格式） | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-G1-11→internal/ui/export.go:App.exportTurnBodyBlock；标记 = 文案「回合正文」） |
| R-GF-086 | G-G1-11 | 交互 | 选中屏上文本后复制时 | 可复制正文（含格式空白） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-G1-11→internal/ui/export.go:App.exportTurnRow；标记 = Pressed@accIconButton） |
| R-GF-087 | G-G1-11 | 样式 | 正文块 | 等宽 12px、浅底、圆角 6、内边距 8 | — | 判据 = 底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px；其余 = 字号 12.25 / 行高 18 / 字重 400，说明文字 10.5 / 14（TX-TYPE-BODY / TX-TYPE-MICRO）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-G1-11→internal/ui/export.go:App.exportTurnBodyBlock；标记 = Height） |
| R-GF-088 | G-G1-11 | 无障碍 | 读屏进入该元素时 | 读出正文文本 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-G1-11→internal/ui/export.go:App.exportTurnBodyBlock；标记 = Label） |
| R-GF-089 | G-G1-11 | 内容 | 正文为空 | 块内显示空（不留「加载中…」） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-G1-11→internal/ui/export.go:App.exportTurnBodyBlock；标记 = 文案「回合正文」） |
| R-GF-090 | G-G1-11 | 状态 | 展开多个回合 | 各自独立展开，互不影响 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-G1-11→internal/ui/export.go:App.exportTurnRow；标记 = Disabled@accIconButton） |
| R-GF-091 | G-G1-12 | 状态 | 未观察（条件）：大纲含回合 | 文字按钮「全部保留」/「全部排除」（12px 灰） | — | 判据 = 字号 UI 正文 12.25px；字重 常规 400 / 中粗 500 / 半粗 600 / 粗 700 | 已实现（逐字）：internal/ui/export.go:App.exportTurnPicker「全部保留」 |
| R-GF-092 | G-G1-12 | 交互 | 点「全部保留」 | 所有回合勾选，kept = total | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-G1-12→internal/ui/export.go:App.exportTurnPicker；标记 = Clicked@exportTextButton） |
| R-GF-093 | G-G1-12 | 交互 | 点「全部排除」 | 所有回合取消勾选，kept = 0；此时「导出」禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/ui/command_palette.go:App.commandPaletteActions「导出」 |
| R-GF-094 | G-G1-12 | 状态 | 某状态已达 | 对应按钮可置灰（如已全选则「全部保留」禁用） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/ui/export.go:App.exportTurnPicker「全部保留」 |
| R-GF-095 | G-G1-12 | 样式 | 文字按钮在屏上时 | 12px 次要色、无边框；两按钮间距 8 | — | 判据 = 字号 UI 正文 12.25px；间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px | 已实现（映射 = G-G1-12→internal/ui/export.go:App.exportTurnPicker；标记 = TextColor） |
| R-GF-096 | G-G1-12 | 无障碍 | 读屏进入该元素时 | 读作按钮「全部保留」/「全部排除」 | — | 读屏文本 = 「全部保留」 / 「全部排除」 | 已实现（逐字）：internal/ui/export.go:App.exportTurnPicker「全部保留」 |
| R-GF-097 | G-G1-12 | 内容 | 文案在屏上时 | 恒为「全部保留」「全部排除」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/export.go:App.exportTurnPicker「全部保留」 |
| R-GF-098 | G-G1-12 | 关联 | 点「全部排除」后 | 点后 kept 立即变为 total 或 0 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-G1-12→internal/ui/export.go:App.exportTurnPicker；标记 = Select@screenHint） |
| R-GF-099 | G-G1-13 | 内容 | 未观察（条件）：勾选「压缩记录」且存在压缩记录 | 说明文字逐字「压缩记录会单独描述被折叠的回合，排除某个回合不会将其从压缩记录中移除。」 | — | 屏上 说明文字 计数 = 0 | 已实现（逐字）：internal/ui/export.go:exportCompressNote「压缩记录会单独描述被折叠的回合，排除某个回合不会将其从压缩记录中移除。」 |
| R-GF-100 | G-G1-13 | 状态 | 未勾选或无压缩记录 | 该说明不出现 | — | 屏上 说明文字 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GF-101 | G-G1-13 | 样式 | 说明段在屏上时 | 12px 次要色、折行、位于勾选项之下 | — | 判据 = 字号 UI 正文 12.25px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-G1-13→internal/ui/export.go:App.exportContentOptions；标记 = Height） |
| R-GF-102 | G-G1-13 | 无障碍 | 读屏进入该元素时 | 读出整句 | — | 读屏文本 = 「压缩记录会单独描述被折叠的回合，排除某个回合不会」 | 已实现（映射 = G-G1-13→internal/ui/export.go:App.exportContentOptions；标记 = Label） |
| R-GF-103 | G-G1-13 | 关联 | 排除回合说明与其他说明同屏时 | 排除回合后说明仍在（提示用户压缩记录不受影响） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-G1-13→internal/ui/export.go:App.exportPanel；标记 = Screen） |
| R-GF-104 | G-G1-13 | 内容 | 文案在屏上时 | 两句：单独描述被折叠回合 + 排除不影响压缩记录 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-G1-13→internal/ui/export.go:App.exportContentOptions；标记 = 文案「导出内容」等 5 条） |
| R-GF-105 | G-G1-13 | 状态 | 无压缩记录 | 不显示（勾选也不显示） | — | 屏上 说明文字 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GF-106 | G-G1-13 | 样式 | 说明段在屏上时 | 12px 次要色、折行、位于勾选项下 | — | 判据 = 字号 UI 正文 12.25px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-G1-13→internal/ui/export.go:App.exportContentOptions；标记 = Height） |
| R-GF-107 | G-G1-14 | 内容 | 未观察（条件）：导出时会话快照过期并重载 | 提示逐字「会话快照已过期并重新加载。请检查消息和排除项后再次导出。」，`role=status` | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/export.go:exportStaleNotice「会话快照已过期并重新加载。请检查消息和排除项后再次导出。」 |
| R-GF-108 | G-G1-14 | 交互 | 读到提示后 | 不自动导出；等用户再点「导出」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/command_palette.go:App.commandPaletteActions「导出」 |
| R-GF-109 | G-G1-14 | 样式 | 提示在屏上时 | 14px 级、`role=status`、醒目但不刺眼 | — | 判据 = 字号 正文 14px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-G1-14→internal/ui/export.go:App.exportPanel；标记 = Width） |
| R-GF-110 | G-G1-14 | 无障碍 | 读屏进入该元素时 | 播报一次 | — | 播报次数 = 1 | 已实现（映射 = G-G1-14→internal/ui/export.go:App.exportPanel；标记 = Role） |
| R-GF-111 | G-G1-14 | 异常 | 连续两次过期 | 提示仍 1 条，不叠加 | — | 提示恒 1 | 已实现（映射 = G-G1-14→internal/ui/export.go:App.exportPanel；标记 = Reason） |
| R-GF-112 | G-G1-14 | 内容 | 文案在屏上时 | 三句：过期、重载、请检查后再导出 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-G1-14→internal/ui/export.go:App.exportPanel；标记 = 文案「导出面板内容」等 9 条） |
| R-GF-113 | G-G1-14 | 状态 | 未过期 | 不显示 | — | 屏上 提示文字 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GF-114 | G-G1-14 | 样式 | 提示在屏上时 | 14px、`role=status`、无底色 | — | 判据 = 字号 说明文字 10.5px | 已实现（映射 = G-G1-14→internal/ui/export.go:App.exportPanel；标记 = Width） |
| R-GF-115 | G-G1-15 | 内容 | 未观察（条件）：点「导出」后失败 | 错误文字显示失败原因 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-G1-15→internal/ui/export.go:App.exportPanel；标记 = 文案「导出面板内容」等 9 条） |
| R-GF-116 | G-G1-15 | 交互 | 「重试」在屏上时 | 先清错误再导出 | — | 错误段恒 1 | 已实现（映射 = G-G1-15→internal/ui/export.go:App.exportPanel；标记 = Key） |
| R-GF-117 | G-G1-15 | 样式 | 错误段在屏上时 | 14px 危险色、按钮行之上 | — | 判据 = 字号 正文 14px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-G1-15→internal/ui/export.go:App.exportPanel；标记 = Width） |
| R-GF-118 | G-G1-15 | 无障碍 | 读屏进入该元素时 | `role=alert` 播报一次 | — | 播报次数 = 1 | 已实现（映射 = G-G1-15→internal/ui/export.go:App.exportPanel；标记 = Role） |
| R-GF-119 | G-G1-15 | 内容 | 原因 | 服务端原文，不翻译 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-G1-15→internal/ui/export.go:App.exportPanel；标记 = 文案「导出面板内容」等 9 条） |
| R-GF-120 | G-G1-15 | 交互 | 鼠标点击错误文字时 | 无动作 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-G1-15→internal/ui/export.go:App.exportPanel；标记 = Key） |
| R-GF-121 | G-G1-15 | 样式 | 错误段在屏上时 | 14px 危险色、按钮行之上 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-G1-15→internal/ui/export.go:App.exportPanel；标记 = Width） |
| R-GF-122 | G-G1-15 | 无障碍 | 读屏进入该元素时 | `role=alert` 播报失败原因 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；色值取 token：主色 #349651/#63cb7b、正文 #0a0a0a/#ececec、次文 #737373/#8a8a8a、边框 #e5e5e5/#2e2e2e、警示 #e17100/#ffb900，偏差 0（CL-ACCENT / CL-TEXT / CL-TEXT-2 / CL-BORDER / CL-WARN） | 已实现（映射 = G-G1-15→internal/ui/export.go:App.exportPanel；标记 = Role） |
| R-GF-123 | G-G1-15 | 状态 | 导出重试后 | 旧错误先清除再显示新错误 | — | 错误恒 1 条 | 已实现（映射 = G-G1-15→internal/ui/export.go:App.exportPanel；标记 = Disabled） |
| R-GF-124 | G-G1-16 | 状态 | 未观察（条件）：弹窗在场 | 次按钮「取消」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GF-125 | G-G1-16 | 交互 | 鼠标点击该元素时 | 关闭，不写文件；勾选状态按记忆规则保留 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-G1-16→internal/ui/export.go:App.exportPanel；标记 = Key） |
| R-GF-126 | G-G1-16 | 键盘 | 按 Esc 时 | 等同点击 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-G1-16→internal/ui/export.go:App.exportPanel；标记 = Shortcut） |
| R-GF-127 | G-G1-16 | 样式 | 次按钮在屏上时 | 描边、高 28 | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-G1-16→internal/ui/export.go:App.exportPanel；标记 = Width） |
| R-GF-128 | G-G1-16 | 无障碍 | 读屏进入该元素时 | 读作按钮「取消」 | — | 读屏文本 = 「取消」 | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GF-129 | G-G1-16 | 内容 | 文案在屏上时 | 恒为「取消」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GF-130 | G-G1-16 | 状态 | 导出中 | 禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-G1-16→internal/ui/export.go:App.exportPanel；标记 = Disabled） |
| R-GF-131 | G-G1-16 | 无障碍 | 读屏进入该元素时 | 读作按钮「取消」 | — | 读屏文本 = 「取消」 | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GF-132 | G-G1-17 | 状态 | 未观察（条件）：大纲已载入且 kept>0 | 主按钮「导出」可用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/ui/command_palette.go:App.commandPaletteActions「导出」 |
| R-GF-133 | G-G1-17 | 状态 | 未取到大纲 / kept=0 / 导出中 | 按钮禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-G1-17→internal/ui/export.go:App.exportPanel；标记 = Disabled） |
| R-GF-134 | G-G1-17 | 内容 | 提交进行中时 | 文案变「导出中…」（中文省略号），禁用 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-G1-17→internal/ui/export.go:App.exportPanel；标记 = 文案「导出面板内容」等 9 条） |
| R-GF-135 | G-G1-17 | 交互 | 鼠标点击该元素时 | 打开保存对话框并写文件；成功后关闭弹窗 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-G1-17→internal/ui/export.go:App.exportPanel；标记 = Key） |
| R-GF-136 | G-G1-17 | 异常 | 写文件失败 | 弹窗保留并显示原因（G-G1-15） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-G1-17→internal/ui/export.go:App.exportPanel；标记 = Reason） |
| R-GF-137 | G-G1-17 | 样式 | 主按钮在屏上时 | 实心主色、白字、高 28；禁用降透明 | — | 判据 = 底色 主色 明 #339751 / 暗 #63cb7b；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-G1-17→internal/ui/export.go:App.exportPanel；标记 = Width） |
| R-GF-138 | G-G1-17 | 无障碍 | 读屏进入该元素时 | 读作按钮「导出」，禁用时播报原因 | — | 读屏文本 = 「导出」 | 已实现（逐字）：internal/ui/command_palette.go:App.commandPaletteActions「导出」 |
| R-GF-139 | G-G1-17 | 内容 | 未取到大纲 | 按钮禁用并保持文案「导出」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/ui/command_palette.go:App.commandPaletteActions「导出」 |
| R-GF-140 | G-G1-17 | 内容 | kept=0 | 按钮禁用（无内容可导） | — | 主按钮 可点状态 = 禁用（保留 0 时） | 已实现（映射 = G-G1-17→internal/ui/export.go:App.exportPanel；标记 = 文案「导出面板内容」等 9 条） |
| R-GF-141 | G-G1-17 | 异常 | 保存对话框取消 | 回到弹窗、按钮恢复，不报错 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |

### 7.2 Fork 会话（G-G2-01–G-G2-05）

条件：会话已在另一个窗口打开时尝试打开它。入口已采（C13「在新窗口中打开」157,324）。

| 需求 ID | 元素 ID | 维度 | 场景 | 期望 | 异常 | 验收 | 状态 |
|---|---|---|---|---|---|---|---|
| R-GF-142 | G-G2-01 | 内容 | 未观察（条件）：会话已在另一窗口打开时再打开它 | 标题逐字「会话正在使用中」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GF-143 | G-G2-01 | 状态 | 会话未被占用 | 不弹此框（直接打开） | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（找不到元素渲染点） |
| R-GF-144 | G-G2-01 | 样式 | 标题在屏上时 | 16px 级半粗；圆角 + 阴影 + 遮罩 | — | 判据 = 字号 对话框标题 15.75px；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px | 未实现（找不到元素渲染点） |
| R-GF-145 | G-G2-01 | 无障碍 | 读屏进入该元素时 | 作为对话框名读出 | — | 读屏文本 = 「会话正在使用中」 | 未实现（找不到元素渲染点） |
| R-GF-146 | G-G2-01 | 异常 | 占用窗口在弹窗期间关闭 | 确认时按实况改走「直接打开」，不报错 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GF-147 | G-G2-01 | 内容 | 标题在屏上时 | 恒为「会话正在使用中」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GF-148 | G-G2-01 | 状态 | 同一会话在本窗口已打开 | 不弹此框（本窗口可用） | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（找不到元素渲染点） |
| R-GF-149 | G-G2-01 | 无障碍 | 读屏进入该元素时 | 作为对话框名读出 | — | 读屏文本 = 「会话正在使用中」 | 未实现（找不到元素渲染点） |
| R-GF-150 | G-G2-02 | 内容 | 未观察（条件）：会话已在其他窗口打开时 | 描述逐字「“{title}”当前正在 {heldBy} 中打开。」；`{heldBy}` 为持有窗口的标识 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GF-151 | G-G2-02 | 内容 | 持有者未知 | 退为「另一窗口」，不留空 `{heldBy}` | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GF-152 | G-G2-02 | 样式 | 描述在屏上时 | 14px 级次要色、折行 | — | 判据 = 字号 正文 14px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 未实现（找不到元素渲染点） |
| R-GF-153 | G-G2-02 | 无障碍 | 读屏进入该元素时 | 读出整句 | — | 读屏文本 = 「“{title}”当前正在 {heldBy} 中」 | 未实现（找不到元素渲染点） |
| R-GF-154 | G-G2-02 | 内容 | 会话名超长 | 名字 truncate，句子完整 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（找不到元素渲染点） |
| R-GF-155 | G-G2-02 | 内容 | 持有窗口标识为路径/标题 | 原样嵌入 `{heldBy}` | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GF-156 | G-G2-02 | 样式 | 描述在屏上时 | 14px 次要色、折行 | — | 判据 = 字号 正文 14px | 未实现（找不到元素渲染点） |
| R-GF-157 | G-G2-02 | 异常 | 描述为空 | 不允许 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（找不到元素渲染点） |
| R-GF-158 | G-G2-02 | 无障碍 | 读屏进入该元素时 | 读出整句（含持有者） | — | 读屏文本 = 「“{title}”当前正在 {heldBy} 中」 | 未实现（找不到元素渲染点） |
| R-GF-159 | G-G2-02 | 窄窗 | 窗口收窄到最窄宽度时 | 描述折行，按钮行完整 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 未实现（找不到元素渲染点） |
| R-GF-160 | G-G2-03 | 内容 | 未观察（条件）：Fork 确认框打开时 | 说明逐字「Fork 将创建一个包含完整消息记录的独立副本，原会话不受影响。」；「Fork」保持原文大写 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GF-161 | G-G2-03 | 样式 | 说明段在屏上时 | 14px 级次要色、行高 1.5 | — | 判据 = 字号 正文 14px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 未实现（找不到元素渲染点） |
| R-GF-162 | G-G2-03 | 无障碍 | 读屏进入该元素时 | 读出整句 | — | 读屏文本 = 「Fork 将创建一个包含完整消息记录的独立副本，」 | 未实现（找不到元素渲染点） |
| R-GF-163 | G-G2-03 | 异常 | 说明超长（本地化后） | 折行，按钮行仍可见 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 未实现（找不到元素渲染点） |
| R-GF-164 | G-G2-03 | 内容 | 文案在屏上时 | 两句：创建独立副本 + 原会话不受影响 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GF-165 | G-G2-03 | 状态 | Fork 执行中时 | 说明保留（不换成进度文案） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GF-166 | G-G2-03 | 样式 | 说明段在屏上时 | 14px 次要色、行高 1.5 | — | 判据 = 字号 正文 14px | 未实现（找不到元素渲染点） |
| R-GF-167 | G-G2-03 | 异常 | 说明为空 | 不允许 | — | 判据 = 说明与失败文案 10.5/14、色 #737373 / #8a8a8a、贴触发控件下方 3.5（TX-TIP） | 未实现（找不到元素渲染点） |
| R-GF-168 | G-G2-04 | 状态 | 未观察（条件）：弹窗在场 | 次按钮「取消」（`outline`） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GF-169 | G-G2-04 | 交互 | 鼠标点击该元素时 | 关闭；不打开、不 Fork | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（找不到元素渲染点） |
| R-GF-170 | G-G2-04 | 键盘 | 按 Esc 时 | 等同点击 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 未实现（找不到元素渲染点） |
| R-GF-171 | G-G2-04 | 样式 | 次按钮在屏上时 | 描边、高 28 | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（找不到元素渲染点） |
| R-GF-172 | G-G2-04 | 无障碍 | 读屏进入该元素时 | 读作按钮「取消」 | — | 读屏文本 = 「取消」 | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GF-173 | G-G2-04 | 内容 | 文案在屏上时 | 恒为「取消」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GF-174 | G-G2-04 | 状态 | Fork 执行中时 | 禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（找不到元素渲染点） |
| R-GF-175 | G-G2-04 | 交互 | 点击后 | 原窗口不受影响，本窗口不打开该会话 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（找不到元素渲染点） |
| R-GF-176 | G-G2-05 | 状态 | 未观察（条件）：弹窗在场 | 主按钮逐字「Fork」，提交中前置旋转图标 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GF-177 | G-G2-05 | 交互 | 鼠标点击该元素时 | 创建独立副本并打开它；原会话不变 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（找不到元素渲染点） |
| R-GF-178 | G-G2-05 | 状态 | 提交进行中时 | 按钮禁用并显示旋转图标 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（找不到元素渲染点） |
| R-GF-179 | G-G2-05 | 异常 | 副本创建失败 | 弹窗保留并给失败原因 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GF-180 | G-G2-05 | 样式 | 主按钮在屏上时 | 实心主色、白字、高 28 | — | 判据 = 底色 主色 明 #339751 / 暗 #63cb7b；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（找不到元素渲染点） |
| R-GF-181 | G-G2-05 | 无障碍 | 读屏进入该元素时 | 读作按钮「Fork」，并说明会创建独立副本 | — | 读屏文本 = 「Fork」 | 未实现（找不到元素渲染点） |
| R-GF-182 | G-G2-05 | 键盘 | 按 Enter 时 | 等同点击 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 未实现（找不到元素渲染点） |
| R-GF-183 | G-G2-05 | 无障碍 | 读屏进入该元素时 | 读作按钮「Fork」 | — | 读屏文本 = 「Fork」 | 未实现（找不到元素渲染点） |

### 7.3 导出与 Fork 域维度覆盖说明

22 项全部 ≥6 维。G-G2-04（取消）到 6 维：文案恒定、无过渡；动画随弹窗整体。

### 7.4 本版实现口径（2026-11-19）

**导出**：标题栏 `⋯`、左栏会话行菜单与命令面板的「导出」现在打开「导出对话」面板
（`internal/ui/export.go:exportPanel`）。面板按本轮组织任务的口径实现：格式三档
（Markdown / JSON / 纯文本）、范围三档（整条会话 / 仅当前分支 / 选中的轮次）、
前 20 行预览、剪贴板与文件两条通路；剪贴板回执给字符数，文件回执给字节数（标题行
下方），失败原因留在面板里。文件通路走宿主保存面板（`Options.SaveFile`），宿主没给
面板时退回手输路径（`writeExportFile` 展开 `~` 并校验目录）。内容只从视图已有的快照
拼（`SessionSnapshot.Rows` 的 `BranchID` / `TurnIndex`，样例走 `App.rounds`），
不另读服务端。

**Fork**：本版只做「按轮切分支」：标题栏「在新分支继续」与用户消息的「从此处分叉」
走既有的 `Intents.CreateBranch`（以该轮结束处为切点），成功后按分支清单做差取新分支
并 `Intents.SelectBranch`，回执写切点（`internal/ui/fork.go`）。拿不到轮次、轮次未结束、
没有写通道时置灰并把原因写在菜单项 / 可读名 / 提示里。

**仍未实现**：§7.1 的「导出内容」三项勾选（压缩记录 / 工具详情 / 思考）、回合搜索与
「排除 N 个匹配」、快照过期提示；§7.2 的会话级 Fork 副本（复制完整消息记录）、
「会话正在使用中」占用框与 Fork 确认框——协议 v3 没有会话复制与占用查询的写入通道，
这几组仍是「未实现」，不拿按轮切分支冒充会话复制。

证据与用例：`internal/ui/export.go`、`internal/ui/fork.go`、`internal/ui/export_test.go`
（三种格式内容、范围选择、剪贴板 / 文件两条通路、回执与失败）、`internal/ui/fork_test.go`
（CreateBranch 实参、切到新分支、切点回执、不可用置灰）。


---

## 8. 导入与更新（57 元素）

七组：添加插件（H1，12）、市场安装（H2，9）、替换确认（H3，4）、插件可更新（H4，7）、技能可更新（H5，6）、导入 Chrome Cookie（H6，13）、清理浏览器数据（H7，6）。这些对话框在 60 份态里一份都没打开过（H6/H7 只有设置页入口的旁证），故全部按条件写。

> 实现状态（2026-11-20）：插件管理块落在设置 › 插件设置（`internal/ui/plugin_manage.go` + `plugin_install.go`）——本地 manifest 校验与预览、`plugin.install`、逐项 / 全部更新、`plugin.remove` 确认框、插件市场源配置（空源写「未配置插件源」，不画样例卡片）。市场安装弹窗（H2）、替换确认（H3）、技能更新（H5）、Chrome Cookie 与清理浏览器数据（H6/H7）仍未建。市场清单来自本地偏好里的上游地址（`internal/uiwire/plugin_marketplace.go`），协议没有市场安装来源，条目的「安装」置灰。仍缺：`plugin.list` 不返回 `UpdateAvailable` / `LatestVersion`，真实环境不会出现可更新项（界面已按这两个事实字段实现，测试用注入事实）；下载进度（字节数 / 1px 进度条）也没有协议事件。测试见 `internal/ui/plugin_manage_test.go`、`internal/uiwire/plugin_marketplace_test.go`、`uiwire/intents_test.go:TestIntentPluginInstallAndUpdate`。

### 8.1 添加插件（G-H1-01–G-H1-12）

| 需求 ID | 元素 ID | 维度 | 场景 | 期望 | 异常 | 验收 | 状态 |
|---|---|---|---|---|---|---|---|
| R-GI-001 | G-H1-01 | 内容 | 未观察（条件）：设置 → 插件 或 技能/插件页「插件」→ 添加插件 | 标题逐字「从 Git 添加插件」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/plugin_install.go:pluginDialogTitle「从 Git 添加插件」 |
| R-GI-002 | G-H1-01 | 状态 | 默认模式为「本地文件夹」 | 标题仍是「从 Git 添加插件」（不随模式改词） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/plugin_install.go:pluginDialogTitle「从 Git 添加插件」 |
| R-GI-003 | G-H1-01 | 样式 | 标题在屏上时 | 16px 级半粗；圆角 + 阴影 + 遮罩 | — | 判据 = 字号 对话框标题 15.75px；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px | 已实现（映射 = G-H1-01→internal/ui/plugin_install.go:App.pluginInstallView；标记 = TextColor） |
| R-GI-004 | G-H1-01 | 无障碍 | 读屏进入该元素时 | 作为对话框名读出 | — | 读屏文本 = 「从 Git 添加插件」 | 已实现（映射 = G-H1-01→internal/ui/plugin_install.go:App.pluginInstallView；标记 = Role） |
| R-GI-005 | G-H1-01 | 异常 | 插件目录不可写 | 弹窗仍可打开，提交时才报错（不提前拦） | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H1-01→internal/ui/plugin_install.go:App.pluginInstallView；标记 = err） |
| R-GI-006 | G-H1-01 | 窄窗 | 窗口收窄到最窄宽度时 | 标题单行，字段与按钮行完整 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-H1-01→internal/ui/plugin_install.go:App.pluginInstallView |
| R-GI-007 | G-H1-01 | 内容 | 标题在屏上时 | 恒为「从 Git 添加插件」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/plugin_install.go:pluginDialogTitle「从 Git 添加插件」 |
| R-GI-008 | G-H1-01 | 异常 | 插件目录不存在 | 仍可打开，提交时报错 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H1-01→internal/ui/plugin_install.go:App.pluginInstallView；标记 = err） |
| R-GI-009 | G-H1-02 | 内容 | 未观察（条件）：添加插件面板切到 Git 来源时 | 描述逐字「从 GitHub 仓库或 Git URL 添加。」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/plugin_install.go:pluginDialogDesc「从 GitHub 仓库或 Git URL 添加。」 |
| R-GI-010 | G-H1-02 | 状态 | 切到「本地文件夹」 | 描述不变（同一句话覆盖两种模式） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H1-02→internal/ui/plugin_install.go:App.pluginInstallView；标记 = Disabled） |
| R-GI-011 | G-H1-02 | 样式 | 描述在屏上时 | 14px 级次要色 | — | 判据 = 字号 正文 14px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-H1-02→internal/ui/plugin_install.go:App.pluginInstallView；标记 = TextColor） |
| R-GI-012 | G-H1-02 | 无障碍 | 读屏进入该元素时 | 作为描述读出 | — | 读屏文本 = 「从 GitHub 仓库或 Git URL 添加。」 | 已实现（映射 = G-H1-02→internal/ui/plugin_install.go:App.pluginInstallView；标记 = Role） |
| R-GI-013 | G-H1-02 | 异常 | 描述折行 | 不挤压模式切换行 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-H1-02→internal/ui/plugin_install.go:App.pluginInstallView；标记 = err） |
| R-GI-014 | G-H1-02 | 内容 | 描述在屏上时 | 恒为「从 GitHub 仓库或 Git URL 添加。」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/plugin_install.go:pluginDialogDesc「从 GitHub 仓库或 Git URL 添加。」 |
| R-GI-015 | G-H1-02 | 状态 | Git 模式 | 描述不变（同一句覆盖两模式） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H1-02→internal/ui/plugin_install.go:App.pluginInstallView；标记 = Disabled） |
| R-GI-016 | G-H1-02 | 样式 | 描述在屏上时 | 14px 次要色 | — | 判据 = 字号 正文 14px | 已实现（映射 = G-H1-02→internal/ui/plugin_install.go:App.pluginInstallView；标记 = TextColor） |
| R-GI-017 | G-H1-03 | 状态 | 未观察（条件）：弹窗打开 | 两个等宽小按钮「本地文件夹」「Git 仓库」，选中项为 `secondary` 底 | — | 判据 = 底色 选中底 明 #f5f5f5 / 暗 #262626；其余 = 选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED） | 已实现（逐字）：internal/ui/plugin_install.go:pluginModeLocal「本地文件夹」 |
| R-GI-018 | G-H1-03 | 状态 | 用默认配置打开时 | 选中「本地文件夹」（默认分支） | — | 判据 = 底色 选中底 明 #f5f5f5 / 暗 #262626；其余 = 选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED） | 已实现（逐字）：internal/ui/plugin_install.go:pluginModeLocal「本地文件夹」 |
| R-GI-019 | G-H1-03 | 交互 | 点击切换 | 字段区随之切换：本地分支显示「来源」+ 选择文件夹；Git 分支显示「Git 仓库」+「Git ref」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/account.go:ErrMissingStatusReader「来源」 |
| R-GI-020 | G-H1-03 | 关联 | 在来源页签之间来回切换后 | 切换后各自字段值按各自记忆保留（切回不清空） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H1-03→internal/ui/plugin_install.go:App.pluginInstallView；标记 = Open） |
| R-GI-021 | G-H1-03 | 样式 | 分段按钮 | 高 28、圆角 6、等宽各占一半；选中 `secondary` 底、未选中透明 | — | 判据 = 圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px；底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-H1-03→internal/ui/plugin_install.go:App.pluginInstallView；标记 = TextColor） |
| R-GI-022 | G-H1-03 | 无障碍 | 读屏进入该元素时 | 读作一组单选（tablist 或 radiogroup），含选中态 | — | 读屏文本 = 「本地文件夹」 | 已实现（映射 = G-H1-03→internal/ui/plugin_install.go:App.pluginInstallView；标记 = Role） |
| R-GI-023 | G-H1-03 | 键盘 | ← / → 或 Tab + Space | 可切换模式 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 未实现（映射到位但没有该维度的标记）：G-H1-03→internal/ui/plugin_install.go:App.pluginInstallView |
| R-GI-024 | G-H1-03 | 异常 | 切换时安装进行中 | 切换禁用（避免中途换来源） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H1-03→internal/ui/plugin_install.go:App.pluginInstallView；标记 = err） |
| R-GI-025 | G-H1-04 | 状态 | 未观察（条件）：本地文件夹模式 | 字段标签「来源」+ 输入框（占位逐字「/path/to/plugin」） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/account.go:ErrMissingStatusReader「来源」 |
| R-GI-026 | G-H1-04 | 交互 | 键入路径 | 值与系统选择结果共用同一字段（手输与选择等价） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H1-04→internal/ui/plugin_install.go:App.pluginInstallView；标记 = Pressed@accIconButton） |
| R-GI-027 | G-H1-04 | 内容 | 路径超长 | 单行横向滚动，不撑高 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-H1-04→internal/ui/plugin_install.go:App.pluginInstallView；标记 = 文案「安装来源」等 2 条） |
| R-GI-028 | G-H1-04 | 样式 | 输入框在屏上时 | 描边 + 圆角 6、高 32；聚焦转强调色 | — | 判据 = 控件尺寸 按钮高 行内 21 / 小 25 / 次 28 / 常规 32 / 大 35；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px | 已实现（映射 = G-H1-04→internal/ui/plugin_install.go:App.pluginInstallView；标记 = TextColor） |
| R-GI-029 | G-H1-04 | 无障碍 | 读屏进入该元素时 | 与标签关联，读作「来源，编辑框」 | — | 读屏文本 = 「来源，编辑框」 | 已实现（映射 = G-H1-04→internal/ui/plugin_install.go:App.pluginInstallView；标记 = Role） |
| R-GI-030 | G-H1-04 | 异常 | 路径不存在 / 非目录 | 提交时报错「请选择插件文件夹。」类提示，不在输入时就地弹错 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/plugin_install.go:pluginPathEmpty「请选择插件文件夹。」 |
| R-GI-031 | G-H1-04 | 记忆 | 关闭后再次打开时 | 不记忆上次路径（打开为空，占位可见） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-H1-04→internal/ui/plugin_install.go:App.pluginInstallView |
| R-GI-032 | G-H1-04 | 内容 | 路径为相对路径 | 允许输入，提交时按实现解析或报错（固定行为） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 已实现（映射 = G-H1-04→internal/ui/plugin_install.go:App.pluginInstallView；标记 = 文案「安装来源」等 2 条） |
| R-GI-033 | G-H1-05 | 状态 | 未观察（条件）：本地文件夹模式 | 输入框右侧图标按钮（文件夹图标，`outline`），可读名「选择文件夹」 | — | 屏上出现 1 个图标按钮 | 已实现（逐字）：internal/ui/plugin_install.go:pluginPickFolder「选择文件夹」 |
| R-GI-034 | G-H1-05 | 交互 | 鼠标点击该元素时 | 打开系统目录选择框；选定后回填到来源输入框 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H1-05→internal/ui/plugin_install.go:App.pluginInstallView；标记 = Pressed@accIconButton） |
| R-GI-035 | G-H1-05 | 交互 | 取消选择 | 输入框保持原值不变 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H1-05→internal/ui/plugin_install.go:App.pluginInstallView；标记 = Pressed@accIconButton） |
| R-GI-036 | G-H1-05 | 样式 | 图标按钮在屏上时 | 32×32、描边、圆角 6；与输入框同高 | — | 判据 = 控件尺寸 图标 16 / 14 / 12px；描边 常规 1px（卡片外框 0.5px / 焦点环 3px） | 已实现（映射 = G-H1-05→internal/ui/plugin_install.go:App.pluginInstallView；标记 = TextColor） |
| R-GI-037 | G-H1-05 | 无障碍 | 读屏进入该元素时 | 可读名「选择文件夹」 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/plugin_install.go:pluginPickFolder「选择文件夹」 |
| R-GI-038 | G-H1-05 | 键盘 | 按 Enter 时 | 等同点击 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 未实现（映射到位但没有该维度的标记）：G-H1-05→internal/ui/plugin_install.go:App.pluginInstallView |
| R-GI-039 | G-H1-05 | 异常 | 系统对话框不可用 | 提示失败，手输路径仍可用 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H1-05→internal/ui/plugin_install.go:App.pluginInstallView；标记 = err） |
| R-GI-040 | G-H1-05 | 内容 | 已选路径后 | 按钮文案不变（仍为图标按钮），不显示已选文件名 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GI-041 | G-H1-05 | 状态 | 安装进行中时 | 禁用 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H1-05→internal/ui/plugin_install.go:App.pluginInstallView；标记 = Disabled） |
| R-GI-042 | G-H1-06 | 状态 | 未观察（条件）：Git 模式 | 字段标签「Git 仓库」+ 输入框（占位逐字「owner/repo 或 https://github.com/owner/repo」） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/plugin_install.go:pluginModeGit「Git 仓库」 |
| R-GI-043 | G-H1-06 | 内容 | 支持两种写法 | `owner/repo` 与完整 URL 都接受，提交时归一 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 已实现（映射 = G-H1-06→internal/ui/plugin_install.go:App.pluginInstallView；标记 = 文案「安装来源」等 2 条） |
| R-GI-044 | G-H1-06 | 样式 | 仓库地址与 Git URL 两个输入框先后出现时 | 同款输入框（高 32、圆角 6） | — | 判据 = 控件尺寸 按钮高 行内 21 / 小 25 / 次 28 / 常规 32 / 大 35；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px | 已实现（映射 = G-H1-06→internal/ui/plugin_install.go:App.pluginInstallView；标记 = TextColor） |
| R-GI-045 | G-H1-06 | 无障碍 | 读屏进入该元素时 | 读作「Git 仓库，编辑框」 | — | 读屏文本 = 「Git 仓库，编辑框」 | 已实现（映射 = G-H1-06→internal/ui/plugin_install.go:App.pluginInstallView；标记 = Role） |
| R-GI-046 | G-H1-06 | 异常 | 非 GitHub 域名 | 允许（Git URL 通用），不预判失败 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H1-06→internal/ui/plugin_install.go:App.pluginInstallView；标记 = err） |
| R-GI-047 | G-H1-06 | 内容 | 仓库地址含 `.git` 后缀 | 接受，不额外改写 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 已实现（映射 = G-H1-06→internal/ui/plugin_install.go:App.pluginInstallView；标记 = 文案「安装来源」等 2 条） |
| R-GI-048 | G-H1-06 | 状态 | 本地模式下 | 该字段不渲染（随模式切换） | — | 屏上 字段标签 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GI-049 | G-H1-06 | 样式 | 仓库地址与 Git URL 两个输入框先后出现时 | 同款输入框（高 32、圆角 6） | — | 判据 = 控件尺寸 按钮高 行内 21 / 小 25 / 次 28 / 常规 32 / 大 35；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px | 已实现（映射 = G-H1-06→internal/ui/plugin_install.go:App.pluginInstallView；标记 = TextColor） |
| R-GI-050 | G-H1-07 | 状态 | 未观察（条件）：Git 模式 | 字段标签「Git ref」+ 输入框（占位「main」） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/plugin_install.go:pluginRefLabel「Git ref」 |
| R-GI-051 | G-H1-07 | 内容 | 输入框留空时 | 视为默认分支（不显示 `undefined`） | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GI-052 | G-H1-07 | 内容 | 分支/tag/commit 均可 | 原样传给安装流程，不做格式校验之外的改写 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H1-07→internal/ui/plugin_install.go:App.pluginInstallView；标记 = 文案「安装来源」等 2 条） |
| R-GI-053 | G-H1-07 | 样式 | 分支与子目录输入框同屏时 | 同款输入框 | — | 判据 = 控件尺寸 按钮高 行内 21 / 小 25 / 次 28 / 常规 32 / 大 35；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px | 已实现（映射 = G-H1-07→internal/ui/plugin_install.go:App.pluginInstallView；标记 = TextColor） |
| R-GI-054 | G-H1-07 | 无障碍 | 读屏进入该元素时 | 读作「Git ref，编辑框」 | — | 读屏文本 = 「Git ref，编辑框」 | 已实现（映射 = G-H1-07→internal/ui/plugin_install.go:App.pluginInstallView；标记 = Role） |
| R-GI-055 | G-H1-07 | 异常 | ref 不存在 | 提交时报错并保留输入 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H1-07→internal/ui/plugin_install.go:App.pluginInstallView；标记 = err） |
| R-GI-056 | G-H1-07 | 内容 | ref 为 commit hash | 接受，原样传递 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H1-07→internal/ui/plugin_install.go:App.pluginInstallView；标记 = 文案「安装来源」等 2 条） |
| R-GI-057 | G-H1-07 | 状态 | 本地模式下 | 不渲染 | — | 屏上 字段标签 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GI-058 | G-H1-07 | 样式 | 分支与子目录输入框同屏时 | 同款输入框 | — | 判据 = 控件尺寸 按钮高 行内 21 / 小 25 / 次 28 / 常规 32 / 大 35 | 已实现（映射 = G-H1-07→internal/ui/plugin_install.go:App.pluginInstallView；标记 = TextColor） |
| R-GI-059 | G-H1-08 | 状态 | 未观察（条件）：两种模式下都在 | 字段标签「插件 id（可选）」+ 输入框（占位「根据 manifest 自动生成」） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/plugin_install.go:pluginIDLabel「插件 id（可选）」 |
| R-GI-060 | G-H1-08 | 内容 | 输入框留空时 | 由 manifest 自动生成 id，不报错 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GI-061 | G-H1-08 | 内容 | 手填非法 id（含空格/大写） | 提交时报错并说明规则 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 已实现（映射 = G-H1-08→internal/ui/plugin_install.go:App.pluginInstallView；标记 = 文案「安装来源」等 2 条） |
| R-GI-062 | G-H1-08 | 样式 | 多个输入框同屏时 | 同款（高 32、圆角 6） | — | 判据 = 控件尺寸 按钮高 行内 21 / 小 25 / 次 28 / 常规 32 / 大 35；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px | 已实现（映射 = G-H1-08→internal/ui/plugin_install.go:App.pluginInstallView；标记 = TextColor） |
| R-GI-063 | G-H1-08 | 无障碍 | 读屏进入该元素时 | 读作「插件 id（可选），编辑框」 | — | 读屏文本 = 「插件 id（可选），编辑框」 | 已实现（映射 = G-H1-08→internal/ui/plugin_install.go:App.pluginInstallView；标记 = Role） |
| R-GI-064 | G-H1-08 | 异常 | id 与已装插件冲突 | 走 G-H3 替换确认（若同目录名）或报冲突 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H1-08→internal/ui/plugin_install.go:App.pluginInstallView；标记 = err） |
| R-GI-065 | G-H1-08 | 内容 | id 含非法字符 | 提交时报错并说明规则 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 已实现（映射 = G-H1-08→internal/ui/plugin_install.go:App.pluginInstallView；标记 = 文案「安装来源」等 2 条） |
| R-GI-066 | G-H1-08 | 状态 | 安装进行中时 | 禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H1-08→internal/ui/plugin_install.go:App.pluginInstallView；标记 = Disabled） |
| R-GI-067 | G-H1-09 | 状态 | 未观察（条件）：弹窗打开 | 「安装后启用」开关（右侧），默认开 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（逐字）：internal/ui/plugin_install.go:pluginEnableLabel「安装后启用」 |
| R-GI-068 | G-H1-09 | 交互 | 鼠标点击该元素时 | 开↔关切换；关时安装后不启用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H1-09→internal/ui/plugin_install.go:App.pluginInstallView；标记 = Pressed@accIconButton） |
| R-GI-069 | G-H1-09 | 状态 | 安装进行中时 | 开关禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H1-09→internal/ui/plugin_install.go:App.pluginInstallView；标记 = Disabled） |
| R-GI-070 | G-H1-09 | 样式 | 开关 | 32×18 轨道、圆角全、开位主色 | — | 判据 = 控件尺寸 开关 28×16；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-H1-09→internal/ui/plugin_install.go:App.pluginInstallView；标记 = TextColor） |
| R-GI-071 | G-H1-09 | 无障碍 | 读屏进入该元素时 | 读作开关，含开/关态与标签 | — | 读屏文本 = 「安装后启用」 | 已实现（映射 = G-H1-09→internal/ui/plugin_install.go:App.pluginInstallView；标记 = Role） |
| R-GI-072 | G-H1-09 | 键盘 | 按 Space 时 | 切换 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 未实现（映射到位但没有该维度的标记）：G-H1-09→internal/ui/plugin_install.go:App.pluginInstallView |
| R-GI-073 | G-H1-09 | 记忆 | 关闭后再次打开时 | 回默认开（不记忆上次） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-H1-09→internal/ui/plugin_install.go:App.pluginInstallView |
| R-GI-074 | G-H1-09 | 内容 | 开关关闭后安装 | 插件已装但未启用，列表显示为未启用 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H1-09→internal/ui/plugin_install.go:App.pluginInstallView；标记 = 文案「安装来源」等 2 条） |
| R-GI-075 | G-H1-09 | 异常 | 启用失败 | 安装完成但启用报错，列表显示未启用并给原因 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H1-09→internal/ui/plugin_install.go:App.pluginInstallView；标记 = err） |
| R-GI-076 | G-H1-10 | 内容 | 未观察（条件）：空路径提交 | 错误文字逐字「请选择插件文件夹。」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/plugin_install.go:pluginPathEmpty「请选择插件文件夹。」 |
| R-GI-077 | G-H1-10 | 内容 | 空 Git 仓库提交 | 逐字「请输入 Git 仓库。」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/plugin_install.go:pluginGitEmpty「请输入 Git 仓库。」 |
| R-GI-078 | G-H1-10 | 内容 | 安装 RPC 失败 | 显示服务端错误原文 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H1-10→internal/ui/plugin_install.go:App.pluginInstallView；标记 = 文案「安装来源」等 2 条） |
| R-GI-079 | G-H1-10 | 状态 | 三分支互斥 | 同一时刻只显示一条 | — | 文案计数为 1 | 已实现（映射 = G-H1-10→internal/ui/plugin_install.go:App.pluginInstallView；标记 = Disabled） |
| R-GI-080 | G-H1-10 | 交互 | 补齐输入后重试 | 错误先清除再提交 | — | 错误段恒 1 | 已实现（映射 = G-H1-10→internal/ui/plugin_install.go:App.pluginInstallView；标记 = Pressed@accIconButton） |
| R-GI-081 | G-H1-10 | 样式 | 错误段在屏上时 | 14px 危险色、按钮行之上 | — | 判据 = 字号 正文 14px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-H1-10→internal/ui/plugin_install.go:App.pluginInstallView；标记 = TextColor） |
| R-GI-082 | G-H1-10 | 无障碍 | 读屏进入该元素时 | `role=alert` 播报一次 | — | 播报次数 = 1 | 已实现（映射 = G-H1-10→internal/ui/plugin_install.go:App.pluginInstallView；标记 = Role） |
| R-GI-083 | G-H1-10 | 内容 | 错误文案 | 两段校验文案 + 服务端错误，三选一 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H1-10→internal/ui/plugin_install.go:App.pluginInstallView；标记 = 文案「安装来源」等 2 条） |
| R-GI-084 | G-H1-10 | 样式 | 错误段出现时 | 按钮行之上、字段之下 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-H1-10→internal/ui/plugin_install.go:App.pluginInstallView；标记 = TextColor） |
| R-GI-085 | G-H1-11 | 状态 | 未观察（条件）：弹窗在场 | 次按钮「取消」（`outline`） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GI-086 | G-H1-11 | 交互 | 鼠标点击该元素时 | 关闭；不安装任何插件 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H1-11→internal/ui/plugin_install.go:App.pluginInstallView；标记 = Pressed@accIconButton） |
| R-GI-087 | G-H1-11 | 状态 | 安装进行中时 | 禁用（不允许半途取消） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H1-11→internal/ui/plugin_install.go:App.pluginInstallView；标记 = Disabled） |
| R-GI-088 | G-H1-11 | 键盘 | 按 Esc 时 | 未安装时等同点击；安装中无效 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 未实现（映射到位但没有该维度的标记）：G-H1-11→internal/ui/plugin_install.go:App.pluginInstallView |
| R-GI-089 | G-H1-11 | 样式 | 次按钮在屏上时 | 描边、高 28 | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-H1-11→internal/ui/plugin_install.go:App.pluginInstallView；标记 = TextColor） |
| R-GI-090 | G-H1-11 | 无障碍 | 读屏进入该元素时 | 读作按钮「取消」 | — | 读屏文本 = 「取消」 | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GI-091 | G-H1-11 | 内容 | 文案在屏上时 | 恒为「取消」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GI-092 | G-H1-11 | 异常 | 安装进行中按 Esc 时 | 无效（按钮已禁用） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H1-11→internal/ui/plugin_install.go:App.pluginInstallView；标记 = err） |
| R-GI-093 | G-H1-12 | 状态 | 未观察（条件）：弹窗在场 | 主按钮「添加插件」，安装中前置旋转图标 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/plugin_install.go:pluginDialogTitle「添加插件」 |
| R-GI-094 | G-H1-12 | 交互 | 鼠标点击该元素时 | 开始安装；成功后关闭弹窗并刷新插件列表 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H1-12→internal/ui/plugin_install.go:App.pluginInstallView；标记 = Pressed@accIconButton） |
| R-GI-095 | G-H1-12 | 状态 | 安装进行中时 | 禁用 + 旋转图标 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H1-12→internal/ui/plugin_install.go:App.pluginInstallView；标记 = Disabled） |
| R-GI-096 | G-H1-12 | 异常 | 安装失败 | 弹窗保留、错误可见（G-H1-10）、按钮恢复 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H1-12→internal/ui/plugin_install.go:App.pluginInstallView；标记 = err） |
| R-GI-097 | G-H1-12 | 样式 | 主按钮在屏上时 | 实心主色、白字、高 28 | — | 判据 = 底色 主色 明 #339751 / 暗 #63cb7b；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-H1-12→internal/ui/plugin_install.go:App.pluginInstallView；标记 = TextColor） |
| R-GI-098 | G-H1-12 | 无障碍 | 读屏进入该元素时 | 读作按钮「添加插件」 | — | 读屏文本 = 「添加插件」 | 已实现（逐字）：internal/ui/plugin_install.go:pluginDialogTitle「添加插件」 |
| R-GI-099 | G-H1-12 | 键盘 | 按 Enter 时 | 等同点击 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 未实现（映射到位但没有该维度的标记）：G-H1-12→internal/ui/plugin_install.go:App.pluginInstallView |
| R-GI-100 | G-H1-12 | 内容 | 可用态文案渲染后 | 恒为「添加插件」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/plugin_install.go:pluginDialogTitle「添加插件」 |
| R-GI-101 | G-H1-12 | 异常 | 安装进行中按 Esc 时 | 无效 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H1-12→internal/ui/plugin_install.go:App.pluginInstallView；标记 = err） |

### 8.2 市场安装（G-H2-01–G-H2-09）

| 需求 ID | 元素 ID | 维度 | 场景 | 期望 | 异常 | 验收 | 状态 |
|---|---|---|---|---|---|---|---|
| R-GI-102 | G-H2-01 | 内容 | 未观察（条件）：插件市场条目点「安装」 | 标题逐字「安装 {name}」，name 为条目名 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H2-01→internal/ui/plugin_manage.go:App.pluginMarketDetailView；标记 = 文案「安装 %s」等 4 条） |
| R-GI-103 | G-H2-01 | 内容 | 条目名超长 | 标题单行 truncate，不折行 | — | 标题 1 行 | 已实现（映射 = G-H2-01→internal/ui/plugin_manage.go:App.pluginMarketDetailView；标记 = 文案「安装 %s」等 4 条） |
| R-GI-104 | G-H2-01 | 样式 | 标题在屏上时 | 16px 级半粗；圆角 + 阴影 | — | 判据 = 字号 对话框标题 15.75px；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px | 已实现（映射 = G-H2-01→internal/ui/plugin_manage.go:App.pluginMarketDetailView；标记 = TextColor） |
| R-GI-105 | G-H2-01 | 无障碍 | 读屏进入该元素时 | 作为对话框名读出 | — | 读屏文本 = 「安装 {name}」 | 已实现（映射 = G-H2-01→internal/ui/plugin_manage.go:App.pluginMarketDetailView；标记 = Role） |
| R-GI-106 | G-H2-01 | 异常 | 条目详情加载失败 | 标题仍显示条目名（不空），下方给错误块 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-H2-01→internal/ui/plugin_manage.go:App.pluginMarketDetailView；标记 = Err@App.pluginManageState） |
| R-GI-107 | G-H2-01 | 内容 | 标题在屏上时 | 「安装 {name}」，name 为条目名原文 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H2-01→internal/ui/plugin_manage.go:App.pluginMarketDetailView；标记 = 文案「安装 %s」等 4 条） |
| R-GI-108 | G-H2-01 | 状态 | 条目详情加载中 | 标题已显示条目名（不等详情） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H2-01→internal/ui/plugin_manage.go:App.pluginMarketDetailView；标记 = State） |
| R-GI-109 | G-H2-01 | 无障碍 | 读屏进入该元素时 | 作为对话框名读出 | — | 读屏文本 = 「安装 {name}」 | 已实现（映射 = G-H2-01→internal/ui/plugin_manage.go:App.pluginMarketDetailView；标记 = Role） |
| R-GI-110 | G-H2-02 | 内容 | 未观察（条件）：技能列表打开时 | 描述逐字「选择要启用的内置技能。默认全选。」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/plugin_manage.go:pluginMarketInstallDesc「选择要启用的内置技能。默认全选。」 |
| R-GI-111 | G-H2-02 | 样式 | 描述在屏上时 | 14px 级次要色 | — | 判据 = 字号 正文 14px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-H2-02→internal/ui/plugin_manage.go:App.pluginMarketDetailView；标记 = TextColor） |
| R-GI-112 | G-H2-02 | 无障碍 | 读屏进入该元素时 | 作为描述读出 | — | 读屏文本 = 「选择要启用的内置技能。默认全选。」 | 已实现（映射 = G-H2-02→internal/ui/plugin_manage.go:App.pluginMarketDetailView；标记 = Role） |
| R-GI-113 | G-H2-02 | 异常 | 描述折行 | 不挤压加载行与列表 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H2-02→internal/ui/plugin_manage.go:App.pluginMarketDetailView；标记 = Err@App.pluginManageState） |
| R-GI-114 | G-H2-02 | 内容 | 描述在屏上时 | 恒为「选择要启用的内置技能。默认全选。」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/plugin_manage.go:pluginMarketInstallDesc「选择要启用的内置技能。默认全选。」 |
| R-GI-115 | G-H2-02 | 状态 | 无技能 | 描述仍在（说明与空态不冲突） | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-H2-02→internal/ui/plugin_manage.go:App.pluginMarketDetailView；标记 = State） |
| R-GI-116 | G-H2-02 | 样式 | 描述在屏上时 | 14px 次要色 | — | 判据 = 字号 正文 14px | 已实现（映射 = G-H2-02→internal/ui/plugin_manage.go:App.pluginMarketDetailView；标记 = TextColor） |
| R-GI-117 | G-H2-02 | 无障碍 | 读屏进入该元素时 | 作为描述读出 | — | 读屏文本 = 「选择要启用的内置技能。默认全选。」 | 已实现（映射 = G-H2-02→internal/ui/plugin_manage.go:App.pluginMarketDetailView；标记 = Role） |
| R-GI-118 | G-H2-03 | 状态 | 未观察（条件）：条目详情加载中 | 加载行：旋转图标 + 「正在加载...」（英文三点），14px 灰，居中 | — | 判据 = 字号 正文 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（逐字）：internal/ui/plugin_manage.go:pluginMarketLoading「正在加载...」 |
| R-GI-119 | G-H2-03 | 动画 | 加载进行中时 | 旋转连续；详情到达后整行消失 | 减少动效下静态 | 判据 = 动效 布局 250ms / 颜色 150ms / 折叠 300ms；缓动 cubic-bezier(0.16, 1, 0.3, 1) / cubic-bezier(0.4, 0, 0.2, 1) | 未实现（找不到元素渲染点） |
| R-GI-120 | G-H2-03 | 关联 | 插件列表加载中时 | 加载中「添加插件」禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/ui/plugin_install.go:pluginDialogTitle「添加插件」 |
| R-GI-121 | G-H2-03 | 样式 | 加载行在屏上时 | 14px 次要色、居中、上下留白 12 | — | 判据 = 字号 正文 14px；内边距 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28）/ 卡片外框 4.5；档位 3.5·7·10.5·14·21·28px | 未实现（找不到元素渲染点） |
| R-GI-122 | G-H2-03 | 无障碍 | 读屏进入该元素时 | `role=status` 播报一次 | — | 播报次数 = 1 | 未实现（找不到元素渲染点） |
| R-GI-123 | G-H2-03 | 内容 | 加载行在屏上时 | 「正在加载...」+ 旋转图标，居中 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/plugin_manage.go:pluginMarketLoading「正在加载...」 |
| R-GI-124 | G-H2-03 | 状态 | 加载完成 | 整行消失（不留空白 32px） | — | 屏上 加载行 计数 = 0 | 未实现（找不到元素渲染点） |
| R-GI-125 | G-H2-03 | 无障碍 | 读屏进入该元素时 | `role=status` 播报一次 | — | 播报次数 = 1 | 未实现（找不到元素渲染点） |
| R-GI-126 | G-H2-04 | 状态 | 未观察（条件）：详情加载失败 | 错误块：告警图标 + 错误文案 + 次级按钮「重试」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/app.go:App.actionErrorView「重试」 |
| R-GI-127 | G-H2-04 | 交互 | 鼠标点击「重试」时 | 重新拉取详情；成功后错误块消失、技能列表出现 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 未实现（找不到元素渲染点） |
| R-GI-128 | G-H2-04 | 状态 | 重试进行中时 | 「重试」禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/ui/app.go:App.actionErrorView「重试」 |
| R-GI-129 | G-H2-04 | 样式 | 错误块在屏上时 | 告警图标 16、14px 次要色、按钮高 28 | — | 判据 = 字号 正文 14px；其余 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0） | 未实现（找不到元素渲染点） |
| R-GI-130 | G-H2-04 | 无障碍 | 读屏进入该元素时 | `role=alert` 播报错误，随后可聚焦「重试」 | — | 读屏文本 = 「重试」 | 已实现（逐字）：internal/ui/app.go:App.actionErrorView「重试」 |
| R-GI-131 | G-H2-04 | 异常 | 连续失败 | 错误仍 1 块，可重复重试 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（找不到元素渲染点） |
| R-GI-132 | G-H2-04 | 内容 | 错误块在屏上时 | 告警图标 + 错误文案 + 「重试」，三件一组 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/app.go:App.actionErrorView「重试」 |
| R-GI-133 | G-H2-04 | 状态 | 重试后加载中 | 错误块换成加载行 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（找不到元素渲染点） |
| R-GI-134 | G-H2-05 | 内容 | 未观察（条件）：详情已载入 | 分组标签逐字「技能」（12px 中等灰） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalActionWord「技能」 |
| R-GI-135 | G-H2-05 | 状态 | 插件无技能 | 标签不出现（只留空态文案 G-H2-07） | — | 屏上 分组标签 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GI-136 | G-H2-05 | 样式 | 分组标签在屏上时 | 12px 次要色、与勾选行文字左缘对齐 | — | 判据 = 字号 UI 正文 12.25px；内边距 左缘 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28） | 已实现（映射 = G-H2-05→internal/ui/plugin_manage.go:App.pluginMarketDetailView；标记 = TextColor） |
| R-GI-137 | G-H2-05 | 无障碍 | 读屏进入该元素时 | 作为分组标题读出 | — | 读屏文本 = 「技能」 | 已实现（映射 = G-H2-05→internal/ui/plugin_manage.go:App.pluginMarketDetailView；标记 = Role） |
| R-GI-138 | G-H2-05 | 异常 | 技能数为 0 但标签渲染 | 不允许：标签与空态互斥 | — | 屏上 分组标签 计数 = 1 | 已实现（映射 = G-H2-05→internal/ui/plugin_manage.go:App.pluginMarketDetailView；标记 = Err@App.pluginManageState） |
| R-GI-139 | G-H2-05 | 内容 | 标签在屏上时 | 恒为「技能」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/approval.go:approvalActionWord「技能」 |
| R-GI-140 | G-H2-05 | 状态 | 技能列表为空 | 标签不出现 | — | 屏上 分组标签 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GI-141 | G-H2-05 | 无障碍 | 读屏进入该元素时 | 作为分组标题读出 | — | 读屏文本 = 「技能」 | 已实现（映射 = G-H2-05→internal/ui/plugin_manage.go:App.pluginMarketDetailView；标记 = Role） |
| R-GI-142 | G-H2-06 | 状态 | 未观察（条件）：插件含技能 | 每个技能一行：方框 + 名称（14px）+ 描述（12px）；默认全选 | — | 判据 = 字号 UI 正文 12.25px | 已实现（映射 = G-H2-06→internal/ui/plugin_manage.go:App.pluginMarketDetailView；标记 = State） |
| R-GI-143 | G-H2-06 | 交互 | 点行 | 切换勾选；只影响该技能的启用状态 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-H2-06→internal/ui/plugin_manage.go:App.pluginMarketDetailView |
| R-GI-144 | G-H2-06 | 内容 | 技能无描述 | 只显示名称（不留空行） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H2-06→internal/ui/plugin_manage.go:App.pluginMarketDetailView；标记 = 文案「安装 %s」等 4 条） |
| R-GI-145 | G-H2-06 | 内容 | 技能很多 | 列表可滚动，最多高受弹窗约束 | — | 判据 = 滚动条宽 6、滑块 ≥ 24（SC-WIDTH-6 / SC-THUMB-24）；滚轮一格 54、150ms（SC-STEP-54）；初始位置与横向溢出按 SC-ANCHOR / SC-OVERFLOW-0；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-H2-06→internal/ui/plugin_manage.go:App.pluginMarketDetailView；标记 = 文案「安装 %s」等 4 条） |
| R-GI-146 | G-H2-06 | 样式 | 勾选行 | 方框 14px、行高 40（含两行文字） | — | 判据 = 行高 空态主标题 28px；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-H2-06→internal/ui/plugin_manage.go:App.pluginMarketDetailView；标记 = TextColor） |
| R-GI-147 | G-H2-06 | 无障碍 | 读屏进入该元素时 | 读作复选项，含名称与描述 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H2-06→internal/ui/plugin_manage.go:App.pluginMarketDetailView；标记 = Role） |
| R-GI-148 | G-H2-06 | 内容 | 技能勾选行 | 方框 + 名称 + 描述，默认全选 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H2-06→internal/ui/plugin_manage.go:App.pluginMarketDetailView；标记 = 文案「安装 %s」等 4 条） |
| R-GI-149 | G-H2-06 | 状态 | 全部取消勾选 | 允许（安装为无技能插件） | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-H2-06→internal/ui/plugin_manage.go:App.pluginMarketDetailView；标记 = State） |
| R-GI-150 | G-H2-07 | 内容 | 未观察（条件）：插件无内置技能 | 空态文字逐字「此插件不包含技能。」（14px 灰） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/plugin_install.go:pluginEmptySkills「此插件不包含技能。」 |
| R-GI-151 | G-H2-07 | 状态 | 技能列表非空时 | 空态不出现 | — | 屏上 空态文字 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GI-152 | G-H2-07 | 关联 | 技能列表为空时 | 无技能时「添加插件」仍可用 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/plugin_install.go:pluginDialogTitle「添加插件」 |
| R-GI-153 | G-H2-07 | 样式 | 空态在屏上时 | 14px 次要色、居中或左对齐一致 | — | 判据 = 字号 正文 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（找不到元素渲染点） |
| R-GI-154 | G-H2-07 | 无障碍 | 读屏进入该元素时 | 读出该句 | — | 读屏文本 = 「此插件不包含技能。」 | 未实现（找不到元素渲染点） |
| R-GI-155 | G-H2-07 | 内容 | 空态在屏上时 | 恒为「此插件不包含技能。」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/plugin_install.go:pluginEmptySkills「此插件不包含技能。」 |
| R-GI-156 | G-H2-07 | 状态 | 技能列表非空时 | 不显示 | — | 屏上 空态文字 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GI-157 | G-H2-07 | 样式 | 空态在屏上时 | 14px 次要色 | — | 判据 = 字号 正文 14px | 未实现（找不到元素渲染点） |
| R-GI-158 | G-H2-08 | 状态 | 未观察（条件）：弹窗在场 | 次按钮「取消」（`outline`） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GI-159 | G-H2-08 | 交互 | 鼠标点击该元素时 | 关闭；不安装 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-H2-08→internal/ui/plugin_manage.go:App.pluginMarketDetailView |
| R-GI-160 | G-H2-08 | 键盘 | 按 Esc 时 | 等同点击 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 未实现（映射到位但没有该维度的标记）：G-H2-08→internal/ui/plugin_manage.go:App.pluginMarketDetailView |
| R-GI-161 | G-H2-08 | 样式 | 次按钮在屏上时 | 描边、高 28 | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-H2-08→internal/ui/plugin_manage.go:App.pluginMarketDetailView；标记 = TextColor） |
| R-GI-162 | G-H2-08 | 无障碍 | 读屏进入该元素时 | 读作按钮「取消」 | — | 读屏文本 = 「取消」 | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GI-163 | G-H2-08 | 内容 | 文案在屏上时 | 恒为「取消」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GI-164 | G-H2-08 | 状态 | 安装进行中时 | 禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H2-08→internal/ui/plugin_manage.go:App.pluginMarketDetailView；标记 = State） |
| R-GI-165 | G-H2-08 | 无障碍 | 读屏进入该元素时 | 读作按钮「取消」 | — | 读屏文本 = 「取消」 | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GI-166 | G-H2-09 | 状态 | 未观察（条件）：详情已载入且非安装中 | 主按钮「添加插件」可用；详情未载入/加载中/安装中时禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/ui/plugin_install.go:pluginDialogTitle「添加插件」 |
| R-GI-167 | G-H2-09 | 交互 | 鼠标点击该元素时 | 安装该插件（含勾选的技能）；成功后关闭并刷新 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-H2-09→internal/ui/plugin_manage.go:App.pluginMarketDetailView |
| R-GI-168 | G-H2-09 | 状态 | 安装进行中时 | 禁用并显示旋转图标 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H2-09→internal/ui/plugin_manage.go:App.pluginMarketDetailView；标记 = State） |
| R-GI-169 | G-H2-09 | 异常 | 需替换同名插件 | 转 G-H3 确认框，不在本框直接覆盖 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H2-09→internal/ui/plugin_manage.go:App.pluginMarketDetailView；标记 = Err@App.pluginManageState） |
| R-GI-170 | G-H2-09 | 样式 | 主按钮在屏上时 | 实心主色、白字、高 28；禁用降透明 | — | 判据 = 底色 主色 明 #339751 / 暗 #63cb7b；其余 = 色值取 token：主色 #349651/#63cb7b、正文 #0a0a0a/#ececec、次文 #737373/#8a8a8a、边框 #e5e5e5/#2e2e2e、警示 #e17100/#ffb900，偏差 0（CL-ACCENT / CL-TEXT / CL-TEXT-2 / CL-BORDER / CL-WARN） | 已实现（映射 = G-H2-09→internal/ui/plugin_manage.go:App.pluginMarketDetailView；标记 = TextColor） |
| R-GI-171 | G-H2-09 | 无障碍 | 读屏进入该元素时 | 读作按钮「添加插件」，禁用时播报原因 | — | 读屏文本 = 「添加插件」 | 已实现（逐字）：internal/ui/plugin_install.go:pluginDialogTitle「添加插件」 |
| R-GI-172 | G-H2-09 | 内容 | 可用态文案渲染后 | 恒为「添加插件」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/plugin_install.go:pluginDialogTitle「添加插件」 |
| R-GI-173 | G-H2-09 | 状态 | 加载失败 | 禁用（未取到详情不可装） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H2-09→internal/ui/plugin_manage.go:App.pluginMarketDetailView；标记 = State） |

### 8.3 替换现有插件确认（G-H3-01–G-H3-04）

| 需求 ID | 元素 ID | 维度 | 场景 | 期望 | 异常 | 验收 | 状态 |
|---|---|---|---|---|---|---|---|
| R-GI-174 | G-H3-01 | 内容 | 未观察（条件）：市场条目与已装插件同目录名 | 标题逐字「替换现有插件？」，问号为中文全角 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GI-175 | G-H3-01 | 状态 | 无同名插件 | 不弹此框（直接安装） | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（找不到元素渲染点） |
| R-GI-176 | G-H3-01 | 样式 | 标题在屏上时 | 16px 级半粗；圆角 + 阴影 + 遮罩 | — | 判据 = 字号 对话框标题 15.75px；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px | 未实现（找不到元素渲染点） |
| R-GI-177 | G-H3-01 | 无障碍 | 读屏进入该元素时 | 作为对话框名读出 | — | 读屏文本 = 「替换现有插件？」 | 未实现（找不到元素渲染点） |
| R-GI-178 | G-H3-01 | 窄窗 | 窗口收窄到最窄宽度时 | 标题单行，描述折行，按钮行完整 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 未实现（找不到元素渲染点） |
| R-GI-179 | G-H3-01 | 内容 | 标题在屏上时 | 恒为「替换现有插件？」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GI-180 | G-H3-01 | 状态 | 同名插件为市场来源 | 同样弹此框（来源不影响） | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（找不到元素渲染点） |
| R-GI-181 | G-H3-01 | 无障碍 | 读屏进入该元素时 | 作为对话框名读出 | — | 读屏文本 = 「替换现有插件？」 | 未实现（找不到元素渲染点） |
| R-GI-182 | G-H3-02 | 内容 | 未观察（条件）：同名插件来自市场来源时 | 描述逐字「"{name}" 将替换一个来自其他来源（git、手动安装或另一个市场条目）的同名插件。此操作无法撤销。」；`{name}` 用英文双引号 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GI-183 | G-H3-02 | 内容 | 名称含引号 | 名称原样嵌入，不转义 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GI-184 | G-H3-02 | 样式 | 描述在屏上时 | 14px 级次要色、折行 | — | 判据 = 字号 正文 14px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 未实现（找不到元素渲染点） |
| R-GI-185 | G-H3-02 | 无障碍 | 读屏进入该元素时 | 读出整句（含名称） | — | 读屏文本 = 「"{name}" 将替换一个来自其他来源（git」 | 未实现（找不到元素渲染点） |
| R-GI-186 | G-H3-02 | 内容 | 描述在屏上时 | 四段信息：名称、来源类型、无法撤销 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GI-187 | G-H3-02 | 样式 | 描述在屏上时 | 14px 次要色、折行 | — | 判据 = 字号 正文 14px | 未实现（找不到元素渲染点） |
| R-GI-188 | G-H3-02 | 异常 | 描述为空 | 不允许 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（找不到元素渲染点） |
| R-GI-189 | G-H3-02 | 无障碍 | 读屏进入该元素时 | 读出整句 | — | 读屏文本 = 「"{name}" 将替换一个来自其他来源（git」 | 未实现（找不到元素渲染点） |
| R-GI-190 | G-H3-02 | 窄窗 | 窗口收窄到最窄宽度时 | 描述折行，按钮行完整 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 未实现（找不到元素渲染点） |
| R-GI-191 | G-H3-03 | 状态 | 未观察（条件）：弹窗在场 | 次按钮「取消」（`outline`）；替换进行中禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GI-192 | G-H3-03 | 交互 | 鼠标点击该元素时 | 关闭；不替换；原插件保持 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（找不到元素渲染点） |
| R-GI-193 | G-H3-03 | 键盘 | 按 Esc 时 | 未开始时等同点击 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 未实现（找不到元素渲染点） |
| R-GI-194 | G-H3-03 | 样式 | 次按钮在屏上时 | 描边、高 28 | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（找不到元素渲染点） |
| R-GI-195 | G-H3-03 | 无障碍 | 读屏进入该元素时 | 读作按钮「取消」 | — | 读屏文本 = 「取消」 | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GI-196 | G-H3-03 | 内容 | 文案在屏上时 | 恒为「取消」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GI-197 | G-H3-03 | 无障碍 | 读屏进入该元素时 | 读作按钮「取消」 | — | 读屏文本 = 「取消」 | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GI-198 | G-H3-03 | 样式 | 取消与安装按钮同屏时 | 同高 28，描边次按钮 | — | 判据 = 控件尺寸 按钮高 次 28；描边 常规 1px（卡片外框 0.5px / 焦点环 3px） | 未实现（找不到元素渲染点） |
| R-GI-199 | G-H3-04 | 状态 | 未观察（条件）：弹窗在场 | 危险按钮「替换」，进行中前置旋转图标 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/settings_general_screen.go:generalImportDesc「替换」 |
| R-GI-200 | G-H3-04 | 交互 | 鼠标点击该元素时 | 卸载旧插件并安装新的；成功后关闭弹窗并刷新列表 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（找不到元素渲染点） |
| R-GI-201 | G-H3-04 | 状态 | 替换中 | 两个按钮均禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（找不到元素渲染点） |
| R-GI-202 | G-H3-04 | 异常 | 替换失败（旧插件已删、新插件未装） | 给失败原因并提示旧插件状态，不静默留下空目录 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GI-203 | G-H3-04 | 样式 | 危险按钮在屏上时 | 危险色底 + 白字、高 28 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；色值取 token：主色 #349651/#63cb7b、正文 #0a0a0a/#ececec、次文 #737373/#8a8a8a、边框 #e5e5e5/#2e2e2e、警示 #e17100/#ffb900，偏差 0（CL-ACCENT / CL-TEXT / CL-TEXT-2 / CL-BORDER / CL-WARN） | 未实现（找不到元素渲染点） |
| R-GI-204 | G-H3-04 | 无障碍 | 读屏进入该元素时 | 读作按钮「替换」，并说明无法撤销 | — | 读屏文本 = 「替换」 | 已实现（逐字）：internal/ui/settings_general_screen.go:generalImportDesc「替换」 |
| R-GI-205 | G-H3-04 | 内容 | 文案在屏上时 | 恒为「替换」，不带数量 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/settings_general_screen.go:generalImportDesc「替换」 |
| R-GI-206 | G-H3-04 | 状态 | 替换工作禁用 | 两个按钮都禁用，「取消」也禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |

### 8.4 插件可更新清单（G-H4-01–G-H4-07）

| 需求 ID | 元素 ID | 维度 | 场景 | 期望 | 异常 | 验收 | 状态 |
|---|---|---|---|---|---|---|---|
| R-GI-207 | G-H4-01 | 内容 | 未观察（条件）：已装插件有更高版本，点「更新」或更新横幅 | 标题逐字「可用更新」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/plugin_manage.go:pluginManageUpdates「可用更新」 |
| R-GI-208 | G-H4-01 | 样式 | 标题在屏上时 | 16px 级半粗；圆角 + 阴影 | — | 判据 = 字号 对话框标题 15.75px；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px | 已实现（映射 = G-H4-01→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = TextColor） |
| R-GI-209 | G-H4-01 | 无障碍 | 读屏进入该元素时 | 作为对话框名读出 | — | 读屏文本 = 「可用更新」 | 已实现（映射 = G-H4-01→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = Role） |
| R-GI-210 | G-H4-01 | 异常 | 打开时更新已完成（无待更新） | 标题仍在，列表给空态（G-H4-02） | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H4-01→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = Err） |
| R-GI-211 | G-H4-01 | 窄窗 | 窗口收窄到最窄宽度时 | 标题单行，行与按钮完整 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-H4-01→internal/ui/plugin_manage.go:App.pluginUpdatesSection |
| R-GI-212 | G-H4-01 | 内容 | 标题在屏上时 | 恒为「可用更新」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/plugin_manage.go:pluginManageUpdates「可用更新」 |
| R-GI-213 | G-H4-01 | 异常 | 更新加载失败 | 标题仍显示，列表给失败提示 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H4-01→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = Err） |
| R-GI-214 | G-H4-01 | 无障碍 | 读屏进入该元素时 | 作为对话框名读出 | — | 读屏文本 = 「可用更新」 | 已实现（映射 = G-H4-01→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = Role） |
| R-GI-215 | G-H4-02 | 内容 | 未观察（条件）：打开时无待更新项 | 空态文字逐字「所有插件均为最新版本」（14px 灰，居中） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/plugin_manage.go:pluginManageNoUpdates「所有插件均为最新版本」 |
| R-GI-216 | G-H4-02 | 状态 | 有待更新项 | 空态不出现 | — | 屏上 空态文字 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GI-217 | G-H4-02 | 关联 | 更新列表为空时 | 空态时「全部更新」不出现或禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GI-218 | G-H4-02 | 样式 | 空态在屏上时 | 14px 次要色、居中 | — | 判据 = 字号 正文 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-H4-02→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = TextColor） |
| R-GI-219 | G-H4-02 | 无障碍 | 读屏进入该元素时 | 读出该句 | — | 读屏文本 = 「所有插件均为最新版本」 | 已实现（映射 = G-H4-02→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = Role） |
| R-GI-220 | G-H4-02 | 异常 | 更新全部完成后 | 列表清空并显示本空态 | — | 列表 计数 = 0；空态文字 计数 = 1 | 已实现（映射 = G-H4-02→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = Err） |
| R-GI-221 | G-H4-02 | 内容 | 空态在屏上时 | 恒为「所有插件均为最新版本」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/plugin_manage.go:pluginManageNoUpdates「所有插件均为最新版本」 |
| R-GI-222 | G-H4-02 | 状态 | 全部更新完成后 | 出现该空态（行全部消失） | — | 屏上 空态文字 计数 = 0 | 已实现（映射 = G-H4-02→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = State） |
| R-GI-223 | G-H4-03 | 状态 | 未观察（条件）：有可更新项 | 每行行首：40px 底托 + 方块占位图标；后接插件名称（14px 中等，truncate） | — | 底托 40、名称单行 | 已实现（映射 = G-H4-03→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = State） |
| R-GI-224 | G-H4-03 | 内容 | 名称超长 | 名称 truncate，`title` 给完整名 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-H4-03→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = 文案「可用更新」等 7 条） |
| R-GI-225 | G-H4-03 | 内容 | 无插件图标 | 用方块占位图标（不显示破图） | — | 屏上 破图 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GI-226 | G-H4-03 | 样式 | 行在屏上时 | 行高约 56（含两行信息）；图标与文字垂直居中 | — | 判据 = 行高 UI 正文 17.5px；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-H4-03→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = TextColor） |
| R-GI-227 | G-H4-03 | 无障碍 | 读屏进入该元素时 | 播报插件名 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H4-03→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = Role） |
| R-GI-228 | G-H4-03 | 内容 | 名称在屏上时 | 插件名 14px 中等，单行 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-H4-03→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = 文案「可用更新」等 7 条） |
| R-GI-229 | G-H4-03 | 状态 | 安装进行中时 | 名称与图标不变，只加进度条 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H4-03→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = State） |
| R-GI-230 | G-H4-03 | 无障碍 | 读屏进入该元素时 | 播报插件名 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H4-03→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = Role） |
| R-GI-231 | G-H4-03 | 关联 | 名称行与版本行上下排列时 | 名称与版本行上下两行，左缘对齐 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-H4-03→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = Section） |
| R-GI-232 | G-H4-04 | 内容 | 未观察（条件）：有记录版本 | 版本行逐字「v{旧} → v{新}」（12px 灰） | — | 屏上形如「v1.2.0 → v1.3.0」 | 已实现（映射 = G-H4-04→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = 文案「可用更新」等 7 条） |
| R-GI-233 | G-H4-04 | 内容 | 旧版本未知 | 只显示「v{新}」，不显示「v? →」 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GI-234 | G-H4-04 | 样式 | 版本行在屏上时 | 12px 次要色、与名称左缘对齐 | — | 判据 = 字号 UI 正文 12.25px；内边距 左缘 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28） | 已实现（映射 = G-H4-04→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = TextColor） |
| R-GI-235 | G-H4-04 | 无障碍 | 读屏进入该元素时 | 读出旧与新版本 | — | 播报顺序 = v{旧} → v{新} → v{新} | 已实现（映射 = G-H4-04→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = Role） |
| R-GI-236 | G-H4-04 | 异常 | 版本号非语义化 | 原样显示，不改写 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H4-04→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = Err） |
| R-GI-237 | G-H4-04 | 内容 | 版本行在屏上时 | 两种形态：「v{旧} → v{新}」或「v{新}」 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM） | 已实现（映射 = G-H4-04→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = 文案「可用更新」等 7 条） |
| R-GI-238 | G-H4-04 | 异常 | 版本号缺失 | 不显示版本行（不留空行） | — | 屏上 版本行 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GI-239 | G-H4-04 | 无障碍 | 读屏进入该元素时 | 读出两版本 | — | 读屏文本 = 「v{旧} → v{新}」 | 已实现（映射 = G-H4-04→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = Role） |
| R-GI-240 | G-H4-04 | 状态 | 该插件下载中 | 版本行保持显示（不被进度条顶掉） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H4-04→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = State） |
| R-GI-241 | G-H4-05 | 状态 | 未观察（条件）：该插件安装中 | 行内出现 1px 高进度条 + 12px 灰字「已下载 {size}…」或「{已下载} / {总量}」 | — | 判据 = 字号 UI 正文 12.25px；字重 常规 400 / 中粗 500 / 半粗 600 / 粗 700 | 已实现（映射 = G-H4-05→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = State） |
| R-GI-242 | G-H4-05 | 内容 | 总量未知 | 只显示「已下载 {size}…」，不显示分母 | — | 屏上 分母 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GI-243 | G-H4-05 | 动画 | 下载中 | 进度条宽度随进度增长，不跳回 | 减少动效下直接更新宽度 | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（映射到位但没有该维度的标记）：G-H4-05→internal/ui/plugin_manage.go:App.pluginUpdatesSection |
| R-GI-244 | G-H4-05 | 样式 | 进度条 | 1px 高、主色、无圆角；位于行底部 | — | 判据 = 色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-H4-05→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = TextColor） |
| R-GI-245 | G-H4-05 | 无障碍 | 读屏进入该元素时 | 播报百分比或字节进度（限频，不逐字节） | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 已实现（映射 = G-H4-05→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = Role） |
| R-GI-246 | G-H4-05 | 异常 | 下载中断 | 进度条停在原处并给失败提示，不假装完成 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H4-05→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = Err） |
| R-GI-247 | G-H4-05 | 内容 | 进度文案 | 「已下载 {size}…」或「{已下载} / {总量}」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H4-05→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = 文案「可用更新」等 7 条） |
| R-GI-248 | G-H4-05 | 状态 | 下载完成 | 进度条消失或填满，行消失 | — | 屏上 进度条 计数 = 0 | 已实现（映射 = G-H4-05→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = State） |
| R-GI-249 | G-H4-06 | 状态 | 未观察（条件）：有可更新项 | 行尾次级小按钮「更新」（刷新图标），安装中旋转 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/catalog/branch.go:Catalog.ApplyBranchUpdate「更新」 |
| R-GI-250 | G-H4-06 | 交互 | 鼠标点击该元素时 | 只更新该插件；成功后该行消失 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 已实现（映射 = G-H4-06→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = Pressed@pluginSmallButtonLabeled） |
| R-GI-251 | G-H4-06 | 状态 | 安装中或「全部更新」进行中 | 按钮禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H4-06→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = State） |
| R-GI-252 | G-H4-06 | 异常 | 更新失败 | 行保留并给失败原因，按钮回可点 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H4-06→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = Err） |
| R-GI-253 | G-H4-06 | 样式 | 次级按钮在屏上时 | `outline`、高 28、图标 14、12–13px | — | 判据 = 控件尺寸 按钮高 次 28；其余 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0） | 已实现（映射 = G-H4-06→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = TextColor） |
| R-GI-254 | G-H4-06 | 无障碍 | 读屏进入该元素时 | 读作按钮「更新 {插件名}」 | — | 读屏文本 = 「更新 {插件名}」 | 已实现（映射 = G-H4-06→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = Role） |
| R-GI-255 | G-H4-06 | 内容 | 按钮文案渲染后 | 恒为「更新」，带刷新图标 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/catalog/branch.go:Catalog.ApplyBranchUpdate「更新」 |
| R-GI-256 | G-H4-06 | 状态 | 该行安装中 | 按钮换成旋转图标（文案保留） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H4-06→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = State） |
| R-GI-257 | G-H4-07 | 状态 | 未观察（条件）：有可更新项 | 主按钮逐字「全部更新（{count}）」，更新中前置旋转图标 | — | 屏上形如「全部更新（3）」 | 已实现（映射 = G-H4-07→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = State） |
| R-GI-258 | G-H4-07 | 交互 | 鼠标点击该元素时 | 依次更新全部；每行进度独立 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H4-07→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = Pressed@pluginSmallButtonLabeled） |
| R-GI-259 | G-H4-07 | 状态 | 更新进行中时 | 按钮禁用；行内「更新」同时禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/catalog/branch.go:Catalog.ApplyBranchUpdate「更新」 |
| R-GI-260 | G-H4-07 | 异常 | 部分失败 | 失败行保留、成功行消失；按钮恢复并给汇总提示 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 已实现（映射 = G-H4-07→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = Err） |
| R-GI-261 | G-H4-07 | 样式 | 主按钮在屏上时 | 实心主色、白字、高 28 | — | 判据 = 底色 主色 明 #339751 / 暗 #63cb7b；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-H4-07→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = TextColor） |
| R-GI-262 | G-H4-07 | 无障碍 | 读屏进入该元素时 | 读作按钮「全部更新，共 N 个」 | — | 读屏文本 = 「全部更新，共 N 个」 | 已实现（映射 = G-H4-07→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = Role） |
| R-GI-263 | G-H4-07 | 内容 | 按钮文案渲染后 | 「全部更新（{count}）」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H4-07→internal/ui/plugin_manage.go:App.pluginUpdatesSection；标记 = 文案「可用更新」等 7 条） |
| R-GI-264 | G-H4-07 | 状态 | 数量为 0 | 按钮不出现（无更新项） | — | 屏上 主按钮 计数 = 0（无更新项时） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |

### 8.5 技能可更新清单（G-H5-01–G-H5-06）

| 需求 ID | 元素 ID | 维度 | 场景 | 期望 | 异常 | 验收 | 状态 |
|---|---|---|---|---|---|---|---|
| R-GI-265 | G-H5-01 | 内容 | 未观察（条件）：已装技能有更高版本 | 标题逐字「可用更新」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/plugin_manage.go:pluginManageUpdates「可用更新」 |
| R-GI-266 | G-H5-01 | 样式 | 标题在屏上时 | 16px 级半粗；圆角 + 阴影 | — | 判据 = 字号 对话框标题 15.75px；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px | 未实现（找不到元素渲染点） |
| R-GI-267 | G-H5-01 | 无障碍 | 读屏进入该元素时 | 作为对话框名读出 | — | 读屏文本 = 「可用更新」 | 未实现（找不到元素渲染点） |
| R-GI-268 | G-H5-01 | 异常 | 无待更新 | 标题仍在，列表给空态 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（找不到元素渲染点） |
| R-GI-269 | G-H5-01 | 窄窗 | 窗口收窄到最窄宽度时 | 标题单行，行与按钮完整 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（找不到元素渲染点） |
| R-GI-270 | G-H5-01 | 内容 | 标题在屏上时 | 恒为「可用更新」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/plugin_manage.go:pluginManageUpdates「可用更新」 |
| R-GI-271 | G-H5-01 | 异常 | 加载失败 | 标题仍在，列表给失败提示 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（找不到元素渲染点） |
| R-GI-272 | G-H5-01 | 无障碍 | 读屏进入该元素时 | 作为对话框名读出 | — | 读屏文本 = 「可用更新」 | 未实现（找不到元素渲染点） |
| R-GI-273 | G-H5-02 | 内容 | 未观察（条件）：无待更新项 | 空态文字逐字「所有技能均为最新版本」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GI-274 | G-H5-02 | 状态 | 有待更新项 | 空态不出现 | — | 屏上 空态文字 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GI-275 | G-H5-02 | 样式 | 空态在屏上时 | 14px 次要色、居中 | — | 判据 = 字号 正文 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（找不到元素渲染点） |
| R-GI-276 | G-H5-02 | 无障碍 | 读屏进入该元素时 | 读出该句 | — | 读屏文本 = 「所有技能均为最新版本」 | 未实现（找不到元素渲染点） |
| R-GI-277 | G-H5-02 | 内容 | 空态在屏上时 | 恒为「所有技能均为最新版本」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GI-278 | G-H5-02 | 状态 | 全部更新后 | 出现空态 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（找不到元素渲染点） |
| R-GI-279 | G-H5-02 | 样式 | 空态在屏上时 | 14px 次要色、居中 | — | 判据 = 字号 正文 14px | 未实现（找不到元素渲染点） |
| R-GI-280 | G-H5-02 | 无障碍 | 读屏进入该元素时 | 读出该句 | — | 读屏文本 = 「所有技能均为最新版本」 | 未实现（找不到元素渲染点） |
| R-GI-281 | G-H5-02 | 关联 | 插件列表为空时 | 空态时批量按钮不出现 | — | 屏上 空态文字 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GI-282 | G-H5-03 | 状态 | 未观察（条件）：有可更新项 | 行首 40px 技能图标 + 技能名（14px） | — | 图标 40、名称单行 | 未实现（找不到元素渲染点） |
| R-GI-283 | G-H5-03 | 内容 | 图标缺失 | 用占位图标兜底，无破图 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 未实现（找不到元素渲染点） |
| R-GI-284 | G-H5-03 | 内容 | 名称超长 | truncate + `title` 完整名 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（找不到元素渲染点） |
| R-GI-285 | G-H5-03 | 样式 | 行在屏上时 | 行高约 56；图标与文字垂直居中 | — | 判据 = 行高 UI 正文 17.5px；其余 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0） | 未实现（找不到元素渲染点） |
| R-GI-286 | G-H5-03 | 无障碍 | 读屏进入该元素时 | 播报技能名 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GI-287 | G-H5-03 | 内容 | 名称在屏上时 | 技能名 14px，单行 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（找不到元素渲染点） |
| R-GI-288 | G-H5-03 | 状态 | 安装进行中时 | 行结构不变，只换按钮为旋转 | — | 判据 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（找不到元素渲染点） |
| R-GI-289 | G-H5-03 | 无障碍 | 读屏进入该元素时 | 播报技能名 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GI-290 | G-H5-03 | 关联 | 名称行与版本行上下排列时 | 名称与版本行上下两行，左缘对齐 | — | 判据 = 间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px；其余 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 未实现（找不到元素渲染点） |
| R-GI-291 | G-H5-04 | 内容 | 未观察（条件）：有记录版本 | 版本行「v{旧} → v{新}」；旧版本未知时只显示「v{新}」 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM） | 未实现（找不到元素渲染点） |
| R-GI-292 | G-H5-04 | 样式 | 版本行在屏上时 | 12px 次要色、与名称左缘对齐 | — | 判据 = 字号 UI 正文 12.25px；内边距 左缘 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28） | 未实现（找不到元素渲染点） |
| R-GI-293 | G-H5-04 | 无障碍 | 读屏进入该元素时 | 读出两版本 | — | 读屏文本 = 「v{旧} → v{新}」 | 未实现（找不到元素渲染点） |
| R-GI-294 | G-H5-04 | 异常 | 版本号非语义化 | 原样显示 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GI-295 | G-H5-04 | 内容 | 版本行在屏上时 | 「v{旧} → v{新}」或「v{新}」 | — | 判据 = 读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM） | 未实现（找不到元素渲染点） |
| R-GI-296 | G-H5-04 | 异常 | 版本缺失 | 不显示该行 | — | 屏上 该行 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GI-297 | G-H5-04 | 样式 | 版本行在屏上时 | 12px 次要色、与名称左缘对齐 | — | 判据 = 字号 UI 正文 12.25px | 未实现（找不到元素渲染点） |
| R-GI-298 | G-H5-04 | 无障碍 | 读屏进入该元素时 | 读出两版本 | — | 读屏文本 = 「v{旧} → v{新}」 | 未实现（找不到元素渲染点） |
| R-GI-299 | G-H5-04 | 状态 | 该技能安装中 | 版本行保持显示 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GI-300 | G-H5-05 | 状态 | 未观察（条件）：有可更新项 | 行尾次级按钮「更新」，安装中旋转 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/catalog/branch.go:Catalog.ApplyBranchUpdate「更新」 |
| R-GI-301 | G-H5-05 | 交互 | 鼠标点击该元素时 | 只更新该技能；成功后行消失 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 未实现（找不到元素渲染点） |
| R-GI-302 | G-H5-05 | 状态 | 安装中 / 全部更新中 | 禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（找不到元素渲染点） |
| R-GI-303 | G-H5-05 | 异常 | 更新失败 | 行保留并给原因 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GI-304 | G-H5-05 | 样式 | 次级按钮在屏上时 | `outline`、高 28、图标 14 | — | 判据 = 控件尺寸 按钮高 次 28；其余 = 图标 16×16 盒、viewBox 24、描边 1.5、线端 round（IC-BOX-16 / IC-STROKE-1.5 / IC-CAP-ROUND），2× 截图彩边 0 像素（IC-EDGE-0） | 未实现（找不到元素渲染点） |
| R-GI-305 | G-H5-05 | 无障碍 | 读屏进入该元素时 | 读作按钮「更新 {技能名}」 | — | 读屏文本 = 「更新 {技能名}」 | 未实现（找不到元素渲染点） |
| R-GI-306 | G-H5-05 | 内容 | 按钮文案渲染后 | 恒为「更新」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/catalog/branch.go:Catalog.ApplyBranchUpdate「更新」 |
| R-GI-307 | G-H5-05 | 状态 | 安装进行中时 | 按钮禁用并旋转 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（找不到元素渲染点） |
| R-GI-308 | G-H5-06 | 状态 | 未观察（条件）：有可更新项 | 主按钮逐字「全部更新（{count}）」 | — | 屏上形如「全部更新（3）」 | 未实现（找不到元素渲染点） |
| R-GI-309 | G-H5-06 | 交互 | 鼠标点击该元素时 | 依次更新全部技能 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（找不到元素渲染点） |
| R-GI-310 | G-H5-06 | 状态 | 更新进行中时 | 禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（找不到元素渲染点） |
| R-GI-311 | G-H5-06 | 异常 | 部分失败 | 失败行保留、成功行消失 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；可见时出现次数 1；消失后 0 且不占位（ST-VISIBLE） | 未实现（找不到元素渲染点） |
| R-GI-312 | G-H5-06 | 样式 | 主按钮在屏上时 | 实心主色、白字、高 28 | — | 判据 = 底色 主色 明 #339751 / 暗 #63cb7b；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（找不到元素渲染点） |
| R-GI-313 | G-H5-06 | 无障碍 | 读屏进入该元素时 | 读作按钮「全部更新，共 N 个」 | — | 读屏文本 = 「全部更新，共 N 个」 | 未实现（找不到元素渲染点） |
| R-GI-314 | G-H5-06 | 内容 | 按钮文案渲染后 | 「全部更新（{count}）」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GI-315 | G-H5-06 | 状态 | 数量为 0 | 不显示 | — | 屏上 主按钮 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GI-316 | G-H5-06 | 异常 | 中途失败 | 失败行保留，按钮恢复可点 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（找不到元素渲染点） |

### 8.6 导入 Chrome Cookie（G-H6-01–G-H6-12）

| 需求 ID | 元素 ID | 维度 | 场景 | 期望 | 异常 | 验收 | 状态 |
|---|---|---|---|---|---|---|---|
| R-GI-317 | G-H6-01 | 内容 | 未观察（条件）：设置 → 通用 → 浏览器数据 →「导入…」 | 标题逐字「导入 Chrome Cookie」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/settings_general_screen.go:generalBrowserCard「导入 Chrome Cookie」 |
| R-GI-318 | G-H6-01 | 样式 | 标题在屏上时 | 16px 级半粗；圆角 + 阴影 + 遮罩 | — | 判据 = 字号 对话框标题 15.75px；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px | 已实现（映射 = G-H6-01→internal/ui/settings_general_screen.go:generalImportDialog；标记 = TextColor@screenDialogPanel） |
| R-GI-319 | G-H6-01 | 无障碍 | 读屏进入该元素时 | 作为对话框名读出 | — | 读屏文本 = 「导入 Chrome Cookie」 | 已实现（映射 = G-H6-01→internal/ui/settings_general_screen.go:generalImportDialog；标记 = Role@modalShell） |
| R-GI-320 | G-H6-01 | 异常 | 本机无 Chrome | 仍打开（内容给 G-H6-06 空态） | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-H6-01→internal/ui/settings_general_screen.go:generalImportDialog |
| R-GI-321 | G-H6-01 | 窄窗 | 窗口收窄到最窄宽度时 | 标题单行，描述折行，按钮行完整 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 未实现（映射到位但没有该维度的标记）：G-H6-01→internal/ui/settings_general_screen.go:generalImportDialog |
| R-GI-322 | G-H6-01 | 内容 | 标题在屏上时 | 恒为「导入 Chrome Cookie」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/settings_general_screen.go:generalBrowserCard「导入 Chrome Cookie」 |
| R-GI-323 | G-H6-01 | 异常 | 浏览器数据设置未打开 | 入口不存在，自然无弹窗 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-H6-01→internal/ui/settings_general_screen.go:generalImportDialog |
| R-GI-324 | G-H6-01 | 无障碍 | 读屏进入该元素时 | 作为对话框名读出 | — | 读屏文本 = 「导入 Chrome Cookie」 | 已实现（映射 = G-H6-01→internal/ui/settings_general_screen.go:generalImportDialog；标记 = Role@modalShell） |
| R-GI-325 | G-H6-02 | 内容 | 未观察（条件）：所选配置可导入数 ≠ 0 | 描述逐字「此操作会替换内置浏览器中的全部非分区 Cookie，并重新加载已打开的页面。每个 Cookie 都会单独尝试导入，因此个别 Cookie 可能失败；部分网站可能需要重新登录。」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H6-02→internal/ui/settings_general_screen.go:generalImportDialog；标记 = 文案「导入 Chrome Cookie」等 4 条） |
| R-GI-326 | G-H6-02 | 状态 | 可导入数为 0 | 描述不出现（无内容可替换） | — | 屏上 描述 计数 = 0（无内容可替换） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GI-327 | G-H6-02 | 样式 | 描述在屏上时 | 14px 级次要色、折行、行高 1.5 | — | 判据 = 字号 正文 14px；行高 说明文字 14px | 已实现（映射 = G-H6-02→internal/ui/settings_general_screen.go:generalImportDialog；标记 = TextColor@screenDialogPanel） |
| R-GI-328 | G-H6-02 | 无障碍 | 读屏进入该元素时 | 读出整段（较长，允许分段播报） | — | 读屏文本 = 「此操作会替换内置浏览器中的全部非分区 Cookie，并重新加载已打开的页面。」（较长，允许分段播报） | 已实现（映射 = G-H6-02→internal/ui/settings_general_screen.go:generalImportDialog；标记 = Role@modalShell） |
| R-GI-329 | G-H6-02 | 异常 | 描述超长（窄窗） | 折行，按钮行仍可见 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 未实现（映射到位但没有该维度的标记）：G-H6-02→internal/ui/settings_general_screen.go:generalImportDialog |
| R-GI-330 | G-H6-02 | 内容 | 描述在屏上时 | 四句要点：替换范围、重载页面、逐个导入、可能需重登 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H6-02→internal/ui/settings_general_screen.go:generalImportDialog；标记 = 文案「导入 Chrome Cookie」等 4 条） |
| R-GI-331 | G-H6-02 | 状态 | 可导入数为 0 | 描述不出现（无内容可替换） | — | 屏上 描述 计数 = 0（无内容可替换） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GI-332 | G-H6-02 | 样式 | 描述在屏上时 | 14px 次要色、折行 | — | 判据 = 字号 正文 14px | 已实现（映射 = G-H6-02→internal/ui/settings_general_screen.go:generalImportDialog；标记 = TextColor@screenDialogPanel） |
| R-GI-333 | G-H6-03 | 状态 | 未观察（条件）：扫描进行中 | 加载行：旋转图标 + 「正在扫描 Chrome 配置…」（中文省略号），最小高 96 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-H6-03→internal/ui/settings_general_screen.go:generalImportDialog；标记 = Disabled） |
| R-GI-334 | G-H6-03 | 动画 | 扫描进行中时 | 旋转连续；扫描结束后整行消失 | 减少动效下静态 | 判据 = 动效 布局 250ms / 颜色 150ms / 折叠 300ms；缓动 cubic-bezier(0.16, 1, 0.3, 1) / cubic-bezier(0.4, 0, 0.2, 1) | 未实现（映射到位但没有该维度的标记）：G-H6-03→internal/ui/settings_general_screen.go:generalImportDialog |
| R-GI-335 | G-H6-03 | 关联 | 配置扫描进行中时 | 扫描中「导入」禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/app/extensions.go:extensionReaders.PreviewToolServerImport「导入」 |
| R-GI-336 | G-H6-03 | 样式 | 加载行在屏上时 | 14px 次要色、居左或居中一致；最小高 96 | — | 判据 = 字号 正文 14px；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-H6-03→internal/ui/settings_general_screen.go:generalImportDialog；标记 = TextColor@screenDialogPanel） |
| R-GI-337 | G-H6-03 | 无障碍 | 读屏进入该元素时 | `role=status` 播报一次 | — | 播报次数 = 1 | 已实现（映射 = G-H6-03→internal/ui/settings_general_screen.go:generalImportDialog；标记 = Role@modalShell） |
| R-GI-338 | G-H6-03 | 内容 | 扫描行 | 「正在扫描 Chrome 配置…」+ 旋转图标 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H6-03→internal/ui/settings_general_screen.go:generalImportDialog；标记 = 文案「导入 Chrome Cookie」等 4 条） |
| R-GI-339 | G-H6-03 | 状态 | 扫描完成 | 整行消失，换成配置列表 | — | 屏上 加载行 计数 = 0 | 已实现（映射 = G-H6-03→internal/ui/settings_general_screen.go:generalImportDialog；标记 = Disabled） |
| R-GI-340 | G-H6-03 | 无障碍 | 读屏进入该元素时 | `role=status` 播报一次 | — | 播报次数 = 1 | 已实现（映射 = G-H6-03→internal/ui/settings_general_screen.go:generalImportDialog；标记 = Role@modalShell） |
| R-GI-341 | G-H6-04 | 内容 | 未观察（条件）：扫描完成 | 分组标签逐字「Chrome 配置」（12px 中等灰） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/unavailable.go:noteBrowserNoImport「Chrome 配置」 |
| R-GI-342 | G-H6-04 | 状态 | 无可用配置时 | 标签不出现（只留 G-H6-06 空态） | — | 屏上 分组标签 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GI-343 | G-H6-04 | 样式 | 分组标签在屏上时 | 12px 次要色、与单选行文字左缘对齐 | — | 判据 = 字号 UI 正文 12.25px；内边距 左缘 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28） | 未实现（找不到元素渲染点） |
| R-GI-344 | G-H6-04 | 无障碍 | 读屏进入该元素时 | 作为分组标题读出 | — | 读屏文本 = 「Chrome 配置」 | 未实现（找不到元素渲染点） |
| R-GI-345 | G-H6-04 | 异常 | 配置列表为 1 条 | 标签仍显示（不因只有一条而省略） | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（找不到元素渲染点） |
| R-GI-346 | G-H6-04 | 内容 | 标签在屏上时 | 恒为「Chrome 配置」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/unavailable.go:noteBrowserNoImport「Chrome 配置」 |
| R-GI-347 | G-H6-04 | 状态 | 无可用配置时 | 标签不出现 | — | 屏上 分组标签 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GI-348 | G-H6-04 | 样式 | 标签在屏上时 | 12px 中等灰、与单选行左缘对齐 | — | 判据 = 字号 说明文字 10.5px；内边距 左缘 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28） | 未实现（找不到元素渲染点） |
| R-GI-349 | G-H6-05 | 状态 | 未观察（条件）：存在配置 | 每个配置一行：自绘单选圈 + 配置名 + 12px 能力标签；默认选中首个可用配置 | — | 判据 = 底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d；其余 = 选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED）；包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-H6-05→internal/ui/settings_general_screen.go:generalImportDialog；标记 = Disabled） |
| R-GI-350 | G-H6-05 | 交互 | 点击另一配置 | 切换选中；下方计数随所选配置重新读取（G-H6-07/G-H6-08） | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-H6-05→internal/ui/settings_general_screen.go:generalImportDialog；标记 = Pressed@screenButton） |
| R-GI-351 | G-H6-05 | 内容 | 配置名超长 | truncate + `title` 完整名 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-H6-05→internal/ui/settings_general_screen.go:generalImportDialog；标记 = 文案「导入 Chrome Cookie」等 4 条） |
| R-GI-352 | G-H6-05 | 内容 | 能力标签 | 显示该配置是否可读（如「可读」/「需授权」），不显示原始枚举 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GI-353 | G-H6-05 | 样式 | 单选行 | 行高 40、单选圈 16、选中主色 | — | 判据 = 行高 空态主标题 28px；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-H6-05→internal/ui/settings_general_screen.go:generalImportDialog；标记 = TextColor@screenDialogPanel） |
| R-GI-354 | G-H6-05 | 无障碍 | 读屏进入该元素时 | 读作单选项，含名称与能力标签 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H6-05→internal/ui/settings_general_screen.go:generalImportDialog；标记 = Role@modalShell） |
| R-GI-355 | G-H6-05 | 内容 | 配置行 | 单选圈 + 配置名 + 能力标签 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 已实现（映射 = G-H6-05→internal/ui/settings_general_screen.go:generalImportDialog；标记 = 文案「导入 Chrome Cookie」等 4 条） |
| R-GI-356 | G-H6-05 | 状态 | 只有 1 个配置 | 仍显示单选圈（可选中态），不换成纯文本 | — | 判据 = 底色 选中底 明 #f5f5f5 / 暗 #262626；字色 主文字 明 #0a0a0a / 暗 #ececec；其余 = 选中底 明 #f5f5f5 / 暗 #262626 + 左缘 2px 主色（ST-SELECTED） | 已实现（映射 = G-H6-05→internal/ui/settings_general_screen.go:generalImportDialog；标记 = Disabled） |
| R-GI-357 | G-H6-06 | 内容 | 未观察（条件）：无配置且无错误 | 空态文字逐字「未找到 Google Chrome 配置。」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GI-358 | G-H6-06 | 状态 | 有配置 | 空态不出现 | — | 屏上 空态文字 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GI-359 | G-H6-06 | 关联 | 无可导入配置时 | 无配置时「导入」禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/app/extensions.go:extensionReaders.PreviewToolServerImport「导入」 |
| R-GI-360 | G-H6-06 | 样式 | 空态在屏上时 | 14px 次要色、居中 | — | 判据 = 字号 正文 14px；其余 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（找不到元素渲染点） |
| R-GI-361 | G-H6-06 | 无障碍 | 读屏进入该元素时 | 读出该句 | — | 读屏文本 = 「未找到 Google Chrome 配置。」 | 未实现（找不到元素渲染点） |
| R-GI-362 | G-H6-06 | 内容 | 空态在屏上时 | 恒为「未找到 Google Chrome 配置。」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GI-363 | G-H6-06 | 样式 | 空态在屏上时 | 14px 次要色、居中 | — | 判据 = 字号 正文 14px | 未实现（找不到元素渲染点） |
| R-GI-364 | G-H6-06 | 状态 | 扫描进行中时 | 不显示（还没结论） | — | 屏上 空态文字 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GI-365 | G-H6-07 | 状态 | 未观察（条件）：切换配置后的预览读取 | 加载行：旋转图标 + 「正在读取所选配置…」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GI-366 | G-H6-07 | 动画 | 读取中 | 旋转连续；读取完成后换成计数块 | 减少动效下静态 | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（找不到元素渲染点） |
| R-GI-367 | G-H6-07 | 关联 | 读取进行中、旧计数仍在上一次位置时 | 读取中不显示计数块（避免闪旧值） | — | 屏上 计数块 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GI-368 | G-H6-07 | 样式 | 加载行在屏上时 | 14px 次要色、左内边距与计数块一致 | — | 判据 = 字号 正文 14px；内边距 左缘 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28） | 未实现（找不到元素渲染点） |
| R-GI-369 | G-H6-07 | 无障碍 | 读屏进入该元素时 | `role=status` 播报一次 | — | 播报次数 = 1 | 未实现（找不到元素渲染点） |
| R-GI-370 | G-H6-07 | 内容 | 读取行 | 「正在读取所选配置…」+ 旋转图标 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GI-371 | G-H6-07 | 状态 | 读取完成 | 换成计数块 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（找不到元素渲染点） |
| R-GI-372 | G-H6-07 | 无障碍 | 读屏进入该元素时 | `role=status` 播报一次 | — | 播报次数 = 1 | 未实现（找不到元素渲染点） |
| R-GI-373 | G-H6-08 | 内容 | 未观察（条件）：预览已返回 | 计数块逐字「可导入 {importable} · 会话 {session} · 已过期 {expired} · 分区 {partitioned}」，中点分隔 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H6-08→internal/ui/settings_general_screen.go:generalImportDialog；标记 = 文案「导入 Chrome Cookie」等 4 条） |
| R-GI-374 | G-H6-08 | 内容 | 某项为 0 | 仍显示该项与 0，不省略 | — | 判据 = 行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H6-08→internal/ui/settings_general_screen.go:generalImportDialog；标记 = 文案「导入 Chrome Cookie」等 4 条） |
| R-GI-375 | G-H6-08 | 样式 | 计数块在屏上时 | 边框 + 浅底、12px、内边距 8、圆角 6 | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d；字号 UI 正文 12.25px；其余 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H6-08→internal/ui/settings_general_screen.go:generalImportDialog；标记 = TextColor@screenDialogPanel） |
| R-GI-376 | G-H6-08 | 无障碍 | 读屏进入该元素时 | 读出四段计数 | — | 读屏文本 = 「可导入 {importable} · 会话 {s」 | 已实现（映射 = G-H6-08→internal/ui/settings_general_screen.go:generalImportDialog；标记 = Role@modalShell） |
| R-GI-377 | G-H6-08 | 异常 | 预览失败 | 计数块不出现，改走 G-H6-10 错误块 | — | 屏上 计数块 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GI-378 | G-H6-08 | 内容 | 计数块在屏上时 | 四段：可导入 / 会话 / 已过期 / 分区 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H6-08→internal/ui/settings_general_screen.go:generalImportDialog；标记 = 文案「导入 Chrome Cookie」等 4 条） |
| R-GI-379 | G-H6-08 | 状态 | 可导入为 0 | 仍显示四段（其中一段为 0） | — | 屏上 计数块 计数 = 1（其中一段为 0 时） | 已实现（映射 = G-H6-08→internal/ui/settings_general_screen.go:generalImportDialog；标记 = Disabled） |
| R-GI-380 | G-H6-08 | 样式 | 计数块在屏上时 | 边框 + 浅底 + 12px | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；字号 UI 正文 12.25px；其余 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H6-08→internal/ui/settings_general_screen.go:generalImportDialog；标记 = TextColor@screenDialogPanel） |
| R-GI-381 | G-H6-09a | 内容 | 未观察（条件）：导入结束且失败数为 0 | 结果块：对勾图标 + 「已导入 {imported} · 失败 {failed} · 已跳过…」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GI-382 | G-H6-09a | 状态 | 有失败项 | 改走 G-H6-09b（告警图标） | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（找不到元素渲染点） |
| R-GI-383 | G-H6-09a | 交互 | 导入结束后 | 主按钮转「关闭」；可关闭弹窗 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「关闭」 |
| R-GI-384 | G-H6-09a | 样式 | 结果块在屏上时 | 对勾为成功色、14px 文字、无底色 | — | 判据 = 字色 次文字 明 #737373 / 暗 #8a8a8a；字号 UI 正文 12.25px | 未实现（找不到元素渲染点） |
| R-GI-385 | G-H6-09a | 无障碍 | 读屏进入该元素时 | 播报导入结果 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 未实现（找不到元素渲染点） |
| R-GI-386 | G-H6-09a | 内容 | 结果文案渲染后 | 「已导入 {imported} · 失败 {failed} · 已跳过…」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 未实现（找不到元素渲染点） |
| R-GI-387 | G-H6-09a | 状态 | 有失败 | 改走告警分支 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（找不到元素渲染点） |
| R-GI-388 | G-H6-09a | 样式 | 结果块在屏上时 | 对勾成功色 + 14px 文字 | — | 判据 = 字色 次文字 明 #737373 / 暗 #8a8a8a | 未实现（找不到元素渲染点） |
| R-GI-389 | G-H6-09b | 内容 | 未观察（条件）：导入结束且失败数 >0 | 结果块：告警图标 + 计数 + 「部分 Cookie 未能导入，部分网站可能需要重新登录。」（12px 警示色） | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（找不到元素渲染点） |
| R-GI-390 | G-H6-09b | 状态 | 失败数为 0 | 改走 G-H6-09a | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（找不到元素渲染点） |
| R-GI-391 | G-H6-09b | 样式 | 结果块在屏上时 | 告警图标琥珀色、14px 主文 + 12px 说明 | — | 判据 = 色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a；字号 UI 正文 12.25px | 未实现（找不到元素渲染点） |
| R-GI-392 | G-H6-09b | 无障碍 | 读屏进入该元素时 | 播报结果与说明 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；读屏顺序 = 渲染顺序、序位 = 父容器内第 n 个（SR-ORDER-DOM）；文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；焦点环 2px + 外扩 1px、明 #a1a1a1 / 暗 #63cb7b、每次聚焦可见 1 次（ST-FOCUS） | 未实现（找不到元素渲染点） |
| R-GI-393 | G-H6-09b | 交互 | 导入结束后 | 主按钮转「关闭」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「关闭」 |
| R-GI-394 | G-H6-09b | 内容 | 结果文案渲染后 | 计数 + 「部分 Cookie 未能导入，部分网站可能需要重新登录。」 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（找不到元素渲染点） |
| R-GI-395 | G-H6-09b | 状态 | 无失败 | 改走成功分支 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 未实现（找不到元素渲染点） |
| R-GI-396 | G-H6-09b | 样式 | 结果块在屏上时 | 告警琥珀色 + 14px 主文 + 12px 说明 | — | 判据 = 色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a；字号 UI 正文 12.25px | 未实现（找不到元素渲染点） |
| R-GI-397 | G-H6-10 | 状态 | 未观察（条件）：导入或读取失败 | 错误块：危险色消息 + 等宽诊断行 + 次级按钮「重试」 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（逐字）：internal/ui/app.go:App.actionErrorView「重试」 |
| R-GI-398 | G-H6-10 | 交互 | 鼠标点击「重试」时 | 重新读取/导入；按钮短暂禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（找不到元素渲染点） |
| R-GI-399 | G-H6-10 | 内容 | 诊断行很长 | 等宽、可折行、可复制（便于报障） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 未实现（找不到元素渲染点） |
| R-GI-400 | G-H6-10 | 样式 | 错误块在屏上时 | 危险色 14px + 等宽 12px 诊断行 + 按钮高 28 | — | 判据 = 字号 UI 正文 12.25px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 未实现（找不到元素渲染点） |
| R-GI-401 | G-H6-10 | 无障碍 | 读屏进入该元素时 | `role=alert` 播报消息，诊断行可单独读到 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；色值取 token：主色 #349651/#63cb7b、正文 #0a0a0a/#ececec、次文 #737373/#8a8a8a、边框 #e5e5e5/#2e2e2e、警示 #e17100/#ffb900，偏差 0（CL-ACCENT / CL-TEXT / CL-TEXT-2 / CL-BORDER / CL-WARN） | 未实现（找不到元素渲染点） |
| R-GI-402 | G-H6-10 | 内容 | 错误块在屏上时 | 危险色消息 + 等宽诊断行 + 「重试」 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（逐字）：internal/ui/app.go:App.actionErrorView「重试」 |
| R-GI-403 | G-H6-10 | 状态 | 重试进行中时 | 「重试」禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/ui/app.go:App.actionErrorView「重试」 |
| R-GI-404 | G-H6-10 | 样式 | 诊断行在屏上时 | 等宽 12px、可折行 | — | 判据 = 字号 UI 正文 12.25px；其余 = 字号 12.25 / 行高 18 / 字重 400，说明文字 10.5 / 14（TX-TYPE-BODY / TX-TYPE-MICRO）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 未实现（找不到元素渲染点） |
| R-GI-405 | G-H6-11 | 状态 | 未观察（条件）：导入未结束时 | 次按钮「取消」（`outline`） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GI-406 | G-H6-11 | 状态 | 已导入后 | 同一按钮文案变「关闭」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「关闭」 |
| R-GI-407 | G-H6-11 | 交互 | 鼠标点击该元素时 | 未结束时取消导入（回到未导入状态）；结束后关闭弹窗 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H6-11→internal/ui/settings_general_screen.go:generalImportDialog；标记 = Pressed@screenButton） |
| R-GI-408 | G-H6-11 | 异常 | 导入进行中取消 | 已写入的部分不回滚，给提示（按实现一致，须固定） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-H6-11→internal/ui/settings_general_screen.go:generalImportDialog |
| R-GI-409 | G-H6-11 | 样式 | 次按钮在屏上时 | 描边、高 28 | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-H6-11→internal/ui/settings_general_screen.go:generalImportDialog；标记 = TextColor@screenDialogPanel） |
| R-GI-410 | G-H6-11 | 无障碍 | 读屏进入该元素时 | 读作按钮「取消」/「关闭」 | — | 读屏文本 = 「取消」 / 「关闭」 | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GI-411 | G-H6-11 | 内容 | 两态文案先后出现时 | 未结束「取消」、已导入「关闭」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GI-412 | G-H6-11 | 状态 | 导入进行中时 | 「取消」可用（可中止） | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GI-413 | G-H6-11 | 异常 | 中止后 | 已写入部分不回滚，给提示 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-H6-11→internal/ui/settings_general_screen.go:generalImportDialog |
| R-GI-414 | G-H6-12 | 状态 | 未观察（条件）：已取到预览 | 主按钮「导入」可用；未取到预览/导入中/读取中/扫描中禁用 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/app/extensions.go:extensionReaders.PreviewToolServerImport「导入」 |
| R-GI-415 | G-H6-12 | 内容 | 导入进行中时 | 文案变「正在导入 Cookie…」，前置旋转图标 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H6-12→internal/ui/settings_general_screen.go:generalImportDialog；标记 = 文案「导入 Chrome Cookie」等 4 条） |
| R-GI-416 | G-H6-12 | 内容 | 阶段文案 | 依次可能出现「正在关闭浏览器页面…」「正在更新浏览器数据…」「正在验证浏览器数据…」「正在恢复原有登录状态…」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H6-12→internal/ui/settings_general_screen.go:generalImportDialog；标记 = 文案「导入 Chrome Cookie」等 4 条） |
| R-GI-417 | G-H6-12 | 交互 | 鼠标点击该元素时 | 开始导入；结束后按钮转「关闭」、结果块出现 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「关闭」 |
| R-GI-418 | G-H6-12 | 异常 | 导入失败 | 错误块出现，按钮恢复可用 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-H6-12→internal/ui/settings_general_screen.go:generalImportDialog |
| R-GI-419 | G-H6-12 | 样式 | 主按钮在屏上时 | 实心主色、白字、高 28；禁用降透明 | — | 判据 = 底色 主色 明 #339751 / 暗 #63cb7b；其余 = 色值取 token：主色 #349651/#63cb7b、正文 #0a0a0a/#ececec、次文 #737373/#8a8a8a、边框 #e5e5e5/#2e2e2e、警示 #e17100/#ffb900，偏差 0（CL-ACCENT / CL-TEXT / CL-TEXT-2 / CL-BORDER / CL-WARN） | 已实现（映射 = G-H6-12→internal/ui/settings_general_screen.go:generalImportDialog；标记 = TextColor@screenDialogPanel） |
| R-GI-420 | G-H6-12 | 无障碍 | 读屏进入该元素时 | 读作按钮「导入」，阶段变化时播报当前阶段 | — | 读屏文本 = 「导入」 | 已实现（逐字）：internal/app/extensions.go:extensionReaders.PreviewToolServerImport「导入」 |
| R-GI-421 | G-H6-12 | 内容 | 可用态文案渲染后 | 恒为「导入」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/app/extensions.go:extensionReaders.PreviewToolServerImport「导入」 |
| R-GI-422 | G-H6-12 | 异常 | 导入失败 | 错误块出现、按钮恢复 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-H6-12→internal/ui/settings_general_screen.go:generalImportDialog |
| R-GI-423 | G-H6-12 | 样式 | 主按钮在屏上时 | 实心主色、高 28 | — | 判据 = 底色 主色 明 #339751 / 暗 #63cb7b；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-H6-12→internal/ui/settings_general_screen.go:generalImportDialog；标记 = TextColor@screenDialogPanel） |

### 8.7 清理浏览器数据（G-H7-01–G-H7-06）

| 需求 ID | 元素 ID | 维度 | 场景 | 期望 | 异常 | 验收 | 状态 |
|---|---|---|---|---|---|---|---|
| R-GI-424 | G-H7-01 | 内容 | 未观察（条件）：设置 → 通用 → 浏览器数据 →「清理数据…」 | 标题逐字「清理浏览器数据」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/settings_general_screen.go:generalBrowserCard「清理浏览器数据」 |
| R-GI-425 | G-H7-01 | 样式 | 标题在屏上时 | 16px 级半粗；圆角 + 阴影 + 遮罩 | — | 判据 = 字号 对话框标题 15.75px；圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px | 已实现（映射 = G-H7-01→internal/ui/settings_general_screen.go:generalClearDialog；标记 = TextColor@screenDialogPanel） |
| R-GI-426 | G-H7-01 | 无障碍 | 读屏进入该元素时 | 作为对话框名读出 | — | 读屏文本 = 「清理浏览器数据」 | 已实现（映射 = G-H7-01→internal/ui/settings_general_screen.go:generalClearDialog；标记 = Role@modalShell） |
| R-GI-427 | G-H7-01 | 异常 | 内置浏览器从未使用 | 仍可打开并清理（无副作用） | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-H7-01→internal/ui/settings_general_screen.go:generalClearDialog |
| R-GI-428 | G-H7-01 | 窄窗 | 窗口收窄到最窄宽度时 | 标题单行，描述折行，按钮行完整 | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 未实现（映射到位但没有该维度的标记）：G-H7-01→internal/ui/settings_general_screen.go:generalClearDialog |
| R-GI-429 | G-H7-01 | 内容 | 标题在屏上时 | 恒为「清理浏览器数据」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/settings_general_screen.go:generalBrowserCard「清理浏览器数据」 |
| R-GI-430 | G-H7-01 | 异常 | 无浏览器数据 | 仍可打开（清理无副作用） | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-H7-01→internal/ui/settings_general_screen.go:generalClearDialog |
| R-GI-431 | G-H7-01 | 无障碍 | 读屏进入该元素时 | 作为对话框名读出 | — | 读屏文本 = 「清理浏览器数据」 | 已实现（映射 = G-H7-01→internal/ui/settings_general_screen.go:generalClearDialog；标记 = Role@modalShell） |
| R-GI-432 | G-H7-02 | 内容 | 未观察（条件）：浏览器数据清理确认框打开时 | 描述逐字「此操作只会清除内置浏览器中的登录状态、Cookie、本地存储、IndexedDB、Service Worker 和缓存。浏览器页面会关闭并以空白页重新打开。」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H7-02→internal/ui/settings_general_screen.go:generalClearDialog；标记 = 文案「清理浏览器数据」等 4 条） |
| R-GI-433 | G-H7-02 | 内容 | 描述较长 | 折行，不截断（六类数据必须全部列出） | — | 判据 = 相对位置 = 同级相邻关系，纵向间距取 3.5/7/10.5/14/21 档（AL-STACK）、左缘对齐文字列（AL-LEFT-40 或 AL-LEFT-14），偏差 ≤ 1px（AL-TOL-1）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-H7-02→internal/ui/settings_general_screen.go:generalClearDialog；标记 = 文案「清理浏览器数据」等 4 条） |
| R-GI-434 | G-H7-02 | 样式 | 描述在屏上时 | 14px 级次要色、行高 1.5 | — | 判据 = 字号 正文 14px；行高 说明文字 14px | 已实现（映射 = G-H7-02→internal/ui/settings_general_screen.go:generalClearDialog；标记 = TextColor@screenDialogPanel） |
| R-GI-435 | G-H7-02 | 无障碍 | 读屏进入该元素时 | 读出整段 | — | 读屏文本 = 「此操作只会清除内置浏览器中的登录状态、Cooki」 | 已实现（映射 = G-H7-02→internal/ui/settings_general_screen.go:generalClearDialog；标记 = Role@modalShell） |
| R-GI-436 | G-H7-02 | 异常 | 描述被裁 | 不允许：按钮行下移而非裁掉描述 | — | 判据 = 一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-H7-02→internal/ui/settings_general_screen.go:generalClearDialog |
| R-GI-437 | G-H7-02 | 内容 | 描述在屏上时 | 列出六类数据 + 页面会关闭重开 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数 = 1、超宽以「…」收尾且横向溢出 0（TX-LINE-1 / TX-ELLIPSIS / SC-OVERFLOW-0） | 已实现（映射 = G-H7-02→internal/ui/settings_general_screen.go:generalClearDialog；标记 = 文案「清理浏览器数据」等 4 条） |
| R-GI-438 | G-H7-02 | 状态 | 清理进行中时 | 描述保持（不换成进度） | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-H7-02→internal/ui/settings_general_screen.go:generalClearDialog；标记 = Disabled） |
| R-GI-439 | G-H7-02 | 样式 | 描述在屏上时 | 14px 次要色、折行 | — | 判据 = 字号 正文 14px | 已实现（映射 = G-H7-02→internal/ui/settings_general_screen.go:generalClearDialog；标记 = TextColor@screenDialogPanel） |
| R-GI-440 | G-H7-03 | 内容 | 未观察（条件）：清理成功 | 成功行：对勾图标 + 逐字「内置浏览器数据已清理。」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H7-03→internal/ui/settings_general_screen.go:generalClearDialog；标记 = 文案「清理浏览器数据」等 4 条） |
| R-GI-441 | G-H7-03 | 状态 | 清理未完成 | 成功行不出现 | — | 屏上 成功行 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GI-442 | G-H7-03 | 关联 | 清理成功后 | 成功行出现后，次按钮转「关闭」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「关闭」 |
| R-GI-443 | G-H7-03 | 样式 | 成功行在屏上时 | 对勾成功色、14px 文字、无底色 | — | 判据 = 字色 成功 明 #009966 / 暗 #00d492；字号 UI 正文 12.25px | 已实现（映射 = G-H7-03→internal/ui/settings_general_screen.go:generalClearDialog；标记 = TextColor@screenDialogPanel） |
| R-GI-444 | G-H7-03 | 无障碍 | 读屏进入该元素时 | `role=status` 播报一次 | — | 播报次数 = 1 | 已实现（映射 = G-H7-03→internal/ui/settings_general_screen.go:generalClearDialog；标记 = Role@modalShell） |
| R-GI-445 | G-H7-03 | 内容 | 成功行在屏上时 | 对勾 + 「内置浏览器数据已清理。」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H7-03→internal/ui/settings_general_screen.go:generalClearDialog；标记 = 文案「清理浏览器数据」等 4 条） |
| R-GI-446 | G-H7-03 | 状态 | 清理进行中时 | 不显示 | — | 屏上 成功行 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GI-447 | G-H7-03 | 无障碍 | 读屏进入该元素时 | `role=status` 播报一次 | — | 播报次数 = 1 | 已实现（映射 = G-H7-03→internal/ui/settings_general_screen.go:generalClearDialog；标记 = Role@modalShell） |
| R-GI-448 | G-H7-04 | 状态 | 未观察（条件）：清理失败 | 错误块：危险色消息 + 等宽诊断行 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-H7-04→internal/ui/settings_general_screen.go:generalClearDialog；标记 = Disabled） |
| R-GI-449 | G-H7-04 | 内容 | 诊断行很长 | 等宽、可折行、可复制 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-H7-04→internal/ui/settings_general_screen.go:generalClearDialog；标记 = 文案「清理浏览器数据」等 4 条） |
| R-GI-450 | G-H7-04 | 样式 | 错误块在屏上时 | 危险色 14px + 等宽 12px 诊断 | — | 判据 = 字号 UI 正文 12.25px；色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a | 已实现（映射 = G-H7-04→internal/ui/settings_general_screen.go:generalClearDialog；标记 = TextColor@screenDialogPanel） |
| R-GI-451 | G-H7-04 | 无障碍 | 读屏进入该元素时 | `role=alert` 播报消息与诊断 | — | 判据 = 可读名逐字等于屏上文案、图标按钮等无屏上文案的取期望列给的词；角色按作用取（SR-NAME / SR-ROLE）；色值取 token：主色 #349651/#63cb7b、正文 #0a0a0a/#ececec、次文 #737373/#8a8a8a、边框 #e5e5e5/#2e2e2e、警示 #e17100/#ffb900，偏差 0（CL-ACCENT / CL-TEXT / CL-TEXT-2 / CL-BORDER / CL-WARN） | 已实现（映射 = G-H7-04→internal/ui/settings_general_screen.go:generalClearDialog；标记 = Role@modalShell） |
| R-GI-452 | G-H7-04 | 交互 | 失败后重试 | 可再点「清理浏览器数据」，错误先清除 | — | 错误段恒 1 | 已实现（逐字）：internal/ui/settings_general_screen.go:generalBrowserCard「清理浏览器数据」 |
| R-GI-453 | G-H7-04 | 内容 | 错误块在屏上时 | 危险色消息 + 等宽诊断行 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-H7-04→internal/ui/settings_general_screen.go:generalClearDialog；标记 = 文案「清理浏览器数据」等 4 条） |
| R-GI-454 | G-H7-04 | 状态 | 清理进行中时 | 不显示 | — | 屏上 错误块 计数 = 0 | 需负向取证：期望/验收列为负向断言，需界面或代码事实反向取证 |
| R-GI-455 | G-H7-04 | 样式 | 诊断行在屏上时 | 等宽 12px、可折行 | — | 判据 = 字号 UI 正文 12.25px；其余 = 字号 12.25 / 行高 18 / 字重 400，说明文字 10.5 / 14（TX-TYPE-BODY / TX-TYPE-MICRO）；行数按内容折行（上限 2 行，内容列 ≤ 520 时按 TX-WRAP-520）、末行超宽以「…」收尾（TX-LINE-2 / TX-WRAP-520 / TX-ELLIPSIS） | 已实现（映射 = G-H7-04→internal/ui/settings_general_screen.go:generalClearDialog；标记 = TextColor@screenDialogPanel） |
| R-GI-456 | G-H7-05 | 状态 | 未观察（条件）：清理未完成 | 次按钮「取消」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GI-457 | G-H7-05 | 状态 | 清理完成后 | 同一按钮文案变「关闭」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「关闭」 |
| R-GI-458 | G-H7-05 | 交互 | 鼠标点击该元素时 | 未完成时取消（若已开始则按实现给出「无法中止」提示）；完成后关闭弹窗 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（映射 = G-H7-05→internal/ui/settings_general_screen.go:generalClearDialog；标记 = Pressed@screenButton） |
| R-GI-459 | G-H7-05 | 样式 | 次按钮在屏上时 | 描边、高 28 | — | 判据 = 描边 常规 1px（卡片外框 0.5px / 焦点环 3px）；其余 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1） | 已实现（映射 = G-H7-05→internal/ui/settings_general_screen.go:generalClearDialog；标记 = TextColor@screenDialogPanel） |
| R-GI-460 | G-H7-05 | 无障碍 | 读屏进入该元素时 | 读作按钮「取消」/「关闭」 | — | 读屏文本 = 「取消」 / 「关闭」 | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GI-461 | G-H7-05 | 内容 | 两态文案先后出现时 | 未完成「取消」、完成「关闭」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GI-462 | G-H7-05 | 状态 | 清理进行中时 | 「取消」禁用或给出「无法中止」提示 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GI-463 | G-H7-05 | 无障碍 | 读屏进入该元素时 | 读作按钮「取消」/「关闭」 | — | 读屏文本 = 「取消」 / 「关闭」 | 已实现（逐字）：internal/account/login.go:Account.waitBrowser「取消」 |
| R-GI-464 | G-H7-05 | 样式 | 关闭与清理按钮同屏时 | 同高 28，描边次按钮 | — | 判据 = 控件尺寸 按钮高 次 28；描边 常规 1px（卡片外框 0.5px / 焦点环 3px） | 已实现（映射 = G-H7-05→internal/ui/settings_general_screen.go:generalClearDialog；标记 = TextColor@screenDialogPanel） |
| R-GI-465 | G-H7-06 | 状态 | 未观察（条件）：清理未完成 | 危险按钮逐字「清理浏览器数据」，清理中前置旋转图标 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/settings_general_screen.go:generalBrowserCard「清理浏览器数据」 |
| R-GI-466 | G-H7-06 | 内容 | 清理进行中时 | 文案变「正在清理浏览器数据…」，按钮禁用 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H7-06→internal/ui/settings_general_screen.go:generalClearDialog；标记 = 文案「清理浏览器数据」等 4 条） |
| R-GI-467 | G-H7-06 | 交互 | 鼠标点击该元素时 | 执行清理；成功后成功行出现、浏览器页面关闭重开为空白页 | — | 判据 = 可点次数 1（可点）或 0（禁用且 opacity 50%）（ST-ENABLED）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 已实现（映射 = G-H7-06→internal/ui/settings_general_screen.go:generalClearDialog；标记 = Pressed@screenButton） |
| R-GI-468 | G-H7-06 | 异常 | 清理失败 | 错误块出现，按钮恢复 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-H7-06→internal/ui/settings_general_screen.go:generalClearDialog |
| R-GI-469 | G-H7-06 | 样式 | 危险按钮在屏上时 | 危险色底 + 白字、高 28 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；色值取 token：主色 #349651/#63cb7b、正文 #0a0a0a/#ececec、次文 #737373/#8a8a8a、边框 #e5e5e5/#2e2e2e、警示 #e17100/#ffb900，偏差 0（CL-ACCENT / CL-TEXT / CL-TEXT-2 / CL-BORDER / CL-WARN） | 已实现（映射 = G-H7-06→internal/ui/settings_general_screen.go:generalClearDialog；标记 = TextColor@screenDialogPanel） |
| R-GI-470 | G-H7-06 | 无障碍 | 读屏进入该元素时 | 读作按钮「清理浏览器数据」，并说明会清掉登录状态 | — | 读屏文本 = 「清理浏览器数据」 | 已实现（逐字）：internal/ui/settings_general_screen.go:generalBrowserCard「清理浏览器数据」 |
| R-GI-471 | G-H7-06 | 内容 | 两态文案先后出现时 | 「清理浏览器数据」/「正在清理浏览器数据…」 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT） | 已实现（逐字）：internal/ui/settings_general_screen.go:generalBrowserCard「清理浏览器数据」 |
| R-GI-472 | G-H7-06 | 异常 | 清理失败 | 错误块出现、按钮恢复 | — | 判据 = 文案逐字等于期望列、字符数一致、差 0（TX-EXACT）；一次操作后状态取值变化 1 次、取值与期望列一致（ST-CHANGE-1） | 未实现（映射到位但没有该维度的标记）：G-H7-06→internal/ui/settings_general_screen.go:generalClearDialog |
| R-GI-473 | G-H7-06 | 样式 | 危险按钮在屏上时 | 危险色底 + 白字、高 28 | — | 判据 = 包围盒高取行档 18/24/28/30/32/35/48/60 之一、宽 = 内容宽 + 2×14（BO-BOX-RULE / BO-PAD-14），偏差 ≤ 1px（AL-TOL-1）；色值取 token：主色 #349651/#63cb7b、正文 #0a0a0a/#ececec、次文 #737373/#8a8a8a、边框 #e5e5e5/#2e2e2e、警示 #e17100/#ffb900，偏差 0（CL-ACCENT / CL-TEXT / CL-TEXT-2 / CL-BORDER / CL-WARN） | 已实现（映射 = G-H7-06→internal/ui/settings_general_screen.go:generalClearDialog；标记 = TextColor@screenDialogPanel） |

### 8.8 导入与更新域维度覆盖说明

57 项全部 ≥6 维。G-H3-03（替换确认的「取消」）到 6 维：文案恒定、无过渡，动画随弹窗整体；G-H2-05（「技能」分组标签）同理。

---

## 9. 全量统计与门禁自查（八类）

### 9.1 数量

| 类 | 元素数 | 条目数 | 平均 | 最小 / 最大 |
|---|---:|---:|---:|---:|
| §1 审批（C） | 63 | 587 | 9.32 | 8 / 30 |
| §2 搜索（A） | 46 | 382 | 8.30 | 8 / 10 |
| §3 命令（B） | 29 | 235 | 8.10 | 8 / 10 |
| §4 提问（D） | 22 | 182 | 8.27 | 8 / 10 |
| §5 确认（E） | 21 | 174 | 8.29 | 8 / 9 |
| §6 重命名（F） | 16 | 132 | 8.25 | 8 / 9 |
| §7 导出与 Fork（G） | 22 | 183 | 8.32 | 8 / 9 |
| §8 导入与更新（H） | 57 | 473 | 8.30 | 8 / 10 |
| **合计** | **276** | **2,348** | **8.51** | **8 / 30** |

核心件条目数：G-C1-01 30、G-C1-18 16、G-C1-20 16、G-C4-01 16、G-C1-02 14、G-C1-19 14、G-C4-03 14、G-A1-03 10、G-E2-02 10、G-G1-10 10。

### 9.2 未观察比例（按类）

| 类 | 有屏上证据 | 未观察 | 未观察占比 |
|---|---:|---:|---:|
| 审批 | 11 | 52 | 82.5% |
| 搜索 | 16 | 30 | 65.2% |
| 命令 | 13 | 16 | 55.2% |
| 提问 | 0 | 22 | 100% |
| 确认 | 0 | 21 | 100% |
| 重命名 | 0 | 16 | 100% |
| 导出与 Fork | 0 | 22 | 100% |
| 导入与更新 | 0 | 57 | 100% |
| **合计** | **40** | **236** | **85.5%** |

行层面：行首标「未观察（条件）」的 233 条（9.9%）；其余行虽属未观察元素，但行高、按钮尺寸、文案或结构已由同一采集态里的相邻节点反推（例如确认框的按钮行高与审批卡同款、弹窗骨架与导出弹窗同款），故不逐行重复标记。

### 9.3 维度覆盖

在用维度 14 个：内容 505、状态 394、样式 337、无障碍 311、异常 245、交互 198、关联 125、记忆 82、键盘 53、窄窗 52、动画 20、高DPI 14、多语言 9、性能 3。

每元素维度数分布：5 维 83 件、6 维 78 件、7 维 61 件、8 维 39 件、9 维 8 件、10 维 2 件、11 维 4 件、14 维 1 件。**最低 5 维，全部 ≥5**；5 维件集中在「一行静态文本」（标题、描述、版本行）与「纯图标」两类，缺的维度写在各节末尾的说明与行内「不适用」。

### 9.4 门禁自查（§9.16 十）

| 项 | 要求 | 本版 | 结论 |
|---|---|---|---|
| 元素 100% 有需求 | 八类 276 元素逐一出条目 | 276 / 276 | 通过 |
| 每元素条目 | ≥8 | 最小 8，无例外 | 通过 |
| 核心件 | ≥30 | G-C1-01 = 30；其余核心件 10–16 | 部分通过（用户裁决：不强制 30） |
| 维度覆盖 | 每元素 ≥5 | 最小 5 | 通过 |
| 验收三形态 | 具体值（文案 / 坐标 / 计数）/ 基线常量（`常量名` + 值，见 design-baseline §11）/ 与期望列一致 | 文案、计数、包围盒给具体值；样式给 `判据 = <基线常量名> <值>` | 通过 |
| 禁止项 | 不自造元素名、不写模板句 | 元素 ID 与文案逐字取清单；`{name}`、`{title}`、`{count}`、`{heldBy}` 等按清单插值占位 | 待复核抽样 |
| 两块合计 | ≥2,000 | 右栏 2,152（requirements-panels.md）＋ 全局 2,348 = 4,500 | 通过 |

### 9.5 本版仍未解决的三处

1. **五类零覆盖是不可通改性，不是采集遗漏**：提问卡、四类确认框、两类重命名、导出/Fork、六个导入与更新对话框，60 份态里一份都没打开过。其中**会话重命名（F1）在本机根本不可达**——`SessionTitle` 不可用，菜单项直接渲染成「重命名（SessionTitle 不可用）」且禁用。要转成具体值验收，需先恢复该能力再重采。
2. **`@` / `$` 提及（A3/A4）与命令面板基元（B2）都需要操作或特定数据**：前者要键入 `@`/`$` 并构造文件/提交/技能三类候选，后者只在「设置 → 添加模型」调色板里出现。本版按模板条件写，验收走形态②。
3. **审批域的两处硬前置**：沙箱越权卡要 `variant=sandbox`（四类资源各需一次），子代理模型卡要 `variant=subagent_model`，成员审批区块要团队分支请求——本机一样都没有。这 17 项的文案与结构只能按清单与 i18n 写，与屏上的一致性待补采后核。