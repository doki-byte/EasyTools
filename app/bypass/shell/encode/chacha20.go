package encode

import (
	"crypto/rand"
	"errors"
	"golang.org/x/crypto/chacha20"
	"io"
)

var (
	ChaCha20DecodeString = []string{`
	"golang.org/x/crypto/chacha20"
`, `

// Encrypt 使用ChaCha20算法加密明文
func ChaCha20Encrypt(key, nonce, plaintext []byte) ([]byte, error) {
	cipher, _ := chacha20.NewUnauthenticatedCipher(key, nonce)
	ciphertext := make([]byte, len(plaintext))
	cipher.XORKeyStream(ciphertext, plaintext)
	return ciphertext, nil
}

// ChaCha20Decrypt 使用ChaCha20算法解密密文
func ChaCha20Decrypt(key, nonce, ciphertext []byte) ([]byte, error) {
	return ChaCha20Encrypt(key, nonce, ciphertext)
}
`, `
	byteArray, _ := ChaCha20Decrypt(//__DATA_ChaCha20Key__, //__DATA_ChaCha20Nonce__, //__DATA_ChaCha20Encrypted__)`, `
	byteArray, _  := ChaCha20Decrypt(ChaCha20Key, ChaCha20Nonce, ChaCha20Encrypted)
`,
	}
)

// GenerateChaCha20Key 生成随机的32字节ChaCha20密钥
func GenerateChaCha20Key() ([]byte, error) {
	key := make([]byte, chacha20.KeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

// GenerateNonce 生成随机的12字节Nonce
func GenerateChaCha20Nonce() ([]byte, error) {
	nonce := make([]byte, chacha20.NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return nonce, nil
}

// ChaCha20Encrypt 使用ChaCha20算法加密明文
func ChaCha20Encrypt(key, nonce, plaintext []byte) ([]byte, error) {
	if len(key) != chacha20.KeySize {
		return nil, errors.New("invalid key length")
	}
	if len(nonce) != chacha20.NonceSize {
		return nil, errors.New("invalid nonce length")
	}

	cipher, err := chacha20.NewUnauthenticatedCipher(key, nonce)
	if err != nil {
		return nil, err
	}

	ciphertext := make([]byte, len(plaintext))
	cipher.XORKeyStream(ciphertext, plaintext)
	return ciphertext, nil
}

// ChaCha20Decrypt 使用ChaCha20算法解密密文
func ChaCha20Decrypt(key, nonce, ciphertext []byte) ([]byte, error) {
	return ChaCha20Encrypt(key, nonce, ciphertext)
}
