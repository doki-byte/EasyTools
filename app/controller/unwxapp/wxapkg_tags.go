package unwxapp

import (
	"os"
	"sort"
	"strings"
)

const (
	wxapkgLabelMini   = "小程序"
	wxapkgLabelPlugin = "插件"
	wxapkgLabelGame   = "小游戏"
	wxapkgLabelOther  = "其他"
)

func wxapkgBaseLabel(base string) string {
	switch {
	case strings.EqualFold(base, "__APP__"):
		return wxapkgLabelMini
	case strings.EqualFold(base, "__PLUGINCODE__"):
		return wxapkgLabelPlugin
	case strings.EqualFold(base, "__WITHOUT_MULTI_PLUGINCODE__"):
		return wxapkgLabelGame
	default:
		return wxapkgLabelOther
	}
}

// wxapkgTagsFromDir 扫描目录下全部 .wxapkg，映射为展示标签（去重、固定展示顺序）
func wxapkgTagsFromDir(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	seen := make(map[string]struct{})
	var tags []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if len(name) < 8 || !strings.HasSuffix(strings.ToLower(name), ".wxapkg") {
			continue
		}
		base := name[:len(name)-len(".wxapkg")]
		label := wxapkgBaseLabel(base)
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		tags = append(tags, label)
	}
	prio := map[string]int{
		wxapkgLabelMini:   0,
		wxapkgLabelPlugin: 1,
		wxapkgLabelGame:   2,
		wxapkgLabelOther:  3,
	}
	sort.Slice(tags, func(i, j int) bool {
		pi, pj := prio[tags[i]], prio[tags[j]]
		if pi != pj {
			return pi < pj
		}
		return tags[i] < tags[j]
	})
	return tags
}
