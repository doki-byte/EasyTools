package util

import (
	"EasyTools/app/bypass/shell/encode"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// runtimeSkipLoaders 不参与通用运行测试的模板编号：
//   - loader10：白加黑专用模板（cgo 导出函数，需要完整白加黑编译链路，不走通用测试）
//   - loader50 / loader51：面向 Linux/macOS 生成目标的模板
//   - loader14 / loader24 / loader25 / loader26 / loader28：运行行为特殊（回调需外部窗口/资源上下文），沿用旧版排除清单
var runtimeSkipLoaders = map[string]bool{
	"loader10": true,
	"loader14": true,
	"loader24": true,
	"loader25": true,
	"loader26": true,
	"loader28": true,
	"loader50": true,
	"loader51": true,
}

// ensureGoModule 确保临时工程存在 go.mod/go.sum（离线缓存优先，失败再走代理）
func ensureGoModule(t *testing.T) {
	t.Helper()
	if err := InitGoModule(); err != nil {
		t.Fatalf("初始化Go模块失败: %v", err)
	}
}

// TestGenLoaderRuntime 端到端验证：加密 payload → 生成 loader → 编译 → 运行。
// 使用 x64 死循环 shellcode（EB FE），loader 正常时进程持续存活。
func TestGenLoaderRuntime(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}

	// 1. 准备 payload（x64 无限循环）
	payload := []byte{0xEB, 0xFE}
	dir := filepath.Join(os.TempDir(), "EasyTools", "file")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "payload.bin"), payload, 0644)

	// 2. 加密（模拟 makeFile 一体模式）
	aesKey := encode.GenerateAesKey()
	aesEnc, _ := encode.Aes_Encrypt(payload, aesKey)
	xorKey, err := encode.GenerateXorKey()
	if err != nil {
		t.Fatal(err)
	}
	xorEnc := encode.XorEncrypt(append([]byte(nil), payload...), xorKey)
	chachaKey, _ := encode.GenerateChaCha20Key()
	chachaNonce, _ := encode.GenerateChaCha20Nonce()
	chachaEnc, _ := encode.ChaCha20Encrypt(chachaKey, chachaNonce, payload)

	os.WriteFile(filepath.Join(dir, "AesEncrypted.txt"), aesEnc, 0644)
	os.WriteFile(filepath.Join(dir, "AesKey.txt"), aesKey, 0644)
	os.WriteFile(filepath.Join(dir, "XorEncrypted.txt"), xorEnc, 0644)
	os.WriteFile(filepath.Join(dir, "XorKey.txt"), encode.XorKeyToBytes(xorKey), 0644)
	os.WriteFile(filepath.Join(dir, "ChaCha20Encrypted.txt"), chachaEnc, 0644)
	os.WriteFile(filepath.Join(dir, "ChaCha20Key.txt"), chachaKey, 0644)
	os.WriteFile(filepath.Join(dir, "ChaCha20Nonce.txt"), chachaNonce, 0644)
	os.WriteFile(filepath.Join(dir, "UrlOrFileEncrypted.txt"), []byte("deadbeef"), 0644)
	os.WriteFile(filepath.Join(dir, "UrlXorKey.txt"), []byte{0x42}, 0644)

	ensureGoModule(t)

	// 3. 生成 windows loader（一体/无沙箱），并编译运行
	allNames := allModuleNames()
	var winLoaders []string
	for _, name := range allNames {
		if !runtimeSkipLoaders[name] && !unixLoaderNames[name] {
			winLoaders = append(winLoaders, name)
		}
	}
	t.Logf("运行测试模板数: %d (loader1..loader%d 中排除特殊模板后)", len(winLoaders), len(allNames))

	for _, loaderName := range winLoaders {
		out, gerr := GenGoFile(loaderName, "一体", "xor", "否", "false", "false", "", "false", "", "", nil, nil)
		if gerr != nil || out == "" {
			t.Fatalf("生成失败 %s: %v", loaderName, gerr)
		}
		bin := filepath.Join(dir, "runtime_"+loaderName+".exe")
		cmd := exec.Command("go", "build", "-o", bin, out)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0")
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("%s 编译失败: %v\n%s", loaderName, err, b)
			continue
		}

		// 运行 1.5 秒：死循环 shellcode 应保持进程存活
		run := exec.Command(bin)
		if err := run.Start(); err != nil {
			t.Errorf("%s 启动失败: %v", loaderName, err)
			continue
		}
		time.Sleep(1500 * time.Millisecond)
		exited := run.ProcessState != nil && run.ProcessState.Exited()
		if exited {
			t.Errorf("%s 进程提前退出（loader 未正常运行，exit=%v）", loaderName, run.ProcessState.ExitCode())
		} else {
			t.Logf("%s 运行正常 ✓", loaderName)
			_ = run.Process.Kill()
		}
	}
}

