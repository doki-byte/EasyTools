package webscan

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"time"

	"EasyTools/app/model"
	"EasyTools/app/util"

	nuclei "github.com/projectdiscovery/nuclei/v3/lib"
	"github.com/projectdiscovery/nuclei/v3/pkg/output"
	syncutil "github.com/projectdiscovery/utils/sync"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// nuclei 最大响应包大小
var maxResponseSize = 1024 * 512

// NewNucleiEngine 创建并运行 Nuclei 引擎（单线程版本）
func NewNucleiEngine(ctx, ctrlCtx context.Context, taskId string, allOptions []model.NucleiOption) {
	NewNucleiEngineWithCallback(ctx, ctrlCtx, taskId, allOptions, nil, nil)
}

// NewNucleiEngineWithCallback 创建并运行 Nuclei 引擎（单线程版本，带回调）
func NewNucleiEngineWithCallback(ctx, ctrlCtx context.Context, taskId string, allOptions []model.NucleiOption, progressCb func(completed, total int, url string), callback func(model.VulnerabilityInfo)) {
	count := len(allOptions)
	for i, o := range allOptions {
		if ctrlCtx.Err() != nil {
			sendLog(ctx, "warning", "用户退出漏洞扫描")
			return
		}
		sendLog(ctx, "info", fmt.Sprintf("漏洞扫描 %d/%d", i+1, count))
		templatesToUse := resolveTemplatesForRun(o)
		if len(templatesToUse) == 0 {
			sendLog(ctx, "warning", fmt.Sprintf("[nuclei] 模板为空，已跳过: %v", o.TemplateFile))
			return
		}
		sendLog(ctx, "info", fmt.Sprintf("[nuclei] 目标: %s, 模板: %s", o.URL, previewTemplates(templatesToUse, 5)))
		if o.SkipNucleiWithoutTags && len(o.Tags) == 0 && len(o.TemplateFile) == 0 {
			sendLog(ctx, "info", fmt.Sprintf("[nuclei] %s 没有标签，跳过扫描", o.URL))
			return
		}
		// 记录代理使用情况
		if o.Proxy != "" {
			maskedProxy := o.Proxy
			if strings.Contains(maskedProxy, "@") {
				parts := strings.Split(maskedProxy, "@")
				if len(parts) == 2 {
					authPart := parts[0]
					if strings.Contains(authPart, ":") {
						authParts := strings.Split(authPart, ":")
						if len(authParts) >= 3 {
							maskedProxy = fmt.Sprintf("%s:%s:***@%s", authParts[0], authParts[1], parts[1])
						}
					}
				}
			}
			sendLog(ctx, "info", fmt.Sprintf("[nuclei] 目标 %s 使用代理: %s", o.URL, maskedProxy))
		}
		options := NewNucleiSDKOptionsWithTemplates(o, templatesToUse, false)
		ne, err := nuclei.NewNucleiEngineCtx(context.Background(), options...)
		if err != nil {
			sendLog(ctx, "error", fmt.Sprintf("[nuclei] 初始化引擎错误: %v", err))
			return
		}
		// load targets and optionally probe non http/https targets
		sendLog(ctx, "info", fmt.Sprintf("[nuclei] 检查漏洞: %s", o.URL))
		ne.LoadTargets([]string{o.URL}, false)
		err = ne.ExecuteWithCallback(func(event *output.ResultEvent) {
			sendLog(ctx, "success", fmt.Sprintf("[%s] [%s] %s", event.TemplateID, event.Info.SeverityHolder.Severity.String(), event.Matched))
			var reference string
			if event.Info.Reference != nil && !event.Info.Reference.IsEmpty() {
				reference = strings.Join(event.Info.Reference.ToSlice(), ",")
			}
			vuln := model.VulnerabilityInfo{
				TaskId:       taskId,
				TemplateID:   event.TemplateID,
				VulnName:     event.Info.Name,
				Description:  event.Info.Description,
				Reference:    reference,
				VulnURL:      showMatched(event),
				Request:      showRequest(event),
				Response:     showResponse(event),
				ResponseTime: getResponseTime(event),
				Extract:      strings.Join(event.ExtractedResults, " | "),
				Protocol:     strings.ToUpper(event.Type),
				Severity:     strings.ToUpper(event.Info.SeverityHolder.Severity.String()),
			}
			// 发送事件到前端
			runtime.EventsEmit(ctx, "nucleiResult", vuln)
			// 如果有回调函数，也调用回调
			if callback != nil {
				callback(vuln)
			}
		})
		if err != nil {
			sendLog(ctx, "error", fmt.Sprintf("[nuclei] 执行回调错误: %v", err))
			return
		}
		defer ne.Close()
		runtime.EventsEmit(ctx, "NucleiProgressID", i+1)
		if progressCb != nil {
			progressCb(i+1, count, o.URL)
		}
	}
}

