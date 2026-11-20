package unwxapp

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

// 与独立脚本默认一致
const (
	mpSearchWxaEmbeddedURL = "https://mp.weixin.qq.com/wxamp/cgi/wxaembedded/searchWxaembedded"
	mpDefaultLang          = "zh_CN"
	defaultMPUserAgent     = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36"
	// 两次使用 Cookie 的 MP 请求之间的最小间隔（不并发、减轻风控）
	minMPRequestInterval = 1200 * time.Millisecond
)

// GetWechatMPCookie 与前端绑定：读取保存的公众号 MP Cookie。
func (u *UnWxapp) GetWechatMPCookie() (string, error) {
	cfg, err := u.getConfig()
	if err != nil {
		return "", err
	}
	return cfg.WechatMPCookie, nil
}

// SetWechatMPCookie 保存公众号 MP Cookie（k=v; 形式，从浏览器复制）。
func (u *UnWxapp) SetWechatMPCookie(raw string) error {
	cfg, err := u.getConfig()
	if err != nil {
		return err
	}
	cfg.WechatMPCookie = strings.TrimSpace(raw)
	return u.saveConfig(cfg)
}

// GetWechatMPToken 与前端绑定：读取保存的公众号 MP token。
func (u *UnWxapp) GetWechatMPToken() (string, error) {
	cfg, err := u.getConfig()
	if err != nil {
		return "", err
	}
	return cfg.WechatMPToken, nil
}

// SetWechatMPToken 保存公众号 MP token（从 mp.weixin.qq.com URL 中复制）。
func (u *UnWxapp) SetWechatMPToken(raw string) error {
	cfg, err := u.getConfig()
	if err != nil {
		return err
	}
	cfg.WechatMPToken = strings.TrimSpace(raw)
	return u.saveConfig(cfg)
}

// queryMiniAppInfoViaMPEmbedded 调用内嵌管理页与独立脚本等价的 GET，并解析为 MiniAppInfo；失败时返回 error 供上层回退到 open API。
func (u *UnWxapp) queryMiniAppInfoViaMPEmbedded(appid string, cfg *WechatConfig) (*MiniAppInfo, error) {
	if strings.TrimSpace(appid) == "" {
		return nil, fmt.Errorf("appid 为空")
	}

	u.appidMPEmbedMu.Lock()
	if !u.appidMPEmbedLast.IsZero() {
		if w := time.Since(u.appidMPEmbedLast); w < minMPRequestInterval {
			time.Sleep(minMPRequestInterval - w)
		}
	}
	defer func() {
		u.appidMPEmbedLast = time.Now()
		u.appidMPEmbedMu.Unlock()
	}()

	token := strings.TrimSpace(cfg.WechatMPToken)
	if token == "" {
		return nil, fmt.Errorf("公众号 MP token 未配置")
	}
	lang := mpDefaultLang

	p := url.Values{}
	p.Set("appid", appid)
	p.Set("token", token)
	p.Set("lang", lang)
	full := mpSearchWxaEmbeddedURL + "?" + p.Encode()

	req, err := http.NewRequest(http.MethodGet, full, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Sec-Ch-Ua", `"Chromium";v="146", "Not-A.Brand";v="24", "Google Chrome";v="146"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", defaultMPUserAgent)
	req.Header.Set("Cookie", strings.TrimSpace(cfg.WechatMPCookie))

	client := u.mpEmbedHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	s := string(body)
	if sc := resp.StatusCode; sc != http.StatusOK {
		if sc == http.StatusFound || sc == http.StatusMovedPermanently {
			loc := resp.Header.Get("Location")
			if strings.Contains(loc, "login") || sc == 302 {
				return nil, fmt.Errorf("微信 MP 重定向(HTTP %d)，Cookie 可能已失效: %s", sc, loc)
			}
		}
		return nil, fmt.Errorf("微信 searchWxaembedded HTTP %d: %s", sc, truncateAppidStatusDetail(s, 200))
	}

	if isLikelyLoginPage(s) {
		return nil, fmt.Errorf("返回页面疑似未登录，请更新 Cookie")
	}

	out := MiniAppInfo{AppID: appid}
	fill := miniInfoFromMPEmbeddedBody(appid, s)
	if fill == nil || !fillMeaningfulMPEmbeddedInfo(fill) {
		return nil, fmt.Errorf("无法从 MP 页面解析到小程序元数据，可检查 token 或与独立脚本参数是否一致")
	}
	out.Nickname = fill.Nickname
	out.Username = fill.Username
	out.Description = fill.Description
	out.Avatar = fill.Avatar
	out.UsesCount = fill.UsesCount
	out.PrincipalName = fill.PrincipalName
	return &out, nil
}

