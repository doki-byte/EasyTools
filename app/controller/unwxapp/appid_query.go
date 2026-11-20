package unwxapp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/tidwall/gjson"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const openAppidQueryURL = "https://open.kainy.cn/api/AppidQuery"

// Appid 元数据拉取方式（与 WechatConfig.AppidQueryMode 一致，空=both）
const (
	AppidQueryModeMP   = "mp"   // 仅内嵌 searchWxaembedded
	AppidQueryModeOpen = "open" // 仅 open.kainy X-API-Key
	AppidQueryModeBoth = "both" // 先 MP，失败再试 X-API-Key
)

// appidKeyRuntimeStatus 单条密钥在最近一次列表刷新批次中的探测结果（仅内存，不入库）。
type appidKeyRuntimeStatus struct {
	Status string // ok | depleted | error | unknown
	Detail string
}

// AppidAPIKeyStatus 供前端表格展示（密钥已脱敏）。
type AppidAPIKeyStatus struct {
	Line   int    `json:"Line"`
	Masked string `json:"Masked"`
	Status string `json:"Status"`
	Detail string `json:"Detail"`
}

// ErrAppidQueryQuotaExhausted 所有已配置的 X-API-Key 均无法完成单次查询时返回；findOrCreateInfo 会降级为仅 AppID 并可选向前端发事件。
var ErrAppidQueryQuotaExhausted = errors.New("APPID_QUERY_API_KEY_QUOTA_EXHAUSTED")

var errAppidQueryNoAPIKeys = errors.New("APPID_QUERY_NO_API_KEYS")

const appidQuotaEmitEvent = "unwxapp:appid_query_quota_exhausted"

func (u *UnWxapp) resetAppidQueryQuotaNoticeForBatch() {
	atomic.StoreInt32(&u.appidQuotaEmitState, 0)
}

func (u *UnWxapp) maybeEmitAppidQueryQuotaNotice() {
	if !atomic.CompareAndSwapInt32(&u.appidQuotaEmitState, 0, 1) {
		return
	}
	if u.Ctx != nil {
		runtime.EventsEmit(u.Ctx, appidQuotaEmitEvent, "")
	}
}

// 预览仅展示末尾一段，便于区分长密钥（前缀统一省略）。
const appidKeyMaskVisibleTail = 12

func maskAppidAPIKeyDisplay(k string) string {
	k = strings.TrimSpace(k)
	if k == "" {
		return "—"
	}
	if len(k) <= appidKeyMaskVisibleTail {
		return k
	}
	return "****" + k[len(k)-appidKeyMaskVisibleTail:]
}

func truncateAppidStatusDetail(s string, max int) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func (u *UnWxapp) beginAppidKeyStatusBatch(n int) {
	u.appidKeyStatusMu.Lock()
	defer u.appidKeyStatusMu.Unlock()
	if n <= 0 {
		u.appidKeyStatuses = nil
		return
	}
	u.appidKeyStatuses = make([]appidKeyRuntimeStatus, n)
	for i := range u.appidKeyStatuses {
		u.appidKeyStatuses[i] = appidKeyRuntimeStatus{Status: "unknown", Detail: ""}
	}
}

func (u *UnWxapp) clearAppidKeyStatuses() {
	u.appidKeyStatusMu.Lock()
	defer u.appidKeyStatusMu.Unlock()
	u.appidKeyStatuses = nil
}

func (u *UnWxapp) mergeAppidKeyStatus(index int, status, detail string) {
	u.appidKeyStatusMu.Lock()
	defer u.appidKeyStatusMu.Unlock()
	if index < 0 || index >= len(u.appidKeyStatuses) {
		return
	}
	detail = truncateAppidStatusDetail(detail, 160)
	prev := u.appidKeyStatuses[index]
	switch status {
	case "ok":
		u.appidKeyStatuses[index] = appidKeyRuntimeStatus{Status: "ok", Detail: ""}
	case "depleted":
		u.appidKeyStatuses[index] = appidKeyRuntimeStatus{Status: "depleted", Detail: detail}
	case "error":
		if prev.Status == "depleted" {
			return
		}
		u.appidKeyStatuses[index] = appidKeyRuntimeStatus{Status: "error", Detail: detail}
	}
}

