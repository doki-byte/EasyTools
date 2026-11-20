package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"log/slog"
	"math/rand"
	"strings"
	"time"
)

// RandString 生成指定长度随机字符串
func RandString(length int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyz"
	data := []byte(str)
	var result []byte
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < length; i++ {
		result = append(result, data[r.Intn(len(data))])
	}
	return string(result)
}

func AesEncrypt(orig string, key string) (string, error) {
	orig = strings.TrimSpace(orig)
	defer func() {
		if err := recover(); err != nil {
			slog.Error("AES加密错误", "err_msg", err)
		}
	}()
	if len(orig) == 0 {
		return "", nil
	}
	if len(key) != 32 {
		return "", errors.New("加密需要的key长度错误")
	}
	// 转成字节数组
	origData := []byte(orig)
	k := []byte(key)
	// 分组秘钥
	block, err := aes.NewCipher(k)
	if err != nil {
		return "", err
	}
	// 获取秘钥块的长度
	blockSize := block.BlockSize()
	//补码
	PKCS7Padding := func(ciphertext []byte, blockSize int) []byte {
		padding := blockSize - len(ciphertext)%blockSize
		padText := bytes.Repeat([]byte{byte(padding)}, padding)
		return append(ciphertext, padText...)
	}
	origData = PKCS7Padding(origData, blockSize)
	// 加密模式
	blockMode := cipher.NewCBCEncrypter(block, k[:blockSize])
	// 创建数组
	crypt := make([]byte, len(origData))
	// 加密
	blockMode.CryptBlocks(crypt, origData)
	return base64.StdEncoding.EncodeToString(crypt), nil
}

func AesDecrypt(crypt string, key string) (string, error) {
	crypt = strings.TrimSpace(crypt)
	if len(crypt) == 0 {
		return "", nil
	}
	if len(crypt) < 1 || len(key) != 32 {
		return "", errors.New("解密需要的key长度错误")
	}
	// 转成字节数组
	cryptByte, err := base64.StdEncoding.DecodeString(crypt)
	if err != nil {
		slog.Error("AES解密错误: base64解码失败", "err_msg", err)
		return "", errors.New("base64解码失败: " + err.Error())
	}

	k := []byte(key)
	// 分组秘钥
	block, err := aes.NewCipher(k)
	if err != nil {
		return "", err
	}
	// 获取秘钥块的长度
	blockSize := block.BlockSize()

	// 检查密文长度是否是块大小的倍数
	if len(cryptByte) < blockSize || len(cryptByte)%blockSize != 0 {
		slog.Error("AES解密错误", "err_msg", "crypto/cipher: input not full blocks", "data_length", len(cryptByte), "block_size", blockSize)
		return "", errors.New("crypto/cipher: input not full blocks")
	}

	// 加密模式
	blockMode := cipher.NewCBCDecrypter(block, k[:blockSize])
	// 创建数组
	orig := make([]byte, len(cryptByte))
	// 解密
	blockMode.CryptBlocks(orig, cryptByte)
	// 去补全码
	PKCS7UnPadding := func(origData []byte) ([]byte, error) {
		if len(origData) == 0 {
			return nil, errors.New("解密数据为空")
		}
		length := len(origData)
		unPadding := int(origData[length-1])
		if unPadding > length || unPadding == 0 {
			return nil, errors.New("无效的填充数据")
		}
		return origData[:(length - unPadding)], nil
	}
	orig, err = PKCS7UnPadding(orig)
	if err != nil {
		slog.Error("AES解密错误: 去除填充失败", "err_msg", err)
		return "", err
	}
	return string(orig), nil
}

// Base64Encode base64编码
func Base64Encode(s string) string {
	if s == "" {
		return ""
	}
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// Base64Decode base64解码
func Base64Decode(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// IsBase64 检查字符串是否是base64编码
func IsBase64(s string) bool {
	if s == "" {
		return false
	}
	// base64编码的字符串只包含特定字符，且长度是4的倍数（加上可能的填充）
	// 简单的检查：尝试解码，如果成功则认为是base64
	_, err := base64.StdEncoding.DecodeString(s)
	return err == nil
}
