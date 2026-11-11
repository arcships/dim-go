# C 成本核算（会话 token 用量与费用）

Status: 第一版完成（口径、脚本、本期实测数据均可复跑）
Owner: C 领域 agent（成本核算）
依据: `docs/project-organization.md` §2 角色约定；本机 DimCode 库的 `timeline_events` 用量事件；DeepSeek 官方价目页
范围: 本项目开发会话（组织会话 + 全部子会话）的 token 用量统计与费用换算。不含代码实现、界面、服务端计费对账。
自包含: 本文只使用库内 token 计数字段与官方公开单价；不引用消息正文、提示词、凭据，也不引用任何本地临时夹具路径。

---

## 0. 结果包

| 项 | 内容 |
|---|---|
| 领域 | C 成本核算 |
| 输入 | 本机 DimCode 库（`~/.dimcode-alpha/rust/dimcode.sqlite` 及其 `-wal`）中的 `timeline_events` 用量事件；DeepSeek 官方价目页；用户指定的汇率 6.9 |
| 产物 | `scripts/cost-report.py`（可重复运行的统计脚本）；本文（口径 + 本期实测 + 限制 + 复跑方法） |
| 验收 | 脚本可重复运行，输出与第 3.4 节一致（差异仅来自仍在进行的会话）；`go test ./...`、`go vet ./...` 不受影响（脚本只在 `scripts/`，不进 Go 构建） |
| 依赖 | 库结构保持 `timeline_events.kind='model/usage'` 与 `agent_sessions.leader_session_id`；官方价目页可访问（不可访问时用本文记录价目表） |
| 上下文需求 | 若需要按项目/时间段出账，由组织侧给出时间段边界与是否含非本项目会话 |

---

## 1. 统计口径

### 1.1 数据源与只读约束

| 项 | 做法 |
|---|---|
| 库位置 | `~/.dimcode-alpha/rust/dimcode.sqlite`；同目录 `-wal`、`-shm` 为未落盘部分 |
| 访问方式 | 先把 `db`、`-wal`、`-shm` 复制到系统临时目录，再用 `sqlite3` 只读 URI（`mode=ro`）查询 |
| 禁止 | 不写原库、不 `VACUUM`、不改 WAL、不在原库上跑写连接 |
| 收尾 | 脚本退出时删除临时副本（`finally` 中 `rmtree`） |
| 读取范围 | 只取 token 计数字段与事件时间戳；不读 `timeline_text`、`preview`、`config.systemPrompt`、`profile.systemPrompt` 等正文/提示词/凭据 |

### 1.2 表与字段

| 项 | 值 |
|---|---|
| 表 | `timeline_events` |
| 过滤 | `kind = 'model/usage'`（一次上游模型请求一条） |
| 用量路径 | `$.record.event.data.usage` |
| 时间戳 | `$.record.event.create_at`（毫秒 Unix 时间） |
| 字段 | `promptTokens`、`completionTokens`、`totalTokens`、`cacheReadTokens`、`cacheWriteTokens`（后者仅部分事件出现） |
| 模型 | `$.record.event.data` 下无模型字段；模型来自 `model/call` 事件的 `request.modelId`，本项目全部为 `deepseek-v4.1-flash` |

事件形状（只保留计数，正文不取）：

```json
{"record":{"event":{"create_at":1794366669030,"type":"model","name":"usage",
  "data":{"usage":{"promptTokens":738065,"completionTokens":283,
                   "totalTokens":738348,"cacheReadTokens":736640}}}}}
```

### 1.3 会话范围

- 先定位本项目**组织会话**：`agent_sessions.leader_session_id IS NULL` 且 `session_catalog.cwd` 等于项目根目录，取 `updated_at_ms` 最新的一条（本期为 `fb798162-20f7-4f00-b7f6-1be931f9d706`）。
- 用递归 CTE 按 `agent_sessions.leader_session_id` 展开该 leader 下的**全部子会话**（含子会话再派生的下级会话），得到会话清单，**不硬编码会话 id**。
- 角色名（领域名）不在会话表里：取自 `session_owners.agent_id → agents.profile` 的 `$.name`（如 `connector`、`integration-dev`、`cost-analyst`），leader 记为「组织」。
- 同一会话的事件可能落在多个 branch（本期 leader 有 11 条用量事件在非 `main` 分支），按 `session_id` 聚合，不按 branch 过滤；实测 `request_id` 无跨分支重复（本期用量事件数与不同 `request_id` 数相等，5329 = 5329；同一份库稍后再查为 5331 = 5331，关系不变），因此不需去重。