func (u *UnWxapp) mpEmbedHTTPClient() *http.Client {
	// 与脚本相同：不自动跟随重定向
	base := u.http
	if base == nil {
		return &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}
	tr := base.Transport
	if tr == nil {
		tr = http.DefaultTransport
	}
	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func isLikelyLoginPage(s string) bool {
	ls := strings.ToLower(s)
	if strings.Contains(ls, "登录") && (strings.Contains(ls, "mp.weixin") || strings.Contains(ls, "扫码")) {
		return true
	}
	if strings.Contains(s, "session expired") {
		return true
	}
	return false
}

func fillMeaningfulMPEmbeddedInfo(m *MiniAppInfo) bool {
	if m == nil {
		return false
	}
	if strings.TrimSpace(m.Nickname) == "" && strings.TrimSpace(m.Username) == "" && strings.TrimSpace(m.Avatar) == "" {
		return false
	}
	return true
}

var reMPKV = struct {
	nick, user, head, principal, usecnt *regexp.Regexp
}{
	nick:      regexp.MustCompile(`(?i)"(nick_?name|nickName|nickname|wxa_nickname|wxaNickname)"\s*:\s*"([^"\\]*)"`),
	user:      regexp.MustCompile(`(?i)"(user_?name|userName|username|wxa_username|wxaUsername|original_?id|originalId)"\s*:\s*"([^"\\]*)"`),
	head:      regexp.MustCompile(`(?i)"(head_?img|round_head_img|avatar|round_headimg|headimg|headImg)"\s*:\s*"(https?://[^"\\]*)"`),
	principal: regexp.MustCompile(`(?i)"(principal_?name|principalName|real_?name|author)"\s*:\s*"([^"\\]*)"`),
	usecnt:    regexp.MustCompile(`(?i)"(use_?s_?count|uses_?count|use_count|visits)"\s*:\s*"?([0-9,]+)"?`),
}

func firstSubMatch(re *regexp.Regexp, s string) string {
	if re == nil {
		return ""
	}
	m := re.FindStringSubmatch(s)
	if len(m) < 3 {
		return ""
	}
	return strings.TrimSpace(m[2])
}

// miniInfoFromMPEmbeddedBody 在 HTML/内嵌 JSON 中尽力抽取与 MiniAppInfo 同名字段。
func miniInfoFromMPEmbeddedBody(appid, body string) *MiniAppInfo {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil
	}
	m := &MiniAppInfo{AppID: appid}
	gj := gjson.Parse(body)
	if gjson.Valid(body) {
		applyGJSONMPCandidates(m, gj)
	}

	paths := [][2]string{
		{"data.list.0", "data.list.0.nickname"},
		{"data.0", "data.0.nickname"},
		{"list.0", "list.0.nick_name"},
		{"appmsg", "appmsg.nick_name"},
		{"app_info", "app_info.nickname"},
		{"wxa_info", "wxa_info.nick_name"},
		{"data.wxa", "data.wxa.nick_name"},
		{"data.appinfo", "data.appinfo.nickname"},
		{"appinfo", "appinfo.nickname"},
	}
	for _, p := range paths {
		if base := gjson.Get(body, p[0]); base.IsObject() || base.IsArray() {
			applyGJSONMPCandidates(m, base)
		} else {
			applyGJSONMPCandidates(m, gjson.Get(body, p[0]))
		}
		applyGJSONMPCandidates(m, gjson.Get(body, p[1]))
	}
	if m.Nickname == "" {
		if n := firstSubMatch(reMPKV.nick, body); n != "" {
			m.Nickname = n
		}
	}
	if m.Username == "" {
		if n := firstSubMatch(reMPKV.user, body); n != "" {
			m.Username = n
		}
	}
	if m.Avatar == "" {
		if n := firstSubMatch(reMPKV.head, body); n != "" {
			m.Avatar = n
		}
	}
	if m.PrincipalName == "" {
		if n := firstSubMatch(reMPKV.principal, body); n != "" {
			m.PrincipalName = n
		}
	}
	if m.UsesCount == "" {
		if n := firstSubMatch(reMPKV.usecnt, body); n != "" {
			m.UsesCount = n
		}
	}
	// 描述
	for _, p := range []string{"data.list.0.description", "list.0.description", "wxa_info.desc", "appinfo.description"} {
		if d := strings.TrimSpace(gjson.Get(body, p).String()); d != "" {
			m.Description = d
			break
		}
	}
	return m
}

