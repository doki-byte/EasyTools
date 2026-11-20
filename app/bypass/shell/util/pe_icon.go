package util

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/tc-hib/winres"
)

// ============================================================================
// 原生 PE 图标替换（跨平台，纯 Go，不依赖 ResourceHacker/rcedit 等 Windows exe）。
// 技术方案参考 EasyShell 的 pe_resources：winres.LoadFromEXE -> SetIcon -> WriteToEXE，
// 在 Windows/macOS/Linux 上均可对 Windows PE 产物做图标资源替换。
// ============================================================================

// IconConfig 前端下发的图标配置（嵌入 /make 请求）。
// 优先级：Data（自定义 base64）> IconName（内置图标库）> Preset（程序化预设）。
// 三字段均空 = 不更换图标（免杀生成默认不自动添加图标）。
type IconConfig struct {
	Preset   string `json:"preset"`   // 内置预设图标 id（如 "docTool"）
	IconName string `json:"iconName"` // 内置图标库文件名（如 "access.ico"）
	Data     string `json:"data"`     // 自定义图标 base64（.ico 或 .png，可带 data: 前缀）
}

const maxIconBytes = 1024 * 1024 // 1MB

// BuiltinIconPresets 内置预设图标 id 列表（前端下拉框展示用）
func BuiltinIconPresets() []string {
	return []string{"docTool", "svcMon", "syncAgent", "panelTool"}
}

// PresetIconDisplayName 预设图标的中文显示名（仅供前端文案，不参与资源生成）
func PresetIconDisplayName(id string) string {
	switch id {
	case "docTool":
		return "文档工具"
	case "svcMon":
		return "系统监控"
	case "syncAgent":
		return "数据同步"
	case "panelTool":
		return "控制面板"
	default:
		return ""
	}
}

// ResolveIconBytes 解析前端图标配置为图标原始字节：
// 优先自定义 base64 数据（.ico/.png），其次内置图标库（.ico），最后回退程序化预设 PNG。
// 未配置任何图标时返回 nil（默认不自动添加图标）。
func ResolveIconBytes(cfg *IconConfig) ([]byte, error) {
	if cfg == nil {
		return nil, nil
	}
	data := stripDataURLPrefix(cfg.Data)
	if data != "" {
		raw, err := decodeIconBase64(data)
		if err != nil {
			return nil, fmt.Errorf("图标 base64 无效: %w", err)
		}
		if len(raw) == 0 {
			return nil, errors.New("图标数据为空")
		}
		if len(raw) > maxIconBytes {
			return nil, fmt.Errorf("图标过大（最多 %d 字节）", maxIconBytes)
		}
		if !isSupportedIconBytes(raw) {
			return nil, errors.New("仅支持 .ico / .png 图标")
		}
		return raw, nil
	}
	if cfg.IconName != "" {
		raw, err := ReadBuiltinIconFile(cfg.IconName)
		if err != nil {
			return nil, err
		}
		return raw, nil
	}
	if cfg.Preset != "" {
		raw, err := GeneratePresetIconPNG(cfg.Preset)
		if err != nil {
			return nil, fmt.Errorf("内置预设图标 %q 生成失败: %w", cfg.Preset, err)
		}
		return raw, nil
	}
	return nil, nil
}

func stripDataURLPrefix(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, ","); i >= 0 && strings.HasPrefix(strings.ToLower(s), "data:") {
		return strings.TrimSpace(s[i+1:])
	}
	return s
}

func decodeIconBase64(b64 string) ([]byte, error) {
	if raw, err := base64.StdEncoding.DecodeString(b64); err == nil {
		return raw, nil
	}
	return base64.RawStdEncoding.DecodeString(b64)
}

// isSupportedIconBytes 判断图标内容是否受支持：ICO 魔数 00 00 01 00 或可解码图片（PNG 等）
func isSupportedIconBytes(raw []byte) bool {
	if len(raw) >= 4 && raw[0] == 0 && raw[1] == 0 && raw[2] == 1 && raw[3] == 0 {
		return true
	}
	if _, _, err := image.Decode(bytes.NewReader(raw)); err == nil {
		return true
	}
	return false
}

