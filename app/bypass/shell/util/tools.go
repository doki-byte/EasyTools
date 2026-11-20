package util

import (
	"EasyTools/app/bypass/shell/loader"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

func CheckGoEnvironment() error {
	missing := []string{}
	if _, err := exec.LookPath("go"); err != nil {
		missing = append(missing, "go")
	}
	if _, err := exec.LookPath("garble"); err != nil {
		missing = append(missing, "garble")
	}
	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("gcc"); err != nil {
			missing = append(missing, "gcc")
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("系统运行必须安装的 %v 存在缺失，请严格按照安装文档介绍进行运行安装", missing)
	}
	return nil
}

func InitGoModule() error {
	err := CheckGoEnvironment()
	if err != nil {
		return err
	}

	projectDir := filepath.Join(os.TempDir(), "EasyTools", "file")
	goModPath := filepath.Join(projectDir, "go.mod")
	goSumPath := filepath.Join(projectDir, "go.sum")

	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// go.mod 缺失时先初始化模块（仅首次/工作目录被清理后需要执行）
	if !pathExists(goModPath) {
		fmt.Printf("正在初始化Go模块...\n")
		cmd := exec.Command("go", "mod", "init", "EasyTools")
		cmd.SysProcAttr = hideWindowAttr()
		cmd.Dir = projectDir
		if output, err := cmd.CombinedOutput(); err != nil {
			if !strings.Contains(string(output), "go.mod already exists") {
				return fmt.Errorf("初始化Go模块失败: %s, 错误: %w", string(output), err)
			}
		}
	}

	missing := missingGoDeps(goModPath)
	if len(missing) == 0 {
		if pathExists(goSumPath) {
			fmt.Printf("Go模块已存在\n")
			return nil
		}
		// go.mod 已有依赖但 go.sum 缺失/损坏：尝试重新生成校验和（本地缓存优先，其次联网）
		if runGoCmd(projectDir, true, "mod", "download") == nil || runGoCmd(projectDir, false, "mod", "download") == nil {
			fmt.Printf("Go模块校验和已补齐\n")
			return nil
		}
		return fmt.Errorf("go.sum 生成失败，请检查网络后重试")
	}

	// 依赖版本必须与本机实际可用的 Go 工具链匹配：模块缓存中常同时存在要求更高 Go 的
	// 版本（例如 x/sys v0.48 要求 go>=1.26），若按"缓存里最高版本"安装，离线安装必然失败
	// （GOPROXY=off 时无法补拉新工具链），于是每次初始化都被迫联网重下依赖，
	// 网络受限时"处理bin文件"就会直接报初始化失败。
	modCache, toolchainVer, envOK := goEnvInfo()
	// GOTOOLCHAIN=auto 时 go 会按 go.mod 的 go 指令切换到更高的工具链，
	// 因此"实际可用版本"取本地工具链版本与该指令的较高者。
	if dirVer, ok := goDirectiveOfModFile(goModPath); ok && versionGreater(dirVer, toolchainVer) {
		toolchainVer = dirVer
	}

	// 本地缓存中可离线使用的依赖版本（空串=缓存中没有可用版本，只能联网取）
	depVers := planDepInstalls(modCache, missing, toolchainVer, envOK)

	// 依赖缺失：先尝试从本地模块缓存离线补齐（免联网、秒级完成），失败再统一走代理下载
	if installDepsOffline(projectDir, missing, depVers) {
		fmt.Printf("Go依赖已就绪（本地缓存）\n")
		return nil
	}

	var depNames []string
	for _, idx := range missing {
		depNames = append(depNames, goDeps[idx].Pkg)
	}
	fmt.Printf("正在安装依赖 %s...\n", strings.Join(depNames, "、"))

	// 所有缺失依赖合并为一次 go get（原实现逐个联网安装，耗时数倍；
	// 用户未配置 GOPROXY 时自动附加国内可达的代理链 goproxy.cn）。
	// 缓存中已有"本机工具链可编译"的版本时直接锁定该版本：否则 go get 会取最新版，
	// 可能强制下载上百 MB 的新工具链，并把 go.mod 的 go 指令一并顶高。
	args := []string{"get"}
	for i, idx := range missing {
		if depVers[i] != "" {
			args = append(args, goDeps[idx].Module+"@"+depVers[i])
			continue
		}
		args = append(args, goDeps[idx].Pkg)
	}
	if err := runGoCmd(projectDir, false, args...); err != nil {
		return fmt.Errorf("安装go依赖库 %s 失败: %s", strings.Join(depNames, "、"), shortOutput(err.Error()))
	}

	fmt.Printf("Go模块初始化完成\n")
	return nil
}

// goDeps 免杀加载器编译所需的第三方依赖（包路径 -> 模块名）
var goDeps = []struct {
	Pkg    string
	Module string
}{
	{"golang.org/x/sys", "golang.org/x/sys"},
	{"golang.org/x/crypto/chacha20", "golang.org/x/crypto"},
	{"github.com/google/uuid", "github.com/google/uuid"},
}

// missingGoDeps 返回 go.mod 中尚未 require 的依赖下标列表
func missingGoDeps(goModPath string) []int {
	var missing []int
	for idx := range goDeps {
		if !goModRequires(goModPath, goDeps[idx].Module) {
			missing = append(missing, idx)
		}
	}
	return missing
}

// goModRequires 判断 go.mod 中是否已 require 指定模块
func goModRequires(goModPath, module string) bool {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == module && strings.HasPrefix(fields[1], "v") {
			return true
		}
	}
	return false
}

