package ui

// 这里是「本版不做」的入口统一使用的用户可见原因。
//
// 依据 docs/domains/requirements/decisions.md 的十八条规定：需要协议方法或宿主
// 能力、而本版没有通道的动作，一律保留入口但置灰，并把原因写在 tooltip 或入口
// 旁。文案不出现内部术语（UI-3）：技术口径（缺哪个方法、D7 的边界）只写在
// 决策记录里，界面上只说「服务端没有提供」或「当前版本不支持」。
//
// 新增未接入口时从这里取原因，不要在屏文件里另写一套句子。
// noteDisabledAlpha 是"可见但禁用"的整棵子树的透明度：0.5 与框架与宠物屏同一
// 口径（禁用态看得见形状，看得出不可点）。
const noteDisabledAlpha = 0.5

const (
	// noteGitWrite 用于暂存 / 取消暂存 / 还原 / 提交 / 推送 / 切分支。
	noteGitWrite = "服务端没有提供 git 写操作（本版不做）"
	// noteMCPWrite 用于 MCP 保存 / 删除 / 移动 / 查看连接 / 断开连接。
	noteMCPWrite = "服务端没有提供 MCP 写入与连接查看（本版不做）"
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
	// noteRuntimeRefresh 用于运行环境页首的「刷新」：读数由连接层按需重读，
	// 界面没有手动刷新通道。
	noteRuntimeRefresh = "本版没有手动刷新通道（读数由连接层重读）"
	// noteRuntimeFeature 用于功能组件卡：服务端不报告组件安装状态。
	noteRuntimeFeature = "服务端没有提供功能组件的安装状态（本版不做）"
	// noteModalitySubmit 用于模态任务的提交表单：能力已声明，但本版没有提交通道。
	noteModalitySubmit = "本版没有提交任务的写入通道（本版不做）"
	// noteModalityGroups 用于模态能力分组：服务端只声明是否支持，没有分组清单。
	noteModalityGroups = "服务端只声明了模态是否可用，没有提供分组与能力清单（本版不列）"
	// noteGitSettings 用于设置 › Git：仓库设置的读写都没有通道。
	noteGitSettings = "服务端没有提供仓库设置的读写（本版不做）"
	// notePetSettings 用于设置 › 宠物：Pet 清单与选择都没有通道。
	notePetSettings = "服务端没有提供 Pet 清单与选择（本版不做）"
	// notePluginSettings 用于设置 › 插件设置：插件 schema 与写入都没有通道。
	notePluginSettings = "服务端没有提供插件设置的 schema 与写入（本版不做）"
	// noteDesktopSettings 用于设置 › 通用里由桌面自己保管的项（唤醒、性能日志、
	// 开机自启、数据根目录、代理、浏览器数据）。
	noteDesktopSettings = "服务端没有提供这些桌面设置的读写（本版不做）"
	// noteStorageUnread 是存储用量还没读到时写在读数位置的说明。
	noteStorageUnread = "服务端还没有给出存储用量（读数未到）"
	// notePersonalizationWrite 用于设置 › 个性化里只改屏内状态的项。
	notePersonalizationWrite = "服务端没有提供这项设置的读写（本版不做）"
	// NoteUpdateInstall 是「重启安装」置灰时的用户可见原因：解包替换 .app 需要
	// 宿主安装能力，本版没有接。导出是因为更新适配器与关于页共用同一句。
	NoteUpdateInstall = "当前版本不支持自动安装（需要宿主安装能力）"
	// noteIsolationUnread 用于设置 › 权限的进程隔离开关：读不到当前值就不摆开关。
	noteIsolationUnread = "服务端没有提供进程隔离设置的读取（本版不做）"
	// noteAppearanceLanguage 用于设置 › 外观 › 程序语言：只有一套界面文案。
	noteAppearanceLanguage = "本版只有简体中文界面，未接入其它语言包"
	// noteSkinMissing 用于设置 › 外观 › 我的皮肤：没有皮肤渲染器与皮肤素材。
	noteSkinMissing = "本版没有皮肤渲染器与皮肤素材，皮肤只在参考图里存在"
	// noteFontMissing 用于某个字体族不在这台机器上。
	noteFontMissing = "本机没有安装这个字体"
	// noteFontNotMonospaced 用于代码字体选了非等宽字体。
	noteFontNotMonospaced = "这个字体不是等宽字体"
	// noteOpenWith 用于标题栏的「打开方式」：外部应用清单与打开调用都没有通道。
	noteOpenWith = "当前版本不支持用外部应用打开"
	// noteApprovalCommand 是审批卡没有命令原文时的说明。
	noteApprovalCommand = "服务端没有给出要执行的命令"
	// noteApprovalBlocked 是同一张卡片上"为什么三个决定都不能点"的一句。
	noteApprovalBlocked = "读不到命令原文，无法判断要执行什么；先不提供决定。"
	// noteMemberApprovalDetail 用于成员审批区块的「重试」：明细要另一条读取通道。
	noteMemberApprovalDetail = "服务端没有提供成员审批明细的读取（本版不做）"
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
	// noteMultiWindow 用于「在新窗口中打开」：这次运行没有窗口管理（界面测试 /
	// 单屏截图），不是「本版不支持」——真入口总接上窗口管理。
	noteMultiWindow = "当前运行方式没有多窗口"
	// noteNoSessionID 用于需要会话标识的复制 / 导出项。
	noteNoSessionID = "当前还没有打开的会话"
	// noteNothingToCopy 是复制项拿到空值时的提示。
	noteNothingToCopy = "没有可复制的内容"
	// noteRetryNow 用于时间线自动重试行的「立即重试」：协议没有立即重试的方法，
	// 倒计时由服务端事实给，界面没有提前触发的通道。
	noteRetryNow = "服务端没有提供立即重试（本版不做）"
	// noteSideChatWrite 用于侧聊的「停止」与「结束分支」：侧聊没有写入与结束通道。
	noteSideChatWrite = "服务端没有提供侧聊的写入与结束分支（本版不做）"
	// noteNoWriteChannel 用于连着 Source 却没有写入通道的入口：既不装作生效，也
	// 不把原因留空。
	noteNoWriteChannel = "没有连接写通道（不可用）"
	// notePetShown 用于斜杠命令 `/pet`：Pet 组件不可用，设置里的宠物显示开关
	// 在同一条件下也置灰（两处同源，不出现「命令说开了、设置说关」）。
	notePetShown = "Pet 组件在当前环境不可用（设置里的宠物显示开关同样置灰）"
	// notePluginReload 用于斜杠命令 `/reload-plugins`：只有读清单的方法，没有重扫。
	notePluginReload = "服务端没有提供重扫插件（本版不做）"
	// noteFastMode 用于斜杠命令 `/fast`：本端没有快速模式的开关与状态。
	noteFastMode = "当前版本不支持快速模式"
	// noteWorkflow 用于斜杠命令 `/workflow`：没有工作流浏览入口。
	noteWorkflow = "服务端没有提供工作流浏览（本版不做）"
)
