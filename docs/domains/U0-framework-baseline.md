# U0 框架基线（mygo / kero / markstream-go）

Status: draft（A2 结果包，待组织评审）
Owner: U0 界面框架基线
依据: mygo v0.8.3（官方文档与源码）、kero tag v2.1.9（commit `09eb1d42`）、markstream-go commit `f73d35cb`（无标签）、AI Logo tag `mygo/v1.20.0`（commit `2afb64e9`）
范围: 只做「框架已经给了什么」与「许可证能不能用」的核对。不做界面设计、不写界面代码、不评估视觉方案。
不做: 页面地图、信息架构、字阶与色板规格、view 映射、性能预算——分别归 B1/C1/C2/C4。

---

## 0. 出处写法与口径

全文的出处用四种来源前缀加一种实测记录，每一种都能在不问作者的情况下核对到原文：

| 前缀 | 指向 | 核对方式 |
|---|---|---|
| `mygo:<路径>` | mygo v0.8.3 模块内的文件 | 本地 `<go env GOMODCACHE>/github.com/egoist/mygo@v0.8.3/<路径>`；公开 https://github.com/egoist/mygo/blob/v0.8.3/&lt;路径&gt; |
| `kero:<路径>` | kero 仓库文件，钉 commit `09eb1d42dfe24c8ffc07600024e152e096532519`（即 tag v2.1.9） | https://github.com/egoist/kero/blob/09eb1d42dfe24c8ffc07600024e152e096532519/&lt;路径&gt; |
| `markstream:<路径>` | markstream-go 仓库文件，钉 commit `f73d35cba791fd740dd720ce9f838a193d68c1ee`（仓库无任何 tag） | https://github.com/zerob13/markstream-go/blob/f73d35cba791fd740dd720ce9f838a193d68c1ee/&lt;路径&gt; |
| `ailogo:<路径>` | AI Logo 仓库文件，钉 tag `mygo/v1.20.0`（commit `2afb64e9d39b407b4f06282f79ab8c3d342b370c`） | https://github.com/yldm-tech/ai-logo/blob/2afb64e9d39b407b4f06282f79ab8c3d342b370c/&lt;路径&gt; |
| `自测:<命令>` | 本次在本机实际跑过的核对（记录在附录 B） | 复跑同一条命令 |

文档内引用本仓库位置时用相对路径（`go.mod`、`docs/domains/A.md`）。`mygo` 的版本号只写标签或伪版本，不写本机缓存目录以外的机器信息。

结论里的「上游数字」都来自框架作者自己的测量（机型、Go 版本、编译选项各不相同），不是本项目实测，引用时保留其原始口径。

---

## 1. 结论（先读这段）

1. **能力上没有缺口。** mygo v0.8.3 的 `ui` 包提供 114 个构造器（25 个是无外观的 `Base`），覆盖布局、文本与富文本、输入与输入法、焦点与快捷键、虚拟列表、表格、树/大纲、对话框与浮层、窗口与菜单、无障碍与测试。本项目界面需要的形态都能在框架内落地，不需要自研控件底座。
2. **有一条硬门槛。** `mygo@v0.8.3` 的 `go.mod` 要求 `go >= 1.27.3`；本仓库现在写的是 `go 1.24`，实测直接构建失败（`自测:go-mod`）。引入 mygo 的第一步是把本仓库的 `go` 指令升到 `1.27.3`，并接受该工具链下限。
3. **可复现的最小闭环已实测。** `go 1.27.3` + `mygo v0.8.3` + `ui.NewTester` 的无窗口渲染在本机通过（`自测:go-mod`）。
4. **kero 是可用的实战样本，不是模板。** 它证明「一个 action 表驱动菜单 + 命令面板 + 可重绑快捷键 + CLI」「自绘外观建在 `Base` 上」「`ui.Tester` 覆盖 view 层（19 个测试文件、91 处）」三条做法在真实产品里站得住；也暴露出「单包 main 2.3 万行」这类我们不该照抄的组织方式。
5. **markstream-go 与 mygo 自带 `ui/markdown` 是两条互斥路线。** 前者功能面大（虚拟化、FileDiff、rich 引擎），但无版本标签、依赖 Chromium（rich 包）、体积与维护面都更大；后者是框架内建、零第三方、已按流式设计。二选一必须在 C4 前拍板（见 §7）。
6. **许可证四项都可商用**：mygo / kero / markstream-go 均 MIT；图标 MIT（fork 自 lobe-icons，MIT）、非 AI 图标来自 Simple Icons（CC0-1.0）并逐条列出例外（CC-BY-4.0 / CC-BY-3.0 / MIT / BSD-3-Clause / Apache-2.0）。义务是保留版权与许可声明；启用 markstream 的 `rich` 包会额外引入 MPL-2.0 的 D2，需要单独评估（§5.4）。
7. **状态模型：能力在，映射不在。** 框架给了 18 项可控视觉态的判断与绘制能力，主题只有 29 个 token（20 色 + 4 尺寸 + 间距/字号/字族/提示延迟各 1），动效机制齐备但没有时长常量。**「哪个态长什么样」必须由我们在 C2 定、在 view 层实现**（详见 §2.5）。

---

## 2. mygo 能力基线

### 2.1 视图模型（整个界面的写法由此决定）

| 事实 | 出处 |
|---|---|
| view 是「状态 → 界面」的函数，主线程上每帧重建；一个 frame 可以有多趟 build，所以同一帧能看到动作的结果 | `mygo:docs/ui/views.md`「Views」；`mygo:ui/doc.go` |
| `ui.Element` 只在一次 build 趟内有效，过期使用在开发构建与 Tester 里会报出；跨帧身份用 `ui.Handle`、`ListState`、`ScrollState`、`Router`，它们必须放在 app 状态里 | `mygo:docs/ui/views.md`「Views」「Events and actions」 |
| 控件在构造返回前就写入用户输入（ImGui/egui 语义）；容器（如 radio group）在 `Children` 返回时结算输入 | `mygo:docs/ui/views.md`「Events and actions」 |
| 其他 goroutine 改状态用 `Window.Update`（改完重绘）；只需重绘用 `Window.Invalidate` / `mygo.RunOnMain` | `mygo:docs/ui/views.md`「Change the state from other goroutines」 |
| 锚点式身份：`c.Key("search")`、`c.Key(item.ID)`；容器带 Key 会为子树另开身份作用域 | `mygo:docs/ui/views.md`「Keys」 |
| `mygo vet` 会查出被存进结构体/包变量/goroutine 的 build 值 | `mygo:docs/ui/views.md`「Check build lifetimes」 |

对本项目的直接含义：**会话视图不能把 Element 放进任何缓存**，能缓存的只有文本、布局结果（由框架持有）与业务状态；这与 `docs/domains/A.md`「镜像状态只由事实驱动」不冲突，但要求写 view 的人守住这条线。

### 2.2 控件清单

`ui` 包 v0.8.3 的构造器共 **114** 个（含泛型），分类与数量如下（逐项名称见附录 A）：

| 类别 | 数量 | 代表 |
|---|---:|---|
| 布局与容器 | 13 | `Column` `Row` `Box` `Spacer` `Divider` `Grid` `Scroll` `ScrollHorizontal` `ScrollBoth` `Split` 系列 |
| 文本与富文本 | 3 | `Text` `Textf` `RichText` |
| 按钮与动作 | 8 | `Button` `PrimaryButton` `Link` `MenuButton` `PopoverMenuButton` `Toggle` `ToggleGroup` `Segmented` |
| 导航与工具栏 | 8 | `Toolbar` `Tabs` `Sidebar` `SidebarItem` `SidebarSection` `Breadcrumbs` `BackButton` `ForwardButton` |
| 选择与调节 | 12 | `Checkbox` `CheckboxGroup` `Switch` `RadioGroup` `Select` `Slider` `StepSlider` `RangeSlider` `Stepper` `Rating` |
| 文本输入 | 12 | `TextInput` `TextArea` `TextInputBuffer` `TextAreaBuffer` `NumberInput` `DecimalInput` `SearchField` `Combobox` `Autocomplete` `TokenField` `EditableText` `FindBar` |
| 日期与颜色 | 5 | `Calendar` `DateInput` `TimeInput` `ColorPicker` `ColorWell` |
| 集合 | 7 | `List` `Table` `Tree` `TreeItem` `GridView` `Outline` `OutlineTable` |
| 表单与折叠 | 6 | `Form` `Field` `Fieldset` `Collapsible` `Accordion` `AccordionItem` |
| 覆盖层与主题 | 5 | `Modal` `AlertDialog` `Popover` `Overlay` `Themed` |
| 状态与媒体 | 9 | `Progress` `Spinner` `Meter` `Badge` `Image` `Video` `Icon` `Avatar` `WebView` |
| 无外观底座 | 25 | `ButtonBase` `TextInputBase` `DialogBase` `PopoverBase` `MenuBase` `ListState` 之外的全部 `*Base` |

子包另有一套成型控件，同样是「框架给了」的部分：

| 子包 | 提供 | 出差 |
|---|---|---|
| `ui/markdown` | `View` `Stream` `Revealed` `DefaultCodeBlock` `Slug` `FileImages` | `mygo:docs/ui/markdown.md` |
| `ui/editor` | `View`（代码编辑器）`DiffView` `NewBuffer` `ParsePatch` `Languages` `TokenHighlighter` `DefaultTheme` | `mygo:docs/ui/code-editor.md`、`mygo:docs/ui/diff-viewer.md` |
| `ui/richtext` | `Editor` `NewDocument` `ParseMarkdown` `ParseHTML` `ParseRTF` | `mygo:docs/ui/rich-text-editor.md` |
| `ui/chart` | 比例尺、轴、`Line` `Area` `Pie` `Stack` `Select` 等 mark | `mygo:docs/ui/chart.md` |
| `plugins/terminal` | `terminal.New/View`，libghostty-vt 以纯 Go 方式在运行时加载 | `mygo:docs/plugins/terminal.md` |

### 2.3 布局

flexbox + grid，单位是 DIP（device-independent pixel）。

| 事实 | 出处 |
|---|---|
| `Column` 纵向并拉满宽度、`Row` 横向并垂直居中、`Reverse` 反向；`Spacer` 吃空白、`Divider`/`Dividers` 画分隔线 | `mygo:docs/ui/layout.md`「Containers」 |
| 尺寸：`Width/Height/Size`、`WidthPercent/HeightPercent`、`Min*`/`Max*` 及其 `Percent`、`Fill*`；`Grow/Shrink/Basis` 同 CSS flex；`AspectRatio` | `mygo:docs/ui/layout.md`「Sizes」 |
| 间距：`Gap/GapX/GapY`、`Padding/Margin` 接受 1/2/4 值、`ui.Auto` 外边距居中、逻辑边 `PaddingStart/End` | `mygo:docs/ui/layout.md`「Spacing」 |
| 对齐：`Justify`（沿轴）、`AlignItems`（跨轴）、`AlignSelf`、`Center`、`Wrap`、`Baseline` | `mygo:docs/ui/layout.md`「Alignment」 |
| 定位：`Absolute` + `Top/Right/Bottom/Left`（含 Percent、`Start/End`）、`Attach(at,self)`、`Sticky`（滚动容器内吸附） | `mygo:docs/ui/layout.md`「Positioning」、`mygo:docs/ui/scroll.md`「Sticky elements」 |
| 层级：`ZIndex` 只在兄弟之间生效，不改树序、不改 Tab 顺序与 a11y 读取顺序 | `mygo:docs/ui/layout.md`「Paint order」 |
| 裁剪与显隐：`Clip/ClipX/ClipY`、`Invisible`（占位不画）、`Hidden(true)`（不占位但保留内部状态） | `mygo:docs/ui/layout.md`「Clipping and visibility」 |
| Grid：行列、子项尺寸按单元格百分比 | `mygo:docs/ui/grid.md` |

对做界面的人意义：没有「绝对坐标画页面」这条路（`Absolute` 是例外用法），也没有 CSS 的层叠与选择器；**布局全部由嵌套与 flex 属性表达**。

### 2.4 样式系统

