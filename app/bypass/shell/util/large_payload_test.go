package util

import (
	"EasyTools/app/bypass/shell/encode"
	"crypto/rand"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestLargePayloadGenerateAndCompile 大载荷链路回归。
//
// 背景：载荷是以 []byte{0x..,...} 字面量内联进生成源码的，源码体积约为载荷的 5 倍
// （OTP 为 10 倍）。历史上大载荷（fscan 8.7MB → Donut PIC 8.8MB → 44MB 源码）在
// 生成/编译阶段稳定失败，而小载荷（mimikatz 约 1MB）正常，表现为"偶发处理失败"。
// 本用例用 3MB 载荷完整走一遍「写数据文件 → 生成源码 → go build」，
// 确保大载荷不再被体积相关的路径（读写、内联、编译）打断。
func TestLargePayloadGenerateAndCompile(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	ensureGoModule(t)

	dir := filepath.Join(os.TempDir(), "EasyTools", "file")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	// 3MB 随机载荷：一半随机（不可压缩）+ 一半重复（模拟真实 shellcode 结构）
	payload := make([]byte, 3<<20)
	if _, err := rand.Read(payload[:len(payload)/2]); err != nil {
		t.Fatal(err)
	}
	half := payload[:len(payload)/2]
	copy(payload[len(payload)/2:], half)

	// 与 makeFile 一体模式一致：混淆落盘 + 各编码数据文件
	payloadPath, err := encode.WritePayloadAtRest(dir, payload)
	if err != nil {
		t.Fatalf("混淆落盘失败: %v", err)
	}
	back, err := encode.ReadPayloadAtRestRetry(payloadPath, 4, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("读回载荷失败: %v", err)
	}
	if len(back) != len(payload) {
		t.Fatalf("载荷读回长度不一致: got=%d want=%d", len(back), len(payload))
	}

	writes := map[string][]byte{}
	aesKey := encode.GenerateAesKey()
	writes["AesKey.txt"] = aesKey
	aesEnc, _ := encode.Aes_Encrypt(append([]byte(nil), payload...), aesKey)
	writes["AesEncrypted.txt"] = aesEnc

	xorKey, err := encode.GenerateXorKey()
	if err != nil {
		t.Fatal(err)
	}
	writes["XorKey.txt"] = encode.XorKeyToBytes(xorKey)
	writes["XorEncrypted.txt"] = encode.XorEncrypt(append([]byte(nil), payload...), xorKey)

	chKey, _ := encode.GenerateChaCha20Key()
	chNonce, _ := encode.GenerateChaCha20Nonce()
	writes["ChaCha20Key.txt"] = chKey
	writes["ChaCha20Nonce.txt"] = chNonce
	chEnc, _ := encode.ChaCha20Encrypt(chKey, chNonce, append([]byte(nil), payload...))
	writes["ChaCha20Encrypted.txt"] = chEnc

	otpBlob, err := encode.OtpEncrypt(append([]byte(nil), payload...))
	if err != nil {
		t.Fatal(err)
	}
	writes["OtpEncrypted.txt"] = otpBlob
	writes["UrlOrFileEncrypted.txt"] = []byte("deadbeef")
	writes["UrlXorKey.txt"] = []byte{0x42}

	for name, data := range writes {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0644); err != nil {
			t.Fatalf("写入 %s 失败: %v", name, err)
		}
	}

	loaderName := mustWindowsLoader(t)
	out, gerr := GenGoFile(loaderName, "一体", "xor", "否", "false", "false", "", "false", "", "", nil, nil)
	if gerr != nil || out == "" {
		t.Fatalf("大载荷生成源码失败（%s）: %v", loaderName, gerr)
	}
	fi, err := os.Stat(out)
	if err != nil {
		t.Fatalf("读取生成源码失败: %v", err)
	}
	// 载荷 3MB → 内联字面量源码应显著大于载荷本身；过小说明内联被跳过/截断
	if fi.Size() < int64(len(payload)) {
		t.Fatalf("生成源码体积异常（疑似载荷未完整内联）: src=%d payload=%d", fi.Size(), len(payload))
	}
	t.Logf("大载荷生成源码: %s（payload=%d, src=%d）", out, len(payload), fi.Size())

	bin := filepath.Join(dir, "large_payload_loader.exe")
	cmd := exec.Command("go", "build", "-o", bin, out)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0")
	if b, berr := cmd.CombinedOutput(); berr != nil {
		t.Fatalf("大载荷编译失败: %v\n%s", berr, string(b))
	}
	if st, serr := os.Stat(bin); serr != nil || st.Size() == 0 {
		t.Fatalf("大载荷编译未产出可执行文件: %v", serr)
	}
	_ = os.Remove(bin)
}
