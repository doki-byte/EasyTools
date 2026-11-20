package connect

import (
	"EasyTools/app/embed/config"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// GetNoSQLCommands 获取NoSQL命令列表（支持分页和过滤）
func GetNoSQLCommands(c *gin.Context) {
	manager := GetConfigManager()

	dbType := c.Query("dbType") // redis 或 mongodb
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("pageSize", "20")
	search := c.Query("search") // 搜索关键词

	commandsMap, err := manager.LoadNoSQLCommands()
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "加载配置失败: " + err.Error()})
		return
	}

	var allCommands []NoSQLCommand
	if dbType != "" {
		if commands, ok := commandsMap[dbType]; ok {
			allCommands = commands
		}
	} else {
		// 返回所有类型的命令
		for _, commands := range commandsMap {
			allCommands = append(allCommands, commands...)
		}
	}

	// 搜索过滤
	if search != "" {
		filtered := make([]NoSQLCommand, 0)
		searchLower := strings.ToLower(search)
		for _, cmd := range allCommands {
			if strings.Contains(strings.ToLower(cmd.Name), searchLower) ||
				strings.Contains(strings.ToLower(cmd.Command), searchLower) {
				filtered = append(filtered, cmd)
			}
		}
		allCommands = filtered
	}

	// 分页
	pageNum, _ := strconv.Atoi(page)
	pageSizeNum, _ := strconv.Atoi(pageSize)
	if pageNum < 1 {
		pageNum = 1
	}
	if pageSizeNum < 1 {
		pageSizeNum = 20
	}

	total := len(allCommands)
	start := (pageNum - 1) * pageSizeNum
	end := start + pageSizeNum

	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	result := allCommands[start:end]

	c.JSON(200, gin.H{
		"code": 0,
		"data": gin.H{
			"list":     result,
			"total":    total,
			"page":     pageNum,
			"pageSize": pageSizeNum,
		},
	})
}

// CreateNoSQLCommand 创建NoSQL命令
func CreateNoSQLCommand(c *gin.Context) {
	manager := GetConfigManager()

	var cmd NoSQLCommand
	if err := c.ShouldBindJSON(&cmd); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "参数绑定失败: " + err.Error()})
		return
	}

	dbType := c.Query("dbType")
	if dbType == "" {
		c.JSON(200, gin.H{"code": 1, "msg": "dbType参数不能为空"})
		return
	}

	commandsMap, err := manager.LoadNoSQLCommands()
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "加载配置失败: " + err.Error()})
		return
	}

	commands, _ := commandsMap[dbType]
	// 生成ID
	if cmd.ID == "" {
		cmd.ID = dbType + "-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	}

	commands = append(commands, cmd)
	commandsMap[dbType] = commands

	if err := manager.SaveNoSQLCommands(commandsMap); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "保存配置失败: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 0, "msg": "创建成功", "data": cmd})
}

// UpdateNoSQLCommand 更新NoSQL命令
func UpdateNoSQLCommand(c *gin.Context) {
	manager := GetConfigManager()

	id := c.Param("id")
	dbType := c.Query("dbType")
	if dbType == "" {
		c.JSON(200, gin.H{"code": 1, "msg": "dbType参数不能为空"})
		return
	}

	var cmd NoSQLCommand
	if err := c.ShouldBindJSON(&cmd); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "参数绑定失败: " + err.Error()})
		return
	}

	commandsMap, err := manager.LoadNoSQLCommands()
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "加载配置失败: " + err.Error()})
		return
	}

	commands := commandsMap[dbType]
	found := false
	for i, c := range commands {
		if c.ID == id {
			cmd.ID = id
			commands[i] = cmd
			found = true
			break
		}
	}

	if !found {
		c.JSON(200, gin.H{"code": 1, "msg": "未找到该命令"})
		return
	}

	commandsMap[dbType] = commands
	if err := manager.SaveNoSQLCommands(commandsMap); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "保存配置失败: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 0, "msg": "更新成功", "data": cmd})
}

// DeleteNoSQLCommand 删除NoSQL命令
func DeleteNoSQLCommand(c *gin.Context) {
	manager := GetConfigManager()

	id := c.Param("id")
	dbType := c.Query("dbType")
	if dbType == "" {
		c.JSON(200, gin.H{"code": 1, "msg": "dbType参数不能为空"})
		return
	}

	commandsMap, err := manager.LoadNoSQLCommands()
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "加载配置失败: " + err.Error()})
		return
	}

	commands := commandsMap[dbType]
	newCommands := make([]NoSQLCommand, 0)
	for _, c := range commands {
		if c.ID != id {
			newCommands = append(newCommands, c)
		}
	}

	commandsMap[dbType] = newCommands
	if err := manager.SaveNoSQLCommands(commandsMap); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "保存配置失败: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 0, "msg": "删除成功"})
}

