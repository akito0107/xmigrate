package xmigrate

import (
	"testing"

	"github.com/akito0107/xsqlparser/sqlast"
	"github.com/google/go-cmp/cmp"
)

func TestInverse(t *testing.T) {

	cases := []struct {
		name    string
		current string
		diff    *SchemaDiff
		expect  *SchemaDiff
	}{
		{
			name:    "add table",
			current: "create table test1(id int primary key);",
			diff: &SchemaDiff{
				Type: AddTable,
				Spec: &AddTableSpec{
					SQL: &sqlast.SQLCreateTable{
						Name: sqlast.NewSQLObjectName("test2"),
						Elements: []sqlast.TableElement{
							&sqlast.SQLColumnDef{
								Name:     sqlast.NewSQLIdent("id"),
								DataType: &sqlast.Int{},
								Constraints: []*sqlast.ColumnConstraint{
									{
										Spec: &sqlast.UniqueColumnSpec{
											IsPrimaryKey: true,
										},
									},
								},
							},
						},
					},
				},
			},
			expect: &SchemaDiff{
				Type: DropTable,
				Spec: &DropTableSpec{
					TableName: "test2",
				},
			},
		},
		{
			name: "drop table",
			current: `create table test1(id int primary key);
create table test2(id int primary key);`,
			diff: &SchemaDiff{
				Type: DropTable,
				Spec: &DropTableSpec{
					TableName: "test2",
				},
			},
			expect: &SchemaDiff{
				Type: AddTable,
				Spec: &AddTableSpec{
					SQL: &sqlast.SQLCreateTable{
						Name: sqlast.NewSQLObjectName("test2"),
						Elements: []sqlast.TableElement{
							&sqlast.SQLColumnDef{
								Name:     sqlast.NewSQLIdent("id"),
								DataType: &sqlast.Int{},
								Constraints: []*sqlast.ColumnConstraint{
									{
										Spec: &sqlast.UniqueColumnSpec{
											IsPrimaryKey: true,
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:    "add column",
			current: `create table test1(id int primary key);`,
			diff: &SchemaDiff{
				Type: AddColumn,
				Spec: &AddColumnSpec{
					TableName: "test1",
					ColumnDef: &sqlast.SQLColumnDef{
						Name:     sqlast.NewSQLIdent("name"),
						DataType: &sqlast.VarcharType{},
						Constraints: []*sqlast.ColumnConstraint{
							{
								Spec: &sqlast.NotNullColumnSpec{},
							},
						},
					},
				},
			},
			expect: &SchemaDiff{
				Type: DropColumn,
				Spec: &DropColumnSpec{
					TableName:  "test1",
					ColumnName: "name",
				},
			},
		},
		{
			name:    "drop column",
			current: `create table test1(id int primary key, name varchar not null);`,
			diff: &SchemaDiff{
				Type: DropColumn,
				Spec: &DropColumnSpec{
					TableName:  "test1",
					ColumnName: "name",
				},
			},
			expect: &SchemaDiff{
				Type: AddColumn,
				Spec: &AddColumnSpec{
					TableName: "test1",
					ColumnDef: &sqlast.SQLColumnDef{
						Name:     sqlast.NewSQLIdent("name"),
						DataType: &sqlast.VarcharType{},
						Constraints: []*sqlast.ColumnConstraint{
							{
								Spec: &sqlast.NotNullColumnSpec{},
							},
						},
					},
				},
			},
		},
		{
			name:    "edit column",
			current: `create table test1(id int primary key, name varchar not null);`,
			diff: &SchemaDiff{
				Type: EditColumn,
				Spec: &EditColumnSpec{
					Type:       EditType,
					TableName:  "test1",
					ColumnName: "name",
					SQL: &sqlast.SQLAlterTable{
						TableName: sqlast.NewSQLObjectName("test1"),
						Action: &sqlast.AlterColumnTableAction{
							ColumnName: sqlast.NewSQLIdent("name"),
							Action: &sqlast.PGAlterDataTypeColumnAction{
								DataType: &sqlast.Int{},
							},
						},
					},
				},
			},
			expect: &SchemaDiff{
				Type: EditColumn,
				Spec: &EditColumnSpec{
					Type:       EditType,
					TableName:  "test1",
					ColumnName: "name",
					SQL: &sqlast.SQLAlterTable{
						TableName: sqlast.NewSQLObjectName("test1"),
						Action: &sqlast.AlterColumnTableAction{
							ColumnName: sqlast.NewSQLIdent("name"),
							Action: &sqlast.PGAlterDataTypeColumnAction{
								DataType: &sqlast.VarcharType{},
							},
						},
					},
				},
			},
		},
		{
			name:    "nullable",
			current: `create table test1(id int primary key, name varchar not null);`,
			diff: &SchemaDiff{
				Type: EditColumn,
				Spec: &EditColumnSpec{
					Type:       DropNotNull,
					TableName:  "test1",
					ColumnName: "name",
					SQL: &sqlast.SQLAlterTable{
						TableName: sqlast.NewSQLObjectName("test1"),
						Action: &sqlast.AlterColumnTableAction{
							ColumnName: sqlast.NewSQLIdent("name"),
							Action:     &sqlast.PGDropNotNullColumnAction{},
						},
					},
				},
			},
			expect: &SchemaDiff{
				Type: EditColumn,
				Spec: &EditColumnSpec{
					Type:       SetNotNull,
					TableName:  "test1",
					ColumnName: "name",
					SQL: &sqlast.SQLAlterTable{
						TableName: sqlast.NewSQLObjectName("test1"),
						Action: &sqlast.AlterColumnTableAction{
							ColumnName: sqlast.NewSQLIdent("name"),
							Action:     &sqlast.PGSetNotNullColumnAction{},
						},
					},
				},
			},
		},
		{
			name:    "nullable",
			current: `create table test1(id int primary key, name varchar);`,
			diff: &SchemaDiff{
				Type: EditColumn,
				Spec: &EditColumnSpec{
					Type:       SetNotNull,
					TableName:  "test1",
					ColumnName: "name",
					SQL: &sqlast.SQLAlterTable{
						TableName: sqlast.NewSQLObjectName("test1"),
						Action: &sqlast.AlterColumnTableAction{
							ColumnName: sqlast.NewSQLIdent("name"),
							Action:     &sqlast.PGSetNotNullColumnAction{},
						},
					},
				},
			},
			expect: &SchemaDiff{
				Type: EditColumn,
				Spec: &EditColumnSpec{
					Type:       DropNotNull,
					TableName:  "test1",
					ColumnName: "name",
					SQL: &sqlast.SQLAlterTable{
						TableName: sqlast.NewSQLObjectName("test1"),
						Action: &sqlast.AlterColumnTableAction{
							ColumnName: sqlast.NewSQLIdent("name"),
							Action:     &sqlast.PGDropNotNullColumnAction{},
						},
					},
				},
			},
		},
		{
			name:    "set default",
			current: `create table test1(id int primary key, name varchar);`,
			diff: &SchemaDiff{
				Type: EditColumn,
				Spec: &EditColumnSpec{
					Type:       SetDefault,
					TableName:  "test1",
					ColumnName: "name",
					SQL: &sqlast.SQLAlterTable{
						TableName: sqlast.NewSQLObjectName("test1"),
						Action: &sqlast.AlterColumnTableAction{
							ColumnName: sqlast.NewSQLIdent("name"),
							Action: &sqlast.SetDefaultColumnAction{
								Default: sqlast.NewLongValue(1),
							},
						},
					},
				},
			},
			expect: &SchemaDiff{
				Type: EditColumn,
				Spec: &EditColumnSpec{
					Type:       DropDefault,
					TableName:  "test1",
					ColumnName: "name",
					SQL: &sqlast.SQLAlterTable{
						TableName: sqlast.NewSQLObjectName("test1"),
						Action: &sqlast.AlterColumnTableAction{
							ColumnName: sqlast.NewSQLIdent("name"),
							Action:     &sqlast.DropDefaultColumnAction{},
						},
					},
				},
			},
		},
		{
			name:    "drop default",
			current: `create table test1(id int primary key default 1, name varchar);`,
			diff: &SchemaDiff{
				Type: EditColumn,
				Spec: &EditColumnSpec{
					Type:       DropDefault,
					TableName:  "test1",
					ColumnName: "id",
					SQL: &sqlast.SQLAlterTable{
						TableName: sqlast.NewSQLObjectName("test1"),
						Action: &sqlast.AlterColumnTableAction{
							ColumnName: sqlast.NewSQLIdent("id"),
							Action:     &sqlast.DropDefaultColumnAction{},
						},
					},
				},
			},
			expect: &SchemaDiff{
				Type: EditColumn,
				Spec: &EditColumnSpec{
					Type:       SetDefault,
					TableName:  "test1",
					ColumnName: "id",
					SQL: &sqlast.SQLAlterTable{
						TableName: sqlast.NewSQLObjectName("test1"),
						Action: &sqlast.AlterColumnTableAction{
							ColumnName: sqlast.NewSQLIdent("id"),
							Action: &sqlast.SetDefaultColumnAction{
								Default: sqlast.NewLongValue(1),
							},
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			def := genTableDef(t, c.current)
			act, err := Inverse(c.diff, def)
			if err != nil {
				t.Fatalf("%+v", err)
			}

			if d := cmp.Diff(act, c.expect, IgnoreMarker); d != "" {
				t.Errorf("Diff: %s", d)
			}
		})
	}

}
