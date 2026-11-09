# 动画与过渡对照

Status: current
Last updated: 2026-11-09

## 规格（来自 dim-electron renderer CSS）

| 类型 | Electron | MyGo |
|---|---|---|
| 布局/面板 | 250ms `cubic-bezier(0.4, 0, 0.2, 1)`（`--layout-transition-ms`） | `motionLayout` 250ms 同曲线 |
| 按钮/颜色 | 150ms `cubic-bezier(0.23, 1, 0.32, 1)` | `motionFast` 150ms 同曲线 |
| 图片淡入 | 300ms `cubic-bezier(0.4, 0, 0.2, 1)` | 未实现（本次无图片层） |

曲线由 `ui.CubicBezier` 实现，测试断言端点、单调性与 reduce-motion（负 Duration）关闭。

## 应用范围

- 侧栏入口与会话行、顶栏按钮/VS Code 入口、composer 图标与发送：**颜色**，fast。
- 设置、keys、context panel：**位置/尺寸/颜色**，layout；用 Key 保持身份；context panel 宽度用 Collapse。
- 主题切换：主卡片、侧栏、thread list、header、composer card、分隔线，layout 颜色过渡。

## 帧证据

生成命令：

```bash
DIMAGENT_UI_MOTION_DIR="$PWD/docs/evidence/ui" \
  go test ./internal/ui -run 'TestMotionPanelFrames|TestMotionHoverFrames|TestMotionThemeFrames' -count=1
```

帧（0/75/150/250ms）：
- `docs/evidence/ui/motion-panel-open-*.png`
- `docs/evidence/ui/motion-panel-close-*.png`
- `docs/evidence/ui/motion-hover-*.png`
- `docs/evidence/ui/motion-theme-*.png`

测试断言中间帧与起点、终点都不同，证明存在过渡而不是瞬间重绘。

## 已知差异

1. mygo 的 `ElementTransition.Colors` 只过渡背景与边框，不过渡文字 ink：主题切换时文字颜色仍会跳变，中间帧对比度短暂偏低。
2. 面板关闭时副本不占布局，conversation 立即扩张，面板副本再收起；打开方向是平滑的。
3. 窗口 resize 本身没有过渡（Electron 也没有对每次 resize 播动画）。
4. 参考侧目前没有同操作的帧序列，只有静态截图；D6 要等两边都有帧才能判“一致”。
