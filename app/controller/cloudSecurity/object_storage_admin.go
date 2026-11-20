package cloudSecurity

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"EasyTools/app/model"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/qiniu/go-sdk/v7/auth"
	"github.com/qiniu/go-sdk/v7/storage"
	"github.com/tencentyun/cos-go-sdk-v5"
	"github.com/upyun/go-sdk/v3/upyun"
	qsconfig "github.com/yunify/qingstor-sdk-go/v3/config"
	qsservice "github.com/yunify/qingstor-sdk-go/v3/service"
)

// putObjectBytesForCred 上传内存中的对象内容（用于空目录占位、小文本等）。
func putObjectBytesForCred(ctx context.Context, c *model.ObjectStorageCredential, bucket, bucketTag, objectKey string, body []byte, contentType string) error {
	p := strings.ToLower(strings.TrimSpace(c.Provider))
	ak, sk := strings.TrimSpace(c.AccessKey), strings.TrimSpace(c.SecretKey)
	key := strings.TrimSpace(objectKey)
	if bucket == "" || key == "" {
		return fmt.Errorf("bucket 与对象键不能为空")
	}
	ex := parseExtra(c)
	ct := strings.TrimSpace(contentType)
	r := bytes.NewReader(body)
	sz := int64(len(body))

	switch p {
	case "qiniu":
		if qiniuUsesS3STS(c) {
			cli, err := newQiniuS3ClientForRegion(ctx, c, bucketTag)
			if err != nil {
				return err
			}
			in := &s3.PutObjectInput{
				Bucket:        aws.String(bucket),
				Key:           aws.String(key),
				Body:          r,
				ContentLength: aws.Int64(sz),
			}
			if ct != "" {
				in.ContentType = aws.String(ct)
			}
			_, err = cli.PutObject(ctx, in)
			return err
		}
		mac := auth.New(ak, sk)
		putPolicy := storage.PutPolicy{Scope: bucket}
		upToken := putPolicy.UploadToken(mac)
		cfg := storage.Config{}
		uploader := storage.NewFormUploader(&cfg)
		ret := storage.PutRet{}
		extra := &storage.PutExtra{}
		if ct != "" {
			extra.MimeType = ct
		}
		return uploader.Put(ctx, &ret, upToken, key, r, sz, extra)

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
		var opt *cos.ObjectPutOptions
		if ct != "" {
			opt = &cos.ObjectPutOptions{
				ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{ContentType: ct},
			}
		}
		_, err = client.Object.Put(ctx, key, r, opt)
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
		var opts []oss.Option
		if ct != "" {
			opts = append(opts, oss.ContentType(ct))
		}
		return bkt.PutObject(key, r, opts...)

	case "aws", "minio", "jd", "ks3":
		endpoint, region, ps := resolveS3Endpoint(p, ex)
		if (p == "minio" || p == "jd" || p == "ks3") && endpoint == "" {
			return fmt.Errorf("请配置 endpoint 或 region")
		}
		cli, err := newS3Client(ctx, ak, sk, region, endpoint, ps, credSessionToken(c))
		if err != nil {
			return err
		}
		in := &s3.PutObjectInput{
			Bucket:        aws.String(bucket),
			Key:           aws.String(key),
			Body:          r,
			ContentLength: aws.Int64(sz),
		}
		if ct != "" {
			in.ContentType = aws.String(ct)
		}
		_, err = cli.PutObject(ctx, in)
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
		putIn := &qsservice.PutObjectInput{
			ContentLength: aws.Int64(sz),
			Body:          io.NopCloser(r),
		}
		if ct != "" {
			putIn.ContentType = aws.String(ct)
		}
		_, err = bkt.PutObject(key, putIn)
		return err

	case "upyun":
		if err := objectStorageRejectSTSUnsupported(c, "upyun"); err != nil {
			return err
		}
		up := newUpyunClient(bucket, ak, sk)
		remotePath := key
		if !strings.HasPrefix(remotePath, "/") {
			remotePath = "/" + remotePath
		}
		cfg := &upyun.PutObjectConfig{
			Path:   remotePath,
			Reader: bytes.NewReader(body),
			UseMD5: false,
		}
		if ct != "" {
			cfg.Headers = map[string]string{"Content-Type": ct}
		}
		return up.Put(cfg)

	default:
		return fmt.Errorf("不支持的云类型: %s", c.Provider)
	}
}

