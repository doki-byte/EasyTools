package redis

import (
	"EasyTools/app/connect/redis/internal/define"
	"EasyTools/app/connect/redis/internal/service"
	"context"
)

type Redis struct {
	ctx context.Context
}

func NewRedis() *Redis {
	return &Redis{}
}

// SetCtx 设置上下文对象
func (b *Redis) SetCtx(ctx context.Context) {
	b.ctx = ctx
}

// ConnectionList 连接列表
func (b *Redis) ConnectionList() H {
	conn, err := service.ConnectionList()
	if err != nil {
		return M{
			"code": -1,
			"msg":  "ERROR : " + err.Error(),
		}
	}
	return M{
		"code": 200,
		"data": conn,
	}
}

// ConnectionCreate 新建连接
func (b *Redis) ConnectionCreate(connection *define.Connection) H {
	err := service.ConnectionCreate(connection)
	if err != nil {
		return M{
			"code": -1,
			"msg":  "ERROR : " + err.Error(),
		}
	}
	return M{
		"code": 200,
		"msg":  "新建成功",
	}
}

// ConnectionEdit 修改连接
func (b *Redis) ConnectionEdit(connection *define.Connection) H {
	err := service.ConnectionEdit(connection)
	if err != nil {
		return M{
			"code": -1,
			"msg":  "ERROR : " + err.Error(),
		}
	}
	return M{
		"code": 200,
		"msg":  "编辑成功",
	}
}

// ConnectionDelete 删除连接
func (b *Redis) ConnectionDelete(identity string) H {
	err := service.ConnectionDelete(identity)
	if err != nil {
		return M{
			"code": -1,
			"msg":  "ERROR : " + err.Error(),
		}
	}
	return M{
		"code": 200,
		"msg":  "删除成功",
	}
}

// DbInfo 数据库详情
func (b *Redis) DbInfo(identity string) H {
	info, err := service.DbInfo(identity)
	if err != nil {
		return M{
			"code": -1,
			"msg":  "ERROR : " + err.Error(),
		}
	}
	return M{
		"code": 200,
		"data": info,
	}
}

// DbList 数据库列表
func (b *Redis) DbList(identity string) H {
	dbs, err := service.DbList(identity)
	if err != nil {
		return M{
			"code": -1,
			"msg":  "ERROR : " + err.Error(),
		}
	}
	return M{
		"code": 200,
		"data": dbs,
	}
}

// KeyList 键列表
func (b *Redis) KeyList(req *define.KeyListRequest) H {
	if req.ConnIdentity == "" {
		return M{
			"code": -1,
			"msg":  "连接的唯一标识不能为空",
		}
	}
	data, err := service.KeyList(req)
	if err != nil {
		return M{
			"code": -1,
			"msg":  "ERROR : " + err.Error(),
		}
	}
	return M{
		"code": 200,
		"data": data,
	}
}

// GetKeyValue 键值对查询
func (b *Redis) GetKeyValue(req *define.KeyValueRequest) H {
	if req.Key == "" || req.ConnIdentity == "" {
		return M{
			"code": -1,
			"msg":  "必填参不能为空",
		}
	}
	data, err := service.GetKeyValue(req)
	if err != nil {
		return M{
			"code": -1,
			"msg":  "ERROR : " + err.Error(),
		}
	}
	return M{
		"code": 200,
		"data": data,
	}
}

// DeleteKeyValue 键值对删除
func (b *Redis) DeleteKeyValue(req *define.KeyValueRequest) H {
	if req.Key == "" || req.ConnIdentity == "" {
		return M{
			"code": -1,
			"msg":  "必填参不能为空",
		}
	}
	err := service.DeleteKeyValue(req)
	if err != nil {
		return M{
			"code": -1,
			"msg":  "ERROR : " + err.Error(),
		}
	}
	return M{
		"code": 200,
		"msg":  "删除成功",
	}
}

// CreateKeyValue 键值对新增
func (b *Redis) CreateKeyValue(req *define.CreateKeyValueRequest) H {
	if req.Key == "" || req.ConnIdentity == "" || req.Type == "" {
		return M{
			"code": -1,
			"msg":  "必填参不能为空",
		}
	}
	err := service.CreateKeyValue(req)
	if err != nil {
		return M{
			"code": -1,
			"msg":  "ERROR : " + err.Error(),
		}
	}
	return M{
		"code": 200,
		"msg":  "新增成功",
	}
}

