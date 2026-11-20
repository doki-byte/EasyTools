package encode

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"testing"
)

// TestAesRoundTrip 验证 Aes_Encrypt 与生成的 loader 内联解密逻辑（模板 AesDecrypt）
// 严格互逆：覆盖边界长度（16 对齐/非对齐）与尾部含 0x00 的载荷。
func TestAesRoundTrip(t *testing.T) {
	sizes := []int{1, 15, 16, 17, 31, 32, 33, 255, 1024, 1025}
	for _, size := range sizes {
		payload := make([]byte, size)
		for i := range payload {
			payload[i] = byte(i*7 + 3)
		}
		// 让部分载荷尾部出现 0x00，验证 PKCS7 填充不会被误剥/残留
		if size > 3 {
			payload[size-1] = 0
			payload[size-2] = 0
		}

		key := GenerateAesKey()
		enc, err := Aes_Encrypt(payload, key)
		if err != nil {
			t.Fatalf("size=%d 加密失败: %v", size, err)
		}

		// 以下解密逻辑与 aes_ecb_raw.go 中 AesDecodeString 模板保持一致
		block, err := aes.NewCipher(key)
		if err != nil {
			t.Fatalf("size=%d NewCipher: %v", size, err)
		}
		blockSize := block.BlockSize()
		iv := enc[:blockSize]
		ct := enc[blockSize:]
		mode := cipher.NewCBCDecrypter(block, iv)
		pt := make([]byte, len(ct))
		mode.CryptBlocks(pt, ct)
		padLen := int(pt[len(pt)-1])
		if padLen > 0 && padLen <= blockSize {
			pt = pt[:len(pt)-padLen]
		}

		if !bytes.Equal(pt, payload) {
			t.Fatalf("size=%d 解密结果与原文不一致: got %d bytes, want %d bytes", size, len(pt), len(payload))
		}
	}
}
