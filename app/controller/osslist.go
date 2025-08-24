package controller

import (
	"crypto/tls"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"github.com/antchfx/xmlquery"
	"github.com/tealeg/xlsx"
)

type Scanner struct {
	totalKeys map[string]bool
	keysMutex sync.Mutex
	baseURL   string
	Base
}

func NewScanner(url string) *Scanner {
	base := url
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	return &Scanner{
		totalKeys: make(map[string]bool),
		baseURL:   base,
	}
}

func (s *Scanner) Process() (string, error) {
	fmt.Printf("[*] 开始解析 URL：%s\n", s.baseURL)

	maxKeys, nextMarker, childTags, filename, err := s.getBucketInfo()
	if err != nil {
		log.Printf("[-] 获取信息失败: %v", err)
		return "", fmt.Errorf("[-] 获取信息失败: %v", err)
	}

	fmt.Println("[+] 解析 XML 数据成功")
	fmt.Printf("[o] 该存储桶默认每页显示 %d 条数据\n", maxKeys)
	if nextMarker == "" {
		fmt.Println("[-] 该存储桶不支持 Web 翻页遍历")
	} else {
		fmt.Println("[+] 该存储桶支持遍历, 正在获取文件及数量")
	}

	if len(childTags) == 0 {
		fmt.Println("[-] 该存储桶不支持遍历, 或检查 URL 是否有误")
		return "", fmt.Errorf("[-] 该存储桶不支持遍历, 或检查 URL 是否有误")
	}

	if err := s.fetchData(maxKeys, filename, childTags, "", 0); err != nil {
		log.Printf("[-] 获取数据失败: %v", err)
		return "", fmt.Errorf("[-] 获取数据失败: %v", err)
	}
	return filename, nil
}

func (s *Scanner) getBucketInfo() (int, string, []string, string, error) {
	client := s.createClient()
	resp, err := client.Get(s.baseURL)
	if err != nil {
		return 0, "", nil, "", err
	}
	defer resp.Body.Close()

	xmlData, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", nil, "", err
	}

	doc, err := xmlquery.Parse(strings.NewReader(string(xmlData)))
	if err != nil {
		return 0, "", nil, "", err
	}

	maxKeys := 1000
	if node := xmlquery.FindOne(doc, "//MaxKeys"); node != nil {
		if v, err := strconv.Atoi(node.InnerText()); err == nil {
			maxKeys = v
		}
	}

	nextMarker := ""
	if node := xmlquery.FindOne(doc, "//NextMarker"); node != nil {
		nextMarker = node.InnerText()
	}

	var tags []string
	if contents := xmlquery.Find(doc, "//Contents"); len(contents) > 0 {
		for n := contents[0].FirstChild; n != nil; n = n.NextSibling {
			if n.Type == xmlquery.ElementNode {
				tagName := n.Data
				if strings.Contains(tagName, "}") {
					parts := strings.SplitN(tagName, "}", 2)
					tagName = parts[1]
				}
				tags = append(tags, tagName)
			}
		}
	}

	filename, err := s.createCSV(tags)
	if err != nil {
		return 0, "", nil, "", err
	}

	return maxKeys, nextMarker, tags, filename, nil
}

