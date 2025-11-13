package service

import (
	"EasyTools/app/controller/connect/redis/internal/define"
	"EasyTools/app/controller/connect/redis/internal/helper"
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
)

// KeyList 键列表
func KeyList(req *define.KeyListRequest) ([]string, error) {
	rdb, err := helper.GetRedisClient(req.ConnIdentity, req.Db)
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

// GetKeyValue 获取键值对
func GetKeyValue(req *define.KeyValueRequest) (*define.KeyValueReply, error) {
	rdb, err := helper.GetRedisClient(req.ConnIdentity, req.Db)
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

// DeleteKeyValue 删除键值对
func DeleteKeyValue(req *define.KeyValueRequest) error {
	// 获取 Redis 客户端
	rdb, err := helper.GetRedisClient(req.ConnIdentity, req.Db)
	if err != nil {
		// 如果获取 Redis 客户端失败，返回错误
		return fmt.Errorf("failed to get Redis client: %v", err)
	}

	// 确保 Redis 客户端非 nil
	if rdb == nil {
		return fmt.Errorf("Redis client is nil for connection: %v, db: %d", req.ConnIdentity, req.Db)
	}

	// 删除键值对
	_, err = rdb.Del(context.Background(), req.Key).Result()
	if err != nil {
		// 如果删除操作失败，返回错误
		return fmt.Errorf("failed to delete key %s from Redis: %v", req.Key, err)
	}

	// 返回 nil 表示成功
	return nil
}

// CreateKeyValue 创建键值对
func CreateKeyValue(req *define.CreateKeyValueRequest) error {
	rdb, err := helper.GetRedisClient(req.ConnIdentity, req.Db)
	if req.Type == "string" {
		err = rdb.Set(context.Background(), req.Key, "value", -1).Err()
	} else if req.Type == "hash" {
		err = rdb.HSet(context.Background(), req.Key, map[string]string{"key": "value"}).Err()
	} else if req.Type == "list" {
		err = rdb.RPush(context.Background(), req.Key, "value").Err()
	} else if req.Type == "set" {
		err = rdb.SAdd(context.Background(), req.Key, "value").Err()
	} else if req.Type == "zset" {
		err = rdb.ZAdd(context.Background(), req.Key, &redis.Z{
			Score:  0,
			Member: "value",
		}).Err()
	}
	return err
}

// UpdateKeyValue 更新键值对数据
func UpdateKeyValue(req *define.UpdateKeyValueRequest) error {
	rdb, err := helper.GetRedisClient(req.ConnIdentity, req.Db)
	// type ==> string
	err = rdb.Set(context.Background(), req.Key, req.Value, req.TTL).Err()
	return err
}
