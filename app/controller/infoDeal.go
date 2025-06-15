package controller

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gogs/chardet"
	"github.com/xuri/excelize/v2"
)

// InfoDeal 控制器
type InfoDeal struct {
	Base
}

// NewInfoDeal 创建新的 InfoDeal 控制器
func NewInfoDeal() *InfoDeal {
	return &InfoDeal{}
}

// 定义 EasyToolsFiles\file 路径
const baseDir = "EasyToolsFiles/file"

// 上传文件
func (i *InfoDeal) UploadFile(fileName, content string) error {
	// 确保目录存在
	if err := os.MkdirAll(baseDir, os.ModePerm); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}

	// 保存文件
	filePath := filepath.Join(baseDir, fileName)
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("保存文件失败: %v", err)
	}
	return nil
}

// FscanResultDeal 处理文件并生成 Excel
func (i *InfoDeal) FscanResultDeal(fileName string) (string, error) {
	// 构造文件路径
	filePath := filepath.Join(baseDir, fileName)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("文件 [%s] 不存在", filePath)
	}

	datalist, datastr := openFile(filePath)

	xlsxFile := excelize.NewFile()

	// 调用各功能函数
	openPort(datalist, xlsxFile)
	aliveIP(datalist, xlsxFile)
	bugExpList(datalist, xlsxFile)
	bugPocList(datalist, xlsxFile)
	osList(datalist, xlsxFile)
	getTitle(datalist, xlsxFile)
	getPassword(datalist, xlsxFile)
	fingerOut(datastr, xlsxFile)
	netInfo(datastr, xlsxFile)
	netBios(datalist, xlsxFile)

	// 删除默认 sheet
	sheet := xlsxFile.GetSheetList()[0]
	xlsxFile.DeleteSheet(sheet)

	// 保存处理后的 Excel 文件
	saveFileName := fmt.Sprintf("%s_%s.xlsx", strings.TrimSuffix(fileName, ".txt"), time.Now().Format("2006-01-02_15-04-05"))
	saveFilePath := filepath.Join(baseDir, saveFileName)
	if err := xlsxFile.SaveAs(saveFilePath); err != nil {
		return "", fmt.Errorf("保存文件失败: %v", err)
	}

	return saveFilePath, nil
}

// GetExcelContent 读取 Excel 文件并返回 Base64 编码内容
func (i *InfoDeal) GetExcelContent(filePath string) (string, error) {
	// 确保文件存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("文件 [%s] 不存在", filePath)
	}

	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %v", err)
	}

	// 编码为 Base64 返回
	return base64.StdEncoding.EncodeToString(content), nil
}

// 检测文件编码类型
func detectEncoding(file string) string {
	data, err := os.ReadFile(file)
	if err != nil {
		fmt.Println("读取文件错误:", err)
		return ""
	}

	detector := chardet.NewTextDetector()
	result, err := detector.DetectBest(data)
	if err != nil {
		fmt.Println("检测编码错误:", err)
		return ""
	}

	return result.Charset
}

// 打开文件并读取内容
func openFile(fileName string) ([]string, string) {
	encoding := detectEncoding(fileName)
	if encoding == "" {
		fmt.Println("检测编码失败，默认使用 UTF-8")
		encoding = "UTF-8"
	}

	file, err := os.Open(fileName)
	if err != nil {
		fmt.Println("打开文件错误:", err)
		return nil, ""
	}
	defer file.Close()

	var datalist []string
	var datastr strings.Builder

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		datalist = append(datalist, line)
		datastr.WriteString(line + "\n")
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("读取文件错误:", err)
	}

	return datalist, datastr.String()
}

// 处理开放端口
func openPort(datalist []string, xlsxFile *excelize.File) {
	sheetName := "OpenPort"
	xlsxFile.NewSheet(sheetName)
	xlsxFile.SetCellValue(sheetName, "A1", "IP")
	xlsxFile.SetCellValue(sheetName, "B1", "Port")

	row := 2
	for _, line := range datalist {
		// 正则匹配IP:端口的组合
		matches := regexp.MustCompile(`\d+\.\d+\.\d+\.\d+:\d+`).FindAllString(line, -1)
		for _, match := range matches {
			// 按冒号分割IP和端口
			split := strings.Split(match, ":")
			if len(split) == 2 {
				xlsxFile.SetCellValue(sheetName, fmt.Sprintf("A%d", row), split[0]) // 设置IP
				xlsxFile.SetCellValue(sheetName, fmt.Sprintf("B%d", row), split[1]) // 设置端口
				row++
			}
		}
	}
}

