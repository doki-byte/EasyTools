package util

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
)

// PE 常量
const (
	peSignature      = "PE\x00\x00"
	subsystemConsole = uint16(0x0003)
	subsystemGUI     = uint16(0x0002)
)

// 可选头里字段偏移（相对 Optional Header 起始）
const (
	optionalHeaderMagicOffset    = 0x00
	optionalHeaderCheckSumOffset = 0x40
	// 注意：在标准 PE32 / PE32+ 结构中，Subsystem 字段相对 Optional Header 起始的偏移都是 0x44
	optionalHeaderSubsystemOffset = 0x44
)

// CloseWindowsConsole 将 PE 可执行文件的 Subsystem 从 Console 修改为 GUI，并修复校验和
func CloseWindowsConsole(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	subsystemOffset, is64, err := locateSubsystemAndCheckSumOffsets(data)
	if err != nil {
		return err
	}

	// 当前 Subsystem
	currentSubsystem := binary.LittleEndian.Uint16(data[subsystemOffset:])
	fmt.Printf("当前Subsystem值: 0x%04X\n", currentSubsystem)

	if currentSubsystem == subsystemGUI {
		fmt.Println("Subsystem 已经是 0x0002 (GUI)，无需修改。")
		return nil
	}

	if currentSubsystem != subsystemConsole {
		fmt.Printf("警告: 非典型 Subsystem 值 (0x%04X)，仍将尝试改为 0x0002。\n", currentSubsystem)
	}

	// 修改为 GUI
	binary.LittleEndian.PutUint16(data[subsystemOffset:], subsystemGUI)
	fmt.Println("Subsystem已修改为: 0x0002")

	// 重新计算校验和
	checkSumOffset := getCheckSumOffset(is64, data)
	if checkSumOffset == 0 {
		return errors.New("无法定位 CheckSum 字段")
	}

	// 先把校验和字段清零再计算
	for i := 0; i < 4; i++ {
		data[checkSumOffset+uint32(i)] = 0
	}

	newSum := calcPEChecksum(data)
	binary.LittleEndian.PutUint32(data[checkSumOffset:], newSum)
	fmt.Println("校验和已修复")

	// 直接覆盖原文件
	if err := os.WriteFile(path, data, 0o666); err != nil {
		return fmt.Errorf("写回文件失败: %w", err)
	}

	return nil
}

// 定位 Subsystem 和 CheckSum 的偏移
func locateSubsystemAndCheckSumOffsets(data []byte) (subsystemOffset uint32, is64 bool, err error) {
	// 1. DOS 头：0x3C 处 4 字节是 PE 头偏移
	if len(data) < 0x40 {
		return 0, false, errors.New("文件过小，非合法 PE")
	}

	peOffset := binary.LittleEndian.Uint32(data[0x3C:])
	if int(peOffset)+4 > len(data) {
		return 0, false, errors.New("PE 头偏移超出范围")
	}

	// 2. 校验 "PE\0\0"
	if string(data[peOffset:peOffset+4]) != peSignature {
		return 0, false, errors.New("未找到 PE 签名")
	}

	// 3. PE 头结构：
	//   DWORD Signature;
	//   IMAGE_FILE_HEADER fileHeader; (20 字节)
	//   IMAGE_OPTIONAL_HEADER32/64 optionalHeader;
	fileHeaderOffset := peOffset + 4
	optionalHeaderOffset := fileHeaderOffset + 20

	if int(optionalHeaderOffset+2) > len(data) {
		return 0, false, errors.New("可选头偏移超出范围")
	}

	magic := binary.LittleEndian.Uint16(data[optionalHeaderOffset+optionalHeaderMagicOffset:])
	switch magic {
	case 0x10b: // PE32
		is64 = false
	case 0x20b: // PE32+
		is64 = true
	default:
		return 0, false, fmt.Errorf("未知 OptionalHeader Magic: 0x%04X", magic)
	}

	// 对于 PE32 和 PE32+，Subsystem 字段相对 Optional Header 起始偏移均为 0x44
	subsystemOffset = optionalHeaderOffset + optionalHeaderSubsystemOffset

	if int(subsystemOffset+2) > len(data) {
		return 0, false, errors.New("Subsystem 偏移超出范围")
	}

	return subsystemOffset, is64, nil
}

// 根据是否 32/64 位，得到 CheckSum 偏移（相对整个文件）
func getCheckSumOffset(is64 bool, data []byte) uint32 {
	if len(data) < 0x40 {
		return 0
	}
	peOffset := binary.LittleEndian.Uint32(data[0x3C:])
	fileHeaderOffset := peOffset + 4
	optionalHeaderOffset := fileHeaderOffset + 20
	checkSumOffset := optionalHeaderOffset + optionalHeaderCheckSumOffset
	if int(checkSumOffset+4) > len(data) {
		return 0
	}
	return checkSumOffset
}

// 按微软 PE 规范计算校验和（与 MapFileAndCheckSum 兼容算法）
func calcPEChecksum(data []byte) uint32 {
	var (
		sum      uint32
		dataLen  = len(data)
		i        int
		word     uint16
		fileSize = uint32(dataLen)
	)

	// 按 16 位累加
	for i = 0; i+1 < dataLen; i += 2 {
		word = binary.LittleEndian.Uint16(data[i:])
		sum += uint32(word)
		sum = (sum & 0xFFFF) + (sum >> 16)
	}

	// 若长度为奇数，最后一个字节按高位补 0 处理
	if i < dataLen {
		word = uint16(data[i])
		sum += uint32(word)
		sum = (sum & 0xFFFF) + (sum >> 16)
	}

	// 再折叠一次
	sum = (sum & 0xFFFF) + (sum >> 16)
	sum = sum & 0xFFFF

	// 按规范加上文件长度
	return sum + fileSize
}
