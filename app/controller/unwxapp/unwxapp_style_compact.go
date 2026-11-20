package unwxapp

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// styleAttrRe 匹配内联 style 属性，仅处理双引号形式：style="..."
var styleAttrRe = regexp.MustCompile(`style="([^"]*?)"`)

// normalizeStyleValue 基于 compact_style_attr.py 的逻辑对 style 值进行规整
func normalizeStyleValue(styleValue string) string {
	parts := make([]string, 0)

	for _, rawDecl := range strings.Split(styleValue, ";") {
		decl := strings.TrimSpace(rawDecl)
		if decl == "" {
			continue
		}

		if idx := strings.IndexByte(decl, ':'); idx != -1 {
			prop := decl[:idx]
			value := decl[idx+1:]

			// 折叠属性和值中的多余空白
			prop = strings.Join(strings.Fields(strings.TrimSpace(prop)), " ")
			value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")

			if value != "" {
				parts = append(parts, prop+": "+value)
			} else {
				parts = append(parts, prop+":")
			}
		} else {
			// 非标准声明，简单折叠空白
			parts = append(parts, strings.Join(strings.Fields(decl), " "))
		}
	}

	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ";") + ";"
}

// transformContent 在整个文件内容上规整所有内联 style 属性，返回新内容与命中数
func transformContent(content string) (string, int) {
	styleHits := 0

	transformed := styleAttrRe.ReplaceAllStringFunc(content, func(match string) string {
		// match 形如 style="..."`
		m := styleAttrRe.FindStringSubmatch(match)
		if len(m) != 2 {
			return match
		}
		originalValue := m[1]
		compactedValue := normalizeStyleValue(originalValue)
		styleHits++
		return `style="` + compactedValue + `"`
	})

	return transformed, styleHits
}

// compactInlineStylesInDir 遍历目录，对所有 wxml 文件的内联 style 做规整
// 返回：总命中次数和可能的首个错误
func (u *UnWxapp) compactInlineStylesInDir(root string) (int, error) {
	totalHits := 0
	var firstErr error

	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// 保留首个错误，但不中断整体遍历
			if firstErr == nil {
				firstErr = err
			}
			return nil
		}

		if info.IsDir() {
			return nil
		}

		// 仅处理小程序模板文件，避免无谓开销
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".wxml" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			return nil
		}

		original := string(data)
		transformed, hits := transformContent(original)
		if hits == 0 || transformed == original {
			return nil
		}

		if err := os.WriteFile(path, []byte(transformed), 0644); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			return nil
		}

		totalHits += hits
		return nil
	})

	return totalHits, firstErr
}