// 处理存活 IP
func aliveIP(datalist []string, xlsxFile *excelize.File) {
	sheetName := "AliveIP"
	xlsxFile.NewSheet(sheetName)
	xlsxFile.SetCellValue(sheetName, "A1", "IP Range")
	xlsxFile.SetCellValue(sheetName, "B1", "Active Count")

	row := 2
	for _, line := range datalist {
		// 正则匹配 IP 段，例如 192.168.1.0/24
		matches := regexp.MustCompile(`\[\*]\sLiveTop\s\d+\.\d+\.\d+\.\d+/\d+.*`).FindAllString(line, -1)
		for _, match := range matches {
			// 提取 IP 范围
			ipRangeMatches := regexp.MustCompile(`\d+\.\d+\.\d+\.\d+/\d+`).FindAllString(match, -1)
			for _, ipRange := range ipRangeMatches {
				// 提取活跃 IP 数量
				countMatches := regexp.MustCompile(`\d+$`).FindAllString(match, -1)
				if len(countMatches) > 0 {
					// 设置表格中的 IP 范围和活跃 IP 数量
					xlsxFile.SetCellValue(sheetName, fmt.Sprintf("A%d", row), ipRange)
					xlsxFile.SetCellValue(sheetName, fmt.Sprintf("B%d", row), countMatches[0])
					row++
				}
			}
		}
	}
}

// 处理漏洞与利用信息
func bugExpList(datalist []string, xlsxFile *excelize.File) {
	sheetName := "BugExpList"
	xlsxFile.NewSheet(sheetName)
	xlsxFile.SetCellValue(sheetName, "A1", "IP")
	xlsxFile.SetCellValue(sheetName, "B1", "Bug/Exploit")

	row := 2
	for _, line := range datalist {
		// 第一种格式：包含 IP 地址
		matches := regexp.MustCompile(`\[\+\] \d+\.\d+\.\d+\.\d+.*`).FindAllString(line, -1)
		for _, match := range matches {
			// 提取 IP 和漏洞信息
			ipMatch := regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`).FindString(match)
			bug := strings.Replace(match, "[+]", "", 1)
			bug = strings.Replace(bug, ipMatch, "", 1)
			bug = strings.TrimSpace(bug)

			if ipMatch != "" && bug != "" {
				xlsxFile.SetCellValue(sheetName, fmt.Sprintf("A%d", row), ipMatch)
				xlsxFile.SetCellValue(sheetName, fmt.Sprintf("B%d", row), bug)
				row++
			}
		}

		// 第二种格式：没有 IP 地址
		newMatches := regexp.MustCompile(`\[\+\] \w+-.*`).FindAllString(line, -1)
		for _, match := range newMatches {
			ipMatch := regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`).FindString(match)
			bug := strings.Replace(match, "[+]", "", 1)
			bug = strings.Replace(bug, ipMatch, "", 1)
			bug = strings.TrimSpace(bug)

			if bug != "" {
				xlsxFile.SetCellValue(sheetName, fmt.Sprintf("A%d", row), ipMatch)
				xlsxFile.SetCellValue(sheetName, fmt.Sprintf("B%d", row), bug)
				row++
			}
		}
	}
}

// 处理漏洞 POC 列表
func bugPocList(datalist []string, xlsxFile *excelize.File) {
	sheetName := "BugPocList"
	xlsxFile.NewSheet(sheetName)
	xlsxFile.SetCellValue(sheetName, "A1", "URL")
	xlsxFile.SetCellValue(sheetName, "B1", "POC 信息")

	row := 2
	for _, line := range datalist {
		// 匹配包含 poc-yaml 的行
		matches := regexp.MustCompile(`\[\+].*poc-yaml[^\s].*`).FindAllString(line, -1)
		for _, match := range matches {
			// 提取 URL
			urlMatches := regexp.MustCompile(`https?://\S+`).FindAllString(match, -1)
			// 提取 poc-yaml 信息
			pocMatches := regexp.MustCompile(`poc-yaml.*`).FindAllString(match, -1)

			if len(urlMatches) > 0 && len(pocMatches) > 0 {
				// 将 URL 和 POC 信息写入 Excel 表格
				xlsxFile.SetCellValue(sheetName, fmt.Sprintf("A%d", row), urlMatches[0])
				xlsxFile.SetCellValue(sheetName, fmt.Sprintf("B%d", row), pocMatches[0])
				row++
			}
		}
	}
}

