package cloudSecurity

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"EasyTools/app/model"

	"github.com/tidwall/gjson"
)

const (
	ucloudAPIBaseCN     = "https://api.ucloud.cn"
	ucloudAPIBaseGlobal = "https://api.ucloud-global.com"
)

// 本文件云主机列举与 UCloud OpenAPI 对齐范围（完整目录见 https://docs.ucloud.cn/api ）：
//   - uhost-api     DescribeUHostInstance（普通云主机）
//   - ulighthost-api DescribeULHostInstance（轻量）
//   - uphost-api    DescribePHost（物理云主机 / 裸金属）
// 订单 UBILL DescribeOrderDetailInfo；接管 IAM CreateUser 等见 cloud_server_takeover.go。

// ucloudAPIGET 默认走国内网关；国际账号可在列举流程中自动切换到 api.ucloud-global.com。
func ucloudAPIGET(params map[string]string) ([]byte, error) {
	return ucloudAPIGETWithBase(ucloudAPIBaseCN, params)
}

func ucloudAPIGETWithBase(base string, params map[string]string) ([]byte, error) {
	base = strings.TrimSpace(base)
	if base == "" {
		base = ucloudAPIBaseCN
	}
	q := url.Values{}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		q.Set(k, params[k])
	}
	reqURL := strings.TrimRight(base, "/") + "/?" + q.Encode()
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// ucloudRegionsExplicitInExtra 为 true 时表示用户显式配置了 regions（非空数组），列举时不再用 GetRegion 覆盖。
func ucloudRegionsExplicitInExtra(extraJSON string) bool {
	raw := strings.TrimSpace(gjson.Get(extraJSON, "regions").Raw)
	if raw == "" || raw == "null" {
		return false
	}
	var arr []string
	if err := json.Unmarshal([]byte(raw), &arr); err != nil {
		return true
	}
	return len(arr) > 0
}

// ucloudFetchAccessibleRegionCodes 调 GetRegion（无需 Region 参数），返回账号可见数据中心代码及成功的 API 网关。
func ucloudFetchAccessibleRegionCodes(pub, priv string) (codes []string, baseOK string, err error) {
	pub, priv = strings.TrimSpace(pub), strings.TrimSpace(priv)
	if pub == "" || priv == "" {
		return nil, "", fmt.Errorf("PublicKey/PrivateKey 为空")
	}
	params := map[string]string{
		"Action":      "GetRegion",
		"PublicKey":   pub,
		"RequestUUID": strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	sig, err := ucloudSignature(priv, params)
	if err != nil {
		return nil, "", err
	}
	params["Signature"] = sig

	var lastErr error
	for _, base := range []string{ucloudAPIBaseCN, ucloudAPIBaseGlobal} {
		body, rerr := ucloudAPIGETWithBase(base, params)
		if rerr != nil {
			lastErr = rerr
			continue
		}
		if gjson.GetBytes(body, "RetCode").Int() != 0 {
			lastErr = fmt.Errorf("UCloud API: %s", gjson.GetBytes(body, "Message").String())
			continue
		}
		out := ucloudParseGetRegionBody(body)
		if len(out) > 0 {
			return out, base, nil
		}
	}
	if lastErr != nil {
		return nil, "", lastErr
	}
	return nil, "", nil
}

func ucloudParseGetRegionBody(body []byte) []string {
	seen := map[string]bool{}
	var out []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		out = append(out, s)
	}
	for _, r := range gjson.GetBytes(body, "Regions").Array() {
		add(r.Get("Region").String())
	}
	if len(out) == 0 {
		for _, r := range gjson.GetBytes(body, "Data.Regions").Array() {
			add(r.Get("Region").String())
		}
	}
	return out
}

