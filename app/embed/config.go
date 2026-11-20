package extractor

import (
	"EasyTools/app/embed/config"
	"EasyTools/app/embed/cyberchef"
	"EasyTools/app/embed/dict"
	"EasyTools/app/embed/restmate"
	"EasyTools/app/embed/unwxapp"
	"path/filepath"
)

// InitExtractor 初始化解压器配置
func InitExtractor() {
	ext := GetExtractor()

	configs := map[string]*ExtractConfig{
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
			TargetDir:    filepath.Join("config", "restmate"),
			Description:  "Restmate工具",
			CheckSymlink: true,
			ExcludeExts:  []string{".go", ".mod", ".sum"},
		},
		"dict": {
			FS:             dict.Assets,
			TargetDir:      filepath.Join("tools", "dict"),
			Description:    "dict字典",
			CheckSymlink:   true,
			ExcludeExts:    []string{".go", ".mod", ".sum"},
			ForceCheckDirs: []string{"dirsearch", "jwt", "domain"},
		},
		"config": {
			FS:           config.Assets,
			TargetDir:    filepath.Join("config"),
			Description:  "配置文件",
			CheckSymlink: true,
			ExcludeExts:  []string{".go", ".mod", ".sum"},
		},
	}

	ext.RegisterBatch(configs)
}
