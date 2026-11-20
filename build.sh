#!/usr/bin/env bash
# EasyTools 打包脚本（macOS / Linux）
#
# 用法：
#   ./build.sh mac arm64      # Apple Silicon
#   ./build.sh mac amd64      # Intel Mac
#   ./build.sh linux amd64
#
# 说明：Wails 依赖原生 WebView，macOS 产物需在 macOS 构建，Linux 产物需在 Linux 构建。
set -euo pipefail

TARGET_OS="${1:-linux}"
TARGET_ARCH="${2:-amd64}"

case "$TARGET_OS" in
  mac|darwin) PLATFORM="darwin/${TARGET_ARCH}"; OUTDIR="darwin_${TARGET_ARCH}" ;;
  linux)      PLATFORM="linux/${TARGET_ARCH}";  OUTDIR="linux_${TARGET_ARCH}"  ;;
  *)          echo "[error] 不支持的目标系统: $TARGET_OS（可选 mac / linux）"; exit 1 ;;
esac

command -v go    >/dev/null 2>&1 || { echo "[error] 未找到 go，请先安装 Go 1.24+"; exit 1; }
command -v npm   >/dev/null 2>&1 || { echo "[error] 未找到 npm，请先安装 Node.js 18+"; exit 1; }
command -v wails >/dev/null 2>&1 || { echo "[error] 未找到 wails CLI：go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.2"; exit 1; }

echo "[1/4] 安装前端依赖..."
if [ ! -d frontend/node_modules ]; then
  (cd frontend && npm install)
else
  echo "  frontend/node_modules 已存在，跳过"
fi

echo "[2/4] 构建 Wails 应用（${PLATFORM}）..."
wails build --clean --platform "$PLATFORM" --trimpath -ldflags="-w -s"

echo "[3/4] 整理产物..."
mkdir -p "build/bin/${OUTDIR}"
case "$TARGET_OS" in
  mac|darwin) mv build/bin/EasyTools.app "build/bin/${OUTDIR}/" ;;
  linux)      mv build/bin/EasyTools "build/bin/${OUTDIR}/" ;;
esac

echo "[4/4] 压缩（可选）..."
if command -v upx >/dev/null 2>&1 && [ "${EASYTOOLS_SKIP_UPX:-0}" != "1" ]; then
  case "$TARGET_OS" in
    linux)
      upx "build/bin/${OUTDIR}/EasyTools" || echo "[warn] UPX 压缩失败，保留原始产物"
      (cd "build/bin/${OUTDIR}" && tar -czf "EasyTools-linux-${TARGET_ARCH}.tar.gz" EasyTools)
      ;;
    mac|darwin)
      (cd "build/bin/${OUTDIR}" && ditto -c -k --sequesterRsrc --keepParent EasyTools.app "EasyTools-darwin-${TARGET_ARCH}.zip")
      ;;
  esac
else
  echo "  跳过 UPX（未安装或 EASYTOOLS_SKIP_UPX=1）"
fi

echo
echo "构建完成：build/bin/${OUTDIR}"
