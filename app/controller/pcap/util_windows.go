//go:build windows

package pcap

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

// CheckTshark 检查tshark环境
func (a *PcapAnalyzer) CheckTshark() string {
	log.Println("=== 开始检查tshark环境 ===")

	// 查找tshark可执行文件
	tsharkPath, err := findTshark()
	if err != nil {
		errorMsg := "❌ 未找到tshark，请安装Wireshark并确保tshark在PATH环境变量中"
		log.Println(errorMsg)
		return errorMsg
	}

	log.Printf("✅ 找到tshark: %s", tsharkPath)

	// 检查版本
	cmd := exec.Command(tsharkPath, "--version")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	output, err := cmd.Output()

	if err != nil {
		errorMsg := fmt.Sprintf("❌ 检查tshark版本时出错: %v", err)
		log.Println(errorMsg)
		return errorMsg
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		log.Printf("✅ %s", lines[0])
	}

	// 检查基本功能
	cmd = exec.Command(tsharkPath, "-h")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	if err := cmd.Run(); err != nil {
		errorMsg := fmt.Sprintf("❌ tshark基本功能检查失败: %v", err)
		log.Println(errorMsg)
		return errorMsg
	}

	successMsg := "✅ tshark环境检查通过 - " + strings.TrimSpace(lines[0])
	log.Println(successMsg)
	return successMsg
}

// extractFilesFromPcap 从PCAP提取文件
func (a *PcapAnalyzer) extractFilesFromPcap(ctx context.Context, filePath string, config AnalysisConfig) ([]ExtractionResult, []AnalysisResult, error) {
	tsharkPath, err := findTshark()
	if err != nil {
		return nil, nil, err
	}

	outputDir := filepath.Join(filepath.Dir(filePath), "extracted_files_"+time.Now().Format("20060102_150405"))
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		a.sendLogEvent("error", fmt.Sprintf("创建输出目录失败: %v", err))
		return nil, nil, fmt.Errorf("创建输出目录失败: %v", err)
	}
	a.sendLogEvent("info", fmt.Sprintf("📂 输出目录: %s", outputDir))

	var extractedFiles []ExtractionResult
	var fileContentResults []AnalysisResult

	// 预编译正则表达式
	regexConfig, err := a.loadConfig()
	if err != nil {
		a.sendLogEvent("warning", fmt.Sprintf("加载正则配置失败，使用默认配置: %v", err))
		regexConfig = defaultRegexConfig
	}

	var flagRe, sqlRe, credentialRe, keyRe *regexp.Regexp
	var regexErr error

	if config.DetectFlags && regexConfig.FlagRegex != "" {
		flagRe, regexErr = regexp.Compile(regexConfig.FlagRegex)
		if regexErr != nil {
			a.sendLogEvent("warning", fmt.Sprintf("Flag正则表达式编译失败: %v", regexErr))
		}
	}

	if config.DetectSQL && regexConfig.SqlRegex != "" {
		sqlRe, regexErr = regexp.Compile(regexConfig.SqlRegex)
		if regexErr != nil {
			a.sendLogEvent("warning", fmt.Sprintf("SQL正则表达式编译失败: %v", regexErr))
		}
	}

	if config.DetectCredentials && regexConfig.CredentialRegex != "" {
		credentialRe, regexErr = regexp.Compile(regexConfig.CredentialRegex)
		if regexErr != nil {
			a.sendLogEvent("warning", fmt.Sprintf("凭据正则表达式编译失败: %v", regexErr))
		}
	}

	if config.DetectKeys && regexConfig.KeyRegex != "" {
		keyRe, regexErr = regexp.Compile(regexConfig.KeyRegex)
		if regexErr != nil {
			a.sendLogEvent("warning", fmt.Sprintf("密钥正则表达式编译失败: %v", regexErr))
		}
	}

	protocols := []string{"http", "dicom", "smb", "tftp"}

	for _, protocol := range protocols {
		select {
		case <-ctx.Done():
			a.sendLogEvent("info", "文件提取被用户取消")
			return extractedFiles, fileContentResults, nil
		default:
			protocolDir := filepath.Join(outputDir, protocol)
			if err := os.MkdirAll(protocolDir, 0755); err != nil {
				logMsg := fmt.Sprintf("⚠️ 创建%s协议目录失败: %v", protocol, err)
				a.sendLogEvent("warning", logMsg)
				continue
			}

			a.sendLogEvent("info", fmt.Sprintf("开始提取%s协议的文件...", protocol))

			cmd := exec.CommandContext(ctx, tsharkPath,
				"-r", filePath,
				"--export-objects", fmt.Sprintf("%s,%s", protocol, protocolDir))
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

			if err := cmd.Run(); err != nil {
				errorMsg := fmt.Sprintf("⚠️ 导出%s对象时出错: %v", protocol, err)
				a.sendLogEvent("warning", errorMsg)
				continue
			}

			files, err := os.ReadDir(protocolDir)
			if err != nil {
				logMsg := fmt.Sprintf("⚠️ 读取%s目录失败: %v", protocol, err)
				a.sendLogEvent("warning", logMsg)
				continue
			}

			fileCount := 0
			for _, file := range files {
				select {
				case <-ctx.Done():
					a.sendLogEvent("info", "文件提取被用户取消")
					return extractedFiles, fileContentResults, nil
				default:
					fileInfo, err := file.Info()
					if err != nil {
						continue
					}

					if fileInfo.Size() == 0 || fileInfo.IsDir() {
						continue
					}

					fullPath := filepath.Join(protocolDir, file.Name())

					content, err := os.ReadFile(fullPath)
					if err != nil {
						continue
					}

					fileType := a.detectFileTypeByContent(content)
					contentType := a.detectContentTypeByContent(content)
					md5Hash := calculateMD5(content)

					result := ExtractionResult{
						FileName:    file.Name(),
						FilePath:    fullPath,
						FileType:    fileType,
						Size:        fileInfo.Size(),
						Protocol:    protocol,
						ContentType: contentType,
						MD5:         md5Hash,
					}

					extractedFiles = append(extractedFiles, result)
					fileCount++

					// 对文件内容进行正则匹配
					fileMatches := a.analyzeFileContent(content, file.Name(), protocol, config, flagRe, sqlRe, credentialRe, keyRe)
					fileContentResults = append(fileContentResults, fileMatches...)

					// 实时发送文件事件
					a.sendFileEvent(result)
					a.sendLogEvent("file", fmt.Sprintf("✅ 提取文件: %s, 协议: %s, 类型: %s, 大小: %s",
						file.Name(), protocol, fileType, formatFileSize(fileInfo.Size())))
				}
			}

			if fileCount > 0 {
				a.sendLogEvent("success", fmt.Sprintf("✅ %s协议提取完成，共提取 %d 个文件", protocol, fileCount))
			}
		}
	}

	a.sendLogEvent("success", fmt.Sprintf("文件提取完成，共提取 %d 个文件，文件内容匹配 %d 个结果", len(extractedFiles), len(fileContentResults)))
	return extractedFiles, fileContentResults, nil
}

