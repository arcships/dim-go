# 负向取证：1,384 条「需负向取证」的用例证据

Status: 负向取证第一轮（脚本写回，可重跑；`python3 scripts/negative-evidence.py`）
Scope: 四份需求表里状态为「需负向取证」的全部条目（`docs/domains/requirements/requirements-*.md`）
Owner: 组织 agent（验收判定侧）；需求文件与元素清单仍归 R 与需求分析领域

## 0. 口径

这些条目的期望是「**某个现象没有发生**」：不出现、不残留、不重复、无响应、不发请求。
正向的文案与符号对账证明不了「没有发生」，所以它们此前单独一档、未计入未实现。

本轮把这一档做成可执行的证据：

| 步骤 | 做法 |
|---|---|
| 聚类 | 按**可观察的负向事实**聚类，而不是按元素逐条写：同一类里的条目共享同一条断言（「触发这个操作 → 屏上不出现那个东西」） |
| 用例 | 一条类一条 Go 用例（`internal/ui/negative_evidence_test.go`），全部从屏幕上的入口做起：点、键入、按键、右键、换读数 |
| 非空转 | 能对照的都给**正面对照**：同一句文案在触发态里必须上屏，只在另一态里不许出现。避免「这条文案代码里根本不画」造成的假通过 |
| 失败 | 用例失败就是实现与需求不一致，**不改断言去迁就实现**；条目状态写「未实现（负向用例失败：<现象>）」，交回实现方 |
| 写回 | 通过的写「已实现（证据 = <测试名>）」；没被任何用例覆盖的行**原样保留**，不替它们下结论 |

写回脚本只改登记过的行，并且只在该行状态仍是「需负向取证」或本脚本自己写的结论时才改；
别的对账轮次给出的结论不会被覆盖。

## 1. 结果

| 项 | 数 |
|---|---:|
| 「需负向取证」条目 | 1,384 |
| 写成用例的类 | 20（17 通过、3 失败；其中 18 条登记了条目，2 条是同类的前置防护） |
| 写回用例证据的条目 | 55（已实现 50、失败 5） |
| 仍是「需负向取证」（未覆盖） | 1,329 |

### 1.1 用例清单

| 用例 | 证明的负向事实 | 覆盖条目 | 结果 |
|---|---|---:|---|
| `TestNegativeApprovalStateHidesComposer` | 有待决审批时输入位只剩审批卡：无输入卡、无发送/停止键，答复前不发出任何写入 | 3 | 通过 |
| `TestNegativeApprovalWithoutLongTermHidesButton` | 审批快照没有长期规则时，长期允许按钮不出现（不是置灰） | 1 | **失败** |
| `TestNegativeRunningStateHidesSendButton` | 运行中发送槽换成停止，两个键不同时在场；该轮结束后停止键不残留 | 2 | 通过 |
| `TestNegativeRepeatedSubmitSendsOnce` | 提交还没回来时连点三次，只发出一条提交 | 2 | **失败** |
| `TestNegativeSessionSwitchLeavesNoStaleRows` | 切到另一会话后时间线与标题按新会话重建，不残留上一会话的行 | 2 | 通过 |
| `TestNegativeSidebarTextClicksAreInert` | 左栏分组标题只是文字：点「会话 / 排序 / 视图 / 已置顶 / 暂无会话」不折叠、不排序、不开菜单 | 5 | 通过 |
| `TestNegativeNoPinnedGroupWithoutPins` | 没有置顶时不渲染「已置顶」组；置顶后出现，取消后再次消失 | 1 | 通过 |
| `TestNegativeSettingsSectionsDoNotShowEachOther` | 逐节点过 19 节：只画当前节，别的节的独占文案一条都不在屏上 | 3 | 通过 |
| `TestNegativeScreensShowNoKeyNames` | 主界面、19 节设置、8 个走查态都不出现键名、`undefined`、`NaN`、`[object Object]` | 13 | 通过 |
| `TestNegativeAccountLeavesNoIdentityWithoutFacts` | 账户事实缺一项时那一行整行不渲染；退出登录后不残留旧邮箱，未登录卡与身份卡不同时在场 | 5 | 通过 |
| `TestNegativeUsageShowsNoFabricatedNumbers` | 用量屏只有「App Server 只读」一支：需连接/加载/失败卡与聚合数字都不出现 | 3 | 通过 |
| `TestNegativeEmptyThreadShowsNoHistory` | 空态里没有历史消息与会话行；状态条也不画（权限入口改由空态控制行承载） | 1 | 通过 |
| `TestNegativeSessionSearchForgetsQuery` | 关掉搜索浮层后可读名与命中一起消失；再打开不记忆上一次查询，空查询没有结果行 | 2 | 通过 |
| `TestNegativeSearchWithoutHitsHidesJumpButtons` | 零命中时不出现「上一个 / 下一个搜索结果」，结果计数点击无响应 | 2 | **失败** |
| `TestNegativeStatusBarWithoutUsageShowsNoPercent` | 没有用量快照时百分比位是 `—`，不编 0%、不出现 `NaN` | 2 | 通过 |
| `TestNegativeTasksWithoutSelectionShowNoDetail` | 后台进程页没有选中行时不画详情区（无「结束于 / 退出码 / 复制命令」） | 1 | 通过 |
| `TestNegativeReviewWithoutChangesShowsNoActions` | 没有改动时不画分组动作与底部统计条 | 3 | 通过 |
| `TestNegativeFilesWithoutEntriesShowNoRows` | 空目录不画任何树行，也没有「加载更多」；空态不借搜索空态或改动页的句子凑话 | 4 | 通过 |

