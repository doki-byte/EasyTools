package service

import (
	"EasyTools/app/connect/redis/internal/define"
	"EasyTools/app/connect/redis/internal/helper"
	"context"
	"errors"
	"strconv"
	"strings"
	"time"
)

// DbList 数据库列表
func DbList(identity string) ([]*define.DbItem, error) {
	if identity == "" {
		return nil, errors.New("连接唯一标识不能为空")
	}
	rdb, err := helper.GetRedisClient(identity, 0)
	if err != nil {
		return nil, err
	}

	// info 获取数据库键的个数
	keySpace, err := rdb.Info(context.Background(), "keyspace").Result()
	if err != nil {
		return nil, err
	}
	// keyspace 数据格式
	// # Keyspace
	// db0:keys=2,avg_ttl...
	//
	m := make(map[string]int)
	v := strings.Split(keySpace, "\n")
	for i := 1; i < len(v)-1; i++ {
		databases := strings.Split(v[i], ":")
		if len(databases) < 2 {
			continue
		}
		vv := strings.Split(databases[1], ",")
		if len(vv) < 1 {
			continue
		}
		keyNumber := strings.Split(vv[0], "=")
		if len(keyNumber) < 2 {
			continue
		}
		num, err := strconv.Atoi(keyNumber[1])
		if err != nil {
			continue
		}
		m[databases[0]] = num
	}
	// config get 获取数据库的个数
	databasesRes, err := rdb.ConfigGet(context.Background(), "databases").Result()
	if err != nil {
		return nil, err
	}
	if len(databasesRes) < 2 {
		return nil, errors.New("连接数据异常")
	}
	dbNum, err := strconv.Atoi(databasesRes[1].(string))
	if err != nil {
		return nil, err
	}
	data := make([]*define.DbItem, 0)
	for i := 0; i < dbNum; i++ {
		item := &define.DbItem{
			Key: "db" + strconv.Itoa(i),
		}
		if n, ok := m["db"+strconv.Itoa(i)]; ok {
			item.Number = n
		}
		data = append(data, item)
	}
	return data, nil
}

// DbInfo 数据库详情
func DbInfo(identity string) ([]*define.KeyValue, error) {
	if identity == "" {
		return nil, errors.New("连接唯一标识不能为空")
	}
	rdb, err := helper.GetRedisClient(identity, 0)
	if err != nil {
		return nil, err
	}

	// info 获取数据库键的个数
	keySpace, err := rdb.Info(context.Background()).Result()
	if err != nil {
		return nil, err
	}
	// info 数据格式
	// key:value
	data := make([]*define.KeyValue, 0)
	infos := strings.Split(keySpace, "\n")
	for _, info := range infos {
		v := strings.Split(info, ":")
		if len(v) == 2 {
			data = append(data, &define.KeyValue{
				Key:   strings.TrimSpace(v[0]),
				Value: strings.TrimSpace(v[1]),
			})
		}
	}

	return data, nil
}

// parseCommand 解析命令字符串
func parseCommand(commandStr string) []interface{} {
	// 简单的命令解析，按空格分割，但处理引号内的内容
	args := make([]interface{}, 0)
	var currentArg strings.Builder
	inQuotes := false
	quoteChar := byte(0)

	for i := 0; i < len(commandStr); i++ {
		c := commandStr[i]

		switch {
		case c == '"' || c == '\'':
			if !inQuotes {
				inQuotes = true
				quoteChar = c
			} else if c == quoteChar {
				inQuotes = false
				if currentArg.Len() > 0 {
					args = append(args, currentArg.String())
					currentArg.Reset()
				}
			} else {
				currentArg.WriteByte(c)
			}
		case c == ' ' && !inQuotes:
			if currentArg.Len() > 0 {
				args = append(args, currentArg.String())
				currentArg.Reset()
			}
		default:
			currentArg.WriteByte(c)
		}
	}

	// 添加最后一个参数
	if currentArg.Len() > 0 {
		args = append(args, currentArg.String())
	}

	return args
}

// toString 将任意类型转换为字符串
func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case nil:
		return "(nil)"
	default:
		return "Unknown type"
	}
}

// GetCommandHistory 获取命令历史记录
func GetCommandHistory(identity string) ([]*define.CommandHistory, error) {
	// 这里可以实现从数据库或文件中读取历史记录
	// 暂时返回空数组
	return []*define.CommandHistory{}, nil
}

// ExecuteCommand 执行Redis命令
func ExecuteCommand(identity string, commandStr string) (*define.CommandResult, error) {
	if identity == "" {
		return nil, errors.New("连接唯一标识不能为空")
	}
	if commandStr == "" {
		return nil, errors.New("命令不能为空")
	}

	rdb, err := helper.GetRedisClient(identity, 0)
	if err != nil {
		return nil, err
	}

	// 解析命令
	args := parseCommand(commandStr)
	if len(args) == 0 {
		return nil, errors.New("无效的命令")
	}

	// 记录开始时间
	startTime := time.Now()

	// 执行命令
	result, err := rdb.Do(context.Background(), args...).Result()
	duration := time.Since(startTime).Milliseconds()

	if err != nil {
		return &define.CommandResult{
			Success:  false,
			Error:    err.Error(),
			Duration: duration,
		}, nil
	}

	// 根据命令类型格式化结果
	formattedResult := formatCommandResultByType(commandStr, result)

	return &define.CommandResult{
		Success:  true,
		Data:     formattedResult,
		Duration: duration,
	}, nil
}

