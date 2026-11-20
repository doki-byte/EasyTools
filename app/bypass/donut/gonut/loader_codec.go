package gonut

import "sync"

func decodeLoaderBlob(enc []byte) []byte {
	key := loaderDecodeKey()
	out := make([]byte, len(enc))
	for i, b := range enc {
		out[i] = b ^ key[i%len(key)]
	}
	return out
}

var (
	loaderX86Once sync.Once
	loaderX86Buf  []byte
	loaderX64Once sync.Once
	loaderX64Buf  []byte
)

func LoaderEXE86() []byte {
	loaderX86Once.Do(func() {
		loaderX86Buf = decodeLoaderBlob(loaderEXE86Obf)
	})
	return loaderX86Buf
}

func LoaderEXE64() []byte {
	loaderX64Once.Do(func() {
		loaderX64Buf = decodeLoaderBlob(loaderEXE64Obf)
	})
	return loaderX64Buf
}
