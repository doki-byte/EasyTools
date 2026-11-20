package model

import "time"

// TableNameObjectStorageCredential 云存储凭据表（明文存于 config.db，仅本地使用）。
const TableNameObjectStorageCredential = "object_storage_credential"

// ObjectStorageCredential 多云对象存储账号配置。
// ExtraJSON 为 JSON 字符串，常用键：region、endpoint、path_style(bool)、bucket（又拍默认服务名）等。
// 阿里云 OSS：可不填，将依次尝试常用地域列出桶；对象操作依赖桶的 Location（刷新列表后点选桶）。也可填 region 或 endpoint（内网/专有云等）。
// SessionToken：临时凭证（STS）会话令牌；阿里云 SecurityToken、腾讯云 x-cos-security-token、AWS/七牛 S3 兼容 x-amz-security-token 等均写入此字段。七牛走原生 SDK 时不要填；走 S3+STS 时需在 ExtraJSON 配置 s3_region（如 cn-east-1）或 endpoint（https://s3.<region>.qiniucs.com）。青云/又拍对象存储当前接入不支持 SessionToken。
type ObjectStorageCredential struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Provider     string    `gorm:"size:32;index;not null" json:"provider"`
	Name         string    `gorm:"size:128" json:"name"`
	AccessKey    string    `gorm:"size:1024" json:"access_key"`
	SecretKey    string    `gorm:"size:1024" json:"secret_key"`
	SessionToken string    `gorm:"size:4096" json:"session_token"`
	ExtraJSON    string    `gorm:"type:text" json:"extra_json"`
	Remark     string    `gorm:"size:512" json:"remark"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (*ObjectStorageCredential) TableName() string {
	return TableNameObjectStorageCredential
}
