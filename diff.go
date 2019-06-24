package xmigrate

import (
	"reflect"
	"strings"

	errors "golang.org/x/xerrors"

	"github.com/akito0107/xsqlparser/sqlast"
)

type DiffType int

const (
	AddColumn DiffType = iota
	DropColumn
	AddTable
	DropTable
	EditColumn
	AddTableConstraint
	DropTableConstraint
)

type SchemaDiff struct {
	Type DiffType
	Spec DiffSpec
}

func Diff(targ []*sqlast.SQLCreateTable, currentTable []*TableDef) ([]*SchemaDiff, error) {
	var diffs []*SchemaDiff

	targetState := make(map[string]*sqlast.SQLCreateTable)
	for _, t := range targ {
		targetState[strings.ToLower(t.Name.ToSQLString())] = t
	}

	currentState := make(map[string]*TableDef)
	for _, c := range currentTable {
		currentState[c.Name] = c
	}

	for n, v := range targetState {
		c, ok := currentState[n]
		if !ok {
			spec := createAddTableSpec(v)
			diffs = append(diffs, &SchemaDiff{
				Type: AddTable,
				Spec: spec,
			})
			continue
		}
		diff, err := computeTableDiff(v, c)
		if err != nil {
			return nil, errors.Errorf("computeTableDiff failed: %w", err)
		}

		diffs = append(diffs, diff...)
	}

	for n, v := range currentState {
		t, ok := targetState[strings.ToLower(n)]
		if !ok {
			spec := createDropTableSpec(v)
			diffs = append(diffs, &SchemaDiff{
				Type: DropTable,
				Spec: spec,
			})
			continue
		}

		for _, currentConstraint := range v.Constrains {
			var found bool
			for _, e := range t.Elements {
				c, ok := e.(*sqlast.TableConstraint)
				if !ok {
					continue
				}
				if strings.EqualFold(c.Name.ToSQLString(), currentConstraint.Name.ToSQLString()) {
					found = true
					break
				}
			}

			if !found {
				diffs = append(diffs, &SchemaDiff{
					Type: DropTableConstraint,
					Spec: &DropTableConstraintSpec{
						TableName:       t.Name.ToSQLString(),
						ConstraintsName: currentConstraint.Name.ToSQLString(),
					},
				})
			}
		}
	}

	return diffs, nil
}

