package domainCollection

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tealeg/xlsx"
)

// DomainCollectionController 域名收集控制器（导出给前端）
type DomainCollectionController struct {
	*DomainCollection
	ctx context.Context
}

// NewDomainCollectionController 创建域名收集控制器
func NewDomainCollectionController() *DomainCollectionController {
	return &DomainCollectionController{
		DomainCollection: NewDomainCollection(),
	}
}

// Startup 启动时设置上下文
func (dcc *DomainCollectionController) Startup(ctx context.Context) {
	dcc.ctx = ctx
	dcc.DomainCollection.Startup(ctx)
}

// CreateCollectionTask 创建收集任务
func (dcc *DomainCollectionController) CreateCollectionTask(name, rootDomain, mode string, configJSON string) (string, error) {
	return dcc.DomainCollection.CreateTask(name, rootDomain, mode, configJSON)
}

// StartCollection 开始收集
func (dcc *DomainCollectionController) StartCollection(taskID string) error {
	return dcc.DomainCollection.StartCollection(taskID)
}

// StopCollection 停止收集
func (dcc *DomainCollectionController) StopCollection(taskID string) error {
	return dcc.DomainCollection.StopCollection(taskID)
}

// GetTask 获取任务信息
func (dcc *DomainCollectionController) GetTask(taskID string) (string, error) {
	task, err := dcc.DomainCollection.GetTask(taskID)
	if err != nil {
		return "", err
	}

	jsonData, err := json.Marshal(task)
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}

// GetAllTasks 获取所有任务（包括内存中的和已保存的）
func (dcc *DomainCollectionController) GetAllTasks() (string, error) {
	dcc.tasksMutex.RLock()

	// 先获取内存中的任务
	memoryTasks := make(map[string]*CollectionTask)
	for _, task := range dcc.tasks {
		taskCopy := *task
		memoryTasks[task.ID] = &taskCopy
	}

	dcc.tasksMutex.RUnlock()

	// 获取已保存的任务
	savedTasks, err := dcc.DomainCollection.GetAllSavedTasks()
	if err != nil {
		// 如果获取已保存任务失败，只返回内存中的任务
		tasks := []*CollectionTask{}
		for _, task := range memoryTasks {
			tasks = append(tasks, task)
		}
		jsonData, err := json.Marshal(tasks)
		if err != nil {
			return "", err
		}
		return string(jsonData), nil
	}

	// 合并内存任务和已保存任务（内存中的任务优先）
	allTasks := make(map[string]*CollectionTask)

	// 先添加已保存的任务
	for _, task := range savedTasks {
		allTasks[task.ID] = task
	}

	// 然后添加内存中的任务（覆盖已保存的任务）
	for _, task := range memoryTasks {
		allTasks[task.ID] = task
	}

	// 转换为列表
	tasks := []*CollectionTask{}
	for _, task := range allTasks {
		tasks = append(tasks, task)
	}

	jsonData, err := json.Marshal(tasks)
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}

// DeleteTask 删除任务（包括内存中的任务和已保存的任务文件）
func (dcc *DomainCollectionController) DeleteTask(taskID string) error {
	// 先删除业务层任务及其文件
	if err := dcc.DomainCollection.DeleteTask(taskID); err != nil {
		return err
	}

	// 再清理控制器内部缓存（如果存在）
	dcc.tasksMutex.Lock()
	delete(dcc.tasks, taskID)
	dcc.tasksMutex.Unlock()

	return nil
}

// GetDictList 获取字典文件列表
func (dcc *DomainCollectionController) GetDictList() (string, error) {
	dictFiles, err := dcc.DomainCollection.GetDictList()
	if err != nil {
		return "", err
	}

	var dictList []map[string]interface{}
	for _, filePath := range dictFiles {
		info, err := os.Stat(filePath)
		if err != nil {
			continue
		}

		dictList = append(dictList, map[string]interface{}{
			"name": info.Name(),
			"path": filePath,
			"size": info.Size(),
		})
	}

	jsonData, err := json.Marshal(dictList)
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}

// LoadDictFile 加载字典文件内容
func (dcc *DomainCollectionController) LoadDictFile(filePath string) (string, error) {
	words, err := dcc.DomainCollection.LoadDictFile(filePath)
	if err != nil {
		return "", err
	}

	jsonData, err := json.Marshal(words)
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}