// installDepsOffline 优先从本地模块缓存离线补齐依赖：命中缓存时免联网安装。
// depVers 与 missing 一一对应，任一为空串即表示缓存中没有可离线使用的版本，直接放弃离线方案。
func installDepsOffline(projectDir string, missing []int, depVers []string) bool {
	args := []string{"get"}
	for i, idx := range missing {
		if depVers[i] == "" {
			return false // 任一依赖本地缓存缺失则放弃离线方案
		}
		args = append(args, goDeps[idx].Module+"@"+depVers[i])
	}
	if err := runGoCmd(projectDir, true, args...); err != nil {
		fmt.Printf("本地缓存恢复依赖失败，转为联网安装: %s\n", shortOutput(err.Error()))
		return false
	}
	return true
}

// planDepInstalls 为缺失依赖挑选"本机可离线使用"的缓存版本，
// 返回值与 missing 一一对应（空串=本地缓存没有可用版本）。
func planDepInstalls(modCache string, missing []int, toolchainVer [3]int, envOK bool) []string {
	vers := make([]string, len(missing))
	if !envOK {
		return vers
	}
	for i, idx := range missing {
		if ver, ok := bestCachedVersion(modCache, goDeps[idx].Module, toolchainVer); ok {
			vers[i] = ver
		}
	}
	return vers
}

// goEnvInfo 返回本机模块缓存目录与本地 Go 工具链版本。
// GOTOOLCHAIN=local：确保拿到的是"本机实际安装的"Go 版本，
// 而不是被 go.mod 的 toolchain/go 指令切过去的版本。
func goEnvInfo() (modCache string, toolchainVer [3]int, ok bool) {
	cmd := exec.Command("go", "env", "GOMODCACHE", "GOVERSION")
	cmd.SysProcAttr = hideWindowAttr()
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	out, err := cmd.Output()
	if err != nil {
		return "", toolchainVer, false
	}
	lines := strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n")
	modCache = strings.TrimSpace(lines[0])
	if modCache == "" {
		return "", toolchainVer, false
	}
	if len(lines) > 1 {
		v := strings.TrimSpace(lines[1])
		toolchainVer = parseGoVersionNum(strings.TrimPrefix(v, "go"))
	}
	return modCache, toolchainVer, true
}

