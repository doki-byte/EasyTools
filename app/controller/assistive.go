package controller

import (
	"EasyTools/app/controller/system"
	"bufio"
	"fmt"
	"github.com/gogs/chardet"
	"github.com/xuri/excelize/v2"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Antivirus 控制器
type Assistive struct {
	system.Base
}

// AntivirusItem 杀软表结构体
type AntivirusItem struct {
	ID          int    `json:"id"`
	ProcessName string `json:"process_name"` // 对应 tasklist 的进程名
	Description string `json:"description"`  // 杀软名称
}

// GoogleQuery 结构体
type GoogleQuery struct {
	ID          int    `json:"id"`
	Category    string `json:"category"`    // 查询分类
	Description string `json:"description"` // 说明
	Command     string `json:"command"`     // 查询命令
}

// PasswordData 密码数据结构体
type PasswordData struct {
	ID       int    `json:"id" gorm:"column:id"`             // 映射数据库 id
	Name     string `json:"name" gorm:"column:name"`         // 映射数据库 name
	Method   string `json:"method" gorm:"column:method"`     // 映射数据库 method
	UserID   string `json:"userId" gorm:"column:userId"`     // 映射数据库 userId
	Password string `json:"password" gorm:"column:password"` // 映射数据库 password
	Level    string `json:"level" gorm:"column:level"`       // 映射数据库 level
}

// TableName 指定表名
func (PasswordData) TableName() string {
	return "password_data"
}

// TableName 指定表名
func (AntivirusItem) TableName() string {
	return "antivirus_list"
}

// TableName 指定表名
func (GoogleQuery) TableName() string {
	return "google_query"
}

// NewAntivirus 创建新的 Antivirus 控制器
func NewAssistive() *Assistive {
	return &Assistive{}
}

// QueryAntivirusProcesses 根据用户输入查询杀软进程（去重优化版）
func (a *Assistive) QueryAntivirusProcesses(tasklistOutput string) ([]map[string]string, error) {
	db := a.Db()
	if db == nil {
		return nil, fmt.Errorf("数据库连接未初始化")
	}

	// 步骤1: 提取并去重用户进程
	seen := make(map[string]struct{})
	var uniqueProcesses []string
	lines := strings.Split(tasklistOutput, "\n")

	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			// 提取第一个非空字段作为进程名
			if firstSpace := strings.Index(trimmed, " "); firstSpace > 0 {
				processName := trimmed[:firstSpace]
				lowerName := strings.ToLower(processName)

				// 去重检查
				if _, exists := seen[lowerName]; !exists {
					seen[lowerName] = struct{}{}
					uniqueProcesses = append(uniqueProcesses, processName)
				}
			}
		}
	}

	if len(uniqueProcesses) == 0 {
		return nil, fmt.Errorf("未能从输入中提取任何程序名")
	}

	// 步骤2: 查询数据库中的杀软进程列表
	var antivirusList []AntivirusItem
	if err := db.Find(&antivirusList).Error; err != nil {
		return nil, fmt.Errorf("查询数据库失败: %v", err)
	}

	// 构建快速查找映射（小写进程名->描述）
	avMap := make(map[string]string)
	for _, av := range antivirusList {
		key := strings.ToLower(av.ProcessName)
		avMap[key] = av.Description
	}

	// 步骤3: 执行匹配（O(n)复杂度）
	var results []map[string]string
	for _, proc := range uniqueProcesses {
		lowerProc := strings.ToLower(proc)
		if desc, exists := avMap[lowerProc]; exists {
			results = append(results, map[string]string{
				"program":     proc,
				"match":       lowerProc,
				"description": desc,
			})
		}
	}

	return results, nil
}

// QueryGoogleQueries 根据域名生成查询语法
func (a *Assistive) QueryGoogleQueries(googleDomain string) ([]map[string]interface{}, error) {
	// 定义返回结果
	var results []map[string]interface{}

	// 获取数据库连接
	db := a.Db()
	if db == nil {
		return nil, fmt.Errorf("数据库连接未初始化")
	}

	// 查询数据库中的 Google 查询列表
	var googleQueries []GoogleQuery
	err := db.Find(&googleQueries).Error
	if err != nil {
		return nil, fmt.Errorf("查询数据库失败: %v", err)
	}

	// 遍历查询结果并替换命令中的 ${googleDomain}
	for _, query := range googleQueries {
		// 替换命令中的 ${googleDomain}
		modifiedCommand := strings.ReplaceAll(query.Command, "${googleDomain}", googleDomain)

		// 将修改后的命令添加到返回结果中
		result := map[string]interface{}{
			"category":    query.Category,
			"description": query.Description,
			"command":     modifiedCommand,
		}
		results = append(results, result)
	}

	return results, nil
}

// QueryPasswords 分页查询全部数据
func (a *Assistive) QueryPasswords(page, pageSize int) ([]PasswordData, int64, error) {
	// 获取数据库连接
	db := a.Db()
	if db == nil {
		return nil, 0, fmt.Errorf("数据库连接未初始化")
	}

	var passwordDataList []PasswordData
	var total int64

	// 执行查询，获取所有数据，不带查询条件
	err := db.Model(&PasswordData{}).
		Count(&total). // 获取总数据数
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&passwordDataList).Error

	if err != nil {
		return nil, 0, fmt.Errorf("查询数据库失败: %v", err)
	}

	return passwordDataList, total, nil
}