另外两条用例暂时没有对应的「需负向取证」条目，作为同类的回归防护保留：

- `TestNegativeComposerHoldsFireWithoutDraftOrConnection`：空草稿或断线时点发送不发请求；
- `TestNegativeDisabledSettingsWriteNothing`：设置里置灰/未接的入口点下去不写任何事实。

## 2. 真缺陷（用例失败，未迁就）

| 需求 ID | 元素 | 现况 | 期望 |
|---|---|---|---|
| R-IN-136 | IN-12 发送键 | 请求还没回来时再点发送仍然发得出去：连点三次发出 3 条提交（没有在途守卫） | 进入进行中态，防止二次提交 |
| R-IN-140 | IN-12 发送键 | 同上：快速连点三次产生 3 条消息 | 只发送一次 |
| R-AP-214 | G-C1-20 长期允许 | 审批快照没有长期规则时，按钮仍渲染为**禁用态** | 按钮不出现（不显示为禁用） |
| R-TB-091 | TB-11 结果计数 | 零命中时「上一个 / 下一个搜索结果」仍渲染为禁用态 | 按钮不出现（无结果可跳） |
| R-TB-094 | TB-11 结果计数 | 结果计数本身可点，点击有响应（问题与上面同源：零命中态没把结果行收掉） | 点击无响应 |

前三条是独立的实现问题；R-TB-091 与 R-TB-094 是同一处（零命中时结果行没换形态）。

## 3. 未覆盖的 1,329 条

没覆盖的不是「认为它们满足」，而是**一条都还没证明**。按缺口分三类：

1. **逐元素条件夹具（约 1,000 条，主体）**：期望多为「该元素不渲染」，触发条件写在场景列（未选中、无搜索词、窄窗、失败态、只读……）。
   要证明它得先造出那个条件，再把该元素的具体文案/读屏名作为「不许出现」的判据。
   这类条目的判据只能逐元素从清单与场景列取（元素名多是描述性的，例如「不可用提示卡（App Server 只读）」），没法自动映射成屏上文案。
2. **像素与几何（约 150 条）**：颜色不随主题改、不出现半像素缝、2× 截图无彩边、行高不抖动一类，需要真机截图比对，进程内测不到。
3. **平台行为（约 180 条）**：输入法组字期间不发送、原生标题栏拖动不拦截、全屏进出不闪、系统时区切换不出现两个「今天」、系统目录选择器、读屏遍历顺序等，需要真实窗口与平台能力。

下一轮建议按元素族推进：每个族先建一个「条件 → 该元素文案」的夹具表（一次可覆盖同族几十条），再照本轮的写法补用例；
用例模板（触发、断言、正面对照）已经落在 `internal/ui/negative_evidence_test.go`，可直接复用。

## 4. 复核方式

```bash
go build ./...
go vet ./...
go test ./internal/ui -run TestNegative -count=1 -v
python3 scripts/negative-evidence.py --dry-run
```

已知的、与本轮无关的红：`internal/ui` 的 `go vet` 报 `mention.go` 自习赋值；`TestUILint`、`TestA11yScreens`、`TestScreens` 在动手做本轮时的未提交改动下报红（都不是本轮的用例）。
本轮的 3 条失败用例是第 2 节的缺陷，修好之前会一直红。
