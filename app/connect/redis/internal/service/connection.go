package service

import (
	"EasyTools/app/connect/redis/internal/define"
	"EasyTools/app/connect/redis/internal/helper"
	"encoding/json"
	"errors"
	uuid "github.com/satori/go.uuid"
	"io/ioutil"
	"os"
)

// ConnectionList 连接列表
func ConnectionList() ([]*define.Connection, error) {
	nowPath := helper.GetConfPath()
	data, err := ioutil.ReadFile(nowPath + string(os.PathSeparator) + define.ConfigName)
	if errors.Is(err, os.ErrNotExist) {
		return nil, errors.New("暂无连接数据")
	}
	conf := new(define.Config)
	err = json.Unmarshal(data, conf)
	if err != nil {
		return nil, err
	}
	return conf.Connections, nil
}

// ConnectionCreate 创建连接
func ConnectionCreate(conn *define.Connection) error {
	if conn.Addr == "" {
		return errors.New("连接地址不能为空")
	}
	// 参数默认值处理
	if conn.Name == "" {
		conn.Name = conn.Addr
	}
	if conn.Port == "" {
		conn.Port = "6379"
	}
	conn.Identity = uuid.NewV4().String()
	conf := new(define.Config)
	nowPath := helper.GetConfPath()
	data, err := ioutil.ReadFile(nowPath + string(os.PathSeparator) + define.ConfigName)
	if errors.Is(err, os.ErrNotExist) {
		// 配置文件的内容初始化
		conf.Connections = []*define.Connection{conn}
		data, _ = json.Marshal(conf)
		// 写入配置内容
		os.MkdirAll(nowPath, 0666)
		ioutil.WriteFile(nowPath+string(os.PathSeparator)+define.ConfigName, data, 0666)
		return nil
	}
	json.Unmarshal(data, conf)
	conf.Connections = append(conf.Connections, conn)
	data, _ = json.Marshal(conf)
	ioutil.WriteFile(nowPath+string(os.PathSeparator)+define.ConfigName, data, 0666)
	return nil
}

// ConnectionEdit 编辑连接
func ConnectionEdit(conn *define.Connection) error {
	if conn.Identity == "" {
		return errors.New("连接唯一标识不能为空")
	}
	if conn.Addr == "" {
		return errors.New("连接地址不能为空")
	}
	// 参数默认值处理
	if conn.Name == "" {
		conn.Name = conn.Addr
	}
	if conn.Port == "" {
		conn.Port = "6379"
	}
	conf := new(define.Config)
	nowPath := helper.GetConfPath()
	data, err := ioutil.ReadFile(nowPath + string(os.PathSeparator) + define.ConfigName)
	if err != nil {
		return err
	}
	json.Unmarshal(data, conf)
	for i, v := range conf.Connections {
		if v.Identity == conn.Identity {
			conf.Connections[i] = conn
		}
	}
	data, _ = json.Marshal(conf)
	ioutil.WriteFile(nowPath+string(os.PathSeparator)+define.ConfigName, data, 0666)
	return nil
}

// ConnectionDelete 删除连接
func ConnectionDelete(identity string) error {
	if identity == "" {
		return errors.New("连接唯一标识不能为空")
	}
	conf := new(define.Config)
	nowPath := helper.GetConfPath()
	data, err := ioutil.ReadFile(nowPath + string(os.PathSeparator) + define.ConfigName)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, conf)
	if err != nil {
		return err
	}
	for i, v := range conf.Connections {
		if v.Identity == identity {
			conf.Connections = append(conf.Connections[:i], conf.Connections[i+1:]...)
			break
		}
	}
	data, _ = json.Marshal(conf)
	ioutil.WriteFile(nowPath+string(os.PathSeparator)+define.ConfigName, data, 0666)
	return nil
}
