package browserScan

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tealeg/xlsx"
)

// filterSensitiveEndpoints 过滤敏感接口
func (bs *BrowserScan) filterSensitiveEndpoints(endpoints []string, keywords []string) []string {
	if len(keywords) == 0 {
		return endpoints
	}
	
	var filtered []string
	for _, endpoint := range endpoints {
		endpointLower := strings.ToLower(endpoint)
		isSensitive := false
		for _, keyword := range keywords {
			if strings.Contains(endpointLower, strings.ToLower(keyword)) {
				isSensitive = true
				break
			}
		}
		if !isSensitive {
			filtered = append(filtered, endpoint)
		}
	}
	return filtered
}

// getSensitiveKeywords 获取敏感关键词（从前端配置读取，这里简化处理）
func (bs *BrowserScan) getSensitiveKeywords() []string {
	// 实际应该从前端配置读取，这里返回空列表
	// 敏感接口的过滤主要在前端完成
	return []string{}
}

// filterStaticFiles 过滤静态文件（图片、CSS、字体等），不对这些文件进行发包测试
func (bs *BrowserScan) filterStaticFiles(endpoints []string) []string {
	// 静态文件扩展名列表（与extractStaticFiles中的定义保持一致）
	staticExts := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".css", ".svg", ".ico", ".woff", ".woff2", ".ttf", ".eot", ".otf", ".mp4", ".mp3", ".avi", ".mov", ".wmv", ".flv", ".pdf", ".zip", ".rar", ".tar", ".gz"}
	
	var filtered []string
	for _, endpoint := range endpoints {
		endpointLower := strings.ToLower(endpoint)
		isStaticFile := false
		
		// 检查URL是否包含静态文件扩展名
		for _, ext := range staticExts {
			if strings.HasSuffix(endpointLower, ext) || strings.Contains(endpointLower, ext+"?") || strings.Contains(endpointLower, ext+"#") {
				isStaticFile = true
				break
			}
		}
		
		// 只保留非静态文件的端点
		if !isStaticFile {
			filtered = append(filtered, endpoint)
		}
	}
	return filtered
}