// versionGreater 判断语义化版本 a 是否高于 b（用于 [3]int 形式的版本号）。
func versionGreater(a, b [3]int) bool {
	for i := 0; i < 3; i++ {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return false
}

// escapeModulePath 复刻 Go 模块缓存对模块路径的转义规则（大写字母 → "!"+小写）。
func escapeModulePath(module string) string {
	var b strings.Builder
	for _, r := range module {
		if r >= 'A' && r <= 'Z' {
			b.WriteByte('!')
			b.WriteRune(r + ('a' - 'A'))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// downloadCacheVersionDir 返回模块在下载缓存中的版本目录。
// GOPROXY=off 时 go 只能从这里取模块，因此离线可用版本必须以此目录为准。
func downloadCacheVersionDir(modCache, module string) string {
	return filepath.Join(modCache, "cache", "download", filepath.FromSlash(escapeModulePath(module)), "@v")
}

// goDirectiveOfModFile 读取缓存 .mod 文件中的 go 指令（如 "go 1.26.0"）。
// 第二个返回值表示是否真的存在该指令（老模块可能没有）。
func goDirectiveOfModFile(modFile string) ([3]int, bool) {
	data, err := os.ReadFile(modFile)
	if err != nil {
		return [3]int{}, false
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "go ") {
			continue
		}
		ver := strings.TrimSpace(strings.TrimPrefix(line, "go "))
		if ver == "" {
			continue
		}
		return parseGoVersionNum(ver), true
	}
	return [3]int{}, false
}

// versionCompatWithToolchain 判断模块自带的 go 指令能否被本机工具链直接编译。
// 无 go 指令（老模块）或工具链版本未知时视为兼容。
func versionCompatWithToolchain(goDirective [3]int, hasDirective bool, toolchainVer [3]int) bool {
	if !hasDirective || toolchainVer == [3]int{} {
		return true
	}
	return !versionGreater(goDirective, toolchainVer)
}

// bestCachedVersion 在模块下载缓存中查找"本机离线可用"的最高版本：
//  1. 版本自带的 go 指令必须不高于本机工具链——否则 go 会尝试切换到更高工具链，
//     而离线场景（GOPROXY=off）无法下载工具链，安装必然失败；
//  2. zip/ziphash 必须在下载缓存中，否则 GOPROXY=off 同样取不到模块。
//
// 历史实现只按"缓存里最高版本"选取：一旦缓存中存在要求更高 Go 的版本，
// 离线安装就永远失败，只能退回联网下载（网络受限时即表现为初始化报错）。
func bestCachedVersion(modCache, module string, toolchainVer [3]int) (string, bool) {
	versionDir := downloadCacheVersionDir(modCache, module)
	entries, err := os.ReadDir(versionDir)
	if err != nil {
		return "", false
	}

	best := ""
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".mod") {
			continue
		}
		ver := strings.TrimSuffix(name, ".mod")
		if ver == "" {
			continue
		}
		if best != "" && compareGoVersion(ver, best) <= 0 {
			continue // 已有更高的可用版本，跳过
		}
		// 下载缓存里没有 zip/ziphash 的版本，GOPROXY=off 时取不到
		if !pathExists(filepath.Join(versionDir, ver+".zip")) && !pathExists(filepath.Join(versionDir, ver+".ziphash")) {
			continue
		}
		if dirVer, ok := goDirectiveOfModFile(filepath.Join(versionDir, ver+".mod")); !versionCompatWithToolchain(dirVer, ok, toolchainVer) {
			continue
		}
		best = ver
	}
	if best == "" {
		return "", false
	}
	return best, true
}

// compareGoVersion 简易语义化版本比较（足够用于挑选缓存版本）
func compareGoVersion(a, b string) int {
	if a == b {
		return 0
	}
	ap, bp := strings.SplitN(a, "-", 2), strings.SplitN(b, "-", 2)
	an, bn := parseGoVersionNum(ap[0]), parseGoVersionNum(bp[0])
	for i := 0; i < 3; i++ {
		if an[i] != bn[i] {
			if an[i] < bn[i] {
				return -1
			}
			return 1
		}
	}
	// 主版本号相同：正式版优先于预发布/伪版本，其次按预发布字符串比较
	switch {
	case len(ap) == 1 && len(bp) == 1:
		return 0
	case len(ap) == 1:
		return 1
	case len(bp) == 1:
		return -1
	default:
		if ap[1] == bp[1] {
			return 0
		}
		if ap[1] < bp[1] {
			return -1
		}
		return 1
	}
}

// parseGoVersionNum 解析 v1.2.3 → [1,2,3]（缺省位补 0）
func parseGoVersionNum(v string) [3]int {
	var num [3]int
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	parts := strings.Split(v, ".")
	for i := 0; i < len(parts) && i < 3; i++ {
		if n, err := strconv.Atoi(parts[i]); err == nil {
			num[i] = n
		}
	}
	return num
}

// runGoCmd 运行 go 子命令：offline=true 时强制仅使用本地缓存（GOPROXY=off），
// 并关闭校验和数据库依赖以便离线补齐 go.sum；offline=false 时附加国内代理链。
func runGoCmd(projectDir string, offline bool, args ...string) error {
	cmd := exec.Command("go", args...)
	cmd.SysProcAttr = hideWindowAttr()
	cmd.Dir = projectDir
	if offline {
		cmd.Env = append(os.Environ(),
			"GOPROXY=off",
			"GOSUMDB=off",
			"GOFLAGS=-mod=mod",
		)
	} else {
		cmd.Env = append(os.Environ(), goProxyExtra()...)
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s", shortOutput(string(output)))
	}
	return nil
}

// goProxyExtra 用户未配置 GOPROXY 时，返回附加的国内可达 Go 模块代理链；已配置则原样使用。
func goProxyExtra() []string {
	if os.Getenv("GOPROXY") != "" {
		return nil
	}
	return []string{"GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct"}
}

// shortOutput 截断过长的命令输出，避免错误提示刷屏
func shortOutput(s string) string {
	s = strings.TrimSpace(s)
	const maxLen = 500
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...(输出过长已截断)"
}

// pathExists 判断路径是否存在
func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func GenerateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes)[:length], nil
}

