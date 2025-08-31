package controller

import (
	"EasyTools/app/model"
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Site 控制器
type Site struct {
	Base
}

// SiteItem 站点结构体
type SiteItem struct {
	ID       int    `json:"id"`
	Category string `json:"category"`
	Title    string `json:"title"`
	Remark   string `json:"remark"`
	URL      string `json:"url"`
	Icon     string `json:"icon,omitempty"` // 可选字段
}

// Category 结构体，用于前端返回
type SiteCategory struct {
	Title string     `json:"title"`
	List  []SiteItem `json:"list"`
}

// NewSite 创建新的 Site 控制器
func NewSite() *Site {
	return &Site{}
}

// TableName 指定表名
func (SiteItem) TableName() string {
	return "sites"
}

// 获取分类
func (s *Site) GetCategoryList() ([]map[string]interface{}, error) {
	db := s.db()
	if db == nil {
		return nil, fmt.Errorf("数据库连接未初始化")
	}

	// 查询分类及最大CateSort，按降序排列
	var categories []struct {
		Category string `gorm:"column:category"`
		CateSort int    `gorm:"column:cate_sort"`
	}

	err := db.Model(&SiteItem{}).
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

// 获取所有站点，按分类返回
func (s *Site) GetAllSites() ([]SiteCategory, error) {
	var sitesCategories []SiteCategory

	db := s.db()
	if db == nil {
		return nil, fmt.Errorf("数据库连接未初始化")
	}

	// 获取分类及排序（与GetCategoryList逻辑保持一致）
	var categories []struct {
		Category string `gorm:"column:category"`
		CateSort int    `gorm:"column:cate_sort"`
	}

	err := db.Model(&SiteItem{}).
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
		var sites []SiteItem
		err := db.Where("category = ?", c.Category).
			Order("site_sort DESC"). // 添加工具排序
			Find(&sites).Error

		if err != nil {
			continue
		}

		if len(sites) > 0 {
			sitesCategories = append(sitesCategories, SiteCategory{
				Title: c.Category,
				List:  sites,
			})
		}
	}

	if len(sitesCategories) == 0 {
		return nil, fmt.Errorf("查询完成，但没有数据")
	}

	return sitesCategories, nil
}

// GetSearchSites 按 title 查询站点并按分类返回
func (s *Site) GetSearchSites(title string) ([]SiteCategory, error) {
	var categories []SiteCategory

	// 获取数据库连接
	db := s.db()
	if db == nil {
		return nil, fmt.Errorf("数据库连接未初始化")
	}

	// 获取符合条件的分类名称
	var categoryNames []string
	err := db.Model(&SiteItem{}).
		Where("title LIKE ?", "%"+title+"%").
		Distinct("category").
		Pluck("category", &categoryNames).
		Error
	if err != nil {
		return nil, fmt.Errorf("获取分类失败: %v", err)
	}

	if len(categoryNames) == 0 {
		return nil, fmt.Errorf("没有符合条件的分类数据")
	}

	// 遍历分类名称，获取对应的站点
	for _, categoryName := range categoryNames {
		var sites []SiteItem
		err := db.Where("category = ? AND title LIKE ?", categoryName, "%"+title+"%").Find(&sites).Error
		if err != nil {
			// 跳过失败的分类，但记录日志
			s.log(fmt.Sprintf("查询分类 '%s' 的站点失败: %v", categoryName, err))
			continue
		}

		// 添加到返回结构
		if len(sites) > 0 {
			categories = append(categories, SiteCategory{
				Title: categoryName,
				List:  sites,
			})
		}
	}

	if len(categories) == 0 {
		return nil, fmt.Errorf("查询完成，但没有符合条件的数据")
	}

	return categories, nil
}

// FetchSiteInfo 自动获取标题、描述并下载 favicon 到 ./icon 目录。
// 返回 map[string]string 包含 keys: "title", "remark", "icon"（icon 是保存的文件名，如 "a1b2c3d4.png"）
func (s *Site) FetchSiteInfo(rawURL string) (map[string]string, error) {
	result := map[string]string{
		"title":  "",
		"remark": "",
		"icon":   "",
	}

	if rawURL == "" {
		return result, fmt.Errorf("url 为空")
	}

	// 如果用户没有写 scheme，默认加上 http://
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "http://" + rawURL
	}

	parsedBase, err := url.Parse(rawURL)
	if err != nil {
		return result, fmt.Errorf("无法解析 URL: %v", err)
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   12 * time.Second,
	}

	// 获取页面
	req, _ := http.NewRequest("GET", rawURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36 Edg/139.0.0.0")

	resp, err := client.Do(req)
	if err != nil {
		return result, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 解析 HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return result, fmt.Errorf("解析 HTML 失败: %v", err)
	}

	// title
	title := strings.TrimSpace(doc.Find("title").First().Text())
	result["title"] = title

	// meta description 优先 name=description，然后 og:description
	desc, _ := doc.Find("meta[name='description']").Attr("content")
	if desc == "" {
		desc, _ = doc.Find("meta[property='og:description']").Attr("content")
	}
	result["remark"] = strings.TrimSpace(desc)

	// 找 favicon 链接：查找 rel 包含 icon 的 <link>
	var iconHref string
	doc.Find("link[rel]").EachWithBreak(func(i int, sSel *goquery.Selection) bool {
		rel, _ := sSel.Attr("rel")
		if rel != "" && strings.Contains(strings.ToLower(rel), "icon") {
			href, ok := sSel.Attr("href")
			if ok && strings.TrimSpace(href) != "" {
				iconHref = strings.TrimSpace(href)
				return false // stop
			}
		}
		return true
	})

	// 若没找到 link，则尝试 site root /favicon.ico
	if iconHref == "" {
		iconHref = "/favicon.ico"
	}

	// 解析 icon URL（支持相对路径）
	iconURL, err := url.Parse(iconHref)
	if err != nil {
		// fallback to root favicon
		iconURL = &url.URL{Path: "/favicon.ico"}
	}
	iconAbs := parsedBase.ResolveReference(iconURL).String()

	// 下载 icon
	saveName, err := downloadAndSaveIcon(client, iconAbs)
	if err != nil {
		// 下载失败：尝试 /favicon.ico 作为最后手段（如果之前不是/fav）
		if !strings.HasSuffix(strings.ToLower(iconAbs), "/favicon.ico") {
			rootFav := parsedBase.Scheme + "://" + parsedBase.Host + "/favicon.ico"
			if saveName2, err2 := downloadAndSaveIcon(client, rootFav); err2 == nil {
				result["icon"] = saveName2
				return result, nil
			}
		}
		// 彻底失败也不返回 error（只是不给 icon），但把错误记录返回
		return result, fmt.Errorf("获取图标失败: %v", err)
	}

	result["icon"] = saveName
	return result, nil
}

// downloadAndSaveIcon 下载 iconUrl 并保存到 ./icon/<md5>.<ext>，返回文件名（不含路径）
func downloadAndSaveIcon(client *http.Client, iconUrl string) (string, error) {
	// 请求 icon
	req, _ := http.NewRequest("GET", iconUrl, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36 Edg/139.0.0.0")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("下载图标失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("下载图标返回状态 %d", resp.StatusCode)
	}

	// 读少量头部判断类型和文件扩展名
	contentType := resp.Header.Get("Content-Type")
	ext := ""
	if contentType != "" {
		exts, _ := mime.ExtensionsByType(strings.Split(contentType, ";")[0])
		if len(exts) > 0 {
			ext = exts[0] // e.g. .png
		}
	}
	// 如果从 URL path 能获取 ext，则优先用它
	u, err := url.Parse(iconUrl)
	if err == nil {
		if pext := path.Ext(u.Path); pext != "" {
			ext = pext
		}
	}
	if ext == "" {
		ext = ".ico" // fallback
	}

	// 计算 md5 作为文件名
	h := md5.New()
	h.Write([]byte(iconUrl))
	name := hex.EncodeToString(h.Sum(nil)) + ext

	baseDir := GetAppBaseDir()

	iconDir := filepath.Join(baseDir, "icon")
	if err := os.MkdirAll(iconDir, 0755); err != nil {
		return "", fmt.Errorf("创建 icon 目录失败: %v", err)
	}

	savePath := filepath.Join(iconDir, name)

	// 如果文件已存在就直接返回
	if _, err := os.Stat(savePath); err == nil {
		return name, nil
	}

	// 写入文件
	out, err := os.Create(savePath)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %v", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		// 如果失败，尝试删除残缺文件
		_ = os.Remove(savePath)
		return "", fmt.Errorf("保存图标失败: %v", err)
	}

	return name, nil
}

