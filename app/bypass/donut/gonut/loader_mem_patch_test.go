package gonut

import (
	"bytes"
	"testing"

	"EasyTools/app/bypass/donut/piclayout"
)

func TestPatchLoaderMemoryHardening(t *testing.T) {
	raw := decodeLoaderBlob(loaderEXE64Obf)
	patched := patchLoaderMemoryHardening(raw)
	if bytes.Equal(patched, raw) {
		t.Fatal("expected VirtualAlloc RWX sites to be patched")
	}
	if bytes.Contains(patched, patchVirtualAllocStackArg) {
		t.Fatal("stack VirtualAlloc arg still RWX")
	}
	// 默认 LoaderEXE64 不再自动打补丁
	got := LoaderEXE64()
	if !bytes.Equal(got, raw) {
		t.Fatal("LoaderEXE64 should return unpatched loader by default")
	}
}

func minimalAMD64PE() []byte {
	b := make([]byte, 512)
	b[0], b[1] = 'M', 'Z'
	off := 0x80
	b[0x3c] = byte(off)
	copy(b[off:], "PE\x00\x00")
	b[off+4] = 0x64
	b[off+5] = 0x86
	b[off+22] = 0x02
	opt := off + 4 + 20
	b[opt] = 0x0b
	b[opt+1] = 0x02
	return b
}

func buildTestPIC(t *testing.T, crypto bool) []byte {
	t.Helper()
	pe := minimalAMD64PE()
	c := DefaultConfig()
	c.Arch = DONUT_ARCH_X64
	c.InputByte = pe
	c.GonutCompress = GONUT_COMPRESS_NONE
	c.Entropy = DONUT_ENTROPY_DEFAULT
	c.LoaderPICCrypto = crypto
	o := New(c)
	for _, step := range []func() error{
		o.ValidateLoaderConfig,
		o.ReadFileInfo,
		o.ValidateFileInfo,
		o.BuildModule,
		o.BuildInstance,
		o.BuildLoader,
	} {
		if err := step(); err != nil {
			t.Fatal(err)
		}
	}
	if err := piclayout.Validate(o.PicData); err != nil {
		t.Fatal(err)
	}
	return o.PicData
}

func TestBuildLoaderPlaintextLoader(t *testing.T) {
	pic := buildTestPIC(t, false)
	raw := LoaderEXE64()
	if !bytes.Contains(pic, raw[:min(128, len(raw))]) {
		t.Fatal("default PIC must embed plaintext loader")
	}
}

func TestBuildLoaderUsesEncryptedLoader(t *testing.T) {
	pic := buildTestPIC(t, true)
	raw := LoaderEXE64()
	if bytes.Contains(pic, raw[:min(128, len(raw))]) {
		t.Fatal("encrypted PIC must not contain plaintext loader prefix")
	}
}

func TestPICCryptoRoundtrip(t *testing.T) {
	raw := LoaderEXE64()
	enc, stub := prepareLoaderForPIC(raw)
	key := stub[len(stub)-picLoaderKeyLen:]
	dec := xorLoaderBody(enc, key)
	if len(dec) != len(raw) {
		t.Fatalf("decrypt len %d != %d", len(dec), len(raw))
	}
	// 解密后应与变异后的 loader 一致（mutate 随机，只做 xor 往返）
	reenc := xorLoaderBody(dec, key)
	if !bytes.Equal(reenc, enc) {
		t.Fatal("pic crypto xor roundtrip failed")
	}
}