### 1.4 缓存读写如何区分

| 量 | 定义 | 计价口径 |
|---|---|---|
| cache-hit 输入 | `cacheReadTokens` | 命中价 |
| cache-miss 输入 | `promptTokens - cacheReadTokens`（下限 0） | 未命中价 |
| cache-write 输入 | `cacheWriteTokens`（本项目本期全为 0） | 已包含在 miss 内，不额外加价 |
| 输出 | `completionTokens` | 输出价 |
| 原始总量 | `totalTokens`（= `promptTokens + completionTokens`，实测相等） | 不做任何缓存折扣，仅作原始累计量 |

`promptTokens` 是**含命中**的输入总量；实测 `promptTokens >= cacheReadTokens` 恒成立（不满足的条数为 0），因此 miss 不会为负。

---

## 2. 本期数据

统计截至 **2026-11-11 11:44（北京时间，UTC+8）**，数据窗口 2026-11-10 13:21 → 2026-11-11 11:44。金额列见第 3 节口径。

### 2.1 复跑输出（脚本自动生成，勿手改）

- 生成时间：2026-11-11 11:44 CST
- 数据源：`~/.dimcode-alpha/rust/dimcode.sqlite` 的只读临时副本；表 `timeline_events`，`kind='model/usage'`
- leader 会话：`fb798162-20f7-4f00-b7f6-1be931f9d706`；会话数：18
- 单价（USD/1M tokens）：miss 0.15、hit 0.003、out 0.6；高峰 = 低谷 × 2.0
- 汇率：1 USD = 6.9 CNY（用户指定）

| 角色 | 领域 agent | 请求数 | prompt | completion | total | cache_read | cache_write | 费用（USD） | 费用（CNY） | 首末（北京时间） |
|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---|
| 组织 | default | 621 | 282,558,506 | 395,741 | 282,954,247 | 278,135,552 | 0 | 2.1033 | 14.5129 | 11-10 13:21 → 11-11 11:41 |
| 子会话 | requirements-analyst | 216 | 71,306,174 | 359,134 | 71,665,308 | 69,703,296 | 0 | 0.9798 | 6.7610 | 11-10 13:59 → 11-11 11:02 |
| 子会话 | architect | 101 | 12,987,544 | 167,745 | 13,155,289 | 12,516,992 | 0 | 0.2494 | 1.7206 | 11-10 15:44 → 11-10 22:24 |
| 子会话 | connector | 412 | 159,717,694 | 359,814 | 160,077,508 | 158,417,024 | 0 | 0.9032 | 6.2319 | 11-10 21:14 → 11-11 09:03 |
| 子会话 | workspace-dev | 111 | 13,296,547 | 113,414 | 13,409,961 | 13,034,368 | 0 | 0.1465 | 1.0107 | 11-10 21:15 → 11-11 01:54 |
| 子会话 | attachment-dev | 38 | 3,291,068 | 75,701 | 3,366,769 | 3,175,296 | 0 | 0.0723 | 0.4990 | 11-10 21:15 → 11-10 21:24 |
| 子会话 | projection-dev | 460 | 176,107,321 | 429,893 | 176,537,214 | 174,718,080 | 0 | 0.9905 | 6.8343 | 11-10 21:31 → 11-11 08:43 |
| 子会话 | acceptance-dev | 47 | 4,925,583 | 75,441 | 5,001,024 | 4,767,360 | 0 | 0.0833 | 0.5748 | 11-10 21:31 → 11-10 21:40 |
| 子会话 | write-dev | 212 | 49,903,139 | 195,197 | 50,098,336 | 49,525,760 | 0 | 0.3223 | 2.2239 | 11-10 21:47 → 11-11 01:43 |
| 子会话 | catalog-dev | 263 | 73,558,353 | 264,495 | 73,822,848 | 73,094,272 | 0 | 0.4476 | 3.0884 | 11-10 21:58 → 11-11 01:33 |
| 子会话 | interaction-dev | 305 | 105,561,796 | 299,802 | 105,861,598 | 105,002,112 | 0 | 0.5788 | 3.9940 | 11-10 21:58 → 11-11 01:55 |
| 子会话 | account-dev | 400 | 137,800,201 | 324,090 | 138,124,291 | 136,634,880 | 0 | 0.7792 | 5.3762 | 11-10 22:10 → 11-11 08:20 |
| 子会话 | extensions-dev | 458 | 201,185,885 | 401,593 | 201,587,478 | 199,684,608 | 0 | 1.0930 | 7.5415 | 11-10 22:21 → 11-11 09:07 |
| 子会话 | settings-dev | 424 | 156,460,881 | 391,316 | 156,852,197 | 155,770,112 | 0 | 0.8057 | 5.5594 | 11-10 22:21 → 11-11 01:37 |
| 子会话 | terminal-dev | 65 | 5,472,480 | 97,472 | 5,569,952 | 5,335,680 | 0 | 0.0950 | 0.6556 | 11-10 22:24 → 11-10 22:39 |
| 子会话 | host-dev | 192 | 48,960,679 | 284,948 | 49,245,627 | 48,283,904 | 0 | 0.4173 | 2.8796 | 11-10 22:39 → 11-11 08:17 |
| 子会话 | integration-dev | 973 | 458,384,079 | 641,543 | 459,025,622 | 456,066,560 | 0 | 2.4003 | 16.5621 | 11-10 22:39 → 11-11 11:05 |
| 子会话 | cost-analyst | 31 | 722,947 | 16,671 | 739,618 | 674,944 | 0 | 0.0385 | 0.2653 | 11-11 11:41 → 11-11 11:44 |
| **合计** | 18 个会话 | 5329 | 1,962,200,877 | 4,894,010 | 1,967,094,887 | 1,944,540,800 | 0 | 12.5060 | 86.2911 | 11-10 13:21 → 11-11 11:44 |