// formatCommandResultByType 根据命令类型格式化结果
func formatCommandResultByType(commandStr string, result interface{}) interface{} {
	// 获取命令名称（第一个参数）
	cmdParts := strings.Fields(strings.ToLower(commandStr))
	if len(cmdParts) == 0 {
		return formatCommandResult(result)
	}

	command := cmdParts[0]

	switch command {
	case "info":
		// 解析 INFO 命令结果
		return parseRedisInfo(result)
	case "client", "client list":
		// 解析 CLIENT LIST 命令结果
		return parseClientList(result)
	case "config", "config get":
		// 解析 CONFIG GET 命令结果
		return parseConfigGet(result)
	case "slowlog", "slowlog get":
		// 解析 SLOWLOG GET 命令结果
		return parseSlowlogGet(result)
	default:
		// 其他命令使用默认格式化
		return formatCommandResult(result)
	}
}

// parseRedisInfo 解析 Redis INFO 命令结果
func parseRedisInfo(result interface{}) interface{} {
	var infoStr string
	switch v := result.(type) {
	case string:
		infoStr = v
	case []byte:
		infoStr = string(v)
	default:
		return formatCommandResult(result)
	}

	sections := make(map[string]map[string]string)
	var currentSection string

	lines := strings.Split(infoStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 检查是否是章节标题
		if strings.HasPrefix(line, "# ") {
			currentSection = strings.TrimSpace(line[2:])
			sections[currentSection] = make(map[string]string)
			continue
		}

		// 解析键值对
		if idx := strings.Index(line, ":"); idx > 0 && currentSection != "" {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])
			sections[currentSection][key] = value
		}
	}

	return sections
}

// parseClientList 解析 CLIENT LIST 命令结果
func parseClientList(result interface{}) interface{} {
	var clientStr string
	switch v := result.(type) {
	case string:
		clientStr = v
	case []byte:
		clientStr = string(v)
	default:
		return formatCommandResult(result)
	}

	clients := make([]map[string]string, 0)
	lines := strings.Split(clientStr, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		client := make(map[string]string)
		// 解析客户端信息，格式: key=value key=value
		fields := strings.Fields(line)
		for _, field := range fields {
			if idx := strings.Index(field, "="); idx > 0 {
				key := field[:idx]
				value := field[idx+1:]
				client[key] = value
			}
		}

		if len(client) > 0 {
			clients = append(clients, client)
		}
	}

	return clients
}

// parseConfigGet 解析 CONFIG GET 命令结果
func parseConfigGet(result interface{}) interface{} {
	switch v := result.(type) {
	case []interface{}:
		config := make(map[string]interface{})
		// CONFIG GET 返回 [key1, value1, key2, value2, ...]
		for i := 0; i < len(v); i += 2 {
			if i+1 < len(v) {
				key := toString(v[i])
				config[key] = v[i+1]
			}
		}
		return config
	default:
		return formatCommandResult(result)
	}
}

// parseSlowlogGet 解析 SLOWLOG GET 命令结果
func parseSlowlogGet(result interface{}) interface{} {
	switch v := result.(type) {
	case []interface{}:
		slowlogs := make([]map[string]interface{}, len(v))
		for i, item := range v {
			if itemSlice, ok := item.([]interface{}); ok {
				slowlog := make(map[string]interface{})
				if len(itemSlice) >= 4 {
					slowlog["id"] = itemSlice[0]
					slowlog["timestamp"] = itemSlice[1]
					slowlog["duration"] = itemSlice[2]
					slowlog["command"] = itemSlice[3]
					// 可能有其他字段
					if len(itemSlice) > 4 {
						slowlog["client"] = itemSlice[4]
					}
					if len(itemSlice) > 5 {
						slowlog["name"] = itemSlice[5]
					}
				}
				slowlogs[i] = slowlog
			}
		}
		return slowlogs
	default:
		return formatCommandResult(result)
	}
}

// formatCommandResult 格式化命令结果（原有函数保持不变）
func formatCommandResult(result interface{}) interface{} {
	switch v := result.(type) {
	case []interface{}:
		// 处理数组类型的结果
		formatted := make([]interface{}, len(v))
		for i, item := range v {
			formatted[i] = formatCommandResult(item)
		}
		return formatted
	case []byte:
		// 处理字节数组类型的结果
		return string(v)
	case string:
		// 处理字符串类型的结果
		return v
	case int64:
		// 处理整数类型的结果
		return v
	case nil:
		// 处理空结果
		return "(nil)"
	default:
		// 其他类型转换为字符串
		return toString(v)
	}
}
