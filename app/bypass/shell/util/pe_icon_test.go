package util

import (
	"bytes"
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/tc-hib/winres"
)

// buildWindowsPEFixture 生成一个最小的 windows amd64 PE 作为测试样本（纯 Go 交叉编译）
func buildWindowsPEFixture(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(os.TempDir(), "EasyTools", "icon_test")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(dir, "fixture.go")
	if err := os.WriteFile(src, []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "fixture.exe")
	cmd := exec.Command("go", "build", "-o", out, src)
	cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0")
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("构建 PE 样本失败: %v\n%s", err, b)
	}
	return out
}

// 构造一个最小合法 ICO：ICONDIR + 单目录项 + PNG 载荷
func wrapPNGAsICO(t *testing.T, pngBytes []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	// ICONDIR: reserved=0 type=1 count=1
	buf.Write([]byte{0, 0, 1, 0, 1, 0})
	// ICONDIRENTRY: width=32 height=32 colors=0 reserved=0 planes=1 bitcount=32
	buf.Write([]byte{32, 32, 0, 0, 0, 1, 0, 32})
	// size + offset(6+16=22)
	buf.Write([]byte{byte(len(pngBytes)), 0, 0, 0})
	buf.Write([]byte{22, 0, 0, 0})
	buf.Write(pngBytes)
	return buf.Bytes()
}

func TestPresetIconPNG(t *testing.T) {
	var prev []byte
	for i, id := range BuiltinIconPresets() {
		raw, err := GeneratePresetIconPNG(id)
		if err != nil {
			t.Fatalf("GeneratePresetIconPNG(%s) 失败: %v", id, err)
		}
		if len(raw) < 100 {
			t.Fatalf("预设 %s 生成过小", id)
		}
		if !isSupportedIconBytes(raw) {
			t.Fatalf("预设 %s 输出不是可解码图片", id)
		}
		if i > 0 && bytes.Equal(prev, raw) {
			t.Fatalf("预设 %s 与上一预设字节完全一致（缺少随机抖动）", id)
		}
		prev = raw
	}
	// 同预设两次生成应因配色抖动而不同
	a, _ := GeneratePresetIconPNG("docTool")
	b, _ := GeneratePresetIconPNG("docTool")
	if bytes.Equal(a, b) {
		t.Fatal("同一预设两次生成字节一致，缺少抖动（哈希打散失效）")
	}
	// 未知预设回退默认样式且不报错
	if _, err := GeneratePresetIconPNG("no-such-preset"); err != nil {
		t.Fatalf("未知预设不应报错: %v", err)
	}
}

// TestApplyPEIcon 端到端：对真实 Windows PE 应用 PNG / ICO 图标并读回校验。
func TestApplyPEIcon(t *testing.T) {
	fixture := buildWindowsPEFixture(t)
	pe, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}

	pngBytes, err := GeneratePresetIconPNG("svcMon")
	if err != nil {
		t.Fatal(err)
	}
	icoBytes := wrapPNGAsICO(t, pngBytes)

	for name, raw := range map[string][]byte{
		"PNG直供": pngBytes,
		"ICO容器": icoBytes,
	} {
		out, err := ApplyPEIcon(pe, raw)
		if err != nil {
			t.Fatalf("[%s] ApplyPEIcon 失败: %v", name, err)
		}
		if len(out) <= len(pe) {
			t.Fatalf("[%s] 输出未增长（资源未写入）", name)
		}
		if out[0] != 'M' || out[1] != 'Z' {
			t.Fatalf("[%s] 输出不再是 PE", name)
		}
		// 读回资源集，确认图标组资源存在
		rs, err := winres.LoadFromEXE(bytes.NewReader(out))
		if err != nil {
			t.Fatalf("[%s] 读回资源失败: %v", name, err)
		}
		icon, err := rs.GetIcon(winres.ID(1))
		if err != nil {
			t.Fatalf("[%s] GetIcon(1) 失败: %v", name, err)
		}
		if icon == nil {
			t.Fatalf("[%s] 图标资源为空", name)
		}
		t.Logf("[%s] 图标资源写入成功 (输出 %d -> %d 字节)", name, len(pe), len(out))
	}

	// 空图标/非 PE 原样返回与报错
	noop, err := ApplyPEIcon(pe, nil)
	if err != nil || !bytes.Equal(noop, pe) {
		t.Fatal("空图标应原样返回")
	}
	if _, err := ApplyPEIcon([]byte("not a pe file at all!!!"), pngBytes); err == nil {
		t.Fatal("非 PE 输入应报错")
	}
}

