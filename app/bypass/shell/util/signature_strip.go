package util

import (
	"fmt"
	"path/filepath"
	"strings"
)

// PostProcessBuiltBinary 编译后自动处理：
// 1. 默认去除黑窗口（keepConsole=true 时保留控制台，供 mimikatz 等命令行工具输出结果）；
// 2. 自动执行静态特征清理（字符串/段名/构建信息等常规特征）。
func PostProcessBuiltBinary(outputPath, targetOS string, keepConsole bool) {
	isPE := strings.EqualFold(NormalizeTargetOS(targetOS), "windows") && IsPEFile(outputPath)
	if isPE && !keepConsole {
		if err := CloseWindowsConsole(outputPath); err != nil {
			fmt.Printf("[!] 去除黑窗口失败 %v\n", err)
		} else {
			fmt.Printf("[+] 去除黑窗口成功: %s\n", filepath.Base(outputPath))
		}
	} else if isPE && keepConsole {
		fmt.Printf("[=] 保留控制台（命令行工具模式）: %s\n", filepath.Base(outputPath))
	}

	// 自动静态特征去除（纯字节层面，不改签名/资源结构）
	result, err := ScanAndEvadeAVSignatures(outputPath)
	if err != nil {
		fmt.Printf("[!] 静态特征清理失败 %s: %v\n", filepath.Base(outputPath), err)
		return
	}
	if len(result.Evaded) > 0 {
		names := make([]string, 0, len(result.Evaded))
		for _, h := range result.Evaded {
			names = append(names, h.Name)
		}
		fmt.Printf("[+] 静态特征清理完成 %s: %v\n", filepath.Base(outputPath), names)
	} else {
		fmt.Printf("[=] 静态特征扫描无命中: %s\n", filepath.Base(outputPath))
	}
	// 真实残留 = 清理后再扫描仍命中的项（原实现打印的是清理前的命中清单，属误导）
	after2, _ := ScanAVSignatures(outputPath)
	if len(after2.Found) > 0 {
		remain := make([]string, 0, len(after2.Found))
		for _, h := range after2.Found {
			remain = append(remain, h.Name)
		}
		fmt.Printf("[!] 警告: %s 仍有 %d 类特征未能自动清理: %v（可手动执行「去UPX/特征」）\n",
			filepath.Base(outputPath), len(after2.Found), remain)
	}

	// 静态启发缓解：写入随机化拟真 PE 版本信息（放最后：等其它字节级改动完成后再重建资源目录）
	if isPE {
		if err := RandomizePEResources(outputPath); err != nil {
			fmt.Printf("[!] 写入版本信息失败 %s: %v\n", filepath.Base(outputPath), err)
		} else {
			fmt.Printf("[+] 已写入随机版本信息: %s\n", filepath.Base(outputPath))
		}
	}
}
