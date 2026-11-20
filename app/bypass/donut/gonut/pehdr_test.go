package gonut

import "testing"

func TestParsePEHeader(t *testing.T) {
	// minimal PE32+ stub: MZ + PE at 0x80, AMD64, not DLL
	b := make([]byte, 512)
	b[0], b[1] = 'M', 'Z'
	off := 0x80
	b[0x3c] = byte(off)
	b[0x3d] = 0
	b[0x3e] = 0
	b[0x3f] = 0
	copy(b[off:], "PE\x00\x00")
	b[off+4] = 0x64
	b[off+5] = 0x86 // AMD64
	b[off+22] = 0x02 // characteristics low — executable
	opt := off + 4 + 20
	b[opt] = 0x0b
	b[opt+1] = 0x02 // PE32+

	h, err := parsePEHeader(b)
	if err != nil {
		t.Fatal(err)
	}
	if h.Machine != imageFileMachineAMD64 {
		t.Fatalf("machine=0x%x", h.Machine)
	}
	if h.IsDLL {
		t.Fatal("expected exe")
	}
}