// TestReplacePEIconFile 文件级原子替换
func TestReplacePEIconFile(t *testing.T) {
	fixture := buildWindowsPEFixture(t)
	tmp := filepath.Join(filepath.Dir(fixture), "replace_target.exe")
	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		t.Fatal(err)
	}

	pngBytes, err := GeneratePresetIconPNG("panelTool")
	if err != nil {
		t.Fatal(err)
	}
	if err := ReplacePEIconFile(tmp, pngBytes); err != nil {
		t.Fatalf("ReplacePEIconFile 失败: %v", err)
	}
	// 替换后文件仍可被资源解析
	out, err := os.ReadFile(tmp)
	if err != nil {
		t.Fatal(err)
	}
	rs, err := winres.LoadFromEXE(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("替换后解析资源失败: %v", err)
	}
	icon, err := rs.GetIcon(winres.ID(1))
	if err != nil || icon == nil {
		t.Fatalf("替换后未读到图标: %v", err)
	}

	// Windows 本机：替换图标后的 exe 仍可正常启动
	if runtime.GOOS == "windows" {
		run := exec.Command(tmp)
		if err := run.Run(); err != nil {
			t.Fatalf("替换图标后的 exe 无法运行: %v", err)
		}
	}
	os.Remove(tmp)
}

// TestResolveIconBytes 前端配置解析（base64 / dataURL / 预设 / 非法输入）
func TestResolveIconBytes(t *testing.T) {
	pngBytes, _ := GeneratePresetIconPNG("docTool")
	b64 := base64.StdEncoding.EncodeToString(pngBytes)

	if raw, err := ResolveIconBytes(&IconConfig{Preset: "docTool"}); err != nil || len(raw) == 0 {
		t.Fatalf("预设解析失败: %v", err)
	}
	if raw, err := ResolveIconBytes(&IconConfig{Data: b64}); err != nil || !bytes.Equal(raw, pngBytes) {
		t.Fatalf("base64 解析失败: %v", err)
	}
	if raw, err := ResolveIconBytes(&IconConfig{Data: "data:image/png;base64," + b64}); err != nil || !bytes.Equal(raw, pngBytes) {
		t.Fatalf("dataURL 解析失败: %v", err)
	}
	if _, err := ResolveIconBytes(&IconConfig{Data: "@@@not-base64@@@"}); err == nil {
		t.Fatal("非法 base64 应报错")
	}
	if raw, err := ResolveIconBytes(nil); err != nil || raw != nil {
		t.Fatal("空配置应返回 nil")
	}
}

// TestBuiltinIconLibrary 内置图标库（原 ResourceHacker 资产保留的 ico）可枚举可读取
func TestBuiltinIconLibrary(t *testing.T) {
	names, err := ListBuiltinIconFiles()
	if err != nil {
		t.Fatalf("列出内置图标失败: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("内置图标库为空")
	}
	for _, n := range names {
		if !ValidBuiltinIconName(n) {
			t.Fatalf("列表包含非法文件名: %s", n)
		}
		raw, err := ReadBuiltinIconFile(n)
		if err != nil {
			t.Fatalf("读取内置图标 %s 失败: %v", n, err)
		}
		if len(raw) < 22 || raw[0] != 0 || raw[1] != 0 || raw[2] != 1 || raw[3] != 0 {
			t.Fatalf("内置图标 %s 不是合法 ICO", n)
		}
	}

	// 库图标可经 ResolveIconBytes 的 IconName 分支解析
	raw, err := ResolveIconBytes(&IconConfig{IconName: names[0]})
	if err != nil || len(raw) == 0 {
		t.Fatalf("IconName 解析失败: %v", err)
	}
	// 路径穿越/非法名被拒绝
	for _, bad := range []string{"../loader1/main.tpl", "..\\secret.ico", "x.exe", "x.ico.bak", ".."} {
		if ValidBuiltinIconName(bad) {
			t.Fatalf("非法名 %q 未被拦截", bad)
		}
	}
	if _, err := ReadBuiltinIconFile("../main.go"); err == nil {
		t.Fatal("路径穿越读取应被拒绝")
	}
	t.Logf("内置图标库共 %d 个: %v", len(names), names)
}

// TestBuiltinIconApply 库图标直接应用于 Windows PE（生成链路真实路径）
func TestBuiltinIconApply(t *testing.T) {
	fixture := buildWindowsPEFixture(t)
	pe, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	names, err := ListBuiltinIconFiles()
	if err != nil || len(names) == 0 {
		t.Fatal("图标库为空")
	}
	raw, err := ReadBuiltinIconFile(names[0])
	if err != nil {
		t.Fatal(err)
	}
	out, err := ApplyPEIcon(pe, raw)
	if err != nil {
		t.Fatalf("应用库图标失败: %v", err)
	}
	rs, err := winres.LoadFromEXE(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("读回资源失败: %v", err)
	}
	icon, err := rs.GetIcon(winres.ID(1))
	if err != nil || icon == nil {
		t.Fatalf("未读到应用后的图标: %v", err)
	}
	t.Logf("库图标 %s 应用成功 (%d -> %d 字节)", names[0], len(pe), len(out))
}
