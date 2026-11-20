package assistive

import (
	"fmt"
	"strings"
)

// AntivirusItem 杀软表结构体
type AntivirusItem struct {
	ID          int    `json:"id"`
	ProcessName string `json:"process_name"` // 对应 tasklist 的进程名
	Description string `json:"description"`  // 杀软名称
}

// TableName 指定表名
func (AntivirusItem) TableName() string {
	return "antivirus_list"
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
