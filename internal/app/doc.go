// Package app 是集成层：把已落地的领域包装配成一个可运行的程序。
//
// 本包不实现任何领域逻辑，只做四件事：
//
//  1. 按架构文档的阶段顺序启动与关闭：装载本地状态 → 连服务（握手）→ 订阅（打开会话）
//     → 可用；退出时先收尾交互、再断连接、最后回收自启服务，每阶段带超时。
//  2. 把连接领域的一条控制连接与每会话一条订阅接到各领域的读取接口上
//     （目录、账号、设置、扩展、投影、交互）。
//  3. 实现写登记要求的两个注入点：Sender（按写意图类型选服务端方法，只发一次）
//     与 Inspector（用投影与目录的只读能力核对结果未知的写入）。
//  4. 对外只暴露只读视图与意图入口，供后续界面使用；本包不含界面。
//
// 装配关系（箭头表示"被装配进"）：
//
//	daemon.Supervisor ──┬─ 控制连接 ── catalogReader / accountReaders / settingsReaders
//	                    │            / extensionReaders / appSender / 通知路由
//	                    └─ OpenStream ── session.Manager ── 每会话一份订阅
//	                                          │
//	                                          └─ projection.Listen ── projection.Folder（会话视图）
//	                                                                  └─ interaction.Arbiter（审批与提问）
//	write.Manager ── appSender（Kind → 服务端方法）
//	              └─ appInspector（投影行 + 目录 + 历史只读扫描）
//	catalog.Catalog ── CatalogReader（session.catalog）+ sessionReleaser + 按轮历史来源
//	account.Account / settings.Manager / extensions.Service ── 各自读取方与写登记
//	attachments.Store / workspace.Workspace / terminal.Registry ── 本地能力，不连服务端
//
// 事实只来自服务端：本包不缓存会话内容，不生成正式标识，不做乐观更新。
package app
