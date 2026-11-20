package cloudSecurity

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"EasyTools/app/model"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	obs "github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
	"github.com/qiniu/go-sdk/v7/auth"
	"github.com/qiniu/go-sdk/v7/storage"
	"github.com/tencentyun/cos-go-sdk-v5"
	"github.com/upyun/go-sdk/v3/upyun"
	qsconfig "github.com/yunify/qingstor-sdk-go/v3/config"
	qsservice "github.com/yunify/qingstor-sdk-go/v3/service"
)

func clampExpireSec(sec int) time.Duration {
	if sec < 60 {
		sec = 60
	}
	if sec > 86400 {
		sec = 86400
	}
	return time.Duration(sec) * time.Second
}

func normalizeQiniuDomain(d string) string {
	d = strings.TrimSpace(d)
	d = strings.TrimRight(d, "/")
	if d == "" {
		return ""
	}
	if !strings.HasPrefix(d, "http://") && !strings.HasPrefix(d, "https://") {
		d = "https://" + d
	}
	return d
}

func upyunRESTPath(key string) string {
	k := strings.TrimSpace(key)
	if k == "" {
		return "/"
	}
	if !strings.HasPrefix(k, "/") {
		k = "/" + k
	}
	return k
}

// presignedGetURLForCred 生成短时 GET 链接，供 kkFileView 拉取对象。
func presignedGetURLForCred(ctx context.Context, c *model.ObjectStorageCredential, bucket, bucketTag, objectKey string, expire time.Duration) (string, error) {
	p := strings.ToLower(strings.TrimSpace(c.Provider))
	ak, sk := strings.TrimSpace(c.AccessKey), strings.TrimSpace(c.SecretKey)
	key := strings.TrimSpace(objectKey)
	if key == "" {
		return "", fmt.Errorf("对象键不能为空")
	}
	ex := parseExtra(c)

	switch p {
	case "tencent":
		rg, err := tencentCosBucketRegion(ex, bucketTag)
		if err != nil {
			return "", err
		}
		u, err := tencentCosVirtualHostedBucketURL(bucket, rg, ex)
		if err != nil {
			return "", err
		}
		client := cosNewClient(&cos.BaseURL{BucketURL: u}, ak, sk, credSessionToken(c))
		pu, err := client.Object.GetPresignedURL(ctx, http.MethodGet, key, ak, sk, expire, cosPresignOpt(credSessionToken(c)))
		if err != nil {
			return "", err
		}
		return pu.String(), nil

	case "aliyun":
		ep, err := aliyunOssEndpointForObjectAPI(ex, bucketTag)
		if err != nil {
			return "", err
		}
		client, err := ossNewAliyun(ep, ak, sk, credSessionToken(c))
		if err != nil {
			return "", err
		}
		bkt, err := client.Bucket(bucket)
		if err != nil {
			return "", err
		}
		return bkt.SignURL(key, oss.HTTPGet, int64(expire.Seconds()))

	case "aws", "minio", "jd", "ks3":
		endpoint, region, ps := resolveS3Endpoint(p, ex)
		if (p == "minio" || p == "jd" || p == "ks3") && endpoint == "" {
			return "", fmt.Errorf("请配置 endpoint 或 region")
		}
		cli, err := newS3Client(ctx, ak, sk, region, endpoint, ps, credSessionToken(c))
		if err != nil {
			return "", err
		}
		pc := s3.NewPresignClient(cli)
		out, err := pc.PresignGetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		}, s3.WithPresignExpires(expire))
		if err != nil {
			return "", err
		}
		return out.URL, nil

	case "qiniu":
		if qiniuUsesS3STS(c) {
			cli, err := newQiniuS3ClientForRegion(ctx, c, bucketTag)
			if err != nil {
				return "", err
			}
			pc := s3.NewPresignClient(cli)
			out, err := pc.PresignGetObject(ctx, &s3.GetObjectInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(key),
			}, s3.WithPresignExpires(expire))
			if err != nil {
				return "", err
			}
			return out.URL, nil
		}
		domain := normalizeQiniuDomain(extraStr(ex, "domain"))
		if domain == "" {
			return "", fmt.Errorf("七牛 kkfile 预览需在扩展 JSON 中配置 domain（绑定域名，如 https://xxx.clouddn.com）")
		}
		mac := auth.New(ak, sk)
		deadline := time.Now().Add(expire).Unix()
		return storage.MakePrivateURL(mac, domain, key, deadline), nil

	case "qingstor":
		if err := objectStorageRejectSTSUnsupported(c, "qingstor"); err != nil {
			return "", err
		}
		return "", fmt.Errorf("%s 暂不支持生成 kkfile 外链，请使用「下载」后本地预览", p)
	case "upyun":
		if err := objectStorageRejectSTSUnsupported(c, "upyun"); err != nil {
			return "", err
		}
		return "", fmt.Errorf("%s 暂不支持生成 kkfile 外链，请使用「下载」后本地预览", p)

	case "huawei":
		ep, err := huaweiObsEndpointForObjectAPI(ex, bucketTag)
		if err != nil {
			return "", err
		}
		st := credSessionToken(c)
		var client *obs.ObsClient
		if strings.TrimSpace(st) != "" {
			client, err = obs.New(ak, sk, ep, obs.WithSecurityToken(st))
		} else {
			client, err = obs.New(ak, sk, ep)
		}
		if err != nil {
			return "", err
		}
		defer client.Close()
		in := &obs.CreateSignedUrlInput{}
		in.Method = http.MethodGet
		in.Bucket = bucket
		in.Key = key
		in.Expires = int(expire.Seconds())
		out, err := client.CreateSignedUrl(in)
		if err != nil {
			return "", err
		}
		return out.SignedUrl, nil

	default:
		return "", fmt.Errorf("不支持的云类型: %s", c.Provider)
	}
}