| 事实 | 出处 |
|---|---|
| 元素样式就是方法链：`Background` `Border(宽,色)` `BorderWidth` `BorderStyle` `Radius` `Shadow` `InsetShadow` `NoShadow` `Opacity` `Blur` `ClipInset` `Cache()` `Material` `Cursor` | `mygo:docs/ui/styling.md` 全文 |
| 渐变与遮罩：`Gradient` `LinearGradient`（Oklab 混合）`ConicGradient` `RadialGradient` `BorderLinearGradient` `BorderConicGradient` `MaskLinearGradient` `Masks` | 同上「Gradients and stripes」「Gradient masks」 |
| 颜色：`ui.RGB` `ui.RGBA` `ui.Hex` `ui.Oklch` `Mix` `MixOklab` `Alpha`；Oklch 可超出 sRGB，自动回退 | 同上末段 |
| **没有样式表、没有状态选择器**：外观由 view 里的 `if row.Hovered() { ... }` 表达，一个元素的状态可以给另一个元素上样式 | 同上「State, not selectors」 |
| 主题：`c.Theme()` 提供明/暗，跟随系统切换；改一份副本后 `c.SetTheme(&t)`；`ui.Themed(c, t)` 只主题化一部分 | 同上「Themes」「A theme for part of the window」 |
| 主题的度量字段：`Spacing`（默认 4，控件内边距与间隙都是它的倍数）、`FontSize`、`Radius`、`ScrollbarWidth`、`t.Space(n)`、`t.Rem(n)` | 同上「Themes」 |
| 焦点环：`FocusRingWidth`（默认 2）、`FocusRingOffset`（默认 1）、`Focus`；元素级 `FocusRingOffset/FocusRingColor/FocusRing(false)` | 同上「The focus ring」 |
| 反色面：`Inverse` / `InverseText`（tooltip 与 toast 默认反色）、`TooltipDelay`（默认 0.6s） | 同上「Themes」 |
| 非活动窗口：`InactiveAccent`，内置控件自动降灰 | 同上「Themes」；`mygo:docs/ui/windows.md`「Active and inactive windows」 |
| 桌面偏好：`c.Preferences()` 的 `Accent` `HighContrast` `TextScale` `ReduceMotion` `Pointer` `Hover()` | 同上「The desktop's preferences」 |

关键判断：**框架给的是「一套跟随系统的主题 + 一堆样式原语」，不是一套设计系统。** 字阶、间距标尺、语义色、三态（default/hover/disabled）都要我们自己定义并在 view 里表达（§7 第 1 条）。

### 2.5 组件的状态模型（可控视觉态、token 与动画）

框架里没有「视觉状态机」这个对象：状态是每帧从输入与数据算出来的布尔，外观是每帧重新算出来的样式方法链。所以「支持哪些视觉态」= 「`Element` 上有哪些状态查询与设置」；「这些态长什么样」= 「主题 token + 你在 view 里写的样式」。下面三张表分别回答：能控哪些态、状态怎么落到 API、可调的 token 有哪些。

#### 2.5.1 可控视觉态清单（18 项）

| 态 | 触发 / API | 内建外观 | 能自定义到什么程度 | 出处 |
|---|---|---|---|---|
| 默认（idle） | — | 主题 `Surface` / `Border` / `Text` | 元素级任意重画；`*Base` 从零画 | `mygo:docs/ui/styling.md`「Themes」 |
| 悬停 | `Hovered()`；菜单与选项项的 `Highlighted()` | `SurfaceHover` / `AccentHover`，内建控件都在用 | 主题换色；元素级 `if row.Hovered()`；配合 `Transition{Colors:true}` 淡入 | `mygo:ui/theme.go:16`、`mygo:ui/theme.go:28`；`mygo:docs/ui/input.md`「The pointer」 |
| 按下 | `Pressed()`、`PressedButton()`（哪个键）、`PressedInside()` | `SurfacePressed` / `AccentPressed` | 同上 | `mygo:ui/theme.go:18`、`mygo:ui/theme.go:29` |
| 选中 | `Selected`（整元素 accent 填充并告知辅助技术）、`AccessSelected`（只告知、不着色）、`Checked` / `Mixed`（开关类） | accent 填充，非活动窗口转 `InactiveAccent` | 可：`AccessSelected` + 自己画，或用 Base 自己画 | `mygo:ui/api_gen.go`（`Selected`/`AccessSelected`/`Checked` 的注释）；`mygo:docs/ui/accessibility.md`「Selected, without the look」 |
| 键盘焦点 | `Focusable()`、`FocusVisible()`、`FocusWithin()`、`FocusLeft()`、`TabStop(false)` | 焦点环：`FocusRingWidth`（默认 2）、`FocusRingOffset`（默认 1）、`Focus` | 主题级改宽/距/色；元素级 `FocusRingOffset` / `FocusRingColor` / `FocusRing(false)`；或 `Painter.FocusRing` 自绘 | `mygo:ui/theme.go:47`；`mygo:docs/ui/styling.md`「The focus ring」 |
| 禁用 | `Disabled(true)`、`AccessDisabled(true)`（保留焦点与点击，给必须可达的控件）、`DimDisabled(false)`、`IsDisabled()` | 整棵子树 50% 不透明度，无点击、不进 Tab | 可：关掉 dim 后按自己的方式表达不可用（文档给了「保留不可用原因可读」的例子） | `mygo:ui/api_gen.go`（`Disabled`/`DimDisabled`/`IsDisabled` 注释）；`mygo:docs/ui/custom-widgets.md` 末段 |
| 只读 | `ReadOnly(true)`（文本输入） | 不变灰；可选中、可复制、无光标 | 行为内建，外观自定 | `mygo:ui/api_gen.go`（`ReadOnly`） |
| 无效 / 错误 | `Invalid(true)`；`Field(...).Error(msg)`；自绘消息用 `ErrorMessage` / `DescribedBy` | 文本输入边框转 `Danger`；`Error` 另加说明文字并与控件建立 a11y 关联 | 只有「边框变色 + 消息位置 + 关联」三条约定，其余自定 | `mygo:docs/ui/form.md`「Form」「Hints and errors of your own」；`mygo:ui/api_gen.go`（`Invalid`） |
| 忙碌 / 加载 | **没有控件级内建态**。辅助技术用 `Busy(true)`；视觉只能借 `Spinner`、`Progress(v)`、`Progress(-1)`（未知长度） | 只有独立组件，没有「按钮 loading」「行骨架」 | **必须自建**：按钮内塞 Spinner + `Disabled` + `Busy`，或自画骨架 | `mygo:docs/ui/progress.md`、`mygo:docs/ui/spinner.md`、`mygo:docs/ui/accessibility.md`「States」 |
| 展开 / 收起 | `Expanded(open)`（只告知辅助技术）；视觉用 `Collapsible`/`Accordion`，其 `Progress` 从 0 到 1 | 开合动画内建（`CollapsibleBase.Progress`） | 部分：动画内建；箭头旋转、图标切换、展开区背景要自画 | `mygo:docs/ui/custom-widgets.md` 的 Base 表；`mygo:ui/api_gen.go`（`Expanded`） |
| 搜索命中高亮 | 无专用 API | 无 | **必须自建**：`TextBackground`、`SelectionColor`、或自画 span | `mygo:docs/ui/text.md`「Style」 |
| 拖拽中 | `Dragging()`、`Dragged()`、`DragEnded()`、`DragMissed()`、`PressCanceled()` | `ui.Drag`/`ui.Drop`/`Draggable`/`Reorder` 的目标高亮与重排内建 | 部分：拖影（ghost）与目标态自定 | `mygo:docs/ui/input.md`「The pointer」；`mygo:docs/ui/drag-and-drop.md` |
| 输入法组合中 | `Composing()` | 内建（组合文本由输入法栈绘制） | 沿用 | `mygo:ui/api_gen.go`（`Composing`） |
| 窗口非激活 | `c.WindowActive()` | 内建：选中行、勾选、滑杆、进度、tab 下划线等自动转 `InactiveAccent` | 可（`InactiveAccent: Accent` 保持原色） | `mygo:docs/ui/windows.md`「Active and inactive windows」 |
| 隐藏 | `Hidden(true)`（不占位、保留内部状态）、`Invisible()`（占位不画）、`c.WindowHidden()` | 内建 | 沿用 | `mygo:docs/ui/layout.md`「Clipping and visibility」；`mygo:docs/ui/windows.md`「Hidden windows」 |
| 滚动条显隐 | 悬停显示、`KeepScrollbars(true)`、`Scrollbars(false)` | 内建三策略 | 可（`Scrollbar` / `ScrollbarWidth` / `ScrollbarInsets`） | `mygo:docs/ui/scroll.md`「Scroll bars」 |
| 进入视野 | `InView()`、`InViewAmount(x)` | 无外观，只给布尔 | **必须自建**进场/曝光动画 | `mygo:ui/api_gen.go`（`InView` / `InViewAmount`） |
| 流式完成 | `markdown.Revealed(el)`（reveal 与淡入队列是否排空，据此才挂时间与操作条） | 内建（`ui/markdown`） | 可（`Options.Reveal{Unit, CatchUp, Fade}`） | `mygo:docs/ui/markdown.md`「Once a reply has shown」 |

#### 2.5.2 状态在 API 上的表达（三条通道 + 一条缺席）

1. **查询**（在 build 期内读一次，返回布尔/数值）：`Hovered` `Pressed` `PressedButton` `Clicked` `DoubleClicked` `RightClicked` `Dragged` `Dragging` `DragEnded` `DragMissed` `PressCanceled` `FocusVisible` `FocusWithin` `FocusLeft` `Composing` `Highlighted` `InView` `InViewAmount` `IsDisabled` `Changed` `Submitted`。出处 `mygo:docs/ui/input.md`「The pointer」「The keyboard focus」。
2. **设置**（幂等的断言式方法，写在构造链上）：`Disabled(bool)` `ReadOnly(bool)` `Selected(bool)` `AccessSelected(bool)` `Checked(bool)` `Mixed()` `Expanded(bool)` `Invalid(bool)` `Busy(bool)` `DimDisabled(bool)` `TabStop(bool)` `AutoFocus()` `Focusable()` `Focus()` `Value(s)` `Range(lo,hi,v)` `Step(n)` `PositionInSet/SetSize` `Role(r)` `Label(s)` `Semantics` 相关（`DescribedBy`/`ErrorMessage`/`Controls`）。**其中前 8 个同时改变外观与辅助技术所见的语义，其余只改语义**。出处 `mygo:ui/api_gen.go`（每个方法的注释即规范）、`mygo:docs/ui/accessibility.md`「States」。
3. **事件**（要执行动作时）：`OnClick` `OnChange` `OnSubmit` `OnShortcut`；查询式则用 `Clicked()` 就地分支。出处 `mygo:docs/ui/views.md`「Events and actions」。
4. **缺席**：没有 `VisualState` 枚举、没有 `:hover` / `:disabled` 选择器、没有 `StatefulWidget`；**同一个元素在各态下的差异只能由 view 里的 if 分叉表达，或交给 `*Base` 自己画**（`mygo:docs/ui/custom-widgets.md`）。也没有「主题里给每种态配一套样式」的机制——token 只覆盖了悬停与按下两组颜色。

一句话：**框架给「判断状态的能力」与「画任意样子的能力」，不给「状态 → 样子」的映射表。**

#### 2.5.3 主题 token 清单（29 个字段）

`ui.Theme`（`mygo:ui/theme.go:10`）是全框架唯一一处集中可调的视觉参数。字段按用途分：

| 类别 | 数量 | 字段 |
|---|---:|---|
| 明暗开关 | 1 | `Dark` |
| 颜色 | 20 | `Background` `Surface` `SurfaceHover` `SurfacePressed` `Border` `Text` `TextMuted` `Accent` `AccentHover` `AccentPressed` `AccentText` `InactiveAccent` `Danger` `Warning` `Success` `Selection` `Focus` `Inverse` `InverseText` `Scrollbar` |
| 尺寸 | 4 | `FocusRingWidth`(2) `FocusRingOffset`(1) `Radius`(6) `ScrollbarWidth`(6) |
| 间距 | 1 | `Spacing`(4)，`t.Space(n)` 取 n 倍 |
| 字号 | 1 | `FontSize`（macOS 13、其他 14，见 `defaultFontSize()`） |
| 字族 | 1 | `Font`（`""` 为系统字体） |
| 时长 | 1 | `TooltipDelay`（默认 0.6 s，负值立即显示） |