func Contains(slice []string, element string) bool {
	for _, value := range slice {
		if value == element {
			return true
		}
	}
	return false
}

func DelDirPath(filePath string) {
	os.RemoveAll(filePath)
}

// NormalizeGoArch 规范化目标架构为 Go 的 GOARCH 值：
// amd64/x64 → "amd64"；386/x86/i386 → "386"；其他默认 "amd64"。
func NormalizeGoArch(arch string) string {
	switch strings.ToLower(strings.TrimSpace(arch)) {
	case "386", "x86", "i386", "32":
		return "386"
	case "amd64", "x64", "64":
		return "amd64"
	default:
		return "amd64"
	}
}

// NormalizeTargetOS 规范化目标操作系统
func NormalizeTargetOS(targetOS string) string {
	switch strings.ToLower(strings.TrimSpace(targetOS)) {
	case "linux":
		return "linux"
	case "darwin", "mac", "macos":
		return "darwin"
	default:
		return "windows"
	}
}

// BinaryExtension 根据目标平台返回可执行文件扩展名
func BinaryExtension(targetOS string) string {
	if NormalizeTargetOS(targetOS) == "windows" {
		return ".exe"
	}
	return ""
}

// 模板目录已全部规范命名为 loaderN，目录名不再携带任何技术/方式信息。
// 个别模板承担特殊角色，统一在此登记编号，避免散落魔法数字，也方便后续按编号维护。
//
// 注意：这些常量只是"首选名"，必须经 DefaultWindowsLoader / DefaultUnixLoader 解析后使用。
// 历史上模板集从 loader1..loader53 收缩到 loader1..loader20 时，常量未同步，
// 导致"未指定运行模式"与"Linux/macOS 目标"两条路径稳定拿到不存在的模板
// （GenGoFile 直接报 loader 模板 loader53/loader51 不存在），且报错信息与真实原因无关。
const (
	// LoaderBaiJiaHei 白加黑专用模板（需配合导出函数/资源处理，仅 Windows 目标）
	LoaderBaiJiaHei = "loader10"
	// LoaderUnixCallback / LoaderUnixMmap 面向 Linux/macOS 生成目标的模板（首选名）
	LoaderUnixCallback = "loader50"
	LoaderUnixMmap     = "loader51"
	// LoaderWindowsDefault 未指定运行方式时的首选默认模板
	LoaderWindowsDefault = "loader53"
	// loaderFallbackWindows 首选默认模板不存在时的兜底模板（长期存在的经典模板）
	loaderFallbackWindows = "loader1"
)

