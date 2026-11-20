package connectors

const (
	Text    = "text"
	Number  = "number"
	Primary = "primary"
)

type Column struct {
	Field        string `db:"field" json:"field"`
	Type         string `db:"type" json:"type"`
	Length       string `db:"length" json:"length"`
	Nullable     bool   `db:"nullable" json:"nullable"`
	Key          string `db:"key" json:"key"`
	Default      string `db:"default" json:"default"`
	Comment      string `db:"comment" json:"comment"`
	AutoIncrement bool  `db:"auto_increment" json:"auto_increment"` // 标识是否是自增列（IDENTITY/AUTO_INCREMENT）
}

type Query struct {
	Columns     []Column         `json:"columns"`
	TableName   string           `json:"table_name"`
	Rows        []map[string]any `json:"rows"`
	Total       int              `json:"total"`
	ElapsedTime float64          `json:"elapsed_time"`
	TotalPage   int              `json:"total_page"`
}

type Table struct {
	Name       string `json:"name,omitempty"`
	Rows       string `json:"rows,omitempty"`
	Collation  string `json:"collation,omitempty"`
	CreateTime string `json:"create_time,omitempty"`
	UpdateTime string `json:"update_time,omitempty"`
}