默认值以 `LightTheme()` / `DarkTheme()` 为基线（`mygo:ui/theme.go:167`、`mygo:ui/theme.go:195`）。自定义三种方式，可叠加：`c.SetTheme(&t)` 整窗替换；`ui.Themed(c, t)` 只主题化一棵子树；元素级链式样式覆盖单个元素（`mygo:docs/ui/styling.md`「Themes」「A theme for part of the window」）。

**token 没覆盖的（必须自建）**：字阶（只有 `FontSize` 一个数）、阴影/高度阶（只有 `Shadow(x,y,blur,spread,color)` 原语）、描边宽度档、动效时长与缓动、密度档（只有 `Spacing` 一个乘数）、层级（`ZIndex` 是元素属性不是 token）、图标尺寸、断点/响应式。

#### 2.5.4 动画能力

| 能力 | API | 默认值 | 出处 |
|---|---|---|---|
| 值缓动（弹簧/曲线） | `Animation{Duration, Ease, Spring, Delay, Oklab, NonMotion}`；快捷方法 `Animate` `AnimateWith` `AnimateTo` `Spring` `AnimateColor` `AnimatePose` | 弹簧 500 ms（`defaultSpringDuration`）；曲线 `EaseOut`；预设 `Smooth` / `Snappy` / `Bouncy` | `mygo:ui/motion.go:16`、`mygo:ui/motion.go:47`、`mygo:ui/motion.go:53` |
| 布局变化过渡 | `Transition(ElementTransition)`：位置、尺寸、颜色、阴影，可逐项开关 | **`Duration` 为 0 时按 200 ms**，负数立即；`ExitDuration` 缺省同 `Duration` | `mygo:ui/transition.go:21`；`mygo:docs/ui/transitions.md`「What moves」 |
| 进出场 | `Enter` / `Exit` 的 `Motion{Collapse:true}` 等 | 内建 | `mygo:docs/ui/transitions.md`「Coming and going」 |
| 错峰 | `Animation.Delay`，如 `time.Duration(i) * 25 * time.Millisecond` | 无默认，靠调用方给 | `mygo:docs/ui/transitions.md`「Delays and staggering」 |
| 文本内容变形 | `ContentTransition`（`Match` / `ScaleContent`） | 内建 | `mygo:docs/ui/transitions.md`「Shared elements」 |
| 关键帧与时间线 | `Keyframes`、`Timeline`、`Loop(key, dur, ease)` | 内建 | `mygo:docs/ui/motion.md`「Keyframes」「Timelines」「Loops and your own motion」 |
| 定时重绘 | `c.After(d)`、`c.AnimationFrame()`、`c.Invalidate()`、`Painter.After` | 内建 | `mygo:docs/ui/views.md`；`mygo:docs/ui/drawing.md`「Animation」 |
| 降低动效 | `Preferences.ReduceMotion` 自动摊平；`Animation.NonMotion` 标记「不动位置的值」（透明度/模糊/交叉淡入）仍可动 | 内建 | `mygo:ui/motion.go:34` |
| 帧诊断 | `MYGO_FRAME_STATS=1` 打印 build/layout/paint/present 耗时构成 | 内建 | `mygo:docs/ui/rendering.md` |

**结论：机制齐备，token 缺席。** 弹簧、曲线、布局过渡、进出场、错峰、关键帧、时间线、降低动效、帧诊断都有；但**没有任何时长/缓动常量**：`200 ms`（过渡）与 `500 ms`（弹簧）只是两处默认值，其余时长由每个调用点自己写。真实项目里这一点立刻显形：kero 的时间常数散成 `50/100/120/150/200/260/400 ms` 六档并存（`自测:kero-durations`；其中 `sidebarMotion` 260 ms、`popupFace` 120 ms），因为框架没有给出可引用的刻度。

#### 2.5.5 结论：沿用 / 配置 / 自建

| 档 | 内容 |
|---|---|
| **可以完全沿用**（机制与外观都由框架给） | 悬停与按下的机制、焦点环、禁用变灰、只读行为、非激活窗口淡化、隐藏三态、滚动条显隐策略、输入法组合、流式 reveal、开合动画 |
| **可沿用但必须先配置**（改 token 即可，前提是先有标尺） | 选中色与 accent、`Danger`/`Warning`/`Success`、圆角、间距单位、字号、滚动条宽度、tooltip 延迟 |
| **必须我们在 view 层实现** | 忙碌与骨架屏（含按钮 loading）、错误呈现规范（位置/聚合/表单级 vs 字段级）、搜索命中高亮、展开箭头与图标切换、进场与曝光动画、**时长与缓动表**、密度档（compact/roomy）、阴影阶、空态 |

这正是 §7 第 1 条要我们拍板的东西：**框架给的是状态与 token 的「名词表」，缺的是「每个态长什么样」的规格**。

### 2.6 文本、字体与中文

| 事实 | 出处 |
|---|---|
| `Text` 按可用宽度换行；`Textf` 格式化 | `mygo:docs/ui/text.md`「Text」 |
| 样式：`FontSize` `FontWeight` `Bold` `Italic` `Font` `LineHeight` `FixedLineHeight` `TextColor` `TextAlign` `Underline` `WavyUnderline` `Strikethrough` `TextBackground` `LetterSpacing` `FontFeatures("tnum")` `TextShadow` `TextStroke` `TabSize`；设在上层元素上会被子孙继承 | 同上「Style」 |
| 截断与断行：`SingleLine`（省略号）、`MaxLines`、`Ellipsis(" →")`、`NoWrap`、`Balance`（标题等分两行）、`Pretty`（段末不留孤字） | 同上「Style」 |
| 行内混排：`RichText` + `ui.Span`（字体/字号/粗细/颜色/下划线/背景/圆角/`PaddingX/Y`） | 同上「Rich text」 |
| 句内元素：`ui.Link`、`ui.Icon` 可嵌在句子中并随行折行 | 同上「Elements within a sentence」 |
| 选择与复制：`Selectable` 可放在容器上把选择范围扩到所有子孙 `Text`/`RichText`；`CopyData` 自定义复制内容；`SelectedTexts` 取选中的片段；`Unselectable` 排除子树 | 同上「Style」 |
| 字体来自系统文本栈（Core Text / DirectWrite / Pango），默认用系统界面字体，缺字回退到系统字体与 emoji | 同上「Fonts」 |
| 自定义字体：`ui.RegisterFont(data, "Inter")`，或 `Font("Inter, Noto Sans JP")` 家族列表；`ui.FontFamilies()` 可列出系统字体（判断 `Monospaced`） | 同上「Fonts」 |
| 中文/日文字形：`Element.Lang("ja")`、`Span.Lang`、`Font.Lang` 决定汉字的字形变体 | `mygo:docs/i18n.md`「Native UI」 |
| 界面语言与区域格式：`c.Locale()`、`c.Region()`，每帧重建所以切语言即时生效 | `mygo:docs/i18n.md`「When the language changes」 |

### 2.7 事件、焦点与快捷键

| 事实 | 出处 |
|---|---|
| 指针是查询式：`Hovered` `Pressed` `Clicked` `DoubleClicked` `RightClicked` `Dragged` `DragVelocity` `DragEnded` `PressCanceled` `PointerPosition` `PressedInside` `ClickModifiers` | `mygo:docs/ui/input.md`「The pointer」 |
| 系统取消的按压（pointercancel）不产生 click/drop，`PressCanceled` 用来回滚手势 | 同上「A press the system cancels」 |
| 回调式：`OnClick` `OnChange` `OnSubmit` `OnShortcut`，在 build 与绑定输入之后、重绘之前执行 | `mygo:docs/ui/views.md`「Events and actions」 |
| Tab / Shift+Tab 移动焦点，键盘移动才显示焦点环；`TabStop(false)` 可聚焦但不进 Tab 序列；`AutoFocus`、`Focus`、`Focused`、`FocusVisible`、`FocusWithin`、`FocusLeft` | `mygo:docs/ui/input.md`「The keyboard focus」 |
| 持久焦点：`ui.Handle` + `.Bind(&app.search)`，可在控件隐藏时挂起请求，`Bounds(c)` 读已提交布局 | 同上「Persistent focus」 |
| 数据驱动焦点：`FocusBind(&app.pane, files)` / `ui.FocusedValue(c, &app.pane)` | 同上「Focus bound to data」 |
| 焦点组：`FocusGroup(ui.Horizontal)` 让一组控件只占一个 Tab 停靠点，方向键在组内移动 | 同上「Focus groups」 |
| 快捷键三层：`c.Shortcut(mods, key)`（全窗）、`Element.Shortcut`（元素焦点子树内）、`Handle.OnShortcut` / `c.OnWindowShortcut`（声明式，构建后按需触发）；`c.ShortcutRepeated()` 区分长按重复 | 同上「Shortcuts」 |
| `c.FocusTakesText()` 是「焦点正在接收文本」的守卫，命令面板/单键快捷键必须用它避让 | 同上「The keyboard focus」 |
| 按键状态：`c.KeyDown`、`c.KeysDown()`、`c.Modifiers()`，只观察不吞键 | 同上「Keys held」 |
| 逐事件输入：`HandleInput`（键按下/抬起、文本、编辑命令、指针、滚轮、捏合），`InputEvent.Code` 是 USB HID 键位，`TextCaret` 打开输入法组合 | 同上「Every key, as it comes」「Every key of the keyboard」 |
| 祖先旁听：`ObserveInput` 不改后续派发 | 同上「Input inside a region」 |
| 落文件：`DroppedFiles`、`FileDragOver` | 同上「Dropped files」 |
| 滚动与捏合事件：`InputScroll`（`Precise` 区分触控板与滚轮）、`InputZoom` | 同上「Scrolling and pinching」 |
| 大写：`ui.Cmd` 在 macOS 是 Command、其他是 Ctrl；`KeyBack`/`KeyForward` 接鼠标侧键 | 同上「Shortcuts」 |

### 2.8 滚动与虚拟列表

| 事实 | 出处 |
|---|---|
| `ui.List` 只构建可见行与少量越界行，行高可变：已见的测量、未见的按已见估算；百万行与十行同价 | `mygo:docs/ui/list.md`「List」 |
| `ListState`：`Key`（行身份，向上插行时滚动位置不跳）、`KeepOffset`、`FollowEnd` + `FollowAnimation`（聊天/日志）、`HoldPlace`、`Header`（分组头吸附）、`Label`（首字母跳转 + a11y 名） | 同上「List state」 |
| 选择：`Selected`（单选）/`Selection`（多选集合）/`DragSelect`/`ExtendTo`/`MenuKeepsSelection`；双击用 `RowDoubleClicked` 避免抢走行的点击 | 同上「Choosing rows」 |
| 滚动 API：`ScrollTo(row, align)`、`ScrollIntoView`、`ScrollToEnd`、`Visible()`、`AtEnd()` | 同上「Scrolling」 |
| 行内交互：`Reorder`（拖拽排序）、`DragRows`（把行拖出去） | 同上「Reordering」「Dragging rows out」 |
| 焦点行滚出视野仍保持构建，编辑不中断 | 同上「The focus」 |
| a11y：列表读作列表，每行报「5 of 10,000」，即使只建了可见行 | 同上「Accessibility」 |
| `Scroll`/`ScrollHorizontal`/`ScrollBoth`：滚轮、触控板、滚动条、键盘；`ScrollState` + `TrackScroll` 读写偏移；`ScrollTo(x,y,动画)`；`ScrollSnap` + `SnapAlign`；`ScrollbarInsets`、`Scrollbars(false)`、`KeepScrollbars(true)` | `mygo:docs/ui/scroll.md` |
| `Table` 复用 `ListState`，列可排序/改宽/移动，`CellSelect` 支持单元格块选择（类电子表格） | `mygo:docs/ui/table.md` |
| `Outline`/`OutlineTable` 是树的大数据集版本；`Tree` 是「在父项内部构建子项」的小树 | `mygo:docs/ui/outline.md`、`mygo:docs/ui/tree.md` |

框架自己的聊天范式（可以直接沿用的正解）：消息列表用 `ui.List` + `ListState{Key, FollowEnd, FollowAnimation: &ui.Smooth}` + `Justify(ui.End)`，见 `mygo:docs/ui/list.md`「A chat」与 `mygo:docs/ui/markdown.md`「A chat」。

