package util

import (
	"EasyTools/app/bypass/shell/encode"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestOtpModulesCompile OTP 一次一密编译门禁：对代表模板族逐一生成 + go build，
// 验证就地解码注入（loader1/loader45 拷贝行族）与函数型回退（loader53 无拷贝行族）
// 生成的源码均可编译。不执行产物。
func TestOtpModulesCompile(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	ensureGoModule(t)
	dir := filepath.Join(os.TempDir(), "EasyTools", "file")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// 准备数据文件（与 makeFile 一体模式一致；OtpEncrypted.txt 必须为 密文‖密钥 blob）
	payload := []byte{0xEB, 0xFE} // x64 死循环
	otpBlob, err := encode.OtpEncrypt(payload)
	if err != nil {
		t.Fatalf("OtpEncrypt 失败: %v", err)
	}
	os.WriteFile(filepath.Join(dir, "payload.bin"), payload, 0644)
	os.WriteFile(filepath.Join(dir, "OtpEncrypted.txt"), otpBlob, 0644)
	os.WriteFile(filepath.Join(dir, "XorEncrypted.txt"), []byte{1, 2}, 0644)
	os.WriteFile(filepath.Join(dir, "XorKey.txt"), []byte{1, 2, 3, 4, 5}, 0644)
	os.WriteFile(filepath.Join(dir, "AesEncrypted.txt"), []byte{1, 2}, 0644)
	os.WriteFile(filepath.Join(dir, "AesKey.txt"), make([]byte, 32), 0644)
	os.WriteFile(filepath.Join(dir, "ChaCha20Encrypted.txt"), []byte{1, 2}, 0644)
	os.WriteFile(filepath.Join(dir, "ChaCha20Key.txt"), make([]byte, 32), 0644)
	os.WriteFile(filepath.Join(dir, "ChaCha20Nonce.txt"), make([]byte, 12), 0644)
	os.WriteFile(filepath.Join(dir, "UrlOrFileEncrypted.txt"), []byte("aa"), 0644)
	os.WriteFile(filepath.Join(dir, "UrlXorKey.txt"), []byte{1}, 0644)

	// 代表族：loader1(RtlMoveMemory 就地) / loader26(callback(shellcode) 就地)
	// 以及默认模板（无拷贝行 -> 函数型回退）。
	// 模板集增删后这里的名字可能不存在，逐个校验后跳过即可。
	names := []string{"loader1", "loader26"}
	if defWin, ok := DefaultWindowsLoader(); ok {
		names = append(names, defWin)
	}
	for _, name := range names {
		if !HasLoaderTemplate(name) {
			continue
		}
		out, gerr := GenGoFile(name, "一体", "otp", "否", "false", "false", "", "false", "", "", nil, nil)
		if gerr != nil || out == "" {
			t.Fatalf("GenGoFile(%s, otp) 失败: %v", name, gerr)
		}
		bin := filepath.Join(dir, "otp_"+name+".exe")
		cmd := exec.Command("go", "build", "-o", bin, out)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0")
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("%s OTP 编译失败: %v\n%s", name, err, b)
			continue
		}
		t.Logf("%s OTP 编译 OK", name)
		os.Remove(bin)
	}
}
