package ui

// 这里是「本版不做」的入口统一使用的用户可见原因。
//
// 依据 docs/domains/requirements/decisions.md 的十八条规定：需要协议方法或宿主
// 能力、而本版没有通道的动作，一律保留入口但置灰，并把原因写在 tooltip 或入口
// 旁。文案不出现内部术语（UI-3）：技术口径（缺哪个方法、D7 的边界）只写在
// 决策记录里，界面上只说「服务端没有提供」或「当前版本不支持」。
//
// 新增未接入口时从这里取原因，不要在屏文件里另写一套句子。
const (
	// noteGitWrite 用于暂存 / 取消暂存 / 还原 / 提交 / 推送 / 切分支。
	noteGitWrite = "服务端没有提供 git 写操作（本版不做）"
	// noteMCPWrite 用于 MCP 保存 / 删除 / 移动 / 查看连接 / 断开连接。
	noteMCPWrite = "服务端没有提供 MCP 写入与连接查看（本版不做）"
	// noteMessageFork 用于用户消息的「从此处分叉」。
	noteMessageFork = "服务端没有提供消息级分叉（本版不做）"
	// noteMessageDelete 用于删除消息与撤回。
	noteMessageDelete = "服务端没有提供消息删除与撤回（本版不做）"
	// noteArtifact 用于产物卡上的「打开」。
	noteArtifact = "服务端没有提供产物内容（本版不做）"
	// notePreviewIO 用于另存为、外部工具打开、在文件管理器中显示。
	notePreviewIO = "当前版本不支持另存与外部打开"
	// noteDirPicker 用于系统目录选择器的替代输入。
	noteDirPicker = "请手动输入路径"
	// notePluginWrite 用于插件设置的保存与重新加载。
	notePluginWrite = "服务端没有提供插件设置写入（本版不做）"
	// noteRuntime 用于运行环境的刷新、重试与下载安装。
	noteRuntime = "服务端没有提供运行环境读取与安装（本版不做）"
	// noteEmailLogin 用于企业邮箱登录的提交。
	noteEmailLogin = "服务端没有提供企业邮箱登录（本版不做）"
	// noteThinking 用于思考强度滑杆。
	noteThinking = "服务端没有提供思考强度写入（本版不做）"
	// noteSchedule 用于定时任务入口与唤醒页。
	noteSchedule = "服务端没有提供定时任务创建与唤醒事实（本版不做）"
	// notePermissionWrite 用于权限设置的保存与刷新。
	notePermissionWrite = "服务端没有提供权限设置保存与刷新（本版不做）"
	// noteAttachmentFacts 是用户消息里附件与引用区的占位说明。
	noteAttachmentFacts = "附件与引用还没有进快照（等事实）"
	// notePet 用于账户菜单的「显示 Pet」：浮动 Pet 要宿主浮窗能力。
	notePet = "当前版本不支持浮动 Pet"
	// noteInvite 用于账户菜单的「邀请朋友」：邀请链接要服务端给出。
	noteInvite = "当前版本不支持邀请链接"
)
