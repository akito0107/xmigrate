package xmigrate

import (
	"log"
	"reflect"

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
)

type SchemaDiff struct {
	Type DiffType
	Spec DiffSpec
}

func Diff(targ []*sqlast.SQLCreateTable, currentTable []*TableDef) ([]*SchemaDiff, error) {
	var diffs []*SchemaDiff

	targetState := make(map[string]*sqlast.SQLCreateTable)
	for _, t := range targ {
		targetState[t.Name.ToSQLString()] = t
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
		_, ok := targetState[n]
		if !ok {
			spec := createDropTableSpec(v)
			diffs = append(diffs, &SchemaDiff{
				Type: DropTable,
				Spec: spec,
			})
		}
	}

	return diffs, nil
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

func computeTableDiff(targ *sqlast.SQLCreateTable, currentTable *TableDef) ([]*SchemaDiff, error) {
	var diffs []*SchemaDiff
	var targNames []string

	cmap := make(map[string]struct{})

	for _, e := range targ.Elements {
		switch tp := e.(type) {
		case *sqlast.SQLColumnDef:
			cmap[tp.Name.ToSQLString()] = struct{}{}
			targNames = append(targNames, tp.Name.ToSQLString())

			curr, ok := currentTable.Columns[tp.Name.ToSQLString()]
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
			log.Printf("table constraint %s uninmplemented \n", e.ToSQLString())
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

	tnn := hasNotNullConstrait(targ)
	cnn := hasNotNullConstrait(current)

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

	if len(diffs) > 0 {
		return true, diffs, nil
	}

	return false, nil, nil
}

func hasNotNullConstrait(def *sqlast.SQLColumnDef) bool {
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
