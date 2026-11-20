package gonut

import (
	"bytes"
	"testing"
)

func TestLoaderPICXorRoundtrip(t *testing.T) {
	raw := LoaderEXE64()
	if len(raw) < 64 {
		t.Fatal("loader too short")
	}
	key := randomPICLoaderKey()
	mutated := mutateLoaderOpcodes(raw)
	enc := xorLoaderBody(mutated, key)
	dec := xorLoaderBody(enc, key)
	if !bytes.Equal(dec, mutated) {
		t.Fatal("xor roundtrip mismatch")
	}
}

func TestAssembleDecryptStubLayout(t *testing.T) {
	key := randomPICLoaderKey()
	stub := assembleX64LoaderDecryptStub(4096, key)
	if len(stub) < 48+picLoaderKeyLen {
		t.Fatalf("stub too short: %d", len(stub))
	}
	gotKey := stub[len(stub)-picLoaderKeyLen:]
	if !bytes.Equal(gotKey, key) {
		t.Fatal("stub tail key mismatch")
	}
	// mov r9, rcx
	if stub[0] != 0x49 || stub[1] != 0x89 || stub[2] != 0xC9 {
		t.Fatalf("stub must start with mov r9, rcx (49 89 C9), got % x", stub[:3])
	}
	// lea rax,[rip+0] — 不得再用 call/pop 破坏栈
	if !bytes.Contains(stub, []byte{0x48, 0x8D, 0x05}) {
		t.Fatal("stub missing lea rax,[rip+disp32]")
	}
	if bytes.Contains(stub, []byte{0xE8, 0x00, 0x00, 0x00, 0x00, 0x58}) {
		t.Fatal("stub must not use call/pop for PIC base")
	}
}

func TestDecryptStubJccTarget(t *testing.T) {
	stub := assembleX64LoaderDecryptStub(100, randomPICLoaderKey())
	for i := 0; i < len(stub)-1; i++ {
		if stub[i] != 0x75 {
			continue
		}
		target := i + 2 + int(int8(stub[i+1]))
		if target < 0 || target+1 >= len(stub) || stub[target] != 0x8A || stub[target+1] != 0x07 {
			t.Fatalf("jnz at %d targets invalid %d (% x)", i, target, stub[max(0, target):min(len(stub), target+4)])
		}
		return
	}
	t.Fatal("decrypt stub missing jnz loop")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestPrepareLoaderForPIC(t *testing.T) {
	raw := LoaderEXE64()
	enc, stub := prepareLoaderForPIC(raw)
	if len(enc) != len(raw) {
		t.Fatalf("encrypted len %d != raw %d", len(enc), len(raw))
	}
	key := stub[len(stub)-picLoaderKeyLen:]
	dec := xorLoaderBody(enc, key)
	reenc := xorLoaderBody(dec, key)
	if !bytes.Equal(reenc, enc) {
		t.Fatal("re-xor mismatch")
	}
}
