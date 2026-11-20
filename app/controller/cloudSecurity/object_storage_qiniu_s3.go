package cloudSecurity

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"EasyTools/app/model"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// 七牛 Kodo 区域码 -> S3 兼容 Region ID（见 https://developer.qiniu.com/kodo/manual/4088/s3-access-domainname ）
var qiniuKodoRegionToS3 = map[string]string{
	"z0":        "cn-east-1",
	"z1":        "cn-north-1",
	"z2":        "cn-south-1",
	"na0":       "us-north-1",
	"as0":       "ap-southeast-1",
	"cn-east-2": "cn-east-2",
}

var qiniuS3HostRe = regexp.MustCompile(`(?i)^https?://s3\.([a-z0-9-]+)\.qiniucs\.com/?`)

// 七牛 S3 兼容公网地域（与官方「服务域名」表一致），用于 STS 下列举桶时多地域探测。
var qiniuS3ProbeRegions = []string{
	"cn-east-1", "cn-east-2", "cn-north-1", "cn-south-1",
	"us-north-1", "ap-southeast-1", "ap-southeast-2", "ap-southeast-3",
}

// qiniuUsesS3STS 七牛临时安全凭证仅能通过官方 S3 兼容接口 + AWS Signature V4 + SessionToken 使用（见临时安全凭证文档）。
func qiniuUsesS3STS(c *model.ObjectStorageCredential) bool {
	return credSessionToken(c) != ""
}

// qiniuS3PinnedRegion 用户是否在扩展 JSON 中固定了 S3 地域（此时只请求该地域，不做全表探测）。
func qiniuS3PinnedRegion(ex map[string]interface{}) (region string, pinned bool) {
	if s := strings.TrimSpace(extraStr(ex, "s3_region")); s != "" {
		return strings.ToLower(s), true
	}
	if raw := strings.TrimSpace(extraStr(ex, "endpoint")); raw != "" {
		raw = strings.TrimRight(raw, "/")
		if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
			raw = "https://" + raw
		}
		if m := qiniuS3HostRe.FindStringSubmatch(raw); len(m) == 2 {
			return m[1], true
		}
	}
	if rg := strings.TrimSpace(extraStr(ex, "region")); rg != "" {
		rl := strings.ToLower(rg)
		if strings.HasPrefix(rl, "cn-") || strings.HasPrefix(rl, "us-") || strings.HasPrefix(rl, "ap-") {
			return rl, true
		}
		if mapped, ok := qiniuKodoRegionToS3[rl]; ok {
			return mapped, true
		}
	}
	return "", false
}

// qiniuSTSListBucketsRows STS 下列桶：未固定地域时依次请求各地域 S3 endpoint，合并结果并为每桶写入 Tag（S3 region），便于后续 ListObject 连对节点。
func qiniuSTSListBucketsRows(ctx context.Context, c *model.ObjectStorageCredential) ([]osBucketRow, error) {
	ak, sk := strings.TrimSpace(c.AccessKey), strings.TrimSpace(c.SecretKey)
	st := credSessionToken(c)
	ex := parseExtra(c)

	if region, pinned := qiniuS3PinnedRegion(ex); pinned {
		ep := "https://s3." + region + ".qiniucs.com"
		cli, err := newS3Client(ctx, ak, sk, region, ep, true, st)
		if err != nil {
			return nil, err
		}
		rows, err := s3ListBucketsRows(ctx, cli)
		if err != nil {
			return nil, err
		}
		for i := range rows {
			rows[i].Tag = region
		}
		return rows, nil
	}

	var lastErr error
	seen := map[string]struct{}{}
	var out []osBucketRow
	for _, rg := range qiniuS3ProbeRegions {
		ep := "https://s3." + rg + ".qiniucs.com"
		cli, err := newS3Client(ctx, ak, sk, rg, ep, true, st)
		if err != nil {
			lastErr = err
			continue
		}
		rows, err := s3ListBucketsRows(ctx, cli)
		if err != nil {
			lastErr = err
			continue
		}
		for _, r := range rows {
			name := strings.TrimSpace(r.Name)
			if name == "" {
				continue
			}
			if _, dup := seen[name]; dup {
				continue
			}
			seen[name] = struct{}{}
			out = append(out, osBucketRow{Name: name, Tag: rg})
		}
	}
	if len(out) == 0 {
		if lastErr != nil {
			return nil, fmt.Errorf("七牛 S3 临时凭证在常用地域列举桶均失败: %w。请核对 STS 策略是否含 s3:ListAllMyBuckets，或在扩展 JSON 指定 s3_region", lastErr)
		}
		return nil, fmt.Errorf("未列举到存储桶：请确认 STS 策略包含 s3:ListAllMyBuckets，或账号下无桶")
	}
	return out, nil
}

