# 贡献指南

感谢你愿意参与 EasyTools 的建设。提交代码前请花两分钟阅读以下约定。

## 开发环境

```bash
# Go 1.24+ / Node 18+
go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.2

cd frontend && npm install && cd ..
wails dev          # 开发模式，前端热更新
```

Windows 生成载荷相关功能还需要 MinGW-w64（`mingw64/bin`、`mingw32/bin` 在 PATH 中或放到常见
安装路径），UPX 为可选项。

## 分支与提交

- 从 `main`（或 `master`）切出功能分支：`feat/xxx`、`fix/xxx`、`docs/xxx`。
- 提交信息建议使用 Conventional Commits：`feat(bypass): 新增 xxx`、`fix(ssh): 修复 xxx`。
- 一个 PR 只做一件事，避免把格式化改动和功能改动混在一起。
- 涉及界面改动的 PR 请附截图或录屏。

## 代码约定

- Go：遵循 `gofmt`，导出符号必须有注释；新增 HTTP 接口要挂到对应模块的 `start()` 中。
- Vue：使用组合式/选项式皆可，但同一文件内保持风格一致；组件名与路由 `name` 保持一致。
- 不要在代码、日志、示例配置中提交任何真实凭据、Token、私有域名或个人信息。
- 新增第三方依赖需在 PR 描述中说明用途，并同步更新 [NOTICE.md](NOTICE.md)。

## 提交前自检

```bash
gofmt -l .                       # 应无输出
go build .                       # 主程序可编译
go test ./app/bypass/shell/util/ -count=1   # 载荷生成相关单测
cd frontend && npm run build     # 前端可构建
```

## 安全与合规

- 本项目仅用于合法的安全测试场景，不接受任何以「扩大未授权攻击能力」为目标的贡献。
- 发现安全问题请按 [SECURITY.md](SECURITY.md) 私下反馈，不要直接开公开 Issue。

## 行为准则

- 保持友善，就事论事；不接受人身攻击、歧视性言论或泄露他人隐私的内容。
- 维护者有权关闭不符合上述约定的 Issue / PR。