// GetAppidQueryAPIKeyStatuses 返回当前各密钥行状态（与配置行顺序一致），供编辑对话框表格使用。
func (u *UnWxapp) GetAppidQueryAPIKeyStatuses() ([]AppidAPIKeyStatus, error) {
	cfg, err := u.getConfig()
	if err != nil {
		return nil, err
	}
	keys := splitAppidQueryAPIKeys(cfg.AppidQueryAPIKeys)
	u.appidKeyStatusMu.Lock()
	st := append([]appidKeyRuntimeStatus(nil), u.appidKeyStatuses...)
	u.appidKeyStatusMu.Unlock()

	out := make([]AppidAPIKeyStatus, 0, len(keys))
	for i, k := range keys {
		row := AppidAPIKeyStatus{
			Line:   i + 1,
			Masked: maskAppidAPIKeyDisplay(k),
			Status: "unknown",
			Detail: "",
		}
		if i < len(st) && st[i].Status != "" {
			row.Status = st[i].Status
			row.Detail = st[i].Detail
		}
		out = append(out, row)
	}
	return out, nil
}

// RemoveDepletedAppidQueryAPIKeys 从配置原文中删除当前内存状态为 depleted 的密钥行（保留空行与 # 注释），返回删除条数。
func (u *UnWxapp) RemoveDepletedAppidQueryAPIKeys() (int, error) {
	cfg, err := u.getConfig()
	if err != nil {
		return 0, err
	}
	raw := cfg.AppidQueryAPIKeys
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")

	u.appidKeyStatusMu.Lock()
	st := append([]appidKeyRuntimeStatus(nil), u.appidKeyStatuses...)
	u.appidKeyStatusMu.Unlock()

	keyIdx := 0
	removed := 0
	outLines := make([]string, 0, len(lines))
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			outLines = append(outLines, line)
			continue
		}
		if keyIdx < len(st) && st[keyIdx].Status == "depleted" {
			removed++
			keyIdx++
			continue
		}
		outLines = append(outLines, line)
		keyIdx++
	}

	cfg.AppidQueryAPIKeys = strings.TrimSpace(strings.Join(outLines, "\n"))
	if err := u.saveConfig(cfg); err != nil {
		return 0, err
	}
	u.clearAppidKeyStatuses()
	return removed, nil
}

func splitAppidQueryAPIKeys(raw string) []string {
	var keys []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			keys = append(keys, line)
		}
	}
	return keys
}

func shouldTryNextAppidQueryKey(status int, body string) bool {
	if status == http.StatusUnauthorized || status == http.StatusForbidden || status == http.StatusTooManyRequests {
		return true
	}
	if strings.Contains(body, "API key 信息有误") || strings.Contains(strings.ToLower(body), "invalid api key") {
		return true
	}
	if strings.Contains(body, "额度") && (strings.Contains(body, "用尽") || strings.Contains(body, "用完") || strings.Contains(body, "不足") || strings.Contains(body, "超限")) {
		return true
	}
	return false
}

func firstNonEmptyGJSON(body string, paths ...string) string {
	for _, p := range paths {
		if s := strings.TrimSpace(gjson.Get(body, p).String()); s != "" {
			return s
		}
	}
	return ""
}

func miniInfoFromOpenAppidQueryJSON(appid, body string) *MiniAppInfo {
	return &MiniAppInfo{
		AppID:         appid,
		Nickname:      firstNonEmptyGJSON(body, "data.nickname", "data.nickName", "nickname"),
		Username:      firstNonEmptyGJSON(body, "data.username", "data.userName", "username"),
		Description:   firstNonEmptyGJSON(body, "data.description", "description"),
		Avatar:        firstNonEmptyGJSON(body, "data.avatar", "avatar"),
		UsesCount:     firstNonEmptyGJSON(body, "data.uses_count", "data.usesCount", "uses_count"),
		PrincipalName: firstNonEmptyGJSON(body, "data.principal_name", "data.principalName", "principal_name"),
	}
}

