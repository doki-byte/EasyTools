package util

import (
	"fmt"
	"strings"
	"testing"
)

// TestDbgOtpInject45 诊断用例：在"含 RtlMoveMemory 拷贝行"的模板上验证 OTP 就地解码注入。
// 模板集增删后不再硬编码编号，改为从注册表中挑选实际命中锚点的模板。
func TestDbgOtpInject45(t *testing.T) {
	anchor := "RtlMoveMemory.Call"
	var name, raw string
	for _, candidate := range allModuleNames() {
		content, ok := LoaderTemplate(candidate)
		if ok && strings.Contains(string(content), anchor) {
			name, raw = candidate, string(content)
			break
		}
	}
	if name == "" {
		t.Skipf("当前模板集没有含 %s 的模板，跳过", anchor)
	}

	out, ok := injectOtpDecode(raw)
	if !ok {
		t.Fatalf("%s 未命中 OTP 就地解码注入锚点", name)
	}
	fmt.Printf("模板=%s ok=%v\n", name, ok)
	// 展示注入后整行拷贝语句区域
	j := strings.Index(out, anchor)
	seg := out[j : min(j+700, len(out))]
	fmt.Println("=== 注入后片段 ===")
	fmt.Println(seg)
}
