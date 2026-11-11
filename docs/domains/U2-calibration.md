# U2 视觉校准：对照 mygo 官方示例与控件

Status: draft（校准结果，待组织评审）
Owner: U2 界面视觉
依据: `github.com/egoist/mygo@v0.8.3` 的 `examples/` 与 `ui/`（只读）；本仓 `internal/ui/`（只读）与 `docs/domains/screens/` 的九张截图；`docs/domains/U2-visual.md`（v2）。
范围: 只对照"看得见的值"——字号层级、间距节奏、颜色分配、控件默认样式、内容列宽、对齐、状态样式。不含区域划分、触发与键位。
产物: 本文 + `docs/domains/calibration/examples/`（9 张示例/探针渲染）+ `docs/domains/calibration/compare/`（7 张对照图）。

---

## 0 结论摘要

| 项 | 数 |
|---|---:|
| 对照的示例 | 4 个（markdown-chat、notes、permissionflow、gallery）+ 框架控件源码 |
| 渲染出的图 | 9 张（示例 7 + 纠正探针 2） |
| 差异条数 | 17（§4）：D1–D17 |
| 其中"我们错" | 9（D1–D8、D16） |
| 其中"我们不同但有据" | 7（D10–D15、D17；见 §6，属刻意） |
| 其中"与官方同值但实现不一致" | 1（D9：分隔线，视觉值相同，只是没用框架控件） |
| 纠正清单 | 8 条必改（C1–C8，覆盖 D1–D9）+ 2 条可选（C9、C10，覆盖 D17、D16） |

最严重的三条，都能从像素上直接看出：

1. **禁用主按钮丢掉"主"的身份**。框架禁用是"保留填充、整体 0.5 不透明度"（`ui/paint.go:312`），我们是"先换成中性面、再被框架压 0.5"（`internal/ui/controls.go:51-54`）。实测：`发送`（主，禁用）面 `#f4f4f5`、边 `#e7e7ea`、字 `#a9a9ae`；`+ 附件`（次，启用）面 `#f4f4f5`、边 `#d9d9de`、字 `#18181b`。两者只剩"更淡"，没有"更主"，而且禁用件的边（1.12:1）比次按钮（1.28:1）还弱；标签对 `surface` 只有 2.13:1，而规范 §7.1 例外 2 写的是 3.31（那正是框架只压 0.5 时的值）。见 `calibration/compare/buttons.png`。
2. **按钮比声称的矮 4px**。`internal/ui/controls.go:9-10` 的注释写"竖向 4 让按钮正好 row-regular 28"，实测 24（行盒 14 + 2×4 + 2×1）。框架自己的按钮是 6/14 内边距、29–30 高。见 §4 D2。
3. **输入卡里套了第二道边框**。卡是 `surface` + `border-strong`，里面的 `TextArea` 用框架默认外观（`surface` + `border`，`ui/textinput.go:18`），两个面同色、两道边只差 1.28:1。框架的两种做法都不是这样：要么卡为 `bg`、字段为 `surface`（gallery），要么容器只画一道边、编辑器无外观（notes）。见 `calibration/compare/nesting.png`。

另外发现两处**规范自相矛盾**（不是实现跑偏，是文档内部打架），见 §4 的 D1、D3 与 §5 的 C1、C3：

- §4.1 写"禁用按钮：`surface` + 1px `border` + `text-secondary`"；§4.2 同一控件写"禁用 `.5` + 不响应悬停"；§7.1 的例外 2 又按 ".5 不透明度"算对比度。
- §4.2 写"输入卡：`surface` + 1px `border`"，§3.5 写 `border-strong` 的使用面"限死在输入卡轮廓"。实现取的是 `border-strong`。

---

## 1 方法与来源

### 1.1 来源（只读）

| 来源 | 用途 |
|---|---|
| `<GOMODCACHE>/github.com/egoist/mygo@v0.8.3/examples/` | 示例的 view 代码、窗口尺寸、控件用法 |
| `<GOMODCACHE>/github.com/egoist/mygo@v0.8.3/ui/` | 主题默认值、控件内边距与状态实现 |
| `internal/ui/` | 我们的 token 与控件外观（只读，未改） |
| `docs/domains/screens/` | 我们的九张截图（1440×900，1x） |

