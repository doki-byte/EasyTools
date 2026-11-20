package util

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// TestScrubKeepsExecutable 回归：全量特征清理不得破坏可执行段。
// 历史 bug：整文件盲擦把 .text 机器码当 ASCII 改写，导致免杀产物全部无法运行。
func TestScrubKeepsExecutable(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("仅 Windows")
	}
	if testing.Short() {
		t.Skip("short mode")
	}
	dir := filepath.Join(os.TempDir(), "EasyTools", "scrub_test")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(dir, "fixture.go")
	if err := os.WriteFile(src, []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "fixture.exe")
	cmd := exec.Command("go", "build", "-o", bin, src)
	cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0")
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("构建样本失败: %v\n%s", err, b)
	}

	// 先验证未清理时能运行
	base := exec.Command(bin)
	if err := base.Run(); err != nil {
		t.Fatalf("未清理样本运行失败: %v", err)
	}

	// 全量特征清理
	if _, err := ScanAndEvadeAVSignatures(bin); err != nil {
		t.Fatal(err)
	}

	// 清理后必须仍能正常启动并退出 0
	run := exec.Command(bin)
	done := make(chan error, 1)
	if err := run.Start(); err != nil {
		t.Fatalf("清理后启动失败: %v", err)
	}
	go func() { done <- run.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("清理后运行异常（可执行段被破坏?）: %v (exit=%d)", err, run.ProcessState.ExitCode())
		}
	case <-time.After(10 * time.Second):
		_ = run.Process.Kill()
		t.Fatal("清理后进程挂起")
	}
	_ = os.Remove(bin)
}