// NewThreadSafeNucleiEngine 创建并运行线程安全的 Nuclei 引擎（多线程版本）
func NewThreadSafeNucleiEngine(ctx, ctrlCtx context.Context, taskId string, allOptions []model.NucleiOption) {
	NewThreadSafeNucleiEngineWithCallback(ctx, ctrlCtx, taskId, allOptions, nil, nil)
}

// NewThreadSafeNucleiEngineWithCallback 创建并运行线程安全的 Nuclei 引擎（多线程版本，带回调）
func NewThreadSafeNucleiEngineWithCallback(ctx, ctrlCtx context.Context, taskId string, allOptions []model.NucleiOption, progressCb func(completed, total int, url string), callback func(model.VulnerabilityInfo)) {
	count := len(allOptions)
	ne, err := nuclei.NewThreadSafeNucleiEngineCtx(context.Background())
	if err != nil {
		sendLog(ctx, "error", fmt.Sprintf("[nuclei] 初始化引擎错误: %v", err))
		return
	}
	var id int32

	// 根据目标数量动态限制并发 worker，避免过多 goroutine 争抢 nuclei 内部锁
	defaultWorkerSize := 2
	workerSize := defaultWorkerSize
	if len(allOptions) < workerSize {
		workerSize = len(allOptions)
	}
	if workerSize <= 0 {
		workerSize = 1
	}

	sg, err := syncutil.New(syncutil.WithSize(workerSize))
	if err != nil {
		sendLog(ctx, "error", fmt.Sprintf("[nuclei] 初始化同步组错误: %v", err))
		return
	}
	sendLog(ctx, "info", fmt.Sprintf("[nuclei] 加载 %d 个目标进行扫描", count))
	ne.GlobalResultCallback(func(event *output.ResultEvent) {
		sendLog(ctx, "success", fmt.Sprintf("[%s] [%s] %s", event.TemplateID, event.Info.SeverityHolder.Severity.String(), event.Matched))
		var reference string
		if event.Info.Reference != nil && !event.Info.Reference.IsEmpty() {
			reference = strings.Join(event.Info.Reference.ToSlice(), ",")
		}
		vuln := model.VulnerabilityInfo{
			TaskId:       taskId,
			TemplateID:   event.TemplateID,
			VulnName:     event.Info.Name,
			Description:  event.Info.Description,
			Reference:    reference,
			VulnURL:      showMatched(event),
			Request:      showRequest(event),
			Response:     showResponse(event),
			ResponseTime: limitDecimalPlaces(event.ResponseTime),
			Extract:      strings.Join(event.ExtractedResults, " | "),
			Protocol:     strings.ToUpper(event.Type),
			Severity:     strings.ToUpper(event.Info.SeverityHolder.Severity.String()),
		}
		// 发送事件到前端
		runtime.EventsEmit(ctx, "nucleiResult", vuln)
		// 如果有回调函数，也调用回调
		if callback != nil {
			callback(vuln)
		}
	})

	// 提交扫描任务
	for _, option := range allOptions {
		if ctrlCtx.Err() != nil {
			sendLog(ctx, "warning", "用户退出漏洞扫描")
			return
		}
		sg.Add()
		templatesToUse := resolveTemplatesForRun(option)
		if len(templatesToUse) == 0 {
			sendLog(ctx, "warning", fmt.Sprintf("[nuclei] 模板为空，已跳过: %v", option.TemplateFile))
			sg.Done()
			continue
		}
		sendLog(ctx, "info", fmt.Sprintf("[nuclei] 目标: %s, 模板: %s", option.URL, previewTemplates(templatesToUse, 5)))
		// 当URL目标为WEB时，如果无指纹以及开启跳过时，则跳过该URL
		// 当URL目标为其他协议时 例如: Mysql时，不需要开启跳过，只要没有指纹就跳过
		if option.SkipNucleiWithoutTags && len(option.Tags) == 0 && len(option.TemplateFile) == 0 {
			sendLog(ctx, "info", fmt.Sprintf("[nuclei] %s 没有标签，跳过扫描", option.URL))
			sg.Done()
			continue
		}
		if !strings.HasPrefix(option.URL, "http") && len(option.Tags) == 0 && len(option.TemplateFile) == 0 {
			sendLog(ctx, "info", fmt.Sprintf("[nuclei] %s 不是 web 协议且没有标签，跳过扫描", option.URL))
			sg.Done()
			continue
		}

		// 为了避免一次性加载所有模板导致 nuclei 内部缓存锁竞争，将模板按批次拆分执行
		go func(url string, opt model.NucleiOption, templates []string) {
			defer sg.Done()
			defer func() {
				if r := recover(); r != nil {
					sendLog(ctx, "error", fmt.Sprintf("[nuclei] goroutine 中捕获到 panic: %v\n%s", r, debug.Stack()))
				}
			}()
			defer func() {
				atomic.AddInt32(&id, 1)
				done := int(id)
				runtime.EventsEmit(ctx, "NucleiProgressID", done)
				sendLog(ctx, "info", fmt.Sprintf("漏洞扫描 %d/%d", done, count))
				if progressCb != nil {
					progressCb(done, count, url)
				}
			}()

			// 按批次分片模板，逐批执行，降低单次模板加载压力
			const batchSize = 300
			templateBatches := chunkStrings(templates, batchSize)
			for batchIdx, batch := range templateBatches {
				if ctrlCtx.Err() != nil {
					sendLog(ctx, "warning", "用户退出漏洞扫描")
					return
				}

				// 线程安全模式下，使用不包含代理选项的 SDK 选项
				// 注意：现在 WithProxy 已经支持线程安全模式，所以可以直接使用
				// NewNucleiSDKOptionsWithTemplates 已经包含了代理选项
				sdkOpt := NewNucleiSDKOptionsWithTemplates(opt, batch, true)
				
				// 记录代理使用情况
				if opt.Proxy != "" {
					maskedProxy := opt.Proxy
					if strings.Contains(maskedProxy, "@") {
						parts := strings.Split(maskedProxy, "@")
						if len(parts) == 2 {
							authPart := parts[0]
							if strings.Contains(authPart, ":") {
								authParts := strings.Split(authPart, ":")
								if len(authParts) >= 3 {
									maskedProxy = fmt.Sprintf("%s:%s:***@%s", authParts[0], authParts[1], parts[1])
								}
							}
						}
					}
					sendLog(ctx, "info", fmt.Sprintf("[nuclei] 目标 %s 使用代理: %s", url, maskedProxy))
				}

				sendLog(ctx, "info", fmt.Sprintf("[nuclei] 检查漏洞: %s (模板批次 %d/%d, 本批数量 %d)", url, batchIdx+1, len(templateBatches), len(batch)))

				// load targets and optionally probe non http/https targets
				err := ne.ExecuteNucleiWithOpts([]string{url}, sdkOpt...)
				if err != nil {
					sendLog(ctx, "error", fmt.Sprintf("[nuclei] 执行回调错误: %v", err))
					// 当前目标后续批次不再继续，防止无限错误刷屏
					return
				}
			}
		}(option.URL, option, templatesToUse)
	}
	sg.Wait()
	defer ne.Close()
}

