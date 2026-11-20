package gonut

import (
	"bytes"
	crand "crypto/rand"
	"encoding/binary"
	"io"
	"reflect"
	"strings"
	"unsafe"
)

func GenRandomString(length int) string {
	if length <= 0 {
		return ""
	}
	out := make([]byte, length)
	if _, err := crand.Read(out); err != nil {
		panic(err)
	}
	// 动态映射到可打印 ASCII，无固定字母表常量
	const base = 33
	const span = 90
	for i := range out {
		out[i] = byte(base + int(out[i])%span)
	}
	return string(out)
}

// GenRandomBytes Generates pseudo-random bytes.
// donut/donut.c
// static int gen_random(void *buf, uint64_t len) { ... }
func GenRandomBytes(n int) []byte {
	b := make([]byte, n)
	if _, err := io.ReadFull(crand.Reader, b); err != nil {
		panic(err)
	}
	return b
}

// shuffleSemicolonList 随机打乱分号分隔列表顺序（内容不变，削弱固定字符串特征）。
func shuffleSemicolonList(s string) string {
	parts := strings.Split(s, ";")
	if len(parts) <= 1 {
		return s
	}
	for i := len(parts) - 1; i > 0; i-- {
		j := randomIndex(i + 1)
		parts[i], parts[j] = parts[j], parts[i]
	}
	return strings.Join(parts, ";")
}

func BytesToStruct(buf []byte, v any) error {
	return binary.Read(bytes.NewReader(buf), binary.LittleEndian, v)
}

func StructToBytes(v any) []byte {
	buffer := bytes.NewBuffer(nil)
	if err := binary.Write(buffer, binary.LittleEndian, v); err != nil {
		panic(err)
	}
	return buffer.Bytes()
}

func UnsafeStructToBytes(ptr any) []byte {
	v := reflect.ValueOf(ptr)

	if v.Kind() != reflect.Pointer {
		panic("need a pointer")
	}

	return *(*[]byte)(unsafe.Pointer(
		&reflect.SliceHeader{
			Data: v.Pointer(),
			Len:  int(v.Elem().Type().Size()),
			Cap:  int(v.Elem().Type().Size()),
		},
	))
}
