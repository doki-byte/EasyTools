package bypassAv

import (
	"EasyTools/app/bypass/shell/encode"
	"EasyTools/app/bypass/shell/util"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestOtpManualStyle 手动 fin2 完全一致的方式（写入精简文件集）构建并验证存活
func TestOtpManualStyle(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	dir := filepath.Join(os.TempDir(), "EasyTools", "file")
	os.MkdirAll(dir, 0o755)
	if err := util.InitGoModule(); err != nil {
		t.Fatal(err)
	}
	payload := []byte{0xEB, 0xFE}
	otpBlob, _ := encode.OtpEncrypt(payload)
	os.WriteFile(filepath.Join(dir, "payload.bin"), payload, 0644)
	os.WriteFile(filepath.Join(dir, "OtpEncrypted.txt"), otpBlob, 0644)
	os.WriteFile(filepath.Join(dir, "XorEncrypted.txt"), []byte{1, 2}, 0644)
	os.WriteFile(filepath.Join(dir, "XorKey.txt"), []byte{1, 2, 3, 4, 5}, 0644)
	os.WriteFile(filepath.Join(dir, "AesEncrypted.txt"), []byte{1, 2}, 0644)
	os.WriteFile(filepath.Join(dir, "AesKey.txt"), make([]byte, 32), 0644)
	os.WriteFile(filepath.Join(dir, "UrlOrFileEncrypted.txt"), []byte("aa"), 0644)
	for i := 0; i < 3; i++ {
		bin := e2eBuildEnc(t, dir, "loader1", "否", "none", "otp")
		run := exec.Command(bin)
		if err := run.Start(); err != nil {
			t.Fatal(err)
		}
		time.Sleep(3 * time.Second)
		err := run.Process.Kill()
		if err != nil {
			run.Wait()
			t.Errorf("run%d OTP loader 3 秒内自行退出", i)
			continue
		}
		run.Wait()
		t.Logf("run%d 存活3秒后被kill ✓", i)
	}
}