// chunkStrings 将字符串切片按指定大小拆分为多个批次
func chunkStrings(in []string, size int) [][]string {
	if size <= 0 || len(in) == 0 {
		return nil
	}
	var chunks [][]string
	for i := 0; i < len(in); i += size {
		end := i + size
		if end > len(in) {
			end = len(in)
		}
		chunks = append(chunks, in[i:end])
	}
	return chunks
}

// NewNucleiSDKOptions 创建 Nuclei SDK 选项
func NewNucleiSDKOptions(o model.NucleiOption) []nuclei.NucleiSDKOptions {
	templates := resolveTemplatesForRun(o)
	return NewNucleiSDKOptionsWithTemplates(o, templates, false)
}

// NewNucleiSDKOptionsWithTemplates 创建 Nuclei SDK 选项（支持线程安全/自定义模板）
func NewNucleiSDKOptionsWithTemplates(o model.NucleiOption, templates []string, threadSafe bool) []nuclei.NucleiSDKOptions {
	options := []nuclei.NucleiSDKOptions{
		nuclei.DisableUpdateCheck(), // -duc
	}
	// 自定义请求头
	if o.CustomHeaders != "" {
		headers := parseHeaders(o.CustomHeaders)
		options = append(options, nuclei.WithHeaders(headers))
	}
	// 判断是使用指定poc文件还是根据标签
	if len(templates) == 0 {
		// 未显式指定 POC 文件时，优先根据标签匹配；
		// 如果有标签但无匹配且未开启“无匹配跳过”，将在 resolveTemplatesForRun 中回退为全量扫描
		options = append(options, nuclei.WithTemplatesOrWorkflows(nuclei.TemplateSources{
			Templates: resolveTemplatesForRun(o),
		}))
	} else {
		options = append(options, nuclei.WithTemplatesOrWorkflows(nuclei.TemplateSources{
			Templates: templates,
		}))
	}
	// 代理选项（现在支持线程安全模式）
	if o.Proxy != "" {
		options = append(options, nuclei.WithProxy([]string{o.Proxy}, false)) // -proxy
	}
	// 网络配置（超时、重试等）- 仅非线程安全模式支持
	if !threadSafe {
		timeout := o.Timeout
		if timeout <= 0 {
			timeout = 10 // 默认10秒
		}
		retries := o.Retries
		if retries < 0 {
			retries = 1 // 默认1次
		}
		options = append(options, nuclei.WithNetworkConfig(nuclei.NetworkConfig{
			Timeout: timeout,
			Retries: retries,
		}))
		// 速率限制
		if o.RateLimit > 0 {
			options = append(options, nuclei.WithGlobalRateLimitCtx(context.Background(), o.RateLimit, time.Second))
		}
	} else {
		// 线程安全模式下，使用WithNetworkConfig（现在支持线程安全模式）
		timeout := o.Timeout
		if timeout <= 0 {
			timeout = 10
		}
		retries := o.Retries
		if retries < 0 {
			retries = 1
		}
		options = append(options, nuclei.WithNetworkConfig(nuclei.NetworkConfig{
			Timeout: timeout,
			Retries: retries,
		}))
		// 速率限制
		if o.RateLimit > 0 {
			options = append(options, nuclei.WithGlobalRateLimitCtx(context.Background(), o.RateLimit, time.Second))
		}
	}
	return options
}

