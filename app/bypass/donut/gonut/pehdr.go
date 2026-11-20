package gonut

import (
	"encoding/binary"
	"fmt"
)

const (
	imageFileDLL          = 0x2000
	imageFileMachineI386  = 0x014c
	imageFileMachineAMD64 = 0x8664
	pe32Magic             = 0x10b
	pe32PlusMagic         = 0x20b
)

type peHeaderInfo struct {
	Machine  uint16
	IsDLL    bool
	ComRVA   uint32
	ComSize  uint32
}

// parsePEHeader 仅读 DOS/COFF/Optional 头，不解析符号表。
// Go 等原生 PE 常令 debug/pe 在符号表处 EOF，但 Donut 封装原生 EXE 只需 Machine/Characteristics/COM 目录。
func parsePEHeader(data []byte) (peHeaderInfo, error) {
	var info peHeaderInfo
	if len(data) < 64 || data[0] != 'M' || data[1] != 'Z' {
		return info, fmt.Errorf("not a PE file (missing MZ header)")
	}
	peOff := int(binary.LittleEndian.Uint32(data[0x3c:0x40]))
	if peOff < 0 || peOff+24 > len(data) {
		return info, fmt.Errorf("invalid PE offset")
	}
	if string(data[peOff:peOff+4]) != "PE\x00\x00" {
		return info, fmt.Errorf("invalid PE signature")
	}
	fh := data[peOff+4:]
	info.Machine = binary.LittleEndian.Uint16(fh[0:2])
	characteristics := binary.LittleEndian.Uint16(fh[18:20])
	info.IsDLL = characteristics&imageFileDLL != 0

	optOff := peOff + 4 + 20
	if optOff+2 > len(data) {
		return info, fmt.Errorf("optional header missing")
	}
	magic := binary.LittleEndian.Uint16(data[optOff:])
	var ddOff int
	switch magic {
	case pe32Magic:
		ddOff = optOff + 96
	case pe32PlusMagic:
		ddOff = optOff + 112
	default:
		return info, fmt.Errorf("unsupported optional header magic: 0x%x", magic)
	}
	comOff := ddOff + 14*8
	if comOff+8 > len(data) {
		return info, nil
	}
	info.ComRVA = binary.LittleEndian.Uint32(data[comOff:])
	info.ComSize = binary.LittleEndian.Uint32(data[comOff+4:])
	return info, nil
}

func (h peHeaderInfo) isManaged() bool {
	return h.ComRVA != 0 && h.ComSize != 0
}
