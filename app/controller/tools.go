package controller

import (
	"EasyTools/app/controller/system"
	"EasyTools/app/model"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

// Tool 系统API
type Tool struct {
	system.Base
}

// ToolsItem 工具结构体
type ToolsItem struct {
	ID       int    `json:"id"`
	Category string `json:"category"`
	Cmd      string `json:"cmd"`
	Param    string `json:"param"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	Desc     string `json:"desc"`
	Icon     string `json:"icon,omitempty"` // 可选字段
	Terminal int    `json:"terminal"`       // 新增字段，1: 需要终端, 0: 不需要
}

// Category 结构体，用于前端返回
type ToolsCategory struct {
	Title string      `json:"title"`
	List  []ToolsItem `json:"list"`
}

func NewTool() *Tool {
	return &Tool{}
}

// TableName 指定表名
func (ToolsItem) TableName() string {
	return "tools"
}

// 获取分类
func (s *Tool) GetCategoryList() ([]map[string]interface{}, error) {
	db := s.Db()
	if db == nil {
		return nil, fmt.Errorf("数据库连接未初始化")
	}

	// 查询分类及最大CateSort，按降序排列
	var categories []struct {
		Category string `gorm:"column:category"`
		CateSort int    `gorm:"column:cate_sort"`
	}

	err := db.Model(&ToolsItem{}).
		Select("category, MAX(cate_sort) as cate_sort").
		Group("category").
		Order("cate_sort DESC").
		Find(&categories).Error

	if err != nil {
		return nil, fmt.Errorf("获取分类失败: %v", err)
	}

	if len(categories) == 0 {
		return nil, fmt.Errorf("没有分类数据")
	}

	// 构造结果
	var result []map[string]interface{}
	for _, c := range categories {
		result = append(result, map[string]interface{}{
			"title": c.Category,
		})
	}

	return result, nil
}

// 获取所有工具，按分类返回
func (s *Tool) GetAllTools() ([]ToolsCategory, error) {
	var toolsCategories []ToolsCategory

	db := s.Db()
	if db == nil {
		return nil, fmt.Errorf("数据库连接未初始化")
	}

	// 获取分类及排序（与GetCategoryList逻辑保持一致）
	var categories []struct {
		Category string `gorm:"column:category"`
		CateSort int    `gorm:"column:cate_sort"`
	}

	err := db.Model(&ToolsItem{}).
		Select("category, MAX(cate_sort) as cate_sort").
		Group("category").
		Order("cate_sort DESC").
		Find(&categories).Error

	if err != nil {
		return nil, fmt.Errorf("获取分类失败: %v", err)
	}

	if len(categories) == 0 {
		return nil, fmt.Errorf("没有分类数据")
	}

	// 遍历分类获取工具列表
	for _, c := range categories {
		var tools []ToolsItem
		err := db.Where("category = ?", c.Category).
			Order("cmd_sort DESC"). // 添加工具排序
			Find(&tools).Error

		if err != nil {
			continue
		}

		if len(tools) > 0 {
			toolsCategories = append(toolsCategories, ToolsCategory{
				Title: c.Category,
				List:  tools,
			})
		}
	}

	if len(toolsCategories) == 0 {
		return nil, fmt.Errorf("查询完成，但没有数据")
	}

	return toolsCategories, nil
}

// 获取搜索内容，按名称进行模糊搜索并按分类返回
func (s *Tool) GetSearchTools(name string) ([]ToolsCategory, error) {
	var categories []ToolsCategory

	// 获取数据库连接
	db := s.Db()
	if db == nil {
		// s.log("数据库连接未初始化")
		return nil, fmt.Errorf("数据库连接未初始化")
	}

	// 获取所有不同的分类
	var categoryNames []string
	err := db.Model(&ToolsItem{}).
		Where("name LIKE ?", "%"+name+"%").
		Distinct("category").
		Pluck("category", &categoryNames).
		Error
	if err != nil {
		// s.log(fmt.Sprintf("获取分类失败: %v", err))
		return nil, fmt.Errorf("获取分类失败: %v", err)
	}

	if len(categoryNames) == 0 {
		// s.log("没有分类数据")
		return nil, fmt.Errorf("没有分类数据")
	}

	// 根据每个分类，查询该分类下的工具（根据name字段模糊查询）
	for _, categoryName := range categoryNames {
		var tools []ToolsItem
		err := db.Where("category = ? AND name LIKE ?", categoryName, "%"+name+"%").Find(&tools).Error
		if err != nil {
			// s.log(fmt.Sprintf("查询分类 '%s' 的工具失败: %v", categoryName, err))
			continue
		}

		s.Log(fmt.Sprintf("分类 '%s' 查询到 %d 个工具", categoryName, len(tools)))

		// 添加到返回结构
		categories = append(categories, ToolsCategory{
			Title: categoryName,
			List:  tools,
		})
	}

	if len(categories) == 0 {
		// s.log("查询完成，但没有数据")
		return nil, fmt.Errorf("查询完成，但没有数据")
	}

	// s.log(fmt.Sprintf("查询完成，共 %d 个分类，返回数据成功", len(categories)))
	return categories, nil
}

// 新增工具
func (s *Tool) AddTool(tool ToolsItem) (int, error) {
	// 获取数据库连接
	db := s.Db()
	if db == nil {
		// s.log("数据库连接未初始化")
		return 0, fmt.Errorf("数据库连接未初始化")
	}

	// 插入新工具
	err := db.Create(&tool).Error
	if err != nil {
		// s.log(fmt.Sprintf("新增工具失败: %v", err))
		return 0, fmt.Errorf("新增工具失败: %v", err)
	}

	// 返回插入的工具ID
	return tool.ID, nil
}

// 修改工具
func (s *Tool) UpdateTool(id int, updatedTool ToolsItem) error {
	// 获取数据库连接
	db := s.Db()
	if db == nil {
		// s.log("数据库连接未初始化")
		return fmt.Errorf("数据库连接未初始化")
	}

	// 根据工具的id更新工具
	err := db.Model(&ToolsItem{}).Where("id = ?", id).Save(updatedTool).Error
	if err != nil {
		// s.log(fmt.Sprintf("修改工具失败: %v", err))
		return fmt.Errorf("修改工具失败: %v", err)
	}

	return nil
}

// 删除工具
func (s *Tool) DeleteTool(id int) error {
	// 获取数据库连接
	db := s.Db()
	if db == nil {
		// s.log("数据库连接未初始化")
		return fmt.Errorf("数据库连接未初始化")
	}

	// 根据工具ID删除工具
	err := db.Where("id = ?", id).Delete(&ToolsItem{}).Error
	if err != nil {
		// s.log(fmt.Sprintf("删除工具失败: %v", err))
		return fmt.Errorf("删除工具失败: %v", err)
	}

	return nil
}

// 修改分类名称（将所有属于 oldCategory 的工具改为 newCategory）
func (s *Tool) UpdateToolCategory(oldCategory, newCategory string) error {
	db := s.Db()
	if db == nil {
		return fmt.Errorf("数据库连接未初始化")
	}

	err := db.Model(&model.Tools{}).Where("category = ?", oldCategory).Update("category", newCategory).Error
	if err != nil {
		return fmt.Errorf("修改分类失败: %v", err)
	}

	return nil
}

// 删除分类及其下所有工具
func (s *Tool) DeleteToolCategory(category string) error {
	db := s.Db()
	if db == nil {
		return fmt.Errorf("数据库连接未初始化")
	}

	err := db.Where("category = ?", category).Delete(&model.Tools{}).Error
	if err != nil {
		return fmt.Errorf("删除分类失败: %v", err)
	}

	return nil
}

// 更新分类排序
func (s *Tool) UpdateCategorySorts(sorts []map[string]interface{}) error {
	//fmt.Printf("调试信息 - 收到排序请求: %+v\n", sorts) // 添加详细日志

	db := s.Db()
	if db == nil {
		return fmt.Errorf("数据库连接未初始化")
	}

	// 添加recover防止panic
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("发生panic: %v\n", r)
		}
	}()

	tx := db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("事务启动失败: %v", tx.Error)
	}

	for _, sort := range sorts {
		// 安全获取category
		category, ok := sort["category"].(string)
		if !ok || category == "" {
			tx.Rollback()
			return fmt.Errorf("无效的分类名称，接收值: %#v", sort["category"])
		}

		// 安全处理数值类型
		var cateSort int
		switch v := sort["cateSort"].(type) {
		case float64:
			cateSort = int(v)
		case int:
			cateSort = v
		default:
			tx.Rollback()
			return fmt.Errorf("无效的排序值类型，期待数值类型，收到: %T (%v)", v, v)
		}

		//fmt.Printf("正在更新分类: %s, 排序值: %d\n", category, cateSort)

		// 使用批量更新优化
		result := tx.Model(&ToolsItem{}).
			Where("category = ?", category).
			Update("cate_sort", cateSort)

		if result.Error != nil {
			tx.Rollback()
			return fmt.Errorf("更新分类[%s]失败: %v", category, result.Error)
		}

		// 检查影响行数（可选）
		if result.RowsAffected == 0 {
			fmt.Printf("警告: 分类[%s]未找到对应数据\n", category)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %v", err)
	}

	//fmt.Println("分类排序更新成功")
	return nil
}

// 强化命令排序更新方法
func (s *Tool) UpdateCommandSorts(sorts []map[string]interface{}) error {
	//fmt.Printf("[DEBUG] 收到命令排序请求: %+v\n", sorts)

	db := s.Db()
	if db == nil {
		return fmt.Errorf("数据库连接未初始化")
	}

	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			fmt.Printf("发生panic: %v\n", r)
		}
	}()

	for _, sort := range sorts {
		// 安全类型转换
		id, err := GetUint(sort["id"])
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("无效ID格式: %v", err)
		}

		cmdSort, err := GetUint(sort["cmdSort"])
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("无效排序值: %v", err)
		}

		category, ok := sort["category"].(string)
		if !ok || category == "" {
			tx.Rollback()
			return fmt.Errorf("无效分类名称")
		}

		// 批量更新
		result := tx.Model(&ToolsItem{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"cmd_sort": cmdSort,
				"category": category,
			})

		if result.Error != nil {
			tx.Rollback()
			return fmt.Errorf("更新记录失败(ID:%d): %v", id, result.Error)
		}

		if result.RowsAffected == 0 {
			fmt.Printf("[WARN] 未找到对应记录 ID: %d\n", id)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("事务提交失败: %v", err)
	}

	//fmt.Println("[INFO] 命令排序更新成功")
	return nil
}

// 通用类型转换函数
func GetUint(value interface{}) (uint, error) {
	switch v := value.(type) {
	case float64:
		if v < 0 {
			return 0, fmt.Errorf("负值转换错误")
		}
		return uint(v), nil
	case int:
		return uint(v), nil
	default:
		return 0, fmt.Errorf("无法转换类型: %T", v)
	}
}

// 在Tool结构体中添加以下方法
func (s *Tool) MoveCommandToCategory(request map[string]interface{}) error {
	db := s.Db()
	if db == nil {
		return fmt.Errorf("数据库连接未初始化")
	}

	// 类型断言获取参数
	id, ok1 := request["id"].(float64) // JavaScript数字会转为float64
	newCategory, ok2 := request["newCategory"].(string)
	newCmdSort, ok3 := request["newCmdSort"].(float64)

	if !ok1 || !ok2 || !ok3 {
		return fmt.Errorf("无效的参数格式")
	}

	// 开启事务
	tx := db.Begin()

	// 更新记录
	result := tx.Model(&ToolsItem{}).
		Where("id = ?", uint(id)).
		Updates(map[string]interface{}{
			"category": newCategory,
			"cmd_sort": int(newCmdSort),
		})

	if result.Error != nil {
		tx.Rollback()
		return fmt.Errorf("更新数据库失败: %v", result.Error)
	}

	if result.RowsAffected == 0 {
		tx.Rollback()
		return fmt.Errorf("未找到对应命令")
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %v", err)
	}

	return nil
}

func (s *Tool) ReadImageAsBase64(path string) string {
	// 添加详细错误日志
	log.Printf("Reading image: %s", path)

	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("读取文件错误: %v", err) // 添加错误详情
		return ""                     // 确保返回空字符串而不是 nil
	}

	// 添加内容类型检测
	mimeType := http.DetectContentType(data)
	if !strings.HasPrefix(mimeType, "image/") {
		log.Printf("非图片文件: %s, MIME: %s", path, mimeType)
		return ""
	}
	log.Printf("Reading image: %s", base64.StdEncoding.EncodeToString(data))
	return base64.StdEncoding.EncodeToString(data)
}
