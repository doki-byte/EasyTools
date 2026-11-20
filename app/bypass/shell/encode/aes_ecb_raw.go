package encode

import (
	"crypto/aes"
	"crypto/cipher"
	cryptorand "crypto/rand" // crypto/rand 别名
	mathrand "math/rand"     // math/rand 别名
	"time"
)

var (
	AesDecodeString = []string{`
	"crypto/aes"
	"crypto/cipher"
`, `

// AES 解密函数，返回字节数组
func AesDecrypt(ciphertext []byte, key []byte) ([]byte, error) {
    block, _ := aes.NewCipher(key)

    blockSize := block.BlockSize()

    iv, ciphertext := ciphertext[:blockSize], ciphertext[blockSize:]
    mode := cipher.NewCBCDecrypter(block, iv)

    plainText := make([]byte, len(ciphertext))
    mode.CryptBlocks(plainText, ciphertext)

    padLen := int(plainText[len(plainText)-1])
    if padLen > 0 && padLen <= blockSize {
        plainText = plainText[:len(plainText)-padLen]
    }

    return plainText, nil // 返回解密后的字节数据
}
`, `
	byteArray, _ := AesDecrypt(//__DATA_AesEncrypted__, //__DATA_AesKey__)`, `
	byteArray, _ := AesDecrypt(EncByteArray, key)
`,
	}
)

// 根据时间戳生成 32 字节的 AES 密钥，返回字节数组
func GenerateAesKey() []byte {
	timestamp := time.Now().Unix()
	mathrand.Seed(timestamp) // 使用 math/rand.Seed
	key := make([]byte, 32)

	// 填充密钥
	for i := 0; i < 32; i++ {
		key[i] = byte(mathrand.Intn(256)) // 使用 math/rand.Intn 填充随机字节
	}

	// 用时间戳的字节填充密钥的前四个字节
	key[0] = byte(timestamp >> 24)
	key[1] = byte(timestamp >> 16)
	key[2] = byte(timestamp >> 8)
	key[3] = byte(timestamp)

	return key // 返回字节数组
}

// AES 加密函数，返回字节数组
func Aes_Encrypt(plaintext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	blockSize := block.BlockSize()
	// PKCS#7 填充：填充值 = 填充字节数（与解码端 AesDecrypt 模板保持一致，
	// 避免旧的"零填充 + 按末字节剥离"不对称导致解密结果尾部残留 0x00）
	padding := blockSize - len(plaintext)%blockSize
	padText := make([]byte, 0, len(plaintext)+padding)
	padText = append(padText, plaintext...)
	for i := 0; i < padding; i++ {
		padText = append(padText, byte(padding))
	}

	ciphertext := make([]byte, len(padText))
	iv := make([]byte, blockSize)

	if _, err := cryptorand.Read(iv); err != nil { // 使用 crypto/rand.Read
		return nil, err
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, padText)

	ivCiphertext := append(iv, ciphertext...) // 将IV和密文连接
	return ivCiphertext, nil                  // 返回字节数组
}
