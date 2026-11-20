package cloudSecurity

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"EasyTools/app/controller/system"
	"EasyTools/app/model"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var errObjectDownloadCancelled = errors.New("user_cancelled")

// ObjectStorage 多云对象存储管理（凭据明文存 config.db，API 在本地进程调用）。
type ObjectStorage struct {
	system.Base
}

// NewObjectStorage 创建实例并注册到 Wails Bind。
func NewObjectStorage() *ObjectStorage {
	return &ObjectStorage{}
}

func (o *ObjectStorage) credByID(id uint) (*model.ObjectStorageCredential, error) {
	db := o.Db()
	if db == nil {
		return nil, errors.New("数据库连接失败")
	}
	var c model.ObjectStorageCredential
	if err := db.First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// ObjectStorageCredentialsJSON 返回全部凭据 JSON 数组（含密钥，仅本地使用）。
// 说明：该接口仅从本地 config.db 读取对象存储凭据列表，不直接访问各云厂商接口。
func (o *ObjectStorage) ObjectStorageCredentialsJSON() (string, error) {
	db := o.Db()
	if db == nil {
		return "", errors.New("数据库连接失败")
	}
	var rows []model.ObjectStorageCredential
	if err := db.Order("id asc").Find(&rows).Error; err != nil {
		return "", err
	}
	b, err := json.Marshal(rows)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ObjectStorageSaveCredentialJSON 保存或更新凭据；id 为 0 时新增。
func (o *ObjectStorage) ObjectStorageSaveCredentialJSON(jsonStr string) error {
	db := o.Db()
	if db == nil {
		return errors.New("数据库连接失败")
	}
	var row model.ObjectStorageCredential
	if err := json.Unmarshal([]byte(jsonStr), &row); err != nil {
		return err
	}
	row.Provider = strings.TrimSpace(strings.ToLower(row.Provider))
	if row.Provider == "" {
		return fmt.Errorf("provider 不能为空")
	}
	if row.ID == 0 {
		return db.Create(&row).Error
	}
	return db.Model(&model.ObjectStorageCredential{}).Where("id = ?", row.ID).Updates(map[string]interface{}{
		"provider":      row.Provider,
		"name":          row.Name,
		"access_key":    row.AccessKey,
		"secret_key":    row.SecretKey,
		"session_token": row.SessionToken,
		"extra_json":    row.ExtraJSON,
		"remark":        row.Remark,
	}).Error
}

// ObjectStorageDeleteCredential 删除一条凭据。
func (o *ObjectStorage) ObjectStorageDeleteCredential(id uint) error {
	db := o.Db()
	if db == nil {
		return errors.New("数据库连接失败")
	}
	return db.Delete(&model.ObjectStorageCredential{}, id).Error
}

// ObjectStorageProbeCredentialJSON 不写入数据库，仅尝试列举存储桶以校验密钥与配置是否可用。
func (o *ObjectStorage) ObjectStorageProbeCredentialJSON(jsonStr string) error {
	var row model.ObjectStorageCredential
	if err := json.Unmarshal([]byte(jsonStr), &row); err != nil {
		return errors.New("凭据 JSON 格式无效")
	}
	row.Provider = strings.TrimSpace(strings.ToLower(row.Provider))
	if row.Provider == "" {
		return errors.New("请选择云平台")
	}
	if strings.TrimSpace(row.AccessKey) == "" || strings.TrimSpace(row.SecretKey) == "" {
		return errors.New("请填写 AccessKey 与 SecretKey")
	}
	ctx := context.Background()
	if _, err := listBucketsForCred(ctx, &row); err != nil {
		return errors.New("无法连接云存储或密钥无效，请核对 AccessKey、SecretKey 与扩展 JSON（如 region、endpoint）")
	}
	return nil
}

// ObjectStorageListBucketsJSON 按凭据列举存储桶，返回 JSON：[{"name":"","tag":""},...]（青云 tag 为 zone；阿里云 tag 为桶 Location，用于对象接口选 endpoint）。
func (o *ObjectStorage) ObjectStorageListBucketsJSON(credentialID uint) (string, error) {
	c, err := o.credByID(credentialID)
	if err != nil {
		return "", err
	}
	ctx := context.Background()
	rows, err := listBucketsForCred(ctx, c)
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(rows)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ObjectStorageListObjectsJSON 列举对象。continuation 为各云分页游标；bucketTag 青云填 zone；阿里云填 ListBuckets 返回的 tag（桶地域）。
func (o *ObjectStorage) ObjectStorageListObjectsJSON(credentialID uint, bucket, bucketTag, prefix, continuation string, maxKeys int) (string, error) {
	c, err := o.credByID(credentialID)
	if err != nil {
		return "", err
	}
	ctx := context.Background()
	res, err := listObjectsForCred(ctx, c, bucket, bucketTag, prefix, continuation, maxKeys, true)
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(res)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ObjectStoragePutObjectBase64 将 Base64 解码后的内容上传为对象（用于小文件、空目录占位等）。contentType 可空。
func (o *ObjectStorage) ObjectStoragePutObjectBase64(credentialID uint, bucket, bucketTag, objectKey, contentBase64, contentType string) error {
	c, err := o.credByID(credentialID)
	if err != nil {
		return err
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(contentBase64))
	if err != nil {
		return fmt.Errorf("Base64 解码失败: %w", err)
	}
	ctx := context.Background()
	return putObjectBytesForCred(ctx, c, bucket, bucketTag, strings.TrimSpace(objectKey), raw, contentType)
}

// ObjectStorageCreateBucket 创建存储桶。青云需在扩展 JSON 配置 zone，或通过 bucketTag 传入地域；阿里云删除/列举依赖地域时请刷新列表后操作。
func (o *ObjectStorage) ObjectStorageCreateBucket(credentialID uint, bucketName, bucketTag string) error {
	c, err := o.credByID(credentialID)
	if err != nil {
		return err
	}
	ctx := context.Background()
	return createBucketForCred(ctx, c, bucketName, bucketTag)
}

// ObjectStorageDeleteBucket 删除空存储桶（各云要求桶内无对象）。
func (o *ObjectStorage) ObjectStorageDeleteBucket(credentialID uint, bucketName, bucketTag string) error {
	c, err := o.credByID(credentialID)
	if err != nil {
		return err
	}
	ctx := context.Background()
	return deleteBucketForCred(ctx, c, bucketName, bucketTag)
}

// ObjectStorageUploadLocalFile 从本机路径上传至指定桶与对象键。
func (o *ObjectStorage) ObjectStorageUploadLocalFile(credentialID uint, bucket, bucketTag, objectKey, localPath string) error {
	c, err := o.credByID(credentialID)
	if err != nil {
		return err
	}
	ctx := context.Background()
	return uploadForCred(ctx, c, bucket, bucketTag, strings.TrimSpace(objectKey), localPath)
}

// ObjectStoragePickLocalFile 打开系统文件选择对话框，返回所选文件路径（取消则空字符串）。
func (o *ObjectStorage) ObjectStoragePickLocalFile() (string, error) {
	ctx := o.GetCtx()
	if ctx == nil {
		return "", errors.New("上下文未就绪")
	}
	path, err := runtime.OpenFileDialog(ctx, runtime.OpenDialogOptions{
		Title: "选择要上传的文件",
	})
	if err != nil {
		return "", err
	}
	return path, nil
}

// ObjectStorageSuggestObjectKey 根据本地路径与可选前缀生成默认对象键。
func (o *ObjectStorage) ObjectStorageSuggestObjectKey(localPath, prefix string) (string, error) {
	return SuggestObjectKeyFromPath(localPath, prefix), nil
}

// ObjectStorageGetObjectBase64 在本地进程内读取对象内容并返回标准 Base64（供前端预览，不受浏览器 CORS 限制）。单对象不超过约 40MB。
func (o *ObjectStorage) ObjectStorageGetObjectBase64(credentialID uint, bucket, bucketTag, objectKey string) (string, error) {
	c, err := o.credByID(credentialID)
	if err != nil {
		return "", err
	}
	ctx := context.Background()
	raw, err := getObjectBytesForCred(ctx, c, bucket, bucketTag, strings.TrimSpace(objectKey))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(raw), nil
}

// ObjectStorageObjectPresignedURL 生成对象 GET 预签名 URL（供 kkFileView 等拉取，expireSec 默认 3600，范围 60～86400）。
func (o *ObjectStorage) ObjectStorageObjectPresignedURL(credentialID uint, bucket, bucketTag, objectKey string, expireSec int) (string, error) {
	c, err := o.credByID(credentialID)
	if err != nil {
		return "", err
	}
	if expireSec <= 0 {
		expireSec = 3600
	}
	ctx := context.Background()
	return presignedGetURLForCred(ctx, c, bucket, bucketTag, objectKey, clampExpireSec(expireSec))
}

// ObjectStorageDeleteObject 删除单个对象。
// 若键以 / 结尾（目录占位），删除后会列举该前缀下是否仍有对象：S3 兼容接口对「删除不存在的键」也常返回成功，
// 若目录内仍有文件却仅删了占位键，若不校验前端会误显「已删除」而列表仍在。
func (o *ObjectStorage) ObjectStorageDeleteObject(credentialID uint, bucket, bucketTag, objectKey string) error {
	c, err := o.credByID(credentialID)
	if err != nil {
		return err
	}
	ctx := context.Background()
	key := strings.TrimSpace(objectKey)
	if err := deleteObjectForCred(ctx, c, bucket, bucketTag, key); err != nil {
		return err
	}
	if key == "" || !strings.HasSuffix(key, "/") {
		return nil
	}
	sub, err := listObjectsForCred(ctx, c, bucket, bucketTag, key, "", 1, true)
	if err != nil {
		return err
	}
	if len(sub.Objects) > 0 || len(sub.CommonPrefixes) > 0 || sub.Truncated {
		return fmt.Errorf("目录未清空：该路径下仍有对象或子文件夹。云存储删除不存在的键也可能返回成功，请先删除内部全部对象后再移除此目录")
	}
	return nil
}

// ObjectStorageDeletePrefixRecursive 递归删除某前缀下全部对象（不可恢复）；空前缀拒绝以防误删整桶。
func (o *ObjectStorage) ObjectStorageDeletePrefixRecursive(credentialID uint, bucket, bucketTag, prefix string) error {
	c, err := o.credByID(credentialID)
	if err != nil {
		return err
	}
	ctx := context.Background()
	return deletePrefixRecursiveForCred(ctx, c, bucket, bucketTag, prefix)
}

// ObjectStorageRenameObject 同桶内重命名（修改对象键）。
func (o *ObjectStorage) ObjectStorageRenameObject(credentialID uint, bucket, bucketTag, oldKey, newKey string) error {
	c, err := o.credByID(credentialID)
	if err != nil {
		return err
	}
	ctx := context.Background()
	return renameObjectForCred(ctx, c, bucket, bucketTag, oldKey, newKey)
}

// ObjectStorageDownloadObject 弹出保存对话框并将对象下载到选定路径。
func (o *ObjectStorage) ObjectStorageDownloadObject(credentialID uint, bucket, bucketTag, objectKey string) error {
	c, err := o.credByID(credentialID)
	if err != nil {
		return err
	}
	ctxW := o.GetCtx()
	if ctxW == nil {
		return errors.New("上下文未就绪")
	}
	suggest := suggestedFilenameFromObjectKey(objectKey)
	dest, err := runtime.SaveFileDialog(ctxW, runtime.SaveDialogOptions{
		Title:            "保存对象到本地",
		DefaultFilename:  suggest,
		CanCreateDirectories: true,
	})
	if err != nil {
		return err
	}
	if strings.TrimSpace(dest) == "" {
		return errObjectDownloadCancelled
	}
	ctx := context.Background()
	return downloadObjectToPathForCred(ctx, c, bucket, bucketTag, objectKey, dest)
}
