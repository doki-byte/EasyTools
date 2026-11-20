package piclayout

import (
	"encoding/binary"
	"fmt"
)

const (
	// DonutInstOff 标准 Donut E8：call(5) 后 instance 起始。
	DonutInstOff = 5
)

// LooksLikeDonutPIC 是否可解析为 Donut PIC（含 x64 LEA 与 E8 等）。
func LooksLikeDonutPIC(pic []byte) bool {
	_, _, ok := Parse(pic)
	return ok
}

func validateInstanceAt(pic []byte, instOff int) (instLen int, ok bool) {
	if instOff+4 > len(pic) {
		return 0, false
	}
	instLen = int(binary.LittleEndian.Uint32(pic[instOff : instOff+4]))
	if instLen < 64 || instOff+instLen > len(pic) {
		return 0, false
	}
	return instLen, true
}

// Parse 解析 PIC：优先 x64 LEA 变体，兼容 E8 标准 Donut 与历史 E5 53 头。
func Parse(pic []byte) (instOff, instLen int, ok bool) {
	if len(pic) < 9 {
		return 0, 0, false
	}
	if off, n, okX64 := tryX64Prologue(pic); okX64 {
		return off, n, true
	}
	if pic[0] == 0xE8 {
		if n, okInst := validateInstanceAt(pic, DonutInstOff); okInst {
			return DonutInstOff, n, true
		}
	}
	if pic[0] == 0xE5 && pic[1] == 0x53 && len(pic) >= 13 {
		n := int(binary.LittleEndian.Uint32(pic[2:6]))
		if n >= 64 && pic[6] == 0x48 && pic[7] == 0x8D && 13+n <= len(pic) {
			return 13, n, true
		}
		if n >= 64 && pic[6] == 0xE8 && 11+n <= len(pic) {
			return 11, n, true
		}
	}
	return 0, 0, false
}

func Instance(pic []byte) ([]byte, bool) {
	off, n, ok := Parse(pic)
	if !ok {
		return nil, false
	}
	return pic[off : off+n], true
}

// StandardDonutEntry 是否为传统 E8@0 Donut（新生成 x64 默认不再使用）。
func StandardDonutEntry(pic []byte) bool {
	return len(pic) >= 9 && pic[0] == 0xE8
}

// Validate 校验 PIC 布局与 x64 前导 jmp 是否跳过 instance 落到 loader 对齐块。
func Validate(pic []byte) error {
	off, n, ok := Parse(pic)
	if !ok {
		if len(pic) >= 2 {
			return fmt.Errorf("donut PIC layout invalid (entry 0x%02x 0x%02x)", pic[0], pic[1])
		}
		return fmt.Errorf("donut PIC layout invalid")
	}
	if off >= 5 && pic[off-5] == 0xE9 {
		rel := int32(binary.LittleEndian.Uint32(pic[off-4 : off]))
		jmpTarget := off + int(rel)
		instEnd := off + n
		if jmpTarget != instEnd {
			return fmt.Errorf("donut PIC jmp target %d != instance end %d (off=%d len=%d)", jmpTarget, instEnd, off, n)
		}
	}
	return nil
}
