package util

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInlinePayloadDataMissingData 回归测试：
// 生成源码中存在 //__DATA_*__ 占位符但对应数据文件缺失/为空时，
// 必须返回错误中止生成，而不是静默内联 []byte{}（否则编译成功却带着
// 空载荷，运行期必然 panic/永不执行）。
func TestInlinePayloadDataMissingData(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "EasyTools", "file")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	orig := filepath.Join(dir, "AesEncrypted.txt")
	backupData, hadFile := []byte(nil), false
	if b, err := os.ReadFile(orig); err == nil {
		backupData, hadFile = b, true
	}
	restore := func() {
		_ = os.Remove(orig)
		if hadFile {
			_ = os.WriteFile(orig, backupData, 0644)
		}
	}
	defer restore()

	source := "byteArray, _ := AesDecrypt(//__DATA_AesEncrypted__, //__DATA_AesKey__)"

	// 场景 1：文件缺失
	_ = os.Remove(orig)
	if _, err := InlinePayloadData(source); err == nil {
		t.Fatal("数据文件缺失时应返回错误，而不是静默生成空载荷")
	}

	// 场景 2：文件存在但为空
	if err := os.WriteFile(orig, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := InlinePayloadData(source); err == nil {
		t.Fatal("数据文件为空时应返回错误，而不是静默生成空载荷")
	}
}

// TestMutateIdentifiersKeepsStringLiterals 回归测试：
// 标识符随机化只能作用于代码标识符，绝不能把字符串字面量里的内容
// （例如 "kernel32.dll" / "ntdll.dll"）一并改名，否则生成的 loader
// NewLazyDLL("随机名.dll") 在运行期必然加载失败、载荷永不执行。
func TestMutateIdentifiersKeepsStringLiterals(t *testing.T) {
	src := `package main

import "golang.org/x/sys/windows"

func main() {
	kernel32 := windows.NewLazyDLL("kernel32.dll")
	ntdll := windows.NewLazyDLL("ntdll.dll")
	user32 := windows.NewLazyDLL("User32.dll")
	proc, _, _ := kernel32.NewProc("VirtualAlloc").Call(0, 0x1000, 0x3000, 0x40)
	_, _, _ = ntdll.NewProc("NtQueryInformationProcess").Call(0)
	_ = user32
	_ = proc
}
`
	out := MutateIdentifiers(src)

	// 字符串字面量必须原样保留
	for _, want := range []string{
		`"kernel32.dll"`,
		`"ntdll.dll"`,
		`"User32.dll"`,
		`"VirtualAlloc"`,
		`"NtQueryInformationProcess"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("变异后字符串字面量被破坏，丢失 %s\n生成结果:\n%s", want, out)
		}
	}

	// 注释内容也必须原样保留
	withComment := "// kernel32 与 ntdll 加载\n" + src
	out2 := MutateIdentifiers(withComment)
	if !strings.Contains(out2, "// kernel32 与 ntdll 加载") {
		t.Errorf("变异后注释被破坏:\n%s", out2)
	}
}
