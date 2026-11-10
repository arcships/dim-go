# DimAgent MyGo Native

MyGo native UI 重实现实验项目。

## 目标

- 使用 [egoist/mygo](https://github.com/egoist/mygo) 的 Go native UI 构建 DimAgent 主界面。
- 通过 protocol v3 连接 Rust `dim-app-server`。
- 不在本项目建立 session、message、run 或 approval 的第二份事实存储。
- WebView 只作为未来隔离的插件/外部内容容器，不作为主界面。

## 当前状态

项目刚初始化，当前阶段是组件和 native UI 技术验证。现有 Electron 项目作为产品行为、视觉设计、协议语义和验收基线。

## 目录

```text
cmd/dimagent/       应用入口
internal/app/       生命周期与 App Shell
internal/ui/        native view tree
internal/appserver/ App Server client（后续）
internal/projection/Timeline/session 投影（后续）
docs/               研究、设计与验收记录
```

## 开发前提

MyGo 当前主分支的 `go.mod` 声明 Go 1.27.3；本机 Go 版本低于该要求时，先升级 Go，再添加依赖和运行构建。

## 参考

- [Agent 组织与开发方案](docs/organization-model.md)
- [dim-electron 产品基线](https://github.com/arcships/dim-electron)