### 2.9 窗口、标题栏与菜单

| 事实 | 出处 |
|---|---|
| `WindowOptions`：`Content`（native UI）、`StateKey`（记住位置尺寸）、`Width/Height`、`Min*/Max*`、`Hidden`、`Frameless`、`TitleBarStyle`、`TrafficLightPosition`、`HideWindowButtons`、`Parent`/`Modal`、`Panel`、`FitContent`、`Vibrancy`/`VibrancyTint`、`CornerRadius`、`BackgroundColor`、`Opacity`、`AlwaysOnTop`、`AutoHideMenuBar` | `mygo:docs/windows.md`「Options」 |
| 自绘标题栏：`TitleBarStyle: mygo.TitleBarHidden` + `c.TitleBar()`（控件占位）+ `DragWindow()`；`HideWindowButtons` 后自己画并调 `Minimize/Maximize/Unmaximize/Close` | `mygo:docs/ui/windows.md`「Title bars」 |
| 内容缩放：`c.SetZoom(z)`（0.25–5，矢量级清晰，不是位图放大）、`c.Size()` 返回内容 DIP | `mygo:docs/ui/windows.md`「Zoom」 |
| 材质：`c.Vibrancy()` 为真时根背景透明才有材质；`VibrancyGlass`/`VibrancyGlassClear` 是 macOS 26 的 Liquid Glass | `mygo:docs/ui/windows.md`「Vibrancy」 |
| 活动/非活动窗口：`c.WindowActive()`，内置控件自动跟随 | `mygo:docs/ui/windows.md`「Active and inactive windows」 |
| 隐藏时暂停工作：`c.WindowHidden()` | `mygo:docs/ui/windows.md`「Hidden windows」 |
| 菜单：`mygo.NewMenu([]*mygo.MenuItem{...})`，`MenuItem` 有 `Label` `Accelerator` `Click` `Type`（normal/checkbox/radio/separator/submenu）`Disabled` `Hidden` `ToolTip` `Icon` `Subtitle` `ID` `Role` | `mygo:docs/menus.md`「Templates」 |
| 内置 Role：编辑（undo/redo/cut/copy/paste/delete/selectAll）、窗口、应用菜单、切换全屏/开发者工具等；Role 的键先给内容 | `mygo:docs/menus.md`「Roles」 |
| 快捷键字符串语法：`CmdOrCtrl+N`、`Alt+Shift+F4`；修饰键 `CmdOrCtrl/Cmd/Ctrl/Alt/Shift`；键名表含 F1–F24、`Plus`、`Num0-9` 等 | `mygo:docs/menus.md`「Accelerators」 |
| 上下文菜单、托盘图标与托盘菜单、Dock 菜单 | `mygo:docs/menus.md`「Context menus」「Tray icons」「The Dock menu」 |
| 窗口内自绘菜单：`PopoverMenu`、`PopoverMenuButton`、以及「整窗自绘菜单」开关，a11y 角色与键盘齐备 | `mygo:docs/ui/context-menu.md`、`mygo:docs/ui/menu-button.md` |

### 2.10 原生对话框与桌面集成

| 事实 | 出处 |
|---|---|
| `mygo.Dialog.Open/Save/Message/Error` 是阻塞调用，要在绑定方法或 goroutine 里调；`Parent` 让它在 macOS 上作为 sheet 贴在窗口上 | `mygo:docs/native.md`「Dialogs」 |
| `OpenDialogOptions` 可选目录、显示隐藏文件、`FileFilter{MIMETypes:["image/*"]}`；`MessageOptions` 支持按钮、`DestructiveButtons`、`CheckboxLabel` | 同上 |
| 通知、全局快捷键、深链、单实例锁、最近文档、Now Playing、系统主题与无障碍设置 | `mygo:docs/native.md` 其余小节；`mygo:docs/app.md` |

### 2.11 无障碍

| 事实 | 出处 |
|---|---|
| VoiceOver / Orca / Narrator·NVDA 把 native UI 当原生应用读：角色、名字、值、状态，并且能操作控件 | `mygo:docs/ui/accessibility.md` 首段 |
| 命名：控件自带；无文本元素必须给 `Label`（同时也是 Tester 的查找键）；`LabelledBy` 指向 caption | 同上「Names」 |
| 角色：从行为推断（可点 + 可聚焦 = button，滚动容器 = scroll area），也可 `Role(ui.RoleSwitch)` 显式给；`RoleHeading`+`Level`、`RoleNavigation`、`RoleTable/Row/Cell/ColumnHeader`、`RoleGrid`、`RoleListBox/Option`、菜单系列 | 同上「Roles」 |
| 状态：`Checked` `Mixed` `AccessSelected` `Expanded` `Value` `Range` `Step` `Busy` `Invalid` `Current` `Sort` `PosInSet/SetSize` `ActiveDescendant` 等，对应 ARIA 的同类状态 | 同上「States」 |
| 描述与错误、方向、键盘快捷键、文本格式、列表与表格位置、grid 单元格坐标 | 同上「Descriptions and errors」「Orientation」「Keyboard shortcuts」「Formats of text」「Lists and tables」「Grids」 |
| 主动播报：`c.Announce(text)`；live region / landmark | 同上「Announcements」「Landmarks, live regions and rules」 |
| 可测：`tt.Access` / `tt.AccessByRole(role, name)` 返回 `ui.AccessInfo`（角色、名字、值、描述、全部状态、Bounds），断言的是「辅助技术拿到什么」而不是「view 设了什么」 | `mygo:docs/ui/testing.md`「What assistive technology sees」 |

### 2.12 测试

| 事实 | 出处 |
|---|---|
| `ui.NewTester(view, w, h)` 无窗口跑 view，按文本或 `Label` 找元素，可点击、输入、按键、滚动、开菜单、看剪贴板与播报 | `mygo:docs/ui/testing.md` |
| 测试器自带时钟，`tt.Advance(d)` 推进；不要 sleep | 同上「What a tester does」「Time」 |
| `tt.Image()` 拿最后一帧位图做快照；`ui.Render(view, w, h, scale)` 单帧渲染 | 同上；`mygo:docs/ui/rendering.md` |
| 端到端：automation 插件让外部进程按 `Label`/`Key` 读写真实窗口 | `mygo:docs/ui/testing.md`「Testing the built app」；`mygo:docs/plugins/automation.md` |

### 2.13 上游给出的性能数字（口径：作者机器、`CGO_ENABLED=0 GOMAXPROCS=1`、CPU 渲染、`ui.Tester`）

| 场景 | 每帧 | 分配 |
|---|---:|---:|
| 1,000 个静态 label | 397 µs | 0 |
| 百万行列表的可见行 | 14.5 µs | 20 次 / 176 B |
| 无改动的 diff 视图 | 389 µs | 2 次 |
| 50 KB Markdown 整篇显示 | 1.3 ms | 353 次 / 15.9 KB |
| 同一篇流式，每帧追加 16 B | 0.8 ms | 350 次 / 35.9 KB |

出处 `mygo:docs/ui/performance.md`；Markdown 解析本身：整篇 0.3 ms / 0.8 MB，追加 4 B 的 token 2 µs（`mygo:docs/ui/markdown.md`「Performance」）。同页另有 1,000 label 的常驻内存约 2.50 MB Go heap。

读法：**帧成本与「建了多少元素」成正比，与数据总量无关**（虚拟化把后者挡在外面）；流式追加的成本是常数级，不是一个 token 一帧。

---

## 3. kero 的实践

kero 是 mygo 生态里最大的公开实战项目（仓库根 ~2.3 万行 Go，19 个测试文件，tag v2.1.9），形态是「给编码 agent 用的终端」。它的价值不在终端，在**一个真实产品怎么用 mygo 组织界面**。

### 3.1 项目组织

| 做法 | 出处 |
|---|---|
| 单一可执行文件兼三职：GUI app、CLI（`kero help`）、daemon（`kero __kerod`），入口按 `os.Args[1]` 分派 | `kero:main.go` 文件头注释与 `main()` |
| 界面代码留在根包（`view.go` `side.go` `palette.go` `settings_view.go` `controls.go` `theme.go`），非界面能力进 `internal/`（`actions` `cli` `client` `daemon` `proto` `remote` `git` `settings` `update` `tailscale` …） | `kero:internal/` 目录 |
| 事实归 daemon（会话、pane、conversation），app 只存偏好：`prefs` 结构（选中的会话、侧栏宽、折叠集合、字号、侧栏 tab、右栏宽），JSON 落盘 | `kero:app.go` 的 `prefs` 与注释「the sessions themselves are kerod's」 |
| 多窗口：主窗（`main.go` `mygo.NewWindow`）、设置窗（`settings_view.go`）、快速终端 flyout（`quick.go`）、更新窗（`plugins/updater/native`） | 同左 |
| 应用元数据集中在 `mygo.json`（name/identifier/version/i18n 目录/updates 公钥/github 仓库/macOS 签名与公证/Info.plist 权限说明） | `kero:mygo.json` |

### 3.2 界面分层

kero 只有两层，没有 MVVM：

- **状态**：`App` 结构体持有全部界面状态与派生缓存（`a.pal`、`a.side`、`a.pal.…`）。
- **视图**：`func (a *App) view(c *ui.Context)` 是唯一顶层入口（`view.go`），向下每个区域一个 `*View(c, …)` 函数（`paletteView`、`settings_view.go`、`side.go` 里的各行构建器）。
- **命令**：`internal/actions` 是一张静态表，`Action{ID, Title, Menu, Keys, Remote, Global}`（`kero:internal/actions/actions.go:`开头 30 行），菜单栏、命令面板、可重绑快捷键、CLI 的 `kero do` 全部从这张表派生；执行统一走 `(*App).run(id, args)` 的 `switch id`（`kero:actions.go`）。

这三层的关系值得直接沿用：**「能做的事」只有一处登记，四处消费**。它同时解决了「菜单与快捷键不一致」「面板里缺某个命令」「CLI 与界面能力漂移」三个常见问题。

kero 的控件是自绘的但**不是自研底座**：`controls.go` 里 30 处 `ui.ButtonBase`、11 处 `ui.TextInputBase`、4 处 `ui.DialogBase`，在框架给的行为（指针、键盘、焦点、a11y）之上重画外观（macOS 27 风格的 pop-up button、带搜索的菜单、圆角文本框）。这正是框架文档推荐的姿势：`mygo:docs/ui/custom-widgets.md`。

### 3.3 键盘优先与命令面板

| 做法 | 出处 |
|---|---|
| 命令面板 = `DialogBase` 面板 + `TextInputBase` 输入框 + 自写 fuzzy 打分（标题命中额外加权、`words` 关键词命中、`top` 直达）+ `↑/↓`（含 Ctrl-N/Ctrl-P）+ Enter，选中项 `ScrollIntoView` | `kero:palette.go`（`paletteState`、`paletteView`） |
| 面板同时列会话、命令、历史会话，空查询时按类分组（会话 / 命令 / 历史）；`Esc` 关闭 | 同上 |
| 快捷键表是产品资产：`docs/shortcuts.md` 逐条列 macOS 与 Windows/Linux 两套键位并带 action id，平台差异显式写出（「Windows/Linux 上 Ctrl+字母属于 shell，所以加 Shift 或用 Alt」） | `kero:docs/shortcuts.md` |
| 有「跳到我该管的事」的快捷键：⌘J 跳到需要你的会话；⌘1–9 直选侧栏会话 | `kero:internal/actions/actions.go`（`session.attention` 等） |
| 键位可重绑，且与配置文件里的 `keybind` 同一套 id | `kero:docs/shortcuts.md` 首段 |

### 3.4 状态呈现方式

