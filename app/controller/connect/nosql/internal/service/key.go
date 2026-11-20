package service

import (
	"EasyTools/app/controller/connect/nosql/internal/define"
	"EasyTools/app/controller/connect/nosql/internal/helper"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// KeyList 键列表
func KeyList(req *define.KeyListRequest) ([]string, error) {
	// 获取连接信息，判断数据库类型
	conn, err := helper.GetConnection(req.ConnIdentity)
	if err != nil {
		return nil, err
	}

	dbType := conn.DbType
	if dbType == "" {
		dbType = "redis"
	}

	dbTypeLower := strings.ToLower(dbType)
	switch dbTypeLower {
	case "redis":
		return getRedisKeyList(req)
	case "mongodb", "mongo":
		return getMongoDBKeyList(req)
	default:
		return nil, fmt.Errorf("不支持的数据库类型: %s", dbType)
	}
}

// getRedisKeyList 获取 Redis 键列表
func getRedisKeyList(req *define.KeyListRequest) ([]string, error) {
	// 将 Db 转换为 int
	dbInt, err := convertDbToInt(req.Db)
	if err != nil {
		return nil, fmt.Errorf("Redis数据库编号必须是数字: %v", err)
	}

	rdb, err := helper.GetRedisClient(req.ConnIdentity, dbInt)
	if err != nil {
		return nil, err
	}
	var count = define.DefaultKeyLen
	if req.Keyword != "" {
		count = define.MaxKeyLen
	}
	res, _, err := rdb.Scan(context.Background(), 0, "*"+req.Keyword+"*", count).Result()
	if err != nil {
		return nil, err
	}
	return res, nil
}

// getMongoDBKeyList 获取 MongoDB 集合列表
func getMongoDBKeyList(req *define.KeyListRequest) ([]string, error) {
	// 将 Db 转换为 string
	dbStr, err := convertDbToString(req.Db)
	if err != nil {
		return nil, fmt.Errorf("MongoDB数据库名必须是字符串: %v", err)
	}

	client, db, err := helper.GetMongoDBClient(req.ConnIdentity, dbStr)
	if err != nil {
		return nil, err
	}
	defer client.Disconnect(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 获取集合列表
	collections, err := db.ListCollectionNames(ctx, map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("获取MongoDB集合列表失败: %v", err)
	}

	// 如果有关键词，进行过滤
	if req.Keyword != "" {
		filtered := make([]string, 0)
		keywordLower := strings.ToLower(req.Keyword)
		for _, col := range collections {
			if strings.Contains(strings.ToLower(col), keywordLower) {
				filtered = append(filtered, col)
			}
		}
		return filtered, nil
	}

	return collections, nil
}

// GetKeyValue 获取键值对
func GetKeyValue(req *define.KeyValueRequest) (*define.KeyValueReply, error) {
	// 获取连接信息，判断数据库类型
	conn, err := helper.GetConnection(req.ConnIdentity)
	if err != nil {
		return nil, err
	}

	dbType := conn.DbType
	if dbType == "" {
		dbType = "redis"
	}

	dbTypeLower := strings.ToLower(dbType)
	switch dbTypeLower {
	case "redis":
		return getRedisKeyValue(req)
	case "mongodb", "mongo":
		return getMongoDBKeyValue(req)
	default:
		return nil, fmt.Errorf("不支持的数据库类型: %s", dbType)
	}
}

// getRedisKeyValue 获取 Redis 键值对
func getRedisKeyValue(req *define.KeyValueRequest) (*define.KeyValueReply, error) {
	// 将 Db 转换为 int
	dbInt, err := convertDbToInt(req.Db)
	if err != nil {
		return nil, fmt.Errorf("Redis数据库编号必须是数字: %v", err)
	}

	rdb, err := helper.GetRedisClient(req.ConnIdentity, dbInt)
	if err != nil {
		return nil, err
	}

	_type, err := rdb.Type(context.Background(), req.Key).Result()
	if err != nil {
		return nil, err
	}
	var reply = &define.KeyValueReply{
		Type: _type,
	}
	if _type == "string" {
		v, err := rdb.Get(context.Background(), req.Key).Result()
		if err != nil {
			return nil, err
		}
		reply.Value = v
	} else if _type == "hash" {
		keys, _, err := rdb.HScan(context.Background(), req.Key, 0, "", define.MaxHashLen).Result()
		if err != nil {
			return nil, err
		}
		data := make([]*define.KeyValue, 0, len(keys)/2)
		for i := 0; i < len(keys); i += 2 {
			data = append(data, &define.KeyValue{
				Key:   keys[i],
				Value: keys[i+1],
			})
		}
		reply.Value = data
	} else if _type == "list" {
		list, err := rdb.LRange(context.Background(), req.Key, 0, define.MaxListLen-1).Result()
		if err != nil {
			return nil, err
		}
		data := make([]*define.KeyValue, 0, len(list))
		for i := 0; i < len(list); i++ {
			data = append(data, &define.KeyValue{
				Value: list[i],
			})
		}
		reply.Value = data
	} else if _type == "set" {
		sets, _, err := rdb.SScan(context.Background(), req.Key, 0, "", define.MaxSetLen).Result()
		if err != nil {
			return nil, err
		}
		data := make([]*define.KeyValue, 0, len(sets))
		for i := 0; i < len(sets); i++ {
			data = append(data, &define.KeyValue{
				Value: sets[i],
			})
		}
		reply.Value = data
	} else if _type == "zset" {
		z, err := rdb.ZRevRangeWithScores(context.Background(), req.Key, 0, define.MaxZSetLen-1).Result()
		if err != nil {
			return nil, err
		}
		reply.Value = z
	}

	ttl, err := rdb.TTL(context.Background(), req.Key).Result()
	if err != nil {
		return nil, err
	}
	reply.TTL = ttl
	return reply, nil
}

// getMongoDBKeyValue 获取 MongoDB 集合文档
func getMongoDBKeyValue(req *define.KeyValueRequest) (*define.KeyValueReply, error) {
	// 将 Db 转换为 string
	dbStr, err := convertDbToString(req.Db)
	if err != nil {
		return nil, fmt.Errorf("MongoDB数据库名必须是字符串: %v", err)
	}
	if dbStr == "" {
		return nil, fmt.Errorf("MongoDB数据库名不能为空")
	}

	// req.Key 是集合名称
	if req.Key == "" {
		return nil, fmt.Errorf("集合名称不能为空")
	}

	client, db, err := helper.GetMongoDBClient(req.ConnIdentity, dbStr)
	if err != nil {
		return nil, err
	}
	defer client.Disconnect(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	collection := db.Collection(req.Key)

	// 获取文档列表（限制数量）
	findOptions := options.Find()
	findOptions.SetLimit(int64(define.MaxListLen))
	findOptions.SetSort(bson.D{{Key: "_id", Value: 1}})

	cursor, err := collection.Find(ctx, bson.D{}, findOptions)
	if err != nil {
		return nil, fmt.Errorf("查询MongoDB集合失败: %v", err)
	}
	defer cursor.Close(ctx)

	// 读取所有文档
	var documents []bson.M
	if err = cursor.All(ctx, &documents); err != nil {
		return nil, fmt.Errorf("读取MongoDB文档失败: %v", err)
	}

	// 转换为 KeyValue 格式
	data := make([]*define.KeyValue, 0, len(documents))
	for _, doc := range documents {
		// 提取 _id 并转换为字符串（保持原始格式以便后续匹配）
		var idStr string
		if idVal, ok := doc["_id"]; ok {
			// 如果是ObjectID，使用Hex()方法获取标准格式
			if oid, ok := idVal.(primitive.ObjectID); ok {
				idStr = oid.Hex()
			} else {
				// 其他类型直接转换为字符串
				idStr = fmt.Sprintf("%v", idVal)
			}
		}
		
		// 将文档转换为 JSON 字符串
		docBytes, err := bson.MarshalExtJSON(doc, false, false)
		if err != nil {
			// 如果转换失败，使用字符串表示
			data = append(data, &define.KeyValue{
				Key:   idStr,
				Value: fmt.Sprintf("%v", doc),
			})
		} else {
			data = append(data, &define.KeyValue{
				Key:   idStr,
				Value: string(docBytes),
			})
		}
	}

	reply := &define.KeyValueReply{
		Type:  "collection", // MongoDB 集合类型
		Value: data,
		TTL:   0, // MongoDB 不支持 TTL（在这个上下文中）
	}

	return reply, nil
}

// DeleteKeyValue 删除键值对
func DeleteKeyValue(req *define.KeyValueRequest) error {
	// 获取连接信息，判断数据库类型
	conn, err := helper.GetConnection(req.ConnIdentity)
	if err != nil {
		return err
	}

	dbType := conn.DbType
	if dbType == "" {
		dbType = "redis"
	}

	dbTypeLower := strings.ToLower(dbType)
	switch dbTypeLower {
	case "redis":
		return deleteRedisKeyValue(req)
	case "mongodb", "mongo":
		return deleteMongoDBKeyValue(req)
	default:
		return fmt.Errorf("不支持的数据库类型: %s", dbType)
	}
}

// deleteRedisKeyValue 删除 Redis 键值对
func deleteRedisKeyValue(req *define.KeyValueRequest) error {
	// 将 Db 转换为 int
	dbInt, err := convertDbToInt(req.Db)
	if err != nil {
		return fmt.Errorf("Redis数据库编号必须是数字: %v", err)
	}

	rdb, err := helper.GetRedisClient(req.ConnIdentity, dbInt)
	if err != nil {
		return fmt.Errorf("failed to get Redis client: %v", err)
	}

	if rdb == nil {
		return fmt.Errorf("Redis client is nil for connection: %v, db: %d", req.ConnIdentity, dbInt)
	}

	_, err = rdb.Del(context.Background(), req.Key).Result()
	if err != nil {
		return fmt.Errorf("failed to delete key %s from Redis: %v", req.Key, err)
	}

	return nil
}

// deleteMongoDBKeyValue 删除 MongoDB 集合
func deleteMongoDBKeyValue(req *define.KeyValueRequest) error {
	// 将 Db 转换为 string
	dbStr, err := convertDbToString(req.Db)
	if err != nil {
		return fmt.Errorf("MongoDB数据库名必须是字符串: %v", err)
	}

	client, db, err := helper.GetMongoDBClient(req.ConnIdentity, dbStr)
	if err != nil {
		return err
	}
	defer client.Disconnect(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// req.Key 是集合名称，删除整个集合
	collection := db.Collection(req.Key)
	err = collection.Drop(ctx)
	if err != nil {
		return fmt.Errorf("删除MongoDB集合失败: %v", err)
	}

	return nil
}

// CreateKeyValue 创建键值对
func CreateKeyValue(req *define.CreateKeyValueRequest) error {
	// 获取连接信息，判断数据库类型
	conn, err := helper.GetConnection(req.ConnIdentity)
	if err != nil {
		return err
	}

	dbType := conn.DbType
	if dbType == "" {
		dbType = "redis"
	}

	dbTypeLower := strings.ToLower(dbType)
	switch dbTypeLower {
	case "redis":
		return createRedisKeyValue(req)
	case "mongodb", "mongo":
		return createMongoDBKeyValue(req)
	default:
		return fmt.Errorf("不支持的数据库类型: %s", dbType)
	}
}

// createRedisKeyValue 创建 Redis 键值对
func createRedisKeyValue(req *define.CreateKeyValueRequest) error {
	// 将 Db 转换为 int
	dbInt, err := convertDbToInt(req.Db)
	if err != nil {
		return fmt.Errorf("Redis数据库编号必须是数字: %v", err)
	}

	rdb, err := helper.GetRedisClient(req.ConnIdentity, dbInt)
	if err != nil {
		return fmt.Errorf("获取Redis客户端失败: %v", err)
	}

	if rdb == nil {
		return fmt.Errorf("Redis客户端为空，连接标识: %v, 数据库: %d", req.ConnIdentity, dbInt)
	}

	// 使用实际传入的值，如果没有提供则使用默认值
	value := req.Value
	if value == "" {
		value = "" // 空字符串作为默认值
	}

	ctx := context.Background()

	switch req.Type {
	case "string":
		// 字符串类型：直接设置值
		err = rdb.Set(ctx, req.Key, value, -1).Err()
		if err != nil {
			return fmt.Errorf("创建Redis字符串键失败: %v", err)
		}
	case "hash":
		// Hash类型：如果提供了值，尝试解析为JSON对象；否则创建空hash
		if value != "" {
			// 尝试解析JSON格式的hash字段
			var hashData map[string]string
			if err := json.Unmarshal([]byte(value), &hashData); err == nil {
				// 成功解析为JSON对象
				err = rdb.HSet(ctx, req.Key, hashData).Err()
			} else {
				// 不是JSON格式，使用单个字段
				err = rdb.HSet(ctx, req.Key, "field", value).Err()
			}
		} else {
			// 创建空的hash
			err = rdb.HSet(ctx, req.Key, "field", "").Err()
		}
		if err != nil {
			return fmt.Errorf("创建Redis Hash键失败: %v", err)
		}
	case "list":
		// List类型：将值添加到列表
		if value != "" {
			err = rdb.RPush(ctx, req.Key, value).Err()
		} else {
			// 创建空列表
			err = rdb.RPush(ctx, req.Key, "").Err()
		}
		if err != nil {
			return fmt.Errorf("创建Redis List键失败: %v", err)
		}
	case "set":
		// Set类型：将值添加到集合
		if value != "" {
			err = rdb.SAdd(ctx, req.Key, value).Err()
		} else {
			// 创建空集合
			err = rdb.SAdd(ctx, req.Key, "").Err()
		}
		if err != nil {
			return fmt.Errorf("创建Redis Set键失败: %v", err)
		}
	case "zset":
		// ZSet类型：需要解析score和member
		score := 0.0
		member := value
		if value != "" {
			// 尝试解析JSON格式：{"score": 0, "member": "value"}
			var zsetData struct {
				Score  float64 `json:"score"`
				Member string  `json:"member"`
			}
			if err := json.Unmarshal([]byte(value), &zsetData); err == nil {
				score = zsetData.Score
				member = zsetData.Member
			}
		}
		err = rdb.ZAdd(ctx, req.Key, &redis.Z{
			Score:  score,
			Member: member,
		}).Err()
		if err != nil {
			return fmt.Errorf("创建Redis ZSet键失败: %v", err)
		}
	default:
		return fmt.Errorf("不支持的Redis数据类型: %s", req.Type)
	}

	return nil
}

// createMongoDBKeyValue 创建 MongoDB 集合
func createMongoDBKeyValue(req *define.CreateKeyValueRequest) error {
	// 将 Db 转换为 string
	dbStr, err := convertDbToString(req.Db)
	if err != nil {
		return fmt.Errorf("MongoDB数据库名必须是字符串: %v", err)
	}

	client, db, err := helper.GetMongoDBClient(req.ConnIdentity, dbStr)
	if err != nil {
		return err
	}
	defer client.Disconnect(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// req.Key 是集合名称
	collection := db.Collection(req.Key)

	// req.Value 应该是 JSON 格式的文档内容
	var doc bson.M
	if req.Value == "" {
		// 如果没有提供内容，创建一个空文档
		doc = bson.M{"_created": time.Now()}
	} else {
		// 解析 JSON 为 BSON 文档
		err = bson.UnmarshalExtJSON([]byte(req.Value), false, &doc)
		if err != nil {
			return fmt.Errorf("解析文档 JSON 失败: %v", err)
		}
	}

	// 插入文档
	_, err = collection.InsertOne(ctx, doc)
	if err != nil {
		return fmt.Errorf("插入MongoDB文档失败: %v", err)
	}

	return nil
}

// UpdateKeyValue 更新键值对数据
func UpdateKeyValue(req *define.UpdateKeyValueRequest) error {
	// 获取连接信息，判断数据库类型
	conn, err := helper.GetConnection(req.ConnIdentity)
	if err != nil {
		return err
	}

	dbType := conn.DbType
	if dbType == "" {
		dbType = "redis"
	}

	dbTypeLower := strings.ToLower(dbType)
	switch dbTypeLower {
	case "redis":
		return updateRedisKeyValue(req)
	case "mongodb", "mongo":
		return updateMongoDBDocument(req)
	default:
		return fmt.Errorf("不支持的数据库类型: %s", dbType)
	}
}

// updateRedisKeyValue 更新 Redis 键值对
func updateRedisKeyValue(req *define.UpdateKeyValueRequest) error {
	// 将 Db 转换为 int
	dbInt, err := convertDbToInt(req.Db)
	if err != nil {
		return fmt.Errorf("Redis数据库编号必须是数字: %v", err)
	}

	rdb, err := helper.GetRedisClient(req.ConnIdentity, dbInt)
	if err != nil {
		return err
	}

	err = rdb.Set(context.Background(), req.Key, req.Value, req.TTL).Err()
	return err
}

// convertDbToInt 将 Db 转换为 int
func convertDbToInt(db interface{}) (int, error) {
	switch v := db.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		// 尝试解析字符串为数字
		return strconv.Atoi(v)
	default:
		return 0, fmt.Errorf("无法转换为int: %T", db)
	}
}

// convertDbToString 将 Db 转换为 string
func convertDbToString(db interface{}) (string, error) {
	switch v := db.(type) {
	case string:
		return v, nil
	case int, int64, float64:
		// 如果是数字，转换为字符串（虽然 MongoDB 不应该使用数字）
		return fmt.Sprintf("%v", v), nil
	default:
		return "", fmt.Errorf("无法转换为string: %T", db)
	}
}

// updateMongoDBDocument 更新 MongoDB 文档
func updateMongoDBDocument(req *define.UpdateKeyValueRequest) error {
	// 将 Db 转换为 string
	dbStr, err := convertDbToString(req.Db)
	if err != nil {
		return fmt.Errorf("MongoDB数据库名必须是字符串: %v", err)
	}

	if dbStr == "" {
		return fmt.Errorf("MongoDB数据库名不能为空")
	}

	client, db, err := helper.GetMongoDBClient(req.ConnIdentity, dbStr)
	if err != nil {
		return fmt.Errorf("获取MongoDB客户端失败: %v", err)
	}
	defer client.Disconnect(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// req.Key 是集合名称
	if req.Key == "" {
		return fmt.Errorf("集合名称不能为空")
	}
	collection := db.Collection(req.Key)

	// req.OldValue 应该包含文档的 _id
	if req.OldValue == "" {
		return fmt.Errorf("文档 _id 不能为空")
	}

	// 解析新文档内容
	var newDoc bson.M
	if req.Value == "" {
		return fmt.Errorf("更新内容不能为空")
	}
	
	err = bson.UnmarshalExtJSON([]byte(req.Value), false, &newDoc)
	if err != nil {
		return fmt.Errorf("解析文档 JSON 失败: %v", err)
	}

	// 构建查询过滤器，支持多种_id格式
	var filter bson.M
	// 尝试将 _id 转换为 ObjectID（如果是标准的 ObjectID 格式）
	if oid, err := primitive.ObjectIDFromHex(req.OldValue); err == nil {
		filter = bson.M{"_id": oid}
	} else {
		// 尝试其他可能的格式
		// 1. 尝试作为数字
		if intVal, err := strconv.Atoi(req.OldValue); err == nil {
			filter = bson.M{"_id": intVal}
		} else if int64Val, err := strconv.ParseInt(req.OldValue, 10, 64); err == nil {
			filter = bson.M{"_id": int64Val}
		} else {
			// 2. 作为字符串
			filter = bson.M{"_id": req.OldValue}
		}
	}

	// 移除新文档中的 _id 字段（如果存在），因为 _id 不能被更新
	delete(newDoc, "_id")

	// 检查是否有字段需要更新
	if len(newDoc) == 0 {
		return fmt.Errorf("没有可更新的字段")
	}

	// 更新文档（使用$set操作符，只更新提供的字段）
	update := bson.M{"$set": newDoc}
	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("更新MongoDB文档失败: %v", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("未找到要更新的文档 (ID: %s)", req.OldValue)
	}

	if result.ModifiedCount == 0 {
		// 文档匹配但未修改（可能是值相同）
		return nil
	}

	return nil
}

// DeleteMongoDBDocument 删除 MongoDB 文档
func DeleteMongoDBDocument(req *define.KeyValueRequest, docID string) error {
	// 将 Db 转换为 string
	dbStr, err := convertDbToString(req.Db)
	if err != nil {
		return fmt.Errorf("MongoDB数据库名必须是字符串: %v", err)
	}

	if dbStr == "" {
		return fmt.Errorf("MongoDB数据库名不能为空")
	}

	client, db, err := helper.GetMongoDBClient(req.ConnIdentity, dbStr)
	if err != nil {
		return fmt.Errorf("获取MongoDB客户端失败: %v", err)
	}
	defer client.Disconnect(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// req.Key 是集合名称
	if req.Key == "" {
		return fmt.Errorf("集合名称不能为空")
	}
	collection := db.Collection(req.Key)

	if docID == "" {
		return fmt.Errorf("文档 _id 不能为空")
	}

	// 构建查询过滤器，支持多种_id格式
	var filter bson.M
	// 尝试将 _id 转换为 ObjectID（如果是标准的 ObjectID 格式）
	if oid, err := primitive.ObjectIDFromHex(docID); err == nil {
		// 成功转换为ObjectID
		filter = bson.M{"_id": oid}
	} else {
		// 尝试其他可能的格式
		// 1. 尝试作为数字
		if intVal, err := strconv.Atoi(docID); err == nil {
			filter = bson.M{"_id": intVal}
		} else if int64Val, err := strconv.ParseInt(docID, 10, 64); err == nil {
			filter = bson.M{"_id": int64Val}
		} else {
			// 2. 作为字符串
			filter = bson.M{"_id": docID}
		}
	}

	// 删除文档
	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("删除MongoDB文档失败: %v", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("未找到要删除的文档 (ID: %s)", docID)
	}

	return nil
}