// parseHeaders 解析请求头字符串
func parseHeaders(headerStr string) []string {
	var headers []string
	lines := strings.Split(headerStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			headers = append(headers, line)
		}
	}
	return headers
}

// findTagsFile 根据标签查找对应的 POC 文件
func findTagsFile(inputTags, templateDirs []string) []string {
	if len(inputTags) == 0 {
		return expandYamlFiles(templateDirs)
	}
	var fileList []string
	tagMap := GetFingerPocMap()
	for _, inputTag := range inputTags {
		for pocName, pocTags := range tagMap {
			if arrayContains(inputTag, pocTags) {
				for _, dir := range templateDirs {
					filepath := path.Join(dir, pocName+".yaml")
					if _, err := os.Stat(filepath); err == nil {
						fileList = append(fileList, filepath)
						break
					}
				}
			}
		}
	}
	return removeDuplicates(fileList)
}

// resolveTemplateFiles 将模板名补全为可访问的绝对路径，自动补 .yaml
func resolveTemplateFiles(templates, templateDirs []string) []string {
	if len(templates) == 0 {
		return templates
	}
	var resolved []string
	seen := map[string]struct{}{}

	tryAdd := func(p string) {
		if p == "" {
			return
		}
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		resolved = append(resolved, p)
	}

	for _, t := range templates {
		if strings.TrimSpace(t) == "" {
			continue
		}

		candidates := []string{}

		// 1) 原始输入（尝试绝对路径或相对路径）
		normalized := filepath.Clean(filepath.FromSlash(t))
		if abs, err := filepath.Abs(normalized); err == nil {
			normalized = abs
		}
		candidates = append(candidates, normalized)

		// 2) 补全后缀
		if !strings.HasSuffix(strings.ToLower(normalized), ".yaml") {
			candidates = append(candidates, normalized+".yaml")
		}

		// 3) 提取文件名（不含扩展名和路径），用于在模板目录中查找
		baseName := normalized
		// 移除扩展名
		if strings.HasSuffix(strings.ToLower(baseName), ".yaml") {
			baseName = strings.TrimSuffix(baseName, ".yaml")
		} else if strings.HasSuffix(strings.ToLower(baseName), ".yml") {
			baseName = strings.TrimSuffix(baseName, ".yml")
		}
		// 提取文件名部分（不含路径）
		fileName := filepath.Base(baseName)

		// 4) 在模板目录中查找
		// 对于相对路径，使用原始输入；对于绝对路径，如果不存在则使用文件名
		isAbs := filepath.IsAbs(t)
		// 检查原始输入是否存在
		originalExists := false
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				originalExists = true
				break
			}
		}

		// 在模板目录中查找
		for _, dir := range templateDirs {
			if dir == "" {
				continue
			}
			var searchPath string
			if !isAbs {
				// 相对路径：使用原始输入在模板目录中查找
				// 确保添加 .yaml 扩展名
				base := t
				if !strings.HasSuffix(strings.ToLower(base), ".yaml") && !strings.HasSuffix(strings.ToLower(base), ".yml") {
					base = base + ".yaml"
				}
				searchPath = filepath.Join(dir, base)
			} else if !originalExists {
				// 绝对路径但不存在：使用文件名在模板目录中查找
				searchPath = filepath.Join(dir, fileName+".yaml")
			} else {
				// 绝对路径且存在：跳过模板目录查找
				continue
			}
			
			joined := filepath.Clean(searchPath)
			if abs, err := filepath.Abs(joined); err == nil {
				joined = abs
			}
			candidates = append(candidates, joined)
		}

		// 选择第一个存在的
		chosen := ""
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				chosen = filepath.ToSlash(c)
				break
			}
		}

		// 如果都不存在，仍然保留原始输入，交给 nuclei 去处理（保持兼容）
		if chosen == "" {
			chosen = filepath.ToSlash(candidates[0])
		}

		tryAdd(chosen)
	}

	return resolved
}

