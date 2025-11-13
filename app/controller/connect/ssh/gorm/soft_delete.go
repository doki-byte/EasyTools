package gorm

import (
	clause2 "EasyTools/app/controller/connect/ssh/gorm/clause"
	"EasyTools/app/controller/connect/ssh/gorm/now"
	schema2 "EasyTools/app/controller/connect/ssh/gorm/schema"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"reflect"
)

type DeletedAt sql.NullTime

// Scan implements the Scanner interface.
func (n *DeletedAt) Scan(value any) error {
	return (*sql.NullTime)(n).Scan(value)
}

// Value implements the driver Valuer interface.
func (n DeletedAt) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Time, nil
}

func (n DeletedAt) MarshalJSON() ([]byte, error) {
	if n.Valid {
		return json.Marshal(n.Time)
	}
	return json.Marshal(nil)
}

func (n *DeletedAt) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		n.Valid = false
		return nil
	}
	err := json.Unmarshal(b, &n.Time)
	if err == nil {
		n.Valid = true
	}
	return err
}

func (DeletedAt) QueryClauses(f *schema2.Field) []clause2.Interface {
	return []clause2.Interface{SoftDeleteQueryClause{Field: f, ZeroValue: parseZeroValueTag(f)}}
}

func parseZeroValueTag(f *schema2.Field) sql.NullString {
	if v, ok := f.TagSettings["ZEROVALUE"]; ok {
		if _, err := now.Parse(v); err == nil {
			return sql.NullString{String: v, Valid: true}
		}
	}
	return sql.NullString{Valid: false}
}

type SoftDeleteQueryClause struct {
	ZeroValue sql.NullString
	Field     *schema2.Field
}

func (sd SoftDeleteQueryClause) Name() string {
	return ""
}

func (sd SoftDeleteQueryClause) Build(clause2.Builder) {
}

func (sd SoftDeleteQueryClause) MergeClause(*clause2.Clause) {
}

func (sd SoftDeleteQueryClause) ModifyStatement(stmt *Statement) {
	if _, ok := stmt.Clauses["soft_delete_enabled"]; !ok && !stmt.Statement.Unscoped {
		if c, ok := stmt.Clauses["WHERE"]; ok {
			if where, ok := c.Expression.(clause2.Where); ok && len(where.Exprs) >= 1 {
				for _, expr := range where.Exprs {
					if orCond, ok := expr.(clause2.OrConditions); ok && len(orCond.Exprs) == 1 {
						where.Exprs = []clause2.Expression{clause2.And(where.Exprs...)}
						c.Expression = where
						stmt.Clauses["WHERE"] = c
						break
					}
				}
			}
		}

		stmt.AddClause(clause2.Where{Exprs: []clause2.Expression{
			clause2.Eq{Column: clause2.Column{Table: clause2.CurrentTable, Name: sd.Field.DBName}, Value: sd.ZeroValue},
		}})
		stmt.Clauses["soft_delete_enabled"] = clause2.Clause{}
	}
}

func (DeletedAt) UpdateClauses(f *schema2.Field) []clause2.Interface {
	return []clause2.Interface{SoftDeleteUpdateClause{Field: f, ZeroValue: parseZeroValueTag(f)}}
}

type SoftDeleteUpdateClause struct {
	ZeroValue sql.NullString
	Field     *schema2.Field
}

func (sd SoftDeleteUpdateClause) Name() string {
	return ""
}

func (sd SoftDeleteUpdateClause) Build(clause2.Builder) {
}

func (sd SoftDeleteUpdateClause) MergeClause(*clause2.Clause) {
}

func (sd SoftDeleteUpdateClause) ModifyStatement(stmt *Statement) {
	if stmt.SQL.Len() == 0 && !stmt.Statement.Unscoped {
		SoftDeleteQueryClause(sd).ModifyStatement(stmt)
	}
}

func (DeletedAt) DeleteClauses(f *schema2.Field) []clause2.Interface {
	return []clause2.Interface{SoftDeleteDeleteClause{Field: f, ZeroValue: parseZeroValueTag(f)}}
}

type SoftDeleteDeleteClause struct {
	ZeroValue sql.NullString
	Field     *schema2.Field
}

func (sd SoftDeleteDeleteClause) Name() string {
	return ""
}

func (sd SoftDeleteDeleteClause) Build(clause2.Builder) {
}

func (sd SoftDeleteDeleteClause) MergeClause(*clause2.Clause) {
}

func (sd SoftDeleteDeleteClause) ModifyStatement(stmt *Statement) {
	if stmt.SQL.Len() == 0 && !stmt.Statement.Unscoped {
		curTime := stmt.DB.NowFunc()
		stmt.AddClause(clause2.Set{{Column: clause2.Column{Name: sd.Field.DBName}, Value: curTime}})
		stmt.SetColumn(sd.Field.DBName, curTime, true)

		if stmt.Schema != nil {
			_, queryValues := schema2.GetIdentityFieldValuesMap(stmt.Context, stmt.ReflectValue, stmt.Schema.PrimaryFields)
			column, values := schema2.ToQueryValues(stmt.Table, stmt.Schema.PrimaryFieldDBNames, queryValues)

			if len(values) > 0 {
				stmt.AddClause(clause2.Where{Exprs: []clause2.Expression{clause2.IN{Column: column, Values: values}}})
			}

			if stmt.ReflectValue.CanAddr() && stmt.Dest != stmt.Model && stmt.Model != nil {
				_, queryValues = schema2.GetIdentityFieldValuesMap(stmt.Context, reflect.ValueOf(stmt.Model), stmt.Schema.PrimaryFields)
				column, values = schema2.ToQueryValues(stmt.Table, stmt.Schema.PrimaryFieldDBNames, queryValues)

				if len(values) > 0 {
					stmt.AddClause(clause2.Where{Exprs: []clause2.Expression{clause2.IN{Column: column, Values: values}}})
				}
			}
		}

		SoftDeleteQueryClause(sd).ModifyStatement(stmt)
		stmt.AddClauseIfNotExists(clause2.Update{})
		stmt.Build(stmt.DB.Callback().Update().Clauses...)
	}
}
