package util

import (
	"embed"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
)

// ============================================================================
// 内置图标库（app/bypass/shell/util/ico/*.ico，来自原 ResourceHacker 资产目录，
// 工具 exe 已移除，仅保留 ico 图标供免杀生成时用户主动选择使用）。
// ============================================================================

//go:embed "ico"
var builtinIconFS embed.FS

var builtinIconNameRe = regexp.MustCompile(`^[A-Za-z0-9_\-]+\.ico$`)

// ListBuiltinIconFiles 返回内置图标库文件名列表（按名称排序，如 access.ico）
func ListBuiltinIconFiles() ([]string, error) {
	entries, err := builtinIconFS.ReadDir("ico")
	if err != nil {
		return nil, fmt.Errorf("读取内置图标库失败: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !builtinIconNameRe.MatchString(e.Name()) {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names, nil
}

// ValidBuiltinIconName 校验内置图标文件名（防路径穿越）
func ValidBuiltinIconName(name string) bool {
	return builtinIconNameRe.MatchString(name)
}

// ReadBuiltinIconFile 读取内置图标字节（name 需为 access.ico 这类文件名）
func ReadBuiltinIconFile(name string) ([]byte, error) {
	name = strings.TrimSpace(name)
	if !ValidBuiltinIconName(name) {
		return nil, fmt.Errorf("非法的内置图标名: %s", name)
	}
	data, err := builtinIconFS.ReadFile(path.Join("ico", name))
	if err != nil {
		return nil, fmt.Errorf("内置图标 %s 不存在: %w", name, err)
	}
	return data, nil
}