// ApplyPEIcon 将 iconBytes（.ico/.png）写入 PE 二进制并返回新文件内容。
// 输入非 PE / 图标为空时原样返回，便于流水线无脑调用。
func ApplyPEIcon(peBinary, iconBytes []byte) ([]byte, error) {
	if len(iconBytes) == 0 || len(peBinary) == 0 {
		return peBinary, nil
	}
	if len(peBinary) < 64 || peBinary[0] != 'M' || peBinary[1] != 'Z' {
		return nil, errors.New("输入不是有效的 Windows PE")
	}

	// 读取现有资源集（无 .rsrc 时返回空资源集）
	rs, err := winres.LoadFromEXE(bytes.NewReader(peBinary))
	if err != nil {
		if !errors.Is(err, winres.ErrNoResources) {
			return nil, fmt.Errorf("读取 PE 资源失败: %w", err)
		}
		rs = &winres.ResourceSet{}
	}
	if rs == nil {
		rs = &winres.ResourceSet{}
	}

	icon, err := buildWinresIcon(iconBytes)
	if err != nil {
		return nil, err
	}
	if err := rs.SetIcon(winres.ID(1), icon); err != nil {
		return nil, fmt.Errorf("设置图标资源失败: %w", err)
	}

	var out bytes.Buffer
	if err := rs.WriteToEXE(&out, bytes.NewReader(peBinary)); err != nil {
		return nil, fmt.Errorf("写入 PE 资源失败: %w", err)
	}
	return out.Bytes(), nil
}

func buildWinresIcon(raw []byte) (*winres.Icon, error) {
	// ICO 容器：ICONDIR 魔数 00 00 01 00
	if len(raw) >= 4 && raw[0] == 0 && raw[1] == 0 && raw[2] == 1 && raw[3] == 0 {
		icon, err := winres.LoadICO(bytes.NewReader(raw))
		if err != nil {
			return nil, fmt.Errorf("解析 ICO 失败: %w", err)
		}
		return icon, nil
	}
	// PNG / 其它图片格式：解码后缩放生成标准尺寸图标
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("图标解码失败（支持 .ico / .png）: %w", err)
	}
	icon, err := winres.NewIconFromResizedImage(img, nil)
	if err != nil {
		return nil, fmt.Errorf("生成图标资源失败: %w", err)
	}
	return icon, nil
}

// ReplacePEIconFile 就地替换 exe 文件的图标（原子写：临时文件 + 重命名）
func ReplacePEIconFile(exePath string, iconBytes []byte) error {
	pe, err := os.ReadFile(exePath)
	if err != nil {
		return fmt.Errorf("读取 exe 失败: %w", err)
	}
	out, err := ApplyPEIcon(pe, iconBytes)
	if err != nil {
		return err
	}
	if len(out) == len(pe) && bytes.Equal(out, pe) {
		return nil // 未做任何修改
	}
	info, err := os.Stat(exePath)
	if err != nil {
		return err
	}
	tmp := filepath.Join(filepath.Dir(exePath), "."+filepath.Base(exePath)+".icon.tmp")
	if err := os.WriteFile(tmp, out, info.Mode().Perm()); err != nil {
		return fmt.Errorf("写入临时文件失败: %w", err)
	}
	if err := os.Rename(tmp, exePath); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("替换 exe 失败: %w", err)
	}
	return nil
}

// ============================================================================
// 内置预设图标：程序化绘制 256x256 PNG（圆角色块 + 简洁图形），每次生成对配色做
// 小幅随机抖动，让相同预设产出的图标字节也不一致（打散图标/文件哈希特征）。
// ============================================================================

type iconPalette struct {
	bgTop, bgBottom color.RGBA
	edge            color.RGBA
	glyph           color.RGBA
	accent          color.RGBA
}