func deleteObjectForCred(ctx context.Context, c *model.ObjectStorageCredential, bucket, bucketTag, objectKey string) error {
	p := strings.ToLower(strings.TrimSpace(c.Provider))
	ak, sk := strings.TrimSpace(c.AccessKey), strings.TrimSpace(c.SecretKey)
	key := strings.TrimSpace(objectKey)
	if key == "" {
		return fmt.Errorf("对象键不能为空")
	}
	ex := parseExtra(c)

	switch p {
	case "qiniu":
		if qiniuUsesS3STS(c) {
			cli, err := newQiniuS3ClientForRegion(ctx, c, bucketTag)
			if err != nil {
				return err
			}
			_, err = cli.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(key),
			})
			return err
		}
		mac := auth.New(ak, sk)
		bm := storage.NewBucketManager(mac, &storage.Config{})
		return bm.Delete(bucket, key)

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
		_, err = client.Object.Delete(ctx, key, nil)
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
		return bkt.DeleteObject(key)

	case "aws", "minio", "jd", "ks3":
		endpoint, region, ps := resolveS3Endpoint(p, ex)
		if (p == "minio" || p == "jd" || p == "ks3") && endpoint == "" {
			return fmt.Errorf("请配置 endpoint 或 region")
		}
		cli, err := newS3Client(ctx, ak, sk, region, endpoint, ps, credSessionToken(c))
		if err != nil {
			return err
		}
		_, err = cli.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		return err

	case "qingstor":
		if err := objectStorageRejectSTSUnsupported(c, "qingstor"); err != nil {
			return err
		}
		zone := strings.TrimSpace(bucketTag)
		if zone == "" {
			zone = extraStr(ex, "zone")
		}
		if zone == "" {
			return fmt.Errorf("青云需 zone")
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
		_, err = bkt.DeleteObject(key)
		return err

	case "upyun":
		if err := objectStorageRejectSTSUnsupported(c, "upyun"); err != nil {
			return err
		}
		up := newUpyunClient(bucket, ak, sk)
		return up.Delete(&upyun.DeleteObjectConfig{Path: upyunRESTPath(key)})

	case "huawei":
		ep, err := huaweiObsEndpointForObjectAPI(ex, bucketTag)
		if err != nil {
			return err
		}
		st := credSessionToken(c)
		var client *obs.ObsClient
		if strings.TrimSpace(st) != "" {
			client, err = obs.New(ak, sk, ep, obs.WithSecurityToken(st))
		} else {
			client, err = obs.New(ak, sk, ep)
		}
		if err != nil {
			return err
		}
		defer client.Close()
		in := &obs.DeleteObjectInput{}
		in.Bucket = bucket
		in.Key = key
		_, err = client.DeleteObject(in)
		return err

	default:
		return fmt.Errorf("不支持的云类型: %s", c.Provider)
	}
}

