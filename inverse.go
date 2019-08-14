package xmigrate

import (
	"strings"

	"github.com/akito0107/xsqlparser/sqlast"
	errors "golang.org/x/xerrors"
)

func Inverse(diff *SchemaDiff, currentTable []*TableDef) (*SchemaDiff, error) {
	switch spec := diff.Spec.(type) {
	case *AddTableSpec:
		return &SchemaDiff{
			Type: DropTable,
			Spec: &DropTableSpec{
				TableName: spec.SQL.Name.ToSQLString(),
			},
		}, nil
	case *AddColumnSpec:
		return &SchemaDiff{
			Type: DropColumn,
			Spec: &DropColumnSpec{
				TableName:  spec.TableName,
				ColumnName: spec.ColumnDef.Name.ToSQLString(),
			},
		}, nil
	case *DropColumnSpec:
		t := getTable(spec.TableName, currentTable)
		col := t.Columns[spec.ColumnName]

		return &SchemaDiff{
			Type: AddColumn,
			Spec: &AddColumnSpec{
				TableName: spec.TableName,
				ColumnDef: refineColumn(col),
			},
		}, nil

	case *DropTableSpec:
		t := getTable(spec.TableName, currentTable)
		var columndef []sqlast.TableElement

		for _, c := range t.Columns {
			columndef = append(columndef, c)
		}

		return &SchemaDiff{
			Type: AddTable,
			Spec: &AddTableSpec{
				SQL: &sqlast.CreateTableStmt{
					Name:     sqlast.NewSQLObjectName(spec.TableName),
					Elements: columndef,
				},
			},
		}, nil

	case *EditColumnSpec:
		switch spec.Type {
		case SetNotNull:
			return &SchemaDiff{
				Type: EditColumn,
				Spec: &EditColumnSpec{
					Type:       DropNotNull,
					TableName:  spec.TableName,
					ColumnName: spec.ColumnName,
					SQL: &sqlast.AlterTableStmt{
						TableName: sqlast.NewSQLObjectName(spec.SQL.TableName.ToSQLString()),
						Action: &sqlast.AlterColumnTableAction{
							ColumnName: sqlast.NewIdent(spec.ColumnName),
							Action:     &sqlast.PGDropNotNullColumnAction{},
						},
					},
				},
			}, nil
		case DropNotNull:
			return &SchemaDiff{
				Type: EditColumn,
				Spec: &EditColumnSpec{
					Type:       SetNotNull,
					TableName:  spec.TableName,
					ColumnName: spec.ColumnName,
					SQL: &sqlast.AlterTableStmt{
						TableName: sqlast.NewSQLObjectName(spec.SQL.TableName.ToSQLString()),
						Action: &sqlast.AlterColumnTableAction{
							ColumnName: sqlast.NewIdent(spec.ColumnName),
							Action:     &sqlast.PGSetNotNullColumnAction{},
						},
					},
				},
			}, nil

		case DropDefault:
			t := getTable(spec.TableName, currentTable)
			col := t.Columns[spec.ColumnName]
			return &SchemaDiff{
				Type: EditColumn,
				Spec: &EditColumnSpec{
					Type:       SetDefault,
					TableName:  spec.TableName,
					ColumnName: spec.ColumnName,
					SQL: &sqlast.AlterTableStmt{
						TableName: spec.SQL.TableName,
						Action: &sqlast.AlterColumnTableAction{
							ColumnName: sqlast.NewIdent(spec.ColumnName),
							Action: &sqlast.SetDefaultColumnAction{
								Default: col.Default,
							},
						},
					},
				},
			}, nil
		case SetDefault:
			return &SchemaDiff{
				Type: EditColumn,
				Spec: &EditColumnSpec{
					Type:       DropDefault,
					TableName:  spec.TableName,
					ColumnName: spec.ColumnName,
					SQL: &sqlast.AlterTableStmt{
						TableName: spec.SQL.TableName,
						Action: &sqlast.AlterColumnTableAction{
							ColumnName: sqlast.NewIdent(spec.ColumnName),
							Action:     &sqlast.DropDefaultColumnAction{},
						},
					},
				},
			}, nil
		case EditType:
			t := getTable(spec.TableName, currentTable)
			col := t.Columns[spec.ColumnName]
			return &SchemaDiff{
				Type: EditColumn,
				Spec: &EditColumnSpec{
					Type:       EditType,
					TableName:  spec.TableName,
					ColumnName: spec.ColumnName,
					SQL: &sqlast.AlterTableStmt{
						TableName: sqlast.NewSQLObjectName(spec.SQL.TableName.ToSQLString()),
						Action: &sqlast.AlterColumnTableAction{
							ColumnName: sqlast.NewIdent(spec.ColumnName),
							Action: &sqlast.PGAlterDataTypeColumnAction{
								DataType: col.DataType,
							},
						},
					},
				},
			}, nil
		default:
			return nil, errors.Errorf("unknown spec %s", diff.Spec.ToSQLString())
		}

	default:
		return nil, errors.Errorf("unknown spec %+v", diff)
	}
}

func getTable(tableName string, ts []*TableDef) *TableDef {
	for _, t := range ts {
		if t.Name == tableName {
			return t
		}
	}
	return nil
}

func refineColumn(org *sqlast.ColumnDef) *sqlast.ColumnDef {
	// convert into serial
	_, ok := org.DataType.(*sqlast.Int)
	if !ok {
		return org
	}

	fn, ok := org.Default.(*sqlast.Function)
	if !ok {
		return org
	}

	if strings.HasPrefix(fn.Name.ToSQLString(), "nextval") {
		return &sqlast.ColumnDef{
			Name:        org.Name,
			DataType:    &sqlast.Custom{Ty: sqlast.NewSQLObjectName("SERIAL")},
			Constraints: org.Constraints,
		}
	}

	return org
}
