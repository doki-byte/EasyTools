package bypassAv

import (
	"EasyTools/app/bypass/shell/util"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// TestOtpDataIntegrity 单进程内验证: e2ePrep(jsq) -> GenGoFile(otp) 后，
// loader1.go 内联 blob 半段 XOR 应还原 jsq(fc 48 ...)
func TestOtpDataIntegrity(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	root := findRepoRoot(t)
	payload, err := os.ReadFile(filepath.Join(root, "jsq.bin"))
	if err != nil {
		t.Skip("缺 jsq.bin")
	}
	dir := e2ePrep(t, payload)
	pristine := append([]byte(nil), payload...) // e2ePrep 内 XorEncrypt 会变异 payload，须留原版参照
	otpTxt, _ := os.ReadFile(filepath.Join(dir, "OtpEncrypted.txt"))
	t.Logf("OtpEncrypted.txt len=%d head=% x", len(otpTxt), otpTxt[:8])

	// 直接 XOR 校验文件本身
	half := len(otpTxt) / 2
	okFile := true
	for i := 0; i < half; i++ {
		if otpTxt[i]^otpTxt[half+i] != pristine[i] {
			okFile = false
			break
		}
	}
	t.Logf("OtpEncrypted.txt 半段XOR还原jsq: %v", okFile)

	// GenGoFile 后读内联数组校验
	srcPath, err := util.GenGoFile("loader1", "一体", "otp", "否", "false", "false", "", "false", "", "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	src, _ := os.ReadFile(srcPath)
	re := regexp.MustCompile(`:= \[\]byte\{([^}]*)\}`)
	m := re.FindStringSubmatch(string(src))
	if m == nil {
		t.Fatal("未找到内联数组")
	}
	var arr []byte
	for _, tok := range strings.Split(m[1], ",") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		v, _ := strconv.ParseUint(tok, 0, 8)
		arr = append(arr, byte(v))
	}
	t.Logf("内联数组 len=%d head=% x", len(arr), arr[:8])
	if len(arr) != len(otpTxt) {
		t.Fatalf("内联数组(%d)与OtpEncrypted.txt(%d)长度不一致!", len(arr), len(otpTxt))
	}
	same := true
	for i := range arr {
		if arr[i] != otpTxt[i] {
			same = false
			break
		}
	}
	t.Logf("内联==文件内容: %v", same)
	half2 := len(arr) / 2
	okInline := true
	for i := 0; i < half2; i++ {
		if arr[i]^arr[half2+i] != pristine[i] {
			okInline = false
			break
		}
	}
	t.Logf("内联数组半段XOR还原jsq: %v", okInline)
	if !okInline {
		t.Errorf("内联数据不能还原jsq → 生成的loader会执行垃圾")
	}
}