func GeneratePresetIconPNG(preset string) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, 256, 256))
	p := jitterPalette(presetPalette(preset))
	drawIconTile(img, p)
	drawIconGlyph(img, p, normalizePreset(preset))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func normalizePreset(id string) string {
	switch id {
	case "docTool", "document", "configTool":
		return "docTool"
	case "svcMon", "monitor", "updateHelper":
		return "svcMon"
	case "syncAgent", "sync", "dataRelay":
		return "syncAgent"
	case "panelTool", "panel", "control":
		return "panelTool"
	default:
		return "docTool"
	}
}

func presetPalette(id string) iconPalette {
	switch normalizePreset(id) {
	case "svcMon":
		return iconPalette{
			bgTop: RGBA8(0xf1, 0xf5, 0xf9), bgBottom: RGBA8(0xe2, 0xe8, 0xf0),
			edge: RGBA8(0x94, 0xa3, 0xb8), glyph: RGBA8(0x33, 0x41, 0x55), accent: RGBA8(0x64, 0x74, 0x8b),
		}
	case "syncAgent":
		return iconPalette{
			bgTop: RGBA8(0xee, 0xf2, 0xff), bgBottom: RGBA8(0xe0, 0xe7, 0xff),
			edge: RGBA8(0xa5, 0xb4, 0xfc), glyph: RGBA8(0x31, 0x2e, 0x81), accent: RGBA8(0x81, 0x8c, 0xfa),
		}
	case "panelTool":
		return iconPalette{
			bgTop: RGBA8(0xff, 0xf7, 0xed), bgBottom: RGBA8(0xff, 0xed, 0xd5),
			edge: RGBA8(0xfb, 0xa9, 0x6c), glyph: RGBA8(0x9a, 0x34, 0x12), accent: RGBA8(0xea, 0x58, 0x0c),
		}
	default: // docTool
		return iconPalette{
			bgTop: RGBA8(0xe8, 0xf4, 0xfc), bgBottom: RGBA8(0xcf, 0xe8, 0xf8),
			edge: RGBA8(0x7d, 0xc4, 0xe8), glyph: RGBA8(0x0c, 0x4a, 0x6e), accent: RGBA8(0x38, 0xbd, 0xf8),
		}
	}
}

func RGBA8(r, g, b uint8) color.RGBA { return color.RGBA{R: r, G: g, B: b, A: 0xff} }

func jitterPalette(p iconPalette) iconPalette {
	return iconPalette{
		bgTop:    shiftRGBA(p.bgTop, 10),
		bgBottom: shiftRGBA(p.bgBottom, 10),
		edge:     shiftRGBA(p.edge, 8),
		glyph:    shiftRGBA(p.glyph, 10),
		accent:   shiftRGBA(p.accent, 12),
	}
}

func shiftRGBA(c color.RGBA, delta int) color.RGBA {
	return color.RGBA{R: clampByte(int(c.R) + iconRand(-delta, delta)), G: clampByte(int(c.G) + iconRand(-delta, delta)), B: clampByte(int(c.B) + iconRand(-delta, delta)), A: c.A}
}

func clampByte(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func iconRand(min, max int) int {
	if min >= max {
		return min
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
	if err != nil {
		return 0
	}
	return min + int(n.Int64())
}

// drawIconTile 圆角背景（纵向渐变）+ 描边
func drawIconTile(img *image.RGBA, p iconPalette) {
	const (
		x0, y0 = 12, 12
		x1, y1 = 244, 244
		r      = 44
	)
	for y := 0; y < 256; y++ {
		t := float64(y) / 255
		row := lerpColor(p.bgTop, p.bgBottom, t)
		for x := 0; x < 256; x++ {
			if insideRoundRect(x, y, x0, y0, x1, y1, r) {
				img.SetRGBA(x, y, row)
			}
		}
	}
	// 描边（只画边缘一圈）
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			if insideRoundRect(x, y, x0, y0, x1, y1, r) && !insideRoundRect(x, y, x0+3, y0+3, x1-3, y1-3, r-3) {
				img.SetRGBA(x, y, p.edge)
			}
		}
	}
}