func DSLToDiff(stmts []sqlast.SQLStmt) ([]*SchemaDiff, error) {
	var diff []*SchemaDiff
	for _, stmt := range stmts {
		switch st := stmt.(type) {
		case *sqlast.SQLCreateTable:
			diff = append(diff, &SchemaDiff{
				Type: AddTable,
				Spec: &AddTableSpec{
					SQL: st,
				},
			})
		case *sqlast.SQLDropTable:
			diff = append(diff, &SchemaDiff{
				Type: DropTable,
				Spec: &DropTableSpec{
					// TODO: allow multiple table names
					TableName: st.TableNames[0].ToSQLString(),
				},
			})
		case *sqlast.SQLAlterTable:
			switch al := (st.Action).(type) {
			case *sqlast.AddColumnTableAction:
				diff = append(diff, &SchemaDiff{
					Type: AddColumn,
					Spec: &AddColumnSpec{
						TableName: st.TableName.ToSQLString(),
						ColumnDef: al.Column,
					},
				})
			case *sqlast.RemoveColumnTableAction:
				diff = append(diff, &SchemaDiff{
					Type: DropColumn,
					Spec: &DropColumnSpec{
						TableName:  st.TableName.ToSQLString(),
						ColumnName: al.Name.ToSQLString(),
					},
				})
			case *sqlast.AddConstraintTableAction:
				diff = append(diff, &SchemaDiff{
					Type: AddTableConstraint,
					Spec: &AddTableConstraintSpec{
						TableName:     st.TableName.ToSQLString(),
						ConstraintDef: al.Constraint,
					},
				})
			case *sqlast.DropConstraintTableAction:
				diff = append(diff, &SchemaDiff{
					Type: DropTableConstraint,
					Spec: &DropTableConstraintSpec{
						TableName:       st.TableName.ToSQLString(),
						ConstraintsName: al.Name.ToSQLString(),
					},
				})
			case *sqlast.AlterColumnTableAction:
				switch al.Action.(type) {
				case *sqlast.SetDefaultColumnAction:
					diff = append(diff, &SchemaDiff{
						Type: EditColumn,
						Spec: &EditColumnSpec{
							Type:       SetDefault,
							TableName:  st.TableName.ToSQLString(),
							ColumnName: al.ColumnName.ToSQLString(),
							SQL:        st,
						},
					})
				case *sqlast.DropDefaultColumnAction:
					diff = append(diff, &SchemaDiff{
						Type: EditColumn,
						Spec: &EditColumnSpec{
							Type:       DropDefault,
							TableName:  st.TableName.ToSQLString(),
							ColumnName: al.ColumnName.ToSQLString(),
							SQL:        st,
						},
					})
				case *sqlast.PGAlterDataTypeColumnAction:
					diff = append(diff, &SchemaDiff{
						Type: EditColumn,
						Spec: &EditColumnSpec{
							Type:       EditType,
							TableName:  st.TableName.ToSQLString(),
							ColumnName: al.ColumnName.ToSQLString(),
							SQL:        st,
						},
					})
				case *sqlast.PGDropNotNullColumnAction:
					diff = append(diff, &SchemaDiff{
						Type: EditColumn,
						Spec: &EditColumnSpec{
							Type:       DropNotNull,
							TableName:  st.TableName.ToSQLString(),
							ColumnName: al.ColumnName.ToSQLString(),
							SQL:        st,
						},
					})
				case *sqlast.PGSetNotNullColumnAction:
					diff = append(diff, &SchemaDiff{
						Type: EditColumn,
						Spec: &EditColumnSpec{
							Type:       SetNotNull,
							TableName:  st.TableName.ToSQLString(),
							ColumnName: al.ColumnName.ToSQLString(),
							SQL:        st,
						},
					})
				}
			default:
				return nil, errors.Errorf("%s is not supported", al.ToSQLString())
			}
		default:
			return nil, errors.Errorf("%s is not supported", stmt.ToSQLString())
		}

	}

	return diff, nil
}

type DiffSpec interface {
	ToSQLString() string
}

type AddTableSpec struct {
	SQL *sqlast.SQLCreateTable
}

func (a *AddTableSpec) ToSQLString() string {
	return a.SQL.ToSQLString()
}

func createAddTableSpec(targ *sqlast.SQLCreateTable) *AddTableSpec {
	return &AddTableSpec{SQL: targ}
}

type DropTableSpec struct {
	TableName string
}

func (d *DropTableSpec) ToSQLString() string {
	sql := &sqlast.SQLDropTable{
		TableNames: []*sqlast.SQLObjectName{sqlast.NewSQLObjectName(d.TableName)},
		IfExists:   true,
	}

	return sql.ToSQLString()
}

func createDropTableSpec(currentTable *TableDef) *DropTableSpec {
	return &DropTableSpec{TableName: currentTable.Name}
}

type AddColumnSpec struct {
	TableName string
	ColumnDef *sqlast.SQLColumnDef
}

func (a *AddColumnSpec) ToSQLString() string {
	sql := sqlast.SQLAlterTable{
		TableName: sqlast.NewSQLObjectName(a.TableName),
		Action: &sqlast.AddColumnTableAction{
			Column: a.ColumnDef,
		},
	}

	return sql.ToSQLString()
}

type DropColumnSpec struct {
	TableName  string
	ColumnName string
}

