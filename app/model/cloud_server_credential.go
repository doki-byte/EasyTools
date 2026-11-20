package model

import "time"

// TableNameCloudServerCredential 云服务器凭据（ECS 等，明文存 config.db，仅本地使用）。
const TableNameCloudServerCredential = "cloud_server_credential"

// CloudServerCredential 多云主机管理账号。
// ExtraJSON 可选键：regions（[]string，要扫描的地域 ID 列表，空则使用各云默认列表）、
// huawei_project_id、huawei_domain_id、ucloud_project_id、aws_session_token（AWS 临时凭证时）等。
type CloudServerCredential struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Vendor       string    `gorm:"size:32;index;not null" json:"vendor"` // aliyun | huawei | ucloud | tencent | aws
	Name         string    `gorm:"size:128" json:"name"`
	AccessKey    string    `gorm:"size:1024" json:"access_key"`
	SecretKey    string    `gorm:"size:1024" json:"secret_key"`
	ExtraJSON    string    `gorm:"type:text" json:"extra_json"`
	Remark       string    `gorm:"size:512" json:"remark"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (*CloudServerCredential) TableName() string {
	return TableNameCloudServerCredential
}
