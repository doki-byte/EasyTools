package service

import (
	"EasyTools/app/connect/redis/internal/define"
	"EasyTools/app/connect/redis/internal/helper"
	"context"
)

func ListValueDelete(req *define.ListValueRequest) error {
	rdb, err := helper.GetRedisClient(req.ConnIdentity, req.Db)
	err = rdb.LRem(context.Background(), req.Key, 1, req.Value).Err()
	return err
}

func ListValueCreate(req *define.ListValueRequest) error {
	rdb, err := helper.GetRedisClient(req.ConnIdentity, req.Db)
	err = rdb.RPush(context.Background(), req.Key, req.Value).Err()
	return err
}