| 做法 | 出处 |
|---|---|
| 每个 pane 一个明确状态：Working / Waiting（需要你）/ Exited / Failed / Unread，用自绘 `statusIcon` 表达（`ui.Box` + `Label` + `Painter` 画环、弧、勾），Working 还能吃程序上报的 0–100 进度画弧 | `kero:view.go:946` `statusIcon` / `statusIconP` |
| 颜色集中在一份 `palette` 结构（明/暗两套 `lightPalette`/`darkPalette`），字段就是语义：`attention` `attnSoft` `done` `danger` `rowSelected` `card` `edge` `veil` … | `kero:theme.go` 开头 |
| 状态色之外还有「不打扰」的层级：非当前 pane 用 `veil` 淡化（同 Ghostty 的 unfocused split opacity） | `kero:theme.go` 注释 + `side.go` 的 `Opacity(box.Animate(...))` |
| 行高与栏高是集中常量（`barMid`、`topBar`、`headerH`、`cardGap`、`rowH`、`rowH2`、`rowInset`），标题栏内容按同一条中线对齐 | `kero:view.go` 常量块 |
| 分组的折叠、结束/归档区、展开的子 pane 都进 `prefs`，重启复原 | `kero:app.go` `prefs` |
| 变更列表用 `ui.List` + `ListState.Header` 让文件头吸附，diff 在头下滑 | `kero:side.go:781` 附近 |

### 3.5 能借鉴 / 不能借鉴

**能借鉴（并已列入 §6）**

1. 单一 action 表驱动菜单 + 面板 + 快捷键 + 外部 CLI（`kero:internal/actions/actions.go`、`kero:actions.go`）。
2. 偏好与事实分家：app 只存可丢的界面状态（`kero:app.go` 的 `prefs`），与 `docs/domains/A.md` 的「客户端只保留可丢的界面状态」同构。
3. 自绘外观建在 `*Base` 上，而不是自研焦点/键盘/a11y（`kero:controls.go`）。
4. 语义颜色结构 + 明暗两套调色板 + `AnimateColor` 过渡（`kero:theme.go`）。
5. 虚拟列表保滚动锚点、分组头用 `ListState.Header` 吸附（`kero:side.go`）。
6. view 层全部用 `ui.NewTester` 测（19 个测试文件、91 处调用），交互行为可回归。
7. 自绘标题栏：`TitleBarHidden` + `c.TitleBar()` + `DragWindow`，并在 Windows/Linux 自己画窗口按钮（`kero:platform.go` 的 `ownControls`）。

**不能借鉴**

1. **单包 main 放 2.3 万行。** 我们按 `docs/domains/A.md` 的模块切，界面代码进 `internal/ui/…`，不重复 kero 的做法。
2. **终端/浏览器/广告拦截/远程主机/隧道** 这些 kero 的核心业务与我们的产品无关，一行都不搬。
3. **键位策略**：kero 面向 shell，Ctrl+字母 属于 shell 所以加 Shift；我们没有这个约束，直接采用平台惯例即可。
4. **平台分支的窗口按钮逻辑**（`ownControls`/`showControls` 里对 Hyprland、Sway、KWin 的探测）属于 D10 宿主外壳的决定，不在这里继承。
5. **没有设计 token 体系。** kero 的调色板与行高常量是「看起来对」的经验值，没有字阶/间距标尺/状态矩阵文档；我们要在 C2 自己立（§7）。

---

## 4. markstream-go

### 4.1 版本、形状与依赖

| 事实 | 出处 |
|---|---|
| 模块 `github.com/zerob13/markstream-go`，`go 1.27.3`，**仓库没有任何 git tag**，只能用 commit 伪版本钉住（本次核对 `f73d35cb`，2026-11-10） | `markstream:go.mod`；`自测:markstream-tags` |
| CHANGELOG 当前条目为 0.2.0（2026-11-10），明确写「this entry does not create a release tag」 | `markstream:CHANGELOG.md` |
| 直接依赖：`mygo`（伪版本 `v0.8.4-0.20261109202725-…`）、`kyokomi/emoji/v2`、`chromedp` + `cdproto`（后两者只被 `rich` 包引用） | `markstream:go.mod`；`自测:markstream-deps` |
| 自述定位：**不复用 mygo 的 markdown 控件**，只复用它的底层 CommonMark/GFM 解析器、native UI/editor 原语；节点模型、渲染器、流式投影、reveal、代码面、虚拟文档、扩展注册都是自己的 | `markstream:README.md` 首段；`markstream:THIRD_PARTY.md` |
| 代码规模：根包 5,622 行 + 11 个测试文件 | `自测:markstream-size` |

### 4.2 API 形状

| 入口 | 签名/行为 | 出处 |
|---|---|---|
| `ms.View(c, source, Props)` | 传「到目前为止的完整源文」，返回 Element；同位置换 `Streaming:false` 即完成态 | `markstream:docs/api.md`「Views and identity」 |
| `ms.Stream(c, session, props)` | 订阅 `Session`，自动推导 `Streaming` | 同上 |
| `ms.Revealed(el)` | reveal 队列与已挂载异步工作是否排空（判断「显示完了」再挂时间/操作条） | 同上 |
| `ms.Session` | 零值可用，实现 `io.Writer`；`Append`/`Write`、`Finish`、`Cancel`、`Fail`、`Reset`、`Snapshot`、`Consume(ctx, r)`；UTF-8 分片缓冲、字节上限、快照不可变、通知合并、带 generation 的 reset | `markstream:docs/api.md`「Concurrent sessions and transport」 |
| `ms.Parser` | `Append(chunk, final)` / `Set`，持有增量语法文档与节点缓存；`Props.Nodes` 可绕过 UI 解析 | 同上「Node model and parser hooks」 |
| 钩子 | `SourceTransform`、`DocumentTransform`、`TransformNodes`、`NodeRenderers`、`CustomHTML`、`OnTaskToggle`、`Block` | 同上 |
| 代码与 diff | `ms.File(c, FileData, FileOptions)`、`ms.FileDiff(c, patch, opts)`、`ms.Compare(c, old, new, opts)`；`CodeOptions{HeaderRight, Filename: FenceFilename, Collapsible, MaxHeight, ShowFontSize}` | 同上「Code, File and Diff」 |
| rich | `rich.New(rich.Options{})` 单例引擎 + `Props.Rich`；渲染 math/mermaid/d2/infographic/html/svg | 同上「Rich engines and asynchronous graphics」 |
| 虚拟化 | `ms.VirtualState`（`Capture`/`Restore`/`Metrics`/`ScrollTo`）+ `Props.Virtual`；超过 `MaxLiveNodes`（默认 220）自动虚拟化 | 同上「Virtual documents and messages」 |
| 消息列表 | `ms.Messages(c, listState, []Message{ID, Source, Session}, props)` | 同上 |

### 4.3 流式更新的输入方式

三种，按「谁持有源文」区分：

1. **`Session`（推荐给网络流）**：生产者 goroutine 里 `Append` 增量文本或 `Write` 字节，UI 帧里 `ms.Stream` 读快照。SSE/provider JSON 由应用自己解码成文本增量，会话不解析 SSE 帧（`markstream:docs/api.md`「Concurrent sessions and transport」）。
2. **应用自持源文**：`ms.View(c, sourceSoFar, ms.Props{Streaming: true})`，适合我们这种「事实来自 App Server、每帧从投影取当前文本」的架构——不需要额外一个 `io.Writer` 生命周期。
3. **节点输入**：`Props.Nodes` 传已解析节点，UI 不再解析；需要保持 `NodeID` 稳定。

增量成本：追加文本只重解析「仍打开的最外层块」，已完成块不重解析、不重布局；节点 ID 在编辑与前插后保持稳定（`markstream:README.md` Features 第 2 条）。

### 4.4 代码块与表格

| 点 | 事实 | 出处 |
|---|---|---|
| 代码高亮 | 用 mygo `editor` 的 tokenizer 与配色，按行接续上一行状态；行分块布局，增长时只重排最后一块 | `markstream:README.md`；`markstream:docs/api.md`「Code, File and Diff」 |
| 代码块外壳 | 语言/文件名头、复制按钮（fence 未闭合时等待）、折叠、换行开关、字号控件、`HeaderRight` 自定义槽 | 同上 |
| 覆盖顺序 | 先按语言的 `Code` 钩子 → 通用 `CodeBlock` → 配置的 rich 渲染器 → `PlainCode`/`RenderCode` | 同上 |
| 表格 | GFM 表格；表头「等分隔行」才成表，避免流式时闪成一行管道符；主题控制表线/表头/斑马纹/内边距/最大列宽 | `markstream:README.md`；`markstream:docs/api.md`「Reveal, theme and labels」 |
| 表格 a11y | table / row / columnheader / cell 角色 | `markstream:README.md`（a11y 段） |

### 4.5 对 view 层的接入点

- 唯一入口是 `ms.View` / `ms.Stream`，返回普通 `ui.Element`：可以放进 `ui.List` 的一行、`ui.Scroll` 的子节点、或 `ms.Messages` 的列表。
- 每条消息用 `c.Key(messageID)` 保持身份；「同一位置从 Stream 换 View」是官方要求（`markstream:docs/api.md`「Views and identity」）。
- 选择与复制：跨段原生选择，Copy 同时产出纯文本 / HTML / Markdown；`ShareSelection` 可把选择权交给外层容器（对应框架的 `Selectable`）。
- 链接：`OnLink` 只收到 `SafeLink` 通过的目标；`LinkMenu` 给单个链接配菜单（可以复用 mygo 的 `Menu.EditItems`）。
- 图片：默认后台加载、有字节与像素上限、`AllowURL` 每次重定向都检查；`LoadImage` 可替换（我们可以接工作区的只读文件）；`DisableImages` 只留描述。
- rich 引擎：应用级单例，`Close` 在退出时调用。

### 4.6 性能特征（上游自测）

| 场景 | 时间 | 分配 |
|---|---:|---:|
| 增量底层文档（6.5 KB 语料、64 B 分片） | 0.458 ms | 270 次 / 363 KB |
| 每个收到的前缀都整篇重解析（对照组） | 13.978 ms | 12,028 次 / 16 MB |
| 稳态 native 帧（800×700 滚动视图） | 1.113 ms | 899 次 / 115 KB |
| 5,000 节点虚拟文档帧（800×600） | 0.779 ms | 308 次 / 32.5 KB |

口径：Linux amd64、Go 1.27.3、4 个可用 P、20 次迭代的开发期采样，**不含 GPU 帧时间与图形引擎启动**（`markstream:docs/performance.md`）。作者自己列的限制：**完整 AST 常驻**；超长未闭合段落仍要重解析；节点投影不是每 token 常数；高度总量含未测块估算。

### 4.7 与 mygo `ui/markdown` 的对照（决定 §7 第 5 条）

| 维度 | `mygo/ui/markdown`（v0.8.3 内建） | `markstream-go` |
|---|---|---|
| 引入代价 | 随框架，零第三方 | 新第三方依赖（需改 `AGENTS.md` §3 的例外条款）+ 伪版本钉法 |
| 流式 | `Stream` + `Append` 只重解析未闭合块；`Reveal` 五档节奏 + 淡入 + ReduceMotion | 同级，另带 grapheme 级字符 reveal |
| 虚拟化 | 无内建虚拟化（整篇建元素）；官方建议「很长的文档分段显示」 | 内建（节点虚拟化 + 消息列表虚拟化 + 位置快照） |
| 代码块 | 高亮 + Copy + 语言头 + `CodeBlock` 钩子 | 同 + 折叠/字号/文件名识别/`HeaderRight` |
| 额外能力 | 链接菜单、Anchor、脚注、CJK 强调修正、CommonMark 0.31.2 全通过 | File/FileDiff/Compare、rich（Mermaid/KaTeX/D2/Infographic）、emoji、custom HTML |
| 依赖风险 | 低 | rich 需本机 Chromium；无 tag；AST 常驻 |
| 许可 | MIT（随 mygo） | MIT，但 rich 引入 MPL-2.0 的 D2（§5.4） |

---

## 5. 许可证核对

### 5.1 结论表