func (s *Scanner) createCSV(headers []string) (string, error) {
	// 确保目录存在
	baseDir := s.getAppPath()
	// 创建文件子目录
	fileDir := filepath.Join(baseDir, "file")
	cleanURL := regexp.MustCompile(`[:/?]`).ReplaceAllString(strings.TrimPrefix(s.baseURL, "http://"), "_")
	filename := filepath.Join(fileDir, fmt.Sprintf("%s.csv", cleanURL))

	file, err := os.Create(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	fullHeaders := append(headers, "url", "filetype")
	return filename, writer.Write(fullHeaders)
}

func (s *Scanner) fetchData(maxKeys int, filename string, headers []string, marker string, page int) error {
	client := s.createClient()
	encodedMarker := url.QueryEscape(marker)
	url := fmt.Sprintf("%s?max-keys=%d&marker=%s", s.baseURL, maxKeys, encodedMarker)
	//fmt.Printf("[+] %s\n", url)

	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	xmlData, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	doc, err := xmlquery.Parse(strings.NewReader(string(xmlData)))
	if err != nil {
		return err
	}

	contents := xmlquery.Find(doc, "//Contents")
	nextMarker := ""
	if node := xmlquery.FindOne(doc, "//NextMarker"); node != nil {
		nextMarker = node.InnerText()
	}

	count, repeat, total, err := s.writeCSV(filename, headers, contents)
	if err != nil {
		return err
	}

	page++
	fmt.Printf("[+] 第%d页写入%d条数据,共计发现%d个文件\n", page, count, total)

	if nextMarker == "" || repeat {
		fmt.Printf("[+] 数据结果已写入文件：%s\n", filename)
		outputFile, err := s.createExcel(filename)
		if err != nil {
			fmt.Printf("[-] 生成Excel失败: %v\n", err)
		} else {
			fmt.Printf("[+] 数据分类已写入文件：%s\n", outputFile)
		}
		return nil
	}

	return s.fetchData(maxKeys, filename, headers, nextMarker, page)
}

func (s *Scanner) writeCSV(filename string, headers []string, nodes []*xmlquery.Node) (int, bool, int, error) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return 0, false, 0, err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	var newCount, repeatCount int
	for _, node := range nodes {
		keyNode := xmlquery.FindOne(node, ".//Key")
		if keyNode == nil {
			continue
		}
		key := keyNode.InnerText()

		s.keysMutex.Lock()
		if s.totalKeys[key] {
			repeatCount++
			s.keysMutex.Unlock()
			continue
		}
		s.totalKeys[key] = true
		s.keysMutex.Unlock()

		newCount++

		row := make([]string, len(headers))
		for i, h := range headers {
			if n := xmlquery.FindOne(node, fmt.Sprintf(".//%s", h)); n != nil {
				row[i] = cleanExcelValue(n.InnerText())
			}
		}

		fileType := "unknown"
		if parts := strings.Split(key, "."); len(parts) > 1 {
			if ext := parts[len(parts)-1]; regexp.MustCompile(`^[a-zA-Z0-9]+$`).MatchString(ext) {
				fileType = ext
			}
		}

		row = append(row, cleanExcelValue(s.baseURL+key), cleanExcelValue(fileType))
		if err := writer.Write(row); err != nil {
			return 0, false, 0, err
		}
	}

	total := len(s.totalKeys)
	repeat := repeatCount > 2
	return newCount, repeat, total, nil
}

func (s *Scanner) createExcel(csvFile string) (string, error) {
	file, err := os.Open(csvFile)
	if err != nil {
		return "", err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return "", err
	}

	if len(records) < 2 {
		return "", fmt.Errorf("CSV文件内容不足")
	}

	groups := make(map[string][][]string)
	headers := records[0]
	fileTypeIndex := -1
	for i, h := range headers {
		if h == "filetype" {
			fileTypeIndex = i
			break
		}
	}
	if fileTypeIndex == -1 {
		return "", fmt.Errorf("缺少filetype列")
	}

	for _, row := range records[1:] {
		if len(row) <= fileTypeIndex {
			continue
		}
		ft := row[fileTypeIndex]
		groups[ft] = append(groups[ft], row)
	}

	xf := xlsx.NewFile()
	usedNames := make(map[string]int)

	for ft, rows := range groups {
		baseName := s.sanitizeSheetName(ft)
		originalName := baseName

		for count := 1; ; count++ {
			if _, exists := usedNames[baseName]; !exists {
				usedNames[baseName] = 1
				break
			}
			suffix := fmt.Sprintf("(%d)", count)
			baseName = originalName
			if len(originalName)+len(suffix) > 31 {
				baseName = originalName[:31-len(suffix)]
			}
			baseName += suffix
		}

		sheet, err := xf.AddSheet(baseName)
		if err != nil {
			continue
		}

		headerRow := sheet.AddRow()
		for _, h := range headers {
			cell := headerRow.AddCell()
			cell.Value = cleanExcelValue(h)
		}

		for _, r := range rows {
			row := sheet.AddRow()
			for _, v := range r {
				cell := row.AddCell()
				cell.Value = cleanExcelValue(v)
			}
		}
	}

	outputFile := strings.TrimSuffix(csvFile, ".csv") + "_Type.xlsx"
	if err := xf.Save(outputFile); err != nil {
		return "", err
	}
	return outputFile, nil
}

func (s *Scanner) sanitizeSheetName(name string) string {
	reg := regexp.MustCompile(`[\\/?*[\]:()]`)
	name = reg.ReplaceAllString(name, "_")
	name = strings.TrimSpace(name)
	if name == "" {
		name = "unknown"
	}

	if len(name) > 0 {
		if unicode.IsDigit(rune(name[0])) {
			name = "S_" + name
		}
	}

	if len(name) > 31 {
		name = name[:31]
	}
	return strings.TrimRight(name, " ")
}

func cleanExcelValue(value string) string {
	value = strings.Map(func(r rune) rune {
		if r < 32 && r != '\t' && r != '\n' && r != '\r' {
			return -1
		}
		return r
	}, value)

	if len(value) > 32767 {
		return value[:32700] + "...[TRUNCATED]"
	}
	return value
}

func (s *Scanner) createClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
}
