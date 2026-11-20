package util

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildSyntheticPE 构造一个带指定段名的极简 PE（仅用于段名解析测试）。
func buildSyntheticPE(sectionNames []string) []byte {
	const (
		dosStubSize = 0x40
		peOff       = 0x40
		optSize     = 0xF0
	)
	buf := make([]byte, peOff+24+optSize+40*len(sectionNames))
	copy(buf[0:], "MZ")
	binary.LittleEndian.PutUint32(buf[0x3C:], peOff)
	copy(buf[peOff:], "PE\x00\x00")
	binary.LittleEndian.PutUint16(buf[peOff+6:], uint16(len(sectionNames)))
	binary.LittleEndian.PutUint16(buf[peOff+20:], optSize)
	secTab := peOff + 24 + optSize
	for i, n := range sectionNames {
		copy(buf[secTab+i*40:secTab+i*40+8], n)
	}
	return buf
}

// TestPEPackInfo UPX 加壳识别回归：
// 加壳 PE 必须在"处理 bin 文件"阶段就被拦下，否则会生成一跑就崩的产物。
func TestPEPackInfo(t *testing.T) {
	cases := []struct {
		name     string
		sections []string
		want     bool
	}{
		{"普通 PE", []string{".text", ".rdata", ".data", ".reloc"}, false},
		{"UPX 三段时间", []string{"UPX0", "UPX1", "UPX2"}, true},
		{"UPX 小写", []string{"upx0", "upx1"}, true},
		{"UPX 混排", []string{".text", "UPX1", ".data"}, true},
	}
	for _, c := range cases {
		packed, detail := PEPackInfo(buildSyntheticPE(c.sections))
		if packed != c.want {
			t.Errorf("%s: packed=%v want=%v (detail=%q)", c.name, packed, c.want, detail)
			continue
		}
		if packed && detail == "" {
			t.Errorf("%s: 命中加壳但未给出细节", c.name)
		}
		t.Logf("%s → packed=%v detail=%q", c.name, packed, detail)
	}

	// 非 PE 数据不应误判、不应 panic
	for _, bad := range [][]byte{nil, {}, []byte("MZ"), []byte("not a pe at all"), make([]byte, 4096)} {
		if packed, _ := PEPackInfo(bad); packed {
			t.Errorf("非 PE 数据被误判为加壳: %q", string(bad))
		}
	}

	// "UPX!" 标记兜底（段名被改名时）
	withMagic := append(buildSyntheticPE([]string{".text", ".data"}), []byte("....UPX!....")...)
	if packed, detail := PEPackInfo(withMagic); !packed {
		t.Errorf("含 UPX! 标记的文件未被识别: %q", detail)
	}
}

// TestPEPackInfoRealFiles 用真实文件验证识别结果（文件缺失时跳过）。
// 注意：仓库里的 fscan 可能是加壳版也可能是用户脱壳后的版本，因此这里不断言加壳与否，
// 只断言"识别结果与独立读出的段名一致"（避免误报/漏报）。
func TestPEPackInfoRealFiles(t *testing.T) {
	p := findRepoGlob("fscan*.exe")
	if p == "" {
		t.Skip("仓库根目录无 fscan，跳过真实文件校验")
	}
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	packed, detail := PEPackInfo(data)

	// 独立解析段名（不复用被测函数），核对一致性
	wantPacked := false
	for _, n := range peSectionNames(data) {
		if strings.HasPrefix(strings.ToUpper(n), "UPX") {
			wantPacked = true
		}
	}
	if !wantPacked && bytes.Contains(data, []byte("UPX!")) {
		wantPacked = true
	}
	if packed != wantPacked {
		t.Errorf("fscan 加壳判定不一致: PEPackInfo=%v（%s） 段名核验=%v", packed, detail, wantPacked)
	}
	t.Logf("仓库 fscan: %s → packed=%v detail=%q", filepath.Base(p), packed, detail)

	// 自建（未加壳）载荷不应被误判
	if exe := filepath.Join(os.TempDir(), "EasyTools", "file", "marker", "marker.exe"); fileExists(exe) {
		if data, err := os.ReadFile(exe); err == nil {
			if packed, detail := PEPackInfo(data); packed {
				t.Errorf("未加壳载荷被误判为加壳: %q", detail)
			}
		}
	}
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