| 组件 | 版本/钉法 | 许可 | 可商用 | 义务 |
|---|---|---|---|---|
| mygo | v0.8.3（tag），也可钉伪版本 | MIT，Copyright (c) 2026 MyGo contributors（`mygo:LICENSE`） | 是 | 分发时保留版权与许可声明 |
| kero | tag v2.1.9 / commit `09eb1d42`；仓库内 `mygo.json` 版本 2.1.35 | MIT，Copyright (c) 2026 EGOIST（`kero:LICENSE`、`kero:README.md`「License」） | 是 | 若复制其代码片段，保留 EGOIST 的版权与许可声明；只借鉴做法不产生义务 |
| markstream-go | 无 tag，commit `f73d35cb` | MIT，Copyright (c) 2026 markstream-go contributors（`markstream:LICENSE`） | 是 | 保留声明；`rich` 子包另带第三方资产（见 5.4） |
| AI Logo 图标 | tag `mygo/v1.20.0`（Go 模块 `github.com/yldm-tech/ai-logo/mygo`，仓库 `package.json` 版本 1.20.0） | MIT，Copyright (c) 2023 LobeHub + (c) 2026 yldm-tech；站点与 README 标 MIT，并注明「Brand marks belong to their respective owners and are used here to identify them」；站点标题行仍显示 v1.19.0，与仓库 1.20.0 有一版之差，取仓库 tag 为准 | 是 | 保留 MIT 声明（含 LobeHub 的原始版权行）；非 AI 图标另见 5.3；商标仅作识别用途 |

### 5.2 mygo / kero / markstream 的核对依据

- mygo：`mygo:LICENSE` 正文为 MIT，「Copyright (c) 2026 MyGo contributors」；仓库 `README.md` 未额外限制；无 copyleft 组件（依赖为 purego、go-text/typesetting、x/image、x/net、x/text、mp3/ogg 解码，均 BSD/MIT 类）。
- kero：`kero:LICENSE` 为 MIT，版权人 EGOIST；`kero:README.md` 末尾「License: MIT」。kero 内置 JetBrains Mono Nerd Font，`kero:fonts.go` 注明 `resources/fonts` 为 **SIL Open Font License**（`OFL.txt`）——只有我们**打包该字体**时才需要遵守 OFL（保留 OFL 文本、改名限制）。我们默认用系统字体，不打包即不触发。
- markstream：`markstream:LICENSE` 为 MIT；`markstream:THIRD_PARTY.md` 明列：MyGo（MIT）、markstream-vue（MIT，仅架构参考、未拷贝源码）、Lobe MyGo（MIT，仅架构参考）、kyokomi/emoji v2（MIT）、chromedp/cdproto（沿用上游许可）。

### 5.3 图标来源（ailogo.yldm.ai）的授权情况

核对渠道：站点正文、仓库 `LICENSE`、`THIRD_PARTY_NOTICES.md`、`mygo/README.md`、`mygo/go.mod`、git tag（`自测:ailogo`）。

| 事实 | 出处 |
|---|---|
| 站点与仓库均标 **MIT**；`LICENSE` 保留 LobeHub 2023 与 yldm-tech 2026 两行版权 | 站点正文「MIT licence」；`ailogo:LICENSE` |
| 是 `lobehub/lobe-icons` 的 fork 并保持 MIT | `ailogo:THIRD_PARTY_NOTICES.md`「lobe-icons」 |
| 品牌图形（AI 供应商/模型）来自 lobe-icons 体系；**商标归各所有者，仅用于识别** | 站点「Brand marks belong to their respective owners…」 |
| 非 AI 服务图标移植自 **Simple Icons v16.32.0**，其图标数据为 **CC0-1.0**，但逐条列出例外：Yarn / Angular / Vivaldi / Godot / Solidity 为 CC-BY-4.0，Git / Android 为 CC-BY-3.0，Deno / JavaScript / Storybook / Sanity / Bootstrap 为 MIT，OpenJDK 为 BSD-3-Clause，Apache Kafka 为 Apache-2.0；**share-alike / 非商用 / 禁改的图形已被刻意剔除** | `ailogo:THIRD_PARTY_NOTICES.md`「Simple Icons and other brand sources」表 |
| Go 模块独立成子模块 `github.com/yldm-tech/ai-logo/mygo`，可单独钉版本（`mygo/v1.20.0`），`go 1.27.3`，依赖 `mygo v0.7.4`（低于我们的目标版本，属向上兼容方向，但需实测） | `ailogo:mygo/go.mod`；`自测:ailogo-tags` |
| 用法：`ailogo.MustParse("openai", ailogo.Mono)` → `ui.Icon(c, logo)`；变体 `mono/color/text/text-color/text-cn/brand/brand-color`；SVG 打在 `icons.zip` 里按需解析，不用的图标不付费 | `ailogo:mygo/icons.go` 包注释；`自测:ailogo-zip` |
| 规模：`src/` 下 1,326 个品牌目录，`icons.zip` 2,709 个 SVG | `自测:ailogo-zip` |

**可商用结论：可以**，条件是把 MIT 声明（含 LobeHub 版权行）与所用 CC-BY/Apache 条目的归属一起带进我们的 `THIRD_PARTY` 记录；品牌标识只用于「表示这家供应商」，不做背书暗示。若只用 AI 供应商图标（我们的主要用途），落到 MIT + 商标注意即可。

**待确认（唯一一处）**：CC-BY-4.0/3.0 的 5–7 个服务图标要求「署名」。我们只挑选用得到的少数品牌图标，届时按来源清单逐条记录署名即可；如果 C2 选定图标集时不需要这些品牌，则该项自动消解。这不影响「能否用」的结论。

### 5.4 若启用 markstream 的 `rich` 包，额外义务

- Mermaid 11.12.2、KaTeX 0.16.25、DOMPurify 3.3.3、AntV Infographic 0.2.3 保留各自上游许可（通知随资产分发）。
- **D2.js 0.1.33 为 MPL-2.0**，需保留 `rich/LICENSE.d2` 并提供对应上游源码地址（文件级 copyleft，不传染我们的代码，但分发时要满足其条款）。
- 结论：**第一阶段不启用 `rich`**；需要图表/公式时先在 C4 单独评估（另见 §8「外部进程」）。

---

## 6. 默认可继承项清单

规则：**框架已经解决的选择默认沿用**；偏离要写理由（`docs/project-organization.md` §9.2）。每条都给了可核对的出处。

### 6.1 视图与状态

| `[沿用]` 项 | 出处 | 适用画面 |
|---|---|---|
| view 写成「状态 → 界面」的函数，每帧重建；子树用 `Children(func(){...})` | `mygo:docs/ui/views.md`「Views」 | 全部画面 |
| 元素值只在一次 build 趟内有效；跨帧身份用 `ui.Handle` / `ListState` / `ScrollState` / `Router` 且放 app 状态 | `mygo:docs/ui/views.md`「Views」「Keys」 | 全部有滚动或焦点的画面 |
| 后台结果回主线程用 `Window.Update`；只重绘用 `Invalidate` / `RunOnMain` | `mygo:docs/ui/views.md`「Change the state from other goroutines」 | 流式回复、连接状态、目录刷新 |
| 集合项用 `c.Key(id)` 给稳定身份，容器 Key 划身份作用域 | `mygo:docs/ui/views.md`「Keys」 | 会话列表、消息时间线、分支切换 |
| 值控件传指针即双向绑定，就地用 `Changed()`/`Submitted()` 判断 | `mygo:docs/ui/views.md`「Events and actions」 | 输入区、设置表单、审批备注 |
| 会改「正在遍历的集合」的动作放 `OnClick`/`OnChange` 回调 | 同上 | 删除消息、会话行右键菜单 |

### 6.2 布局

| `[沿用]` 项 | 出处 | 适用画面 |
|---|---|---|
| `Column`/`Row`/`Box` + `Gap`/`Padding`/`Margin`，不写坐标 | `mygo:docs/ui/layout.md`「Containers」「Spacing」 | 全部画面 |
| `Grow`/`Shrink`/`Fill`/`Min*`/`Max*`（含 Percent）表达伸缩 | 同上「Sizes」 | 侧栏 + 主区、输入区贴底 |
| `Justify`/`AlignItems`/`Baseline`/`Center` 表达对齐 | 同上「Alignment」 | 工具条、元信息行、表格数字右对齐 |
| `Absolute` + `Attach` 处理角标与浮层 | 同上「Positioning」 | 未读角标、浮动「回到最新」 |
| `Sticky` 处理滚动容器内的吸附头 | 同上；`mygo:docs/ui/scroll.md`「Sticky elements」 | 日期分隔、设置分页头 |
| `Clip` / `Invisible` / `Hidden(true)` 区分三种「不显示」 | 同上「Clipping and visibility」 | 折叠面板、切换分页 |
| `Spacer` / `Divider` / `Dividers(w,c)` 画分隔 | 同上「Containers」 | 列表分组、工具条分组 |

### 6.3 主题与外观

| `[沿用]` 项 | 出处 | 适用画面 |
|---|---|---|
| `c.Theme()` 明暗主题跟随系统；改副本后 `c.SetTheme` | `mygo:docs/ui/styling.md`「Themes」 | 全局 |
| `Spacing` 作为唯一间距单位（`t.Space(n)`） | 同上「Themes」 | 全部内边距与间隙 |
| 主题的 `FontSize` / `Radius` / `ScrollbarWidth` 控制全局观感 | 同上 | 全局 |
| `Accent` / `InactiveAccent` / `Focus` / `FocusRingWidth` / `FocusRingOffset` | 同上「Themes」「The focus ring」 | 主按钮、选中行、焦点环 |
| `Inverse` / `InverseText` 给 tooltip 与 toast 做反色 | 同上「Themes」 | 快捷键提示、结果反馈 |
| 状态即代码：用 `Hovered()` / `Pressed()` 改自身样式，不引入选择器体系 | 同上「State, not selectors」 | 行 hover、按钮按下、卡片选中 |
| `AnimateColor` / `AnimateTo` + `ui.Smooth` 等曲线，`ReduceMotion` 自动降级 | `mygo:docs/ui/styling.md`；`mygo:docs/ui/motion.md`；`mygo:docs/ui/transitions.md` | 侧栏收放、选中过渡、流式淡入 |
| `c.Preferences()` 的 `Accent` / `HighContrast` / `TextScale` / `ReduceMotion` / `Pointer` | `mygo:docs/ui/styling.md`「The desktop's preferences」 | 全局（高对比、字号、触控目标） |

### 6.4 文本

| `[沿用]` 项 | 出处 | 适用画面 |
|---|---|---|
| `Text`/`Textf` + 继承式文本样式链 | `mygo:docs/ui/text.md`「Text」「Style」 | 全部正文 |
| `SingleLine` / `MaxLines` / `Ellipsis` / `NoWrap` 做列表行截断 | 同上「Style」 | 会话标题、工具名、路径 |
| `Balance` / `Pretty` 改善标题与段末断行 | 同上「Style」 | 空态文案、说明段 |
| `RichText` + `ui.Span` 做行内混排与内联代码 | 同上「Rich text」 | 正文内代码、状态徽标 |
| `Selectable` + `CopyData` 做跨段选择与格式化复制 | 同上「Style」 | 消息正文、日志、错误详情 |
| 系统字体栈 + `Font("monospace")`；需要时 `RegisterFont`、`FontFamilies` | 同上「Fonts」 | 全局、代码块、字体设置页 |
| `Element.Lang` 控制中文/日文字形 | `mygo:docs/i18n.md`「Native UI」 | 中文正文 |
| `c.Locale()` / `c.Region()` 取语言与区域格式 | `mygo:docs/i18n.md`「When the language changes」 | 全部文案与数字格式 |

### 6.5 输入、焦点、快捷键

| `[沿用]` 项 | 出处 | 适用画面 |
|---|---|---|
| 查询式指针事件（`Hovered`/`Clicked`/`RightClicked`/`PressCanceled`…） | `mygo:docs/ui/input.md`「The pointer」 | 全部可点区域 |
| 三层快捷键：`c.Shortcut` / `Element.Shortcut` / `Handle.OnShortcut` 与 `OnWindowShortcut` | 同上「Shortcuts」 | 全局快捷键、输入区快捷键 |
| `c.FocusTakesText()` 守卫单键快捷键 | 同上「The keyboard focus」 | 命令面板、单键切换 |
| Tab 序列 + `TabStop(false)` + `FocusGroup` | 同上「The keyboard focus」「Focus groups」 | 卡片、工具条、分段控件 |
| `FocusBind(&field, value)` / `ui.FocusedValue` 做数据驱动焦点 | 同上「Focus bound to data」 | 面板切换、对话框默认焦点 |
| 覆盖层关闭后把焦点还给打开它的元素 | `mygo:docs/ui/overlays.md`「The keyboard」 | 对话框、重命名、命令面板 |
| `HandleInput` 收逐事件（终端、画布），`TextCaret` 打开输入法 | `mygo:docs/ui/input.md`「Every key, as it comes」 | 终端块、代码编辑器 |
| `ObserveInput` 让容器旁听而不拦截 | 同上「Input inside a region」 | 快捷键提示、演示中止 |
| 落文件用 `DroppedFiles` / `FileDragOver` | 同上「Dropped files」 | 附件拖入输入区 |