// ExportResults 导出扫描结果到Excel（所有标签页信息）
func (bs *BrowserScan) ExportResults(taskID string) (string, error) {
	task, err := bs.GetTask(taskID)
	if err != nil {
		return "", err
	}

	if task.Results == nil {
		return "", fmt.Errorf("任务结果为空")
	}

	// 创建Excel文件
	file := xlsx.NewFile()

	// URLs sheet
	sheet, _ := file.AddSheet("URLs")
	headerRow := sheet.AddRow()
	headerRow.AddCell().Value = "URL"
	for _, url := range task.Results.URLs {
		row := sheet.AddRow()
		row.AddCell().Value = url
	}

	// API端点 sheet
	sheet, _ = file.AddSheet("API端点")
	headerRow = sheet.AddRow()
	headerRow.AddCell().Value = "API端点"
	for _, endpoint := range task.Results.APIEndpoints {
		row := sheet.AddRow()
		row.AddCell().Value = endpoint
	}

	// 域名 sheet
	sheet, _ = file.AddSheet("域名")
	headerRow = sheet.AddRow()
	headerRow.AddCell().Value = "域名"
	for _, domain := range task.Results.Domains {
		row := sheet.AddRow()
		row.AddCell().Value = domain
	}

	// IP地址 sheet
	sheet, _ = file.AddSheet("IP地址")
	headerRow = sheet.AddRow()
	headerRow.AddCell().Value = "IP地址"
	for _, ip := range task.Results.IPs {
		row := sheet.AddRow()
		row.AddCell().Value = ip
	}

	// IP端口 sheet
	sheet, _ = file.AddSheet("IP端口")
	headerRow = sheet.AddRow()
	headerRow.AddCell().Value = "IP端口"
	for _, ipPort := range task.Results.IPPorts {
		row := sheet.AddRow()
		row.AddCell().Value = ipPort
	}

	// 路径 sheet
	sheet, _ = file.AddSheet("路径")
	headerRow = sheet.AddRow()
	headerRow.AddCell().Value = "路径"
	for _, path := range task.Results.Paths {
		row := sheet.AddRow()
		row.AddCell().Value = path
	}

	// 不完整路径 sheet
	sheet, _ = file.AddSheet("不完整路径")
	headerRow = sheet.AddRow()
	headerRow.AddCell().Value = "不完整路径"
	for _, path := range task.Results.IncompletePaths {
		row := sheet.AddRow()
		row.AddCell().Value = path
	}

	// JS文件 sheet
	sheet, _ = file.AddSheet("JS文件")
	headerRow = sheet.AddRow()
	headerRow.AddCell().Value = "JS文件"
	for _, jsFile := range task.Results.JSFiles {
		row := sheet.AddRow()
		row.AddCell().Value = jsFile
	}

	// 静态文件 sheet
	sheet, _ = file.AddSheet("静态文件")
	headerRow = sheet.AddRow()
	headerRow.AddCell().Value = "静态文件"
	for _, staticFile := range task.Results.StaticFiles {
		row := sheet.AddRow()
		row.AddCell().Value = staticFile
	}

	// 算法 sheet
	sheet, _ = file.AddSheet("算法")
	headerRow = sheet.AddRow()
	headerRow.AddCell().Value = "算法"
	for _, algorithm := range task.Results.Algorithms {
		row := sheet.AddRow()
		row.AddCell().Value = algorithm
	}

	// JWT sheet
	sheet, _ = file.AddSheet("JWT")
	headerRow = sheet.AddRow()
	headerRow.AddCell().Value = "JWT Token"
	for _, jwt := range task.Results.JWT {
		row := sheet.AddRow()
		row.AddCell().Value = jwt
	}

	// 手机号 sheet
	sheet, _ = file.AddSheet("手机号")
	headerRow = sheet.AddRow()
	headerRow.AddCell().Value = "手机号"
	for _, mobile := range task.Results.Mobiles {
		row := sheet.AddRow()
		row.AddCell().Value = mobile
	}

	// 邮箱 sheet
	sheet, _ = file.AddSheet("邮箱")
	headerRow = sheet.AddRow()
	headerRow.AddCell().Value = "邮箱"
	for _, email := range task.Results.Emails {
		row := sheet.AddRow()
		row.AddCell().Value = email
	}

	// SecurityKey sheet
	sheet, _ = file.AddSheet("SecurityKey")
	headerRow = sheet.AddRow()
	headerRow.AddCell().Value = "SecurityKey"
	for _, key := range task.Results.SecurityKeys {
		row := sheet.AddRow()
		row.AddCell().Value = key
	}

	// Password sheet
	sheet, _ = file.AddSheet("Password")
	headerRow = sheet.AddRow()
	headerRow.AddCell().Value = "Password"
	for _, password := range task.Results.Passwords {
		row := sheet.AddRow()
		row.AddCell().Value = password
	}

	// Username sheet
	sheet, _ = file.AddSheet("Username")
	headerRow = sheet.AddRow()
	headerRow.AddCell().Value = "Username"
	for _, username := range task.Results.Usernames {
		row := sheet.AddRow()
		row.AddCell().Value = username
	}

	// AKSK sheet
	sheet, _ = file.AddSheet("AKSK")
	headerRow = sheet.AddRow()
	headerRow.AddCell().Value = "AKSK"
	for _, aksk := range task.Results.AKSKs {
		row := sheet.AddRow()
		row.AddCell().Value = aksk
	}

	// AppID sheet
	sheet, _ = file.AddSheet("AppID")
	headerRow = sheet.AddRow()
	headerRow.AddCell().Value = "AppID"
	for _, appID := range task.Results.AppIDs {
		row := sheet.AddRow()
		row.AddCell().Value = appID
	}

	// AccessKey sheet
	sheet, _ = file.AddSheet("AccessKey")
	headerRow = sheet.AddRow()
	headerRow.AddCell().Value = "AccessKey"
	for _, accessKey := range task.Results.AccessKeys {
		row := sheet.AddRow()
		row.AddCell().Value = accessKey
	}

	// API Key sheet
	sheet, _ = file.AddSheet("API Key")
	headerRow = sheet.AddRow()
	headerRow.AddCell().Value = "API Key"
	for _, apiKey := range task.Results.APIKeys {
		row := sheet.AddRow()
		row.AddCell().Value = apiKey
	}

	// Secret Key sheet
	sheet, _ = file.AddSheet("Secret Key")
	headerRow = sheet.AddRow()
	headerRow.AddCell().Value = "Secret Key"
	for _, secretKey := range task.Results.SecretKeys {
		row := sheet.AddRow()
		row.AddCell().Value = secretKey
	}

	// 身份证 sheet
	if len(task.Results.SFZ) > 0 || len(task.Results.IDCards) > 0 {
		sheet, _ = file.AddSheet("身份证")
		headerRow = sheet.AddRow()
		headerRow.AddCell().Value = "身份证"
		// 合并SFZ和IDCards（去重）
		idCardMap := make(map[string]bool)
		for _, idCard := range task.Results.SFZ {
			if !idCardMap[idCard] {
				row := sheet.AddRow()
				row.AddCell().Value = idCard
				idCardMap[idCard] = true
			}
		}
		for _, idCard := range task.Results.IDCards {
			if !idCardMap[idCard] {
				row := sheet.AddRow()
				row.AddCell().Value = idCard
				idCardMap[idCard] = true
			}
		}
	}

	// Vue / Router sheet
	if task.Results.VueAnalysis != nil && (len(task.Results.VueAnalysis.AllRoutes) > 0 || task.Results.VueAnalysis.VueDetected) {
		sheet, _ = file.AddSheet("Vue路由")
		sum := sheet.AddRow()
		sum.AddCell().Value = "摘要"
		sum.AddCell().Value = fmt.Sprintf("vueDetected=%v routerDetected=%v version=%s routerBase=%s currentPath=%s err=%s",
			task.Results.VueAnalysis.VueDetected, task.Results.VueAnalysis.RouterDetected,
			task.Results.VueAnalysis.VueVersion, task.Results.VueAnalysis.RouterBase,
			task.Results.VueAnalysis.CurrentPath, task.Results.VueAnalysis.Error)
		headerRow = sheet.AddRow()
		headerRow.AddCell().Value = "名称"
		headerRow.AddCell().Value = "路径"
		for _, r := range task.Results.VueAnalysis.AllRoutes {
			row := sheet.AddRow()
			row.AddCell().Value = r.Name
			row.AddCell().Value = r.Path
		}
	}

	// 数据包日志 sheet
	if len(task.PacketLogs) > 0 {
		sheet, _ = file.AddSheet("数据包日志")
		headerRow = sheet.AddRow()
		headerRow.AddCell().Value = "URL"
		headerRow.AddCell().Value = "方法"
		headerRow.AddCell().Value = "状态码"
		headerRow.AddCell().Value = "耗时"
		headerRow.AddCell().Value = "提取信息数量"
		headerRow.AddCell().Value = "提取数据"
		for _, packet := range task.PacketLogs {
			row := sheet.AddRow()
			row.AddCell().Value = packet.URL
			row.AddCell().Value = packet.Method
			row.AddCell().Value = fmt.Sprintf("%d", packet.StatusCode)
			row.AddCell().Value = packet.Duration
			row.AddCell().Value = fmt.Sprintf("%d", len(packet.ExtractedInfo))
			// 提取所有的数据值
			var values []string
			for _, info := range packet.ExtractedInfo {
				if info.Value != "" {
					values = append(values, info.Value)
				}
			}
			row.AddCell().Value = strings.Join(values, "; ")
		}
	}

	// 保存文件
	outputDir := filepath.Join("EasyToolsFiles", "file")
	_ = os.MkdirAll(outputDir, 0755)
	filename := filepath.Join(outputDir, fmt.Sprintf("browser_scan_%s_%s.xlsx", taskID, time.Now().Format("20060102_150405")))

	err = file.Save(filename)
	if err != nil {
		return "", err
	}

	return filename, nil
}