// QueryPasswordsWithQuery 分页查询带查询条件的数据
func (a *Assistive) QueryPasswordsWithQuery(page, pageSize int, query string) ([]PasswordData, int64, error) {
	// 获取数据库连接
	db := a.Db()
	if db == nil {
		return nil, 0, fmt.Errorf("数据库连接未初始化")
	}

	var passwordDataList []PasswordData
	var total int64

	// 构造查询条件
	queryString := "%" + query + "%"

	// 执行查询，带上查询条件
	err := db.Model(&PasswordData{}).
		//Where("name LIKE ? OR method LIKE ? OR userId LIKE ? OR password LIKE ? OR level LIKE ?", queryString, queryString, queryString, queryString, queryString).
		Where("name LIKE ? ", queryString).
		Count(&total). // 获取总数据数
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&passwordDataList).Error

	if err != nil {
		return nil, 0, fmt.Errorf("查询数据库失败: %v", err)
	}

	return passwordDataList, total, nil
}

// QueryPasswordsAPI 分页查询接口
func (a *Assistive) QueryPasswordsAPI(page, pageSize int, query string) (map[string]interface{}, error) {
	var data []PasswordData
	var total int64
	var err error

	// 如果没有查询条件，执行默认查询
	if query == "" {
		data, total, err = a.QueryPasswords(page, pageSize)
	} else {
		// 如果有查询条件，执行带条件的查询
		data, total, err = a.QueryPasswordsWithQuery(page, pageSize, query)
	}

	if err != nil {
		return nil, fmt.Errorf("查询失败: %v", err)
	}

	// 如果数据为空，返回一个空数组而不是 nil
	if data == nil {
		data = []PasswordData{}
	}

	// 返回查询结果和总数
	return map[string]interface{}{
		"data":  data,
		"total": total,
	}, nil
}

// 上传文件
func (a *Assistive) UploadFile(fileName, content string) error {
	// 确保目录存在
	baseDir := a.GetAppPath()
	// 创建文件子目录
	fileDir := filepath.Join(baseDir, "file")

	if err := os.MkdirAll(fileDir, os.ModePerm); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}

	// 保存文件
	filePath := filepath.Join(fileDir, fileName)
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("保存文件失败: %v", err)
	}
	return nil
}

func (a *Assistive) FscanTextDeal(content string) (string, error) {
	// 确保目录存在
	baseDir := a.GetAppPath()
	fileDir := filepath.Join(baseDir, "file")

	if err := os.MkdirAll(fileDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("创建目录失败: %v", err)
	}

	xlsxFile := excelize.NewFile()

	// 解析文本内容
	lines := strings.Split(content, "\n")

	// 调用资产解析函数
	parseAssets(lines, xlsxFile)

	// 删除默认 sheet
	if len(xlsxFile.GetSheetList()) > 0 {
		defaultSheet := xlsxFile.GetSheetList()[0]
		xlsxFile.DeleteSheet(defaultSheet)
	}

	// 保存处理后的 Excel 文件
	saveFileName := fmt.Sprintf("assets_%s.xlsx", time.Now().Format("2006-01-02_15-04-05"))
	saveFilePath := filepath.Join(fileDir, saveFileName)
	if err := xlsxFile.SaveAs(saveFilePath); err != nil {
		return "", fmt.Errorf("保存文件失败: %v", err)
	}

	return saveFilePath, nil
}

// 解析资产信息
func parseAssets(lines []string, xlsxFile *excelize.File) {
	assets := map[string][]string{
		"URL地址": {},
		"IP地址":  {},
		"IP端口":  {},
		"地址段":   {},
		"主域名":   {},
		"子域名":   {},
		"其他信息":  {},
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		assetType, value := classifyAsset(line)
		assets[assetType] = append(assets[assetType], value)
	}

	// 创建各个sheet并写入数据
	for sheetName, data := range assets {
		if len(data) > 0 {
			xlsxFile.NewSheet(sheetName)
			xlsxFile.SetCellValue(sheetName, "A1", "资产信息")

			for i, item := range data {
				xlsxFile.SetCellValue(sheetName, fmt.Sprintf("A%d", i+2), item)
			}

			// 设置列宽
			xlsxFile.SetColWidth(sheetName, "A", "A", 50)
		}
	}
}

// 资产分类函数
func classifyAsset(text string) (string, string) {
	// URL解析
	urlRegex := regexp.MustCompile(`^(https?|ftp)://[^\s/$.?#].[^\s]*$`)
	if urlRegex.MatchString(text) {
		return "URL地址", text
	}

	// IP:端口解析
	ipPortRegex := regexp.MustCompile(`^(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}):(\d+)$`)
	if ipPortRegex.MatchString(text) {
		return "IP端口", text
	}

	// IP地址解析
	ipRegex := regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)
	if ipRegex.MatchString(text) {
		return "IP地址", text
	}

	// 地址段解析 (CIDR)
	cidrRegex := regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\/\d{1,2}$`)
	if cidrRegex.MatchString(text) {
		return "地址段", text
	}

	// 域名解析
	domainRegex := regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)
	if domainRegex.MatchString(text) {
		parts := strings.Split(text, ".")
		if len(parts) == 2 {
			return "主域名", text
		} else {
			return "子域名", text
		}
	}

	// 其他信息
	return "其他信息", text
}

// FscanResultDeal 处理文件并生成 Excel
func (a *Assistive) FscanResultDeal(fileName string) (string, error) {
	// 确保目录存在
	baseDir := a.GetAppPath()
	// 创建文件子目录
	fileDir := filepath.Join(baseDir, "file")
	// 构造文件路径
	filePath := filepath.Join(fileDir, fileName)

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
	saveFilePath := filepath.Join(fileDir, saveFileName)
	if err := xlsxFile.SaveAs(saveFilePath); err != nil {
		return "", fmt.Errorf("保存文件失败: %v", err)
	}

	return saveFilePath, nil
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
