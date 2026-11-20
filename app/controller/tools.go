package controller

import (
	"EasyTools/app/controller/system"
	"EasyTools/app/model"
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mat/besticon/v3/ico"
	"github.com/orcastor/fico"
	lnk "github.com/parsiya/golnk"
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

// 自动生成图标并保存到 icon 目录
func (s *Tool) AutoGenerateIcon(filePath string) (string, error) {
	// 确保 icon 目录存在
	iconDir := filepath.Join(system.GetAppBaseDir(), "icon")

	// 提取图标数据
	iconData, err := s.extractIcon(filePath)
	if err != nil {
		return "", fmt.Errorf("提取图标失败: %v", err)
	}

	// 如果提取失败，返回空字符串（使用默认图标）
	if len(iconData) == 0 {
		return "", nil
	}

	// 生成唯一的图标文件名
	baseName := filepath.Base(filePath)
	nameWithoutExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	fileName := fmt.Sprintf("%s_%d.png", nameWithoutExt, time.Now().UnixNano())
	iconPath := filepath.Join(iconDir, fileName)

	// 保存图标文件
	if err := os.WriteFile(iconPath, iconData, 0644); err != nil {
		return "", fmt.Errorf("保存图标文件失败: %v", err)
	}

	return fileName, nil
}

// 提取图标的核心方法
func (s *Tool) extractIcon(filePath string) ([]byte, error) {
	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("文件不存在: %s", filePath)
	}

	ext := strings.ToLower(filepath.Ext(filePath))

	// 根据文件类型使用不同的提取方法
	switch ext {
	case ".lnk":
		return s.extractLnkIcon(filePath)
	case ".exe", ".dll":
		return s.extractExeIcon(filePath)
	case ".ico":
		return s.extractIcoFile(filePath)
	default:
		// 对于其他文件类型，尝试使用 fico 提取
		return s.extractWithFico(filePath)
	}
}

// 提取 lnk 文件的图标
func (s *Tool) extractLnkIcon(lnkPath string) ([]byte, error) {
	// 解析 lnk 文件获取目标路径
	lnkFile, err := lnk.File(lnkPath)
	if err != nil {
		return nil, fmt.Errorf("解析lnk文件失败: %v", err)
	}

	targetPath := lnkFile.LinkInfo.LocalBasePath
	if targetPath == "" {
		return nil, fmt.Errorf("无法获取lnk目标路径")
	}

	// 递归提取目标文件的图标
	return s.extractIcon(targetPath)
}

// 提取 exe/dll 文件的图标
func (s *Tool) extractExeIcon(exePath string) ([]byte, error) {
	var buf bytes.Buffer

	// 使用 fico 提取图标
	config := fico.Config{
		Format: "png",
		Width:  64,
		Height: 64,
	}

	err := fico.F2ICO(&buf, exePath, config)
	if err != nil {
		// 如果 fico 失败，尝试备用方法
		return s.extractWithFico(exePath)
	}

	// 检查是否成功提取到图标数据
	if buf.Len() == 0 {
		return nil, fmt.Errorf("未找到图标数据")
	}

	return buf.Bytes(), nil
}

// 使用 fico 库提取图标（通用方法）
func (s *Tool) extractWithFico(filePath string) ([]byte, error) {
	var buf bytes.Buffer

	config := fico.Config{
		Format: "png",
		Width:  64,
		Height: 64,
	}

	err := fico.F2ICO(&buf, filePath, config)
	if err != nil {
		return nil, fmt.Errorf("fico提取图标失败: %v", err)
	}

	if buf.Len() == 0 {
		return nil, fmt.Errorf("fico未找到图标")
	}

	return buf.Bytes(), nil
}

// 提取 ico 文件
func (s *Tool) extractIcoFile(icoPath string) ([]byte, error) {
	// 读取 ico 文件
	file, err := os.Open(icoPath)
	if err != nil {
		return nil, fmt.Errorf("打开ico文件失败: %v", err)
	}
	defer file.Close()

	// 解码 ico 文件
	img, err := ico.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("解码ico文件失败: %v", err)
	}

	// 转换为 png
	return s.imageToPng(img)
}

// 将 image.Image 转换为 PNG 字节数据
func (s *Tool) imageToPng(img image.Image) ([]byte, error) {
	var buf bytes.Buffer

	// 调整图片大小到标准尺寸
	resizedImg := s.resizeImage(img, 64, 64)

	// 编码为 PNG
	err := png.Encode(&buf, resizedImg)
	if err != nil {
		return nil, fmt.Errorf("PNG编码失败: %v", err)
	}

	return buf.Bytes(), nil
}

// 调整图片大小
func (s *Tool) resizeImage(img image.Image, width, height int) image.Image {
	// 简单的调整大小实现
	bounds := img.Bounds()
	imgWidth := bounds.Dx()
	imgHeight := bounds.Dy()

	// 如果图片已经小于目标尺寸，直接返回
	if imgWidth <= width && imgHeight <= height {
		return img
	}

	// 计算缩放比例
	scaleX := float64(width) / float64(imgWidth)
	scaleY := float64(height) / float64(imgHeight)
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	newWidth := int(float64(imgWidth) * scale)
	newHeight := int(float64(imgHeight) * scale)

	// 创建新图片
	newImg := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// 简单的最近邻缩放
	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			srcX := int(float64(x) / scale)
			srcY := int(float64(y) / scale)
			newImg.Set(x, y, img.At(srcX, srcY))
		}
	}

	return newImg
}

// 分析文件路径信息
func (s *Tool) AnalyzeFilePath(fullPath string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// 获取文件信息
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %v", err)
	}

	// 提取路径信息
	dir := filepath.Dir(fullPath)
	fileName := filepath.Base(fullPath)
	fileExt := filepath.Ext(fileName)
	nameWithoutExt := strings.TrimSuffix(fileName, fileExt)

	result["directory"] = dir
	result["filename"] = fileName
	result["filenameWithoutExt"] = nameWithoutExt
	result["extension"] = strings.ToLower(fileExt)
	result["fullPath"] = fullPath
	result["isDirectory"] = fileInfo.IsDir()

	// 根据文件类型推断建议的命令和名称
	suggestedCmd, suggestedName := s.getSuggestedCommand(fileName, fileExt, nameWithoutExt)
	result["suggestedCmd"] = suggestedCmd
	result["suggestedName"] = suggestedName

	return result, nil
}

// 获取建议的命令和名称
func (s *Tool) getSuggestedCommand(fileName, fileExt, nameWithoutExt string) (string, string) {
	ext := strings.ToLower(fileExt)

	switch ext {
	case ".exe":
		return fileName, nameWithoutExt
	case ".jar":
		return "java -jar " + fileName, nameWithoutExt
	case ".bat", ".cmd":
		return fileName, nameWithoutExt
	case ".py":
		return "python " + fileName, nameWithoutExt
	case ".ps1":
		return "powershell -File " + fileName, nameWithoutExt
	case ".sh":
		return "./" + fileName, nameWithoutExt
	case ".lnk":
		return fileName, nameWithoutExt
	default:
		// 对于未知类型，尝试直接执行
		return fileName, nameWithoutExt
	}
}
