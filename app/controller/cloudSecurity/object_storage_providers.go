package cloudSecurity

import (
	"EasyTools/app/model"
	proxy "EasyTools/app/controller/proxy"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	obs "github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
	"github.com/qiniu/go-sdk/v7/auth"
	"github.com/qiniu/go-sdk/v7/storage"
	"github.com/tencentyun/cos-go-sdk-v5"
	"github.com/upyun/go-sdk/v3/upyun"
	qsconfig "github.com/yunify/qingstor-sdk-go/v3/config"
	qsservice "github.com/yunify/qingstor-sdk-go/v3/service"
)

type osBucketRow struct {
	Name string `json:"name"`
	Tag  string `json:"tag,omitempty"`
}

type osObjectRow struct {
	Key          string `json:"key"`
	Size         int64  `json:"size"`
	LastModified string `json:"last_modified"`
	IsDir        bool   `json:"is_dir"`
}

type osListResult struct {
	Objects        []osObjectRow `json:"objects"`
	CommonPrefixes []string      `json:"common_prefixes"`
	Truncated      bool          `json:"truncated"`
	NextMarker     string        `json:"next_marker"`
}

func parseExtra(c *model.ObjectStorageCredential) map[string]interface{} {
	m := map[string]interface{}{}
	if strings.TrimSpace(c.ExtraJSON) == "" {
		return m
	}
	_ = json.Unmarshal([]byte(c.ExtraJSON), &m)
	return m
}

func credSessionToken(c *model.ObjectStorageCredential) string {
	if c == nil {
		return ""
	}
	return strings.TrimSpace(c.SessionToken)
}

func ossNewAliyun(endpoint, ak, sk, sessionToken string) (*oss.Client, error) {
	ep := strings.TrimRight(strings.TrimSpace(endpoint), "/")
	var opts []oss.ClientOption
	if t := strings.TrimSpace(sessionToken); t != "" {
		opts = append(opts, oss.SecurityToken(t))
	}
	return oss.New(ep, strings.TrimSpace(ak), strings.TrimSpace(sk), opts...)
}

func newCosHTTPClient(ak, sk, sessionToken string) *http.Client {
	tr := &cos.AuthorizationTransport{
		SecretID:  strings.TrimSpace(ak),
		SecretKey: strings.TrimSpace(sk),
	}
	if t := strings.TrimSpace(sessionToken); t != "" {
		tr.SessionToken = t
	}
	// 复用全局代理上的底层 Transport（含 HTTP/SOCKS5 代理与连接池配置）。
	if base := proxy.GetGlobalHTTPClient(); base != nil && base.Transport != nil {
		tr.Transport = base.Transport
	}
	return cosHTTPClient(tr)
}

func cosNewClient(base *cos.BaseURL, ak, sk, sessionToken string) *cos.Client {
	return cos.NewClient(base, newCosHTTPClient(ak, sk, sessionToken))
}

// cosPresignOpt 临时密钥预签名 URL 需带 x-cos-security-token；长期密钥返回 nil。
func cosPresignOpt(sessionToken string) interface{} {
	st := strings.TrimSpace(sessionToken)
	if st == "" {
		return nil
	}
	q := &url.Values{}
	q.Add("x-cos-security-token", st)
	return &cos.PresignedURLOptions{Query: q, Header: &http.Header{}}
}

func extraStr(m map[string]interface{}, k string) string {
	v, ok := m[k]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		return strconv.FormatInt(int64(t), 10)
	case bool:
		return strconv.FormatBool(t)
	default:
		return fmt.Sprint(t)
	}
}

func extraBool(m map[string]interface{}, k string, def bool) bool {
	v, ok := m[k]
	if !ok || v == nil {
		return def
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return strings.EqualFold(t, "true") || t == "1"
	default:
		return def
	}
}

// tencentCosBucketRegion 解析桶所在地域：优先扩展 JSON 的 region，否则使用列举/选桶时返回的 bucketTag（ListBuckets 的 Location）。
func tencentCosBucketRegion(ex map[string]interface{}, bucketTag string) (string, error) {
	rg := strings.TrimSpace(extraStr(ex, "region"))
	if rg != "" {
		return rg, nil
	}
	bt := strings.TrimSpace(bucketTag)
	if bt != "" {
		return bt, nil
	}
	return "", fmt.Errorf("腾讯云需桶地域：创建新桶时请在「新建存储桶」对话框填写地域，或在扩展 JSON 填写 region（如 ap-guangzhou）；已有桶可刷新列表后点选桶以自动带上地域")
}