func createBucketForCred(ctx context.Context, c *model.ObjectStorageCredential, bucketName, bucketTag string) error {
	p := strings.ToLower(strings.TrimSpace(c.Provider))
	ak, sk := strings.TrimSpace(c.AccessKey), strings.TrimSpace(c.SecretKey)
	name := strings.TrimSpace(bucketName)
	if name == "" {
		return fmt.Errorf("存储桶名称不能为空")
	}
	ex := parseExtra(c)

	switch p {
	case "qiniu":
		if qiniuUsesS3STS(c) {
			cli, err := newQiniuS3ClientForRegion(ctx, c, bucketTag)
			if err != nil {
				return err
			}
			_, err = cli.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(name)})
			return err
		}
		mac := auth.New(ak, sk)
		bm := storage.NewBucketManager(mac, &storage.Config{})
		rid := storage.RegionID(strings.TrimSpace(extraStr(ex, "region")))
		if rid == "" {
			rid = "z0"
		}
		return bm.CreateBucket(name, rid)

	case "tencent":
		rg, err := tencentCosBucketRegion(ex, bucketTag)
		if err != nil {
			return err
		}
		u, err := tencentCosVirtualHostedBucketURL(name, rg, ex)
		if err != nil {
			return err
		}
		client := cosNewClient(&cos.BaseURL{BucketURL: u}, ak, sk, credSessionToken(c))
		_, err = client.Bucket.Put(ctx, nil)
		return err

	case "aliyun":
		ep, fromEx, err := resolveAliyunOssEndpointFromExtra(ex)
		if err != nil {
			return err
		}
		var client *oss.Client
		if fromEx {
			client, err = ossNewAliyun(ep, ak, sk, credSessionToken(c))
			if err != nil {
				return err
			}
		} else {
			client, err = aliyunOssNewClientByProbing(ctx, c)
			if err != nil {
				return err
			}
		}
		return client.CreateBucket(name)

	case "aws", "minio", "jd", "ks3":
		endpoint, region, ps := resolveS3EndpointForBucketLifecycle(p, ex, bucketTag)
		if (p == "minio" || p == "jd" || p == "ks3") && endpoint == "" {
			return fmt.Errorf("请填写 endpoint 或 region（可在凭据扩展 JSON 或新建桶对话框中填写地域）")
		}
		cli, err := newS3Client(ctx, ak, sk, region, endpoint, ps, credSessionToken(c))
		if err != nil {
			return err
		}
		in := &s3.CreateBucketInput{Bucket: aws.String(name)}
		if region != "" && region != "us-east-1" {
			loc := types.BucketLocationConstraint(region)
			in.CreateBucketConfiguration = &types.CreateBucketConfiguration{
				LocationConstraint: loc,
			}
		}
		_, err = cli.CreateBucket(ctx, in)
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
			return fmt.Errorf("青云创建桶需指定可用区（扩展 JSON 的 zone，或创建时传入地域标签）")
		}
		conf, err := qsconfig.New(ak, sk)
		if err != nil {
			return err
		}
		svc, err := qsservice.Init(conf)
		if err != nil {
			return err
		}
		bkt, err := svc.Bucket(name, zone)
		if err != nil {
			return err
		}
		_, err = bkt.Put()
		return err

	case "upyun":
		if err := objectStorageRejectSTSUnsupported(c, "upyun"); err != nil {
			return err
		}
		return fmt.Errorf("又拍云存储服务请在控制台创建，本工具仅支持列举与对象操作")

	default:
		return fmt.Errorf("不支持的云类型: %s", c.Provider)
	}
}

func deleteBucketForCred(ctx context.Context, c *model.ObjectStorageCredential, bucketName, bucketTag string) error {
	p := strings.ToLower(strings.TrimSpace(c.Provider))
	ak, sk := strings.TrimSpace(c.AccessKey), strings.TrimSpace(c.SecretKey)
	name := strings.TrimSpace(bucketName)
	if name == "" {
		return fmt.Errorf("存储桶名称不能为空")
	}
	ex := parseExtra(c)

	switch p {
	case "qiniu":
		if qiniuUsesS3STS(c) {
			cli, err := newQiniuS3ClientForRegion(ctx, c, bucketTag)
			if err != nil {
				return err
			}
			_, err = cli.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: aws.String(name)})
			return err
		}
		mac := auth.New(ak, sk)
		bm := storage.NewBucketManager(mac, &storage.Config{})
		return bm.DropBucket(name)

	case "tencent":
		rg, err := tencentCosBucketRegion(ex, bucketTag)
		if err != nil {
			return err
		}
		u, err := tencentCosVirtualHostedBucketURL(name, rg, ex)
		if err != nil {
			return err
		}
		client := cosNewClient(&cos.BaseURL{BucketURL: u}, ak, sk, credSessionToken(c))
		_, err = client.Bucket.Delete(ctx)
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
		return client.DeleteBucket(name)

	case "aws", "minio", "jd", "ks3":
		endpoint, region, ps := resolveS3EndpointForBucketLifecycle(p, ex, bucketTag)
		if (p == "minio" || p == "jd" || p == "ks3") && endpoint == "" {
			return fmt.Errorf("请填写 endpoint 或 region（可在凭据扩展 JSON 或操作时的地域参数中填写）")
		}
		cli, err := newS3Client(ctx, ak, sk, region, endpoint, ps, credSessionToken(c))
		if err != nil {
			return err
		}
		_, err = cli.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: aws.String(name)})
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
			return fmt.Errorf("青云删除桶需 zone（选中列表项上的地域标签或扩展 JSON）")
		}
		conf, err := qsconfig.New(ak, sk)
		if err != nil {
			return err
		}
		svc, err := qsservice.Init(conf)
		if err != nil {
			return err
		}
		bkt, err := svc.Bucket(name, zone)
		if err != nil {
			return err
		}
		_, err = bkt.Delete()
		return err

	case "upyun":
		if err := objectStorageRejectSTSUnsupported(c, "upyun"); err != nil {
			return err
		}
		return fmt.Errorf("又拍云存储服务请在控制台删除")

	default:
		return fmt.Errorf("不支持的云类型: %s", c.Provider)
	}
}
