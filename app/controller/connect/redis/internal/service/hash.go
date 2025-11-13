package service

import (
	"EasyTools/app/controller/connect/redis/internal/define"
	"EasyTools/app/controller/connect/redis/internal/helper"
	"context"
	"fmt"
)

// HashFieldDelete hash 字段删除
func HashFieldDelete(req *define.HashFieldDeleteRequest) error {
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

	// 执行删除字段操作
	err = rdb.HDel(context.Background(), req.Key, req.Field...).Err()
	if err != nil {
		// 如果删除字段失败，返回错误
		return fmt.Errorf("failed to delete field(s) from hash %s: %v", req.Key, err)
	}

	// 返回 nil 表示成功
	return nil
}

// HashAddOrUpdateField hash 字段新增、修改
func HashAddOrUpdateField(req *define.HashAddOrUpdateFieldRequest) error {
	rdb, err := helper.GetRedisClient(req.ConnIdentity, req.Db)
	err = rdb.HSet(context.Background(), req.Key, map[string]interface{}{req.Field: req.Value}).Err()
	return err
}