// 修改 performAnalysis 方法，添加实时事件发送
func (a *PcapAnalyzer) performAnalysis(ctx context.Context, config AnalysisConfig) ([]AnalysisResult, error) {
	tsharkPath, err := findTshark()
	if err != nil {
		a.sendLogEvent("error", fmt.Sprintf("找不到tshark: %v", err))
		return nil, fmt.Errorf("找不到tshark: %v", err)
	}

	// 构建tshark命令
	cmd := exec.CommandContext(ctx, tsharkPath,
		"-r", config.FilePath,
		"-T", "fields",
		"-e", "frame.number",
		"-e", "frame.time",
		"-e", "ip.src",
		"-e", "ip.dst",
		"-e", "tcp.payload",
		"-e", "udp.payload",
		"-e", "http.request.uri",
		"-e", "http.file_data",
		"-E", "separator=|\n",
		"-E", "occurrence=f")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		a.sendLogEvent("error", fmt.Sprintf("创建输出管道失败: %v", err))
		return nil, fmt.Errorf("创建输出管道失败: %v", err)
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	if err := cmd.Start(); err != nil {
		a.sendLogEvent("error", fmt.Sprintf("启动tshark失败: %v", err))
		return nil, fmt.Errorf("启动tshark失败: %v", err)
	}

	var results []AnalysisResult
	reader := bufio.NewReader(stdout)

	packetNum := 0
	packetStartTime := time.Now()
	lastProgressTime := time.Now()

	a.sendLogEvent("info", "🔍 开始解析数据包...")

	// 使用通道来监控命令结束
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	// 状态更新goroutine
	statusTicker := time.NewTicker(2 * time.Second)
	defer statusTicker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-statusTicker.C:
				a.sendStatusEvent(packetNum, len(results))
			}
		}
	}()

	// 主处理循环
	processing := true
	for processing {
		select {
		case <-ctx.Done():
			cmd.Process.Kill()
			<-done
			a.sendLogEvent("info", "分析被用户取消")
			return results, nil
		case err := <-done:
			if err != nil {
				a.sendLogEvent("warning", fmt.Sprintf("tshark命令执行出错: %v", err))
			}
			processing = false
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				if strings.Contains(err.Error(), "token too long") {
					continue
				}
				continue
			}

			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			packetNum++

			fields := strings.Split(line, "|")
			if len(fields) < 3 {
				continue
			}

			packetResults := a.analyzePacketFields(fields, packetNum, config)
			if len(packetResults) > 0 {
				results = append(results, packetResults...)

				// 实时发送匹配结果
				for _, result := range packetResults {
					logMessage := fmt.Sprintf("🎯 匹配到 %s: %s (数据包 #%d)",
						a.getResultTypeLabel(result.Type),
						truncateString(result.Content, 50),
						result.PacketNum)
					a.sendLogEvent("match", logMessage)
					a.sendMatchEvent(result)
				}
			}

			// 每100个数据包或每3秒输出一次进度
			if packetNum%100 == 0 || time.Since(lastProgressTime) > 3*time.Second {
				elapsed := time.Since(packetStartTime)
				progressMsg := fmt.Sprintf("📊 已处理 %d 个数据包, 找到 %d 个匹配项, 耗时: %v",
					packetNum, len(results), elapsed)
				a.sendLogEvent("progress", progressMsg)
				lastProgressTime = time.Now()
			}
		}
	}

	// 停止状态更新
	statusTicker.Stop()

	completionMsg := fmt.Sprintf("✅ 数据包解析完成，共处理 %d 个数据包，找到 %d 个匹配项", packetNum, len(results))
	a.sendLogEvent("success", completionMsg)
	return results, nil
}
