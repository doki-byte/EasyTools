package service

import (
	"EasyTools/app/connect/redis/internal/define"
	"EasyTools/app/connect/redis/internal/helper"
	"context"
)

func SetValueDelete(req *define.SetValueRequest) error {
	rdb, err := helper.GetRedisClient(req.ConnIdentity, req.Db)
	err = rdb.SRem(context.Background(), req.Key, req.Value).Err()
	return err
}

func SetValueCreate(req *define.SetValueRequest) error {
	rdb, err := helper.GetRedisClient(req.ConnIdentity, req.Db)
	err = rdb.SAdd(context.Background(), req.Key, req.Value).Err()
	return err
}
