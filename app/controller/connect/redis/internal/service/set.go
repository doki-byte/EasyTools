package service

import (
	"EasyTools/app/controller/connect/redis/internal/define"
	"EasyTools/app/controller/connect/redis/internal/helper"
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