// UpdateKeyValue 键值对更新
func (b *Redis) UpdateKeyValue(req *define.UpdateKeyValueRequest) H {
	if req.Key == "" || req.ConnIdentity == "" || req.Value == "" {
		return M{
			"code": -1,
			"msg":  "必填参不能为空",
		}
	}
	err := service.UpdateKeyValue(req)
	if err != nil {
		return M{
			"code": -1,
			"msg":  "ERROR : " + err.Error(),
		}
	}
	return M{
		"code": 200,
		"msg":  "更新成功",
	}
}

// HashFieldDelete hash字段删除
func (b *Redis) HashFieldDelete(req *define.HashFieldDeleteRequest) H {
	if req.Key == "" || req.ConnIdentity == "" || len(req.Field) == 0 {
		return M{
			"code": -1,
			"msg":  "必填参不能为空",
		}
	}
	err := service.HashFieldDelete(req)
	if err != nil {
		return M{
			"code": -1,
			"msg":  "ERROR : " + err.Error(),
		}
	}
	return M{
		"code": 200,
		"msg":  "删除成功",
	}
}

// HashAddOrUpdateField hash字段新增、更新
func (b *Redis) HashAddOrUpdateField(req *define.HashAddOrUpdateFieldRequest) H {
	if req.Key == "" || req.ConnIdentity == "" || req.Field == "" {
		return M{
			"code": -1,
			"msg":  "必填参不能为空",
		}
	}
	err := service.HashAddOrUpdateField(req)
	if err != nil {
		return M{
			"code": -1,
			"msg":  "ERROR : " + err.Error(),
		}
	}
	return M{
		"code": 200,
		"msg":  "修改成功",
	}
}

// ListValueDelete 列表值删除
func (b *Redis) ListValueDelete(req *define.ListValueRequest) H {
	if req.Key == "" || req.ConnIdentity == "" || req.Value == "" {
		return M{
			"code": -1,
			"msg":  "必填参不能为空",
		}
	}
	err := service.ListValueDelete(req)
	if err != nil {
		return M{
			"code": -1,
			"msg":  "ERROR : " + err.Error(),
		}
	}
	return M{
		"code": 200,
		"msg":  "删除成功",
	}
}

// ListValueCreate 列表值新增
func (b *Redis) ListValueCreate(req *define.ListValueRequest) H {
	if req.Key == "" || req.ConnIdentity == "" || req.Value == "" {
		return M{
			"code": -1,
			"msg":  "必填参不能为空",
		}
	}
	err := service.ListValueCreate(req)
	if err != nil {
		return M{
			"code": -1,
			"msg":  "ERROR : " + err.Error(),
		}
	}
	return M{
		"code": 200,
		"msg":  "创建成功",
	}
}

// SetValueDelete 集合值删除
func (b *Redis) SetValueDelete(req *define.SetValueRequest) H {
	if req.Key == "" || req.ConnIdentity == "" || req.Value == "" {
		return M{
			"code": -1,
			"msg":  "必填参不能为空",
		}
	}
	err := service.SetValueDelete(req)
	if err != nil {
		return M{
			"code": -1,
			"msg":  "ERROR : " + err.Error(),
		}
	}
	return M{
		"code": 200,
		"msg":  "删除成功",
	}
}

// SetValueCreate 集合新增
func (b *Redis) SetValueCreate(req *define.SetValueRequest) H {
	if req.Key == "" || req.ConnIdentity == "" || req.Value == "" {
		return M{
			"code": -1,
			"msg":  "必填参不能为空",
		}
	}
	err := service.SetValueCreate(req)
	if err != nil {
		return M{
			"code": -1,
			"msg":  "ERROR : " + err.Error(),
		}
	}
	return M{
		"code": 200,
		"msg":  "创建成功",
	}
}

// ZSetValueDelete 有序集合值删除
func (b *Redis) ZSetValueDelete(req *define.ZSetValueRequest) H {
	if req.Key == "" || req.ConnIdentity == "" || req.Member == nil {
		return M{
			"code": -1,
			"msg":  "必填参不能为空",
		}
	}
	err := service.ZSetValueDelete(req)
	if err != nil {
		return M{
			"code": -1,
			"msg":  "ERROR : " + err.Error(),
		}
	}
	return M{
		"code": 200,
		"msg":  "删除成功",
	}
}

// ZSetValueCreate 有序集合新增
func (b *Redis) ZSetValueCreate(req *define.ZSetValueRequest) H {
	if req.Key == "" || req.ConnIdentity == "" || req.Member == nil {
		return M{
			"code": -1,
			"msg":  "必填参不能为空",
		}
	}
	err := service.ZSetValueCreate(req)
	if err != nil {
		return M{
			"code": -1,
			"msg":  "ERROR : " + err.Error(),
		}
	}
	return M{
		"code": 200,
		"msg":  "创建成功",
	}
}