func (d *DropColumnSpec) ToSQLString() string {
	sql := sqlast.SQLAlterTable{
		TableName: sqlast.NewSQLObjectName(d.TableName),
		Action: &sqlast.RemoveColumnTableAction{
			Name: sqlast.NewSQLIdent(d.ColumnName),
		},
	}

	return sql.ToSQLString()
}

type AddTableConstraintSpec struct {
	TableName     string
	ConstraintDef *sqlast.TableConstraint
}

func (a *AddTableConstraintSpec) ToSQLString() string {
	sql := &sqlast.SQLAlterTable{
		TableName: sqlast.NewSQLObjectName(a.TableName),
		Action: &sqlast.AddConstraintTableAction{
			Constraint: a.ConstraintDef,
		},
	}

	return sql.ToSQLString()
}

type DropTableConstraintSpec struct {
	TableName       string
	ConstraintsName string
}

func (d *DropTableConstraintSpec) ToSQLString() string {
	sql := &sqlast.SQLAlterTable{
		TableName: sqlast.NewSQLObjectName(d.TableName),
		Action: &sqlast.DropConstraintTableAction{
			Name: sqlast.NewSQLIdent(d.ConstraintsName),
		},
	}

	return sql.ToSQLString()
}

func computeTableDiff(targ *sqlast.SQLCreateTable, currentTable *TableDef) ([]*SchemaDiff, error) {
	var diffs []*SchemaDiff
	var targNames []string

	cmap := make(map[string]struct{})

	for _, e := range targ.Elements {
		switch tp := e.(type) {
		case *sqlast.SQLColumnDef:
			cmap[strings.ToLower(tp.Name.ToSQLString())] = struct{}{}
			targNames = append(targNames, tp.Name.ToSQLString())

			curr, ok := currentTable.Columns[strings.ToLower(tp.Name.ToSQLString())]
			if !ok {
				diffs = append(diffs, &SchemaDiff{
					Type: AddColumn,
					Spec: &AddColumnSpec{
						TableName: currentTable.Name,
						ColumnDef: tp,
					},
				})
				continue
			}

			hasDiff, diff, err := computeColumnDiff(targ.Name.ToSQLString(), tp, curr)
			if err != nil {
				return nil, errors.Errorf("computeColumnDiff failed: %w", err)
			}

			if hasDiff {
				diffs = append(diffs, diff...)
			}

		case *sqlast.TableConstraint:
			var found bool
			for _, c := range currentTable.Constrains {
				if strings.EqualFold(tp.Name.ToSQLString(), c.Name.ToSQLString()) {
					found = true
					break
				}
			}

			if !found {
				diffs = append(diffs, &SchemaDiff{
					Type: AddTableConstraint,
					Spec: &AddTableConstraintSpec{
						TableName:     targ.Name.ToSQLString(),
						ConstraintDef: tp,
					},
				})
			}
		default:
			return nil, errors.Errorf("unknown elements %s", e.ToSQLString())
		}
	}

	for c := range currentTable.Columns {
		_, ok := cmap[c]
		if !ok {
			diffs = append(diffs, &SchemaDiff{
				Type: DropColumn,
				Spec: &DropColumnSpec{
					TableName:  currentTable.Name,
					ColumnName: c,
				},
			})
		}
	}

	return diffs, nil
}

type EditColumnType int

const (
	EditType EditColumnType = iota
	SetNotNull
	DropNotNull
	SetDefault
	DropDefault
)

type EditColumnSpec struct {
	Type       EditColumnType
	TableName  string
	ColumnName string
	SQL        *sqlast.SQLAlterTable
}

func (e *EditColumnSpec) ToSQLString() string {
	return e.SQL.ToSQLString()
}

type EditColumnAction interface {
	ToSQLString() string
}