func lerpColor(a, b color.RGBA, t float64) color.RGBA {
	lerp := func(u, v uint8) uint8 { return uint8(float64(u) + (float64(v)-float64(u))*t) }
	return color.RGBA{R: lerp(a.R, b.R), G: lerp(a.G, b.G), B: lerp(a.B, b.B), A: 0xff}
}

func insideRoundRect(x, y, x0, y0, x1, y1, r int) bool {
	if x < x0 || x > x1 || y < y0 || y > y1 {
		return false
	}
	cx := x
	if x < x0+r {
		cx = x0 + r
	} else if x > x1-r {
		cx = x1 - r
	}
	cy := y
	if y < y0+r {
		cy = y0 + r
	} else if y > y1-r {
		cy = y1 - r
	}
	dx, dy := x-cx, y-cy
	return dx*dx+dy*dy <= r*r
}

// drawIconGlyph 按预设绘制中心图形
func drawIconGlyph(img *image.RGBA, p iconPalette, id string) {
	switch id {
	case "svcMon":
		fillRoundRect(img, 72, 56, 184, 148, 18, p.glyph)
		fillRoundRect(img, 84, 68, 172, 136, 12, p.accent)
		fillRect(img, 116, 148, 140, 176, p.glyph)
		fillRoundRect(img, 84, 176, 172, 186, 5, p.glyph)
	case "syncAgent":
		// 左箭头
		fillTriangle(img, 150, 64, 150, 112, 108, 88, p.glyph)
		// 右箭头
		fillTriangle(img, 106, 144, 106, 192, 148, 168, p.accent)
	case "panelTool":
		fillRoundRect(img, 64, 64, 120, 120, 12, p.glyph)
		fillRoundRect(img, 136, 64, 192, 120, 12, p.accent)
		fillRoundRect(img, 64, 136, 120, 192, 12, p.accent)
		fillRoundRect(img, 136, 136, 192, 192, 12, p.glyph)
	default: // docTool
		fillRoundRect(img, 84, 56, 172, 200, 14, p.glyph)
		fillRoundRect(img, 84, 140, 172, 200, 14, p.accent)
		fillRect(img, 100, 76, 156, 86, p.accent)
		fillRect(img, 100, 98, 156, 108, p.accent)
		fillRect(img, 100, 120, 156, 130, p.accent)
	}
}

func fillRect(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			if x >= 0 && x < 256 && y >= 0 && y < 256 {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func fillRoundRect(img *image.RGBA, x0, y0, x1, y1, r int, c color.RGBA) {
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			if insideRoundRect(x, y, x0, y0, x1, y1, r) {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func fillTriangle(img *image.RGBA, ax, ay, bx, by, cx, cy int, col color.RGBA) {
	minX, maxX := min(ax, min(bx, cx)), max(ax, max(bx, cx))
	minY, maxY := min(ay, min(by, cy)), max(ay, max(by, cy))
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if pointInTriangle(x, y, ax, ay, bx, by, cx, cy) {
				img.SetRGBA(x, y, col)
			}
		}
	}
}

func pointInTriangle(px, py, ax, ay, bx, by, cx, cy int) bool {
	sign := func(x1, y1, x2, y2, x3, y3 int) int { return (x1-x3)*(y2-y3) - (x2-x3)*(y1-y3) }
	d1 := sign(px, py, ax, ay, bx, by)
	d2 := sign(px, py, bx, by, cx, cy)
	d3 := sign(px, py, cx, cy, ax, ay)
	neg := (d1 < 0) || (d2 < 0) || (d3 < 0)
	pos := (d1 > 0) || (d2 > 0) || (d3 > 0)
	return !(neg && pos)
}
