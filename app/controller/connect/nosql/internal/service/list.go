package service

import (
	"EasyTools/app/controller/connect/nosql/internal/define"
	"EasyTools/app/controller/connect/nosql/internal/helper"
	"context"
	"fmt"
)

func ListValueDelete(req *define.ListValueRequest) error {
	// 将 Db 转换为 int
	dbInt, err := convertDbToInt(req.Db)
	if err != nil {
		return fmt.Errorf("Redis数据库编号必须是数字: %v", err)
	}

	rdb, err := helper.GetRedisClient(req.ConnIdentity, dbInt)
	if err != nil {
		return err
	}
	err = rdb.LRem(context.Background(), req.Key, 1, req.Value).Err()
	return err
}

func ListValueCreate(req *define.ListValueRequest) error {
	// 将 Db 转换为 int
	dbInt, err := convertDbToInt(req.Db)
	if err != nil {
		return fmt.Errorf("Redis数据库编号必须是数字: %v", err)
	}

	rdb, err := helper.GetRedisClient(req.ConnIdentity, dbInt)
	if err != nil {
		return err
	}
	err = rdb.RPush(context.Background(), req.Key, req.Value).Err()
	return err
}
