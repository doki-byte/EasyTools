package browserScan

import (
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/go-rod/rod"
)

//go:embed vue_crack_detector.js
var vueCrackDetectorJS string

// VueRouteItem 单条路由（与 VueCrack detector 输出一致）
type VueRouteItem struct {
	Name string                 `json:"name"`
	Path string                 `json:"path"`
	Meta map[string]interface{} `json:"meta"`
}

// VuePrefixStat 页面链接统计的前缀
type VuePrefixStat struct {
	Prefix string `json:"prefix"`
	Count  int    `json:"count"`
}

// VuePageAnalysis 页面链接推断的基础路径
type VuePageAnalysis struct {
	DetectedBasePath string          `json:"detectedBasePath"`
	CommonPrefixes   []VuePrefixStat `json:"commonPrefixes"`
}

// VueAnalysisResult 页面内 Vue / Vue Router 分析结果（参考 VueCrack detector.js）
type VueAnalysisResult struct {
	VueDetected    bool             `json:"vueDetected"`
	RouterDetected bool             `json:"routerDetected"`
	VueVersion     string           `json:"vueVersion"`
	ModifiedRoutes []VueRouteItem   `json:"modifiedRoutes"`
	AllRoutes      []VueRouteItem   `json:"allRoutes"`
	RouterBase     string           `json:"routerBase"`
	PageAnalysis   *VuePageAnalysis `json:"pageAnalysis,omitempty"`
	CurrentPath    string           `json:"currentPath"`
	Error          string           `json:"error,omitempty"`
}

// vueAnalysisJSON 与页面 postMessage 载荷一致；pageAnalysis 在 detector 里可能被 sanitize 成字符串，不能强绑为对象。
type vueAnalysisJSON struct {
	VueDetected    bool           `json:"vueDetected"`
	RouterDetected bool           `json:"routerDetected"`
	VueVersion     string         `json:"vueVersion"`
	ModifiedRoutes []VueRouteItem `json:"modifiedRoutes"`
	AllRoutes      []VueRouteItem `json:"allRoutes"`
	RouterBase     string         `json:"routerBase"`
	PageAnalysis   json.RawMessage `json:"pageAnalysis"`
	CurrentPath    string         `json:"currentPath"`
	Error          string         `json:"error"`
}

func parsePageAnalysisRaw(raw json.RawMessage) *VuePageAnalysis {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	// 非 JSON 对象（例如 "[Object]" 字符串）则忽略
	if raw[0] != '{' {
		return nil
	}
	var p VuePageAnalysis
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil
	}
	return &p
}

func decodeVueAnalysisFromRouterJSON(b []byte) (*VueAnalysisResult, error) {
	var j vueAnalysisJSON
	if err := json.Unmarshal(b, &j); err != nil {
		return nil, err
	}
	out := &VueAnalysisResult{
		VueDetected:    j.VueDetected,
		RouterDetected: j.RouterDetected,
		VueVersion:     j.VueVersion,
		RouterBase:     j.RouterBase,
		CurrentPath:    j.CurrentPath,
		Error:          j.Error,
		PageAnalysis:   parsePageAnalysisRaw(j.PageAnalysis),
	}
	if j.ModifiedRoutes != nil {
		out.ModifiedRoutes = j.ModifiedRoutes
	} else {
		out.ModifiedRoutes = []VueRouteItem{}
	}
	if j.AllRoutes != nil {
		out.AllRoutes = j.AllRoutes
	} else {
		out.AllRoutes = []VueRouteItem{}
	}
	return out, nil
}

// runVueCrackAnalysis 在已加载的 Rod 页面中注入 VueCrack 的 detector 逻辑并读取 postMessage 结果。
func (bs *BrowserScan) runVueCrackAnalysis(page *rod.Page) *VueAnalysisResult {
	if page == nil || vueCrackDetectorJS == "" {
		return nil
	}
	b64 := base64.StdEncoding.EncodeToString([]byte(vueCrackDetectorJS))
	// strconv.Quote(b64) 生成合法 JS 字符串字面量，嵌入 eval(atob(...))
	injectJS := fmt.Sprintf(`() => {
		try {
			window.__ETVueCapture = null;
			window.__ETVueDetection = null;
			window.__ETVueErr = null;
			const __origPM = window.postMessage.bind(window);
			window.postMessage = function() {
				const msg = arguments[0];
				if (msg && typeof msg === "object") {
					if (msg.type === "VUE_ROUTER_ANALYSIS_RESULT") { window.__ETVueCapture = msg.result; }
					if (msg.type === "VUE_DETECTION_RESULT") { window.__ETVueDetection = msg.result; }
				}
				return __origPM.apply(window, arguments);
			};
			eval(atob(%s));
			return true;
		} catch (e) {
			window.__ETVueErr = String(e);
			return false;
		}
	}`, strconv.Quote(b64))

	_, err := page.Eval(injectJS)
	if err != nil {
		return &VueAnalysisResult{Error: "注入脚本失败: " + err.Error()}
	}

	// detector 含延迟重试（最多约 900ms）+ performFullAnalysis 内 50ms
	time.Sleep(2600 * time.Millisecond)

	res, err := page.Eval(`() => JSON.stringify({
		"router": window.__ETVueCapture !== undefined && window.__ETVueCapture !== null ? window.__ETVueCapture : null,
		"detection": window.__ETVueDetection !== undefined && window.__ETVueDetection !== null ? window.__ETVueDetection : null,
		"injectErr": window.__ETVueErr !== undefined && window.__ETVueErr !== null ? window.__ETVueErr : null
	})`)
	if err != nil {
		return &VueAnalysisResult{Error: "读取结果失败: " + err.Error()}
	}

	rawStr := res.Value.Str()
	var bundle struct {
		Router    json.RawMessage `json:"router"`
		Detection *struct {
			Detected bool   `json:"detected"`
			Method   string `json:"method"`
		} `json:"detection"`
		InjectErr *string `json:"injectErr"`
	}
	if err := json.Unmarshal([]byte(rawStr), &bundle); err != nil {
		return &VueAnalysisResult{Error: "解析 JSON 失败: " + err.Error()}
	}
	if bundle.InjectErr != nil && *bundle.InjectErr != "" {
		return &VueAnalysisResult{Error: "页面内执行异常: " + *bundle.InjectErr}
	}
	if len(bundle.Router) > 0 && string(bundle.Router) != "null" {
		router, err := decodeVueAnalysisFromRouterJSON(bundle.Router)
		if err != nil {
			return &VueAnalysisResult{Error: "解析 JSON 失败: " + err.Error()}
		}
		return router
	}
	if bundle.Detection != nil {
		return &VueAnalysisResult{
			VueDetected:    bundle.Detection.Detected,
			RouterDetected: false,
			VueVersion:     "",
			AllRoutes:      []VueRouteItem{},
			ModifiedRoutes: []VueRouteItem{},
		}
	}
	return &VueAnalysisResult{
		VueDetected:    false,
		RouterDetected: false,
		AllRoutes:      []VueRouteItem{},
		ModifiedRoutes: []VueRouteItem{},
	}
}
