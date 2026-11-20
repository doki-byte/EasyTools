package webscan

import (
	"context"

	"EasyTools/app/model"

	"gorm.io/gorm"
)

// PocDatabase POC 数据库服务
type PocDatabase struct {
	ctx context.Context
	db  *gorm.DB
}

// NewPocDatabase 创建 POC 数据库服务
func NewPocDatabase(ctx context.Context, db *gorm.DB) *PocDatabase {
	return &PocDatabase{
		ctx: ctx,
		db:  db,
	}
}

// Initialize 初始化数据库表
func (p *PocDatabase) Initialize() error {
	return p.db.AutoMigrate(&model.VulnerabilityInfo{})
}

// AddPocResult 添加 POC 扫描结果
func (p *PocDatabase) AddPocResult(result model.VulnerabilityInfo) error {
	return p.db.Create(&result).Error
}

// RetrievePocResults 根据任务ID检索 POC 扫描结果
func (p *PocDatabase) RetrievePocResults(taskId string) ([]model.VulnerabilityInfo, error) {
	var results []model.VulnerabilityInfo
	err := p.db.Where("task_id = ?", taskId).Find(&results).Error
	return results, err
}

// RemovePocResult 移除某个 POC 扫描结果
func (p *PocDatabase) RemovePocResult(taskId, templateId, vulnURL string) error {
	return p.db.Where("task_id = ? AND template_id = ? AND vuln_url = ?", taskId, templateId, vulnURL).Delete(&model.VulnerabilityInfo{}).Error
}

// RemovePocResultsByTaskId 根据任务ID删除所有结果
func (p *PocDatabase) RemovePocResultsByTaskId(taskId string) error {
	return p.db.Where("task_id = ?", taskId).Delete(&model.VulnerabilityInfo{}).Error
}

// ClearAllPocResults 删除所有 POC 扫描结果
func (p *PocDatabase) ClearAllPocResults() error {
	// 允许全表删除，清空所有历史漏洞扫描结果
	return p.db.Session(&gorm.Session{AllowGlobalUpdate: true}).
		Delete(&model.VulnerabilityInfo{}).Error
}

// GetAllPocResults 获取所有 POC 扫描结果
func (p *PocDatabase) GetAllPocResults() ([]model.VulnerabilityInfo, error) {
	var results []model.VulnerabilityInfo
	err := p.db.Find(&results).Error
	return results, err
}

// GetPocResultsBySeverity 根据严重程度获取结果
func (p *PocDatabase) GetPocResultsBySeverity(severity string) ([]model.VulnerabilityInfo, error) {
	var results []model.VulnerabilityInfo
	err := p.db.Where("severity = ?", severity).Find(&results).Error
	return results, err
}

// GetPocResultCount 获取结果数量
func (p *PocDatabase) GetPocResultCount(taskId string) (int64, error) {
	var count int64
	err := p.db.Model(&model.VulnerabilityInfo{}).Where("task_id = ?", taskId).Count(&count).Error
	return count, err
}