kero 项目：**本机没有副本**（`<HOME>/Code` 下无同名目录，`<HOME>/.config/kero` 为空目录），本次未做对照。这一项按"缺来源"记录，不用推测替代。

### 1.2 渲染

示例的 view 函数与窗口状态原样使用，只把入口从 `mygo.NewWindow` 换成框架的无窗口渲染：

1. 把示例源码复制成普通包（只改 `package main` → 包名，view 与 main 函数一行未动）；
2. 每个示例加一个初始化函数，照抄它 `main()` 里构造状态的那一行；
3. `ui.NewTesterWith(view, 宽, 高, TesterOptions{Dark:…})` 取 `Tester.Image()`，`png.Encode` 落盘。

窗口尺寸取示例自己的值：markdown-chat 820×720、notes 900×640、permissionflow 520×640、gallery 980×720。比例 1x，所以图上 1 px = 1 DIP。

### 1.3 量法

对 PNG 逐列/逐行取连续同色段，读出边界坐标；差值即 DIP。文字行距按"上一行字形带上沿到下一行字形带上沿"量。所有数字可由 §7 的步骤重跑得到。

---

## 2 示例与渲染

| 示例 | 用来对照什么 | 图 |
|---|---|---|
| `examples/markdown-chat` | 消息流、用户气泡、正文列宽、输入条 | `calibration/examples/markdown-chat.png`、`markdown-chat-dark.png` |
| `examples/notes` | 侧栏（230）、会话行、工具条、单容器编辑器 | `calibration/examples/notes.png` |
| `examples/permissionflow` | 卡片行、主/次/禁用按钮、状态文字 | `calibration/examples/permissionflow.png` |
| `examples/gallery` | 框架自己的应用外壳与全部控件默认样式（Overview / Controls / List） | `calibration/examples/gallery-overview.png`、`gallery-controls.png`、`gallery-list.png` |
| 纠正探针 | 只换框架能力（`TextAreaBase` + 6/14 内边距 + 不禁用换面）后的输入位 | `calibration/examples/probe-composer-before.png`、`probe-composer-after.png` |

对照图（左/上为官方，右/下为我们）：

| 图 | 内容 |
|---|---|
| `calibration/compare/stream.png` | 消息流整体 |
| `calibration/compare/composer.png` | 输入条 vs 输入卡 |
| `calibration/compare/buttons.png` | 禁用主按钮两种画法（3x） |
| `calibration/compare/nesting.png` | 输入框嵌套（4x） |
| `calibration/compare/sidebar.png` | 侧栏行高、内缩、选中行 |
| `calibration/compare/statusmarks.png` | 状态位：字形 vs 图标（3x） |
| `calibration/compare/fix-preview.png` | 纠正前后（100%，同布局） |

---

## 3 官方做法抽取

### 3.1 字号层级

| 观察 | 数值 | 位置 |
|---|---|---|
| 主题只有一个根字号，所有控件继承 | macOS 13、其他 14 | `ui/theme.go` `defaultFontSize()` |
| 页面标题 | 26 / 700（gallery）、20 / 700（permissionflow、timeline） | `examples/gallery/main.go:374`、`examples/permissionflow/main.go:44` |
| 卡片标题 | 15 / 700 | `examples/gallery/main.go:533` |
| 正文 | 继承根字号（13） | 各处 `ui.Text` 不设字号 |
| 次要文本 | 12（permissionflow 状态词）、13（notes 工具栏的 Select） | `examples/permissionflow/main.go:77`、`examples/notes/main.go:151` |
| 侧栏分组标题 | 根字号 − 1（框架根字号 13 时 = 12；我们的根字号 14 时 = 13），600，`TextMuted` | `ui/sidebar.go:134`（`FontSize(t.FontSize - 1)`） |
| 徽标 | 根字号 − 2（13 时 = 11） | `ui/sidebar.go:225` |
| 小字最小档 | 根字号 × 0.75 | `ui/table.go:637` 的页码标签 |
| 纯文本行高 | 不设，按字体度量（实测 12px 两行间距 15 → 1.25） | `examples/permissionflow/main.go:47-48` 实测 |
| Markdown 行高 | 1.5 | `ui/markdown/theme.go` |

层级只有一个锚点（`Theme.FontSize`），**没有第二套界面级字阶**；示例里出现的 11/12/15/20/22/26 全是相对根字号或散点，不成阶梯。