### 2.2 结构特征

| 观察 | 数值 |
|---|---|
| 请求数 | 5,329（平均每请求 prompt ≈ 368k、输出 ≈ 918 tokens） |
| cache 命中率 | 99.1%（1,944,540,800 / 1,962,200,877） |
| 组织会话占比 | 请求 11.7%、费用 16.8%（2.1033 / 12.5060） |
| 子会话里的两个大户 | `integration-dev` 2.4003 USD（19.2%）、`extensions-dev` 1.0930 USD（8.7%） |
| `cost-analyst` 自身 | 本领域会话仍在进行，复跑时数字会继续增大 |

---

## 3. 费用计算

### 3.1 单价（官方，USD / 每百万 token）

来源：DeepSeek 官方价目页 `https://api-docs.deepseek.com/quick_start/pricing`，抓取日期 **2026-11-11**（UTC 03:42）。对应模型 `deepseek-flash`（Model Version `DeepSeek-V4.1-Flash-1017`），本项目 `model/call` 事件的 `modelId` 全部为 `deepseek-v4.1-flash`。

| 计费项 | 低谷（off-peak） | 高峰（peak） |
|---|---:|---:|
| cache-hit 输入 | $0.003 | $0.006 |
| cache-miss 输入 | $0.15 | $0.30 |
| 输出 | $0.60 | $1.20 |

- 官方规则：低谷价 = 高峰价的一半。
- 高峰时段：UTC **周一至周五 01:00–04:00 与 06:00–10:00**，中国法定节假日除外；其余时段一律低谷（含周末与节假日全天）。
- 计费按事件时刻判定峰谷，逐条累加，不是按总量套一档。

### 3.2 汇率

- 汇率：**1 USD = 6.9 CNY**，**2026-11-11**，**由用户指定**（非市场即时汇率，也不是本领域自选）。
- 换算公式：`CNY = USD × 汇率`；单条计费项同理 `CNY_i = tokens_i / 1e6 × price_i × 峰谷倍数 × 汇率`。
- 换汇率只需复跑：`--rate <新汇率>`，其余保持不变即可得到同一用量下的新人民币金额。