// GetAppidQueryAPIKeys 返回配置的多行 X-API-Key 原文（一行一个）。
func (u *UnWxapp) GetAppidQueryAPIKeys() (string, error) {
	cfg, err := u.getConfig()
	if err != nil {
		return "", err
	}
	return cfg.AppidQueryAPIKeys, nil
}

// GetAppidQueryMode 返回 mp / open / both 之一，未配置时等同 both。
func (u *UnWxapp) GetAppidQueryMode() (string, error) {
	cfg, err := u.getConfig()
	if err != nil {
		return "", err
	}
	return normalizeAppidQueryMode(cfg.AppidQueryMode), nil
}

// SetAppidQueryMode 设置 mp、open 或 both。
func (u *UnWxapp) SetAppidQueryMode(mode string) error {
	m := strings.TrimSpace(mode)
	switch m {
	case AppidQueryModeMP, AppidQueryModeOpen, AppidQueryModeBoth, "":
	default:
		return fmt.Errorf("不支持的 appid 查询方式: %s", mode)
	}
	cfg, err := u.getConfig()
	if err != nil {
		return err
	}
	if m == "" {
		m = AppidQueryModeBoth
	}
	cfg.AppidQueryMode = m
	if err := u.saveConfig(cfg); err != nil {
		return err
	}
	u.clearAppidKeyStatuses()
	return nil
}

func normalizeAppidQueryMode(raw string) string {
	switch strings.TrimSpace(raw) {
	case AppidQueryModeMP, AppidQueryModeOpen, AppidQueryModeBoth:
		return strings.TrimSpace(raw)
	case "":
		return AppidQueryModeBoth
	default:
		return AppidQueryModeBoth
	}
}

// SetAppidQueryAPIKeys 保存多行 X-API-Key（一行一个，空行与 # 开头行会被忽略）。
func (u *UnWxapp) SetAppidQueryAPIKeys(raw string) error {
	cfg, err := u.getConfig()
	if err != nil {
		return err
	}
	cfg.AppidQueryAPIKeys = strings.TrimSpace(raw)
	if err := u.saveConfig(cfg); err != nil {
		return err
	}
	u.clearAppidKeyStatuses()
	return nil
}

