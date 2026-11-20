package util

import (
	"os"
	"path/filepath"
	"testing"
)

// writeModFile 在合成模块缓存中写入一个版本的 .mod 文件。
func writeModFile(t *testing.T, modCache, module, ver, goDirective string) {
	t.Helper()
	dir := downloadCacheVersionDir(modCache, module)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("创建缓存目录失败: %v", err)
	}
	content := "module " + module + "\n"
	if goDirective != "" {
		content += "\ngo " + goDirective + "\n"
	}
	if err := os.WriteFile(filepath.Join(dir, ver+".mod"), []byte(content), 0644); err != nil {
		t.Fatalf("写入 .mod 失败: %v", err)
	}
}

// touchVersionArtifacts 生成 GOPROXY=off 时取模块所需的 zip/ziphash 占位文件。
func touchVersionArtifacts(t *testing.T, modCache, module, ver string) {
	t.Helper()
	dir := downloadCacheVersionDir(modCache, module)
	for _, ext := range []string{".zip", ".ziphash"} {
		if err := os.WriteFile(filepath.Join(dir, ver+ext), []byte("x"), 0644); err != nil {
			t.Fatalf("写入 %s%s 失败: %v", ver, ext, err)
		}
	}
}

// TestBestCachedVersionRespectsToolchain 回归：离线可选版本必须能被本机工具链直接编译。
// 历史 bug：只取“缓存里最高版本”，缓存中存在要求 go>=1.26 的版本而本机是 go1.25 时，
// 离线安装必然失败（GOPROXY=off 无法补拉工具链），每次初始化都被迫联网。
func TestBestCachedVersionRespectsToolchain(t *testing.T) {
	modCache := t.TempDir()
	const module = "golang.org/x/sys"

	// v0.48.0 要求 go>=1.26（本机 go1.25 装不了，会被 go 尝试切换工具链而失败）
	writeModFile(t, modCache, module, "v0.48.0", "1.26.0")
	touchVersionArtifacts(t, modCache, module, "v0.48.0")
	// v0.41.0 与工具链兼容，但下载缓存里没有 zip/ziphash，GOPROXY=off 同样取不到
	writeModFile(t, modCache, module, "v0.41.0", "1.18")
	// v0.40.0 兼容且缓存完整 —— 应被选中
	writeModFile(t, modCache, module, "v0.40.0", "1.18")
	touchVersionArtifacts(t, modCache, module, "v0.40.0")

	got, ok := bestCachedVersion(modCache, module, [3]int{1, 25, 3})
	if !ok {
		t.Fatalf("应选出可用版本，实际返回不可用")
	}
	if got != "v0.40.0" {
		t.Fatalf("应选中与本机工具链兼容且有缓存的 v0.40.0，实际 %s", got)
	}

	// 工具链升到 1.26 后，最高版本重新变得可用
	got, ok = bestCachedVersion(modCache, module, [3]int{1, 26, 0})
	if !ok || got != "v0.48.0" {
		t.Fatalf("go1.26 下应选中 v0.48.0，实际 ok=%v ver=%s", ok, got)
	}
}

// TestBestCachedVersionRejectsMissingDownloadCache 回归：模块只被解压、下载缓存被清理时，
// 不能把该版本当作“可离线使用”，否则离线 go get 必然失败。
func TestBestCachedVersionRejectsMissingDownloadCache(t *testing.T) {
	modCache := t.TempDir()
	// 仅存在解压后的模块目录（历史实现正是据此选版本）
	extracted := filepath.Join(modCache, "golang.org", "x", "sys@v0.18.0")
	if err := os.MkdirAll(extracted, 0755); err != nil {
		t.Fatalf("创建解压目录失败: %v", err)
	}

	if _, ok := bestCachedVersion(modCache, "golang.org/x/sys", [3]int{1, 25, 3}); ok {
		t.Fatalf("下载缓存缺失时不应返回可用版本")
	}
}

// TestGoDirectiveOfModFile 校验 go 指令解析（缺失时报告不存在）。
func TestGoDirectiveOfModFile(t *testing.T) {
	dir := t.TempDir()

	withGo := filepath.Join(dir, "with.mod")
	if err := os.WriteFile(withGo, []byte("module m\n\ngo 1.18\n\nrequire x v1.0.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	ver, ok := goDirectiveOfModFile(withGo)
	if !ok || ver != [3]int{1, 18, 0} {
		t.Fatalf("应解析出 go 1.18，实际 ok=%v ver=%v", ok, ver)
	}

	withoutGo := filepath.Join(dir, "without.mod")
	if err := os.WriteFile(withoutGo, []byte("module m\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, ok := goDirectiveOfModFile(withoutGo); ok {
		t.Fatalf("无 go 指令时不应报告存在")
	}
	if _, ok := goDirectiveOfModFile(filepath.Join(dir, "missing.mod")); ok {
		t.Fatalf("文件不存在时不应报告存在")
	}
}

// TestVersionCompatWithToolchain 兼容性判定：未知工具链/无 go 指令时放行。
func TestVersionCompatWithToolchain(t *testing.T) {
	if versionCompatWithToolchain([3]int{1, 26, 0}, true, [3]int{1, 25, 3}) {
		t.Fatalf("go1.26 要求不应被 go1.25 工具链视为兼容")
	}
	if !versionCompatWithToolchain([3]int{1, 18, 0}, true, [3]int{1, 25, 3}) {
		t.Fatalf("go1.18 要求应被视为兼容")
	}
	if !versionCompatWithToolchain([3]int{0, 0, 0}, false, [3]int{1, 25, 3}) {
		t.Fatalf("无 go 指令应视为兼容")
	}
	if !versionCompatWithToolchain([3]int{1, 99, 0}, true, [3]int{0, 0, 0}) {
		t.Fatalf("工具链版本未知时应放行（交由 go 自行判定）")
	}
}

// TestEscapeModulePath 模块路径转义需与 Go 缓存目录规则一致（大写字母 → !小写）。
func TestEscapeModulePath(t *testing.T) {
	cases := map[string]string{
		"golang.org/x/sys":        "golang.org/x/sys",
		"github.com/BurntSushi/x": "github.com/!burnt!sushi/x",
		"github.com/google/uuid":  "github.com/google/uuid",
	}
	for in, want := range cases {
		if got := escapeModulePath(in); got != want {
			t.Fatalf("escapeModulePath(%q) = %q，期望 %q", in, got, want)
		}
	}
}