// unixLoaderNames 非 Windows 生成目标可用的模板集合
var unixLoaderNames = map[string]bool{
	LoaderUnixMmap:     true,
	LoaderUnixCallback: true,
}

// loaderNumber 解析 loaderN 的编号；非规范名返回 false。
func loaderNumber(name string) (int, bool) {
	if !strings.HasPrefix(name, "loader") {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimPrefix(name, "loader"))
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

// HasLoaderTemplate 判断模板是否真实存在。
func HasLoaderTemplate(name string) bool {
	raw, ok := loader.Modules[name]
	return ok && len(raw) > 0
}

// LoaderTemplate 返回模板内容（不存在时 ok=false）。
func LoaderTemplate(name string) ([]byte, bool) {
	raw, ok := loader.Modules[name]
	if !ok || len(raw) == 0 {
		return nil, false
	}
	return raw, true
}

// SortedLoaderNames 返回全部模板名（按 loaderN 编号升序，非规范名排在后面）。
func SortedLoaderNames() []string {
	names := make([]string, 0, len(loader.Modules))
	for k := range loader.Modules {
		names = append(names, k)
	}
	sort.Slice(names, func(i, j int) bool {
		ni, oki := loaderNumber(names[i])
		nj, okj := loaderNumber(names[j])
		switch {
		case oki && okj:
			return ni < nj
		case oki != okj:
			return oki
		default:
			return names[i] < names[j]
		}
	})
	return names
}

// DefaultWindowsLoader 解析可用的默认 Windows 模板：
//  1. 首选 LoaderWindowsDefault（模板集扩充后自动生效，保持原设计意图）
//  2. 其次 loader1（经典模板）
//  3. 最后取编号最小的 Windows 模板
func DefaultWindowsLoader() (string, bool) {
	if HasLoaderTemplate(LoaderWindowsDefault) {
		return LoaderWindowsDefault, true
	}
	if HasLoaderTemplate(loaderFallbackWindows) {
		return loaderFallbackWindows, true
	}
	for _, name := range SortedLoaderNames() {
		if !unixLoaderNames[name] {
			return name, true
		}
	}
	return "", false
}

// DefaultUnixLoader 解析可用的 Linux/macOS 模板；当前模板集不含 Unix 模板时返回 false。
func DefaultUnixLoader() (string, bool) {
	for _, name := range []string{LoaderUnixMmap, LoaderUnixCallback} {
		if HasLoaderTemplate(name) {
			return name, true
		}
	}
	for _, name := range SortedLoaderNames() {
		if unixLoaderNames[name] {
			return name, true
		}
	}
	return "", false
}

// IsWindowsLoader 判断 loader 是否仅适用于 Windows 生成目标
func IsWindowsLoader(name string) bool {
	return !unixLoaderNames[name]
}

// FilterLoadersByTargetOS 按目标平台过滤 loader 列表
func FilterLoadersByTargetOS(allLoaders []string, targetOS string) []string {
	isUnix := NormalizeTargetOS(targetOS) != "windows"
	filtered := make([]string, 0, len(allLoaders))
	for _, name := range allLoaders {
		if isUnix == unixLoaderNames[name] {
			filtered = append(filtered, name)
		}
	}
	return filtered
}
