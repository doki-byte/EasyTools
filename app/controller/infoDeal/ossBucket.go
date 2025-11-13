package infoDeal

import (
	"EasyTools/app/controller/system"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// OssBucker 控制器
type OssBucker struct {
	system.Base
}

// NewOssBucker 创建新的 OssBucker 控制器
func NewOssBucker() *OssBucker {
	return &OssBucker{}
}

// OSS存储桶遍历
func (i *OssBucker) DealOssList(ossURL string) (string, error) {
	s := NewScanner(ossURL)
	excelPath, err := s.Process()
	if err != nil {
		return "", fmt.Errorf("处理失败：%v", err)
	}
	return excelPath, nil
}

func (i *OssBucker) StartVulnScan(urlStr string, configJSON string) map[string]interface{} {
	scanner := NewScanner(urlStr)

	var config VulnScanConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   "配置解析失败: " + err.Error(),
		}
	}

	results, err := scanner.VulnScan(configJSON)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
	}

	return map[string]interface{}{
		"success":       true,
		"results":       results,
		"cloudProvider": scanner.baseURL, // 返回检测到的云服务商信息
	}
}

func (i *OssBucker) DetectCloudProvider(urlStr string) map[string]interface{} {
	detector := &CloudDetector{}
	cloudProvider := detector.DetectCloudProvider(urlStr)

	if cloudProvider == "unknown" {
		return map[string]interface{}{
			"success": false,
			"error":   "无法识别的云服务商URL格式",
		}
	}

	bucket, region := detector.ExtractBucketInfo(urlStr, cloudProvider)

	result := map[string]interface{}{
		"success":       true,
		"cloudProvider": cloudProvider,
		"bucket":        bucket,
		"region":        region,
	}

	// 特殊处理腾讯云APPID
	if cloudProvider == "tencent" && strings.Contains(bucket, "-") {
		parts := strings.Split(bucket, "-")
		if len(parts) > 1 {
			result["appid"] = parts[len(parts)-1]
			result["bucket"] = strings.Join(parts[:len(parts)-1], "-")
		}
	}

	// 特殊处理Azure账户名
	if cloudProvider == "azure" {
		// Azure URL格式: https://account.blob.core.windows.net/container
		if re := regexp.MustCompile(`https?://([^.]+)\.blob\.core\.windows\.net/([^/]+)`); re.MatchString(urlStr) {
			matches := re.FindStringSubmatch(urlStr)
			if len(matches) == 3 {
				result["account"] = matches[1]
				result["bucket"] = matches[2]
			}
		}
	}

	return result
}
