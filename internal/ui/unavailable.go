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
	// noteMCPWrite 用于 MCP 查看连接 / 断开连接。技术口径：protocol v3 没有连接
	// 查看与断开方法。
	noteMCPWrite = "当前版本不支持修改 MCP 连接"
	// noteMCPNoSave / noteMCPNoTest / noteMCPNoRemove 用于 hello 没有声明对应 MCP
	// 方法时的入口：置灰并写原因，不留点了只报错的按钮。
	noteMCPNoSave   = "当前服务端版本不支持保存 MCP 配置"
	noteMCPNoTest   = "当前服务端版本不支持测试 MCP 连接"
	noteMCPNoRemove = "当前服务端版本不支持删除 MCP 服务器"
	// noteMCPEditSave 用于编辑已有条目时的保存。技术口径：缺 = 保存前读回配置的
	// 合并通道（mcp.configuration），直接保存会把没回显的密钥一起清掉。
	noteMCPEditSave = "当前版本不支持在保留已存值的前提下修改 MCP 服务器"
	// noteMCPPartialSave 用于草稿里填了写入通道带不过去的字段（Bearer、请求头、
	// 工作目录、环境变量透传）时的保存。
	noteMCPPartialSave = "当前版本保存不了 Bearer 令牌、请求头与工作目录"
	// noteMCPMove 用于编辑器里的「移动作用域」。技术口径：缺 = mcp.move。
	noteMCPMove = "当前版本不支持移动 MCP 条目的作用域"
	// noteMessageDelete 用于删除消息与撤回。技术口径：协议没有消息删除 / 撤回方法。
	noteMessageDelete = "当前版本不支持删除与撤回消息"
	// noteArtifact 用于产物卡上的「打开」。技术口径：协议没有产物内容读取方法。
	noteArtifact = "当前版本不支持打开产物"
	// notePreviewIO 用于另存为、外部工具打开、在文件管理器中显示。技术口径：宿主
	// 没有文件写出与外部应用调用通道。
	notePreviewIO = "当前版本不支持另存与外部打开"
	// noteDirPicker 用于系统目录选择器的替代输入。技术口径：宿主没有目录选择器。
	noteDirPicker = "请手动输入路径"
	// noteSessionNavSample 用于样例态的会话前后导航：没有会话目录事实，不切会话。
	noteSessionNavSample = "没有会话列表可切换"
	// noteUnreadFact / noteStarFact 用于会话行的「标为未读」与「星标会话」：会话
	// 目录没有未读与星标字段，本端不拿本地状态冒充服务端事实。
	noteUnreadFact = "会话目录没有未读状态"
	noteStarFact   = "会话目录没有星标状态"
	// noteAgentNoWrite 用于状态条的 Agent 配置选择器：清单读到了，但会话没有声明
	// session.agent.update 或本端没有写通道。
	noteAgentNoWrite = "当前会话不能切换 Agent 配置"
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
	// noteBrowserNoImport 是 G-H6 导入 Chrome Cookie 的缺项原因。技术口径：扫描
	// Chrome 配置、按配置读计数、逐个导入并给回执都要宿主的浏览器数据通道，
	// 本版没有接（协议 v3 也没有浏览器数据方法）。
	noteBrowserNoImport = "当前版本不支持导入 Chrome Cookie"
	// noteBrowserNoClear 是 G-H7 清理浏览器数据的缺项原因。技术口径：清理与回执
	// 同一套浏览器数据通道，本版没有接。
	noteBrowserNoClear = "当前版本不支持清理浏览器数据"
	// notePersonalizationWrite 用于设置 › 个性化里只改屏内状态的项。技术口径：协议
	// 没有这项设置的写入方法。
	notePersonalizationWrite = "当前版本不支持修改这项设置"
	// NoteUpdateInstall 是「重启安装」置灰时的用户可见原因。技术口径：解包替换 .app
	// 需要宿主安装能力，本版没有接。导出是因为更新适配器与关于页共用同一句。
	NoteUpdateInstall = "当前版本不支持自动安装"
	// noteIsolationUnread 用于设置 › 权限的进程隔离开关：读不到当前值就不摆开关。
	noteIsolationUnread = "暂时读不到进程隔离的状态"
	// noteAppearanceLanguage 用于设置 › 外观 › 程序语言：只有一套界面文案。
	// 缺 = i18n 层（S-appearance-10 / 11 的切换与「本节文案同步变化」做不到），
	// 所以下拉保留形态、置灰，并写这句。
	noteAppearanceLanguage = "当前版本只有简体中文界面"
	// noteSystemAppearanceUnknown 用于设置 › 外观 › 跟随系统卡：宿主报不出系统外观
	// （无头运行 / 平台没有该通道）时按浅色渲染，并在卡下写这一句。
	noteSystemAppearanceUnknown = "暂时读不到系统外观，已按浅色显示"
	// noteThemeIconMissing 用于主题卡的预览图标：图标资源缺失时用等尺寸底色块占位。
	noteThemeIconMissing = "主题预览图标资源缺失，已用底色块占位"
	// noteThemeMarkMissing 用于主题卡的选中角标：对勾资源缺失时用纯色圆点兜底。
	noteThemeMarkMissing = "选中标记资源缺失，已用圆点占位"
	// noteFontListMissing 用于字体面板：系统报不出字体清单时写这句并给「重试」。
	noteFontListMissing = "无法读取系统字体列表。"
	// noteSectionIconMissing 用于设置导航里某个分节的图标资源缺失：图标位保留
	// 原宽度与命中区，只缺图形。
	noteSectionIconMissing = "这个分节的图标资源缺失"
	// noteProviderEnableWrite 用于模型供应详情头部的启用开关。技术口径：缺 =
	// provider.update 不带 enabled 字段（S-configuration-13 的写入与失败回滚做不了）。
	noteProviderEnableWrite = "当前版本不支持修改供应商的启用状态"
	// noteProviderAuthWrite 用于认证方式一行。技术口径：缺 = provider.update 不带
	// 认证类型字段（S-configuration-44 的分段切换）。
	noteProviderAuthWrite = "当前版本不支持修改认证方式"
	// noteProviderProtocolWrite 用于协议一行。技术口径：缺 = provider.update 不带
	// 协议字段（S-configuration-55 的协议单选）。
	noteProviderProtocolWrite = "当前版本不支持修改协议"
	// configCopyRawProtocol 是协议名认不出来时那一行的 tooltip（S-configuration-55
	// 的异常行）：详情与编辑表单共用这一句，两处各写一份就会走形。
	configCopyRawProtocol = "协议名未映射，显示原始标识"
	// noteProviderConnectWrite 用于「连接并获取模型」。技术口径：缺 = 新供应商的
	// 探测/拉清单方法（provider.probe 只对已存在的 provider 有效，S-configuration-49）。
	noteProviderConnectWrite = "当前版本不支持在保存前连接并拉取模型"
	// noteProviderAWSCreds 用于 AWS 认证的三件套。技术口径：缺 = provider.update
	// 不带 Access Key ID / Secret Access Key / Session Token（S-configuration-45）。
	noteProviderAWSCreds = "当前版本不支持填写 AWS 访问密钥"
	// noteProviderRegion 用于 Region / 基础 URL 那一行。技术口径：缺 = provider.update
	// 不带上游 Region（S-configuration-47）。
	noteProviderRegion = "当前版本不支持设置上游 Region"
	// noteProviderUsage 用于供应商详情里的用量 / 账户面板。技术口径：缺 = 按供应商
	// 读用量与账户的协议方法（S-configuration-51 / 52）。
	noteProviderUsage = "当前版本读不到按供应商的用量与账户"
	// noteProviderOrg 用于 OpenAI 的组织 / 项目字段。技术口径：缺 = provider.update
	// 不带 Organization ID / Project ID（S-configuration-53）。
	noteProviderOrg = "当前版本不支持填写组织与项目标识"
	// noteProviderTestUnavailable 用于「测试连接」在没选中读数项时置灰的原因。
	noteProviderTestUnavailable = "选中一个已保存的供应商后才能测试连接"
	// noteProviderProbeBusy 用于探测进行中的「刷新」。
	noteProviderProbeBusy = "正在探测，完成后可再刷新"
	// noteProviderModelListUnavailable 用于协议不支持从上游拉清单时的「刷新」。
	noteProviderModelListUnavailable = "当前版本不支持从上游拉取模型清单"
	// noteProviderPlans 用于「查看套餐」：套餐页地址由账户领域给，本屏不拼 URL。
	noteProviderPlans = "当前版本不能打开套餐页面"
	// noteProviderDisconnectWrite 用于「断开此提供商」确认框。技术口径：缺 =
	// 断开（清除凭据）的协议方法（S-configuration-62）。
	noteProviderDisconnectWrite = "当前版本不支持断开提供商"
	// noteProviderDeleteWrite 用于「删除此提供商」确认框。技术口径：缺 = 删除
	// provider 的协议方法（S-configuration-63）。
	noteProviderDeleteWrite = "当前版本不支持删除提供商"
	// noteProviderUserAgentWrite 用于编辑态的 User-Agent。技术口径：缺 =
	// provider.update 不带 User-Agent 字段（S-configuration-59）。
	noteProviderUserAgentWrite = "当前版本不支持单独设置 User-Agent"
	// noteProviderBuiltin 用于「内置供应商不提供删除」。技术口径：provider.list 的
	// 回执不带「内置 / 自定义」标记，界面分不出哪些是内置（S-configuration-63）。
	noteProviderBuiltin = "当前版本分不出内置与自定义供应商"
	// noteModelCreateWrite 用于「添加模型」对话框的保存。技术口径：缺 = 给已有
	// 供应商新增模型的方法（provider.update 的补丁里没有模型，S-configuration-77）。
	noteModelCreateWrite = "当前版本不支持给已保存的供应商新增模型"
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
	// noteTaskOutput 用于任务详情的只读输出框与「重新连接」。技术口径：会话快照
	// 与协议都没有任务输出，本端读不到。
	noteTaskOutput = "当前版本读不到这个任务的输出"
	// noteOffline 用于连接断开时仍需在屏上的入口（停止任务等）：请求发不出去，
	// 入口置灰并写这句，不留一个点了没反应的按钮。
	noteOffline = "连接已断开，暂时不能操作"
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
	// noteRetryInFlight 用于轮末重试在途时（发送中或还有轮在跑）：按钮禁用并写
	// 这句，避免连点重跑出两轮（R-XR-123 / R-XR-124）。
	noteRetryInFlight = "正在运行，结束后可重试"
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
	// noteNoOpenSession 是每个面板与选择器共用的空态标题：同一个事实不在一处写
	// 「未打开会话」、在另一处写「还没有打开会话」。
	noteNoOpenSession = "未打开会话"
	// noteNoSessionGoal 是没有打开会话时设定目标被挡住的原因：目标归会话。
	noteNoSessionGoal = "未打开会话，暂时设定不了目标"
	// noteNoWriteChannel 用于连着 Source 却没有写入通道的入口：既不装作生效，也
	// 不把原因留空。
	noteNoWriteChannel = "当前连接不支持写入"
	// notePermissionNoSession 用于权限档位选择器在没有打开会话时置灰的原因：
	// 档位写的是会话状态（session.set_permission_tier 要会话标识），空态里没有
	// 可写的对象——与筛选文件同一句法（「打开一个会话后才能…」）。
	notePermissionNoSession = "打开一个会话后才能切换权限"
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
