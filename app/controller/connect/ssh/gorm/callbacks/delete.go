package callbacks

import (
	gorm2 "EasyTools/app/controller/connect/ssh/gorm"
	clause2 "EasyTools/app/controller/connect/ssh/gorm/clause"
	schema2 "EasyTools/app/controller/connect/ssh/gorm/schema"
	"EasyTools/app/controller/connect/ssh/gorm/utils"
	"reflect"
	"strings"
)

func BeforeDelete(db *gorm2.DB) {
	if db.Error == nil && db.Statement.Schema != nil && !db.Statement.SkipHooks && db.Statement.Schema.BeforeDelete {
		callMethod(db, func(value any, tx *gorm2.DB) bool {
			if i, ok := value.(BeforeDeleteInterface); ok {
				db.AddError(i.BeforeDelete(tx))
				return true
			}

			return false
		})
	}
}

func DeleteBeforeAssociations(db *gorm2.DB) {
	if db.Error == nil && db.Statement.Schema != nil {
		selectColumns, restricted := db.Statement.SelectAndOmitColumns(true, false)
		if !restricted {
			return
		}

		for column, v := range selectColumns {
			if !v {
				continue
			}

			rel, ok := db.Statement.Schema.Relationships.Relations[column]
			if !ok {
				continue
			}

			switch rel.Type {
			case schema2.HasOne, schema2.HasMany:
				queryConds := rel.ToQueryConditions(db.Statement.Context, db.Statement.ReflectValue)
				modelValue := reflect.New(rel.FieldSchema.ModelType).Interface()
				tx := db.Session(&gorm2.Session{NewDB: true}).Model(modelValue)
				withoutConditions := false
				if db.Statement.Unscoped {
					tx = tx.Unscoped()
				}

				if len(db.Statement.Selects) > 0 {
					selects := make([]string, 0, len(db.Statement.Selects))
					for _, s := range db.Statement.Selects {
						if s == clause2.Associations {
							selects = append(selects, s)
						} else if columnPrefix := column + "."; strings.HasPrefix(s, columnPrefix) {
							selects = append(selects, strings.TrimPrefix(s, columnPrefix))
						}
					}

					if len(selects) > 0 {
						tx = tx.Select(selects)
					}
				}

				for _, cond := range queryConds {
					if c, ok := cond.(clause2.IN); ok && len(c.Values) == 0 {
						withoutConditions = true
						break
					}
				}

				if !withoutConditions && db.AddError(tx.Clauses(clause2.Where{Exprs: queryConds}).Delete(modelValue).Error) != nil {
					return
				}
			case schema2.Many2Many:
				var (
					queryConds     = make([]clause2.Expression, 0, len(rel.References))
					foreignFields  = make([]*schema2.Field, 0, len(rel.References))
					relForeignKeys = make([]string, 0, len(rel.References))
					modelValue     = reflect.New(rel.JoinTable.ModelType).Interface()
					table          = rel.JoinTable.Table
					tx             = db.Session(&gorm2.Session{NewDB: true}).Model(modelValue).Table(table)
				)

				for _, ref := range rel.References {
					if ref.OwnPrimaryKey {
						foreignFields = append(foreignFields, ref.PrimaryKey)
						relForeignKeys = append(relForeignKeys, ref.ForeignKey.DBName)
					} else if ref.PrimaryValue != "" {
						queryConds = append(queryConds, clause2.Eq{
							Column: clause2.Column{Table: rel.JoinTable.Table, Name: ref.ForeignKey.DBName},
							Value:  ref.PrimaryValue,
						})
					}
				}

				_, foreignValues := schema2.GetIdentityFieldValuesMap(db.Statement.Context, db.Statement.ReflectValue, foreignFields)
				column, values := schema2.ToQueryValues(table, relForeignKeys, foreignValues)
				queryConds = append(queryConds, clause2.IN{Column: column, Values: values})

				if db.AddError(tx.Clauses(clause2.Where{Exprs: queryConds}).Delete(modelValue).Error) != nil {
					return
				}
			}
		}

	}
}

func Delete(config *Config) func(db *gorm2.DB) {
	supportReturning := utils.Contains(config.DeleteClauses, "RETURNING")

	return func(db *gorm2.DB) {
		if db.Error != nil {
			return
		}

		if db.Statement.Schema != nil {
			for _, c := range db.Statement.Schema.DeleteClauses {
				db.Statement.AddClause(c)
			}
		}

		if db.Statement.SQL.Len() == 0 {
			db.Statement.SQL.Grow(100)
			db.Statement.AddClauseIfNotExists(clause2.Delete{})

			if db.Statement.Schema != nil {
				_, queryValues := schema2.GetIdentityFieldValuesMap(db.Statement.Context, db.Statement.ReflectValue, db.Statement.Schema.PrimaryFields)
				column, values := schema2.ToQueryValues(db.Statement.Table, db.Statement.Schema.PrimaryFieldDBNames, queryValues)

				if len(values) > 0 {
					db.Statement.AddClause(clause2.Where{Exprs: []clause2.Expression{clause2.IN{Column: column, Values: values}}})
				}

				if db.Statement.ReflectValue.CanAddr() && db.Statement.Dest != db.Statement.Model && db.Statement.Model != nil {
					_, queryValues = schema2.GetIdentityFieldValuesMap(db.Statement.Context, reflect.ValueOf(db.Statement.Model), db.Statement.Schema.PrimaryFields)
					column, values = schema2.ToQueryValues(db.Statement.Table, db.Statement.Schema.PrimaryFieldDBNames, queryValues)

					if len(values) > 0 {
						db.Statement.AddClause(clause2.Where{Exprs: []clause2.Expression{clause2.IN{Column: column, Values: values}}})
					}
				}
			}

			db.Statement.AddClauseIfNotExists(clause2.From{})

			db.Statement.Build(db.Statement.BuildClauses...)
		}

		checkMissingWhereConditions(db)

		if !db.DryRun && db.Error == nil {
			ok, mode := hasReturning(db, supportReturning)
			if !ok {
				result, err := db.Statement.ConnPool.ExecContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)
				if db.AddError(err) == nil {
					db.RowsAffected, _ = result.RowsAffected()
				}

				return
			}

			if rows, err := db.Statement.ConnPool.QueryContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...); db.AddError(err) == nil {
				gorm2.Scan(rows, db, mode)
				db.AddError(rows.Close())
			}
		}
	}
}

func AfterDelete(db *gorm2.DB) {
	if db.Error == nil && db.Statement.Schema != nil && !db.Statement.SkipHooks && db.Statement.Schema.AfterDelete {
		callMethod(db, func(value any, tx *gorm2.DB) bool {
			if i, ok := value.(AfterDeleteInterface); ok {
				db.AddError(i.AfterDelete(tx))
				return true
			}
			return false
		})
	}
}
