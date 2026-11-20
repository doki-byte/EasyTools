package util

import (
	godonut "EasyTools/app/bypass/donut"
	"EasyTools/app/bypass/shell/encode"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const execMarkerName = "easytools_exec_ok.txt"

// buildMarkerPayload 生成"可观测副作用"的测试载荷：
// 一个最小 Windows EXE（写出标记文件后短暂驻留），经 Donut 转成 PIC。
// 标记文件出现 = loader 真的把 payload 执行起来了（比"进程存活"更强的证据）。
func buildMarkerPayload(t *testing.T, dir string) []byte {
	t.Helper()
	srcDir := filepath.Join(dir, "marker")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := `package main

import (
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	p := os.Getenv("TEMP") + "\\` + execMarkerName + `"
	_ = os.WriteFile(p, []byte("pid="+strconv.Itoa(os.Getpid())+" args="+strings.Join(os.Args[1:], " ")), 0644)
	time.Sleep(2 * time.Second)
}
`
	srcPath := filepath.Join(srcDir, "main.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	exePath := filepath.Join(srcDir, "marker.exe")
	cmd := exec.Command("go", "build", "-o", exePath, srcPath)
	cmd.Dir = srcDir
	cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("构建标记载荷 EXE 失败: %v\n%s", err, string(out))
	}
	exeBytes, err := os.ReadFile(exePath)
	if err != nil {
		t.Fatal(err)
	}
	pic, err := godonut.BuildPIC(exeBytes, "", "x64")
	if err != nil {
		t.Fatalf("Donut 转换标记载荷失败: %v", err)
	}
	t.Logf("标记载荷: exe=%d → PIC=%d", len(exeBytes), len(pic))
	return pic
}

// writeOneShotDataFiles 写入 makeFile「一体」模式所需的全部数据文件（xor 路径）。
func writeOneShotDataFiles(t *testing.T, dir string, payload []byte) {
	t.Helper()
	// 载荷混淆落盘 + 读回（顺带覆盖 payload-at-rest 链路）
	payloadPath, err := encode.WritePayloadAtRest(dir, payload)
	if err != nil {
		t.Fatalf("载荷混淆落盘失败: %v", err)
	}
	back, err := encode.ReadPayloadAtRestRetry(payloadPath, 4, 200*time.Millisecond)
	if err != nil {
		t.Fatalf("载荷读回失败: %v", err)
	}
	if len(back) != len(payload) {
		t.Fatalf("载荷读回长度不一致: got=%d want=%d", len(back), len(payload))
	}

	xorKey, err := encode.GenerateXorKey()
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{
		"XorKey.txt":            encode.XorKeyToBytes(xorKey),
		"XorEncrypted.txt":      encode.XorEncrypt(append([]byte(nil), payload...), xorKey),
		"AesKey.txt":            make([]byte, 32),
		"AesEncrypted.txt":      {1, 2},
		"ChaCha20Key.txt":       make([]byte, 32),
		"ChaCha20Nonce.txt":     make([]byte, 12),
		"ChaCha20Encrypted.txt": {1, 2},
		"UrlOrFileEncrypted.txt": []byte("deadbeef"),
		"UrlXorKey.txt":         {0x42},
	}
	otpBlob, err := encode.OtpEncrypt(append([]byte(nil), payload...))
	if err != nil {
		t.Fatal(err)
	}
	files["OtpEncrypted.txt"] = otpBlob

	for name, data := range files {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0644); err != nil {
			t.Fatalf("写入 %s 失败: %v", name, err)
		}
	}
}

// TestAllLoadersExecutePayload 全量 loader 执行验证：
// 每个 Windows 模板都走「生成源码 → 编译 → 后处理 → 运行」，并断言
// Donut 载荷真的执行（标记文件出现）。这是"免杀产物能不能用"的最终判据。
func TestAllLoadersExecutePayload(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	ensureGoModule(t)
	dir := filepath.Join(os.TempDir(), "EasyTools", "file")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	markerPath := filepath.Join(os.TempDir(), execMarkerName)

	pic := buildMarkerPayload(t, dir)
	writeOneShotDataFiles(t, dir, pic)

	type result struct {
		loader string
		ok     bool
		note   string
	}
	var results []result

	for _, name := range allModuleNames() {
		if unixLoaderNames[name] {
			continue // Unix 模板不在本机执行
		}
		if name == LoaderBaiJiaHei {
			// 白加黑专用模板带 import "C"，必须走 c-shared 白加黑链路，
			// 一体模式编译必然失败（属预期组合，非缺陷）；由
			// TestBaiJiaHeiDllExecutesPayload 单独验证其 DLL 执行链路。
			results = append(results, result{name, true, "白加黑专用模板，见 DLL 执行专项验证"})
			continue
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					results = append(results, result{name, false, fmt.Sprintf("panic: %v", r)})
				}
			}()

			out, gerr := GenGoFile(name, "一体", "xor", "否", "false", "false", "", "false", "", "", nil, nil)
			if gerr != nil || out == "" {
				results = append(results, result{name, false, "生成源码失败: " + fmt.Sprint(gerr)})
				return
			}
			bin := filepath.Join(dir, "exec_"+name+".exe")
			_ = os.Remove(bin)
			cmd := exec.Command("go", "build", "-o", bin, out)
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0")
			if b, berr := cmd.CombinedOutput(); berr != nil {
				results = append(results, result{name, false, "编译失败: " + firstLine(string(b))})
				return
			}
			PostProcessBuiltBinary(bin, "windows", false)

			_ = os.Remove(markerPath)
			run := exec.Command(bin)
			if serr := run.Start(); serr != nil {
				results = append(results, result{name, false, "启动失败: " + serr.Error()})
				return
			}
			deadline := time.Now().Add(10 * time.Second)
			seen := false
			for time.Now().Before(deadline) {
				if _, serr := os.Stat(markerPath); serr == nil {
					seen = true
					break
				}
				time.Sleep(200 * time.Millisecond)
			}
			if run.Process != nil {
				_ = run.Process.Kill()
			}
			_, _ = run.Process.Wait()
			if seen {
				results = append(results, result{name, true, "payload 已执行"})
			} else {
				results = append(results, result{name, false, "标记文件未出现（payload 未执行）"})
			}
		}()
	}

	passed, failed := 0, 0
	for _, r := range results {
		status := "✓"
		if !r.ok {
			status = "✗"
			failed++
		} else {
			passed++
		}
		t.Logf("[%s] %-9s %s", status, r.loader, r.note)
	}
	t.Logf("执行验证结果: 通过 %d / 失败 %d", passed, failed)
	_ = os.Remove(markerPath)
	if failed > 0 {
		t.Errorf("%d 个 loader 未能执行 payload", failed)
	}
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