// ucloudListOrderRows 调用 UBILL DescribeOrderDetailInfo，返回与前端订单表一致的行。
func ucloudListOrderRows(cr *model.CloudServerCredential, region string, beginTS, endTS int64) ([]map[string]interface{}, error) {
	pub := strings.TrimSpace(cr.AccessKey)
	priv := strings.TrimSpace(cr.SecretKey)
	projectID := strings.TrimSpace(gjson.Get(cr.ExtraJSON, "ucloud_project_id").String())
	if projectID == "" {
		projectID = strings.TrimSpace(gjson.Get(cr.ExtraJSON, "project_id").String())
	}
	if pub == "" || priv == "" {
		return nil, fmt.Errorf("UCloud 需填写 PublicKey 与 PrivateKey")
	}
	if region == "" {
		region = "cn-bj2"
	}
	if endTS <= beginTS {
		endTS = time.Now().Unix()
		if beginTS == 0 {
			beginTS = endTS - 90*24*3600
		}
	}

	params := map[string]string{
		"Action":      "DescribeOrderDetailInfo",
		"PublicKey":   pub,
		"Region":      region,
		"BeginTime":   strconv.FormatInt(beginTS, 10),
		"EndTime":     strconv.FormatInt(endTS, 10),
		"Limit":       "100",
		"Offset":      "0",
		"RequestUUID": strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	if projectID != "" {
		params["ProjectId"] = projectID
	}
	sig, err := ucloudSignature(priv, params)
	if err != nil {
		return nil, err
	}
	params["Signature"] = sig

	body, err := ucloudAPIGET(params)
	if err != nil {
		return nil, err
	}
	if gjson.GetBytes(body, "RetCode").Int() != 0 {
		return nil, fmt.Errorf("UCloud API: %s", gjson.GetBytes(body, "Message").String())
	}

	arr := gjson.GetBytes(body, "OrderInfos").Array()
	if len(arr) == 0 {
		arr = gjson.GetBytes(body, "Data.OrderInfos").Array()
	}
	var rows []map[string]interface{}
	for _, o := range arr {
		product := ucloudFormatOrderProduct(o)
		st := o.Get("StartTime").Int()
		et := o.Get("EndTime").Int()
		rows = append(rows, map[string]interface{}{
			"orderId":      o.Get("OrderNo").String(),
			"resourceId":   o.Get("ResourceId").String(),
			"resourceType": o.Get("ResourceType").String(),
			"start":        ucloudFormatUnixTime(st),
			"end":          ucloudFormatUnixTime(et),
			"product":      product,
			"billing":      o.Get("ChargeType").String(),
			"amount":       firstNonEmptyStr(o.Get("AmountReal").String(), o.Get("Amount").String()),
			"orderState":   o.Get("OrderState").String(),
		})
	}
	return rows, nil
}

func ucloudFormatOrderProduct(o gjson.Result) string {
	var parts []string
	for _, d := range o.Get("OrderDetails").Array() {
		pn := strings.TrimSpace(d.Get("ProductName").String())
		v := strings.TrimSpace(d.Get("Value").String())
		if pn != "" && v != "" {
			parts = append(parts, pn+"/"+v)
		} else if pn != "" {
			parts = append(parts, pn)
		}
	}
	if len(parts) > 0 {
		return strings.Join(parts, ", ")
	}
	return o.Get("OrderType").String()
}

func ucloudFormatUnixTime(ts int64) string {
	if ts <= 0 {
		return ""
	}
	return time.Unix(ts, 0).In(time.Local).Format("2006-01-02 15:04:05")
}

func firstNonEmptyStr(a, b string) string {
	a, b = strings.TrimSpace(a), strings.TrimSpace(b)
	if a != "" {
		return a
	}
	return b
}

func ucloudListInstances(cr *model.CloudServerCredential) (map[string]interface{}, error) {
	pub := strings.TrimSpace(cr.AccessKey)
	priv := strings.TrimSpace(cr.SecretKey)
	explicit := strings.TrimSpace(gjson.Get(cr.ExtraJSON, "ucloud_project_id").String())
	if explicit == "" {
		explicit = strings.TrimSpace(gjson.Get(cr.ExtraJSON, "project_id").String())
	}
	if pub == "" || priv == "" {
		return nil, fmt.Errorf("UCloud 需填写 PublicKey(AccessKey) 与 PrivateKey(SecretKey)")
	}

	regions := regionsFromExtra(cr.ExtraJSON, "ucloud")
	apiBase := ucloudAPIBaseCN
	if !ucloudRegionsExplicitInExtra(cr.ExtraJSON) {
		if codes, b, err := ucloudFetchAccessibleRegionCodes(pub, priv); err == nil && len(codes) > 0 {
			regions = codes
			apiBase = b
		}
	} else {
		if _, b, err := ucloudFetchAccessibleRegionCodes(pub, priv); err == nil && b != "" {
			apiBase = b
		}
	}

	var projects []string
	if explicit != "" {
		projects = []string{explicit}
	} else {
		ids, err := ucloudFetchProjectIDs(pub, priv, regions, apiBase)
		if err != nil {
			projects = []string{""}
		} else if len(ids) == 0 {
			projects = []string{""}
		} else {
			projects = ids
		}
	}

	byRegion := make(map[string][]csInstanceRow)
	seen := make(map[string]bool)
	var lastErr error
	for _, rid := range regions {
		rid = strings.TrimSpace(rid)
		if rid == "" {
			continue
		}
		for _, pid := range projects {
			hosts, errU := ucloudDescribeUHostWithRegionFallback(pub, priv, pid, rid, apiBase)
			if errU != nil {
				lastErr = errU
			}
			ulhosts, errL := ucloudDescribeULHostWithRegionFallback(pub, priv, pid, rid, apiBase)
			if errL != nil && errU == nil {
				lastErr = errL
			}
			phosts, errP := ucloudDescribePHostWithRegionFallback(pub, priv, pid, rid, apiBase)
			if errP != nil && errU == nil && errL == nil {
				lastErr = errP
			}
			for _, h := range hosts {
				k := rid + "\x00" + h.InstanceID
				if h.InstanceID == "" || seen[k] {
					continue
				}
				seen[k] = true
				byRegion[rid] = append(byRegion[rid], h)
			}
			for _, h := range ulhosts {
				k := rid + "\x00" + h.InstanceID
				if h.InstanceID == "" || seen[k] {
					continue
				}
				seen[k] = true
				byRegion[rid] = append(byRegion[rid], h)
			}
			for _, h := range phosts {
				k := rid + "\x00" + h.InstanceID
				if h.InstanceID == "" || seen[k] {
					continue
				}
				seen[k] = true
				byRegion[rid] = append(byRegion[rid], h)
			}
		}
	}

	var blocks []csRegionBlock
	for _, rid := range regions {
		rid = strings.TrimSpace(rid)
		if rid == "" {
			continue
		}
		if inst, ok := byRegion[rid]; ok && len(inst) > 0 {
			blocks = append(blocks, csRegionBlock{
				RegionID:   rid,
				RegionName: rid,
				Instances:  inst,
			})
		}
	}
	if len(blocks) == 0 && lastErr != nil {
		return nil, fmt.Errorf("UCloud 列举主机失败（香港等资源请确认控制台 API 所属线路；可设扩展 JSON：{\"regions\":[\"hk\"],\"ucloud_project_id\":\"项目ID\"}）：%w", lastErr)
	}
	if len(blocks) == 0 {
		return nil, fmt.Errorf("UCloud 未查询到计算实例（已尝试 DescribeUHostInstance、DescribeULHostInstance、DescribePHost；产品索引见 https://docs.ucloud.cn/api ）。未配置 regions 时已根据 GetRegion 拉取可用地域；若仍为空请填写 ucloud_project_id，并在 regions 中指定控制台「地域」代码（香港机房一般为 hk，与可用区如 hk-02 不同）")
	}
	return map[string]interface{}{"regions": blocks}, nil
}

func ucloudErrIsRetryableRegion(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "not available") ||
		strings.Contains(s, "invalid region") ||
		strings.Contains(s, "invalidregion")
}

