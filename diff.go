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
	AddIndex
	RemoveIndex
)

type TargetTable struct {
	TableDef []*sqlast.CreateTableStmt
	IndexDef []*sqlast.CreateIndexStmt
}

type SchemaDiff struct {
	Type DiffType
	Spec DiffSpec
}

func Diff(targ *TargetTable, currentTable []*TableDef) ([]*SchemaDiff, error) {
	var diffs []*SchemaDiff

	targetState := make(map[string]*sqlast.CreateTableStmt)
	targetIndexes := partitionIndexByName(targ.IndexDef)

	for _, t := range targ.TableDef {
		targetState[strings.ToLower(t.Name.ToSQLString())] = t
	}

	currentIndexes := make(map[string]*sqlast.CreateIndexStmt)
	currentState := make(map[string]*TableDef)
	for _, c := range currentTable {
		currentState[c.Name] = c
		for n, i := range c.Indexes {
			currentIndexes[n] = i
		}
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

	diffs = append(diffs, computeIndexDiff(currentIndexes, targetIndexes)...)

	return diffs, nil
}

func DSLToDiff(stmts []sqlast.Stmt) ([]*SchemaDiff, error) {
	var diff []*SchemaDiff
	for _, stmt := range stmts {
		switch st := stmt.(type) {
		case *sqlast.CreateTableStmt:
			diff = append(diff, &SchemaDiff{
				Type: AddTable,
				Spec: &AddTableSpec{
					SQL: st,
				},
			})
		case *sqlast.DropTableStmt:
			diff = append(diff, &SchemaDiff{
				Type: DropTable,
				Spec: &DropTableSpec{
					// TODO: allow multiple table names
					TableName: st.TableNames[0].ToSQLString(),
				},
			})
		case *sqlast.AlterTableStmt:
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

func partitionIndexByName(indexes []*sqlast.CreateIndexStmt) map[string]*sqlast.CreateIndexStmt {
	p := make(map[string]*sqlast.CreateIndexStmt)

	for _, i := range indexes {
		p[i.IndexName.ToSQLString()] = i
	}

	return p
}

type DiffSpec interface {
	ToSQLString() string
}

type AddTableSpec struct {
	SQL *sqlast.CreateTableStmt
}

func (a *AddTableSpec) ToSQLString() string {
	return a.SQL.ToSQLString()
}

func createAddTableSpec(targ *sqlast.CreateTableStmt) *AddTableSpec {
	return &AddTableSpec{SQL: targ}
}

type DropTableSpec struct {
	TableName string
}

func (d *DropTableSpec) ToSQLString() string {
	sql := &sqlast.DropTableStmt{
		TableNames: []*sqlast.ObjectName{sqlast.NewSQLObjectName(d.TableName)},
		IfExists:   true,
	}

	return sql.ToSQLString()
}

func createDropTableSpec(currentTable *TableDef) *DropTableSpec {
	return &DropTableSpec{TableName: currentTable.Name}
}

type AddColumnSpec struct {
	TableName string
	ColumnDef *sqlast.ColumnDef
}

func (a *AddColumnSpec) ToSQLString() string {
	sql := sqlast.AlterTableStmt{
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
	sql := sqlast.AlterTableStmt{
		TableName: sqlast.NewSQLObjectName(d.TableName),
		Action: &sqlast.RemoveColumnTableAction{
			Name: sqlast.NewIdent(d.ColumnName),
		},
	}

	return sql.ToSQLString()
}

type AddTableConstraintSpec struct {
	TableName     string
	ConstraintDef *sqlast.TableConstraint
}

func (a *AddTableConstraintSpec) ToSQLString() string {
	sql := &sqlast.AlterTableStmt{
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
	sql := &sqlast.AlterTableStmt{
		TableName: sqlast.NewSQLObjectName(d.TableName),
		Action: &sqlast.DropConstraintTableAction{
			Name: sqlast.NewIdent(d.ConstraintsName),
		},
	}

	return sql.ToSQLString()
}

type AddIndexSpec struct {
	Def *sqlast.CreateIndexStmt
}

func (a *AddIndexSpec) ToSQLString() string {
	return a.Def.ToSQLString()
}

type DropIndexSpec struct {
	IndexName string
}

func (d *DropIndexSpec) ToSQLString() string {
	sql := &sqlast.DropIndexStmt{
		IndexNames: []*sqlast.Ident{sqlast.NewIdent(d.IndexName)},
	}
	return sql.ToSQLString()
}

func computeTableDiff(targ *sqlast.CreateTableStmt, currentTable *TableDef) ([]*SchemaDiff, error) {
	var diffs []*SchemaDiff
	var targNames []string

	cmap := make(map[string]struct{})

	for _, e := range targ.Elements {
		switch tp := e.(type) {
		case *sqlast.ColumnDef:
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
	SQL        *sqlast.AlterTableStmt
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
func computeColumnDiff(tableName string, targ *sqlast.ColumnDef, current *sqlast.ColumnDef) (bool, []*SchemaDiff, error) {
	var diffs []*SchemaDiff

	tgTp, tok := targ.DataType.(*sqlast.Custom)
	cuTp, cok := targ.DataType.(*sqlast.Custom)
	// custom type is not comparable via reflection
	if tok && cok {
		if strings.EqualFold(tgTp.ToSQLString(), cuTp.ToSQLString()) {
			return false, nil, nil
		}
	}
	if !reflect.DeepEqual(targ.DataType, current.DataType) {
		sql := &sqlast.AlterTableStmt{
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
		sql := &sqlast.AlterTableStmt{
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
		sql := &sqlast.AlterTableStmt{
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
		sql := &sqlast.AlterTableStmt{
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
		sql := &sqlast.AlterTableStmt{
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

func hasNotNullConstraint(def *sqlast.ColumnDef) bool {
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

func hasDefaultClause(def *sqlast.ColumnDef) bool {
	return def.Default != nil
}

func computeIndexDiff(current, target map[string]*sqlast.CreateIndexStmt) []*SchemaDiff {
	var diffs []*SchemaDiff
	for n, i := range current {
		_, ok := target[n]
		if !ok {
			diffs = append(diffs, &SchemaDiff{
				Type: RemoveIndex,
				Spec: &DropIndexSpec{
					IndexName: i.IndexName.ToSQLString(),
				},
			})
		}
	}

	for n, i := range target {
		_, ok := current[n]
		if !ok {
			diffs = append(diffs, &SchemaDiff{
				Type: AddIndex,
				Spec: &AddIndexSpec{
					Def: i,
				},
			})
		}
	}

	return diffs
}
