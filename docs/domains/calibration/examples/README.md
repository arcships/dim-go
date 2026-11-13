# 官方示例的渲染（mygo v0.8.3）

这里的图是宿主框架自带示例的渲染，**作为组合层对照的左侧切片**（UI-24）。它们不是
我们的界面，也不是"设计稿"：功能型示例、风格朴素，价值在于**组合手法**——标题
层级、分组方式、面层、留白节奏、输入区体量。

| 文件 | 尺寸 | 示例 | 示范了哪些组合手法（对照时指的区域） |
|---|---|---|---|
| `gallery-overview.png` | 980×720 | `examples/gallery`（Overview 页） | ① 页面标题 20–24/700 + 一句辅助说明（13，muted）；② 左导航：图标 + 文字、选中行 `surface-pressed` 圆角条、行距 2；③ 内容**卡片**：白底 + 1px `border` + 圆角 8 + 内边距 16–20；④ 卡片内小标题 15/600 + 正文 13 muted；⑤ 卡片间距 20 |
| `gallery-controls.png` | 980×720 | `examples/gallery`（Controls 页） | 控件成组摆放时的行距（12）与标签—控件间距（12）；同一卡片内多控件对齐 |
| `gallery-list.png` | 980×720 | `examples/gallery`（List 页） | 长列表行高 28–32、行内左图标槽 16 + 间距 8、选中态靠底色 |
| `notes.png` | 900×640 | `examples/notes` | ① 三级标题：列表标题 15/600、文档标题 24/700、正文 14；② 工具栏与编辑区分开（一行工具 → 一张编辑卡）；③ 编辑卡：白底 + `border` + 圆角 8 + 内边距 20；④ 底部状态行 12 muted；⑤ 侧栏行高 28 |
| `permissionflow.png` | 520×640 | `examples/permissionflow` | ① 页面标题 20/700 + 两行说明（13，muted，最大宽度约束）；② **每条权限是一张卡片行**：白底 + `border` + 圆角 8 + 内边距 12–16；③ 行内左右分工：左标题 14/600 + 状态词 13（彩色），右按钮组（次 + 主，间距 8）；④ 卡片间距 12 |
| `markdown-chat.png` | 820×720 | `examples/markdown-chat` | ① Markdown 的 H2 18/700、表格（表头 `surface` 底 + 1px `border`）、代码块（`surface` 底 + 圆角 6 + 内边距 12）、引用块（左竖线 + muted）、链接 accent；② 消息气泡右对齐、圆角 14、内边距 8/12；③ 输入区一行：输入框占满 + 右侧主按钮同高；④ 正文左右留白 24、段落间距 12–16 |
| `markdown-chat-dark.png` | 820×720 | 同上（深色） | 面层在深色下的层级：`background` < `surface` < `surface-hover`，边框仍是 1px `border` |
| `probe-composer-before.png` / `probe-composer-v3.png` | 900×220 | 输入区探针 | 输入卡从"裸输入框"到"带体量的卡"的两次改法（仅作参考，不是我们的最终形态） |

**用这张表的方式**：对照某条组合手法时，直接写「文件名 + 区域坐标」，例如
`permissionflow.png (8,80)–(512,280)`。不要写"官方风格是……"这种印象话。