### 3.2 间距

| 观察 | 数值 | 位置 |
|---|---|---|
| 基础单位 | 4，控件内边距全是它的倍数或半档 | `ui/theme.go` `Spacing` |
| 按钮内边距 | 6 / 14（= 1.5u / 3.5u） | `ui/widgets.go:154` |
| 输入框内边距 | 6 / 10（= 1.5u / 2.5u） | `ui/textinput.go:18` |
| 侧栏容器内边距 | 10（= 2.5u），行间 2（0.5u） | `ui/sidebar.go:43` |
| 侧栏行内边距 | 6 / 8 | `ui/sidebar.go:164` |
| gallery 内容列 | 上 12、左右 32、下 28、块间 18 | `examples/gallery/main.go:362` |
| gallery 卡片 | 内 18、块间 12 | `examples/gallery/main.go:529-530` |
| notes 主列 | 12 / 20、块间 10 | `examples/notes/main.go:114` |
| chat 列表 | 16 / 20、消息间 16 | `examples/markdown-chat/main.go:58` |
| chat 输入条 | 12 / 20，上边 1px 分隔 | `examples/markdown-chat/main.go:63` |

要点：**5 档以内、同一界面内只出现 2–3 个档位**；半档（0.5u、1.5u、2.5u、3.5u）是框架自己在用的，不是禁例。

### 3.3 颜色分配

| 观察 | 做法 | 位置 |
|---|---|---|
| 面 | 窗口 `Background`、控件/卡片/侧栏 `Surface`，悬停与按下另两档 | `ui/theme.go` |
| 文字 | 正文 `Text`，次要 `TextMuted`；两者只用于文字，不作填充 | 各处 |
| 强调 | 只给主按钮填充、选中项填充、图标、链接、焦点环 | `ui/sidebar.go:188`、`ui/widgets.go:157` |
| 状态色 | 只在状态文字上（granted 用 `Success`、error 用 `Danger`） | `examples/permissionflow/main.go:70-76` |
| 卡片与字段的关系 | 卡为 `Background`（白），字段为 `Surface`（灰）——**靠面区分，不靠第二道边** | `examples/gallery/main.go:530,692-703` |
| 暗色 | 强调色提亮（`#3b82f6`），不是沿用明色 | `ui/theme.go` `DarkTheme` |

### 3.4 控件默认样式

| 控件 | 默认 | 位置 |
|---|---|---|
| 按钮 | `Surface` + 1px `Border` + 圆角 6 + 内边距 6/14 + 标签根字号 | `ui/widgets.go:154-176` |
| 主按钮 | `Accent` 填充 + `AccentText`，**不描边（透明）** | 同上 |
| 禁用 | 不禁用换面：保留原填充，整棵子树 0.5 不透明度 | `ui/paint.go:312`；`ui/element.go:1907-1914`（`DimDisabled` 允许自绘禁用外观） |
| 输入 | `Surface` + 1px `Border` + 圆角 6 + 内边距 6/10 | `ui/textinput.go:18` |
| 无外观输入 | 有 `TextInputBase` / `TextAreaBase` 两个变体，专门给"容器自己画边"的场合 | `ui/base.go:718-720` |
| 侧栏 | 容器 `Surface` + 内边距 10；行 `Padding(6,8)` + 圆角 6；选中且聚焦 = `Accent` 填充，选中未聚焦 = `SurfacePressed`，悬停 = `SurfaceHover` | `ui/sidebar.go:43,164,183-196` |
| 侧栏行图标 | 独立 icon 槽：16px SVG + 8 间隙，文字从固定列起 | `ui/sidebar.go:164,211-216` |
| 分隔线 | `ui.Divider`：按父容器方向给 1px 宽/高，颜色 `Border` | `ui/widgets.go:39-46` |
| 卡片 | 圆角 10、内边距 18、`Background` + 1px `Border` | `examples/gallery/main.go:528-533` |
| 消息气泡 | 圆角 14、内边距 8/12、最大宽 520、右对齐 | `examples/markdown-chat/main.go:84` |

### 3.5 内容列宽与对齐