### 6.6 列表、滚动与表格

| `[沿用]` 项 | 出处 | 适用画面 |
|---|---|---|
| `ui.List` 的虚拟化（只建可见行 + 高度估算） | `mygo:docs/ui/list.md`「List」 | 时间线、会话目录、审批队列 |
| `ListState.Key` 在向上插行时保滚动位置与行状态 | 同上「List state」 | 时间线向前翻页 |
| `FollowEnd` + `FollowAnimation(&ui.Smooth)` 的聊天范式 | 同上「List state」「A chat」 | 会话时间线 |
| `ListState.Header` 让分组头吸附 | 同上「List state」 | 按日期/轮次分组 |
| `Selected` / `Selection` / `DragSelect` / `ExtendTo` 做单选与多选 | 同上「Choosing rows」 | 目录多选、批量操作 |
| `ScrollTo` / `ScrollIntoView` / `ScrollToEnd` / `Visible` / `AtEnd` | 同上「Scrolling」 | 跳到某条消息、加载更早、跳到最新 |
| `Scroll` + `TrackScroll` + `ScrollSnap` + `ScrollbarInsets` + `KeepScrollbars` | `mygo:docs/ui/scroll.md` | 设置页、长文档、横向滚动代码块 |
| `Table` + 同一套 `ListState`，列可排序/改宽/移动 | `mygo:docs/ui/table.md` | 用量与成本表、插件列表 |
| `Outline`/`OutlineTable` 承载大树/大表 | `mygo:docs/ui/outline.md` | 文件树、技能与钩子列表 |
| `Tree` 承载「在父项内构建子项」的小树 | `mygo:docs/ui/tree.md` | 设置分类、嵌套配置 |

### 6.7 覆盖层与反馈

| `[沿用]` 项 | 出处 | 适用画面 |
|---|---|---|
| `ui.Modal(&open, fn)` + `Dismissed()` | `mygo:docs/ui/dialog.md` | 重命名、确认、命令面板 |
| `ui.AlertDialog(&open, 标题, 说明, 按钮…)` | `mygo:docs/ui/alert-dialog.md` | 删除会话、危险审批 |
| `Popover` + `AttachTo` + `AttachEdgeMargin` + `AttachedAnchors` | `mygo:docs/ui/overlays.md`「Your own overlays」 | 模型选择、更多菜单、筛选器 |
| `Tooltip` / `c.AddToast` / `ui.Toast` | `mygo:docs/ui/tooltip.md`、`mygo:docs/ui/toast.md` | 快捷键提示、复制成功、失败反馈 |
| `ui.Overlay` 做顶层横幅 | `mygo:docs/ui/overlays.md` | 断线横幅、更新提示 |
| Esc 关最上层、点击外部关 popover、对话框焦点不外泄 | 同上「The keyboard」 | 全部覆盖层 |

### 6.8 窗口与桌面

| `[沿用]` 项 | 出处 | 适用画面 |
|---|---|---|
| `WindowOptions` 的 `Content` / `StateKey` / `Min*` / `Parent` / `FitContent` | `mygo:docs/windows.md`「Options」 | 主窗、设置窗、快速入口 |
| 自绘标题栏：`TitleBarHidden` + `c.TitleBar()` + `DragWindow()` | `mygo:docs/ui/windows.md`「Title bars」 | 主窗顶部 |
| `c.SetZoom`（0.25–5）做界面缩放 | 同上「Zoom」 | 视图菜单的放大/缩小/实际大小 |
| `c.WindowActive()` / `c.WindowHidden()` 跟随窗口状态 | 同上「Active and inactive windows」「Hidden windows」 | 全局 |
| 菜单模板 + `Role` + `Accelerator` 字符串语法 | `mygo:docs/menus.md`「Templates」「Roles」「Accelerators」 | 菜单栏、上下文菜单、托盘 |
| 原生对话框 `Dialog.Open/Save/Message`（阻塞，走 goroutine） | `mygo:docs/native.md`「Dialogs」 | 选工作目录、导出、覆盖确认 |

### 6.9 无障碍与测试

| `[沿用]` 项 | 出处 | 适用画面 |
|---|---|---|
| 无文本元素必须给 `Label`；能显式给 `Role` 就给 | `mygo:docs/ui/accessibility.md`「Names」「Roles」 | 图标按钮、自绘卡片、状态点 |
| 自绘控件上报 ARIA 对应状态（`Checked`/`Expanded`/`Value`/`Range`/`Busy`/`Invalid`…） | 同上「States」 | 自绘开关、进度、错误态 |
| 关键状态变化用 `c.Announce` 播报 | 同上「Announcements」 | 审批到达、连接断开、任务结束 |
| view 层用 `ui.NewTester` 覆盖交互，用假时钟而非 sleep | `mygo:docs/ui/testing.md` | 全部 view 的测试基线 |
| 用 `tt.AccessByRole` 断言辅助技术拿到的角色/名字/状态 | 同上「What assistive technology sees」 | 关键画面（输入区、审批、列表） |
| `mygo vet` 纳入 CI，拦住 build 值逃逸 | `mygo:docs/ui/views.md`「Check build lifetimes」 | 全局 |

### 6.10 流式 Markdown（按 `ui/markdown` 计；换 markstream 见 §7 第 5 条）

| `[沿用]` 项 | 出处 | 适用画面 |
|---|---|---|
| `markdown.Stream(c, src, &opts)` 与完成态 `markdown.View` 同位置切换 | `mygo:docs/ui/markdown.md`「Streaming replies」 | 助手回复 |
| `Options.Reveal{Unit, CatchUp, Fade}` 的节奏与淡入（含 ReduceMotion 降级） | 同上「Pace and fade」 | 助手回复 |
| 代码块：语言高亮、Copy（fence 未闭合时等待）、`CodeBlock`/`Code` 钩子 | 同上「Code」 | 回复里的代码与工具输出 |
| 链接 `OnLink` / `LinkMenu` / `Anchor`，图片 `LoadImage` / `FileImages` | 同上「Links and images」 | 回复里的链接、附件预览 |
| 选择与「Copy as Markdown」 | 同上「Selecting and copying」 | 回复复制 |
| a11y：标题/列表/链接/表格/任务项按语义上报 | 同上「Accessibility」 | 助手回复 |

**合计：72 条**（分节：视图 6、布局 7、主题 8、文本 8、输入 9、列表 10、覆盖层 6、窗口 6、无障碍与测试 6、流式 6）。

---

## 7. 需要我们自己决定的部分

以下都是「框架明确不做、或给了两条以上等价路线」的项。每条写清：框架现状、谁定、何时定。

| # | 要决定的事 | 框架现状（不出） | 谁定 | 何时 |
|---|---|---|---|---|
| 1 | **视觉标尺**：字阶（H1–H4/正文/说明/代码）、间距刻度、圆角与描边阶梯、语义色（正文/次要/禁用/危险/成功/强调/选中）、三态（default/hover/active/disabled/focus） | 只给 `Spacing`（默认 4）、`FontSize`、`Radius`、`Accent` 四个标量 + 一组样式原语；没有设计 token，没有状态矩阵（现状与缺口清单见 §2.5） | U2 | C2 |
| 2 | **信息架构与页面/面板划分**：单窗多区还是多窗；会话区、目录、审批、设置、扩展、终端各自占哪里 | `Router` 提供页面路由与历史（`mygo:docs/ui/navigation.md`），`Sidebar`/`Tabs`/`Split` 提供容器；划分本身不给 | U1 | C1 |
| 3 | **虚拟化单位**：一条消息一行（官方聊天范式）还是「轮次一行」/「块一行」；超长消息内部是否再分页 | `List` 的虚拟化以「行」为单位，行高可变但必须可测量；框架不提供「行内再虚拟化」 | U4 | C4 |
| 4 | **流式 Markdown 选型**：`ui/markdown` 还是 `markstream-go`（§4.7 对照） | 两条等价路线，都能跑 | 用户（涉第三方依赖例外） | C4 前，先出设计选择（`docs/domains/A.md` 体例） |
| 5 | **命令面板的范围与索引**：面板里放什么（Go 侧动作 / 会话 / 分支 / 设置项 / 服务端能力），谁提供清单 | `DialogBase` + `TextInput` + `List` 可拼出面板，但「有哪些命令」不给 | U3 | C3 |
| 6 | **审批与提问的呈现方式**：覆盖层强制呈现、还是时间线内联卡 + 侧栏队列；一次一条的强制定序怎么被用户感知 | `Modal`/`AlertDialog`/内联 `Card` 都可行；「一次只推一条」的顺序语义属于 A 的交互裁决，界面呈现未定 | 用户（B2 拍板概念） | B2 |
| 7 | **分支/目标的可见性**：切换器放标题栏、侧栏、还是时间线内联；已结束分支的只读态怎么表达 | 无内建「分支切换器」控件；`Segmented`/`Popover`/`Breadcrumbs` 都可用 | U1 | C1/C3 |
| 8 | **终端块**：用 mygo `plugins/terminal`（libghostty-vt 运行时加载）还是 D9 的 PTY 方案；两者边界（会话内 shell vs 本机 shell） | 插件提供完整终端视图（`mygo:docs/plugins/terminal.md`），但引入一个运行时原生库与 `go generate` 步骤 | D9 + U4 | C4 |
| 9 | **图标集**：品牌图标用 ailogo；界面图标用哪套（Lucide/Heroicons/Tabler/Material Symbols），打包方式与许可登记 | `ui.Icon` 直接吃 SVG，`ui.MustParseSVG` 解析；图标集本身不给 | U2 | C2 |
| 10 | **字体策略**：纯系统字体，还是打包一套 UI 字体 + 代码字体（会触发 OFL 义务与体积） | 系统文本栈为默认；`RegisterFont` 可注册 | U2 | C2 |
| 11 | **平台差异**：Windows/Linux 是否自绘窗口按钮、快捷键如何在 `CmdOrCtrl` 之外分平台、Linux 无 WebKitGTK 依赖下的最小运行环境 | `TitleBarStyle`/`HideWindowButtons`/`Accelerator` 语法给了机制；策略不给 | D10 + U4 | C4 |
| 12 | **多窗口策略**：设置、快速入口、通知详情是独立窗口还是同窗面板（kero 三样都用独立窗口） | `Parent`/`Panel`/`Flyout` 都支持 | U3 | C3 |
| 13 | **无障碍基线**：哪些自绘元素必须显式给 Role、哪些状态变化必须 Announce、验收用什么断言 | 能力齐（角色/状态/播报/可测），基线标准不给 | U6（E 阶段判定） | C3/E |
| 14 | **view 层测试基线**：哪些画面必须 `ui.Tester` 覆盖、快照要不要进仓库 | `ui.Tester` + `tt.Image()` 都在；门槛不给 | U5 + 组织 | D 阶段前 |
| 15 | **工具链升级**：`go.mod` 的 `go` 指令升到 `1.27.3`（mygo 的硬要求），并明确 CI 用同一工具链 | 无选择余地，但改动落在谁身上要定 | 组织 + D12 | 首批实现前 |

---

## 8. 风险与限制

### 8.1 纯 Go 渲染的性能边界

- 框架按「元素数」而不是「数据量」计费：1,000 label 397 µs/帧、百万行列表的可见行 14.5 µs/帧（`mygo:docs/ui/performance.md`）。**所以风险不在总量，在「一屏里塞了多少元素」**：一条长消息整篇建成元素就是几千个元素。
- 流式追加是常数成本（每帧 +16 B 时 0.8 ms，整篇 1.3 ms，`mygo:docs/ui/markdown.md`「Performance」），但每帧要重建「所有可见块」的元素；官方明确说「很长的文档宜分段显示」。
- GPU 为主（Metal/D3D11/OpenGL），Linux 上 OpenGL 加载占约 50 MB；无 GPU 时回退 CPU（大窗口约 1 ms/帧，`mygo:docs/ui/rendering.md`）。
- 空闲不画帧；两秒后释放内存帧；macOS 动画期保留三帧（约 4 B/像素）（同上）。
- 真实窗口的帧时间、GPU 上采样、滚动性能**本次未实测**（§9）。

