# Workspace 功能规格（对象 / 流程 / 合同）

Status: draft（先定义语义，不改代码）
Scope: 只定义 Desktop 的 “workspace” 是什么、从哪来、能做什么、失败时怎样；不定义 session/timeline 内部。

## 1. 定义

Desktop 没有独立的 workspace 实体。**Workspace = 一组共享同一个绝对 `cwd` 的 session 的展示分组。身份就是 `cwd`。**

- 不存在 `workspaceId`、`workspace.create/rename/delete` 公开方法。
- 删除最后一个 session，该分组消失。
- “新建 workspace” 的准确语义是：**用一个新的 cwd 新建 session**，不是创建一个空目录对象。

依据：`session.catalog` / `session.create` 的 `cwd`；Desktop `sessions.list_workspace_page` 也是按 `cwd` 聚合 `SessionWorkspaceSummary`。没有 workspace CRUD 合同。

## 2. 事实与所有权

| 数据 | 所有者 | 说明 |
|---|---|---|
| session 的 `cwd` | App Server | 事实源；不可由 UI 改写 |
| 分组（cwd → sessions） | Desktop 派生 | 由 catalog 计算，不落盘 |
| builtin workspace cwd | Desktop 本地配置 | `app.projects.list_builtin`；仅默认值 |
| 分组别名 / 排序 / 折叠 / 置顶 | Desktop 本地展示状态 | 可丢；不得当 session 事实 |
| 最近使用 cwd | Desktop 本地展示状态 | 仅辅助选择，不证明目录存在 |

**不变量**

1. `cwd` 必须是绝对路径、非空；规范化后比较（尾斜杠/`..` 归一）。
2. 一个 session 只属于一个 cwd 分组。
3. 分组不能没有 session 而仅存在于内存（builtin 默认项除外）。
4. 别名/排序/折叠不改变 session 的 cwd，也不影响 App Server。
5. 切换或删除 workspace 不得改写其他 session 的 cwd。

## 3. 对象

```text
WorkspaceGroup
  cwd            string   // 身份，绝对路径
  displayName    string   // alias 或 basename(cwd)
  sessionCount   int
  latestUpdatedAt int64
  builtin        bool     // 来自 app.projects.list_builtin
  alias          string?  // 本地
  pinned         bool     // 本地
  expanded       bool     // 本地
```

本地存储（settings.json，可丢）：

```json
{
  "workspaces": {
    "aliases": {"<cwd>": "Display name"},
    "order": ["<cwd>", "..."],
    "pinned": ["<cwd>"],
    "expanded": {"<cwd>": true}
  }
}
```

## 4. 命令与流程

### 4.1 新建 workspace（= 新 cwd 新建 session）

入口：侧栏 “New session” 旁的目录选择，或 composer 的 Workspace 下拉 → “选择目录…”。

流程：

1. 用户选择目录（native picker，目录模式）。
2. 校验：绝对路径、存在、是目录、可读；重复 cwd 不阻止，直接复用该分组。
3. 调用 `session.create {cwd}`；不先创建本地 workspace 行。
4. 成功后：catalog 刷新；若分组已存在则追加 session，否则新分组出现；自动选中新 session。
5. 取消选择：不创建任何东西，不改变选中。
6. 失败（不存在/无权限/daemon 拒绝）：保留原选中，显示 `ActionError`；不写本地分组。

状态：空、选择中（native dialog）、创建中（按钮禁用）、成功、取消、失败。

### 4.2 切换 workspace

没有全局 “当前 workspace”。切换的是 session；composer 显示该 session 的 cwd。

1. 点击任一分组下的 session → 选中。
2. 顶部/composer 的 Workspace 显示该 session 的 cwd basename（tooltip 全路径）。
3. 运行中切换 session：沿用现有 per-session 状态，不迁移 turn/草稿。
4. cwd 缺失的 catalog 行：归入 “Unassigned”，不可新建/重命名。

### 4.3 重命名 / 排序 / 置顶（仅展示）

1. 别名只写本地 `workspaces.aliases[cwd]`；清空恢复 `basename(cwd)`。
2. 排序/置顶只改本地展示顺序，不改变 catalog 顺序或 session 的 `updatedAt`。
3. 别名与排序不触发任何 App Server 调用。

### 4.4 删除

没有 “删除 workspace” 命令。只有删除 session；最后一个 session 删除后分组消失，同时清理该 cwd 的 alias/order/pinned/expanded。

### 4.5 builtin workspace

- 启动时读取 `app.projects.list_builtin`，得到默认 cwd；只作为新建 session 的默认值。
- 不持久化该路径；下次启动重新解析。
- 解析失败：无默认 cwd，新建 session 时要求用户选择目录。

## 5. 协议映射

| 行为 | 公开方法 | 备注 |
|---|---|---|
| 列出分组 | `session.catalog`（读 session cwd） | Desktop 自行聚合；没有 workspace.list RPC 时不能伪造 |
| 新建 workspace | `session.create {cwd}` | 成功即事实 |
| 切换 | 选择 session | 无 RPC |
| 别名/排序/置顶 | 无 | Desktop 本地展示状态 |
| 删除 workspace | 无 | 只能删 session |
| builtin 默认目录 | `app.projects.list_builtin`（本地 IPC） | 不是 App Server 事实 |

如果上游未来提供 workspace CRUD，只替换数据源，不改 UI 语义。

## 6. 当前实现差距

- 侧栏 “Workspace” 是固定文案，不是 `cwd` 分组；项目分组用 `basename(cwd)`，没有全路径/别名/排序/置顶。
- `session.create` 固定使用进程 `cwd`，没有目录选择器。
- 没有本地 alias/order/pinned/expanded。
- 没有 builtin workspace 解析。
- 没有 “Unassigned” 分组与空分组规则。

## 7. 验收场景

1. **新建**：选择 `/tmp/ws-a` → `session.create {cwd:"/tmp/ws-a"}` → 新分组出现且新 session 选中。
2. **复用**：再次选择 `/tmp/ws-a` → 不新建分组，session 数 +1。
3. **取消**：打开 picker 后取消 → 无 session、无分组、选中不变。
4. **失败**：选择不存在目录 → 错误可见，无本地分组。
5. **切换**：点击 `/tmp/ws-a` 下 session → composer 显示 `ws-a`，tooltip 全路径。
6. **别名**：重命名分组为 “Alpha” → 只改本地显示；重启后保留；清空恢复 `ws-a`。
7. **删除**：删除该组最后一个 session → 分组、别名、排序、置顶一起消失。
8. **builtin**：无用户选择时，新 session 默认使用 builtin cwd；解析失败时要求选择。
9. **运行中切换**：turn 运行中切到另一组 session → 原 run 不受影响，草稿仍在原 session。

## 8. 非目标

- 不做 workspace 级权限、MCP、插件配置，除非上游提供 workspace 级合同。
- 不把分组/别名写成 session 事实。
- 不创建“空 workspace”对象；没有 session 就没有分组。