### 3.3 计算公式

```
miss_tokens = promptTokens − cacheReadTokens
hit_tokens  = cacheReadTokens
out_tokens  = completionTokens

USD_miss = Σ miss_tokens × p_miss × peak_mult(t) / 1e6
USD_hit  = Σ hit_tokens  × p_hit  × peak_mult(t) / 1e6
USD_out  = Σ out_tokens  × p_out  × peak_mult(t) / 1e6
USD      = USD_miss + USD_hit + USD_out
CNY      = USD × 6.9
```

其中 `peak_mult(t)` 在高峰时段取 2、其余取 1；`p_*` 为 3.1 表的低谷单价。

### 3.4 本期结果

| 计费项 | tokens | 单价（USD/1M） | 金额（USD） | 金额（CNY） |
|---|---:|---:|---:|---:|
| cache-miss 输入 | 17,660,077 | 0.15 | 2.8432 | 19.6181 |
| cache-hit 输入 | 1,944,540,800 | 0.003 | 6.4240 | 44.3258 |
| 输出 | 4,894,010 | 0.6 | 3.2387 | 22.3472 |
| **合计** | — | — | **12.5060** | **86.2911** |

费用结构：cache-hit 输入 51.4%、输出 25.9%、cache-miss 输入 22.7%。命中率高，但命中量级极大（19.4 亿 tokens），命中项仍是最大成本。

### 3.5 峰谷价差与「不吃缓存」对照

| 对照口径 | 金额（USD） | 金额（CNY） | 相对实际 |
|---|---:|---:|---:|
| 本次实际（按事件时刻分峰谷、计缓存折扣） | 12.5060 | 86.2911 | +0.0% |
| 若全部按低谷价（缓存仍计价） | 11.4190 | 78.7914 | -8.7% |
| 若全部按高峰价（缓存仍计价） | 22.8381 | 157.5828 | +82.6% |
| 不吃缓存：全部输入按 miss 价（分峰谷） | 324.0446 | 2,235.9079 | +2491.1% |

- 峰谷价差：高峰单价是低谷的 2 倍。本期仅约 10% 的 token 落在高峰（miss 7.3%、hit 10.1%、out 10.3%），故实际比「全低谷」高 8.7%，比「全高峰」低 45.2%。
- 缓存折扣：命中价是未命中价的 1/50。若不命中缓存，同样用量要 324.04 USD（¥2,235.91），是实际的 25.9 倍——缓存在本项目里是最大的成本杠杆。
- 输出单价是 miss 输入价的 4 倍、hit 输入价的 200 倍；输出只占 token 总量 0.25%，却贡献 25.9% 的费用。

---

## 4. 限制

1. **临时 home 跑测试的用量不在本库**：用临时 `HOME`/临时数据目录启动的测试实例写的是另一份库，本报告只覆盖 `~/.dimcode-alpha/rust/dimcode.sqlite` 这一份，因此真实总量可能被低估。
2. **`totalTokens` 是原始累计量**：不按缓存折扣折算，不能直接乘单价当作费用；计费必须用 `promptTokens`、`cacheReadTokens`、`completionTokens` 三项分开算（本文 3.3）。
3. **`completionTokens` 不含思考 token（推断）**：库内 `usage` 只有 `promptTokens/completionTokens/totalTokens/cacheReadTokens/cacheWriteTokens` 五个字段，没有单独的思考或推理 token 字段。本项目平均每请求输出仅约 918 tokens，而同时期有 4,191 条 `step/thinking` 事件（约每 1.3 个请求一条），输出量级明显偏低，因此判断思考 token 未计入 `completionTokens`。这是从数据形状推断，**不是服务端计费口径的确认**；若上游把思考 token 另计，实际费用会高于本报告。
4. **缓存写入未单独定价**：`cacheWriteTokens` 本期全为 0；其他项目的事件里有非 0 值。本文把它并入 miss 输入（不加价），若上游对写入有加价系数，费用会略增。
5. **峰谷判定不含中国法定节假日**：脚本按 UTC 星期与时段判定，可用 `--holidays` 手工排除；默认不排除时段内节假日，故高峰占比可能被高估（费用偏保守，最多 +9%）。
6. **进行中的会话会继续增长**：`cost-analyst` 等会话在统计时刻仍在跑，复跑数字会略大于第 2 节；比对复跑结果时应以同一时刻为准。
7. **只统计本 leader 会话树**：同机其他项目、其他 leader 的会话不计入；本报告不含任何 UI 或服务端侧的用量核对。
8. **单价随官方调整**：价目页会变，本文记录的是 2026-11-11 抓取值；换价只需改 `--price-*` 或本文 3.1 表并注明抓取日期。