// 与 cos-go-sdk-v5 虚拟主机域名一致：https://{名称-APPID}.cos.{地域}.myqcloud.com（见 NewBucketURL / checkURL）。
var (
	tencentCosVirtualHostBucketNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*-[0-9]+$`)
	tencentCosAppIDDigitsOnlyRe     = regexp.MustCompile(`^[0-9]+$`)
	tencentCosBucketShortNameRe     = regexp.MustCompile(`^[a-z0-9-]+$`)
)

// tencentCosNormalizeBucketVirtualHostName 将桶名规范为虚拟主机中的「桶名」段（自定义名-APPID）。仅短名时需扩展 JSON 提供 appid / cos_app_id / uin。
func tencentCosNormalizeBucketVirtualHostName(bucketName string, ex map[string]interface{}) (string, error) {
	name := strings.TrimSpace(strings.ToLower(bucketName))
	if name == "" {
		return "", fmt.Errorf("存储桶名称不能为空")
	}
	if tencentCosVirtualHostBucketNameRe.MatchString(name) {
		return name, nil
	}
	appid := strings.TrimSpace(extraStr(ex, "appid"))
	if appid == "" {
		appid = strings.TrimSpace(extraStr(ex, "cos_app_id"))
	}
	if appid == "" {
		appid = strings.TrimSpace(extraStr(ex, "uin"))
	}
	if !tencentCosAppIDDigitsOnlyRe.MatchString(appid) {
		return "", fmt.Errorf("腾讯云 COS 域名为「名称-APPID.cos.地域.myqcloud.com」；请使用控制台完整桶名（如 mybucket-1250000000），或在扩展 JSON 填写 appid/uin/cos_app_id（纯数字 APPID）以便拼接")
	}
	if !tencentCosBucketShortNameRe.MatchString(name) {
		return "", fmt.Errorf("COS 桶短名仅允许小写字母、数字、连字符")
	}
	if len(name) > 0 && name[0] >= '0' && name[0] <= '9' {
		return "", fmt.Errorf("COS 自定义桶名片段不能以数字开头")
	}
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return "", fmt.Errorf("桶名不能以连字符开头或结尾")
	}
	full := name + "-" + appid
	if len(full) > 60 {
		return "", fmt.Errorf("拼接后的完整桶名超过 COS 长度限制（≤60）")
	}
	return full, nil
}

// tencentCosVirtualHostedBucketURL 生成 cos-go-sdk 认可的 BucketURL。
func tencentCosVirtualHostedBucketURL(bucketName, region string, ex map[string]interface{}) (*url.URL, error) {
	normalized, err := tencentCosNormalizeBucketVirtualHostName(bucketName, ex)
	if err != nil {
		return nil, err
	}
	rg := strings.TrimSpace(strings.ToLower(region))
	u, err := cos.NewBucketURL(normalized, rg, true)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func clampMaxKeys(n int) int {
	if n <= 0 {
		return 100
	}
	if n > 1000 {
		return 1000
	}
	return n
}

func md5HexLower(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

func newS3Client(ctx context.Context, ak, sk, region, endpoint string, pathStyle bool, sessionToken string) (*s3.Client, error) {
	if region == "" {
		region = "us-east-1"
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(ak, sk, strings.TrimSpace(sessionToken))),
	)
	if err != nil {
		return nil, err
	}
	ep := strings.TrimRight(strings.TrimSpace(endpoint), "/")
	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		if ep != "" {
			o.BaseEndpoint = aws.String(ep)
		}
		o.UsePathStyle = pathStyle
	}), nil
}

func s3ListBucketsRows(ctx context.Context, cli *s3.Client) ([]osBucketRow, error) {
	outResp, err := cli.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}
	out := make([]osBucketRow, 0, len(outResp.Buckets))
	for _, b := range outResp.Buckets {
		if b.Name != nil {
			out = append(out, osBucketRow{Name: *b.Name})
		}
	}
	return out, nil
}

func s3ListObjectsV2Rows(ctx context.Context, cli *s3.Client, bucket, prefix, continuation string, mk int, hierarchical bool) (osListResult, error) {
	var empty osListResult
	in := &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int32(int32(mk)),
	}
	if hierarchical {
		in.Delimiter = aws.String("/")
	}
	if continuation != "" {
		in.ContinuationToken = aws.String(continuation)
	}
	out, err := cli.ListObjectsV2(ctx, in)
	if err != nil {
		return empty, err
	}
	res := osListResult{Truncated: aws.ToBool(out.IsTruncated), NextMarker: aws.ToString(out.NextContinuationToken)}
	for _, cp := range out.CommonPrefixes {
		if cp.Prefix != nil {
			res.CommonPrefixes = append(res.CommonPrefixes, *cp.Prefix)
		}
	}
	for _, o := range out.Contents {
		lm := ""
		if o.LastModified != nil {
			lm = o.LastModified.Format(time.RFC3339)
		}
		sz := int64(0)
		if o.Size != nil {
			sz = *o.Size
		}
		k := aws.ToString(o.Key)
		res.Objects = append(res.Objects, osObjectRow{Key: k, Size: sz, LastModified: lm, IsDir: strings.HasSuffix(k, "/")})
	}
	return res, nil
}

func resolveS3Endpoint(provider string, ex map[string]interface{}) (endpoint string, region string, pathStyle bool) {
	region = extraStr(ex, "region")
	endpoint = extraStr(ex, "endpoint")
	pathStyle = extraBool(ex, "path_style", false)
	switch provider {
	case "minio":
		pathStyle = true
		if region == "" {
			region = "us-east-1"
		}
	case "jd":
		pathStyle = true
		if endpoint == "" && region != "" {
			endpoint = "https://s3." + region + ".jdcloud-oss.com"
		}
	case "ks3":
		pathStyle = true
		if endpoint == "" && region != "" {
			endpoint = "https://ks3-" + region + ".ksyuncs.com"
		}
	case "aws":
		if region == "" {
			region = "us-east-1"
		}
	}
	return endpoint, region, pathStyle
}

// resolveS3EndpointForBucketLifecycle 创建/删除存储桶：扩展 JSON 未填写 region 时，可使用 bucketTag 传入地域（与前端「新建桶」对话框一致）；京东云/金山云会据地域补全 endpoint。
func resolveS3EndpointForBucketLifecycle(provider string, ex map[string]interface{}, bucketTag string) (endpoint string, region string, pathStyle bool) {
	endpoint, region, pathStyle = resolveS3Endpoint(provider, ex)
	exRegion := strings.TrimSpace(extraStr(ex, "region"))
	if tag := strings.TrimSpace(bucketTag); tag != "" && exRegion == "" {
		region = tag
	}
	p := strings.ToLower(strings.TrimSpace(provider))
	switch p {
	case "jd":
		if endpoint == "" && region != "" {
			endpoint = "https://s3." + region + ".jdcloud-oss.com"
		}
	case "ks3":
		if endpoint == "" && region != "" {
			endpoint = "https://ks3-" + region + ".ksyuncs.com"
		}
	}
	return endpoint, region, pathStyle
}

// aliyunOssProbeRegionIDs 公网 OSS 常用地域（用于未配置 endpoint/region 时依次探测 ListBuckets）。
// 专有云、内网域名等不在此列，需在扩展 JSON 填写 endpoint。
var aliyunOssProbeRegionIDs = []string{
	"cn-hangzhou", "cn-shanghai", "cn-beijing", "cn-shenzhen", "cn-guangzhou",
	"cn-chengdu", "cn-hongkong", "cn-qingdao", "cn-zhangjiakou", "cn-huhehaote",
	"cn-wulanchabu", "cn-heyuan", "cn-nanjing", "cn-fuzhou", "cn-wuhan",
	"ap-southeast-1", "ap-southeast-2", "ap-southeast-3", "ap-southeast-5", "ap-southeast-6", "ap-southeast-7",
	"ap-northeast-1", "ap-northeast-2", "ap-south-1",
	"us-east-1", "us-west-1",
	"eu-central-1", "eu-west-1",
	"me-east-1",
	"cn-hangzhou-finance-1", "cn-shanghai-finance-1", "cn-shenzhen-finance-1",
	"cn-north-2-gov-1",
}

// resolveAliyunOssEndpointFromExtra 解析扩展 JSON 中的 endpoint 或 region；均未配置时 ok=false。
func resolveAliyunOssEndpointFromExtra(ex map[string]interface{}) (ep string, ok bool, err error) {
	ep = strings.TrimSpace(extraStr(ex, "endpoint"))
	if ep != "" {
		return strings.TrimRight(ep, "/"), true, nil
	}
	rg := strings.TrimSpace(strings.ToLower(extraStr(ex, "region")))
	if rg == "" {
		return "", false, nil
	}
	rg = strings.TrimPrefix(rg, "oss-")
	if rg == "" {
		return "", true, fmt.Errorf("region 无效")
	}
	return "https://oss-" + rg + ".aliyuncs.com", true, nil
}

// aliyunOssEndpointFromBucketTag 由 ListBuckets 返回的 Location/Region（如 oss-cn-hangzhou）拼公网 endpoint。
func aliyunOssEndpointFromBucketTag(tag string) (string, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return "", fmt.Errorf("桶地域为空")
	}
	if strings.Contains(tag, "://") {
		u, err := url.Parse(tag)
		if err != nil || u.Host == "" {
			return "", fmt.Errorf("无效的 endpoint URL")
		}
		if u.Scheme == "" {
			return "", fmt.Errorf("无效的 endpoint URL")
		}
		return u.Scheme + "://" + u.Host, nil
	}
	tl := strings.ToLower(tag)
	if strings.Contains(tl, ".aliyuncs.com") {
		host := tl
		if i := strings.Index(host, "/"); i >= 0 {
			host = host[:i]
		}
		return "https://" + strings.TrimPrefix(strings.TrimPrefix(host, "https://"), "http://"), nil
	}
	if strings.HasPrefix(tl, "oss-") {
		return "https://" + tl + ".aliyuncs.com", nil
	}
	return "https://oss-" + tl + ".aliyuncs.com", nil
}

func aliyunOssNewClientByProbing(ctx context.Context, c *model.ObjectStorageCredential) (*oss.Client, error) {
	ak, sk := strings.TrimSpace(c.AccessKey), strings.TrimSpace(c.SecretKey)
	st := credSessionToken(c)
	var lastErr error
	for _, rid := range aliyunOssProbeRegionIDs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		ep := "https://oss-" + rid + ".aliyuncs.com"
		cli, err := ossNewAliyun(ep, ak, sk, st)
		if err != nil {
			lastErr = err
			continue
		}
		_, err = cli.ListBuckets(oss.MaxKeys(1))
		if err == nil {
			return cli, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, fmt.Errorf("已遍历常用地域仍无法访问 OSS（请核对 AK/SK、STS 令牌、网络，或在扩展 JSON 填写 endpoint）: %w", lastErr)
	}
	return nil, fmt.Errorf("已遍历常用地域仍无法访问 OSS")
}

func aliyunOssListAllBucketRows(client *oss.Client) ([]osBucketRow, error) {
	var out []osBucketRow
	marker := ""
	for {
		opts := []oss.Option{oss.MaxKeys(1000)}
		if marker != "" {
			opts = append(opts, oss.Marker(marker))
		}
		ls, err := client.ListBuckets(opts...)
		if err != nil {
			return nil, err
		}
		for _, b := range ls.Buckets {
			tag := strings.TrimSpace(b.Location)
			if tag == "" {
				tag = strings.TrimSpace(b.Region)
			}
			out = append(out, osBucketRow{Name: b.Name, Tag: tag})
		}
		if !ls.IsTruncated {
			break
		}
		marker = ls.NextMarker
		if marker == "" {
			break
		}
	}
	return out, nil
}

func aliyunOssEndpointForObjectAPI(ex map[string]interface{}, bucketTag string) (string, error) {
	if bt := strings.TrimSpace(bucketTag); bt != "" {
		return aliyunOssEndpointFromBucketTag(bt)
	}
	ep, fromEx, err := resolveAliyunOssEndpointFromExtra(ex)
	if err != nil {
		return "", err
	}
	if !fromEx {
		return "", fmt.Errorf("请先「刷新桶列表」并选择存储桶（以带上桶所在地域），或在扩展 JSON 填写 region/endpoint")
	}
	return ep, nil
}

// --- Huawei Cloud OBS（正式 OBS 接口）---

// huaweiObsEndpointFromExtra 从扩展 JSON 中解析华为 OBS endpoint 或 region。
// 推荐直接配置 endpoint，如：https://obs.cn-north-4.myhuaweicloud.com
// 也支持仅配置 region/location（如 cn-north-4），此时自动拼接公网域名。
func huaweiObsEndpointFromExtra(ex map[string]interface{}) (string, error) {
	ep := strings.TrimSpace(extraStr(ex, "endpoint"))
	if ep != "" {
		if !strings.Contains(ep, "://") {
			ep = "https://" + ep
		}
		return strings.TrimRight(ep, "/"), nil
	}
	rg := strings.TrimSpace(extraStr(ex, "region"))
	if rg == "" {
		rg = strings.TrimSpace(extraStr(ex, "location"))
	}
	if rg == "" {
		return "", fmt.Errorf("华为云 OBS 未配置 endpoint 或 region/location")
	}
	return fmt.Sprintf("https://obs.%s.myhuaweicloud.com", rg), nil
}

// huaweiObsEndpointForObjectAPI 解析对象级操作使用的 endpoint：优先扩展 JSON，其次使用桶 Location/tag 推导地域。
// 这样可避免在未显式配置 endpoint/region 时，因使用探测得到的任意地域导致某些地域的桶对象操作返回 NoSuchBucket。
func huaweiObsEndpointForObjectAPI(ex map[string]interface{}, bucketTag string) (string, error) {
	// 1) 若扩展 JSON 已配置 endpoint/region/location，则与凭据探测逻辑保持一致。
	if ep, err := huaweiObsEndpointFromExtra(ex); err == nil {
		return ep, nil
	}

	// 2) 否则尝试使用列桶时返回的 Location（bucketTag）推导地域。
	bt := strings.TrimSpace(bucketTag)
	if bt == "" {
		return "", fmt.Errorf("华为云 OBS 需桶地域：请先「刷新桶列表」并点击选择存储桶（自动带上地域），或在扩展 JSON 填写 endpoint/region")
	}
	// bucketTag 一般为 cn-north-4 等地域标识；也容忍用户直接填入完整域名。
	if strings.Contains(bt, "://") {
		u, err := url.Parse(bt)
		if err != nil || u.Host == "" {
			return "", fmt.Errorf("无效的华为 OBS endpoint: %q", bt)
		}
		scheme := u.Scheme
		if scheme == "" {
			scheme = "https"
		}
		return fmt.Sprintf("%s://%s", scheme, u.Host), nil
	}
	host := bt
	// 允许直接传入 obs.cn-north-4.myhuaweicloud.com 之类的 Host。
	if !strings.Contains(host, ".") {
		host = fmt.Sprintf("obs.%s.myhuaweicloud.com", host)
	}
	if !strings.Contains(host, "://") {
		host = "https://" + host
	}
	return strings.TrimRight(host, "/"), nil
}

// 华为云 OBS 常用地域（用于仅填 AK/SK 时自动遍历尝试 ListBuckets）。
var huaweiObsProbeRegionIDs = []string{
	"cn-north-4", "cn-north-1", "cn-east-3", "cn-east-2",
	"cn-south-1", "ap-southeast-1", "ap-southeast-3",
}

// newHuaweiObsClient 基于 AK/SK/endpoint 创建 OBS 客户端，支持可选安全令牌。
func newHuaweiObsClient(c *model.ObjectStorageCredential, ex map[string]interface{}) (*obs.ObsClient, error) {
	ak := strings.TrimSpace(c.AccessKey)
	sk := strings.TrimSpace(c.SecretKey)
	if ak == "" || sk == "" {
		return nil, fmt.Errorf("华为云 OBS 需填写 AccessKey 与 SecretKey")
	}

	// 1) 优先使用扩展 JSON 中的 endpoint/region。
	endpoint, err := huaweiObsEndpointFromExtra(ex)
	if err != nil {
		// 2) 未配置 endpoint/region 时，遍历常用地域自动探测可用 endpoint。
		var lastErr error
		for _, rid := range huaweiObsProbeRegionIDs {
			ep := fmt.Sprintf("https://obs.%s.myhuaweicloud.com", strings.TrimSpace(rid))
			client, e := obs.New(ak, sk, ep)
			if e != nil {
				lastErr = e
				continue
			}
			// 尝试列一次桶验证可用性。
			if _, e = client.ListBuckets(nil); e == nil {
				return client, nil
			}
			lastErr = e
			client.Close()
		}
		if lastErr != nil {
			return nil, fmt.Errorf("已遍历常用华为 OBS 地域仍无法访问: %w", lastErr)
		}
		return nil, fmt.Errorf("已遍历常用华为 OBS 地域仍无法访问")
	}

	st := credSessionToken(c)
	if st != "" {
		return obs.New(ak, sk, endpoint, obs.WithSecurityToken(st))
	}
	return obs.New(ak, sk, endpoint)
}

func listBucketsForCred(ctx context.Context, c *model.ObjectStorageCredential) ([]osBucketRow, error) {
	p := strings.ToLower(strings.TrimSpace(c.Provider))
	ak, sk := strings.TrimSpace(c.AccessKey), strings.TrimSpace(c.SecretKey)
	ex := parseExtra(c)

	switch p {
	case "qiniu":
		if qiniuUsesS3STS(c) {
			return qiniuSTSListBucketsRows(ctx, c)
		}
		mac := auth.New(ak, sk)
		cfg := storage.Config{}
		bm := storage.NewBucketManager(mac, &cfg)
		names, err := bm.Buckets(false)
		if err != nil {
			return nil, err
		}
		out := make([]osBucketRow, 0, len(names))
		for _, n := range names {
			out = append(out, osBucketRow{Name: n})
		}
		return out, nil

	case "huawei":
		// 使用正式 OBS 接口列举桶列表，同时为后续对象操作尽量补全桶地域信息。
		if ak == "" || sk == "" {
			return nil, fmt.Errorf("华为云 OBS 需填写 AccessKey 与 SecretKey")
		}
		ex := parseExtra(c)
		client, err := newHuaweiObsClient(c, ex)
		if err != nil {
			return nil, err
		}
		defer client.Close()
		out, err := client.ListBuckets(nil)
		if err != nil {
			return nil, err
		}
		rows := make([]osBucketRow, 0, len(out.Buckets))
		for _, b := range out.Buckets {
			// 不在列举阶段逐桶查询 Location（N 个桶即 N 次 API）；地域可在选中桶后由对象接口或扩展 JSON 提供。
			rows = append(rows, osBucketRow{Name: b.Name, Tag: strings.TrimSpace(b.Location)})
		}
		return rows, nil

	case "tencent":
		// 与控制台一致：仅用 AK/SK 即可列举全部桶。官方域名 https://service.cos.myqcloud.com/
		// 可选扩展 JSON：{"service_url":"https://service.cos.tencentcos.cn"} 等国际站。
		svcURL := strings.TrimSpace(extraStr(ex, "service_url"))
		if svcURL == "" {
			svcURL = "https://service.cos.myqcloud.com"
		} else if !strings.Contains(svcURL, "://") {
			svcURL = "https://" + svcURL
		}
		u, err := url.Parse(strings.TrimRight(svcURL, "/"))
		if err != nil {
			return nil, err
		}
		client := cosNewClient(&cos.BaseURL{ServiceURL: u}, ak, sk, credSessionToken(c))
		var out []osBucketRow
		marker := ""
		for {
			opt := &cos.ServiceGetOptions{MaxKeys: 1000}
			if marker != "" {
				opt.Marker = marker
			}
			res, _, err := client.Service.Get(ctx, opt)
			if err != nil {
				return nil, err
			}
			for _, b := range res.Buckets {
				tag := strings.TrimSpace(b.Region)
				out = append(out, osBucketRow{Name: b.Name, Tag: tag})
			}
			if !res.IsTruncated {
				break
			}
			marker = strings.TrimSpace(res.NextMarker)
			if marker == "" {
				break
			}
		}
		return out, nil

	case "aliyun":
		ep, fromEx, err := resolveAliyunOssEndpointFromExtra(ex)
		if err != nil {
			return nil, err
		}
		var client *oss.Client
		if fromEx {
			client, err = ossNewAliyun(ep, ak, sk, credSessionToken(c))
			if err != nil {
				return nil, err
			}
		} else {
			client, err = aliyunOssNewClientByProbing(ctx, c)
			if err != nil {
				return nil, err
			}
		}
		return aliyunOssListAllBucketRows(client)

	case "aws", "minio", "jd", "ks3":
		endpoint, region, ps := resolveS3Endpoint(p, ex)
		if (p == "minio" || p == "jd" || p == "ks3") && endpoint == "" {
			return nil, fmt.Errorf("请填写 endpoint 或 region（用于拼接 S3 兼容地址）")
		}
		cli, err := newS3Client(ctx, ak, sk, region, endpoint, ps, credSessionToken(c))
		if err != nil {
			return nil, err
		}
		return s3ListBucketsRows(ctx, cli)

	case "qingstor":
		if err := objectStorageRejectSTSUnsupported(c, "qingstor"); err != nil {
			return nil, err
		}
		conf, err := qsconfig.New(ak, sk)
		if err != nil {
			return nil, err
		}
		svc, err := qsservice.Init(conf)
		if err != nil {
			return nil, err
		}
		loc := extraStr(ex, "location")
		in := &qsservice.ListBucketsInput{}
		if loc != "" {
			in.Location = aws.String(loc)
		}
		res, err := svc.ListBuckets(in)
		if err != nil {
			return nil, err
		}
		out := make([]osBucketRow, 0, len(res.Buckets))
		for _, b := range res.Buckets {
			if b.Name == nil {
				continue
			}
			tag := ""
			if b.Location != nil {
				tag = *b.Location
			}
			out = append(out, osBucketRow{Name: *b.Name, Tag: tag})
		}
		return out, nil

	case "upyun":
		if err := objectStorageRejectSTSUnsupported(c, "upyun"); err != nil {
			return nil, err
		}
		return upyunListBucketsAPI(ak, sk)

	default:
		return nil, fmt.Errorf("不支持的云类型: %s", c.Provider)
	}
}

type upyunBucketAPIResp struct {
	Buckets []struct {
		BucketName string `json:"bucket_name"`
	} `json:"buckets"`
}

func upyunListBucketsAPI(operator, password string) ([]osBucketRow, error) {
	try := func(pass string) ([]osBucketRow, error) {
		req, err := http.NewRequest(http.MethodGet, "https://api.upyun.com/v2/buckets/?limit=200", nil)
		if err != nil {
			return nil, err
		}
		req.SetBasicAuth(operator, pass)
		resp, err := shortTimeoutHTTPClient().Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("又拍列举服务失败: HTTP %d %s", resp.StatusCode, strings.TrimSpace(string(b)))
		}
		var wr upyunBucketAPIResp
		if err := json.Unmarshal(b, &wr); err != nil {
			return nil, fmt.Errorf("又拍 API 解析失败: %v", err)
		}
		out := make([]osBucketRow, 0, len(wr.Buckets))
		for _, x := range wr.Buckets {
			if x.BucketName != "" {
				out = append(out, osBucketRow{Name: x.BucketName})
			}
		}
		return out, nil
	}
	out, err := try(md5HexLower(password))
	if err == nil && len(out) > 0 {
		return out, nil
	}
	out2, err2 := try(password)
	if err2 == nil {
		return out2, nil
	}
	if err == nil {
		return out, nil
	}
	return nil, err
}

func newUpyunClient(bucket, operator, password string) *upyun.UpYun {
	return upyun.NewUpYun(&upyun.UpYunConfig{
		Bucket:   bucket,
		Operator: operator,
		Password: password,
	})
}

// hierarchical=true 时使用「/」分隔符，仅列当前层级；false 时扁平列出前缀下全部对象键（用于递归删除等）。
func listObjectsForCred(ctx context.Context, c *model.ObjectStorageCredential, bucket, bucketTag, prefix, continuation string, maxKeys int, hierarchical bool) (osListResult, error) {
	var empty osListResult
	p := strings.ToLower(strings.TrimSpace(c.Provider))
	ak, sk := strings.TrimSpace(c.AccessKey), strings.TrimSpace(c.SecretKey)
	if bucket == "" {
		return empty, fmt.Errorf("bucket 不能为空")
	}
	mk := clampMaxKeys(maxKeys)
	ex := parseExtra(c)

	switch p {
	case "qiniu":
		if qiniuUsesS3STS(c) {
			cli, err := newQiniuS3ClientForRegion(ctx, c, bucketTag)
			if err != nil {
				return empty, err
			}
			return s3ListObjectsV2Rows(ctx, cli, bucket, prefix, continuation, mk, hierarchical)
		}
		mac := auth.New(ak, sk)
		bm := storage.NewBucketManager(mac, &storage.Config{})
		delim := "/"
		if !hierarchical {
			delim = ""
		}
		entries, cps, next, hasNext, err := bm.ListFiles(bucket, prefix, delim, continuation, mk)
		if err != nil {
			return empty, err
		}
		res := osListResult{Truncated: hasNext, NextMarker: next, CommonPrefixes: append([]string(nil), cps...)}
		for _, e := range entries {
			res.Objects = append(res.Objects, osObjectRow{
				Key:          e.Key,
				Size:         e.Fsize,
				LastModified: "",
				IsDir:        strings.HasSuffix(e.Key, "/"),
			})
		}
		return res, nil

	case "tencent":
		rg, err := tencentCosBucketRegion(ex, bucketTag)
		if err != nil {
			return empty, err
		}
		u, err := tencentCosVirtualHostedBucketURL(bucket, rg, ex)
		if err != nil {
			return empty, err
		}
		client := cosNewClient(&cos.BaseURL{BucketURL: u}, ak, sk, credSessionToken(c))
		opt := &cos.BucketGetOptions{Prefix: prefix, MaxKeys: mk, Marker: continuation}
		if hierarchical {
			opt.Delimiter = "/"
		}
		v, _, err := client.Bucket.Get(ctx, opt)
		if err != nil {
			return empty, err
		}
		res := osListResult{Truncated: v.IsTruncated, NextMarker: v.NextMarker, CommonPrefixes: append([]string(nil), v.CommonPrefixes...)}
		for _, o := range v.Contents {
			res.Objects = append(res.Objects, osObjectRow{Key: o.Key, Size: o.Size, LastModified: o.LastModified})
		}
		return res, nil

	case "aliyun":
		ep, err := aliyunOssEndpointForObjectAPI(ex, bucketTag)
		if err != nil {
			return empty, err
		}
		client, err := ossNewAliyun(ep, ak, sk, credSessionToken(c))
		if err != nil {
			return empty, err
		}
		bkt, err := client.Bucket(bucket)
		if err != nil {
			return empty, err
		}
		var ls oss.ListObjectsResult
		if hierarchical {
			ls, err = bkt.ListObjects(oss.Prefix(prefix), oss.MaxKeys(mk), oss.Marker(continuation), oss.Delimiter("/"))
		} else {
			ls, err = bkt.ListObjects(oss.Prefix(prefix), oss.MaxKeys(mk), oss.Marker(continuation))
		}
		if err != nil {
			return empty, err
		}
		res := osListResult{Truncated: ls.IsTruncated, NextMarker: ls.NextMarker}
		for _, o := range ls.Objects {
			res.Objects = append(res.Objects, osObjectRow{Key: o.Key, Size: o.Size, LastModified: o.LastModified.Format(time.RFC3339)})
		}
		for _, cp := range ls.CommonPrefixes {
			res.CommonPrefixes = append(res.CommonPrefixes, cp)
		}
		return res, nil

	case "huawei":
		// 使用正式 OBS ListObjects 接口（按桶地域选择正确 endpoint，避免 NoSuchBucket）。
		if ak == "" || sk == "" {
			return empty, fmt.Errorf("华为云 OBS 需填写 AccessKey 与 SecretKey")
		}
		endpoint, err := huaweiObsEndpointForObjectAPI(ex, bucketTag)
		if err != nil {
			return empty, err
		}
		var client *obs.ObsClient
		st := credSessionToken(c)
		if strings.TrimSpace(st) != "" {
			client, err = obs.New(ak, sk, endpoint, obs.WithSecurityToken(st))
		} else {
			client, err = obs.New(ak, sk, endpoint)
		}
		if err != nil {
			return empty, err
		}
		defer client.Close()
		input := &obs.ListObjectsInput{
			Bucket: bucket,
		}
		if prefix != "" {
			input.Prefix = prefix
		}
		input.MaxKeys = mk
		if hierarchical {
			input.Delimiter = "/"
		}
		if continuation != "" {
			input.Marker = continuation
		}
		out, err := client.ListObjects(input)
		if err != nil {
			return empty, err
		}
		res := osListResult{
			Truncated:  out.IsTruncated,
			NextMarker: out.NextMarker,
		}
		for _, cp := range out.CommonPrefixes {
			if cp != "" {
				res.CommonPrefixes = append(res.CommonPrefixes, cp)
			}
		}
		for _, o := range out.Contents {
			lm := ""
			if !o.LastModified.IsZero() {
				lm = o.LastModified.Format(time.RFC3339)
			}
			res.Objects = append(res.Objects, osObjectRow{
				Key:          o.Key,
				Size:         o.Size,
				LastModified: lm,
				IsDir:        strings.HasSuffix(o.Key, "/"),
			})
		}
		return res, nil

	case "aws", "minio", "jd", "ks3":
		endpoint, region, ps := resolveS3Endpoint(p, ex)
		if (p == "minio" || p == "jd" || p == "ks3") && endpoint == "" {
			return empty, fmt.Errorf("请配置 endpoint 或 region")
		}
		cli, err := newS3Client(ctx, ak, sk, region, endpoint, ps, credSessionToken(c))
		if err != nil {
			return empty, err
		}
		return s3ListObjectsV2Rows(ctx, cli, bucket, prefix, continuation, mk, hierarchical)

	case "qingstor":
		if err := objectStorageRejectSTSUnsupported(c, "qingstor"); err != nil {
			return empty, err
		}
		zone := strings.TrimSpace(bucketTag)
		if zone == "" {
			zone = extraStr(ex, "zone")
		}
		if zone == "" {
			return empty, fmt.Errorf("青云需传入 bucket 所在 zone（选桶时的 tag 或 extra.zone）")
		}
		conf, err := qsconfig.New(ak, sk)
		if err != nil {
			return empty, err
		}
		svc, err := qsservice.Init(conf)
		if err != nil {
			return empty, err
		}
		bkt, err := svc.Bucket(bucket, zone)
		if err != nil {
			return empty, err
		}
		limit := mk
		in := &qsservice.ListObjectsInput{
			Prefix: aws.String(prefix),
			Limit:  aws.Int(limit),
		}
		if hierarchical {
			in.Delimiter = aws.String("/")
		}
		if continuation != "" {
			in.Marker = aws.String(continuation)
		}
		out, err := bkt.ListObjects(in)
		if err != nil {
			return empty, err
		}
		res := osListResult{}
		if out.HasMore != nil {
			res.Truncated = *out.HasMore
		}
		if out.NextMarker != nil {
			res.NextMarker = *out.NextMarker
		}
		for _, cp := range out.CommonPrefixes {
			if cp != nil {
				res.CommonPrefixes = append(res.CommonPrefixes, *cp)
			}
		}
		for _, k := range out.Keys {
			if k == nil || k.Key == nil {
				continue
			}
			sz := int64(0)
			if k.Size != nil {
				sz = *k.Size
			}
			lm := ""
			if k.Created != nil {
				lm = k.Created.Format(time.RFC3339)
			}
			res.Objects = append(res.Objects, osObjectRow{Key: *k.Key, Size: sz, LastModified: lm})
		}
		return res, nil

	case "upyun":
		if err := objectStorageRejectSTSUnsupported(c, "upyun"); err != nil {
			return empty, err
		}
		if !hierarchical {
			return empty, fmt.Errorf("又拍云不支持扁平列举 API，请使用专用递归删除")
		}
		up := newUpyunClient(bucket, ak, sk)
		base := normalizeUpyunPath(prefix)
		files, nextIter, err := up.ListObjects(&upyun.ListObjectsConfig{
			Path:  base,
			Iter:  continuation,
			Limit: mk,
		})
		if err != nil {
			return empty, err
		}
		res := osListResult{Truncated: nextIter != "", NextMarker: nextIter}
		for _, f := range files {
			key := strings.TrimSuffix(base, "/") + "/" + f.Name
			if key == "//" {
				key = "/" + f.Name
			}
			if f.IsDir {
				key += "/"
				res.CommonPrefixes = append(res.CommonPrefixes, key)
				continue
			}
			res.Objects = append(res.Objects, osObjectRow{
				Key:          key,
				Size:         f.Size,
				LastModified: f.Time.Format(time.RFC3339),
				IsDir:        false,
			})
		}
		return res, nil

	default:
		return empty, fmt.Errorf("不支持的云类型: %s", c.Provider)
	}
}

const maxRecursiveDeleteObjects = 100000

// deletePrefixRecursiveForCred 删除前缀下全部对象（扁平列举后逐个 Delete），直至分层列举下该前缀为空。
func deletePrefixRecursiveForCred(ctx context.Context, c *model.ObjectStorageCredential, bucket, bucketTag, prefix string) error {
	p := strings.ToLower(strings.TrimSpace(c.Provider))
	pfx := strings.TrimSpace(prefix)
	pfx = strings.TrimPrefix(pfx, "/")
	if pfx != "" && !strings.HasSuffix(pfx, "/") {
		pfx += "/"
	}
	if pfx == "" {
		return fmt.Errorf("禁止递归删除空前缀，请指定具体目录")
	}
	if p == "upyun" {
		return upyunDeletePrefixRecursive(ctx, c, bucket, bucketTag, pfx)
	}

	deleted := 0
	token := ""
	mk := clampMaxKeys(1000)
	for {
		res, err := listObjectsForCred(ctx, c, bucket, bucketTag, pfx, token, mk, false)
		if err != nil {
			return err
		}
		for _, o := range res.Objects {
			k := strings.TrimSpace(o.Key)
			if k == "" {
				continue
			}
			k = strings.TrimPrefix(k, "/")
			if deleted >= maxRecursiveDeleteObjects {
				return fmt.Errorf("递归删除已达上限 %d 个对象，已中止", maxRecursiveDeleteObjects)
			}
			if err := deleteObjectForCred(ctx, c, bucket, bucketTag, k); err != nil {
				return fmt.Errorf("删除 %q 失败: %w", k, err)
			}
			deleted++
		}
		if !res.Truncated {
			break
		}
		token = res.NextMarker
		if token == "" {
			break
		}
	}

	sub, err := listObjectsForCred(ctx, c, bucket, bucketTag, pfx, "", 1, true)
	if err != nil {
		return err
	}
	if len(sub.Objects) > 0 || len(sub.CommonPrefixes) > 0 || sub.Truncated {
		return fmt.Errorf("递归删除后该路径下仍有内容，请刷新后重试")
	}
	return nil
}

// upyunDeletePrefixRecursive 深度优先删除目录下文件，并尝试删除各级目录占位。
func upyunDeletePrefixRecursive(ctx context.Context, c *model.ObjectStorageCredential, bucket, bucketTag, objectPrefix string) error {
	ak, sk := strings.TrimSpace(c.AccessKey), strings.TrimSpace(c.SecretKey)
	up := newUpyunClient(bucket, ak, sk)
	base := normalizeUpyunPath(objectPrefix)
	var walk func(dir string) error
	walk = func(dir string) error {
		iter := ""
		for {
			if err := ctx.Err(); err != nil {
				return err
			}
			files, nextIter, err := up.ListObjects(&upyun.ListObjectsConfig{
				Path:  dir,
				Iter:  iter,
				Limit: 1000,
			})
			if err != nil {
				return err
			}
			for _, f := range files {
				if f.IsDir {
					sub := strings.TrimSuffix(dir, "/") + "/" + f.Name + "/"
					if !strings.HasPrefix(sub, "/") {
						sub = "/" + sub
					}
					if err := walk(sub); err != nil {
						return err
					}
					continue
				}
				full := strings.TrimSuffix(dir, "/") + "/" + f.Name
				key := strings.TrimPrefix(full, "/")
				if err := deleteObjectForCred(ctx, c, bucket, bucketTag, key); err != nil {
					return fmt.Errorf("删除 %q 失败: %w", key, err)
				}
			}
			if nextIter == "" {
				break
			}
			iter = nextIter
		}
		dirKey := strings.TrimPrefix(strings.TrimSuffix(dir, "/"), "/")
		if dirKey != "" {
			_ = deleteObjectForCred(ctx, c, bucket, bucketTag, dirKey+"/")
		}
		return nil
	}
	if err := walk(base); err != nil {
		return err
	}
	sub, err := listObjectsForCred(ctx, c, bucket, bucketTag, objectPrefix, "", 1, true)
	if err != nil {
		return err
	}
	if len(sub.Objects) > 0 || len(sub.CommonPrefixes) > 0 || sub.Truncated {
		return fmt.Errorf("又拍云递归删除后该路径下仍有内容，请刷新后重试")
	}
	return nil
}

func normalizeUpyunPath(prefix string) string {
	p := strings.TrimSpace(prefix)
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if p != "/" && !strings.HasSuffix(p, "/") {
		p += "/"
	}
	return p
}

func uploadForCred(ctx context.Context, c *model.ObjectStorageCredential, bucket, bucketTag, objectKey, localPath string) error {
	p := strings.ToLower(strings.TrimSpace(c.Provider))
	ak, sk := strings.TrimSpace(c.AccessKey), strings.TrimSpace(c.SecretKey)
	if bucket == "" || objectKey == "" {
		return fmt.Errorf("bucket 与对象键不能为空")
	}
	localPath = strings.TrimSpace(localPath)
	if localPath == "" {
		return fmt.Errorf("本地文件路径不能为空")
	}
	if _, err := os.Stat(localPath); err != nil {
		return err
	}
	ex := parseExtra(c)

	switch p {
	case "qiniu":
		if qiniuUsesS3STS(c) {
			cli, err := newQiniuS3ClientForRegion(ctx, c, bucketTag)
			if err != nil {
				return err
			}
			f, err := os.Open(localPath)
			if err != nil {
				return err
			}
			defer f.Close()
			st, err := f.Stat()
			if err != nil {
				return err
			}
			_, err = cli.PutObject(ctx, &s3.PutObjectInput{
				Bucket:        aws.String(bucket),
				Key:           aws.String(objectKey),
				Body:          f,
				ContentLength: aws.Int64(st.Size()),
			})
			return err
		}
		mac := auth.New(ak, sk)
		putPolicy := storage.PutPolicy{Scope: bucket}
		upToken := putPolicy.UploadToken(mac)
		cfg := storage.Config{}
		uploader := storage.NewFormUploader(&cfg)
		ret := storage.PutRet{}
		return uploader.PutFile(ctx, &ret, upToken, objectKey, localPath, nil)

	case "tencent":
		rg, err := tencentCosBucketRegion(ex, bucketTag)
		if err != nil {
			return err
		}
		u, err := tencentCosVirtualHostedBucketURL(bucket, rg, ex)
		if err != nil {
			return err
		}
		client := cosNewClient(&cos.BaseURL{BucketURL: u}, ak, sk, credSessionToken(c))
		_, err = client.Object.PutFromFile(ctx, objectKey, localPath, nil)
		return err

	case "aliyun":
		ep, err := aliyunOssEndpointForObjectAPI(ex, bucketTag)
		if err != nil {
			return err
		}
		client, err := ossNewAliyun(ep, ak, sk, credSessionToken(c))
		if err != nil {
			return err
		}
		bkt, err := client.Bucket(bucket)
		if err != nil {
			return err
		}
		return bkt.PutObjectFromFile(objectKey, localPath)

	case "aws", "minio", "jd", "ks3":
		endpoint, region, ps := resolveS3Endpoint(p, ex)
		if (p == "minio" || p == "jd" || p == "ks3") && endpoint == "" {
			return fmt.Errorf("请配置 endpoint 或 region")
		}
		cli, err := newS3Client(ctx, ak, sk, region, endpoint, ps, credSessionToken(c))
		if err != nil {
			return err
		}
		f, err := os.Open(localPath)
		if err != nil {
			return err
		}
		defer f.Close()
		st, err := f.Stat()
		if err != nil {
			return err
		}
		_, err = cli.PutObject(ctx, &s3.PutObjectInput{
			Bucket:        aws.String(bucket),
			Key:           aws.String(objectKey),
			Body:          f,
			ContentLength: aws.Int64(st.Size()),
		})
		return err

	case "qingstor":
		zone := strings.TrimSpace(bucketTag)
		if zone == "" {
			zone = extraStr(ex, "zone")
		}
		if zone == "" {
			return fmt.Errorf("青云需要 zone")
		}
		conf, err := qsconfig.New(ak, sk)
		if err != nil {
			return err
		}
		svc, err := qsservice.Init(conf)
		if err != nil {
			return err
		}
		bkt, err := svc.Bucket(bucket, zone)
		if err != nil {
			return err
		}
		f, err := os.Open(localPath)
		if err != nil {
			return err
		}
		defer f.Close()
		st, err := f.Stat()
		if err != nil {
			return err
		}
		sz := st.Size()
		_, err = bkt.PutObject(objectKey, &qsservice.PutObjectInput{
			ContentLength: aws.Int64(sz),
			Body:          f,
		})
		return err

	case "upyun":
		up := newUpyunClient(bucket, ak, sk)
		remotePath := objectKey
		if !strings.HasPrefix(remotePath, "/") {
			remotePath = "/" + remotePath
		}
		return up.Put(&upyun.PutObjectConfig{
			Path:      remotePath,
			LocalPath: localPath,
			UseMD5:    false,
		})

	default:
		return fmt.Errorf("不支持的云类型: %s", c.Provider)
	}
}

// SuggestObjectKeyFromPath 用本地文件名作为默认对象键（可含子路径前缀）。
func SuggestObjectKeyFromPath(localPath, prefix string) string {
	base := filepath.Base(localPath)
	p := strings.Trim(prefix, "/")
	if p == "" {
		return base
	}
	return p + "/" + base
}
