package restmate

type JSResp struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
	Data    any    `json:"data,omitempty"`
}

type KeyValue struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Value  string `json:"value"`
	Active bool   `json:"active"`
}
type FormData struct {
	ID     string   `json:"id"`
	Key    string   `json:"key"`
	Value  string   `json:"value"`
	Files  []string `json:"files"`
	Type   string   `json:"type"`
	Active bool     `json:"active"`
}
type Body struct {
	BodyType string     `json:"bodyType"`
	BodyRaw  string     `json:"bodyRaw"`
	FormData []FormData `json:"formData"`
}
type Request struct {
	ID      string     `json:"id"`
	Name    string     `json:"name"`
	Url     string     `json:"url"`
	Method  string     `json:"method"`
	Headers []KeyValue `json:"headers"`
	Params  []KeyValue `json:"params"`
	Body    Body       `json:"body"`
	CollId  string     `json:"coll_id"`
}

type Collection struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Schema   string    `json:"schema"`
	Requests []Request `json:"requests"`
}

type ExportCollection struct {
	Info struct {
		Schema string `json:"schema"`
		Name   string `json:"name"`
	} `json:"info"`
	Collection Collection `json:"collection"`
}
type KV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
type Result struct {
	StatusCode     int    `json:"statusCode"`
	HttpStatus     string `json:"httpStatus"`
	BodyContent    string `json:"bodyContent"`
	ErrorContent   string `json:"errorContent"`
	ContentType    string `json:"contentType"`
	Duration       string `json:"duration"`
	Headers        []KV   `json:"headers"`
	RequestPacket  string `json:"requestPacket,omitempty"`
	ResponsePacket string `json:"responsePacket,omitempty"`
	IsBinary       bool   `json:"isBinary,omitempty"`
	BinaryData     string `json:"binaryData,omitempty"`     // Base64 编码的二进制数据
	SuggestedExt   string `json:"suggestedExt,omitempty"`   // 建议的文件扩展名
}
type ReqRsp struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Url    string `json:"url"`
	Method string `json:"method"`
	CollId string `json:"coll_id"`
}
type CollRsp struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Requests []ReqRsp `json:"requests"`
}

type Env struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Selected bool              `json:"selected"`
	Variable map[string]string `json:"variable"`
}
type PMCollection struct {
	Info struct {
		Schema string `json:"schema"`
		Name   string `json:"name"`
	} `json:"info"`
	Item []Item `json:"item"`
}
type Item struct {
	Name    string `json:"name"`
	Item    []Item `json:"item,omitempty"`
	Request struct {
		Method string `json:"method"`
		Header []KV   `json:"header"`
		URL    struct {
			Raw string `json:"raw"`
		} `json:"url"`
		Body struct {
			Mode     string `json:"mode"`
			FormData []struct {
				KV
				Type string `json:"type"`
			} `json:"formdata"`
			Raw     string `json:"raw"`
			Options struct {
				Raw struct {
					Language string `json:"language"`
				} `json:"raw"`
			} `json:"options"`
		} `json:"body"`
	} `json:"request,omitempty"`
}

// BatchConfig 批量发包配置
type BatchConfig struct {
	Concurrency int `json:"concurrency"` // 并发数（每秒请求数）
	RetryCount  int `json:"retryCount"`  // 重试次数
}
