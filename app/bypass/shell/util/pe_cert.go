package util

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
)

const peCertTableEntry = 4 // IMAGE_DIRECTORY_ENTRY_SECURITY

// CopyCertificateFromDonor 原生复制 PE 签名证书（替代 sigthief.exe，跨平台）
func CopyCertificateFromDonor(targetPath, donorPath string) error {
	target, err := os.ReadFile(targetPath)
	if err != nil {
		return fmt.Errorf("读取目标文件失败: %w", err)
	}
	donor, err := os.ReadFile(donorPath)
	if err != nil {
		return fmt.Errorf("读取 donor 文件失败: %w", err)
	}
	if !bytesIsPE(target) || !bytesIsPE(donor) {
		return fmt.Errorf("目标或 donor 不是有效 PE 文件")
	}

	donorCert, donorOff, err := extractPECertificate(donor)
	if err != nil {
		return fmt.Errorf("提取 donor 证书失败: %w", err)
	}
	if len(donorCert) == 0 {
		return fmt.Errorf("donor 文件不含数字签名")
	}
	_ = donorOff

	out, certOffset, err := appendPECertificate(target, donorCert)
	if err != nil {
		return fmt.Errorf("写入证书失败: %w", err)
	}
	_ = certOffset

	if err := os.WriteFile(targetPath, out, 0o666); err != nil {
		return err
	}
	return fixPEChecksum(targetPath)
}

// CopyCertificateFromEmbedded 使用内嵌 donor 证书
func CopyCertificateFromEmbedded(targetPath string) error {
	donorPath, err := extractEmbeddedDonorCert()
	if err != nil {
		return err
	}
	defer os.Remove(donorPath)
	return CopyCertificateFromDonor(targetPath, donorPath)
}

func extractPECertificate(data []byte) ([]byte, uint32, error) {
	off, size, err := peCertDirectory(data)
	if err != nil {
		return nil, 0, err
	}
	if size == 0 {
		return nil, 0, nil
	}
	end := int(off) + int(size)
	if end > len(data) {
		return nil, 0, fmt.Errorf("证书表越界")
	}
	cert := append([]byte(nil), data[off:end]...)
	return cert, off, nil
}

func peCertDirectory(data []byte) (offset, size uint32, err error) {
	if len(data) < 0x40 {
		return 0, 0, fmt.Errorf("文件过小")
	}
	peOff := binary.LittleEndian.Uint32(data[0x3C:])
	optOff := peOff + 4 + 20
	if int(optOff)+2 > len(data) {
		return 0, 0, fmt.Errorf("可选头越界")
	}
	magic := binary.LittleEndian.Uint16(data[optOff:])
	var certDirOff uint32
	switch magic {
	case 0x10b:
		certDirOff = optOff + 128 // PE32 data directories start + 4*8
	case 0x20b:
		certDirOff = optOff + 144 // PE32+
	default:
		return 0, 0, fmt.Errorf("未知 PE  magic")
	}
	// SECURITY directory is index 4
	secDir := certDirOff + peCertTableEntry*8
	if int(secDir)+8 > len(data) {
		return 0, 0, fmt.Errorf("证书目录越界")
	}
	offset = binary.LittleEndian.Uint32(data[secDir:])
	size = binary.LittleEndian.Uint32(data[secDir+4:])
	return offset, size, nil
}

func appendPECertificate(data, cert []byte) ([]byte, uint32, error) {
	// 移除旧证书
	existingOff, existingSize, _ := peCertDirectory(data)
	if existingSize > 0 && int(existingOff)+int(existingSize) <= len(data) {
		data = append([]byte(nil), data[:existingOff]...)
	}

	// 8 字节对齐
	for len(data)%8 != 0 {
		data = append(data, 0)
	}
	certOffset := uint32(len(data))
	out := append(data, cert...)

	// 更新证书目录
	peOff := binary.LittleEndian.Uint32(out[0x3C:])
	optOff := peOff + 4 + 20
	magic := binary.LittleEndian.Uint16(out[optOff:])
	var certDirOff uint32
	if magic == 0x10b {
		certDirOff = optOff + 128
	} else {
		certDirOff = optOff + 144
	}
	secDir := certDirOff + peCertTableEntry*8
	binary.LittleEndian.PutUint32(out[secDir:], certOffset)
	binary.LittleEndian.PutUint32(out[secDir+4:], uint32(len(cert)))
	return out, certOffset, nil
}

func extractEmbeddedDonorCert() (string, error) {
	data, err := GetEmbeddedDonorCert()
	if err != nil {
		return "", err
	}
	tmp := filepath.Join(os.TempDir(), "EasyTools", "donor_cert.exe")
	if err := os.MkdirAll(filepath.Dir(tmp), 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return "", err
	}
	return tmp, nil
}
