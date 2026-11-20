package service

import (
	"EasyTools/app/controller/connect/nosql/internal/define"
	"EasyTools/app/controller/connect/nosql/internal/helper"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/ssh"
)

// DbList 数据库列表
func DbList(identity string) ([]*define.DbItem, error) {
	if identity == "" {
		return nil, errors.New("连接唯一标识不能为空")
	}

	// 获取连接信息
	conn, err := helper.GetConnection(identity)
	if err != nil {
		return nil, err
	}

	// 如果没有指定数据库类型，默认为 Redis
	dbType := conn.DbType
	if dbType == "" {
		dbType = "redis"
	}

	// 根据数据库类型选择列表方法
	dbTypeLower := strings.ToLower(dbType)
	switch dbTypeLower {
	case "redis":
		return getRedisDbList(identity)
	case "mongodb", "mongo":
		return getMongoDBDbList(identity)
	default:
		return nil, fmt.Errorf("不支持的数据库类型: %s，支持 Redis 和 MongoDB", dbType)
	}
}

// getRedisDbList 获取 Redis 数据库列表
func getRedisDbList(identity string) ([]*define.DbItem, error) {
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

	// 获取连接信息，判断数据库类型
	conn, err := helper.GetConnection(identity)
	if err != nil {
		return nil, err
	}

	// 如果没有指定数据库类型，默认为 Redis
	dbType := conn.DbType
	if dbType == "" {
		dbType = "redis"
	}

	// 根据数据库类型选择列表方法
	dbTypeLower := strings.ToLower(dbType)
	switch dbTypeLower {
	case "redis":
		return getRedisDbInfo(identity)
	case "mongodb", "mongo":
		return getMongoDBDbInfo(identity)
	default:
		return nil, fmt.Errorf("不支持的数据库类型: %s，支持 Redis 和 MongoDB", dbType)
	}
}

// getRedisDbInfo 获取 Redis 数据库详情
func getRedisDbInfo(identity string) ([]*define.KeyValue, error) {
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

// getMongoDBDbInfo 获取 MongoDB 数据库详情
func getMongoDBDbInfo(identity string) ([]*define.KeyValue, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 连接 MongoDB（使用 admin 数据库获取服务器信息）
	client, _, err := helper.GetMongoDBClient(identity, "admin")
	if err != nil {
		return nil, err
	}
	defer client.Disconnect(ctx)

	// 执行 serverStatus 命令获取服务器状态
	adminDb := client.Database("admin")
	var result bson.M
	err = adminDb.RunCommand(ctx, bson.D{{Key: "serverStatus", Value: 1}}).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("获取MongoDB服务器状态失败: %v", err)
	}

	// 转换为 KeyValue 格式（扁平化处理主要信息）
	data := make([]*define.KeyValue, 0)

	// 添加版本信息
	if version, ok := result["version"].(string); ok {
		data = append(data, &define.KeyValue{
			Key:   "version",
			Value: version,
		})
	}

	// 添加主机信息
	if host, ok := result["host"].(string); ok {
		data = append(data, &define.KeyValue{
			Key:   "host",
			Value: host,
		})
	}

	// 添加进程ID
	if pid, ok := result["pid"].(int32); ok {
		data = append(data, &define.KeyValue{
			Key:   "pid",
			Value: fmt.Sprintf("%d", pid),
		})
	}

	// 添加运行时间（秒）
	if uptime, ok := result["uptime"].(int64); ok {
		data = append(data, &define.KeyValue{
			Key:   "uptime",
			Value: fmt.Sprintf("%d秒", uptime),
		})
	}

	// 添加连接数信息
	if connections, ok := result["connections"].(bson.M); ok {
		if current, ok := connections["current"].(int32); ok {
			data = append(data, &define.KeyValue{
				Key:   "connections.current",
				Value: fmt.Sprintf("%d", current),
			})
		}
		if available, ok := connections["available"].(int32); ok {
			data = append(data, &define.KeyValue{
				Key:   "connections.available",
				Value: fmt.Sprintf("%d", available),
			})
		}
	}

	return data, nil
}

