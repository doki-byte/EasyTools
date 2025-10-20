package define

import "time"

type Connection struct {
	Identity string `json:"identity"`
	Name     string `json:"name"`
	Addr     string `json:"addr"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Type     string `json:"type"` // 连接类型
	SSHConfig
}

type SSHConfig struct {
	SSHAddr     string `json:"ssh_addr"`
	SSHPort     string `json:"ssh_port"`
	SSHUsername string `json:"ssh_username"`
	SSHPassword string `json:"ssh_password"`
}

// CommandResult 命令执行结果
type CommandResult struct {
	Success  bool        `json:"success"`  // 是否成功
	Data     interface{} `json:"data"`     // 执行结果数据
	Error    string      `json:"error"`    // 错误信息
	Duration int64       `json:"duration"` // 执行耗时(毫秒)
}

// CommandHistory 命令历史记录
type CommandHistory struct {
	Command   string `json:"command"`   // 命令
	Success   bool   `json:"success"`   // 是否成功
	Timestamp int64  `json:"timestamp"` // 执行时间戳
}

var (
	// DefaultKeyLen 键列表的默认查询长度
	DefaultKeyLen int64 = 100
	// MaxKeyLen 键列表的最大查询长度
	MaxKeyLen int64 = 2000
	// MaxHashLen hash列表的最大查询长度
	MaxHashLen int64 = 200
	// MaxListLen list列表的最大查询长度
	MaxListLen int64 = 200
	// MaxSetLen set的最大查询长度
	MaxSetLen int64 = 200
	// MaxZSetLen zset的最大查询长度
	MaxZSetLen int64 = 200
)

type Config struct {
	Connections []*Connection `json:"connections"`
}

type DbItem struct {
	Key    string `json:"key"`    // db0 , db1
	Number int    `json:"number"` // 个数
}

type KeyListRequest struct {
	ConnIdentity string `json:"conn_identity"`
	Db           int    `json:"db"`
	Keyword      string `json:"keyword"`
}

type KeyValue struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

type KeyValueRequest struct {
	ConnIdentity string `json:"conn_identity"`
	Db           int    `json:"db"`
	Key          string `json:"key"`
}

type KeyValueReply struct {
	Value interface{}   `json:"value"`
	TTL   time.Duration `json:"ttl"`
	Type  string        `json:"type"`
}

type CreateKeyValueRequest struct {
	KeyValueRequest
	Type string `json:"type"`
}

type UpdateKeyValueRequest struct {
	KeyValueRequest
	TTL   time.Duration `json:"ttl"`
	Value string        `json:"value"`
}

type HashFieldDeleteRequest struct {
	KeyValueRequest
	Field []string `json:"field"`
}

type HashAddOrUpdateFieldRequest struct {
	KeyValueRequest
	Field string `json:"field"`
	Value string `json:"value"`
}

type ListValueRequest struct {
	KeyValueRequest
	Value string `json:"value"`
}

type SetValueRequest struct {
	KeyValueRequest
	Value string `json:"value"`
}

type ZSetValueRequest struct {
	KeyValueRequest
	Score  float64     `json:"score"`
	Member interface{} `json:"member"`
}
