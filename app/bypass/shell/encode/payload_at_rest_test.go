package encode

import (
	"bytes"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestPayloadAtRestRoundTrip 载荷混淆落盘/读回必须完全一致，
// 且磁盘上不得出现原始载荷字节（避免被安全软件静态识别）。
func TestPayloadAtRestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	// 构造带明显特征的"载荷"（模拟 Donut PIC / shellcode 明文头）
	plain := make([]byte, 64*1024)
	copy(plain, []byte("MZ_THIS_IS_A_FAKE_SHELLCODE_PAYLOAD_MARKER"))
	if _, err := rand.Read(plain[64:]); err != nil {
		t.Fatal(err)
	}

	binPath, err := WritePayloadAtRest(dir, plain)
	if err != nil {
		t.Fatalf("WritePayloadAtRest: %v", err)
	}
	if filepath.Base(binPath) != PayloadBinName {
		t.Fatalf("载荷文件名异常: %s", binPath)
	}
	if _, err := os.Stat(filepath.Join(dir, PayloadKeyName)); err != nil {
		t.Fatalf("密钥文件未生成: %v", err)
	}

	onDisk, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(onDisk, plain) {
		t.Fatal("磁盘上的载荷仍为明文")
	}
	if bytes.Contains(onDisk, []byte("MZ_THIS_IS_A_FAKE_SHELLCODE_PAYLOAD_MARKER")) {
		t.Fatal("磁盘载荷中仍可检索到原始明文特征")
	}

	got, err := ReadPayloadAtRest(binPath)
	if err != nil {
		t.Fatalf("ReadPayloadAtRest: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("读回内容不一致: got=%d want=%d", len(got), len(plain))
	}
}

// TestPayloadAtRestLegacyPlaintext 无密钥文件时必须按旧流程当明文读取（兼容历史工作目录）。
func TestPayloadAtRestLegacyPlaintext(t *testing.T) {
	dir := t.TempDir()
	plain := []byte("LEGACY-PLAINTEXT-PAYLOAD")
	path := filepath.Join(dir, PayloadBinName)
	if err := os.WriteFile(path, plain, 0644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadPayloadAtRest(path)
	if err != nil {
		t.Fatalf("读取旧格式载荷失败: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("旧格式载荷读回不一致: %q", string(got))
	}
}

// TestReadPayloadAtRestRetry 文件缺失时应重试后返回错误（不 panic、不无限等待）。
func TestReadPayloadAtRestRetry(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "not-exists.bin")
	start := time.Now()
	if _, err := ReadPayloadAtRestRetry(missing, 3, 10*time.Millisecond); err == nil {
		t.Fatal("文件缺失时应返回错误")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("重试耗时异常: %v", elapsed)
	}
}

// TestOpenBinToByteReadsFully 回归：OpenBinToByte 曾经只调用一次 file.Read，
// 大文件可能短读且 err==nil，导致载荷尾部被静默补零。这里校验读满与长度校验。
func TestOpenBinToByteReadsFully(t *testing.T) {
	dir := t.TempDir()
	data := make([]byte, 8<<20) // 8MB，覆盖大文件路径
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}
	// 尾部放特征，短读时最容易丢
	copy(data[len(data)-16:], []byte("TAIL-MARKER-16B!!"))

	path := filepath.Join(dir, "big.bin")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	got, err := OpenBinToByte(path)
	if err != nil {
		t.Fatalf("OpenBinToByte: %v", err)
	}
	if len(got) != len(data) {
		t.Fatalf("读取长度不符: got=%d want=%d", len(got), len(data))
	}
	if !bytes.Equal(got, data) {
		t.Fatal("读取内容与源文件不一致（存在短读/补零）")
	}
}