func applyGJSONMPCandidates(m *MiniAppInfo, j gjson.Result) {
	if !j.Exists() {
		return
	}
	if m.Nickname == "" {
		for _, k := range []string{"nick_name", "nickName", "nickname", "wxa_nickname", "wxaNickname", "name"} {
			if v := strings.TrimSpace(j.Get(k).String()); v != "" {
				m.Nickname = v
				break
			}
		}
	}
	if m.Username == "" {
		for _, k := range []string{"user_name", "userName", "username", "wxa_username", "original_id", "originalId"} {
			if v := strings.TrimSpace(j.Get(k).String()); v != "" {
				m.Username = v
				break
			}
		}
	}
	if m.Avatar == "" {
		for _, k := range []string{"round_headimg", "headimg", "head_img", "avatar", "round_head_img"} {
			if v := strings.TrimSpace(j.Get(k).String()); v != "" {
				m.Avatar = v
				break
			}
		}
	}
	if m.PrincipalName == "" {
		for _, k := range []string{"principal_name", "principalName", "realname", "author", "open_app_info.principal_name"} {
			if v := strings.TrimSpace(j.Get(k).String()); v != "" {
				m.PrincipalName = v
				break
			}
		}
	}
	if m.UsesCount == "" {
		for _, k := range []string{"uses_count", "use_count", "visit_total"} {
			if v := strings.TrimSpace(j.Get(k).String()); v != "" {
				m.UsesCount = v
				break
			}
		}
	}
}

// resetMPEmbedBatchGate 在每次 GetAllMiniApp 枚举前调用，重置「连续 MP 失败则本批跳过」状态。
func (u *UnWxapp) resetMPEmbedBatchGate() {
	u.mpEmbedBatchSkipMP.Store(false)
	u.mpEmbedBatchConsecutiveFails.Store(0)
}

func (u *UnWxapp) recordMPEmbedBatchMPSuccess() {
	u.mpEmbedBatchConsecutiveFails.Store(0)
}

// recordMPEmbedBatchMPFailure 在内嵌 searchWxaembedded 单次失败时调用；连续 3 次后本批不再请求 MP。
func (u *UnWxapp) recordMPEmbedBatchMPFailure() {
	n := u.mpEmbedBatchConsecutiveFails.Add(1)
	if n >= 3 {
		if u.mpEmbedBatchSkipMP.CompareAndSwap(false, true) {
			log.Printf("内嵌 searchWxaembedded 已连续失败 %d 次，本批剩余项将跳过 MP 查询；请检查公众号 MP Cookie/Token。已配置 Open API 时仍会继续尝试", n)
		}
	}
}