### 8.2 滚动大列表

- `List` 自带虚拟化与高度估算，向上插行时靠 `Key` 保位（`mygo:docs/ui/list.md`）。真正的坑是**行高不可测**：折叠的 thinking、展开的工具输出、图片加载完成都会改变行高，估算误差会表现为滚动跳动。对策：行内给固定可测高度上限（折叠 + 「展开」），图片先占位。
- 官方限制：焦点行滚出视野仍保持构建（这是有意的，代价是常驻元素）；`Selectable` 只覆盖已构建的段落，**虚拟列表里做整篇选择会漏**——`mygo:docs/ui/text.md`「Selectable」明写「use a regular scroll container when selecting an entire document」。
- 百万行级别不是我们的场景（时间线以千条计），但「一次导出/一次全选」会触到这条边界。

### 8.3 字体与输入法

- 文本由系统文本栈排布（Core Text / DirectWrite / Pango），默认系统界面字体；中文/日文字形靠 `Lang` 区分（`mygo:docs/ui/text.md`「Fonts」、`mygo:docs/i18n.md`）。**跨平台字形一致性不在框架承诺内**：同一个汉字在 macOS 与 Windows 上可能不同字形。
- 输入法：普通 `TextInput`/`TextArea` 开箱可用（`mygo:docs/ui/input.md`「Input methods」）；一旦用 `HandleInput` 自绘编辑器，必须同时 `TextCaret(...)` 才能组合输入，`InputMethod(false)` 用于关掉组合但仍要打字的状态（同「Every key, as it comes」）。
- 终端块（本机 shell）依赖插件与运行时原生库（`mygo:docs/plugins/terminal.md`），其 IME 与全屏程序表现需单独验证。
- Linux 上 IME 取决于 GTK 与桌面环境；Windows AltGr 事件形状与 macOS Caps Lock 的行为都有已知差异（`mygo:docs/ui/input.md`「Every key of the keyboard」末尾的限制清单）。

### 8.4 无障碍现状

- 覆盖面好：角色、名字、值、状态、位置、landmark、live region、主动播报、可断言（`mygo:docs/ui/accessibility.md` 全篇）。
- 但**自绘即自证**：任何我们画出来的非标准控件（状态点、进度环、自绘行、自定义卡片）默认对辅助技术不存在，必须自己给 `Label`/`Role`/状态。kero 的 `statusIcon` 就是正面例子（`kero:view.go:946` 里逐个 `Label(locales.Status…)`）。
- 虚拟列表报「5 of 10,000」由框架处理，但**滚动到视野外需要屏读器触发滚动**才能建行（`mygo:docs/ui/list.md`「Accessibility」），行为依赖平台，需在 E 阶段实测。
- 本次未在带屏读器的真实窗口上验证（§9）。

### 8.5 依赖与版本风险

- **mygo 未到 1.0**：最新 tag 是 v0.8.3（`自测:mygo-tags`），而 kero 与 markstream 都已钉到 v0.8.3 之后的伪版本（`kero:go.mod`、`markstream:go.mod`），说明上游 API 仍在动。我们要么钉 tag v0.8.3，要么钉伪版本；升级要按「框架行为变化」单独走评审。
- **工具链硬门槛**：`go >= 1.27.3`（`自测:go-mod`）。本仓库现在 `go 1.24`，需要一次性升级；CI、开发者工具链、`go vet` 全部跟着动。
- **markstream 无 tag**：只能钉 commit，升级无版本语义；且它自身依赖 mygo 的伪版本，两个模块对 mygo 版本的要求会互相牵制。
- 仓库现行规则是「第三方依赖默认禁止」，例外只有 `internal/terminal` 的伪终端库与 `internal/ui/render` 的 markstream-go（`AGENTS.md` §3）。引入 mygo 本身属于框架依赖（`AGENTS.md` §1 已定为载体），但需要把 markstream / 图标 / 终端插件的例外写实。

### 8.6 外部内容与隔离

- `WebView` 控件能把网页嵌进 native UI（`mygo:docs/ui/web-view.md`），但仓库规则是「外部内容进独立隔离容器，不接入主界面」（`AGENTS.md` §3）。**默认不使用 `WebView`**。
- markstream 的 `rich` 包通过本机 Chromium 渲染图表/公式/HTML（`markstream:docs/api.md`「Rich engines and asynchronous graphics」）：这既是外部进程、又引入 MPL-2.0 的 D2，与上面的隔离约束方向一致地「不启用」。

### 8.7 许可风险

- 三种 MIT 组件都要带版权声明；`mygo` 的 MIT 声明随模块分发即可，但**打包成 app 时要主动带一份 THIRD_PARTY**（框架自身不代劳）。
- 图标里的 CC-BY 条目要逐条署名；品牌标识只作识别。
- 若启用 `rich`：MPL-2.0 的 D2 需按其条款分发（文件级 copyleft + 源码可得）。

---

## 9. 本次未验证 / 后续核对

这一节是「不许用替身结果」的落点：下面每一条都是本次**没有**做到的，写清怎么补。

| 未验证项 | 为什么 | 谁来补、怎么补 |
|---|---|---|
| 真实窗口的帧时间、滚动流畅度、缩放清晰度 | 本次只跑无窗口 `ui.Tester`（`自测:go-mod`），未开窗口 | U4/U5：用 `MYGO_FRAME_STATS=1` 在真实窗口上采样，写进 C4 的性能预算与 D 阶段的走查 |
| Windows / Linux 上的行为 | 只在本机 macOS arm64 核对 | 按 `docs/project-organization.md` §7.1 的规则：不在范围内的平台不记 blocked、不出证据；需要时由 D10/D12 另排 |
| markstream-go 集成后的实际表现 | 未把该模块加入本仓库，未构建 | C4 出设计选择后，在独立分支试接 `ms.View` + 消息列表，量一帧成本 |
| markstream `rich`（Mermaid/KaTeX/D2/HTML） | 需要本机 Chromium，且与 §8.6 冲突 | 暂不评估；需要时单独立项 |
| 图标模块与 mygo v0.8.3 的兼容 | `ailogo:mygo/go.mod` 声明 `mygo v0.7.4` | 接入时以 v0.8.3 实测 `go build`；若 `ui.Icon`/SVG 解析 API 有变，以其 README 的升级说明为准 |
| 中文输入法在自绘输入框里的行为 | 未写代码，未测 | U4：给 `HandleInput` 型控件补 `TextCaret` 的实测用例 |
| 屏读器（VoiceOver）实际朗读效果 | 未开真实窗口 | E 阶段（U6）手工走查，配合 `tt.AccessByRole` 的断言做双轨 |

---

## 附录 A　控件清单（mygo v0.8.3，共 114 个构造器）

- 布局与容器（13）：`Column` `Box` `Row` `Spacer` `Divider` `Grid` `Scroll` `ScrollHorizontal` `ScrollBoth` `Split` `SplitPercent` `SplitVertical` `SplitVerticalPercent`
- 文本与富文本（3）：`Text` `Textf` `RichText`
- 按钮与动作（8）：`Button` `PrimaryButton` `Link` `MenuButton` `PopoverMenuButton` `Toggle` `ToggleGroup` `Segmented`
- 导航与工具栏（8）：`Toolbar` `Tabs` `Sidebar` `SidebarItem` `SidebarSection` `Breadcrumbs` `BackButton` `ForwardButton`
- 选择与调节（12）：`Checkbox` `CheckboxGroup` `CheckboxState` `Switch` `RadioGroup` `Radio` `Select` `Slider` `StepSlider` `RangeSlider` `Stepper` `Rating`
- 文本输入（12）：`TextInput` `TextArea` `TextInputBuffer` `TextAreaBuffer` `NumberInput` `DecimalInput` `SearchField` `Combobox` `Autocomplete` `TokenField` `EditableText` `FindBar`
- 日期与颜色（5）：`Calendar` `DateInput` `TimeInput` `ColorPicker` `ColorWell`
- 集合（7）：`List` `Table` `Tree` `TreeItem` `GridView` `Outline` `OutlineTable`
- 表单与折叠（6）：`Form` `Field` `Fieldset` `Collapsible` `Accordion` `AccordionItem`
- 覆盖层与主题（5）：`Modal` `AlertDialog` `Popover` `Overlay` `Themed`
- 状态与媒体（9）：`Progress` `Spinner` `Meter` `Badge` `Image` `Video` `Icon` `Avatar` `WebView`
- 无外观底座（25）：`AccordionBase` `ButtonBase` `CheckboxBase` `CheckboxStateBase` `CollapsibleBase` `ComboboxBase` `ContextMenuBase` `DialogBase` `MenuBase` `PopoverBase` `RadioBase` `SegmentedBase` `SelectBase` `SliderBase` `SwitchBase` `TabsBase` `TextAreaBase` `TextAreaBufferBase` `TextInputBase` `TextInputBufferBase` `ToastBase` `ToastViewportBase` `ToggleBase` `TooltipBase` `TreeBase`
- 另有 1 个辅助构造器 `SelectBaseOpen`（不计入 114 的分类统计）

子包控件：`ui/markdown`（`View` `Stream` …）、`ui/editor`（`View` `DiffView` …）、`ui/richtext`（`Editor` …）、`ui/chart`（`Plot` `Line` `Area` `Pie` …）、`plugins/terminal`（`New` `View`）。

---

## 附录 B　本次实测记录

不写本机路径与会话标识，只记命令与观察到的输出。

| 编号 | 命令 | 观察 |
|---|---|---|
| `go-mod` | 在临时模块里写 `go 1.24` + `require github.com/egoist/mygo v0.8.3`，然后构建 | 失败：`go: github.com/egoist/mygo@v0.8.3 requires go >= 1.27.3 (running go 1.24.3)` |
| `go-mod` | 把该模块的 `go` 指令改为 `1.27.3`，用 `ui.NewTester` 渲染一个含 `ui.Text(c,"hi")` 的 view 并断言文本 | 通过，输出 `true`（无窗口渲染在本机可用） |
| `mygo-tags` | `git ls-remote --tags https://github.com/egoist/mygo` | 最新 tag 为 `v0.8.3` |
| `kero-tags` | `git ls-remote --tags https://github.com/egoist/kero` | 最新 tag 为 `v2.1.9` |
| `markstream-tags` | `git ls-remote --tags https://github.com/zerob13/markstream-go` | 无任何 tag（输出为空） |
| `markstream-deps` | 读 `markstream:go.mod` 并 `grep -rl chromedp --include='*.go'` | `chromedp` 只出现在 `rich/engine.go` 与其集成测试中，核心包不引用 |
| `markstream-size` | 统计根包行数与测试文件数 | 根包 5,622 行，11 个测试文件 |
| `ailogo-tags` | `git ls-remote --tags https://github.com/yldm-tech/ai-logo` | 存在 `mygo/v1.20.0`（子模块可单独钉版本） |
| `ailogo-zip` | 解压 `ailogo:mygo/icons.zip` 并数条目 | 2,709 个 SVG；`src/` 下 1,326 个品牌目录 |
| `widgets` | 在 mygo v0.8.3 的 `ui` 包里枚举 `func X(c *Context…)` 与泛型构造器并去重 | 114 个名字（25 个 `*Base` + 1 个 `SelectBaseOpen` + 88 个成型控件） |
| `kero-durations` | 在 kero 仓库统计 `[0-9]+ * time.Millisecond` 的出现次数 | 至少 50 / 100 / 120 / 150 / 200 / 260 / 400 ms 并存，说明框架没给时长刻度、各调用点各自取值 |
| `kero-usage` | 在 kero 仓库统计 `ui.NewTester`、`ui.*Base`、`ui.List`、`ui.Scroll` 的出现次数 | `NewTester` 91 处（19 个测试文件）；`ButtonBase` 30、`TextInputBase` 11、`DialogBase` 4；`List` 3、`Scroll` 6 |

**无法获取的来源：无。** mygo（本地模块缓存 + 公开仓库）、kero、markstream-go、AI Logo 四处都取到了原文；`ailogo.yldm.ai` 站点正文另行核对（HTTP 200）。上表列出的是「取到了但没实测」的项目，与「取不到」区分开。
