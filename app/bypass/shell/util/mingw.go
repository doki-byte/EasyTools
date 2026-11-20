package util

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// MinGWInfo MinGW-w64 交叉编译器信息
type MinGWInfo struct {
	BinDir string // 编译器 bin 目录（含 CC 可执行文件）
	CC     string // 编译器可执行文件完整路径
	PATH   string // 追加到进程 PATH 的片段（含分隔符）
}

// mingwCCFileName 返回当前平台下 MinGW 交叉编译器的可执行文件名：
//   - Windows：gcc.exe（本地 MinGW）
//   - Linux/macOS：x86_64-w64-mingw32-gcc / i686-w64-mingw32-gcc（mingw-w64 交叉编译器）
func mingwCCFileName(arch string) string {
	if runtime.GOOS == "windows" {
		return "gcc.exe"
	}
	if strings.EqualFold(arch, "x86") || strings.EqualFold(arch, "mingw32") {
		return "i686-w64-mingw32-gcc"
	}
	return "x86_64-w64-mingw32-gcc"
}

// pathListSep 返回当前平台的 PATH 列表分隔符。
func pathListSep() string {
	if runtime.GOOS == "windows" {
		return ";"
	}
	return ":"
}

// FindMinGW 跨平台查找 MinGW-w64 编译器（白加黑 DLL 交叉编译用）。
// arch 支持 "mingw64"/"x64"（x86_64-w64-mingw32-gcc）与 "mingw32"/"x86"（i686-w64-mingw32-gcc）。
// 找不到时返回 nil。
func FindMinGW(arch string) *MinGWInfo {
	arch = strings.ToLower(strings.TrimSpace(arch))
	if arch == "" {
		arch = "mingw64"
	}

	// 1. 环境变量优先（MINGW64_PATH / MINGW32_PATH，兼容旧逻辑）
	envKey := "MINGW_" + strings.ToUpper(arch) + "_PATH"
	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		if info := mingwFromDir(v, arch); info != nil {
			return info
		}
	}
	if arch == "mingw64" {
		if v := strings.TrimSpace(os.Getenv("MINGW64_PATH")); v != "" {
			if info := mingwFromDir(v, arch); info != nil {
				return info
			}
		}
	} else if arch == "mingw32" {
		if v := strings.TrimSpace(os.Getenv("MINGW32_PATH")); v != "" {
			if info := mingwFromDir(v, arch); info != nil {
				return info
			}
		}
	}

	// 2. Windows：常见安装路径
	if runtime.GOOS == "windows" {
		commonPaths := []string{
			"D:\\mingw\\" + arch,
			"C:\\mingw\\" + arch,
			"D:\\mingw32\\" + arch,
			"C:\\mingw32\\" + arch,
			"D:\\mingw64\\" + arch,
			"C:\\mingw64\\" + arch,
			"D:\\msys64\\" + arch,
			"C:\\msys64\\" + arch,
			filepath.Join(os.Getenv("ProgramFiles"), "mingw-w64", arch),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "mingw-w64", arch),
		}
		for _, basePath := range commonPaths {
			if info := mingwFromDir(basePath, arch); info != nil {
				return info
			}
		}
	}

	// 3. PATH 中查找交叉编译器（Linux/macOS 的 mingw-w64 及 Windows 的 PATH 安装）
	ccName := mingwCCFileName(arch)
	if ccPath, err := exec.LookPath(ccName); err == nil {
		binDir := filepath.Dir(ccPath)
		return &MinGWInfo{
			BinDir: binDir,
			CC:     ccPath,
			PATH:   binDir + pathListSep() + os.Getenv("PATH"),
		}
	}

	// 4. PATH 中查找通用 gcc/gcc.exe（兼容用户自行软链的情况）
	gccName := "gcc"
	if runtime.GOOS == "windows" {
		gccName = "gcc.exe"
	}
	if ccPath, err := exec.LookPath(gccName); err == nil {
		binDir := filepath.Dir(ccPath)
		return &MinGWInfo{
			BinDir: binDir,
			CC:     ccPath,
			PATH:   binDir + pathListSep() + os.Getenv("PATH"),
		}
	}

	return nil
}

// mingwFromDir 检查目录是否包含可用的 MinGW 编译器。
func mingwFromDir(basePath, arch string) *MinGWInfo {
	if basePath == "" {
		return nil
	}
	binDir := filepath.Join(basePath, "bin")
	ccName := mingwCCFileName(arch)
	ccPath := filepath.Join(binDir, ccName)
	if _, err := os.Stat(ccPath); err != nil {
		return nil
	}
	return &MinGWInfo{
		BinDir: binDir,
		CC:     ccPath,
		PATH:   binDir + pathListSep() + os.Getenv("PATH"),
	}
}

// FindMinGWPath 查找 MinGW 安装路径（兼容旧调用，返回 bin 的父目录即安装根目录）。
// 跨平台：Linux/macOS 下返回编译器所在目录。
func FindMinGWPath(arch string) string {
	info := FindMinGW(arch)
	if info == nil {
		return ""
	}
	return filepath.Dir(info.BinDir)
}

// ExtractBitFromHeader 从 SkyShadow payload header 中提取架构信息
// header 格式示例：x64 1234.00MB 有数字签名 xxx.exe
func ExtractBitFromHeader(header string) string {
	header = strings.TrimSpace(header)
	if strings.HasPrefix(strings.ToLower(header), "x64") {
		return "x64"
	} else if strings.HasPrefix(strings.ToLower(header), "x86") {
		return "x86"
	}
	// 默认返回 x64
	return "x64"
}