// pattern:
// type change
// NULL <=> NOT NULL
// constraints change
//   unique
//   check
func computeColumnDiff(tableName string, targ *sqlast.SQLColumnDef, current *sqlast.SQLColumnDef) (bool, []*SchemaDiff, error) {
	var diffs []*SchemaDiff

	// change data type
	if !reflect.DeepEqual(targ.DataType, current.DataType) {
		sql := &sqlast.SQLAlterTable{
			TableName: sqlast.NewSQLObjectName(tableName),
			Action: &sqlast.AlterColumnTableAction{
				ColumnName: targ.Name,
				Action: &sqlast.PGAlterDataTypeColumnAction{
					DataType: targ.DataType,
				},
			},
		}

		diffs = append(diffs, &SchemaDiff{
			Type: EditColumn,
			Spec: &EditColumnSpec{
				TableName:  tableName,
				Type:       EditType,
				ColumnName: targ.Name.ToSQLString(),
				SQL:        sql,
			},
		})
	}

	tnn := hasNotNullConstraint(targ)
	cnn := hasNotNullConstraint(current)

	if tnn && !cnn {
		sql := &sqlast.SQLAlterTable{
			TableName: sqlast.NewSQLObjectName(tableName),
			Action: &sqlast.AlterColumnTableAction{
				ColumnName: targ.Name,
				Action:     &sqlast.PGSetNotNullColumnAction{},
			},
		}

		diffs = append(diffs, &SchemaDiff{
			Type: EditColumn,
			Spec: &EditColumnSpec{
				TableName:  tableName,
				Type:       SetNotNull,
				ColumnName: targ.Name.ToSQLString(),
				SQL:        sql,
			},
		})
	} else if !tnn && cnn {
		sql := &sqlast.SQLAlterTable{
			TableName: sqlast.NewSQLObjectName(tableName),
			Action: &sqlast.AlterColumnTableAction{
				ColumnName: targ.Name,
				Action:     &sqlast.PGDropNotNullColumnAction{},
			},
		}

		diffs = append(diffs, &SchemaDiff{
			Type: EditColumn,
			Spec: &EditColumnSpec{
				TableName:  tableName,
				Type:       DropNotNull,
				ColumnName: targ.Name.ToSQLString(),
				SQL:        sql,
			},
		})
	}

	tdef := hasDefaultClause(targ)
	cdef := hasDefaultClause(current)

	if tdef && !cdef {
		sql := &sqlast.SQLAlterTable{
			TableName: sqlast.NewSQLObjectName(tableName),
			Action: &sqlast.AlterColumnTableAction{
				ColumnName: targ.Name,
				Action: &sqlast.SetDefaultColumnAction{
					Default: targ.Default,
				},
			},
		}
		diffs = append(diffs, &SchemaDiff{
			Type: EditColumn,
			Spec: &EditColumnSpec{
				TableName:  tableName,
				Type:       SetDefault,
				ColumnName: targ.Name.ToSQLString(),
				SQL:        sql,
			},
		})
	} else if !tdef && cdef {
		sql := &sqlast.SQLAlterTable{
			TableName: sqlast.NewSQLObjectName(tableName),
			Action: &sqlast.AlterColumnTableAction{
				ColumnName: targ.Name,
				Action:     &sqlast.DropDefaultColumnAction{},
			},
		}
		diffs = append(diffs, &SchemaDiff{
			Type: EditColumn,
			Spec: &EditColumnSpec{
				TableName:  tableName,
				Type:       DropDefault,
				ColumnName: targ.Name.ToSQLString(),
				SQL:        sql,
			},
		})
	}

	if len(diffs) > 0 {
		return true, diffs, nil
	}

	return false, nil, nil
}

func hasNotNullConstraint(def *sqlast.SQLColumnDef) bool {
	for _, c := range def.Constraints {
		if _, ok := c.Spec.(*sqlast.NotNullColumnSpec); ok {
			return true
		}
		if _, ok := c.Spec.(*sqlast.UniqueColumnSpec); ok {
			return true
		}
	}
	return false
}

func hasDefaultClause(def *sqlast.SQLColumnDef) bool {
	return def.Default != nil
}