| 观察 | 数值 | 位置 |
|---|---|---|
| 助手正文最大宽 | 760（窗口 820） | `examples/markdown-chat/main.go:88` |
| 用户气泡最大宽 | 520 | 同上 :84 |
| 列表对齐 | 左对齐，列不居中；两侧只留列表内边距 | `examples/markdown-chat/main.go:58` |
| gallery 主列 | 占满剩余宽度，不设上限 | `examples/gallery/main.go:359-362` |
| notes 主列 | 占满剩余宽度 | `examples/notes/main.go:114` |

框架的示例里**没有"居中 + 最大宽度"的内容列**；唯一的宽度上限是正文 760 与气泡 520，并且左对齐。

### 3.6 状态与禁用

- 禁用：不换面，0.5 不透明度；框架文档明确允许"自绘禁用外观"的控件关掉这种变暗（`ui/element.go:1907-1914` 的 `DimDisabled`），但前提是**自绘后仍要能被识别为禁用**。
- 选中：强调填充（聚焦时）或 `SurfacePressed`（未聚焦时）；没有"底色 + 竖条"的写法可参照。
- 状态文字：状态词用小字 + 状态色，标题保持正文色（permissionflow 的 `granted` / `denied`）。

---

## 4 差异表

"官方做法"一列都带数值与源码位置；"实测"指本文的像素量法。