func renameObjectForCred(ctx context.Context, c *model.ObjectStorageCredential, bucket, bucketTag, oldKey, newKey string) error {
	oldKey = strings.TrimSpace(oldKey)
	newKey = strings.TrimSpace(newKey)
	if oldKey == "" || newKey == "" {
		return fmt.Errorf("对象键不能为空")
	}
	if oldKey == newKey {
		return nil
	}
	p := strings.ToLower(strings.TrimSpace(c.Provider))
	ak, sk := strings.TrimSpace(c.AccessKey), strings.TrimSpace(c.SecretKey)
	ex := parseExtra(c)

	switch p {
	case "qiniu":
		if qiniuUsesS3STS(c) {
			cli, err := newQiniuS3ClientForRegion(ctx, c, bucketTag)
			if err != nil {
				return err
			}
			src := fmt.Sprintf("%s/%s", bucket, oldKey)
			_, err = cli.CopyObject(ctx, &s3.CopyObjectInput{
				Bucket:     aws.String(bucket),
				CopySource: aws.String(src),
				Key:        aws.String(newKey),
			})
			if err != nil {
				return err
			}
			_, err = cli.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(oldKey),
			})
			return err
		}
		mac := auth.New(ak, sk)
		bm := storage.NewBucketManager(mac, &storage.Config{})
		return bm.Move(bucket, oldKey, bucket, newKey, true)

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
		src := fmt.Sprintf("%s/%s", bucket, oldKey)
		_, _, err = client.Object.Copy(ctx, newKey, src, nil)
		if err != nil {
			return err
		}
		_, err = client.Object.Delete(ctx, oldKey, nil)
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
		_, err = bkt.CopyObject(oldKey, newKey)
		if err != nil {
			return err
		}
		return bkt.DeleteObject(oldKey)

	case "aws", "minio", "jd", "ks3":
		endpoint, region, ps := resolveS3Endpoint(p, ex)
		if (p == "minio" || p == "jd" || p == "ks3") && endpoint == "" {
			return fmt.Errorf("请配置 endpoint 或 region")
		}
		cli, err := newS3Client(ctx, ak, sk, region, endpoint, ps, credSessionToken(c))
		if err != nil {
			return err
		}
		_, err = cli.CopyObject(ctx, &s3.CopyObjectInput{
			Bucket:     aws.String(bucket),
			CopySource: aws.String(fmt.Sprintf("%s/%s", bucket, oldKey)),
			Key:        aws.String(newKey),
		})
		if err != nil {
			return err
		}
		_, err = cli.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(oldKey),
		})
		return err

	case "qingstor":
		if err := objectStorageRejectSTSUnsupported(c, "qingstor"); err != nil {
			return err
		}
		zone := strings.TrimSpace(bucketTag)
		if zone == "" {
			zone = extraStr(ex, "zone")
		}
		if zone == "" {
			return fmt.Errorf("青云需 zone")
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
		src := fmt.Sprintf("/%s/%s", bucket, oldKey)
		_, err = bkt.PutObject(newKey, &qsservice.PutObjectInput{
			XQSCopySource: aws.String(src),
		})
		if err != nil {
			return err
		}
		_, err = bkt.DeleteObject(oldKey)
		return err

	case "upyun":
		if err := objectStorageRejectSTSUnsupported(c, "upyun"); err != nil {
			return err
		}
		up := newUpyunClient(bucket, ak, sk)
		return up.Move(&upyun.MoveObjectConfig{
			SrcPath:  upyunRESTPath(oldKey),
			DestPath: upyunRESTPath(newKey),
		})

	default:
		return fmt.Errorf("不支持的云类型: %s", c.Provider)
	}
}

// maxObjectPreviewBytes 编辑器内预览/编辑单次拉取上限（与前端大文件提示配合，避免进程 OOM）。
const maxObjectPreviewBytes = 10 << 20

func readAllLimitedClose(r io.ReadCloser, limit int64) ([]byte, error) {
	defer r.Close()
	lr := io.LimitReader(r, limit+1)
	b, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("对象内容超过 %d MB，无法在编辑器内打开，请使用下载到本地", limit/(1<<20))
	}
	return b, nil
}