// getMongoDBDbList 获取 MongoDB 数据库列表
func getMongoDBDbList(identity string) ([]*define.DbItem, error) {
	// 获取连接信息
	conn, err := helper.GetConnection(identity)
	if err != nil {
		return nil, err
	}

	if conn.Port == "" {
		conn.Port = "27017"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 构建 MongoDB 连接URI
	var uri string
	if conn.Username != "" && conn.Password != "" {
		uri = fmt.Sprintf("mongodb://%s:%s@%s:%s/?connectTimeoutMS=10000&serverSelectionTimeoutMS=10000",
			conn.Username, conn.Password, conn.Addr, conn.Port)
	} else {
		uri = fmt.Sprintf("mongodb://%s:%s/?connectTimeoutMS=10000&serverSelectionTimeoutMS=10000",
			conn.Addr, conn.Port)
	}

	// 创建 MongoDB 客户端选项
	clientOptions := options.Client().ApplyURI(uri)

	// 如果是 SSH 连接，需要通过SSH隧道
	var sshClient *ssh.Client
	if conn.Type == "ssh" {
		// 验证SSH配置完整性
		if conn.SSHAddr == "" {
			return nil, fmt.Errorf("SSH隧道模式下，SSH地址不能为空")
		}
		if conn.SSHPort == "" {
			conn.SSHPort = "22"
		}
		if conn.SSHUsername == "" {
			return nil, fmt.Errorf("SSH隧道模式下，SSH用户名不能为空")
		}
		if conn.SSHPassword == "" {
			return nil, fmt.Errorf("SSH隧道模式下，SSH密码不能为空")
		}

		sshConfig := &ssh.ClientConfig{
			User: conn.SSHUsername,
			Auth: []ssh.AuthMethod{
				ssh.Password(conn.SSHPassword),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         10 * time.Second,
		}

		// 建立 SSH 连接
		sshAddr := fmt.Sprintf("%s:%s", conn.SSHAddr, conn.SSHPort)
		sshClient, err = ssh.Dial("tcp", sshAddr, sshConfig)
		if err != nil {
			return nil, fmt.Errorf("SSH连接失败 [%s@%s]: %v", conn.SSHUsername, sshAddr, err)
		}
		defer sshClient.Close()

		// 设置自定义Dialer，通过SSH隧道连接MongoDB
		clientOptions.SetDialer(&sshContextDialer{sshClient: sshClient})
	}

	// 连接MongoDB（支持直连和SSH隧道两种方式）
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("MongoDB连接失败: %v", err)
	}
	defer client.Disconnect(ctx)

	// 测试连接
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("MongoDB连接测试失败: %v", err)
	}

	// 获取数据库列表
	dbNames, err := client.ListDatabaseNames(ctx, map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("获取MongoDB数据库列表失败: %v", err)
	}

	// 转换为DbItem格式
	data := make([]*define.DbItem, 0, len(dbNames))
	for _, dbName := range dbNames {
		// 为每个数据库创建独立的 context，避免超时影响其他数据库
		collectionCtx, collectionCancel := context.WithTimeout(context.Background(), 5*time.Second)

		// 获取每个数据库的集合数量
		db := client.Database(dbName)
		collections, err := db.ListCollectionNames(collectionCtx, map[string]interface{}{})
		collectionCount := 0
		if err == nil {
			collectionCount = len(collections)
		} else {
			// 如果获取集合列表失败（可能是权限问题或连接问题），记录但继续处理
			// 对于某些系统数据库（如 admin, config, local），可能没有权限访问
			// EOF 错误通常表示连接问题，但我们仍然返回数据库信息，只是集合数为 0
		}
		collectionCancel()

		data = append(data, &define.DbItem{
			Key:    dbName,
			Number: collectionCount,
		})
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

// ExecuteCommand 执行数据库命令（Redis 或 MongoDB）
func ExecuteCommand(identity string, commandStr string) (*define.CommandResult, error) {
	if identity == "" {
		return nil, errors.New("连接唯一标识不能为空")
	}
	if commandStr == "" {
		return nil, errors.New("命令不能为空")
	}

	// 获取连接信息，判断数据库类型
	conn, err := helper.GetConnection(identity)
	if err != nil {
		return nil, err
	}

	dbType := strings.ToLower(conn.DbType)
	switch dbType {
	case "redis":
		return executeCommand(identity, commandStr)
	case "mongodb", "mongo":
		return executeMongoDBCommand(identity, commandStr)
	default:
		return nil, fmt.Errorf("不支持的数据库类型: %s", dbType)
	}
}

// executeCommand 执行 Redis 命令
func executeCommand(identity string, commandStr string) (*define.CommandResult, error) {
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

// executeMongoDBCommand 执行 MongoDB 命令
func executeMongoDBCommand(identity string, commandStr string) (*define.CommandResult, error) {
	// MongoDB 命令格式：dbName.collectionName.operation(args)
	// 例如：admin.runCommand({serverStatus: 1})
	//      test.users.find({})
	//      test.users.count()

	startTime := time.Now()

	// 解析命令：简单支持 dbName.runCommand(command) 格式
	// 更复杂的命令可能需要 JavaScript 解释器

	commandStr = strings.TrimSpace(commandStr)

	// 检查是否是 runCommand 格式
	if strings.Contains(commandStr, ".runCommand(") {
		result, err := executeMongoRunCommand(identity, commandStr)
		duration := time.Since(startTime).Milliseconds()

		if err != nil {
			return &define.CommandResult{
				Success:  false,
				Error:    err.Error(),
				Duration: duration,
			}, nil
		}

		return &define.CommandResult{
			Success:  true,
			Data:     result,
			Duration: duration,
		}, nil
	}

	// 检查是否是集合操作格式（如 db.collection.find()）
	if strings.Contains(commandStr, ".find(") ||
		strings.Contains(commandStr, ".count(") ||
		strings.Contains(commandStr, ".insertOne(") ||
		strings.Contains(commandStr, ".deleteOne(") {
		result, err := executeMongoCollectionCommand(identity, commandStr)
		duration := time.Since(startTime).Milliseconds()

		if err != nil {
			return &define.CommandResult{
				Success:  false,
				Error:    err.Error(),
				Duration: duration,
			}, nil
		}

		return &define.CommandResult{
			Success:  true,
			Data:     result,
			Duration: duration,
		}, nil
	}

	duration := time.Since(startTime).Milliseconds()
	return &define.CommandResult{
		Success:  false,
		Error:    "不支持的 MongoDB 命令格式。支持格式：\n1. dbName.runCommand({command: 1})\n2. dbName.collection.find({})\n3. dbName.collection.count()",
		Duration: duration,
	}, nil
}

// executeMongoRunCommand 执行 MongoDB runCommand
func executeMongoRunCommand(identity string, commandStr string) (interface{}, error) {
	// 解析命令：dbName.runCommand({command})
	// 例如：admin.runCommand({serverStatus: 1})

	parts := strings.Split(commandStr, ".runCommand(")
	if len(parts) != 2 {
		return nil, fmt.Errorf("无效的 runCommand 格式")
	}

	dbName := strings.TrimSpace(parts[0])
	commandJSON := strings.TrimSpace(parts[1])
	commandJSON = strings.TrimSuffix(commandJSON, ")")

	// 连接到 MongoDB
	client, db, err := helper.GetMongoDBClient(identity, dbName)
	if err != nil {
		return nil, err
	}
	defer client.Disconnect(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 解析命令 JSON
	var command bson.M
	err = bson.UnmarshalExtJSON([]byte(commandJSON), false, &command)
	if err != nil {
		return nil, fmt.Errorf("解析命令 JSON 失败: %v", err)
	}

	// 执行命令
	var result bson.M
	err = db.RunCommand(ctx, command).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("执行命令失败: %v", err)
	}

	// 转换为 JSON 字符串
	resultJSON, err := bson.MarshalExtJSON(result, false, false)
	if err != nil {
		return result, nil
	}

	return string(resultJSON), nil
}

// executeMongoCollectionCommand 执行 MongoDB 集合命令
func executeMongoCollectionCommand(identity string, commandStr string) (interface{}, error) {
	// 解析命令：dbName.collection.operation(args)
	// 例如：test.users.find({})

	// 查找第一个点，分离数据库名
	firstDot := strings.Index(commandStr, ".")
	if firstDot == -1 {
		return nil, fmt.Errorf("无效的命令格式")
	}

	dbName := commandStr[:firstDot]
	remaining := commandStr[firstDot+1:]

	// 查找第二个点，分离集合名
	secondDot := strings.Index(remaining, ".")
	if secondDot == -1 {
		return nil, fmt.Errorf("无效的命令格式")
	}

	collectionName := remaining[:secondDot]
	operationPart := remaining[secondDot+1:]

	// 连接到 MongoDB
	client, db, err := helper.GetMongoDBClient(identity, dbName)
	if err != nil {
		return nil, err
	}
	defer client.Disconnect(context.Background())

	collection := db.Collection(collectionName)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 解析操作类型
	if strings.HasPrefix(operationPart, "find(") {
		// find 操作
		filterStr := strings.TrimPrefix(operationPart, "find(")
		filterStr = strings.TrimSuffix(filterStr, ")")

		var filter bson.M
		if filterStr == "" || filterStr == "{}" {
			filter = bson.M{}
		} else {
			err = bson.UnmarshalExtJSON([]byte(filterStr), false, &filter)
			if err != nil {
				return nil, fmt.Errorf("解析过滤条件失败: %v", err)
			}
		}

		cursor, err := collection.Find(ctx, filter, options.Find().SetLimit(100))
		if err != nil {
			return nil, fmt.Errorf("查询失败: %v", err)
		}
		defer cursor.Close(ctx)

		var results []bson.M
		if err = cursor.All(ctx, &results); err != nil {
			return nil, fmt.Errorf("读取结果失败: %v", err)
		}

		// 转换为 JSON 字符串数组
		resultStrs := make([]string, len(results))
		for i, doc := range results {
			docJSON, err := bson.MarshalExtJSON(doc, false, false)
			if err != nil {
				resultStrs[i] = fmt.Sprintf("%v", doc)
			} else {
				resultStrs[i] = string(docJSON)
			}
		}

		return strings.Join(resultStrs, "\n"), nil

	} else if strings.HasPrefix(operationPart, "count(") {
		// count 操作
		count, err := collection.CountDocuments(ctx, bson.M{})
		if err != nil {
			return nil, fmt.Errorf("统计失败: %v", err)
		}
		return count, nil
	}

	return nil, fmt.Errorf("不支持的操作: %s", operationPart)
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