// GetProjectList 的 Region 必须为主账号在控制台可见、且该 API 支持的地域；不能仅用「仅有香港资源」时的 hk 作为唯一锚点。
func ucloudGetProjectListAnchorCandidates(userRegions []string) []string {
	core := []string{"cn-bj2", "cn-sh2", "cn-gd", "cn-wlcb", "hk", "cn-hk", "sg", "jpn-tky", "us-ca", "tw-ka"}
	seen := map[string]bool{}
	var out []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		out = append(out, s)
	}
	for _, s := range core {
		add(s)
	}
	for _, s := range userRegions {
		add(s)
	}
	return out
}

// ucloudFetchProjectIDs 在未配置 project 时，依次用多个 Region 调 GetProjectList，直到成功拿到项目列表。
func ucloudFetchProjectIDs(pub, priv string, userRegions []string, apiBase string) ([]string, error) {
	var lastErr error
	for _, anchor := range ucloudGetProjectListAnchorCandidates(userRegions) {
		ids, err := ucloudFetchProjectIDsOnce(pub, priv, anchor, apiBase)
		if err != nil {
			lastErr = err
			if ucloudErrIsRetryableRegion(err) {
				continue
			}
			return nil, err
		}
		if len(ids) > 0 {
			return ids, nil
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, nil
}

func ucloudFetchProjectIDsOnce(pub, priv, region, apiBase string) ([]string, error) {
	region = strings.TrimSpace(region)
	if region == "" {
		region = "cn-bj2"
	}
	params := map[string]string{
		"Action":      "GetProjectList",
		"PublicKey":   pub,
		"Region":      region,
		"RequestUUID": strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	sig, err := ucloudSignature(priv, params)
	if err != nil {
		return nil, err
	}
	params["Signature"] = sig
	body, err := ucloudAPIGETWithBase(apiBase, params)
	if err != nil {
		return nil, err
	}
	if gjson.GetBytes(body, "RetCode").Int() != 0 {
		return nil, fmt.Errorf("UCloud API: %s", gjson.GetBytes(body, "Message").String())
	}
	raw := gjson.GetBytes(body, "ProjectSet").Array()
	if len(raw) == 0 {
		raw = gjson.GetBytes(body, "Data.ProjectSet").Array()
	}
	var ids []string
	for _, p := range raw {
		id := strings.TrimSpace(p.Get("ProjectId").String())
		if id == "" {
			id = strings.TrimSpace(p.Get("ProjectID").String())
		}
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// DescribeUHostInstance / DescribeULHostInstance 在不同文档/线路下香港等地域代号可能为 hk、cn-hk 等，依次尝试。
func ucloudDescribeRegionCandidates(logical string) []string {
	logical = strings.TrimSpace(logical)
	u := strings.ToLower(logical)
	seen := map[string]bool{}
	var out []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		out = append(out, s)
	}
	add(logical)
	switch u {
	case "hk", "hongkong", "cn-hk", "ap-hongkong", "香港":
		add("hk")
		add("cn-hk")
		add("ap-hongkong")
	}
	return out
}

func ucloudDescribeUHostWithRegionFallback(publicKey, privateKey, projectID, logicalRegion, apiBase string) ([]csInstanceRow, error) {
	var lastErr error
	for _, r := range ucloudDescribeRegionCandidates(logicalRegion) {
		hosts, err := ucloudDescribeUHost(publicKey, privateKey, projectID, r, apiBase)
		if err != nil {
			lastErr = err
			if ucloudErrIsRetryableRegion(err) {
				continue
			}
			return nil, err
		}
		return hosts, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("UCloud API: 无可用地域候选")
}

func ucloudDescribeUHost(publicKey, privateKey, projectID, region, apiBase string) ([]csInstanceRow, error) {
	const pageSize = 50
	var all []csInstanceRow
	offset := 0
	for page := 0; page < 200; page++ {
		params := map[string]string{
			"Action":      "DescribeUHostInstance",
			"PublicKey":   publicKey,
			"Region":      region,
			"Limit":       strconv.Itoa(pageSize),
			"Offset":      strconv.Itoa(offset),
			"ProjectId":   projectID,
			"RequestUUID": strconv.FormatInt(time.Now().UnixNano()+int64(page), 10),
		}
		if strings.TrimSpace(projectID) == "" {
			delete(params, "ProjectId")
		}
		sig, err := ucloudSignature(privateKey, params)
		if err != nil {
			return nil, err
		}
		params["Signature"] = sig
		q := url.Values{}
		keys := make([]string, 0, len(params))
		for k := range params {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			q.Set(k, params[k])
		}
		base := strings.TrimSpace(apiBase)
		if base == "" {
			base = ucloudAPIBaseCN
		}
		reqURL := strings.TrimRight(base, "/") + "/?" + q.Encode()
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			cancel()
			return nil, err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			cancel()
			return nil, err
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		cancel()
		if err != nil {
			return nil, err
		}
		if gjson.GetBytes(body, "RetCode").Int() != 0 {
			return nil, fmt.Errorf("UCloud API: %s", gjson.GetBytes(body, "Message").String())
		}
		arr := ucloudUHostArrayFromResponse(body)
		for _, h := range arr {
			pubIP, privIP := ucloudPickIP(h)
			uhid := strings.TrimSpace(h.Get("UHostId").String())
			if uhid == "" {
				uhid = strings.TrimSpace(h.Get("ResourceId").String())
			}
			name := strings.TrimSpace(h.Get("Name").String())
			if name == "" {
				name = strings.TrimSpace(h.Get("UHostName").String())
			}
			cpu := h.Get("CPU").String()
			mem := h.Get("Memory").String()
			if cpu == "" {
				cpu = h.Get("Cpu").String()
			}
			all = append(all, csInstanceRow{
				InstanceID: uhid,
				HostName:   name,
				PublicIP:   pubIP,
				PrivateIP:  privIP,
				Config:     cpu + "核 " + mem + "MB",
				OS:         firstNonEmptyStr(h.Get("OsName").String(), h.Get("OSName").String()),
				Status:     firstNonEmptyStr(h.Get("State").String(), h.Get("Status").String()),
				CreatedAt:  firstNonEmptyStr(h.Get("CreateTime").String(), h.Get("CreateTimeUnix").String()),
				ExpiredAt:  h.Get("ExpireTime").String(),
			})
		}
		total := int(gjson.GetBytes(body, "TotalCount").Int())
		if len(arr) < pageSize || (total > 0 && len(all) >= total) {
			break
		}
		offset += pageSize
	}
	return all, nil
}

func ucloudDescribeULHostWithRegionFallback(publicKey, privateKey, projectID, logicalRegion, apiBase string) ([]csInstanceRow, error) {
	var lastErr error
	for _, r := range ucloudDescribeRegionCandidates(logicalRegion) {
		hosts, err := ucloudDescribeULHost(publicKey, privateKey, projectID, r, apiBase)
		if err != nil {
			lastErr = err
			if ucloudErrIsRetryableRegion(err) {
				continue
			}
			return nil, err
		}
		return hosts, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("UCloud API: 无可用地域候选")
}

// ucloudDescribeULHost 调用 DescribeULHostInstance（轻量应用云主机 ULightHost），与 UHost 并列列举。
// 文档：https://docs.ucloud.cn/api/ulighthost-api/README
func ucloudDescribeULHost(publicKey, privateKey, projectID, region, apiBase string) ([]csInstanceRow, error) {
	const pageSize = 50
	var all []csInstanceRow
	offset := 0
	for page := 0; page < 200; page++ {
		params := map[string]string{
			"Action":      "DescribeULHostInstance",
			"PublicKey":   publicKey,
			"Region":      region,
			"Limit":       strconv.Itoa(pageSize),
			"Offset":      strconv.Itoa(offset),
			"ProjectId":   projectID,
			"RequestUUID": strconv.FormatInt(time.Now().UnixNano()+int64(page), 10),
		}
		if strings.TrimSpace(projectID) == "" {
			delete(params, "ProjectId")
		}
		sig, err := ucloudSignature(privateKey, params)
		if err != nil {
			return nil, err
		}
		params["Signature"] = sig
		q := url.Values{}
		keys := make([]string, 0, len(params))
		for k := range params {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			q.Set(k, params[k])
		}
		base := strings.TrimSpace(apiBase)
		if base == "" {
			base = ucloudAPIBaseCN
		}
		reqURL := strings.TrimRight(base, "/") + "/?" + q.Encode()
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			cancel()
			return nil, err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			cancel()
			return nil, err
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		cancel()
		if err != nil {
			return nil, err
		}
		if gjson.GetBytes(body, "RetCode").Int() != 0 {
			return nil, fmt.Errorf("UCloud API: %s", gjson.GetBytes(body, "Message").String())
		}
		arr := ucloudULHostArrayFromResponse(body)
		for _, h := range arr {
			pubIP, privIP := ucloudPickIP(h)
			id := strings.TrimSpace(h.Get("ULHostId").String())
			if id == "" {
				id = strings.TrimSpace(h.Get("UHostId").String())
			}
			if id == "" {
				id = strings.TrimSpace(h.Get("ResourceId").String())
			}
			name := strings.TrimSpace(h.Get("Name").String())
			if name == "" {
				name = strings.TrimSpace(h.Get("ULHostName").String())
			}
			cpu := h.Get("CPU").String()
			mem := h.Get("Memory").String()
			if cpu == "" {
				cpu = h.Get("Cpu").String()
			}
			createdAt := ""
			if ts := h.Get("CreateTime").Int(); ts > 0 {
				createdAt = ucloudFormatUnixTime(ts)
			}
			if createdAt == "" {
				createdAt = firstNonEmptyStr(h.Get("CreateTime").String(), h.Get("CreateTimeUnix").String())
			}
			expiredAt := ""
			if ts := h.Get("ExpireTime").Int(); ts > 0 {
				expiredAt = ucloudFormatUnixTime(ts)
			}
			if expiredAt == "" {
				expiredAt = strings.TrimSpace(h.Get("ExpireTime").String())
			}
			all = append(all, csInstanceRow{
				InstanceID: id,
				HostName:   name,
				PublicIP:   pubIP,
				PrivateIP:  privIP,
				Config:     "轻量 " + cpu + "核 " + mem + "MB",
				OS:         firstNonEmptyStr(h.Get("ImageName").String(), firstNonEmptyStr(h.Get("OsName").String(), h.Get("OSName").String())),
				Status:     firstNonEmptyStr(h.Get("State").String(), h.Get("Status").String()),
				CreatedAt:  createdAt,
				ExpiredAt:  expiredAt,
			})
		}
		total := int(gjson.GetBytes(body, "TotalCount").Int())
		if len(arr) < pageSize || (total > 0 && len(all) >= total) {
			break
		}
		offset += pageSize
	}
	return all, nil
}

func ucloudDescribePHostWithRegionFallback(publicKey, privateKey, projectID, logicalRegion, apiBase string) ([]csInstanceRow, error) {
	var lastErr error
	for _, r := range ucloudDescribeRegionCandidates(logicalRegion) {
		hosts, err := ucloudDescribePHost(publicKey, privateKey, projectID, r, apiBase)
		if err != nil {
			lastErr = err
			if ucloudErrIsRetryableRegion(err) {
				continue
			}
			return nil, err
		}
		return hosts, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("UCloud API: 无可用地域候选")
}

// ucloudDescribePHost 调用 DescribePHost（物理云主机 UPHost）。
// 文档：https://docs.ucloud.cn/api/uphost-api/README
func ucloudDescribePHost(publicKey, privateKey, projectID, region, apiBase string) ([]csInstanceRow, error) {
	const pageSize = 50
	var all []csInstanceRow
	offset := 0
	for page := 0; page < 200; page++ {
		params := map[string]string{
			"Action":      "DescribePHost",
			"PublicKey":   publicKey,
			"Region":      region,
			"Limit":       strconv.Itoa(pageSize),
			"Offset":      strconv.Itoa(offset),
			"ProjectId":   projectID,
			"RequestUUID": strconv.FormatInt(time.Now().UnixNano()+int64(page), 10),
		}
		if strings.TrimSpace(projectID) == "" {
			delete(params, "ProjectId")
		}
		sig, err := ucloudSignature(privateKey, params)
		if err != nil {
			return nil, err
		}
		params["Signature"] = sig
		q := url.Values{}
		keys := make([]string, 0, len(params))
		for k := range params {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			q.Set(k, params[k])
		}
		base := strings.TrimSpace(apiBase)
		if base == "" {
			base = ucloudAPIBaseCN
		}
		reqURL := strings.TrimRight(base, "/") + "/?" + q.Encode()
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			cancel()
			return nil, err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			cancel()
			return nil, err
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		cancel()
		if err != nil {
			return nil, err
		}
		if gjson.GetBytes(body, "RetCode").Int() != 0 {
			return nil, fmt.Errorf("UCloud API: %s", gjson.GetBytes(body, "Message").String())
		}
		arr := ucloudPHostArrayFromResponse(body)
		for _, h := range arr {
			pubIP, privIP := ucloudPickIP(h)
			id := strings.TrimSpace(h.Get("PHostId").String())
			name := strings.TrimSpace(h.Get("Name").String())
			cpuSet := h.Get("CPUSet")
			cores := int(cpuSet.Get("CoreCount").Int())
			if cores <= 0 {
				cores = int(cpuSet.Get("Count").Int())
			}
			memStr := strconv.FormatInt(h.Get("Memory").Int(), 10)
			if memStr == "0" {
				memStr = strings.TrimSpace(h.Get("Memory").String())
			}
			cfg := "裸金属"
			if cores > 0 && memStr != "" {
				cfg = "裸金属 " + strconv.Itoa(cores) + "核 " + memStr + "MB"
			} else if cores > 0 {
				cfg = "裸金属 " + strconv.Itoa(cores) + "核"
			} else if memStr != "" {
				cfg = "裸金属 " + memStr + "MB"
			}
			createdAt := ""
			if ts := h.Get("CreateTime").Int(); ts > 0 {
				createdAt = ucloudFormatUnixTime(ts)
			}
			expiredAt := ""
			if ts := h.Get("ExpireTime").Int(); ts > 0 {
				expiredAt = ucloudFormatUnixTime(ts)
			}
			all = append(all, csInstanceRow{
				InstanceID: id,
				HostName:   name,
				PublicIP:   pubIP,
				PrivateIP:  privIP,
				Config:     cfg,
				OS:         firstNonEmptyStr(h.Get("ImageName").String(), firstNonEmptyStr(h.Get("OSname").String(), h.Get("OSType").String())),
				Status:     firstNonEmptyStr(h.Get("PMStatus").String(), h.Get("PowerState").String()),
				CreatedAt:  createdAt,
				ExpiredAt:  expiredAt,
			})
		}
		total := int(gjson.GetBytes(body, "TotalCount").Int())
		if len(arr) < pageSize || (total > 0 && len(all) >= total) {
			break
		}
		offset += pageSize
	}
	return all, nil
}

func ucloudPHostArrayFromResponse(body []byte) []gjson.Result {
	for _, path := range []string{"PHostSet", "Data.PHostSet"} {
		if arr := gjson.GetBytes(body, path).Array(); len(arr) > 0 {
			return arr
		}
	}
	return nil
}

func ucloudULHostArrayFromResponse(body []byte) []gjson.Result {
	// 国际站文档字段为 ULHostInstanceSets；部分线路可能沿用 UHostSet 等别名。
	for _, path := range []string{"ULHostInstanceSets", "Data.ULHostInstanceSets", "ULHostSet", "Data.ULHostSet", "UHostSet", "Data.UHostSet", "DataSet", "ULHosts"} {
		if arr := gjson.GetBytes(body, path).Array(); len(arr) > 0 {
			return arr
		}
	}
	return nil
}

func ucloudUHostArrayFromResponse(body []byte) []gjson.Result {
	for _, path := range []string{"UHostSet", "Data.UHostSet", "DataSet", "UHosts"} {
		if arr := gjson.GetBytes(body, path).Array(); len(arr) > 0 {
			return arr
		}
	}
	return nil
}

func ucloudFirstIPString(ip gjson.Result) string {
	for _, k := range []string{"IPAddr", "IP", "Ip", "IPv4", "IpAddress"} {
		if s := strings.TrimSpace(ip.Get(k).String()); s != "" {
			return s
		}
	}
	return ""
}

func ucloudPickIP(h gjson.Result) (pub, priv string) {
	arrs := [][]gjson.Result{
		h.Get("IPSet").Array(),
		h.Get("IPInfos").Array(),
	}
	for _, ips := range arrs {
		for _, ip := range ips {
			addr := ucloudFirstIPString(ip)
			if addr == "" {
				continue
			}
			t := strings.ToLower(strings.TrimSpace(ip.Get("Type").String()))
			if t == "" {
				t = strings.ToLower(strings.TrimSpace(ip.Get("OperatorName").String()))
			}
			// 文档与历史版本：Private / International(BGP) / Elastic 等；PHost 为 OperatorName
			switch t {
			case "private", "virtual", "subnet", "intranet":
				if priv == "" {
					priv = addr
				}
			case "international", "internation", "bgp", "duplet", "elastic", "eip", "default":
				pub = addr
			default:
				if ucloudLooksLikePrivateIPv4(addr) {
					if priv == "" {
						priv = addr
					}
				} else if pub == "" {
					pub = addr
				}
			}
		}
	}
	return pub, priv
}

func ucloudLooksLikePrivateIPv4(addr string) bool {
	if !strings.Contains(addr, ".") {
		return false
	}
	parts := strings.Split(addr, ".")
	if len(parts) != 4 {
		return false
	}
	a, err1 := strconv.Atoi(parts[0])
	b, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return false
	}
	if a == 10 {
		return true
	}
	if a == 172 && b >= 16 && b <= 31 {
		return true
	}
	if a == 192 && b == 168 {
		return true
	}
	return false
}

// ucloudIAMUpdateLoginPassword 在 CreateUser 之后设置子账号控制台密码（UpdateLoginProfile）。
func ucloudIAMUpdateLoginPassword(publicKey, privateKey, projectID, userName, password string) error {
	userName = strings.TrimSpace(userName)
	password = strings.TrimSpace(password)
	if userName == "" || password == "" {
		return fmt.Errorf("UserName 或 Password 为空")
	}
	params := map[string]string{
		"Action":                "UpdateLoginProfile",
		"PublicKey":             publicKey,
		"UserName":              userName,
		"Password":              password,
		"PasswordResetRequired": "false",
		"RequestUUID":           strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	if strings.TrimSpace(projectID) != "" {
		params["ProjectId"] = strings.TrimSpace(projectID)
	}
	sig, err := ucloudSignature(privateKey, params)
	if err != nil {
		return err
	}
	params["Signature"] = sig
	body, err := ucloudAPIGET(params)
	if err != nil {
		return err
	}
	if gjson.GetBytes(body, "RetCode").Int() != 0 {
		return fmt.Errorf("%s", gjson.GetBytes(body, "Message").String())
	}
	return nil
}

// ucloudSignature 与 UCloud 官方 SDK 一致：排序后的「参数名+参数值」拼接，末尾追加 PrivateKey，SHA1 后输出小写十六进制（非 Base64）。
// 参考：https://github.com/ucloud/ucloud-sdk-go/blob/master/ucloud/auth/signature.go
func ucloudSignature(privateKey string, params map[string]string) (string, error) {
	privateKey = strings.TrimSpace(privateKey)
	if privateKey == "" {
		return "", fmt.Errorf("PrivateKey 为空")
	}
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "Signature" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteString(params[k])
	}
	hash := sha1.Sum([]byte(sb.String() + privateKey))
	return hex.EncodeToString(hash[:]), nil
}