// 处理操作系统信息
func osList(datalist []string, xlsxFile *excelize.File) {
	sheetName := "OSList"
	xlsxFile.NewSheet(sheetName)
	xlsxFile.SetCellValue(sheetName, "A1", "IP")
	xlsxFile.SetCellValue(sheetName, "B1", "操作系统信息")

	replaceList := []string{"[*]", "\t", "\x01", "\x02"}

	row := 2
	for _, line := range datalist {
		// 匹配包含 IP 地址的行
		matches := regexp.MustCompile(`\[\*]\s\d+\.\d+\.\d+\.\d+.*`).FindAllString(line, -1)
		for _, match := range matches {
			// 提取 IP 地址
			ipMatch := regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`).FindString(match)

			// 删除无用字符
			cleanedMatch := match
			for _, replace := range replaceList {
				cleanedMatch = strings.ReplaceAll(cleanedMatch, replace, "")
			}

			// 去掉 IP 地址，获取操作系统信息
			osInfo := strings.Replace(cleanedMatch, ipMatch, "", 1)
			osInfo = strings.TrimSpace(osInfo)

			// 如果提取到的操作系统信息非空，则写入 Excel 表格
			if ipMatch != "" && osInfo != "" {
				xlsxFile.SetCellValue(sheetName, fmt.Sprintf("A%d", row), ipMatch)
				xlsxFile.SetCellValue(sheetName, fmt.Sprintf("B%d", row), osInfo)
				row++
			}
		}
	}
}

// 处理标题信息
func getTitle(datalist []string, xlsxFile *excelize.File) {
	sheetName := "Titles"
	xlsxFile.NewSheet(sheetName)
	xlsxFile.SetCellValue(sheetName, "A1", "URL")
	xlsxFile.SetCellValue(sheetName, "B1", "Code")
	xlsxFile.SetCellValue(sheetName, "C1", "Len")
	xlsxFile.SetCellValue(sheetName, "D1", "Title")

	row := 2
	for _, line := range datalist {
		// 匹配 WebTitle 信息
		matches := regexp.MustCompile(`\[\*]\sWebTitle.*`).FindAllString(line, -1)
		for _, match := range matches {
			// 提取 URL
			urlMatches := regexp.MustCompile(`http[^\s]+`).FindAllString(match, -1)
			// 提取 code
			codeMatches := regexp.MustCompile(`code:\s*([^\s]+)`).FindStringSubmatch(match)
			// 提取 len
			lenMatches := regexp.MustCompile(`len:\s*([^\s]+)`).FindStringSubmatch(match)
			// 提取 title
			titleMatches := regexp.MustCompile(`title:\s*(.*)`).FindStringSubmatch(match)

			// 如果所有字段都有提取到
			if len(urlMatches) > 0 && len(codeMatches) > 0 && len(lenMatches) > 0 && len(titleMatches) > 0 {
				// 将提取到的信息写入 Excel 表格
				xlsxFile.SetCellValue(sheetName, fmt.Sprintf("A%d", row), urlMatches[0])
				xlsxFile.SetCellValue(sheetName, fmt.Sprintf("B%d", row), codeMatches[1])  // 使用索引 1 获取捕获的代码
				xlsxFile.SetCellValue(sheetName, fmt.Sprintf("C%d", row), lenMatches[1])   // 使用索引 1 获取捕获的长度
				xlsxFile.SetCellValue(sheetName, fmt.Sprintf("D%d", row), titleMatches[1]) // 使用索引 1 获取捕获的标题
				row++
			}
		}
	}
}

// 提取弱口令信息
func getPassword(datalist []string, xlsxFile *excelize.File) {
	sheetName := "Passwords"
	xlsxFile.NewSheet(sheetName)
	xlsxFile.SetCellValue(sheetName, "A1", "IP")
	xlsxFile.SetCellValue(sheetName, "B1", "Port")
	xlsxFile.SetCellValue(sheetName, "C1", "Server")
	xlsxFile.SetCellValue(sheetName, "D1", "User&Passwd")

	row := 2
	seen := make(map[string]bool) // 用于记录已写入的行

	// 定义正则表达式
	weakPassRegex := regexp.MustCompile(`(?i)(oracle|mysql|mssql|SMB|RDP|Postgres|SSH|redis|mongodb|memcached):(\d+\.\d+\.\d+\.\d+):(\d+):(.*)`)

	for _, line := range datalist {
		if match := weakPassRegex.FindStringSubmatch(line); match != nil {
			// 提取各字段
			server := match[1]                         // 服务名称
			ip := match[2]                             // IP 地址
			port := match[3]                           // 端口号
			userAndPass := strings.TrimSpace(match[4]) // 用户和密码

			// 如果没有用户和密码，默认填充为 "unauthorized"
			if userAndPass == "" || strings.HasPrefix(userAndPass, ":") {
				userAndPass = "unauthorized"
			}

			// 构造唯一键，用于去重
			key := fmt.Sprintf("%s:%s:%s:%s", ip, port, server, userAndPass)
			if !seen[key] {
				// 写入 Excel
				xlsxFile.SetCellValue(sheetName, fmt.Sprintf("A%d", row), ip)
				xlsxFile.SetCellValue(sheetName, fmt.Sprintf("B%d", row), port)
				xlsxFile.SetCellValue(sheetName, fmt.Sprintf("C%d", row), server)
				xlsxFile.SetCellValue(sheetName, fmt.Sprintf("D%d", row), userAndPass)

				// 标记此行已处理
				seen[key] = true
				row++
			}
		}
	}
}

// 提取指纹信息
func fingerOut(datastr string, xlsxFile *excelize.File) {
	sheetName := "Fingerprints"
	xlsxFile.NewSheet(sheetName)
	xlsxFile.SetCellValue(sheetName, "A1", "URL")
	xlsxFile.SetCellValue(sheetName, "B1", "指纹信息")

	row := 2
	lines := strings.Split(datastr, "\n")
	for _, line := range lines {
		// 如果包含 "InfoScan"
		if strings.Contains(line, "InfoScan") {
			// 提取 URL 和指纹信息
			urlMatch := regexp.MustCompile(`http[^\s]+`).FindString(line)
			if urlMatch != "" {
				// 提取指纹信息
				fingerPrint := strings.SplitN(line, urlMatch, 2)
				if len(fingerPrint) > 1 {
					fingerPrint[1] = strings.TrimSpace(fingerPrint[1]) // 去除多余的空格
					// 输出到 Excel
					xlsxFile.SetCellValue(sheetName, fmt.Sprintf("A%d", row), urlMatch)
					xlsxFile.SetCellValue(sheetName, fmt.Sprintf("B%d", row), fingerPrint[1])
					row++
				}
			}
		}
	}
}

// 网络信息提取
func netInfo(datastr string, xlsxFile *excelize.File) {
	sheetName := "NetInfo"
	xlsxFile.NewSheet(sheetName)
	xlsxFile.SetCellValue(sheetName, "A1", "IP")
	xlsxFile.SetCellValue(sheetName, "B1", "网络信息")

	row := 2
	// 正则表达式用于匹配包含 NetInfo 的行
	pattern := `(.*NetInfo.*\n.*(\n.*\[->].*)+)`
	re := regexp.MustCompile(pattern)
	infoMatches := re.FindAllStringSubmatch(datastr, -1)

	// 遍历所有匹配的内容
	for _, match := range infoMatches {
		// 提取 IP 地址
		ipMatches := regexp.MustCompile(`\[\*](\d+\.\d+\.\d+\.\d+)`).FindStringSubmatch(match[0])
		if len(ipMatches) > 1 {
			ip := ipMatches[1] // 获取 IP 地址
			// 提取网络信息
			netInfoMatches := regexp.MustCompile(`(\n?.*\[->].*)+`).FindStringSubmatch(match[0])
			if len(netInfoMatches) > 1 {
				netInfo := netInfoMatches[0] // 获取网络信息
				// 将 IP 地址和网络信息写入 Excel 文件
				xlsxFile.SetCellValue(sheetName, fmt.Sprintf("A%d", row), ip)
				xlsxFile.SetCellValue(sheetName, fmt.Sprintf("B%d", row), netInfo)
				row++
			}
		}
	}
}

// 提取 NetBIOS 信息
func netBios(datalist []string, xlsxFile *excelize.File) {
	sheetName := "NetBIOS"
	xlsxFile.NewSheet(sheetName)
	xlsxFile.SetCellValue(sheetName, "A1", "IP")
	xlsxFile.SetCellValue(sheetName, "B1", "NetBIOS 信息")

	row := 2
	// 遍历所有行，查找包含 "NetBIOS" 的行
	for _, line := range datalist {
		if strings.Contains(line, "NetBios") {
			// 提取 IP 地址
			ipMatches := regexp.MustCompile(`(\d+\.\d+\.\d+\.\d+)`).FindStringSubmatch(line)
			if len(ipMatches) > 0 {
				ip := ipMatches[0] // 获取 IP 地址
				// 提取 NetBIOS 信息
				netBiosInfo := line
				// 将 IP 地址和 NetBIOS 信息写入 Excel 文件
				xlsxFile.SetCellValue(sheetName, fmt.Sprintf("A%d", row), ip)
				xlsxFile.SetCellValue(sheetName, fmt.Sprintf("B%d", row), netBiosInfo)
				row++
			}
		}
	}
}

// OSS存储桶遍历
func (i *InfoDeal) DealOssList(ossURL string) (string, error) {
	s := NewScanner(ossURL)
	excelPath, err := s.Process()
	if err != nil {
		return "", fmt.Errorf("处理失败：%v", err)
	}
	return excelPath, nil
}
