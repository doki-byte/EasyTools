package encode

import (
	"crypto/rand"
	"io"
)

// XorKey 存储 XOR 加密的密钥
type XorKey struct {
	Key1    byte
	Key2    byte
	Key3    byte
	Key4    byte
	RotateN uint
}

var (
	XorDecodeString = []string{``, `
// XOR 对字节切片执行按位异或操作
func XOR(data []byte, key byte) []byte {
	for i := range data {
		data[i] ^= key
	}
	return data
}

// NOT 对字节切片执行按位取反操作
func NOT(data []byte) []byte {
	for i := range data {
		data[i] = ^data[i]
	}
	return data
}

// RotateRight 对字节切片执行右旋转操作，旋转 n 位
func RotateRight(data []byte, n uint) []byte {
	for i := range data {
		data[i] = (data[i] >> n) | (data[i] << (8 - n)) // 右旋转
	}
	return data
}


// Xor_Decrypt 现在接受并返回 string
func XorDecrypt(data []byte, key1, key2, key3, key4 byte, rotateN uint) []byte {
	// 逆向执行加密步骤
	NOT(data)
	XOR(data, key4)
	NOT(data)
	XOR(data, key3)
	NOT(data)
	RotateRight(data, rotateN)
	NOT(data)
	XOR(data, key2)
	NOT(data)
	XOR(data, key1)

	return data
}
`, `
	XorKeyData := //__DATA_XorKey__
	key1 := XorKeyData[0]
	key2 := XorKeyData[1]
	key3 := XorKeyData[2]
	key4 := XorKeyData[3]
	rotateN := uint(XorKeyData[4])
	byteArray := XorDecrypt(//__DATA_XorEncrypted__, key1, key2, key3, key4, rotateN)`,
	}
)

// XOR 对字节切片执行按位异或操作
func XOR(data []byte, key byte) []byte {
	for i := range data {
		data[i] ^= key
	}
	return data
}

// NOT 对字节切片执行按位取反操作
func NOT(data []byte) []byte {
	for i := range data {
		data[i] = ^data[i]
	}
	return data
}

// RotateLeft 对字节切片执行左旋转操作，旋转 n 位
func RotateLeft(data []byte, n uint) []byte {
	for i := range data {
		data[i] = (data[i] << n) | (data[i] >> (8 - n)) // 左旋转
	}
	return data
}

// RotateRight 对字节切片执行右旋转操作，旋转 n 位
func RotateRight(data []byte, n uint) []byte {
	for i := range data {
		data[i] = (data[i] >> n) | (data[i] << (8 - n)) // 右旋转
	}
	return data
}

// GenerateXorKey 生成随机的 XOR 密钥
func GenerateXorKey() (*XorKey, error) {
	key := &XorKey{}
	keyBytes := make([]byte, 5)
	if _, err := io.ReadFull(rand.Reader, keyBytes); err != nil {
		return nil, err
	}

	key.Key1 = keyBytes[0]
	key.Key2 = keyBytes[1]
	key.Key3 = keyBytes[2]
	key.Key4 = keyBytes[3]
	// 旋转位数限制在 1-7 之间（避免 0 和 8，因为旋转 0 或 8 位没有意义）
	key.RotateN = uint(keyBytes[4]%7) + 1

	return key, nil
}

// XorKeyToBytes 将 XorKey 转换为字节切片（用于保存到文件）
func XorKeyToBytes(key *XorKey) []byte {
	result := make([]byte, 5)
	result[0] = key.Key1
	result[1] = key.Key2
	result[2] = key.Key3
	result[3] = key.Key4
	result[4] = byte(key.RotateN)
	return result
}

// BytesToXorKey 从字节切片恢复 XorKey（用于从文件读取）
func BytesToXorKey(data []byte) *XorKey {
	if len(data) < 5 {
		return nil
	}
	return &XorKey{
		Key1:    data[0],
		Key2:    data[1],
		Key3:    data[2],
		Key4:    data[3],
		RotateN: uint(data[4]),
	}
}

// XorEncrypt 使用指定的密钥对数据进行加密
func XorEncrypt(data []byte, key *XorKey) []byte {
	XOR(data, key.Key1)
	NOT(data)
	XOR(data, key.Key2)
	NOT(data)
	RotateLeft(data, key.RotateN)
	NOT(data)
	XOR(data, key.Key3)
	NOT(data)
	XOR(data, key.Key4)
	NOT(data)

	return data
}

// XorDecrypt 使用指定的密钥对数据进行解密
func XorDecrypt(data []byte, key *XorKey) []byte {
	// 逆向执行加密步骤
	NOT(data)
	XOR(data, key.Key4)
	NOT(data)
	XOR(data, key.Key3)
	NOT(data)
	RotateRight(data, key.RotateN)
	NOT(data)
	XOR(data, key.Key2)
	NOT(data)
	XOR(data, key.Key1)

	return data
}