func (u *UnWxapp) queryMimiAPPInfo(appid string) (*MiniAppInfo, error) {
	cfg, err := u.getConfig()
	if err != nil {
		return nil, err
	}

	mode := normalizeAppidQueryMode(cfg.AppidQueryMode)
	cookie := strings.TrimSpace(cfg.WechatMPCookie)
	token := strings.TrimSpace(cfg.WechatMPToken)
	tryMP := (mode == AppidQueryModeMP || mode == AppidQueryModeBoth) && cookie != "" && token != "" && !u.mpEmbedBatchSkipMP.Load()
	tryOpen := mode == AppidQueryModeOpen || mode == AppidQueryModeBoth

	if mode == AppidQueryModeMP && (cookie == "" || token == "") {
		return nil, errAppidQueryNoAPIKeys
	}

	if tryMP {
		if info, errMP := u.queryMiniAppInfoViaMPEmbedded(appid, cfg); errMP == nil && info != nil {
			u.recordMPEmbedBatchMPSuccess()
			return info, nil
		} else if errMP != nil {
			u.recordMPEmbedBatchMPFailure()
			log.Printf("内嵌 searchWxaembedded 查询 %s: %v", appid, errMP)
			if mode == AppidQueryModeMP {
				return nil, errMP
			}
		} else {
			// 异常空结果
			u.recordMPEmbedBatchMPFailure()
			if mode == AppidQueryModeMP {
				return nil, fmt.Errorf("searchWxaembedded 无有效数据: %s", appid)
			}
		}
	}

	if !tryOpen {
		return nil, errAppidQueryNoAPIKeys
	}

	keys := splitAppidQueryAPIKeys(cfg.AppidQueryAPIKeys)
	if len(keys) == 0 {
		return nil, errAppidQueryNoAPIKeys
	}

	payload, err := json.Marshal(map[string]string{"appid": appid})
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %w", err)
	}

	for i, apiKey := range keys {
		req, err := http.NewRequest(http.MethodPost, openAppidQueryURL, bytes.NewReader(payload))
		if err != nil {
			return nil, fmt.Errorf("创建API请求失败: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", apiKey)

		resp, err := u.http.Do(req)
		if err != nil {
			log.Printf("AppidQuery 请求失败 (密钥 #%d): %v", i+1, err)
			u.mergeAppidKeyStatus(i, "error", "网络: "+err.Error())
			continue
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Printf("AppidQuery 读取响应失败 (密钥 #%d): %v", i+1, err)
			u.mergeAppidKeyStatus(i, "error", "读取响应: "+err.Error())
			continue
		}
		body := string(bodyBytes)

		if shouldTryNextAppidQueryKey(resp.StatusCode, body) {
			log.Printf("AppidQuery 密钥 #%d 不可用 (HTTP %d)，轮换下一密钥", i+1, resp.StatusCode)
			u.mergeAppidKeyStatus(i, "depleted", fmt.Sprintf("HTTP %d %s", resp.StatusCode, truncateAppidStatusDetail(body, 80)))
			continue
		}

		if resp.StatusCode != http.StatusOK {
			u.mergeAppidKeyStatus(i, "error", fmt.Sprintf("HTTP %d %s", resp.StatusCode, truncateAppidStatusDetail(body, 80)))
			return nil, fmt.Errorf("AppidQuery HTTP %d: %s", resp.StatusCode, strings.TrimSpace(body))
		}

		// 顶层 error 字符串（如无 code 字段）
		if flatErr := strings.TrimSpace(gjson.Get(body, "error").String()); flatErr != "" && !gjson.Get(body, "code").Exists() {
			if shouldTryNextAppidQueryKey(0, flatErr) {
				log.Printf("AppidQuery 密钥 #%d 被拒绝: %s", i+1, flatErr)
				u.mergeAppidKeyStatus(i, "depleted", flatErr)
				continue
			}
			u.mergeAppidKeyStatus(i, "error", flatErr)
			return nil, fmt.Errorf("AppidQuery: %s", flatErr)
		}

		nestedErr := strings.TrimSpace(gjson.Get(body, "error.message").String())
		if nestedErr != "" && !gjson.Get(body, "code").Exists() {
			if shouldTryNextAppidQueryKey(0, nestedErr) {
				u.mergeAppidKeyStatus(i, "depleted", nestedErr)
				continue
			}
			u.mergeAppidKeyStatus(i, "error", nestedErr)
			return nil, fmt.Errorf("AppidQuery: %s", nestedErr)
		}

		code := gjson.Get(body, "code").Int()
		errMsg := strings.TrimSpace(gjson.Get(body, "errors").String())
		if errMsg == "" {
			errMsg = strings.TrimSpace(gjson.Get(body, "error").String())
		}

		if code == 2 {
			u.mergeAppidKeyStatus(i, "ok", "")
			return &MiniAppInfo{AppID: appid}, nil
		}
		if code != 0 {
			if shouldTryNextAppidQueryKey(0, body) || shouldTryNextAppidQueryKey(0, errMsg) {
				log.Printf("AppidQuery 密钥 #%d 业务响应需轮换: code=%d %s", i+1, code, errMsg)
				u.mergeAppidKeyStatus(i, "depleted", fmt.Sprintf("code=%d %s", code, errMsg))
				continue
			}
			u.mergeAppidKeyStatus(i, "error", fmt.Sprintf("code=%d %s", code, errMsg))
			return nil, fmt.Errorf("AppidQuery 业务错误 (code=%d): %s", code, errMsg)
		}

		u.mergeAppidKeyStatus(i, "ok", "")
		return miniInfoFromOpenAppidQueryJSON(appid, body), nil
	}

	return nil, ErrAppidQueryQuotaExhausted
}
