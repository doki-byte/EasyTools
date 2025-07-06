package controller

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type JwtCrackController struct {
	Base
}

type JwtResult struct {
	JwtToken  string                 `json:"jwt_token"`
	Header    map[string]interface{} `json:"header"`
	Payload   map[string]interface{} `json:"payload"`
	Signature string                 `json:"signature"`
	Valid     bool                   `json:"valid"`
	Secret    string                 `json:"secret,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

func NewJwtCrackController() *JwtCrackController {
	return &JwtCrackController{}
}

func (j *JwtCrackController) EncodeJWTWithAlg(alg string, keyPath string, payload map[string]interface{}) JwtResult {
	var signingMethod jwt.SigningMethod
	var key interface{}
	var err error

	switch alg {
	case "HS256":
		signingMethod = jwt.SigningMethodHS256
		key = []byte(keyPath)
	case "HS384":
		signingMethod = jwt.SigningMethodHS384
		key = []byte(keyPath)
	case "HS512":
		signingMethod = jwt.SigningMethodHS512
		key = []byte(keyPath)
	case "RS256", "RS384", "RS512", "PS256", "PS384", "PS512":
		signingMethod = jwt.GetSigningMethod(alg)
		key, err = loadPrivateKeyFromPEM(keyPath)
	case "ES256", "ES384", "ES512":
		signingMethod = jwt.GetSigningMethod(alg)
		key, err = loadECDSAPrivateKeyFromPEM(keyPath)
	case "EdDSA":
		signingMethod = jwt.SigningMethodEdDSA
		key, err = loadEd25519PrivateKeyFromPEM(keyPath)
	case "none", "None":
		signingMethod = jwt.SigningMethodNone
		key = jwt.UnsafeAllowNoneSignatureType
	default:
		return JwtResult{
			Error: "unsupported algorithm: " + alg,
		}
	}

	if err != nil {
		return JwtResult{
			Error: err.Error(),
		}
	}

	claims := jwt.MapClaims{}
	for k, v := range payload {
		claims[k] = v
	}
	claims["exp"] = time.Now().Add(time.Hour * 1).Unix()
	claims["iat"] = time.Now().Unix()

	token := jwt.NewWithClaims(signingMethod, claims)

	// 先生成完整 JWT 字符串
	signedToken, err := token.SignedString(key)
	if err != nil {
		return JwtResult{
			Error: "签名失败: " + err.Error(),
		}
	}

	// 提取签名部分
	sigBytes, err := extractSignature(signedToken)
	if err != nil {
		return JwtResult{
			Error: "提取签名失败: " + err.Error(),
		}
	}

	return JwtResult{
		JwtToken:  signedToken,
		Header:    token.Header,
		Payload:   claims,
		Signature: hex.EncodeToString(sigBytes),
		Valid:     true,
	}
}

func (j *JwtCrackController) DecodeJWTWithAlg(tokenStr string, alg string, keyPath string) JwtResult {
	// 如果没有密钥路径，走“只解码”逻辑
	if keyPath == "" || alg == "none" || alg == "None" {
		return decodeWithoutVerify(tokenStr)
	}

	var key interface{}
	var err error

	switch alg {
	case "HS256", "HS384", "HS512":
		key = []byte(keyPath)
	case "RS256", "RS384", "RS512", "PS256", "PS384", "PS512":
		key, err = loadPublicKeyFromPEM(keyPath)
	case "ES256", "ES384", "ES512":
		key, err = loadECDSAPublicKeyFromPEM(keyPath)
	case "EdDSA":
		key, err = loadEd25519PublicKeyFromPEM(keyPath)
	default:
		return JwtResult{
			Header:    nil,
			Payload:   nil,
			Signature: "null",
			Valid:     false,
			Error:     "unsupported algorithm:" + alg,
		}
	}

	if err != nil {
		return JwtResult{
			Header:    nil,
			Payload:   nil,
			Signature: "null",
			Valid:     false,
			Error:     err.Error(),
		}
	}

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != alg {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return key, nil
	})

	if err != nil {
		return JwtResult{
			Header:    nil,
			Payload:   nil,
			Signature: "null",
			Valid:     false,
			Error:     err.Error(),
		}
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		sig, _ := extractSignature(tokenStr)
		return JwtResult{
			Header:    token.Header,
			Payload:   claims,
			Signature: hex.EncodeToString(sig),
			Valid:     token.Valid,
		}
	}

	return JwtResult{
		Header:    nil,
		Payload:   nil,
		Signature: "null",
		Valid:     false,
		Error:     "invalid token",
	}
}

func (j *JwtCrackController) ChooseJwtFile() (string, error) {

	return runtime.OpenFileDialog(j.ctx, runtime.OpenDialogOptions{
		Title: "选择jwt字典",
	})
}

// BruteForceJWT 尝试使用 keyList 爆破 JWT 签名密钥
func (j *JwtCrackController) BruteForceJWT(tokenStr string, alg string, filepath string) JwtResult {

	keyList, err := readKeysFromFile(filepath)
	if err != nil {
		return JwtResult{
			Error: "读取密钥字典失败: " + err.Error(),
		}
	}

	if tokenStr == "" || alg == "" || len(keyList) == 0 {
		return JwtResult{
			Error: "参数不能为空或字典为空",
		}
	}

	var lastErr error

	for _, key := range keyList {
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if token.Method.Alg() != alg {
				return nil, fmt.Errorf("不匹配的算法: 期望 %s, 实际 %s", alg, token.Method.Alg())
			}
			return []byte(key), nil
		})
		if err != nil {
			lastErr = err
			continue
		}

		if token.Valid {
			// 解析成功，返回密钥、payload、header
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				sig, _ := extractSignature(tokenStr)
				return JwtResult{
					Header:    token.Header,
					Payload:   claims,
					Signature: hex.EncodeToString(sig),
					Valid:     token.Valid,
					Secret:    key,
				}
			}
			return JwtResult{
				Secret: key,
				Error:  "key 解析成功，但无法获取 claims",
			}
		}
	}

	// 所有密钥都尝试失败
	return JwtResult{
		Error: "所有密钥尝试失败: " + lastErr.Error()}
}

// 从文件读取密钥列表
func readKeysFromFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var keys []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			keys = append(keys, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return keys, nil
}

func decodeWithoutVerify(tokenStr string) JwtResult {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return JwtResult{
			Header:    nil,
			Payload:   nil,
			Signature: "null",
			Valid:     false,
			Error:     errors.New("invalid JWT format").Error(),
		}
	}

	decode := func(s string) ([]byte, error) {
		return base64.RawURLEncoding.DecodeString(s)
	}

	headerBytes, err := decode(parts[0])
	if err != nil {
		return JwtResult{
			Header:    nil,
			Payload:   nil,
			Signature: "null",
			Valid:     false,
			Error:     "header error:" + err.Error(),
		}
	}

	payloadBytes, err := decode(parts[1])
	if err != nil {
		return JwtResult{
			Header:    nil,
			Payload:   nil,
			Signature: "null",
			Valid:     false,
			Error:     "payload error:" + err.Error(),
		}
	}

	var header map[string]interface{}
	var payload map[string]interface{}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return JwtResult{
			Header:    nil,
			Payload:   nil,
			Signature: "null",
			Valid:     false,
			Error:     "invalid header JSON:" + err.Error(),
		}
	}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return JwtResult{
			Header:    nil,
			Payload:   nil,
			Signature: "null",
			Valid:     false,
			Error:     "invalid payload JSON:" + err.Error(),
		}
	}

	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		// 不是 fatal 错误，只打印签名原文
		sig = []byte(parts[2])
	}

	return JwtResult{
		Header:    header,
		Payload:   payload,
		Signature: hex.EncodeToString(sig),
		Valid:     false,
		Error:     "",
	}
}

// 提取 JWT 签名（第三段）
func extractSignature(tokenStr string) ([]byte, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid JWT format")
	}
	return base64.RawURLEncoding.DecodeString(parts[2])
}

// --- 密钥加载函数 ---

func loadPrivateKeyFromPEM(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("invalid PEM format")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func loadPublicKeyFromPEM(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("invalid PEM format")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return pub.(*rsa.PublicKey), nil
}

func loadECDSAPrivateKeyFromPEM(path string) (*ecdsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("invalid PEM format")
	}
	return x509.ParseECPrivateKey(block.Bytes)
}

func loadECDSAPublicKeyFromPEM(path string) (*ecdsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("invalid PEM format")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return pub.(*ecdsa.PublicKey), nil
}

func loadEd25519PrivateKeyFromPEM(path string) (ed25519.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("invalid PEM format")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	edKey, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, errors.New("not an Ed25519 private key")
	}

	return edKey, nil
}

func loadEd25519PublicKeyFromPEM(path string) (ed25519.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("invalid PEM format")
	}

	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	edKey, ok := key.(ed25519.PublicKey)
	if !ok {
		return nil, errors.New("not an Ed25519 public key")
	}

	return edKey, nil
}
