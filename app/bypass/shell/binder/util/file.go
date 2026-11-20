package util

import (
	"EasyTools/app/bypass/shell/binder/loader"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
)

// 1.判断文件是否存在
func FileExists(filePath string) (bool, error) {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false, fmt.Errorf("文件 %s 不存在", filePath)
	}
	return err == nil, err
}

// 3.生成go文件 bin文件 模板文件 生成的文件路径
func GenGoFile(argSlice []string) (resFilePath string) {
	loaderName := argSlice[0]
	fileName := argSlice[1]
	randomKey := argSlice[2]
	encryptPayload := argSlice[3]
	encryptFile := argSlice[4]
	resultDir := argSlice[5]
	// 根据loaderName获取指定的shellcode模板内容,然后使用Decode1string替换里面的内容即可
	loaderStr := string(loader.Modules[loaderName])

	// 替换模板文件内容
	//fullLoaderStr := strings.ReplaceAll(loaderStr, "payloadFileName", payloadFileName)
	fullLoaderStr := strings.ReplaceAll(loaderStr, "fileName", fileName)
	fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "randomKey", randomKey)
	fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "encryptPayload", encryptPayload)
	fullLoaderStr = strings.ReplaceAll(fullLoaderStr, "encryptFile", encryptFile)

	// 使用随机go名字
	randomString, _ := GenerateRandomString(10)
	outputFileName := randomString + ".go"
	resultFilePath := filepath.Join(resultDir, outputFileName)
	err := os.WriteFile(resultFilePath, []byte(fullLoaderStr), 0644)
	if err != nil {
		//gologger.Error().Msgf("文件生成失败: %s\n", err)
		return resFilePath
	}
	//gologger.Info().Msgf("文件生成成功: %s\n", resultFilePath)
	return resultFilePath
}

// 4.生成随机字符串
func GenerateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes)[:length], nil
}