// resolveQiniuS3Access 解析 S3 兼容 endpoint 与 region（path-style，与官方示例一致）。
func resolveQiniuS3Access(ex map[string]interface{}) (endpoint, region string, err error) {
	rawEp := strings.TrimSpace(extraStr(ex, "endpoint"))
	if rawEp != "" {
		rawEp = strings.TrimRight(rawEp, "/")
		if !strings.HasPrefix(rawEp, "http://") && !strings.HasPrefix(rawEp, "https://") {
			rawEp = "https://" + rawEp
		}
		u, perr := url.Parse(rawEp)
		if perr != nil || u.Host == "" {
			return "", "", fmt.Errorf("扩展 JSON 中 endpoint 无效")
		}
		endpoint = strings.TrimRight(rawEp, "/")
		if m := qiniuS3HostRe.FindStringSubmatch(endpoint); len(m) == 2 {
			region = m[1]
		} else {
			region = strings.TrimSpace(extraStr(ex, "s3_region"))
			if region == "" {
				region = strings.TrimSpace(extraStr(ex, "region"))
			}
		}
		if region == "" {
			region = "cn-east-1"
		}
		return endpoint, region, nil
	}

	s3r := strings.TrimSpace(extraStr(ex, "s3_region"))
	if s3r != "" {
		return "https://s3." + s3r + ".qiniucs.com", s3r, nil
	}

	rg := strings.TrimSpace(extraStr(ex, "region"))
	if rg != "" {
		rl := strings.ToLower(rg)
		if strings.HasPrefix(rl, "cn-") || strings.HasPrefix(rl, "us-") || strings.HasPrefix(rl, "ap-") {
			return "https://s3." + rl + ".qiniucs.com", rl, nil
		}
		if mapped, ok := qiniuKodoRegionToS3[rl]; ok {
			return "https://s3." + mapped + ".qiniucs.com", mapped, nil
		}
	}

	// 无配置时默认华东，与七牛文档常见默认一致；若桶不在该区域，请显式配置 s3_region
	return "https://s3.cn-east-1.qiniucs.com", "cn-east-1", nil
}

func newQiniuS3Client(ctx context.Context, c *model.ObjectStorageCredential) (*s3.Client, error) {
	ex := parseExtra(c)
	endpoint, region, err := resolveQiniuS3Access(ex)
	if err != nil {
		return nil, err
	}
	ak, sk := strings.TrimSpace(c.AccessKey), strings.TrimSpace(c.SecretKey)
	return newS3Client(ctx, ak, sk, region, endpoint, true, credSessionToken(c))
}

// newQiniuS3ClientForRegion 对象操作必须使用桶所在 S3 地域；s3RegionFromBucket 来自列举桶返回的 Tag（多地域探测时必填，否则可能连错 endpoint）。
func newQiniuS3ClientForRegion(ctx context.Context, c *model.ObjectStorageCredential, s3RegionFromBucket string) (*s3.Client, error) {
	if rg := strings.TrimSpace(s3RegionFromBucket); rg != "" {
		ep := "https://s3." + rg + ".qiniucs.com"
		ak, sk := strings.TrimSpace(c.AccessKey), strings.TrimSpace(c.SecretKey)
		return newS3Client(ctx, ak, sk, rg, ep, true, credSessionToken(c))
	}
	return newQiniuS3Client(ctx, c)
}

// objectStorageRejectSTSUnsupported 青云 QingStor / 又拍云当前接入方式不支持 SessionToken。
func objectStorageRejectSTSUnsupported(c *model.ObjectStorageCredential, provider string) error {
	if credSessionToken(c) == "" {
		return nil
	}
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "qingstor":
		return fmt.Errorf("青云 QingStor 官方 Go SDK 仅支持长期 AccessKey/SecretKey，无 SessionToken；对象存储协议与 RAM/STS 三件套不同。请清空「会话令牌」使用长期密钥，或向青云确认是否提供可配置的 S3+STS 类接入")
	case "upyun":
		return fmt.Errorf("又拍云对象存储为操作员+密码签名模型，官方不提供与 STS 相同的 AccessKey+SecretKey+SessionToken。请清空「会话令牌」，仅填写操作员与密码")
	default:
		return nil
	}
}
