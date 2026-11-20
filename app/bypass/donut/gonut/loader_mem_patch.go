package gonut

import "bytes"

const (
	pageReadWrite        = 0x04
	pageExecuteRead      = 0x20
	pageExecuteReadWrite = 0x40
)

var (
	patchVirtualAllocStackArg   = []byte{0xC7, 0x44, 0x24, 0x20, pageExecuteReadWrite, 0x00, 0x00, 0x00}
	patchVirtualAllocStackArgRW = []byte{0xC7, 0x44, 0x24, 0x20, pageReadWrite, 0x00, 0x00, 0x00}
	patchVirtualAllocRegArg     = []byte{0x41, 0xB8, pageExecuteReadWrite, 0x00, 0x00, 0x00}
	patchVirtualAllocRegArgRW   = []byte{0x41, 0xB8, pageReadWrite, 0x00, 0x00, 0x00}
	memCommitReserve            = []byte{0xBA, 0x1F, 0x00, 0x0F, 0x00} // MEM_COMMIT|MEM_RESERVE
)

// patchLoaderMemoryHardening 仅在 VirtualAlloc 调用点附近把 RWX 改为 RW，避免误改 mov r8d,64 等常量。
func patchLoaderMemoryHardening(loader []byte) []byte {
	out := append([]byte(nil), loader...)
	patchVirtualAllocStackSites(out)
	patchVirtualAllocRegSites(out)
	return out
}

func patchVirtualAllocStackSites(out []byte) {
	for i := 0; i+len(patchVirtualAllocStackArg)+len(memCommitReserve) <= len(out); i++ {
		if !bytes.Equal(out[i:i+len(patchVirtualAllocStackArg)], patchVirtualAllocStackArg) {
			continue
		}
		j := i + len(patchVirtualAllocStackArg)
		if !bytes.Equal(out[j:j+len(memCommitReserve)], memCommitReserve) {
			continue
		}
		copy(out[i:i+len(patchVirtualAllocStackArg)], patchVirtualAllocStackArgRW)
	}
}

func patchVirtualAllocRegSites(out []byte) {
	for i := 0; i+len(patchVirtualAllocRegArg) <= len(out); i++ {
		if !bytes.Equal(out[i:i+len(patchVirtualAllocRegArg)], patchVirtualAllocRegArg) {
			continue
		}
		if !looksLikeVirtualAllocCall(out, i+len(patchVirtualAllocRegArg)) {
			continue
		}
		copy(out[i:i+len(patchVirtualAllocRegArg)], patchVirtualAllocRegArgRW)
	}
}

func looksLikeVirtualAllocCall(out []byte, from int) bool {
	limit := from + 12
	if limit > len(out) {
		limit = len(out)
	}
	for j := from; j+3 <= limit; j++ {
		// ff 53 60 / ff 96 xx xx xx xx — loader 经 instance 表/indirect 调 VirtualAlloc
		if out[j] == 0xFF && (out[j+1] == 0x53 || out[j+1] == 0x96) {
			return true
		}
	}
	return false
}

func prepareX64LoaderBody(raw []byte, cfg *Config) []byte {
	body := append([]byte(nil), raw...)
	if cfg != nil && cfg.LoaderMemHarden {
		body = patchLoaderMemoryHardening(body)
	}
	return body
}
