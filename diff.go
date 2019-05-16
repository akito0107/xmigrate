package xmigrate

import (
	"log"

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
		diff, err := computeDiff(v, c)
		if err != nil {
			return nil, errors.Errorf("computeDiff failed: %w", err)
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

func computeDiff(targ *sqlast.SQLCreateTable, currentTable *TableDef) ([]*SchemaDiff, error) {
	var diffs []*SchemaDiff
	var targNames []string

	cmap := make(map[string]struct{})

	for _, e := range targ.Elements {
		switch tp := e.(type) {
		case *sqlast.SQLColumnDef:
			cmap[tp.Name.ToSQLString()] = struct{}{}
			targNames = append(targNames, tp.Name.ToSQLString())

			_, ok := currentTable.Columns[tp.Name.ToSQLString()]
			if !ok {
				diffs = append(diffs, &SchemaDiff{
					Type: AddColumn,
					Spec: &AddColumnSpec{
						TableName: currentTable.Name,
						ColumnDef: tp,
					},
				})
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