// 新增工具
func (s *Site) AddSite(site SiteItem) (int, error) {
	// 获取数据库连接
	db := s.db()
	if db == nil {
		// s.log("数据库连接未初始化")
		return 0, fmt.Errorf("数据库连接未初始化")
	}

	// 插入新工具
	err := db.Create(&site).Error
	if err != nil {
		// s.log(fmt.Sprintf("新增工具失败: %v", err))
		return 0, fmt.Errorf("新增工具失败: %v", err)
	}

	// 返回插入的工具ID
	return site.ID, nil
}

// 修改工具
func (s *Site) UpdateSite(id int, UpdateSite SiteItem) error {
	// 获取数据库连接
	db := s.db()
	if db == nil {
		// s.log("数据库连接未初始化")
		return fmt.Errorf("数据库连接未初始化")
	}

	// 根据工具的id更新工具
	err := db.Model(&SiteItem{}).Where("id = ?", id).Save(UpdateSite).Error
	if err != nil {
		// s.log(fmt.Sprintf("修改工具失败: %v", err))
		return fmt.Errorf("修改工具失败: %v", err)
	}

	return nil
}

// 删除工具
func (s *Site) DeleteSite(id int) error {
	// 获取数据库连接
	db := s.db()
	if db == nil {
		return fmt.Errorf("数据库连接未初始化")
	}

	// 先查询站点信息，获取图标文件名
	var site SiteItem
	err := db.Where("id = ?", id).First(&site).Error
	if err != nil {
		return fmt.Errorf("查询站点失败: %v", err)
	}

	// 删除数据库记录
	err = db.Where("id = ?", id).Delete(&SiteItem{}).Error
	if err != nil {
		return fmt.Errorf("删除工具失败: %v", err)
	}

	// 如果站点有图标，删除对应的图标文件
	if site.Icon != "" {
		baseDir := GetAppBaseDir()
		iconPath := filepath.Join(baseDir, "icon", site.Icon)

		// 检查文件是否存在
		if _, err := os.Stat(iconPath); err == nil {
			// 文件存在，尝试删除
			if err := os.Remove(iconPath); err != nil {
				// 删除失败，记录错误但不中断流程
				s.log(fmt.Sprintf("删除图标文件失败: %v", err))
			} else {
				s.log(fmt.Sprintf("已删除图标文件: %s", iconPath))
			}
		}
	}

	return nil
}

