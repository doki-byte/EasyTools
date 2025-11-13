package pgsql

import (
	gorm2 "EasyTools/app/controller/connect/ssh/gorm"
	"EasyTools/app/controller/connect/ssh/gorm/callbacks"
	clause2 "EasyTools/app/controller/connect/ssh/gorm/clause"
	"EasyTools/app/controller/connect/ssh/gorm/logger"
	"EasyTools/app/controller/connect/ssh/gorm/migrator"
	schema2 "EasyTools/app/controller/connect/ssh/gorm/schema"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Dialector struct {
	*Config
}

type Config struct {
	DriverName           string
	DSN                  string
	WithoutQuotingCheck  bool
	PreferSimpleProtocol bool
	WithoutReturning bool
	Conn             gorm2.ConnPool
}

var (
	timeZoneMatcher         = regexp.MustCompile("(time_zone|TimeZone)=(.*?)($|&| )")
	defaultIdentifierLength = 63 //maximum identifier length for postgres
)

func Open(dsn string) gorm2.Dialector {
	return &Dialector{&Config{DSN: dsn, DriverName: "postgres"}}
}

func New(config Config) gorm2.Dialector {
	return &Dialector{Config: &config}
}

func (dialector Dialector) Name() string {
	return "postgres"
}

func (dialector Dialector) Apply(config *gorm2.Config) error {
	if config.NamingStrategy == nil {
		config.NamingStrategy = schema2.NamingStrategy{
			IdentifierMaxLength: defaultIdentifierLength,
		}
		return nil
	}

	switch v := config.NamingStrategy.(type) {
	case *schema2.NamingStrategy:
		if v.IdentifierMaxLength <= 0 {
			v.IdentifierMaxLength = defaultIdentifierLength
		}
	case schema2.NamingStrategy:
		if v.IdentifierMaxLength <= 0 {
			v.IdentifierMaxLength = defaultIdentifierLength
			config.NamingStrategy = v
		}
	}

	return nil
}

func (dialector Dialector) Initialize(db *gorm2.DB) (err error) {
	callbackConfig := &callbacks.Config{
		CreateClauses: []string{"INSERT", "VALUES", "ON CONFLICT"},
		UpdateClauses: []string{"UPDATE", "SET", "FROM", "WHERE"},
		DeleteClauses: []string{"DELETE", "FROM", "WHERE"},
	}
	// register callbacks
	if !dialector.WithoutReturning {
		callbackConfig.CreateClauses = append(callbackConfig.CreateClauses, "RETURNING")
		callbackConfig.UpdateClauses = append(callbackConfig.UpdateClauses, "RETURNING")
		callbackConfig.DeleteClauses = append(callbackConfig.DeleteClauses, "RETURNING")
	}
	callbacks.RegisterDefaultCallbacks(db, callbackConfig)

	if dialector.Conn != nil {
		db.ConnPool = dialector.Conn
	} else /*dialector.DriverName != ""*/ {
		db.ConnPool, err = sql.Open(dialector.DriverName, dialector.Config.DSN)
	}
	return
}

func (dialector Dialector) Migrator(db *gorm2.DB) gorm2.Migrator {
	return Migrator{migrator.Migrator{Config: migrator.Config{
		DB:                          db,
		Dialector:                   dialector,
		CreateIndexAfterCreateTable: true,
	}}}
}

func (dialector Dialector) DefaultValueOf(field *schema2.Field) clause2.Expression {
	return clause2.Expr{SQL: "DEFAULT"}
}

func (dialector Dialector) BindVarTo(writer clause2.Writer, stmt *gorm2.Statement, v any) {
	writer.WriteByte('$')
	index := 0
	varLen := len(stmt.Vars)

	// huang
	//if varLen > 0 {
	//switch stmt.Vars[0].(type) {
	//
	//case pgx.QueryExecMode:
	//index++
	//}
	//}
	writer.WriteString(strconv.Itoa(varLen - index))
}

func (dialector Dialector) QuoteTo(writer clause2.Writer, str string) {
	if dialector.WithoutQuotingCheck {
		writer.WriteString(str)
		return
	}

	var (
		underQuoted, selfQuoted bool
		continuousBacktick      int8
		shiftDelimiter          int8
	)

	for _, v := range []byte(str) {
		switch v {
		case '"':
			continuousBacktick++
			if continuousBacktick == 2 {
				writer.WriteString(`""`)
				continuousBacktick = 0
			}
		case '.':
			if continuousBacktick > 0 || !selfQuoted {
				shiftDelimiter = 0
				underQuoted = false
				continuousBacktick = 0
				writer.WriteByte('"')
			}
			writer.WriteByte(v)
			continue
		default:
			if shiftDelimiter-continuousBacktick <= 0 && !underQuoted {
				writer.WriteByte('"')
				underQuoted = true
				if selfQuoted = continuousBacktick > 0; selfQuoted {
					continuousBacktick -= 1
				}
			}

			for ; continuousBacktick > 0; continuousBacktick -= 1 {
				writer.WriteString(`""`)
			}

			writer.WriteByte(v)
		}
		shiftDelimiter++
	}

	if continuousBacktick > 0 && !selfQuoted {
		writer.WriteString(`""`)
	}
	writer.WriteByte('"')
}

var numericPlaceholder = regexp.MustCompile(`\$(\d+)`)

func (dialector Dialector) Explain(sql string, vars ...any) string {
	return logger.ExplainSQL(sql, numericPlaceholder, `'`, vars...)
}

func (dialector Dialector) DataTypeOf(field *schema2.Field) string {
	switch field.DataType {
	case schema2.Bool:
		return "boolean"
	case schema2.Int, schema2.Uint:
		size := field.Size
		if field.DataType == schema2.Uint {
			size++
		}
		if field.AutoIncrement {
			switch {
			case size <= 16:
				return "smallserial"
			case size <= 32:
				return "serial"
			default:
				return "bigserial"
			}
		} else {
			switch {
			case size <= 16:
				return "smallint"
			case size <= 32:
				return "integer"
			default:
				return "bigint"
			}
		}
	case schema2.Float:
		if field.Precision > 0 {
			if field.Scale > 0 {
				return fmt.Sprintf("numeric(%d, %d)", field.Precision, field.Scale)
			}
			return fmt.Sprintf("numeric(%d)", field.Precision)
		}
		return "decimal"
	case schema2.String:
		if field.Size > 0 {
			return fmt.Sprintf("varchar(%d)", field.Size)
		}
		return "text"
	case schema2.Time:
		if field.Precision > 0 {
			return fmt.Sprintf("timestamptz(%d)", field.Precision)
		}
		return "timestamptz"
	case schema2.Bytes:
		return "bytea"
	default:
		return dialector.getSchemaCustomType(field)
	}
}

func (dialector Dialector) getSchemaCustomType(field *schema2.Field) string {
	sqlType := string(field.DataType)

	if field.AutoIncrement && !strings.Contains(strings.ToLower(sqlType), "serial") {
		size := field.Size
		if field.GORMDataType == schema2.Uint {
			size++
		}
		switch {
		case size <= 16:
			sqlType = "smallserial"
		case size <= 32:
			sqlType = "serial"
		default:
			sqlType = "bigserial"
		}
	}

	return sqlType
}

func (dialector Dialector) SavePoint(tx *gorm2.DB, name string) error {
	tx.Exec("SAVEPOINT " + name)
	return nil
}

func (dialector Dialector) RollbackTo(tx *gorm2.DB, name string) error {
	tx.Exec("ROLLBACK TO SAVEPOINT " + name)
	return nil
}

func getSerialDatabaseType(s string) (dbType string, ok bool) {
	switch s {
	case "smallserial":
		return "smallint", true
	case "serial":
		return "integer", true
	case "bigserial":
		return "bigint", true
	default:
		return "", false
	}
}
