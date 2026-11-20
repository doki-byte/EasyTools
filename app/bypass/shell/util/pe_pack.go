package util

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
)

// PE 加壳（packer）识别。
//
// 背景（实测结论，勿删）：
// UPX 加壳的 PE 会**剥离重定位表**（DataDirectory[5] = 0），且其 UPX stub 依赖
// "自身是被系统正常映射的镜像"这一前提（PEB 模块表 / 镜像映射语义）才能定位并解压
// 原始数据。经 Donut 反射式内存加载后：
//   - 没有重定位表 → 无法重定位；
//   - 内存区是 VirtualAlloc 私有区、不在 PEB 模块表中；
//   - 结果：loader 一旦执行到该 PIC 就以 0xC0000005 崩溃。
//
// 对照实验（同一份标记载荷，仅差一次 UPX 加壳，最小 runner 直接执行 PIC）：
//   未加壳 marker（3.09MB, 有重定位表）→ 载荷正常执行
//   同一文件 UPX 加壳（1.20MB, 重定位表为空）→ 0xC0000005 崩溃
// 且已排除：loader 模板差异、宿主镜像基址冲突（0x10000000 基址宿主同样崩）、
// Donut 的 HEADERS_OVERWRITE/KEEP、以及载荷体积（9.4MB 未加壳载荷可正常执行）。
//
// 因此在"处理 bin 文件"阶段就必须拒绝加壳输入并给出可执行指引，
// 否则会生成一个"编译成功但一跑就崩"的产物，用户无从判断原因。

// PEPackInfo 判断 PE 是否被已知壳打包。
// 返回 packed 与可读的细节描述（用于直接展示给用户）。
func PEPackInfo(data []byte) (bool, string) {
	names := peSectionNames(data)
	var upx []string
	for _, n := range names {
		if strings.HasPrefix(strings.ToUpper(n), "UPX") {
			upx = append(upx, n)
		}
	}
	if len(upx) > 0 {
		return true, fmt.Sprintf("UPX（段: %s）", strings.Join(upx, ","))
	}
	// 兜底：部分 UPX 变体会改名段，但文件内仍保留 "UPX!" 标记
	if bytes.Contains(data, []byte("UPX!")) {
		return true, "UPX（含 UPX! 标记）"
	}
	return false, ""
}

// relocDirSize 返回 PE 重定位目录大小（字节）；非 PE 或解析失败返回 -1。
//
// 用途：UPX 等加壳器会剥离重定位表（size=0），而加壳镜像无法被反射加载；
// 反之正常脱壳/编译产物必然带重定位目录，可作为"是否完整脱壳"的快速判据。
func relocDirSize(data []byte) int {
	if len(data) < 0x40 || data[0] != 'M' || data[1] != 'Z' {
		return -1
	}
	peOff := int(binary.LittleEndian.Uint32(data[0x3C:]))
	if peOff <= 0 || peOff+24 > len(data) || !bytes.Equal(data[peOff:peOff+4], []byte("PE\x00\x00")) {
		return -1
	}
	magic := binary.LittleEndian.Uint16(data[peOff+24:])
	ddOff := peOff + 24 + 96
	if magic == 0x20b {
		ddOff = peOff + 24 + 112
	}
	reloc := ddOff + 5*8
	if reloc+8 > len(data) {
		return -1
	}
	// 目录项为 (RVA, Size)；此处返回 Size（加壳剥离后为 0）
	return int(binary.LittleEndian.Uint32(data[reloc+4:]))
}

// peSectionNames 解析 PE 段名（解析失败返回 nil）。
func peSectionNames(data []byte) []string {
	if len(data) < 0x40 || data[0] != 'M' || data[1] != 'Z' {
		return nil
	}
	peOff := int(binary.LittleEndian.Uint32(data[0x3C:]))
	if peOff <= 0 || peOff+24 > len(data) || !bytes.Equal(data[peOff:peOff+4], []byte("PE\x00\x00")) {
		return nil
	}
	numSec := int(binary.LittleEndian.Uint16(data[peOff+6:]))
	optSize := int(binary.LittleEndian.Uint16(data[peOff+20:]))
	if numSec <= 0 || numSec > 96 { // PE 规范上限 96 段
		return nil
	}
	secTab := peOff + 24 + optSize
	const secSize = 40
	names := make([]string, 0, numSec)
	for i := 0; i < numSec; i++ {
		off := secTab + i*secSize
		if off+secSize > len(data) {
			break
		}
		raw := data[off : off+8]
		if idx := bytes.IndexByte(raw, 0); idx >= 0 {
			raw = raw[:idx]
		}
		names = append(names, string(raw))
	}
	return names
}