// getObjectBytesForCred 在进程内读取对象正文（不经浏览器，不受存储桶 CORS 影响）。
func getObjectBytesForCred(ctx context.Context, c *model.ObjectStorageCredential, bucket, bucketTag, objectKey string) ([]byte, error) {
	key := strings.TrimSpace(objectKey)
	if key == "" {
		return nil, fmt.Errorf("对象键不能为空")
	}
	p := strings.ToLower(strings.TrimSpace(c.Provider))
	ak, sk := strings.TrimSpace(c.AccessKey), strings.TrimSpace(c.SecretKey)
	ex := parseExtra(c)

	switch p {
	case "aliyun":
		ep, err := aliyunOssEndpointForObjectAPI(ex, bucketTag)
		if err != nil {
			return nil, err
		}
		client, err := ossNewAliyun(ep, ak, sk, credSessionToken(c))
		if err != nil {
			return nil, err
		}
		bkt, err := client.Bucket(bucket)
		if err != nil {
			return nil, err
		}
		rc, err := bkt.GetObject(key)
		if err != nil {
			return nil, err
		}
		return readAllLimitedClose(rc, maxObjectPreviewBytes)

	case "upyun":
		if err := objectStorageRejectSTSUnsupported(c, "upyun"); err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		up := newUpyunClient(bucket, ak, sk)
		_, err := up.Get(&upyun.GetObjectConfig{
			Path:   upyunRESTPath(key),
			Writer: &buf,
		})
		if err != nil {
			return nil, err
		}
		if int64(buf.Len()) > maxObjectPreviewBytes {
			return nil, fmt.Errorf("对象内容超过 %d MB，无法在编辑器内打开，请使用下载到本地", maxObjectPreviewBytes/(1<<20))
		}
		return buf.Bytes(), nil

	case "qiniu":
		if qiniuUsesS3STS(c) {
			cli, err := newQiniuS3ClientForRegion(ctx, c, bucketTag)
			if err != nil {
				return nil, err
			}
			out, err := cli.GetObject(ctx, &s3.GetObjectInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(key),
			})
			if err != nil {
				return nil, err
			}
			return readAllLimitedClose(out.Body, maxObjectPreviewBytes)
		}
		mac := auth.New(ak, sk)
		bm := storage.NewBucketManager(mac, &storage.Config{})
		out, err := bm.Get(bucket, key, &storage.GetObjectInput{Context: ctx})
		if err != nil {
			return nil, err
		}
		return readAllLimitedClose(out, maxObjectPreviewBytes)

	case "tencent":
		rg, err := tencentCosBucketRegion(ex, bucketTag)
		if err != nil {
			return nil, err
		}
		u, err := tencentCosVirtualHostedBucketURL(bucket, rg, ex)
		if err != nil {
			return nil, err
		}
		client := cosNewClient(&cos.BaseURL{BucketURL: u}, ak, sk, credSessionToken(c))
		resp, err := client.Object.Get(ctx, key, nil)
		if err != nil {
			return nil, err
		}
		return readAllLimitedClose(resp.Body, maxObjectPreviewBytes)

	case "aws", "minio", "jd", "ks3":
		endpoint, region, ps := resolveS3Endpoint(p, ex)
		if (p == "minio" || p == "jd" || p == "ks3") && endpoint == "" {
			return nil, fmt.Errorf("请配置 endpoint 或 region")
		}
		cli, err := newS3Client(ctx, ak, sk, region, endpoint, ps, credSessionToken(c))
		if err != nil {
			return nil, err
		}
		out, err := cli.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			return nil, err
		}
		return readAllLimitedClose(out.Body, maxObjectPreviewBytes)

	case "qingstor":
		if err := objectStorageRejectSTSUnsupported(c, "qingstor"); err != nil {
			return nil, err
		}
		zone := strings.TrimSpace(bucketTag)
		if zone == "" {
			zone = extraStr(ex, "zone")
		}
		if zone == "" {
			return nil, fmt.Errorf("青云需 zone")
		}
		conf, err := qsconfig.New(ak, sk)
		if err != nil {
			return nil, err
		}
		svc, err := qsservice.Init(conf)
		if err != nil {
			return nil, err
		}
		bkt, err := svc.Bucket(bucket, zone)
		if err != nil {
			return nil, err
		}
		gout, err := bkt.GetObject(key, nil)
		if err != nil {
			return nil, err
		}
		if gout.Body == nil {
			return []byte{}, nil
		}
		return readAllLimitedClose(gout.Body, maxObjectPreviewBytes)

	case "huawei":
		ep, err := huaweiObsEndpointForObjectAPI(ex, bucketTag)
		if err != nil {
			return nil, err
		}
		st := credSessionToken(c)
		var client *obs.ObsClient
		if strings.TrimSpace(st) != "" {
			client, err = obs.New(ak, sk, ep, obs.WithSecurityToken(st))
		} else {
			client, err = obs.New(ak, sk, ep)
		}
		if err != nil {
			return nil, err
		}
		defer client.Close()
		in := &obs.GetObjectInput{}
		in.Bucket = bucket
		in.Key = key
		out, err := client.GetObject(in)
		if err != nil {
			return nil, err
		}
		return readAllLimitedClose(out.Body, maxObjectPreviewBytes)

	default:
		return nil, fmt.Errorf("不支持的云类型: %s", c.Provider)
	}
}

