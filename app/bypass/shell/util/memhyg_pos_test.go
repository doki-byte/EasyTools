package util

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMemHygienePos 注入位置单测：两行 VirtualAlloc 语句后守卫应落在整条语句闭合之后
func TestMemHygienePos(t *testing.T) {
	src := "package main\n\nimport (\n\t\"unsafe\"\n\t\"time\"\n\n\t\"golang.org/x/sys/windows\"\n)\n\nfunc main() {\n" +
		"\tkernel32 := windows.NewLazyDLL(\"kernel32.dll\")\n" +
		"\tVirtualAlloc := kernel32.NewProc(\"VirtualAlloc\")\n" +
		"\tRtlCopyMemory := kernel32.NewProc(\"RtlCopyMemory\")\n" +
		"\tVirtualProtect := kernel32.NewProc(\"VirtualProtect\")\n" +
		"\tbyteArray := []byte{1, 2, 3}\n" +
		"\taddr, _, _ := VirtualAlloc.Call(0, uintptr(len(byteArray)),\n" +
		"\t\t0x1000, 0x04)\n" +
		"\t_, _, _ = RtlCopyMemory.Call(addr, (uintptr)(unsafe.Pointer(&byteArray[0])),\n" +
		"\t\tuintptr(len(byteArray)))\n" +
		"\t_, _, _ = VirtualProtect.Call(addr, uintptr(len(byteArray)), 0x20, uintptr(unsafe.Pointer(&oldProtect)))\n" +
		"\toldProtect := 0x04\n" +
		"}\n"
	out := injectMemHygiene(src)
	guardStart := strings.Index(out, "go func() {")
	if guardStart < 0 {
		t.Fatal("守卫未注入")
	}
	// 守卫应出现在 "0x1000, 0x04)" 完整闭合之后
	allocEnd := strings.Index(out, "0x1000, 0x04)")
	allocLineEnd := strings.Index(out[allocEnd:], "\n")
	if allocEnd < 0 {
		t.Fatal("alloc 语句缺失")
	}
	if guardStart < allocEnd+allocLineEnd {
		t.Fatalf("守卫插进了 alloc 语句内部: allocEnd=%d guardStart=%d", allocEnd, guardStart)
	}
	t.Log("注入位置正确：守卫位于 VirtualAlloc 语句闭合之后")
	// 记录到临时文件便于人工核对
	_ = os.WriteFile(filepath.Join(os.TempDir(), "EasyTools", "file", "memhyg_pos_check.go"), []byte(out), 0644)
}