// resolveTemplatesForRun 统一模板决策（模板文件优先，否则按标签展开）
func resolveTemplatesForRun(o model.NucleiOption) []string {
	if len(o.TemplateFile) == 0 {
		// 优先根据标签匹配模板
		tags := finalTags(o.Tags, o.CustomTags)
		// 如果没有任何标签，则直接全量扫描
		if len(tags) == 0 {
			return expandYamlFiles(o.TemplateFolders)
		}
		files := findTagsFile(tags, o.TemplateFolders)
		// 如果有标签但没有任何匹配到的模板：
		// - 勾选了“无匹配跳过”时，在上层会因为模板为空而跳过该 URL
		// - 未勾选时，这里回退为全量扫描
		if len(files) == 0 && !o.SkipNucleiWithoutTags {
			return expandYamlFiles(o.TemplateFolders)
		}
		return files
	}
	return resolveTemplateFiles(o.TemplateFile, o.TemplateFolders)
}

// previewTemplates 简要展示模板列表，避免日志过长
func previewTemplates(list []string, limit int) string {
	if len(list) <= limit {
		return strings.Join(list, ", ")
	}
	head := list[:limit]
	return fmt.Sprintf("%s ... (total %d)", strings.Join(head, ", "), len(list))
}

// buildCmdPreview 构造近似 nuclei 命令，便于排查
func buildCmdPreview(o model.NucleiOption, templates []string) string {
	var parts []string
	parts = append(parts, "nuclei")
	parts = append(parts, "-duc") // Disable update check
	if len(templates) > 0 {
		parts = append(parts, "-t")
		parts = append(parts, strings.Join(templates, ","))
	}
	if o.URL != "" {
		parts = append(parts, "-u")
		parts = append(parts, o.URL)
	}
	if o.Proxy != "" {
		parts = append(parts, "-proxy")
		parts = append(parts, o.Proxy)
	}
	if o.RateLimit > 0 {
		parts = append(parts, "-rl")
		parts = append(parts, fmt.Sprintf("%d", o.RateLimit))
	}
	if o.Timeout > 0 {
		parts = append(parts, "-timeout")
		parts = append(parts, fmt.Sprintf("%d", o.Timeout))
	}
	if o.Retries > 0 {
		parts = append(parts, "-retries")
		parts = append(parts, fmt.Sprintf("%d", o.Retries))
	}
	if o.Threads > 0 {
		parts = append(parts, "-c")
		parts = append(parts, fmt.Sprintf("%d", o.Threads))
	}
	if o.CustomHeaders != "" {
		parts = append(parts, "-H")
		parts = append(parts, fmt.Sprintf("%q", o.CustomHeaders))
	}
	return strings.Join(parts, " ")
}

