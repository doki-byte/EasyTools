package gonut

import (
	crand "crypto/rand"
	"encoding/binary"

	"EasyTools/app/bypass/donut/piclayout"
)

// x64 栈对齐：与 Donut LOADER_EXE_X64_RSP_ALIGN 一致（内部 call 跳到 loader）。
var x64AlignBlock = []byte{
	0x55,
	0x48, 0x89, 0xE5,
	0x48, 0x83, 0xE4, 0xF0,
	0x48, 0x83, 0xEC, 0x20,
	0xE8, 0x05, 0x00, 0x00, 0x00,
	0x48, 0x89, 0xEC,
	0x5D,
	0xC3,
}

var x64AlignVariants = [][]byte{x64AlignBlock}

var loaderJunkPool = [][]byte{
	{},
	{0x90},
	{0x66, 0x90},
	{0x0F, 0x1F, 0x00},
	{0x0F, 0x1F, 0x40, 0x00},
	{0x0F, 0x1F, 0x44, 0x00, 0x00},
	{0x40, 0x90},
	{0x48, 0x87, 0xC0},
}

func randomIndex(max int) int {
	if max <= 1 {
		return 0
	}
	var buf [8]byte
	if _, err := crand.Read(buf[:]); err != nil {
		return 0
	}
	return int(binary.LittleEndian.Uint64(buf[:]) % uint64(max))
}

func pickLoaderJunk() []byte {
	return loaderJunkPool[randomIndex(len(loaderJunkPool))]
}

func pickX64Align() []byte {
	return x64AlignVariants[randomIndex(len(x64AlignVariants))]
}

// writeX64InstancePtr 用 LEA 等 PIC 指令取得 instance 指针写入 RCX，避免 E8 入口特征。
// 通用 loader 仍只需 VirtualAlloc + CreateThread(base)，从 offset 0 执行即可。
func writeX64InstancePtr(pl *PicGenerator) {
	v := piclayout.X64PrologueVariants[randomIndex(len(piclayout.X64PrologueVariants))]
	pl.PutBytes(v.Prefix)
}

// writeDonutCallPopPrologue x86 / x96 仍用标准 call/pop（x86 PIC 约束）。
func writeDonutCallPopPrologue(pl *PicGenerator, instLen uint32) {
	pl.PutByte(0xE8)
	pl.PutUint32(instLen)
}