// GetSQLTemplates 获取SQL模板列表（支持分页和过滤）
func GetSQLTemplates(c *gin.Context) {
	manager := GetConfigManager()

	dbType := c.Query("dbType")
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("pageSize", "20")
	search := c.Query("search")
	category := c.Query("category")

	templates, err := manager.LoadSQLTemplates()
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "加载配置失败: " + err.Error()})
		return
	}

	// 过滤
	filtered := make([]SQLTemplate, 0)
	for _, tpl := range templates {
		if dbType != "" && tpl.DBType != dbType {
			continue
		}
		if category != "" && tpl.Category != category {
			continue
		}
		if search != "" {
			searchLower := strings.ToLower(search)
			if !strings.Contains(strings.ToLower(tpl.Name), searchLower) &&
				!strings.Contains(strings.ToLower(tpl.Template), searchLower) {
				continue
			}
		}
		filtered = append(filtered, tpl)
	}

	// 分页
	pageNum, _ := strconv.Atoi(page)
	pageSizeNum, _ := strconv.Atoi(pageSize)
	if pageNum < 1 {
		pageNum = 1
	}
	if pageSizeNum < 1 {
		pageSizeNum = 20
	}

	total := len(filtered)
	start := (pageNum - 1) * pageSizeNum
	end := start + pageSizeNum

	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	result := filtered[start:end]

	c.JSON(200, gin.H{
		"code": 0,
		"data": gin.H{
			"list":     result,
			"total":    total,
			"page":     pageNum,
			"pageSize": pageSizeNum,
		},
	})
}

// CreateSQLTemplate 创建SQL模板
func CreateSQLTemplate(c *gin.Context) {
	manager := GetConfigManager()

	var tpl SQLTemplate
	if err := c.ShouldBindJSON(&tpl); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "参数绑定失败: " + err.Error()})
		return
	}

	templates, err := manager.LoadSQLTemplates()
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "加载配置失败: " + err.Error()})
		return
	}

	// 生成ID
	if tpl.ID == "" {
		tpl.ID = "sql-" + tpl.DBType + "-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	}

	templates = append(templates, tpl)

	if err := manager.SaveSQLTemplates(templates); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "保存配置失败: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 0, "msg": "创建成功", "data": tpl})
}

// UpdateSQLTemplate 更新SQL模板
func UpdateSQLTemplate(c *gin.Context) {
	manager := GetConfigManager()

	id := c.Param("id")

	var tpl SQLTemplate
	if err := c.ShouldBindJSON(&tpl); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "参数绑定失败: " + err.Error()})
		return
	}

	templates, err := manager.LoadSQLTemplates()
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "加载配置失败: " + err.Error()})
		return
	}

	found := false
	for i, t := range templates {
		if t.ID == id {
			tpl.ID = id
			templates[i] = tpl
			found = true
			break
		}
	}

	if !found {
		c.JSON(200, gin.H{"code": 1, "msg": "未找到该模板"})
		return
	}

	if err := manager.SaveSQLTemplates(templates); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "保存配置失败: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 0, "msg": "更新成功", "data": tpl})
}

// DeleteSQLTemplate 删除SQL模板
func DeleteSQLTemplate(c *gin.Context) {
	manager := GetConfigManager()

	id := c.Param("id")

	templates, err := manager.LoadSQLTemplates()
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "加载配置失败: " + err.Error()})
		return
	}

	newTemplates := make([]SQLTemplate, 0)
	for _, t := range templates {
		if t.ID != id {
			newTemplates = append(newTemplates, t)
		}
	}

	if err := manager.SaveSQLTemplates(newTemplates); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "保存配置失败: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 0, "msg": "删除成功"})
}

// ResetNoSQLCommands 恢复NoSQL命令为初始值
func ResetNoSQLCommands(c *gin.Context) {
	manager := GetConfigManager()

	// 从嵌入文件读取默认配置
	data, err := config.Assets.ReadFile("nosql_commands.json")
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "读取默认配置失败: " + err.Error()})
		return
	}

	var defaultCommands map[string][]NoSQLCommand
	if err := json.Unmarshal(data, &defaultCommands); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "解析默认配置失败: " + err.Error()})
		return
	}

	// 保存为当前配置
	if err := manager.SaveNoSQLCommands(defaultCommands); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "保存配置失败: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 0, "msg": "恢复初始值成功"})
}

