package ui

// 这里是「当前版本不支持」的入口统一使用的用户可见原因。
//
// 依据 docs/domains/requirements/decisions.md 的十八条规定：需要协议方法或宿主
// 能力、而本版没有通道的动作，一律保留入口但置灰，并把原因写在 tooltip 或入口
// 旁。
//
// 上屏文案只写用户能读的结果（「当前版本不支持…」「暂时读不到…」），不写内部
// 口径：不出现协议方法名、宿主能力、快照 / 事实 / 读数这类开发术语，也不出现
// 「本版不做」。技术口径（缺哪个方法、D7 的边界）只写在下面的注释与决策记录里，
// 供代码引用。
//
// 新增未接入口时从这里取原因，不要在屏文件里另写一套句子。
// noteDisabledAlpha 是"可见但禁用"的整棵子树的透明度：0.5 与框架与宠物屏同一
// 口径（禁用态看得见形状，看得出不可点）。
const noteDisabledAlpha = 0.5

const (
	// noteGitWrite 用于暂存 / 取消暂存 / 还原 / 提交 / 推送 / 切分支。技术口径：
	// 协议没有 git 写方法，D7 不做 git 写。
	noteGitWrite = "当前版本不支持 git 写操作"
	// noteMCPWrite 用于 MCP 保存 / 删除 / 移动 / 查看连接 / 断开连接。技术口径：
	// protocol v3 没有 MCP 写入与连接查看方法。
	noteMCPWrite = "当前版本不支持修改 MCP 连接"
	// noteMessageDelete 用于删除消息与撤回。技术口径：协议没有消息删除 / 撤回方法。
	noteMessageDelete = "当前版本不支持删除与撤回消息"
	// noteArtifact 用于产物卡上的「打开」。技术口径：协议没有产物内容读取方法。
	noteArtifact = "当前版本不支持打开产物"
	// notePreviewIO 用于另存为、外部工具打开、在文件管理器中显示。技术口径：宿主
	// 没有文件写出与外部应用调用通道。
	notePreviewIO = "当前版本不支持另存与外部打开"
	// noteDirPicker 用于系统目录选择器的替代输入。技术口径：宿主没有目录选择器。
	noteDirPicker = "请手动输入路径"
	// notePluginWrite 用于插件设置的保存与重新加载。技术口径：协议没有插件设置
	// 写入方法。
	notePluginWrite = "当前版本不支持保存插件设置"
	// noteRuntime 用于运行环境的刷新、重试与下载安装。技术口径：协议没有运行环境
	// 读取与安装方法。
	noteRuntime = "当前版本不支持读取与安装运行环境"
	// noteRuntimeRefresh 用于运行环境页首的「刷新」：读数由连接层按需重读，
	// 界面没有手动刷新通道。
	noteRuntimeRefresh = "当前版本不支持手动刷新，数据会自动更新"
	// noteRuntimeFeature 用于功能组件卡。技术口径：协议不报告组件安装状态。
	noteRuntimeFeature = "当前版本读不到功能组件的安装状态"
	// noteModalitySubmit 用于模态任务的提交表单。技术口径：能力已声明，但协议没有
	// 提交通道。
	noteModalitySubmit = "当前版本还不支持提交模态任务"
	// noteModalityGroups 用于模态能力分组。技术口径：协议只声明模态是否可用，没有
	// 分组与能力清单。
	noteModalityGroups = "当前版本只能显示模态是否可用，没有分组清单"
	// noteGitSettings 用于设置 › Git。技术口径：仓库设置的读写都没有方法。
	noteGitSettings = "当前版本不支持读取或修改仓库设置"
	// notePetSettings 用于设置 › 宠物。技术口径：Pet 清单与选择都没有方法。
	notePetSettings = "当前版本不支持选择 Pet"
	// notePluginSettings 用于设置 › 插件设置。技术口径：插件 schema 与写入都没有方法。
	notePluginSettings = "当前版本不支持读取或修改插件设置"
	// noteDesktopSettings 用于设置 › 通用里由桌面自己保管的项（唤醒、性能日志、
	// 开机自启、数据根目录、代理、浏览器数据）。技术口径：协议没有这些设置的读写。
	noteDesktopSettings = "当前版本不支持修改这些桌面设置"
	// noteStorageUnread 是存储用量还没读到时写在读数位置的空态。
	noteStorageUnread = "暂时读不到存储用量"
	// notePersonalizationWrite 用于设置 › 个性化里只改屏内状态的项。技术口径：协议
	// 没有这项设置的写入方法。
	notePersonalizationWrite = "当前版本不支持修改这项设置"
	// NoteUpdateInstall 是「重启安装」置灰时的用户可见原因。技术口径：解包替换 .app
	// 需要宿主安装能力，本版没有接。导出是因为更新适配器与关于页共用同一句。
	NoteUpdateInstall = "当前版本不支持自动安装"
	// noteIsolationUnread 用于设置 › 权限的进程隔离开关：读不到当前值就不摆开关。
	noteIsolationUnread = "暂时读不到进程隔离的状态"
	// noteAppearanceLanguage 用于设置 › 外观 › 程序语言：只有一套界面文案。
	noteAppearanceLanguage = "当前版本只有简体中文界面"
	// noteSkinMissing 用于设置 › 外观 › 我的皮肤。技术口径：没有皮肤渲染器与皮肤
	// 素材。
	noteSkinMissing = "当前版本不支持更换皮肤"
	// noteFontMissing 用于某个字体族不在这台机器上。
	noteFontMissing = "本机没有安装这个字体"
	// noteFontNotMonospaced 用于代码字体选了非等宽字体。
	noteFontNotMonospaced = "这个字体不是等宽字体"
	// noteOpenWith 用于标题栏的「打开方式」。技术口径：外部应用清单与打开调用都没有
	// 通道。
	noteOpenWith = "当前版本不支持用外部应用打开"
	// noteApprovalCommand 是审批卡没有命令原文时的说明。
	noteApprovalCommand = "没有读到要执行的命令"
	// noteApprovalInput 是审批卡有工具名、却没有入参时的说明：没有内容区可画，
	// 但要写清读不到的是哪一样。
	noteApprovalInput = "无法读取该工具的入参"
	// noteApprovalBlocked 是同一张卡片上"为什么三个决定都不能点"的一句。
	noteApprovalBlocked = "读不到命令原文，无法判断要执行什么；先不提供决定。"
	// noteMemberApprovalDetail 用于成员审批区块的「重试」。技术口径：明细要另一条
	// 读取方法。
	noteMemberApprovalDetail = "当前版本不支持查看成员审批明细"
	// noteEmailLogin 用于企业邮箱登录的提交。技术口径：协议没有企业邮箱登录方法。
	noteEmailLogin = "当前版本不支持企业邮箱登录"
	// noteThinking 用于思考强度滑杆。技术口径：协议没有思考强度写入方法。
	noteThinking = "当前版本不支持调整思考强度"
	// noteSchedule 用于定时任务入口与唤醒页。技术口径：协议没有定时任务创建与唤醒
	// 事实。
	noteSchedule = "当前版本不支持定时任务"
	// noteTaskLog 用于进程详情的「打开日志」。技术口径：日志在服务端，本端没有打开
	// 文件的宿主通道，入口保留但置灰（K-27）。
	noteTaskLog = "当前版本不能打开日志文件"
	// noteDiffContent 用于行内 diff 读不到内容时的块内提示。技术口径：协议没有
	// diff 文本方法，D7 的 git 只读口径只到文件级（D-47-443）。
	noteDiffContent = "当前版本读不到这个文件的改动内容"
	// notePermissionWrite 用于权限设置的保存与刷新。技术口径：协议没有权限设置保存
	// 与刷新方法。
	notePermissionWrite = "当前版本不支持修改权限设置"
	// notePet 用于账户菜单的「显示 Pet」。技术口径：浮动 Pet 要宿主浮窗能力。
	notePet = "当前版本不支持浮动 Pet"
	// noteInvite 用于账户菜单的「邀请朋友」。技术口径：邀请链接要服务端给出。
	noteInvite = "当前版本不支持邀请链接"
	// noteMultiWindow 用于「在新窗口中打开」：这次运行没有窗口管理（界面测试 /
	// 单屏截图），不是「本版不支持」——真入口总接上窗口管理。
	noteMultiWindow = "当前运行方式没有多窗口"
	// noteNoSessionID 用于需要会话标识的复制 / 导出项。
	noteNoSessionID = "当前还没有打开的会话"
	// noteNoWorkspacePath 用于算不出路径时置灰的复制项：没有工作目录就拼不出
	// 绝对路径，也换不出相对路径；灰着却不写原因就成了没有说明的灰项。
	noteNoWorkspacePath = "没有工作目录，无法计算路径"
	// noteNothingToCopy 是复制项拿到空值时的提示。
	noteNothingToCopy = "没有可复制的内容"
	// noteRetryNow 用于时间线自动重试行的「立即重试」。技术口径：协议没有立即重试的
	// 方法，倒计时由服务端事实给，界面没有提前触发的通道。
	noteRetryNow = "当前版本不支持立即重试"
	// noteSideChatWrite 用于侧聊的「停止」「结束分支」以及输入区。技术口径：侧聊没有
	// 写入与结束通道；连上 Source 时输入框与发送一起置灰，原因写这句。
	noteSideChatWrite = "当前版本不支持侧边对话的发送与结束"
	// noteSideChatFacts 用于侧聊内容区的空态：连着 Source 时分支与消息都在服务端，
	// 本端没有读取它们的方法，所以内容区只能是空的。
	noteSideChatFacts = "当前版本读不到侧边对话的消息"
	// noteSubagentFacts 用于子代理面板的空态与「重试」。技术口径：协议没有子代理
	// 会话流。
	noteSubagentFacts = "当前版本不支持子代理会话"
	// noteSubagentNoSession 是还没有打开会话时的空态原因：这时缺的不是子代理
	// 通道，而是整个会话。
	noteSubagentNoSession = "打开一个会话后，这里会显示它的子代理输出"
	// noteNoWriteChannel 用于连着 Source 却没有写入通道的入口：既不装作生效，也
	// 不把原因留空。
	noteNoWriteChannel = "当前连接不支持写入"
	// notePetShown 用于斜杠命令 `/pet`：Pet 组件不可用，设置里的宠物显示开关
	// 在同一条件下也置灰（两处同源，不出现「命令说开了、设置说关」）。
	notePetShown = "Pet 组件在当前环境不可用（设置里的宠物显示开关同样置灰）"
	// notePluginReload 用于斜杠命令 `/reload-plugins`。技术口径：只有读清单的方法，
	// 没有重扫。
	notePluginReload = "当前版本不支持重新扫描插件"
	// noteFastMode 用于斜杠命令 `/fast`：本端没有快速模式的开关与状态。
	noteFastMode = "当前版本不支持快速模式"
	// noteWorkflow 用于斜杠命令 `/workflow`：没有工作流浏览入口。
	noteWorkflow = "当前版本不支持浏览工作流"
	// noteTodoFacts 用于活动条的「待办」。技术口径：会话快照没有待办清单。
	noteTodoFacts = "当前版本不支持待办清单"
	// noteSubagentActivity 用于活动条的「子代理」：与子代理面板同一句原因。
	noteSubagentActivity = noteSubagentFacts
	// noteZDR 用于模型浮层的零数据保留标记。技术口径：模型目录没有这个字段。
	noteZDR = "当前版本不显示零数据保留标记"
	// noteModelsRetry 用于模型浮层的「重试」：读数由连接层重读，没有手动刷新通道。
	noteModelsRetry = "当前版本不支持手动刷新模型清单，列表会自动更新"
	// noteSiteIcon 用于行内链接与网页工具行的站点图标：本版不取 favicon（没有网络
	// 取图）。
	noteSiteIcon = "当前版本不加载站点图标"
	// noteTerminalDetail 用于终端错误详情：没有调试详情时说明缺哪一样。
	noteTerminalDetail = "没有读到终端调试详情"
	// noteStopArm 是停止按钮第一次点击时的提示：再点一次才真的停。
	noteStopArm = "再点一次停止回复（Esc 取消）"
	// noteRestore 是时间线恢复遮罩上的说明：恢复期间不摆内容。
	noteRestore = "正在恢复历史位置"
	// noteSendUnknown 是发送能力还没确定时输入区给的原因。
	noteSendUnknown = "正在确认发送能力"
)
