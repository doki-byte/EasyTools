package encode

import (
	"bytes"
	"testing"
)

// TestOtpRoundTrip 验证 OtpEncrypt 生成 密文‖密钥(2N) 结构且可正确还原明文。
func TestOtpRoundTrip(t *testing.T) {
	sizes := []int{1, 2, 15, 16, 255, 1024, 4096}
	for _, n := range sizes {
		payload := make([]byte, n)
		for i := range payload {
			payload[i] = byte(i*7 + 3)
		}
		blob, err := OtpEncrypt(payload)
		if err != nil {
			t.Fatalf("OtpEncrypt(%d) 失败: %v", n, err)
		}
		if len(blob) != 2*n {
			t.Fatalf("OtpEncrypt(%d) blob 长度=%d（应为 %d）", n, len(blob), 2*n)
		}
		// 密文不得与明文相同（随机密钥）
		if bytes.Equal(blob[:n], payload) {
			t.Fatalf("OtpEncrypt(%d) 密文与明文相同，密钥未生效", n)
		}
		// 密钥区不得与密文区相同（随机独立）
		if bytes.Equal(blob[:n], blob[n:]) {
			t.Fatalf("OtpEncrypt(%d) 密钥区与密文区相同", n)
		}
		got, err := OtpDecrypt(blob)
		if err != nil {
			t.Fatalf("OtpDecrypt(%d) 失败: %v", n, err)
		}
		if !bytes.Equal(got, payload) {
			t.Fatalf("OtpDecrypt 还原结果不一致 (n=%d)", n)
		}
	}
}

// TestOtpEncryptRandomness 验证每次生成密钥不同（同一明文两次加密结果不同）。
func TestOtpEncryptRandomness(t *testing.T) {
	payload := []byte("the quick brown fox jumps over the lazy dog 0123456789")
	b1, _ := OtpEncrypt(payload)
	b2, _ := OtpEncrypt(payload)
	if bytes.Equal(b1, b2) {
		t.Fatal("两次 OTP 加密结果相同，随机密钥未生效")
	}
}

// TestOtpDecryptBadInput OTP blob 长度为奇数时报错。
func TestOtpDecryptBadInput(t *testing.T) {
	if _, err := OtpDecrypt([]byte{1, 2, 3}); err == nil {
		t.Fatal("奇数长度 blob 应报错")
	}
	if _, err := OtpEncrypt(nil); err == nil {
		t.Fatal("空载荷应报错")
	}
}
