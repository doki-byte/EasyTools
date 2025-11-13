package tests

import (
	gorm2 "EasyTools/app/controller/connect/ssh/gorm"
	"EasyTools/app/controller/connect/ssh/gorm/callbacks"
	clause2 "EasyTools/app/controller/connect/ssh/gorm/clause"
	"EasyTools/app/controller/connect/ssh/gorm/logger"
	"EasyTools/app/controller/connect/ssh/gorm/schema"
)

type DummyDialector struct {
	TranslatedErr error
}

func (DummyDialector) Name() string {
	return "dummy"
}

func (DummyDialector) Initialize(db *gorm2.DB) error {
	callbacks.RegisterDefaultCallbacks(db, &callbacks.Config{
		CreateClauses:        []string{"INSERT", "VALUES", "ON CONFLICT", "RETURNING"},
		UpdateClauses:        []string{"UPDATE", "SET", "WHERE", "RETURNING"},
		DeleteClauses:        []string{"DELETE", "FROM", "WHERE", "RETURNING"},
		LastInsertIDReversed: true,
	})

	return nil
}

func (DummyDialector) DefaultValueOf(field *schema.Field) clause2.Expression {
	return clause2.Expr{SQL: "DEFAULT"}
}

func (DummyDialector) Migrator(*gorm2.DB) gorm2.Migrator {
	return nil
}

func (DummyDialector) BindVarTo(writer clause2.Writer, stmt *gorm2.Statement, v any) {
	writer.WriteByte('?')
}

func (DummyDialector) QuoteTo(writer clause2.Writer, str string) {
	var (
		underQuoted, selfQuoted bool
		continuousBacktick      int8
		shiftDelimiter          int8
	)

	for _, v := range []byte(str) {
		switch v {
		case '`':
			continuousBacktick++
			if continuousBacktick == 2 {
				writer.WriteString("``")
				continuousBacktick = 0
			}
		case '.':
			if continuousBacktick > 0 || !selfQuoted {
				shiftDelimiter = 0
				underQuoted = false
				continuousBacktick = 0
				writer.WriteByte('`')
			}
			writer.WriteByte(v)
			continue
		default:
			if shiftDelimiter-continuousBacktick <= 0 && !underQuoted {
				writer.WriteByte('`')
				underQuoted = true
				if selfQuoted = continuousBacktick > 0; selfQuoted {
					continuousBacktick -= 1
				}
			}

			for ; continuousBacktick > 0; continuousBacktick -= 1 {
				writer.WriteString("``")
			}

			writer.WriteByte(v)
		}
		shiftDelimiter++
	}

	if continuousBacktick > 0 && !selfQuoted {
		writer.WriteString("``")
	}
	writer.WriteByte('`')
}

func (DummyDialector) Explain(sql string, vars ...any) string {
	return logger.ExplainSQL(sql, nil, `"`, vars...)
}

func (DummyDialector) DataTypeOf(*schema.Field) string {
	return ""
}

func (d DummyDialector) Translate(err error) error {
	return d.TranslatedErr
}