| # | 项 | 官方做法（数值 + 位置） | 我们的做法 | 差异 | 结论 |
|---|---|---|---|---|---|
| D1 | 禁用主按钮 | 保留填充，整体 0.5 不透明度（`ui/paint.go:312`）：禁用主按钮实测面 `#8dacf0`，禁用次按钮面 `#f4f4f5`、标签 `#868687`（3.31:1）——两者仍靠"面"区分 | 先换中性面再被压 0.5（`internal/ui/controls.go:51-54`）：实测面 `#f4f4f5`、边 `#e7e7ea`（1.12:1）、字 `#a9a9ae`（2.13:1） | accent 填充消失；与启用次按钮（边 `#d9d9de` 1.28:1、字 `#18181b` 16.12:1）只差"更淡"，且边框比次按钮还弱 | **我们错**；U2 §4.1 与 §4.2、§7.1 三处指向两种做法 |
| D2 | 按钮高度 | 内边距 6/14 + 标签 13 → 实测 29–30（permissionflow y 95..123） | 内边距 4/12 + 标签 12.25 → 实测 24（附件 y 855..878；标题行按钮 25） | −5~6px；且注释声称"正好 28"，实测 24 | **我们错**（值不成立） |
| D3 | 输入卡的第二道边 | 卡 `Background` + 字段 `Surface`（`gallery/main.go:530,692`），或容器一道边 + 编辑器无外观（`notes/main.go:114-121`，实测内无第二道边） | 卡 `surface` + `border-strong`，内部 `TextArea` 用默认外观 `surface` + `border`（`internal/ui/composer.go:47-49,57`） | 同面双层描边；实测卡边 y=732 `#85858b`、内框 y=774 `#d9d9de`，对内框下的面只有 1.28:1 | **我们错** |
| D4 | 选中行 | 聚焦 = `Accent` 填充 + `AccentText`；未聚焦 = `SurfacePressed`（`ui/sidebar.go:183-196`） | 直接用框架默认；U2 §4.1 要求"`surface-hover` 底 + 2px `accent-ink` 竖条" | 实测选中行 `#dddde1`（= `SurfacePressed`），**竖条不存在** | **我们错**（规范未落地） |
| D5 | 审批卡描边 | 1px 边要单独承担识别时用 ≥3:1 的色（本仓 U2 §3.5 #9 的 `border-strong` 逻辑） | 规范 §4.1 写 `accent-ink` `#1d4ed8`（对 surface 6.10）；实现 `composer.go:93` 用 `t.Accent` `#2563eb`（4.70） | 实测卡边 `#2563eb`，与规范值不符 | **我们错**（规范/实现值不一致） |
| D6 | 状态位形态 | SVG 图标 + `ui.Icon`，`FontSize(16)`（`ui/sidebar.go:212`）；24 网格 1.5 描边 → 16px 时 1.0px | 文本字形 `✓ ◷ ✕ !`，字号随 `type-secondary` 12.25（`internal/ui/timeline.go:130`） | 实测字形高约 9px；大小随字号变、基线随字体变 | **我们错**（§4.1/§6.4 已写 16px 图标） |
| D7 | 侧栏行首标记 | 独立 icon 槽：16px + 8 间隙，文字从固定列起（`ui/sidebar.go:164,211-216`） | 标记字符拼进 label（`internal/ui/sidebar.go:91` 的 `sessionLabel`） | 有标记的行文字右移约一个字形宽（state-running 第一行） | **我们错**（用错框架能力） |
| D8 | 附件按钮标签 | 图标走 icon 槽或纯文字标签（notes 的 `New`、gallery 的 `Open …`） | 标签里用全角 `＋`（`internal/ui/composer.go:68`、`internal/ui/sidebar.go:22`） | 全角字形占 1em、与右侧字面重心错位 | **我们错**（小） |
| D9 | 右栏分隔线 | `ui.Divider`，按父方向取 1px 宽/高（`ui/widgets.go:39-46`） | `Box().Height(1).Background(t.Border)` 自画（`internal/ui/rightpanel.go`） | 视觉值相同；方向不随容器、多一处手写 | 建议换（低影响） |
| D10 | 根字号 | 13（macOS，`ui/theme.go`） | 14（`internal/ui/tokens.go`；U2 §1.2 有实测依据） | +1；连带所有行盒 +1 左右 | **刻意**（§6） |
| D11 | 六个颜色改值 | `TextMuted #71717a`、`Success #16a34a`、`Warning #d97706`、`Danger #dc2626`、`Focus` 半透明、`Selection .25`（`ui/theme.go`） | 各自改值并附重算对比度（U2 §3.5 A 类） | 逐条 ≥4.5 / ≥3 的门槛差 | **刻意**（§6） |
| D12 | 暗色强调色 | `#3b82f6` / `#60a5fa` / `#2563eb` | `#2563eb` / `#1d4ed8` / `#1e40af`（与明色同值） | 暗色填充对 `bg` 3.43（框架 3.68 白字不达标） | **刻意**（§6） |
| D13 | 正文行高 | 纯文本不设行高（实测 12px 行距 15 → 1.25）；Markdown 1.5 | 正文 1.5 = 21（`tokens.go` `typeBody`） | 与框架纯文本松约 0.25×；与框架 Markdown 相同 | **刻意**（§6） |
| D14 | 内容列宽与对齐 | 正文最大宽 760、左对齐、两侧 20（`markdown-chat/main.go:58,88`） | 最大宽 784、居中、两侧 16、纵向 24（`internal/ui/timeline.go:22-25`） | 宽 +24；水平 −4；纵向 +8；居中 vs 左对齐 | **刻意**（§6） |
| D15 | 轮次间距 | 消息间 16（`markdown-chat/main.go:58`） | 轮次 24、轮内 12/8（U2 §2.3） | +8 | **刻意**（§6） |
| D16 | 次要文本字号 | 侧栏分组标题 = 根字号 − 1 = 13、600（`ui/sidebar.go:134`）；徽标 = −2（框架 13 时 11；我们 14 时 12）（`ui/sidebar.go:225`） | 自己的次要级 12.25（0.875×）、极小 10.5（0.75×）；侧栏分组标题仍是框架的 13 | 同屏并存的"次要文本"有两个字号（13 与 12.25） | **我们错**（小）：同一语义两种字号 |
| D17 | 气泡圆角 | 14（`markdown-chat/main.go:84`），与控件 6 区分开 | 6（`tokens.go` `radius`；U2 §8.3 统一 6） | −8 | **我们不同但有据**，但见 C9 |

### 4.1 两处规范内部矛盾（不是实现跑偏）

| 位置 | 冲突 | 建议 |
|---|---|---|
| §4.1「不可用」vs §4.2 按钮行 + §7.1 例外 2 | §4.1 写"禁用按钮：`surface` + 1px `border` + `text-secondary`"；§4.2 写"`.5` + 不响应悬停"；§7.1 例外 2 又按"禁用态不透明度 .5（明等效 3.31 / 暗 4.55）"算对比度——三处指向两种做法 | 统一为后者（与框架一致），见 C1 |
| §4.2 输入卡 vs §3.5 #9 | 前者"`surface` + 1px `border`"，后者把 `border-strong` 限死在"输入卡轮廓" | 统一为 `border-strong`（实现已是），见 C3 |

---

## 5 纠正清单（按影响排序）

每条给：改什么值 / 换哪个框架能力 / 判定方法。**不含布局与交互改动**。

