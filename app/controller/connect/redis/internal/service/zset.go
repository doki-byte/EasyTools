package service

import (
	"EasyTools/app/controller/connect/redis/internal/define"
	"EasyTools/app/controller/connect/redis/internal/helper"
	"context"
	"github.com/go-redis/redis/v8"
)

func ZSetValueDelete(req *define.ZSetValueRequest) error {
	rdb, err := helper.GetRedisClient(req.ConnIdentity, req.Db)
	err = rdb.ZRem(context.Background(), req.Key, req.Member).Err()
	return err
}

func ZSetValueCreate(req *define.ZSetValueRequest) error {
	rdb, err := helper.GetRedisClient(req.ConnIdentity, req.Db)
	err = rdb.ZAdd(context.Background(), req.Key, &redis.Z{
		Score:  req.Score,
		Member: req.Member,
	}).Err()
	return err
}