// ExportResults 导出结果到Excel
func (dcc *DomainCollectionController) ExportResults(taskID string) (string, error) {
	task, err := dcc.DomainCollection.GetTask(taskID)
	if err != nil {
		return "", err
	}

	if task.Results == nil {
		return "", fmt.Errorf("任务结果为空")
	}

	// 创建Excel文件
	file := xlsx.NewFile()

	// 有效域名 sheet
	sheet, _ := file.AddSheet("有效域名")
	headerRow := sheet.AddRow()
	headerRow.AddCell().Value = "域名"
	headerRow.AddCell().Value = "IP地址"
	headerRow.AddCell().Value = "状态"
	headerRow.AddCell().Value = "HTTP状态码"
	headerRow.AddCell().Value = "HTTP标题"
	headerRow.AddCell().Value = "发现时间"

	for _, domainInfo := range task.Results.ValidDomains {
		row := sheet.AddRow()
		row.AddCell().Value = domainInfo.Domain
		row.AddCell().Value = fmt.Sprintf("%v", domainInfo.IPs)
		row.AddCell().Value = domainInfo.Status
		row.AddCell().Value = fmt.Sprintf("%d", domainInfo.HTTPStatus)
		row.AddCell().Value = domainInfo.HTTPTitle
		row.AddCell().Value = domainInfo.FoundTime
	}

	// 无效域名 sheet
	if len(task.Results.InvalidDomains) > 0 {
		sheet, _ = file.AddSheet("无效域名")
		headerRow = sheet.AddRow()
		headerRow.AddCell().Value = "域名"

		for _, domain := range task.Results.InvalidDomains {
			row := sheet.AddRow()
			row.AddCell().Value = domain
		}
	}

	// 统计信息 sheet
	sheet, _ = file.AddSheet("统计信息")
	headerRow = sheet.AddRow()
	headerRow.AddCell().Value = "项目"
	headerRow.AddCell().Value = "数值"

	stats := [][]string{
		{"根域名", task.RootDomain},
		{"收集模式", task.Mode},
		{"总生成数", fmt.Sprintf("%d", task.Results.TotalGenerated)},
		{"有效域名数", fmt.Sprintf("%d", len(task.Results.ValidDomains))},
		{"无效域名数", fmt.Sprintf("%d", len(task.Results.InvalidDomains))},
		{"开始时间", task.StartTime.Format("2006-01-02 15:04:05")},
	}

	if task.EndTime != nil {
		stats = append(stats, []string{"结束时间", task.EndTime.Format("2006-01-02 15:04:05")})
		duration := task.EndTime.Sub(task.StartTime)
		stats = append(stats, []string{"耗时", duration.String()})
	}

	for _, stat := range stats {
		row := sheet.AddRow()
		row.AddCell().Value = stat[0]
		row.AddCell().Value = stat[1]
	}

	// 保存文件
	outputDir := filepath.Join("EasyToolsFiles", "file")
	_ = os.MkdirAll(outputDir, 0755)
	filename := filepath.Join(outputDir, fmt.Sprintf("domain_collection_%s_%s.xlsx", taskID, time.Now().Format("20060102_150405")))

	err = file.Save(filename)
	if err != nil {
		return "", err
	}

	return filename, nil
}

// GetTaskLogs 获取任务日志
func (dcc *DomainCollectionController) GetTaskLogs(taskID string, offset, limit int) (string, error) {
	task, err := dcc.DomainCollection.GetTask(taskID)
	if err != nil {
		return "", err
	}

	logs := task.Logs
	if offset < 0 {
		offset = 0
	}
	if offset >= len(logs) {
		return "[]", nil
	}

	end := offset + limit
	if end > len(logs) {
		end = len(logs)
	}

	logsSlice := logs[offset:end]
	jsonData, err := json.Marshal(logsSlice)
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}

// ClearAllTasks 清空所有任务
func (dcc *DomainCollectionController) ClearAllTasks() error {
	return dcc.DomainCollection.ClearAllTasks()
}

// SaveTaskLog 保存任务日志
func (dcc *DomainCollectionController) SaveTaskLog(taskID string) error {
	return dcc.DomainCollection.SaveTaskLog(taskID)
}
