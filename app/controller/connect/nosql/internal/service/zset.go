package service

import (
	"EasyTools/app/controller/connect/nosql/internal/define"
	"EasyTools/app/controller/connect/nosql/internal/helper"
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

func ZSetValueDelete(req *define.ZSetValueRequest) error {
	// 将 Db 转换为 int
	dbInt, err := convertDbToInt(req.Db)
	if err != nil {
		return fmt.Errorf("Redis数据库编号必须是数字: %v", err)
	}

	rdb, err := helper.GetRedisClient(req.ConnIdentity, dbInt)
	if err != nil {
		return err
	}
	err = rdb.ZRem(context.Background(), req.Key, req.Member).Err()
	return err
}

func ZSetValueCreate(req *define.ZSetValueRequest) error {
	// 将 Db 转换为 int
	dbInt, err := convertDbToInt(req.Db)
	if err != nil {
		return fmt.Errorf("Redis数据库编号必须是数字: %v", err)
	}

	rdb, err := helper.GetRedisClient(req.ConnIdentity, dbInt)
	if err != nil {
		return err
	}
	err = rdb.ZAdd(context.Background(), req.Key, &redis.Z{
		Score:  req.Score,
		Member: req.Member,
	}).Err()
	return err
}