// TestAllWindowsModulesCompile 编译门禁：全部 Windows 模板逐一 go build（CGO=0），
// 用于拦截模板级语法/结构回归（如内存加固改动）。不执行产物。
// 跳过：loader10 白加黑（依赖 cgo 导出，单独链路）、loader50/51（面向 Linux/macOS 的模板）。
func TestAllWindowsModulesCompile(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	ensureGoModule(t)
	dir := filepath.Join(os.TempDir(), "EasyTools", "file")
	for _, name := range allModuleNames() {
		if name == LoaderBaiJiaHei || unixLoaderNames[name] {
			continue
		}
		out, gerr := GenGoFile(name, "一体", "xor", "否", "false", "false", "", "false", "", "", nil, nil)
		if gerr != nil || out == "" {
			t.Fatalf("生成失败 %s: %v", name, gerr)
		}
		bin := filepath.Join(dir, "c_"+name+".exe")
		cmd := exec.Command("go", "build", "-o", bin, out)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0")
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("%s 编译失败: %v\n%s", name, err, strings.TrimSpace(string(b)))
			continue
		}
		_ = os.Remove(bin)
	}
}

// TestUnixLoaderCrossCompile 验证 Unix 模板可在本机交叉编译为 Linux/macOS 目标（免杀模块跨平台生成兼容性）。
func TestUnixLoaderCrossCompile(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}

	// 当前模板集可能不含 Unix 模板：此时该能力不可用，跳过而不是误报失败
	unixLoader, ok := DefaultUnixLoader()
	if !ok {
		t.Skip("当前模板集不含 Linux/macOS 模板，跳过交叉编译验证")
	}

	payload := []byte{0xEB, 0xFE}
	dir := filepath.Join(os.TempDir(), "EasyTools", "file")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "payload.bin"), payload, 0644)
	os.WriteFile(filepath.Join(dir, "XorEncrypted.txt"), payload, 0644)
	os.WriteFile(filepath.Join(dir, "XorKey.txt"), []byte{0x01, 0x02, 0x03, 0x04, 0x02}, 0644)
	ensureGoModule(t)

	for _, tc := range []struct {
		loader string
		goos   string
		goarch string
	}{
		{unixLoader, "linux", "amd64"},
		{unixLoader, "darwin", "arm64"},
		{unixLoader, "darwin", "amd64"},
	} {
		name := tc.loader
		if !HasLoaderTemplate(name) {
			t.Fatalf("模板 %s 不存在", name)
		}
		out, gerr := GenGoFile(name, "一体", "xor", "否", "false", "false", "", "false", "", "", nil, nil)
		if gerr != nil || out == "" {
			t.Fatalf("生成失败 %s: %v", name, gerr)
		}
		bin := filepath.Join(dir, "runtime_"+name+"_"+tc.goos+"_"+tc.goarch)
		cmd := exec.Command("go", "build", "-o", bin, out)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GOOS="+tc.goos, "GOARCH="+tc.goarch, "CGO_ENABLED=0",
		)
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("%s -> %s/%s 交叉编译失败: %v\n%s", name, tc.goos, tc.goarch, err, strings.TrimSpace(string(b)))
			continue
		}
		_ = os.Remove(bin)
		t.Logf("%s -> %s/%s 交叉编译成功 ✓", name, tc.goos, tc.goarch)
	}
}
