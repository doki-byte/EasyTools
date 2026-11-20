# SkyShadowTools（运行期需自行准备）

白加黑（SkyShadow）模块在运行期需要下列微软工具链，本仓库**不包含**这些二进制文件：

- `dumpbin.exe`（PE 导入表解析）
- `link.exe` + `mspdbcore.dll`（导出表伪造/链接阶段）
- `sigcheck64.exe`（数字签名信息读取）
- `tbbmalloc.dll`（上述工具的运行期依赖）

`app/bypass/skyshadow/core.go` 中的 `//go:embed SkyShadowTools/*` 只用于保留该目录结构，
运行时按下面的方式检查工具是否存在。

## 使用方式

1. 安装 **Visual Studio Build Tools**（勾选 MSVC 与 Windows SDK），从安装目录取得
   `dumpbin.exe`、`link.exe`、`mspdbcore.dll`、`tbbmalloc.dll`；
2. 安装 **Sysinternals Suite**，取得 `sigcheck64.exe`；
3. 将上述文件放入运行目录下的 `skyshadowTools` 文件夹（Windows 下通常是
   `EasyToolsFiles\skyshadowTools`）；
4. 重新进入「免杀 → 白加黑」页面，缺失提示会消失。

缺少这些工具时，其余模块不受影响，仅白加黑扫描/生成链路会提示「缺少必要工具」。