// ResetSQLTemplates 恢复SQL模板为初始值
func ResetSQLTemplates(c *gin.Context) {
	manager := GetConfigManager()

	// 从嵌入文件读取默认配置
	data, err := config.Assets.ReadFile("sql_templates.json")
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "读取默认配置失败: " + err.Error()})
		return
	}

	var defaultData struct {
		Templates []SQLTemplate `json:"templates"`
	}
	if err := json.Unmarshal(data, &defaultData); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "解析默认配置失败: " + err.Error()})
		return
	}

	// 保存为当前配置
	if err := manager.SaveSQLTemplates(defaultData.Templates); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "保存配置失败: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 0, "msg": "恢复初始值成功"})
}

// SQLTemplateCategory SQL模板分类结构
type SQLTemplateCategory struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GetSQLTemplateCategories 获取SQL模板分类列表
func GetSQLTemplateCategories(c *gin.Context) {
	manager := GetConfigManager()

	templates, err := manager.LoadSQLTemplates()
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "加载配置失败: " + err.Error()})
		return
	}

	// 从所有模板中提取唯一的分类
	categoryMap := make(map[string]bool)
	for _, tpl := range templates {
		if tpl.Category != "" {
			categoryMap[tpl.Category] = true
		}
	}

	// 转换为分类列表
	categories := make([]SQLTemplateCategory, 0)
	for name := range categoryMap {
		categories = append(categories, SQLTemplateCategory{
			ID:   name, // 使用名称作为ID
			Name: name,
		})
	}

	c.JSON(200, gin.H{
		"code": 0,
		"data": categories,
	})
}

// CreateSQLTemplateCategory 创建SQL模板分类
func CreateSQLTemplateCategory(c *gin.Context) {
	manager := GetConfigManager()

	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "参数绑定失败: " + err.Error()})
		return
	}

	if req.Name == "" {
		c.JSON(200, gin.H{"code": 1, "msg": "分类名称不能为空"})
		return
	}

	// 检查分类是否已存在
	templates, err := manager.LoadSQLTemplates()
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "加载配置失败: " + err.Error()})
		return
	}

	// 检查分类是否已存在
	for _, tpl := range templates {
		if tpl.Category == req.Name {
			c.JSON(200, gin.H{"code": 1, "msg": "分类已存在"})
			return
		}
	}

	// 分类创建成功（实际上分类是通过SQL模板使用的，这里只是验证）
	c.JSON(200, gin.H{
		"code": 0,
		"msg":  "创建分类成功",
		"data": SQLTemplateCategory{
			ID:   req.Name,
			Name: req.Name,
		},
	})
}

// UpdateSQLTemplateCategory 更新SQL模板分类
func UpdateSQLTemplateCategory(c *gin.Context) {
	manager := GetConfigManager()

	id := c.Param("id")
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "参数绑定失败: " + err.Error()})
		return
	}

	if req.Name == "" {
		c.JSON(200, gin.H{"code": 1, "msg": "分类名称不能为空"})
		return
	}

	// 加载所有模板
	templates, err := manager.LoadSQLTemplates()
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "加载配置失败: " + err.Error()})
		return
	}

	// 更新所有使用该分类的模板
	updated := false
	for i := range templates {
		if templates[i].Category == id {
			templates[i].Category = req.Name
			updated = true
		}
	}

	if !updated {
		c.JSON(200, gin.H{"code": 1, "msg": "未找到该分类"})
		return
	}

	// 保存更新后的模板
	if err := manager.SaveSQLTemplates(templates); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "保存配置失败: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"code": 0,
		"msg":  "更新分类成功",
		"data": SQLTemplateCategory{
			ID:   req.Name,
			Name: req.Name,
		},
	})
}

// DeleteSQLTemplateCategory 删除SQL模板分类
func DeleteSQLTemplateCategory(c *gin.Context) {
	manager := GetConfigManager()

	id := c.Param("id")

	// 加载所有模板
	templates, err := manager.LoadSQLTemplates()
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "加载配置失败: " + err.Error()})
		return
	}

	// 检查是否有模板使用该分类
	hasTemplates := false
	for _, tpl := range templates {
		if tpl.Category == id {
			hasTemplates = true
			break
		}
	}

	if hasTemplates {
		// 删除分类时，将所有使用该分类的模板的分类字段清空
		for i := range templates {
			if templates[i].Category == id {
				templates[i].Category = ""
			}
		}

		// 保存更新后的模板
		if err := manager.SaveSQLTemplates(templates); err != nil {
			c.JSON(200, gin.H{"code": 1, "msg": "保存配置失败: " + err.Error()})
			return
		}
	}

	c.JSON(200, gin.H{"code": 0, "msg": "删除分类成功"})
}
