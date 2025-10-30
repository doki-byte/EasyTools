package extractor

import (
	"EasyTools/app/embed/cyberchef"
	"EasyTools/app/embed/dict"
	"EasyTools/app/embed/note"
	"EasyTools/app/embed/restmate"
	"EasyTools/app/embed/unwxapp"
	"path/filepath"
)

// InitExtractor 初始化解压器配置
func InitExtractor() {
	ext := GetExtractor()

	configs := map[string]*ExtractConfig{
		"notes": {
			FS:           note.Assets,
			TargetDir:    filepath.Join("notes"),
			Description:  "笔记文件",
			CheckSymlink: true,
			ExcludeExts:  []string{".go", ".mod", ".sum"}, // 排除Go文件
		},
		"unwxapp": {
			FS:           unwxapp.Assets,
			TargetDir:    filepath.Join("tools", "Unwxapp"),
			Description:  "Unwxapp工具",
			CheckSymlink: true,
			ExcludeExts:  []string{".go", ".mod", ".sum"},
		},
		"cyberchef": {
			FS:           cyberchef.Assets,
			TargetDir:    filepath.Join("tools", "CyberChef"),
			Description:  "CyberChef工具",
			CheckSymlink: true,
			ExcludeExts:  []string{".go", ".mod", ".sum"},
		},
		"restmate": {
			FS:           restmate.Assets,
			TargetDir:    filepath.Join("tools", "restmate"),
			Description:  "Restmate工具",
			CheckSymlink: true,
			ExcludeExts:  []string{".go", ".mod", ".sum"},
		},
		"dict": {
			FS:           dict.Assets,
			TargetDir:    filepath.Join("tools", "dict"),
			Description:  "dict字典",
			CheckSymlink: true,
			ExcludeExts:  []string{".go", ".mod", ".sum"},
		},
	}

	ext.RegisterBatch(configs)
}
