package util

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestFindMinGW 验证跨平台 MinGW 查找的各分支。
func TestFindMinGW(t *testing.T) {
	// 1. 环境变量路径指向不存在目录时，应回退到常见路径/PATH 扫描（结果可为 nil 或回退命中）
	old := os.Getenv("MINGW64_PATH")
	os.Setenv("MINGW64_PATH", filepath.Join(t.TempDir(), "nope"))
	t.Cleanup(func() { os.Setenv("MINGW64_PATH", old) })
	fallback := FindMinGW("mingw64")
	t.Logf("无效环境变量后回退结果: %v", func() string {
		if fallback == nil {
			return "nil（未安装，符合预期）"
		}
		return "命中 " + fallback.CC + "（回退扫描）"
	}())

	// 2. 构造一个临时的"编译器目录"（<root>/bin/<cc>），验证环境变量分支
	fakeRoot := t.TempDir()
	fakeBin := filepath.Join(fakeRoot, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatal(err)
	}
	ccName := "gcc.exe"
	if runtime.GOOS != "windows" {
		ccName = "x86_64-w64-mingw32-gcc"
	}
	ccPath := filepath.Join(fakeBin, ccName)
	if err := os.WriteFile(ccPath, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	os.Setenv("MINGW64_PATH", fakeRoot)
	info := FindMinGW("mingw64")
	if info == nil {
		t.Fatalf("有效 MINGW64_PATH 应返回编译器信息")
	}
	if info.CC != ccPath {
		t.Errorf("CC 应为 %s，得到 %s", ccPath, info.CC)
	}
	if info.PATH == "" {
		t.Errorf("PATH 不应为空")
	}
	t.Logf("找到 MinGW: CC=%s", info.CC)

	// 3. 清理环境变量后，PATH 中查找（本机不一定有，返回 nil 也可接受）
	os.Setenv("MINGW64_PATH", "")
	if info := FindMinGW("mingw64"); info != nil {
		t.Logf("PATH 中找到 MinGW: CC=%s", info.CC)
	} else {
		t.Logf("PATH 中未找到 MinGW（本机未安装，符合预期）")
	}

	// 4. 兼容旧接口 FindMinGWPath
	os.Setenv("MINGW64_PATH", fakeRoot)
	if p := FindMinGWPath("mingw64"); p == "" {
		t.Errorf("FindMinGWPath 应返回有效路径")
	} else {
		t.Logf("FindMinGWPath = %s", p)
	}
}