---

## 5. 复跑方法

### 5.1 脚本

`scripts/cost-report.py`（Python 3 标准库，无第三方依赖，不进 Go 构建）。

```bash
# 默认：按当前 git 仓库根目录自动发现本项目 leader 会话，汇率 6.9，官方低谷单价
python3 scripts/cost-report.py

# 指定 leader（其他项目或历史会话）
python3 scripts/cost-report.py --leader fb798162-20f7-4f00-b7f6-1be931f9d706

# 换汇率 / 换单价
python3 scripts/cost-report.py --rate 7.1
python3 scripts/cost-report.py --price-miss 0.15 --price-hit 0.003 --price-out 0.6

# 不分峰谷，全部按低谷价
python3 scripts/cost-report.py --flat

# 写文件而不是标准输出
python3 scripts/cost-report.py --out cost.md
```

| 参数 | 默认 | 说明 |
|---|---|---|
| `--db` | `~/.dimcode-alpha/rust/dimcode.sqlite` | 库路径；脚本自行复制到临时目录 |
| `--leader` | 自动 | leader 会话 id；缺省按 `--project` 找最近活跃的主会话 |
| `--project` | 当前 git 仓库根目录 | 自动发现主会话用的 `session_catalog.cwd` |
| `--rate` | `6.9` | 1 USD = N CNY |
| `--price-miss` / `--price-hit` / `--price-out` | `0.15` / `0.003` / `0.6` | 低谷单价（USD/1M） |
| `--peak-mult` | `2.0` | 高峰 = 低谷 × N |
| `--flat` | 关 | 全部按低谷计价 |
| `--holidays` | 空 | 逗号分隔的 UTC 日期，按低谷处理 |
| `--rate-note` | 用户指定 | 输出里的汇率来源说明 |
| `--out` | 标准输出 | 输出文件路径 |

### 5.2 预期输出

脚本输出 Markdown 三段：会话表（角色、领域 agent、请求数、prompt、completion、total、cache_read、cache_write、费用、首末时间）、计费项表（miss/hit/out 与合计，USD+CNY）、对照表（全低谷/全高峰/不吃缓存）加命中率与峰谷 token 分布。

以本文实测为基准，重复运行时应当：

- 会话数、leader id、单价、汇率行完全一致；
- 已结束会话（如 `attachment-dev`）各行数字**逐位一致**；
- 仍在进行的会话（`requirements-analyst`、`cost-analyst`、`integration-dev` 等）数字**只增不减**，合计随之上浮；
- 费用行应满足 `USD_miss + USD_hit + USD_out = USD 合计`（本文 2.8432 + 6.4240 + 3.2387 = 12.5059，与 12.5060 的差为四位小数舍入）。

---

## 6. 结论

1. 本期（2026-11-10 13:21 → 2026-11-11 11:44）本项目 18 个会话、5,329 次模型请求，用量 19.62 亿输入 tokens（其中 19.45 亿为 cache 命中）、489 万输出 tokens。
2. 费用 **12.5060 USD ≈ 86.2911 CNY**（汇率 6.9，2026-11-11，用户指定）；组织会话 2.1033 USD（16.8%），子会话 10.4027 USD（83.2%）。
3. 成本主要由 cache-hit 输入（51.4%）与输出（25.9%）构成；未命中输入只占 22.7%。要降本，优先压输入上下文（长上下文在低命中价下仍然累积），其次减输出。
4. 峰谷只影响约 9% 的费用；缓存命中把输入成本压到 1/50，是决定性因素。
