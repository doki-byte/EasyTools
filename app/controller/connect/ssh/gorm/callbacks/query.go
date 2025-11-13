package callbacks

import (
	gorm2 "EasyTools/app/controller/connect/ssh/gorm"
	clause2 "EasyTools/app/controller/connect/ssh/gorm/clause"
	"EasyTools/app/controller/connect/ssh/gorm/schema"
	"EasyTools/app/controller/connect/ssh/gorm/utils"
	"fmt"
	"reflect"
	"strings"
)

func Query(db *gorm2.DB) {
	if db.Error == nil {
		BuildQuerySQL(db)

		if !db.DryRun && db.Error == nil {
			rows, err := db.Statement.ConnPool.QueryContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)
			if err != nil {
				db.AddError(err)
				return
			}
			defer func() {
				db.AddError(rows.Close())
			}()
			gorm2.Scan(rows, db, 0)
		}
	}
}

func BuildQuerySQL(db *gorm2.DB) {
	if db.Statement.Schema != nil {
		for _, c := range db.Statement.Schema.QueryClauses {
			db.Statement.AddClause(c)
		}
	}

	if db.Statement.SQL.Len() == 0 {
		db.Statement.SQL.Grow(100)
		clauseSelect := clause2.Select{Distinct: db.Statement.Distinct}

		if db.Statement.ReflectValue.Kind() == reflect.Struct && db.Statement.ReflectValue.Type() == db.Statement.Schema.ModelType {
			var conds []clause2.Expression
			for _, primaryField := range db.Statement.Schema.PrimaryFields {
				if v, isZero := primaryField.ValueOf(db.Statement.Context, db.Statement.ReflectValue); !isZero {
					conds = append(conds, clause2.Eq{Column: clause2.Column{Table: db.Statement.Table, Name: primaryField.DBName}, Value: v})
				}
			}

			if len(conds) > 0 {
				db.Statement.AddClause(clause2.Where{Exprs: conds})
			}
		}

		if len(db.Statement.Selects) > 0 {
			clauseSelect.Columns = make([]clause2.Column, len(db.Statement.Selects))
			for idx, name := range db.Statement.Selects {
				if db.Statement.Schema == nil {
					clauseSelect.Columns[idx] = clause2.Column{Name: name, Raw: true}
				} else if f := db.Statement.Schema.LookUpField(name); f != nil {
					clauseSelect.Columns[idx] = clause2.Column{Name: f.DBName}
				} else {
					clauseSelect.Columns[idx] = clause2.Column{Name: name, Raw: true}
				}
			}
		} else if db.Statement.Schema != nil && len(db.Statement.Omits) > 0 {
			selectColumns, _ := db.Statement.SelectAndOmitColumns(false, false)
			clauseSelect.Columns = make([]clause2.Column, 0, len(db.Statement.Schema.DBNames))
			for _, dbName := range db.Statement.Schema.DBNames {
				if v, ok := selectColumns[dbName]; (ok && v) || !ok {
					clauseSelect.Columns = append(clauseSelect.Columns, clause2.Column{Table: db.Statement.Table, Name: dbName})
				}
			}
		} else if db.Statement.Schema != nil && db.Statement.ReflectValue.IsValid() {
			queryFields := db.QueryFields
			if !queryFields {
				switch db.Statement.ReflectValue.Kind() {
				case reflect.Struct:
					queryFields = db.Statement.ReflectValue.Type() != db.Statement.Schema.ModelType
				case reflect.Slice:
					queryFields = db.Statement.ReflectValue.Type().Elem() != db.Statement.Schema.ModelType
				}
			}

			if queryFields {
				stmt := gorm2.Statement{DB: db}
				// smaller struct
				if err := stmt.Parse(db.Statement.Dest); err == nil && (db.QueryFields || stmt.Schema.ModelType != db.Statement.Schema.ModelType) {
					clauseSelect.Columns = make([]clause2.Column, len(stmt.Schema.DBNames))

					for idx, dbName := range stmt.Schema.DBNames {
						clauseSelect.Columns[idx] = clause2.Column{Table: db.Statement.Table, Name: dbName}
					}
				}
			}
		}

		// inline joins
		fromClause := clause2.From{}
		if v, ok := db.Statement.Clauses["FROM"].Expression.(clause2.From); ok {
			fromClause = v
		}

		if len(db.Statement.Joins) != 0 || len(fromClause.Joins) != 0 {
			if len(db.Statement.Selects) == 0 && len(db.Statement.Omits) == 0 && db.Statement.Schema != nil {
				clauseSelect.Columns = make([]clause2.Column, len(db.Statement.Schema.DBNames))
				for idx, dbName := range db.Statement.Schema.DBNames {
					clauseSelect.Columns[idx] = clause2.Column{Table: db.Statement.Table, Name: dbName}
				}
			}

			specifiedRelationsName := make(map[string]any)
			for _, join := range db.Statement.Joins {
				if db.Statement.Schema != nil {
					var isRelations bool // is relations or raw sql
					var relations []*schema.Relationship
					relation, ok := db.Statement.Schema.Relationships.Relations[join.Name]
					if ok {
						isRelations = true
						relations = append(relations, relation)
					} else {
						// handle nested join like "Manager.Company"
						nestedJoinNames := strings.Split(join.Name, ".")
						if len(nestedJoinNames) > 1 {
							isNestedJoin := true
							gussNestedRelations := make([]*schema.Relationship, 0, len(nestedJoinNames))
							currentRelations := db.Statement.Schema.Relationships.Relations
							for _, relname := range nestedJoinNames {
								// incomplete match, only treated as raw sql
								if relation, ok = currentRelations[relname]; ok {
									gussNestedRelations = append(gussNestedRelations, relation)
									currentRelations = relation.FieldSchema.Relationships.Relations
								} else {
									isNestedJoin = false
									break
								}
							}

							if isNestedJoin {
								isRelations = true
								relations = gussNestedRelations
							}
						}
					}

					if isRelations {
						genJoinClause := func(joinType clause2.JoinType, parentTableName string, relation *schema.Relationship) clause2.Join {
							tableAliasName := relation.Name
							if parentTableName != clause2.CurrentTable {
								tableAliasName = utils.NestedRelationName(parentTableName, tableAliasName)
							}

							columnStmt := gorm2.Statement{
								Table: tableAliasName, DB: db, Schema: relation.FieldSchema,
								Selects: join.Selects, Omits: join.Omits,
							}

							selectColumns, restricted := columnStmt.SelectAndOmitColumns(false, false)
							for _, s := range relation.FieldSchema.DBNames {
								if v, ok := selectColumns[s]; (ok && v) || (!ok && !restricted) {
									clauseSelect.Columns = append(clauseSelect.Columns, clause2.Column{
										Table: tableAliasName,
										Name:  s,
										Alias: utils.NestedRelationName(tableAliasName, s),
									})
								}
							}

							exprs := make([]clause2.Expression, len(relation.References))
							for idx, ref := range relation.References {
								if ref.OwnPrimaryKey {
									exprs[idx] = clause2.Eq{
										Column: clause2.Column{Table: parentTableName, Name: ref.PrimaryKey.DBName},
										Value:  clause2.Column{Table: tableAliasName, Name: ref.ForeignKey.DBName},
									}
								} else {
									if ref.PrimaryValue == "" {
										exprs[idx] = clause2.Eq{
											Column: clause2.Column{Table: parentTableName, Name: ref.ForeignKey.DBName},
											Value:  clause2.Column{Table: tableAliasName, Name: ref.PrimaryKey.DBName},
										}
									} else {
										exprs[idx] = clause2.Eq{
											Column: clause2.Column{Table: tableAliasName, Name: ref.ForeignKey.DBName},
											Value:  ref.PrimaryValue,
										}
									}
								}
							}

							{
								onStmt := gorm2.Statement{Table: tableAliasName, DB: db, Clauses: map[string]clause2.Clause{}}
								for _, c := range relation.FieldSchema.QueryClauses {
									onStmt.AddClause(c)
								}

								if join.On != nil {
									onStmt.AddClause(join.On)
								}

								if cs, ok := onStmt.Clauses["WHERE"]; ok {
									if where, ok := cs.Expression.(clause2.Where); ok {
										where.Build(&onStmt)

										if onSQL := onStmt.SQL.String(); onSQL != "" {
											vars := onStmt.Vars
											for idx, v := range vars {
												bindvar := strings.Builder{}
												onStmt.Vars = vars[0 : idx+1]
												db.Dialector.BindVarTo(&bindvar, &onStmt, v)
												onSQL = strings.Replace(onSQL, bindvar.String(), "?", 1)
											}

											exprs = append(exprs, clause2.Expr{SQL: onSQL, Vars: vars})
										}
									}
								}
							}

							return clause2.Join{
								Type:  joinType,
								Table: clause2.Table{Name: relation.FieldSchema.Table, Alias: tableAliasName},
								ON:    clause2.Where{Exprs: exprs},
							}
						}

						parentTableName := clause2.CurrentTable
						for _, rel := range relations {
							// joins table alias like "Manager, Company, Manager__Company"
							nestedAlias := utils.NestedRelationName(parentTableName, rel.Name)
							if _, ok := specifiedRelationsName[nestedAlias]; !ok {
								fromClause.Joins = append(fromClause.Joins, genJoinClause(join.JoinType, parentTableName, rel))
								specifiedRelationsName[nestedAlias] = nil
							}

							if parentTableName != clause2.CurrentTable {
								parentTableName = utils.NestedRelationName(parentTableName, rel.Name)
							} else {
								parentTableName = rel.Name
							}
						}
					} else {
						fromClause.Joins = append(fromClause.Joins, clause2.Join{
							Expression: clause2.NamedExpr{SQL: join.Name, Vars: join.Conds},
						})
					}
				} else {
					fromClause.Joins = append(fromClause.Joins, clause2.Join{
						Expression: clause2.NamedExpr{SQL: join.Name, Vars: join.Conds},
					})
				}
			}

			db.Statement.AddClause(fromClause)
		} else {
			db.Statement.AddClauseIfNotExists(clause2.From{})
		}

		db.Statement.AddClauseIfNotExists(clauseSelect)

		db.Statement.Build(db.Statement.BuildClauses...)
	}
}

func Preload(db *gorm2.DB) {
	if db.Error == nil && len(db.Statement.Preloads) > 0 {
		if db.Statement.Schema == nil {
			db.AddError(fmt.Errorf("%w when using preload", gorm2.ErrModelValueRequired))
			return
		}

		joins := make([]string, 0, len(db.Statement.Joins))
		for _, join := range db.Statement.Joins {
			joins = append(joins, join.Name)
		}

		tx := preloadDB(db, db.Statement.ReflectValue, db.Statement.Dest)
		if tx.Error != nil {
			return
		}

		db.AddError(preloadEntryPoint(tx, joins, &tx.Statement.Schema.Relationships, db.Statement.Preloads, db.Statement.Preloads[clause2.Associations]))
	}
}

func AfterQuery(db *gorm2.DB) {
	// clear the joins after query because preload need it
	db.Statement.Joins = nil
	if db.Error == nil && db.Statement.Schema != nil && !db.Statement.SkipHooks && db.Statement.Schema.AfterFind && db.RowsAffected > 0 {
		callMethod(db, func(value any, tx *gorm2.DB) bool {
			if i, ok := value.(AfterFindInterface); ok {
				db.AddError(i.AfterFind(tx))
				return true
			}
			return false
		})
	}
}
