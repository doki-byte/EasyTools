package util

import (
	"EasyTools/app/bypass/shell/encode"
	"EasyTools/app/bypass/shell/loader"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// loaderNumericSort 按 loaderN 的序号排序（模板目录已统一规范命名为 loader1..loaderN）
func loaderNumericSort(names []string) []string {
	sorted := make([]string, len(names))
	copy(sorted, names)
	sort.Slice(sorted, func(i, j int) bool {
		ni, _ := strconv.Atoi(strings.TrimPrefix(sorted[i], "loader"))
		nj, _ := strconv.Atoi(strings.TrimPrefix(sorted[j], "loader"))
		return ni < nj
	})
	return sorted
}

// allModuleNames 返回全部模板编号（loader1..loaderN，按序号升序）
func allModuleNames() []string {
	var names []string
	for k := range loader.Modules {
		names = append(names, k)
	}
	return loaderNumericSort(names)
}

// TestGenLoaderEndToEnd 验证：生成模板 -> 内联数据 -> 变异 -> 输出可编译源码。
// 覆盖全部已注册 loader 模板，并断言生成源码中不残留任何固定特征/占位标记。
func TestGenLoaderEndToEnd(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "EasyTools", "file")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	// 写入模拟的加密数据文件
	os.WriteFile(filepath.Join(dir, "XorEncrypted.txt"), []byte{0xde, 0xad, 0xbe, 0xef}, 0644)
	os.WriteFile(filepath.Join(dir, "XorKey.txt"), []byte{0x01, 0x02, 0x03, 0x04, 0x02}, 0644)
	os.WriteFile(filepath.Join(dir, "AesEncrypted.txt"), []byte{1, 2, 3}, 0644)
	os.WriteFile(filepath.Join(dir, "AesKey.txt"), []byte("0123456789abcdef0123456789abcdef"), 0644)
	os.WriteFile(filepath.Join(dir, "ChaCha20Encrypted.txt"), []byte{9, 9}, 0644)
	os.WriteFile(filepath.Join(dir, "ChaCha20Key.txt"), make([]byte, 32), 0644)
	os.WriteFile(filepath.Join(dir, "ChaCha20Nonce.txt"), make([]byte, 12), 0644)
	os.WriteFile(filepath.Join(dir, "UrlOrFileEncrypted.txt"), []byte("68656c6c6f"), 0644)
	os.WriteFile(filepath.Join(dir, "UrlXorKey.txt"), []byte{0x42}, 0644)
	otpBlob, _ := encode.OtpEncrypt([]byte{0xEB, 0xFE})
	os.WriteFile(filepath.Join(dir, "OtpEncrypted.txt"), otpBlob, 0644)

	// 全部模板（含白加黑/Unix 模板）都执行一次生成流水线
	all := allModuleNames()
	if len(all) == 0 {
		t.Fatalf("loader 模板数量异常: %d", len(all))
	}

	// 有代表性的组合保留专项覆盖（加密方式/加载方式/沙箱口令路径）。
	// 注意：默认模板与 Unix 模板一律经注册表解析，模板集增删后测试不会失配。
	special := []struct {
		loader   string
		loadWay  string
		encode   string
		passBox  string
		targetOS string
	}{
		{mustWindowsLoader(t), "一体", "aes", "命令行运行", "windows"},
		{loaderN(t, 3), "分离", "xor", "双击运行", "windows"},
		{loaderN(t, 9), "远程", "xor", "双击运行", "windows"},
		// OTP 一次一密：loader1 走就地解码注入；默认 loader 走函数型回退
		{loaderN(t, 1), "一体", "otp", "双击运行", "windows"},
		{mustWindowsLoader(t), "一体", "otp", "双击运行", "windows"},
	}
	if unixLoader, ok := DefaultUnixLoader(); ok {
		special = append(special,
			struct {
				loader   string
				loadWay  string
				encode   string
				passBox  string
				targetOS string
			}{unixLoader, "一体", "xor", "双击运行", "linux"},
			struct {
				loader   string
				loadWay  string
				encode   string
				passBox  string
				targetOS string
			}{unixLoader, "一体", "aes", "双击运行", "darwin"},
		)
	} else {
		t.Logf("当前模板集不含 Linux/macOS 模板，跳过 Unix 生成用例")
	}
	allWindows := 0
	for _, name := range all {
		if IsWindowsLoader(name) {
			allWindows++
		}
	}
	t.Logf("注册模板总数=%d (windows=%d unix=%d)", len(all), allWindows, len(all)-allWindows)

	for _, name := range all {
		// 常规组合覆盖：一体 + 按序号轮换加密方式，扩大路径覆盖面
		idx := strings.TrimPrefix(name, "loader")
		encodings := []string{"xor", "aes", "chacha20"}
		encode := encodings[(len(idx)*7)%len(encodings)]
		runGenCase(t, name, "一体", encode, "双击运行", "windows")
	}
	for _, sc := range special {
		runGenCase(t, sc.loader, sc.loadWay, sc.encode, sc.passBox, sc.targetOS)
	}
}

// loaderN 仅用于测试：返回 loaderN 的规范名称（N 超范围时直接报错）
func loaderN(t *testing.T, n int) string {
	name := "loader" + strconv.Itoa(n)
	if !HasLoaderTemplate(name) {
		t.Fatalf("模板 %s 不存在", name)
	}
	return name
}

// mustWindowsLoader 返回当前模板集解析出的默认 Windows 模板
func mustWindowsLoader(t *testing.T) string {
	t.Helper()
	name, ok := DefaultWindowsLoader()
	if !ok {
		t.Fatal("没有可用的 Windows 模板")
	}
	return name
}

func runGenCase(t *testing.T, loaderName, loadWay, encode, passBox, targetOS string) {
	t.Helper()
	out, gerr := GenGoFile(loaderName, loadWay, encode, passBox, "false", "false", "", "false", "", "", nil, nil)
	if gerr != nil || out == "" {
		t.Fatalf("GenGoFile(%s, %s, %s) 失败: %v", loaderName, loadWay, encode, gerr)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("读取生成文件失败: %v", err)
	}
	content := string(data)

	// 固定特征检查：不应残留 go:embed / embeddedFiles / 固定文件名 / 固定口令 / 占位标记 / 模块名
	for _, bad := range []string{"//go:embed", "embeddedFiles", "XorEncrypted.txt", "AesEncrypted.txt",
		"EasyTools_1473692580", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36",
		"//__DATA_", "//__URL_XOR_KEY__", "//__SANDBOX_TOKEN__", "//__USER_AGENT__"} {
		if idx := indexOf(content, bad); idx >= 0 {
			t.Fatalf("loader=%s 仍包含固定特征 [%s] @%d\n%s", loaderName, bad, idx, content[idx:min(idx+120, len(content))])
		}
	}
	// 不应残留任何 //__XXX__ 占位符
	if idx := indexOf(content, "//__"); idx >= 0 {
		t.Fatalf("loader=%s 仍残留占位符标记 @%d\n%s", loaderName, idx, content[idx:min(idx+160, len(content))])
	}
	// 无 embed import
	for _, bad := range []string{"\"embed\"", "_ \"embed\""} {
		if indexOf(content, bad) >= 0 {
			t.Fatalf("loader=%s 仍包含 embed import [%s]", loaderName, bad)
		}
	}
	t.Logf("loader=%s loadWay=%s encode=%s OK (len=%d)", loaderName, loadWay, encode, len(content))
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