// downloadObjectToPathForCred 将对象写入本地路径（已含 Save 对话框选定的路径）。
func downloadObjectToPathForCred(ctx context.Context, c *model.ObjectStorageCredential, bucket, bucketTag, objectKey, destPath string) error {
	key := strings.TrimSpace(objectKey)
	if key == "" {
		return fmt.Errorf("对象键不能为空")
	}
	destPath = strings.TrimSpace(destPath)
	if destPath == "" {
		return fmt.Errorf("保存路径不能为空")
	}
	p := strings.ToLower(strings.TrimSpace(c.Provider))
	ak, sk := strings.TrimSpace(c.AccessKey), strings.TrimSpace(c.SecretKey)
	ex := parseExtra(c)

	switch p {
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
		return bkt.GetObjectToFile(key, destPath)

	case "upyun":
		if err := objectStorageRejectSTSUnsupported(c, "upyun"); err != nil {
			return err
		}
		up := newUpyunClient(bucket, ak, sk)
		_, err := up.Get(&upyun.GetObjectConfig{
			Path:      upyunRESTPath(key),
			LocalPath: destPath,
		})
		return err
	case "huawei":
		ep, err := huaweiObsEndpointForObjectAPI(ex, bucketTag)
		if err != nil {
			return err
		}
		st := credSessionToken(c)
		var client *obs.ObsClient
		if strings.TrimSpace(st) != "" {
			client, err = obs.New(ak, sk, ep, obs.WithSecurityToken(st))
		} else {
			client, err = obs.New(ak, sk, ep)
		}
		if err != nil {
			return err
		}
		defer client.Close()
		in := &obs.GetObjectInput{}
		in.Bucket = bucket
		in.Key = key
		out, err := client.GetObject(in)
		if err != nil {
			return err
		}
		defer out.Body.Close()
		f, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(f, out.Body)
		return err
	}

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	switch p {
	case "qiniu":
		if qiniuUsesS3STS(c) {
			cli, err := newQiniuS3ClientForRegion(ctx, c, bucketTag)
			if err != nil {
				return err
			}
			out, err := cli.GetObject(ctx, &s3.GetObjectInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(key),
			})
			if err != nil {
				return err
			}
			defer out.Body.Close()
			_, err = io.Copy(f, out.Body)
			return err
		}
		mac := auth.New(ak, sk)
		bm := storage.NewBucketManager(mac, &storage.Config{})
		out, err := bm.Get(bucket, key, &storage.GetObjectInput{Context: ctx})
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(f, out)
		return err

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
		resp, err := client.Object.Get(ctx, key, nil)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		_, err = io.Copy(f, resp.Body)
		return err

	case "aws", "minio", "jd", "ks3":
		endpoint, region, ps := resolveS3Endpoint(p, ex)
		if (p == "minio" || p == "jd" || p == "ks3") && endpoint == "" {
			return fmt.Errorf("请配置 endpoint 或 region")
		}
		cli, err := newS3Client(ctx, ak, sk, region, endpoint, ps, credSessionToken(c))
		if err != nil {
			return err
		}
		out, err := cli.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			return err
		}
		defer out.Body.Close()
		_, err = io.Copy(f, out.Body)
		return err

	case "qingstor":
		zone := strings.TrimSpace(bucketTag)
		if zone == "" {
			zone = extraStr(ex, "zone")
		}
		if zone == "" {
			return fmt.Errorf("青云需 zone")
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
		gout, err := bkt.GetObject(key, nil)
		if err != nil {
			return err
		}
		if gout.Body != nil {
			defer gout.Body.Close()
			_, err = io.Copy(f, gout.Body)
		}
		return err

	default:
		return fmt.Errorf("不支持的云类型: %s", c.Provider)
	}
}

// suggestedFilenameFromObjectKey 用于保存对话框默认文件名。
func suggestedFilenameFromObjectKey(objectKey string) string {
	base := filepath.Base(strings.TrimSuffix(strings.TrimSpace(objectKey), "/"))
	if base == "" || base == "." {
		return "download"
	}
	return base
}
