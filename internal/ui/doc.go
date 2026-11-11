// Package ui 是 DimAgent 主界面的 native view 实现（mygo/ui）。
//
// 本批次只搭“首片界面骨架”：按 docs/domains/U1-ia.md 的六个区域与尺寸，
// 用静态样例把五个状态画出来，用来尽早看到风格；不连 App Server、不读事实、
// 不写本地状态、不引导任何写路径。界面之外的一切仍然是其他领域的职责。
//
// 纪律：
//   - 只有视图状态（哪个状态被切换、左栏是否收起、右栏是否展开、摘要是否展开、
//     输入的草稿）在这里；这些都是可丢的界面状态，不参与任何写入。
//   - 视觉值只取 tokens.go：它是 docs/domains/U2-visual.md §8 的 token 表，
//     颜色经主题（tokens.go 的 theme）与两个视图 token 取用，元素上不写字面色值。
//   - 视图每帧重建，Element 不跨帧保存：App 只保存文本、布尔与 ScrollState。
//
// 五个状态用启动参数 -state 或调试键 1–5 切换；Tab 在区域之间移动焦点。
package ui
