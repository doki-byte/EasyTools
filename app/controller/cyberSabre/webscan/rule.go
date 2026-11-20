package webscan

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// WorkFlowDB POC 标签映射：key 为相对模板路径（不含 .yaml），value 为指纹标签列表（可为空）。
// 无 info.tags 的模板仍会入库，供「指定漏洞」等与 ListPocs 对齐的列表展示；指纹匹配仅使用非空 tags。
var WorkFlowDB map[string][]string

var mutx sync.RWMutex

// TemplateInfo 模板信息结构（兼容 tags 为字符串或数组的两种写法）
type TemplateInfo struct {
	Tags interface{} `yaml:"tags"`
}

// Template 模板结构
type Template struct {
	Info TemplateInfo `yaml:"info"`
}

// normalizeTags 将 info.tags 归一化为字符串切片，兼容多种 YAML 写法
func normalizeTags(raw interface{}) []string {
	if raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		parts := strings.Split(v, ",")
		var out []string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		return out
	case []interface{}:
		var out []string
		for _, item := range v {
			s := strings.TrimSpace(toString(item))
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		var out []string
		for _, item := range v {
			s := strings.TrimSpace(item)
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// toString 辅助函数：将 interface{} 尽量转换为字符串
func toString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	default:
		return ""
	}
}

// GetTagsList 从模板文件夹中获取所有 POC 的标签列表
func GetTagsList(templateFolders []string) error {
	tmp := make(map[string][]string)
	for _, folder := range templateFolders {
		if _, err := os.Stat(folder); os.IsNotExist(err) {
			continue
		}
		// 遍历所有模板文件
		err := filepath.WalkDir(folder, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.HasSuffix(d.Name(), ".yaml") {
				file, err := os.ReadFile(path)
				if err != nil {
					return nil
				}
				relPath, relErr := filepath.Rel(folder, path)
				if relErr != nil {
					relPath = d.Name()
				}
				relPath = filepath.ToSlash(relPath)
				pocName := strings.TrimSuffix(relPath, filepath.Ext(relPath))
				if pocName == "" {
					return nil
				}

				var tags []string
				var template Template
				if err := yaml.Unmarshal(file, &template); err == nil {
					tags = normalizeTags(template.Info.Tags)
				}
				// 每个 yaml 都入库（tags 可为空），与 ListPocs 一致，避免「指定漏洞」下拉只有带 tags 的项
				tmp[pocName] = tags
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	// 一次性替换，避免并发读写
	mutx.Lock()
	WorkFlowDB = tmp
	mutx.Unlock()
	return nil
}

// GetFingerPocMap 获取指纹到 POC 的映射关系（供前端调用）
func GetFingerPocMap() map[string][]string {
	mutx.RLock()
	defer mutx.RUnlock()
	
	// 返回副本，避免并发修改
	// 注意：range nil map 是安全的，会返回空结果
	result := make(map[string][]string)
	for k, v := range WorkFlowDB {
		result[k] = make([]string, len(v))
		copy(result[k], v)
	}
	return result
}

// ReloadPocTags 重新加载 POC 标签（用于更新 POC 后调用）
func ReloadPocTags(ctx context.Context, templateFolders []string) error {
	return GetTagsList(templateFolders)
}