### C1 禁用主按钮保留填充（影响：高）

- **改**：删掉 `internal/ui/controls.go:51-54` 的"禁用 → `surface` + `text-secondary` + `border`"分支，让框架的 0.5 不透明度生效（不调 `DimDisabled`）。若将来要自绘禁用外观（`DimDisabled(false)`），前提是**保留 `accent` 填充**，只压不透明度。
- **规范**：U2 §4.1「不可用」一行的禁用按钮描述改为"`accent` 填充 + `.5` 不透明度；次按钮 `surface` + `border` + `.5`"。
- **判定**：截图中禁用主按钮的填充 ≠ `surface`（明色应为 `#8dacf0` 一类的淡强调面）；标签对 `surface` ≥3.31；与相邻次按钮在"填充色"一项上不同。

### C2 按钮内边距 4/12 → 6/14（影响：高）

- **改**：`controls.go:11-14` 的 `buttonPadV/buttonPadH` 改为 `t.Space(1.5)` / `t.Space(3.5)`（6/14，框架值）。实测高度由 24 → 28，正好落在 §6.1 的 `row-regular`。
- **规范**：§2.2 增加一句"控件内边距允许 `t.Space(0.5)` 的半档（框架自身即如此：侧栏 2.5u、按钮 1.5u/3.5u）"，否则本改动会违反"只出现 4 的倍数"。
- **判定**：量按钮外框高度 = 28（1x 截图即 28px 高）；标签 12.25 不变。

### C3 输入卡只留一道边（影响：高）

- **改**：`internal/ui/composer.go:57` 的 `gui.TextArea` → `gui.TextAreaBase`（无外观），编辑器不再有 `surface` + `border`；卡的 `border-strong` 保留 1px。
- **备选及否决**：把卡面换成 `bg`、编辑器保留默认外观（gallery 的"白卡灰字段"）。否决理由：白卡落在白底上，边界只剩 1px `border`（对页面底 1.41:1），违反 U2 §3.5 #9 的"边界单独承担识别 ≥3:1"；若改描 `border-strong`，卡与编辑器就是两道强边，回到原问题。
- **验证图**：`docs/domains/calibration/compare/fix-preview.png` 的 after 一栏是这条路线的渲染结果。
- **判定**：卡内边距范围内不再出现第二条 1px 边；输入区左边界只出现一次描边。

### C4 选中行落地（影响：高）

- **改**：任选其一并在 §4.1 写死：(a) 实现"`surface-hover` 底 + 2px `accent-ink` 竖条"（`DrawOver` 画在内侧，上下各留 8）；(b) 采用框架语义"选中 = `Accent` 填充 + `AccentText`"。
- **不建议**保持现状：现状是框架的"未聚焦选中"灰色（`#dddde1`），而规范描述的是另一个样子，走查无法判定。
- **判定**：选中行与相邻未选中行在"底色"或"竖条"上至少一项不同；键盘聚焦侧栏时状态按所选方案变化。

### C5 状态位换成 16px 盒（影响：中）

- **改**：`internal/ui/timeline.go:130` 的 `statusMark` 改成固定 16×16 盒（`ui.Icon` + Heroicons SVG；接入前至少用固定尺寸的 `Box` 居中字形），**不要**用 `type-secondary` 的字形（字号一改，状态位就变大小）。
- **判定**：量状态位外接框 = 16×16；把 `Theme.FontSize` 调到 16 时状态位尺寸不变。

### C6 侧栏行首标记走 icon 槽（影响：中）

- **改**：`internal/ui/sidebar.go` 不再把 `● ◷ ✕` 拼进 `label`；用 `gui.SidebarItem(c, id, icon, label)` 的 icon 参数（或固定 16px 标记盒）。
- **判定**：带标记与不带标记的行，文字左边界相同。

### C7 审批卡描边 `accent` → `accent-ink`（影响：中）

- **改**：`internal/ui/composer.go:93` 的 `t.Accent` → `colorAccentInk.in(t)`。明色对 surface 4.70 → 6.10（U2 §3.2 数值）。
- **判定**：卡边像素 = `#1d4ed8`（明）/ `#60a5fa`（暗）。

### C8 分隔线与全角加号（影响：低）

- **改**：`internal/ui/rightpanel.go` 的自画分隔 → `gui.Divider(c)`；`＋ 附件` → `+ 附件`（或 icon 槽 + "附件"）。
- **判定**：分隔线随容器方向取 1px；标签里不再出现全角 `＋`。

