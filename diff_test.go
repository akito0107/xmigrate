package xmigrate

import (
	"bytes"
	"reflect"
	"testing"
	"unicode"

	"github.com/akito0107/xsqlparser"
	"github.com/akito0107/xsqlparser/dialect"
	"github.com/akito0107/xsqlparser/sqlast"
	"github.com/google/go-cmp/cmp"
)

func TestDiff(t *testing.T) {
	cases := []struct {
		name    string
		target  string
		current string
		expect  []*SchemaDiff
	}{
		{
			name:    "add table",
			target:  "create table test1(id int primary key);",
			current: "",
			expect: []*SchemaDiff{
				{
					Type: AddTable,
					Spec: &AddTableSpec{
						SQL: &sqlast.SQLCreateTable{
							Name: sqlast.NewSQLObjectName("test1"),
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
		},
		{
			name:   "drop table",
			target: "create table test1(id int primary key);",
			current: `create table test1(id int primary key);
create table test2(id int primary key);
`,
			expect: []*SchemaDiff{
				{
					Type: DropTable,
					Spec: &DropTableSpec{
						TableName: "test2",
					},
				},
			},
		},
		{
			name: "add column",
			target: `create table test1(
	id int primary key,
	name varchar not null
);`,
			current: "create table test1(id int primary key);",
			expect: []*SchemaDiff{
				{
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
		},
		{
			name:   "drop column",
			target: "create table test1(id int primary key);",
			current: `create table test1(
	id int primary key,
	name varchar not null
);`,
			expect: []*SchemaDiff{
				{
					Type: DropColumn,
					Spec: &DropColumnSpec{
						TableName:  "test1",
						ColumnName: "name",
					},
				},
			},
		},
		{
			name:    "edit column (change type)",
			target:  "create table test1(id int primary key, name varchar);",
			current: "create table test1(id int primary key, name int);",
			expect: []*SchemaDiff{
				{
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
		},
		{
			name:    "edit column (not null)",
			target:  "create table test1(id int primary key, name varchar not null);",
			current: "create table test1(id int primary key, name varchar);",
			expect: []*SchemaDiff{
				{
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
		},
		{
			name:    "edit column (nullable)",
			target:  "create table test1(id int primary key, name varchar);",
			current: "create table test1(id int primary key, name varchar not null);",
			expect: []*SchemaDiff{
				{
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
		},
		{
			name:    "edit column (set default)",
			target:  "create table test1(id int primary key default 1, name varchar not null);",
			current: "create table test1(id int primary key, name varchar not null);",
			expect: []*SchemaDiff{
				{
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
		},
		{
			name:    "edit column (drop default)",
			target:  "create table test1(id int primary key, name varchar not null);",
			current: "create table test1(id int primary key default 1, name varchar not null);",
			expect: []*SchemaDiff{
				{
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
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			targ := parseCreateTable(t, c.target)
			curr := genTableDef(t, c.current)

			diff, err := Diff(targ, curr)
			if err != nil {
				t.Fatalf("%+v", err)
			}

			if d := cmp.Diff(diff, c.expect, IgnoreMarker); d != "" {
				t.Errorf("diff: %s", d)
			}
		})
	}
}

func TestDSLToDiff(t *testing.T) {
	cases := []struct {
		name   string
		dsl    string
		expect []*SchemaDiff
	}{
		{
			name: "add table",
			dsl:  "create table test1(id int primary key);",
			expect: []*SchemaDiff{
				{
					Type: AddTable,
					Spec: &AddTableSpec{
						SQL: &sqlast.SQLCreateTable{
							Name: sqlast.NewSQLObjectName("test1"),
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
		},
		{
			name: "drop table",
			dsl:  "drop table test2;",
			expect: []*SchemaDiff{
				{
					Type: DropTable,
					Spec: &DropTableSpec{
						TableName: "test2",
					},
				},
			},
		},
		{
			name: "add column",
			dsl:  "ALTER TABLE test1 ADD COLUMN name varchar not null",
			expect: []*SchemaDiff{
				{
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
		},
		{
			name: "drop column",
			dsl:  "ALTER TABLE test1 DROP COLUMN name",
			expect: []*SchemaDiff{
				{
					Type: DropColumn,
					Spec: &DropColumnSpec{
						TableName:  "test1",
						ColumnName: "name",
					},
				},
			},
		},
		{
			name: "edit column (change type)",
			dsl:  "ALTER TABLE test1 ALTER COLUMN name TYPE varchar",
			expect: []*SchemaDiff{
				{
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
		},
		{
			name: "edit column (not null)",
			dsl:  "ALTER TABLE test1 ALTER COLUMN name DROP NOT NULL",
			expect: []*SchemaDiff{
				{
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
		},
		{
			name: "edit column (nullable)",
			dsl:  "ALTER TABLE test1 ALTER COLUMN name SET NOT NULL",
			expect: []*SchemaDiff{
				{
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
		},
		{
			name: "edit column (set default)",
			dsl:  "ALTER TABLE test1 ALTER COLUMN id SET DEFAULT 1",
			expect: []*SchemaDiff{
				{
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
		},
		{
			name: "edit column (drop default)",
			dsl:  "ALTER TABLE test1 ALTER COLUMN id DROP DEFAULT",
			expect: []*SchemaDiff{
				{
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
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			parser, err := xsqlparser.NewParser(bytes.NewBufferString(c.dsl), &dialect.PostgresqlDialect{})
			if err != nil {
				t.Fatalf("%+v", err)
			}

			stmts, err := parser.ParseSQL()
			if err != nil {
				t.Fatalf("%+v", err)
			}
			expect, err := DSLToDiff(stmts)
			if err != nil {
				t.Fatalf("%+v", err)
			}
			if diff := cmp.Diff(expect, c.expect, IgnoreMarker); diff != "" {
				t.Errorf(diff)
			}
		})
	}
}

func genTableDef(t *testing.T, def string) []*TableDef {
	t.Helper()
	creates := parseCreateTable(t, def)

	var defs []*TableDef
	for _, c := range creates {
		columns := make(map[string]*sqlast.SQLColumnDef)

		for _, col := range c.Elements {
			column, ok := col.(*sqlast.SQLColumnDef)
			if !ok {
				t.Fatalf("%s is not columndef", col.ToSQLString())
			}

			columns[column.Name.ToSQLString()] = column
		}
		defs = append(defs, &TableDef{
			Name:    c.Name.ToSQLString(),
			Columns: columns,
		})
	}

	return defs
}

func parseCreateTable(t *testing.T, in string) []*sqlast.SQLCreateTable {
	t.Helper()
	parser, err := xsqlparser.NewParser(bytes.NewBufferString(in), &dialect.PostgresqlDialect{})
	if err != nil {
		t.Fatal(err)
	}

	stmts, err := parser.ParseSQL()
	if err != nil {
		t.Fatal(err)
	}

	var creates []*sqlast.SQLCreateTable

	for _, s := range stmts {
		c, ok := s.(*sqlast.SQLCreateTable)
		if !ok {
			t.Fatalf("%s is not a create table stmts", s.ToSQLString())
		}

		creates = append(creates, c)
	}

	return creates
}

var IgnoreMarker = cmp.FilterPath(func(paths cmp.Path) bool {
	s := paths.Last().Type()
	name := s.Name()
	r := []rune(name)
	return s.Kind() == reflect.Struct && len(r) > 0 && unicode.IsLower(r[0])
}, cmp.Ignore())