// 修改分类名称（将所有属于 oldCategory 的工具改为 newCategory）
func (s *Site) UpdateSiteCategory(oldCategory, newCategory string) error {
	db := s.db()
	if db == nil {
		return fmt.Errorf("数据库连接未初始化")
	}

	err := db.Model(&model.Sites{}).Where("category = ?", oldCategory).Update("category", newCategory).Error
	if err != nil {
		return fmt.Errorf("修改分类失败: %v", err)
	}

	return nil
}

// 删除分类及其下所有工具
func (s *Site) DeleteSiteCategory(category string) error {
	db := s.db()
	if db == nil {
		return fmt.Errorf("数据库连接未初始化")
	}

	err := db.Where("category = ?", category).Delete(&model.Sites{}).Error
	if err != nil {
		return fmt.Errorf("删除分类失败: %v", err)
	}

	return nil
}

// 更新分类排序
func (s *Site) UpdateCategorySorts(sorts []map[string]interface{}) error {
	//fmt.Printf("调试信息 - 收到排序请求: %+v\n", sorts) // 添加详细日志

	db := s.db()
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
		result := tx.Model(&SiteItem{}).
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
func (s *Site) UpdateCommandSorts(sorts []map[string]interface{}) error {
	//fmt.Printf("[DEBUG] 收到命令排序请求: %+v\n", sorts)

	db := s.db()
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
		result := tx.Model(&SiteItem{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"site_sort": cmdSort,
				"category":  category,
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

// 在Site结构体中添加以下方法
func (s *Site) MoveCommandToCategory(request map[string]interface{}) error {
	db := s.db()
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
	result := tx.Model(&SiteItem{}).
		Where("id = ?", uint(id)).
		Updates(map[string]interface{}{
			"category":  newCategory,
			"site_sort": int(newCmdSort),
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
