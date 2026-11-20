# 第三方组件与许可声明

EasyTools 自身以 MIT 许可证发布（见 [LICENSE](LICENSE)）。项目内包含、内置或链接了以下第三方
组件，它们分别遵循各自的许可证。分发本项目或其衍生作品时，请一并保留相应声明。

## 一、随仓库源码分发的组件

| 组件 | 位置 | 许可证 |
|---|---|---|
| nuclei（含其 pkg 子树） | `app/lib/nuclei` | MIT |
| CyberChef 构建产物（内置离线版） | `app/embed/cyberchef` | Apache-2.0 |
| 小程序反编译运行时（node_modules 依赖树） | `app/embed/unwxapp/node_modules` | 各依赖自身许可证，见该目录下各包的 LICENSE |
| 目录扫描 / 口令 / 域名等字典 | `app/embed/dict` | 来自公开字典集合（dirsearch、SecLists 等），遵循其原始许可 |

nuclei 通过 `go.mod` 的 `replace github.com/projectdiscovery/nuclei/v3 => ./app/lib/nuclei` 引入，
如需升级请直接同步上游仓库。

## 二、构建期依赖（Go）

主要依赖及其许可证（完整列表见 `go.sum`）：

| 依赖 | 许可证 |
|---|---|
| github.com/wailsapp/wails/v2 | MIT |
| github.com/gin-gonic/gin | MIT |
| gorm.io/gorm、gorm.io/driver/sqlite | MIT |
| github.com/projectdiscovery/* | MIT |
| github.com/aliyun/alibaba-cloud-sdk-go | Apache-2.0 |
| github.com/aws/aws-sdk-go-v2/* | Apache-2.0 |
| github.com/baidubce/bce-sdk-go | Apache-2.0 |
| github.com/tencentcloud/tencentcloud-sdk-go | Apache-2.0 |
| github.com/huaweicloud/huaweicloud-sdk-go-v3 | Apache-2.0 |
| github.com/ucloud/ucloud-sdk-go | Apache-2.0 |
| github.com/energye/systray（fork: doki-byte/systray） | MIT |
| gitee.com/chunanyong/dm | Apache-2.0 |

## 三、构建期依赖（前端）

| 依赖 | 许可证 |
|---|---|
| vue、vue-router、pinia、vuex | MIT |
| element-plus、@element-plus/icons-vue | MIT |
| monaco-editor | MIT |
| xterm.js（@xterm/*） | MIT |
| axios、lodash、dayjs、marked、prismjs、nanoid、nprogress、tippy.js | MIT |
| xlsx（SheetJS 社区版） | Apache-2.0 |
| vite、sass | MIT |

## 四、未随仓库分发的工具（需自行准备）

「白加黑（SkyShadow）」模块在运行期需要以下工具，本项目**不再内置**，请自行从官方渠道获取：

- `dumpbin.exe`、`link.exe`、`mspdbcore.dll`、`tbbmalloc.dll`：Microsoft Visual Studio Build Tools
- `sigcheck64.exe`：Microsoft Sysinternals Suite

它们受 Microsoft 的许可约束，不得随本项目再分发。放置方式见
`app/bypass/skyshadow/SkyShadowTools/README.md`。

UPX 为可选构建/处理工具，遵循 UPX 自身许可证（GPL 例外条款允许压缩商业程序，但 UPX 本体为 GPL）。

## 五、免责

本清单尽力保持准确，但不构成法律意见。若你是某个组件的权利人并认为本项目使用方式不当，
请通过仓库 Issue 与我们联系，我们会尽快处理。