// expandYamlFiles 展开 YAML 文件列表
func expandYamlFiles(dirs []string) []string {
	var files []string
	for _, dir := range dirs {
		filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() && strings.HasSuffix(path, ".yaml") {
				files = append(files, path)
			}
			return nil
		})
	}
	return files
}

// finalTags 确定最终使用的标签
func finalTags(detectTags, customTags []string) []string {
	if len(customTags) != 0 {
		return customTags
	}
	return detectTags
}

// showMatched 显示匹配的 URL
func showMatched(event *output.ResultEvent) string {
	if event.Matched != "" {
		return event.Matched
	}
	return event.URL
}

// showRequest 显示请求内容
func showRequest(event *output.ResultEvent) string {
	if event.Request != "" {
		return event.Request
	}
	if event.Interaction != nil {
		return event.Interaction.RawRequest
	}
	return ""
}

// showResponse 显示响应内容（限制大小）
func showResponse(event *output.ResultEvent) string {
	if event.Response != "" {
		return limitResponse(event.Response, maxResponseSize, "")
	}
	if event.Interaction != nil {
		return event.Interaction.RawResponse
	}
	return ""
}

// limitResponse 限制响应大小
func limitResponse(response string, maxSize int, suffix string) string {
	if len(response) <= maxSize {
		return response
	}
	return response[:maxSize] + suffix
}

// getResponseTime 获取响应时间
func getResponseTime(event *output.ResultEvent) string {
	// ResponseTime 字段在 ResultEvent 结构体中存在
	return limitDecimalPlaces(event.ResponseTime)
}

// limitDecimalPlaces 限制小数为2位，用于截取时间字符串
func limitDecimalPlaces(value string) string {
	if value == "" {
		return ""
	}
	parts := strings.Split(value, ".")
	if len(parts) == 2 && len(parts[1]) > 2 {
		value = parts[0] + "." + parts[1][:2] // 截取小数点后两位
	}
	return value
}

// sendLog 发送日志
func sendLog(ctx context.Context, level, message string) {
	if ctx == nil {
		return
	}
	util.NewUtil().Log(fmt.Sprintf("[漏洞扫描][%s] %s", strings.ToUpper(level), message))
	runtime.EventsEmit(ctx, "log_event", map[string]interface{}{
		"level":   level,
		"message": message,
		"source":  "漏洞扫描",
	})
}

// arrayContains 检查数组中是否包含元素
func arrayContains(item string, arr []string) bool {
	item = strings.ToLower(strings.TrimSpace(item))
	if item == "" {
		return false
	}
	for _, v := range arr {
		tag := strings.ToLower(strings.TrimSpace(v))
		if tag == "" {
			continue
		}
		// 支持模糊匹配：完全相等、包含关系均视为匹配
		if tag == item || strings.Contains(tag, item) || strings.Contains(item, tag) {
			return true
		}
	}
	return false
}

// removeDuplicates 去除重复元素
func removeDuplicates(arr []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, item := range arr {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}