### C9 气泡圆角（可选，影响：低）

- **改**：`internal/ui/timeline.go` 的用户气泡 `radius` → 14（框架 chat 值），与控件 6 拉开形态差。
- **代价**：需要给 §8.3 加一档 `radius-bubble`，多一个 token；不加则维持现状，属可接受。
- **判定**：气泡四角半径 = 14；控件仍为 6。

### C10 统一"次要文本"字号（可选，影响：低）

- **改**：二选一——把 `type-secondary` 提到 13（= 框架 `FontSize − 1`，与侧栏分组标题一致），或给侧栏分组标题单独覆盖到 12.25。前者更省事，且与 D16 的框架惯例同向。
- **代价**：14 × 0.875 = 12.25 是 U2 §1.1 的推导；改成 13 需要改 §1.1 那一行的来源说明（框架 `FontSize − 1` 也是框架自身在用的比例）。

---

## 6 我们刻意做的差异（有据，不建议改）

| 项 | 我们的值 | 依据 | 与官方的实际差距 |
|---|---|---|---|
| 根字号 14（D10） | `tokens.go` `typeBodySize` | 现役客户端实测 14（U2 §1.2），非 macOS 平台框架默认也是 14 | +1；行盒 +1 左右，按钮/输入框整体高 1–2px |
| 六个颜色改值（D11） | `#5f5f68` / `#15803d` / `#b45309` / `#b91c1c` / 不透明 `Focus` / `Selection .20` | 全部是 WCAG 门槛（U2 §3.5 A 类，附重算前后比值） | 框架的对应值在要求面上 2.33–4.40，不达标 |
| 暗色强调色（D12） | `#2563eb` / `#1d4ed8` / `#1e40af` | 白字 ≥4.5（框架暗色 3.68） | 填充更暗；禁用态在暗色下会更暗，属已知代价 |
| 正文行高 1.5（D13） | 21 | 与框架 Markdown 同值；消息正文最终由 markdown 渲染 | 对纯文本比框架默认松 0.25× |
| 内容列 784 居中（D14） | `ContentMaxWidth` | 可读行宽 + 气泡两侧留白对称 | 比框架的 760 宽 24；框架示例里没有居中列，属新增做法 |
| 轮次 24（D15） | `space-6` | 轮次是语义分组，框架的 16 用于连续消息流 | +8，信息密度更松 |
| 气泡圆角 6（D17） | `radius` | §8.3"控件与卡片同值" | 与框架 chat 的 14 不同；若接受 C9 则改 |

---

## 7 复现步骤

1. 渲染示例（本机任意空目录，临时模块，`require github.com/egoist/mygo v0.8.3`，`GOPROXY=off` 可离线）：
   - 复制 `examples/<名>/main.go` 等文件，把首行 `package main` 改成包名；
   - 在包里加一个函数照抄 `main()` 里构造状态的那一行，返回 view；
   - `ui.NewTesterWith(view, 宽, 高, ui.TesterOptions{Dark: 明暗}).Image()` → `png.Encode`。
2. 量图：沿列/行取连续同色段，读边界坐标（本文的数值都是这样得到的）。
3. 纠正探针：`calibration/compare/fix-preview.png` 的两栏分别用 `TextArea` 与 `TextAreaBase`、内边距 4/12 与 6/14、禁用换面与不换面渲染，其余完全一致。

我们的截图保持 `internal/ui` 的现有渲染路径：`go test ./internal/ui -run TestScreens`（只比对）或加 `-update` 重写。

---

## 8 本次未覆盖

| 项 | 原因 |
|---|---|
| kero 项目界面 | 本机无副本（见 §1.1），不做推测对照 |
| 动效时长与曲线 | 静态渲染看不到；U2 §5 已逐条采样框架源码（200/500/350/1060/600/400ms），本轮无需重复 |
| 图标集实际形状 | Heroicons 尚未接入，只对照了"图标 vs 字形"这一层（D6） |
| 框架 web 页面（todo、launcher） | 非 native view，不属本次对照面 |
| 悬停/聚焦像素 | 需要指针事件驱动渲染，本文只用源码判定（`ui/sidebar.go:183-196`、`ui/textinput.go:22`） |
