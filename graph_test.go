package xmigrate

import (
	"bytes"
	"testing"

	"github.com/akito0107/xsqlparser"
	"github.com/akito0107/xsqlparser/dialect"
	"github.com/google/go-cmp/cmp"

	"github.com/akito0107/xmigrate/toposort"
)

func TestCalcGraph(t *testing.T) {
	cases := []struct {
		name   string
		diffs  string
		expect []string
	}{
		{
			name: "one deps",
			diffs: `CREATE TABLE test1 (id int PRIMARY KEY);
					ALTER TABLE test2 ADD COLUMN t1_ref int REFERENCES test1(id);`,
			expect: []string{
				"CREATE TABLE test1 (id int PRIMARY KEY)",
				"ALTER TABLE test2 ADD COLUMN t1_ref int REFERENCES test1(id)",
			},
		},
		{
			name: "reverse deps",
			diffs: `ALTER TABLE test2 ADD COLUMN t1_ref int REFERENCES test1(id);
					CREATE TABLE test1 (id int PRIMARY KEY);`,
			expect: []string{
				"CREATE TABLE test1 (id int PRIMARY KEY)",
				"ALTER TABLE test2 ADD COLUMN t1_ref int REFERENCES test1(id)",
			},
		},
		{
			name: "create table",
			diffs: `CREATE TABLE test2 (id int primary key, t1_ref int REFERENCES test1(id));
					CREATE TABLE test1 (id int PRIMARY KEY);`,
			expect: []string{
				"CREATE TABLE test1 (id int PRIMARY KEY)",
				"CREATE TABLE test2 (id int PRIMARY KEY, t1_ref int REFERENCES test1(id))",
			},
		},
		{
			name: "three deps",
			diffs: `CREATE TABLE test3 (id int primary key, t2_ref int, CONSTRAINT t2_ref FOREIGN KEY(t2_ref) REFERENCES test2(id));
					CREATE TABLE test2 (id int primary key, t1_ref int REFERENCES test1(id));
					CREATE TABLE test1 (id int PRIMARY KEY);`,
			expect: []string{
				"CREATE TABLE test1 (id int PRIMARY KEY)",
				"CREATE TABLE test2 (id int PRIMARY KEY, t1_ref int REFERENCES test1(id))",
				"CREATE TABLE test3 (id int PRIMARY KEY, t2_ref int, CONSTRAINT t2_ref FOREIGN KEY(t2_ref) REFERENCES test2(id))",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			parser, err := xsqlparser.NewParser(bytes.NewBufferString(c.diffs), &dialect.PostgresqlDialect{})
			if err != nil {
				t.Fatalf("%+v", err)
			}

			stmts, err := parser.ParseSQL()
			if err != nil {
				t.Fatalf("%+v", err)
			}
			diffs, err := DSLToDiff(stmts)
			if err != nil {
				t.Fatalf("%+v", err)
			}
			graph := CalcGraph(diffs)
			resolved, err := toposort.ResolveGraph(graph)
			if err != nil {
				t.Fatal(err)
			}

			var act []string

			for _, r := range resolved.Nodes {
				act = append(act, r.(*DiffNode).Diff.Spec.ToSQLString())
			}

			if d := cmp.Diff(act, c.expect); d != "" {
				t.Errorf("diff: %s", d)
			}
		})

	}

}
