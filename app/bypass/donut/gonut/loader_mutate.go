package gonut

import (
	crand "crypto/rand"
)

const picLoaderKeyLen = 16

// xorLoaderBody 对 loader 做逐字节 rolling XOR（PIC 内二次加密，与仓库静态 obf 独立）。
func xorLoaderBody(in, key []byte) []byte {
	out := make([]byte, len(in))
	for i, b := range in {
		out[i] = b ^ key[i%len(key)]
	}
	return out
}

func randomPICLoaderKey() []byte {
	k := make([]byte, picLoaderKeyLen)
	if _, err := crand.Read(k); err != nil {
		panic(err)
	}
	return k
}

// mutateLoaderOpcodes 对 loader 做等价指令替换，削弱固定 opcode 指纹（不改变语义）。
func mutateLoaderOpcodes(loader []byte) []byte {
	out := append([]byte(nil), loader...)
	// 2-byte reg-reg xor zero → alternate encoding (33 xx → 31 xx)
	for i := 0; i+1 < len(out); i++ {
		if out[i] != 0x33 {
			continue
		}
		switch out[i+1] {
		case 0xC0, 0xC9, 0xD2, 0xDB, 0xFF, 0xED, 0xF6:
			if randomIndex(2) == 1 {
				out[i] = 0x31
			}
		}
	}
	return out
}

// prepareLoaderForPIC 返回 PIC 内加密 loader 与 x64 解密 stub（stub 尾部含 key）。
func prepareLoaderForPIC(raw []byte) (encrypted []byte, stub []byte) {
	key := randomPICLoaderKey()
	mutated := mutateLoaderOpcodes(raw)
	encrypted = xorLoaderBody(mutated, key)
	stub = assembleX64LoaderDecryptStub(len(encrypted), key)
	return encrypted, stub
}
