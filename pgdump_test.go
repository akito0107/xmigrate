package xmigrate

import (
	"context"
	"testing"

	"github.com/akito0107/xsqlparser/sqlast"
	"github.com/google/go-cmp/cmp"
)

func TestPGDump_DumpHelpers(t *testing.T) {

	conf := &PGConf{
		DBName:   "xmigrate_test",
		DBHost:   "localhost",
		DBPort:   "5432",
		UserName: "postgres",
	}

	dumper := NewPGDump(conf)

	t.Run("getTableNames", func(t *testing.T) {
		ctx := context.Background()

		tableNames, err := dumper.getTableNames(ctx)
		if err != nil {
			t.Fatalf("%+v", err)
		}

		expected := []string{"account", "category", "item", "subcategory", "subitem"}

		if diff := cmp.Diff(tableNames, expected); diff != "" {
			t.Errorf("should be same but %s", diff)
		}
	})

	t.Run("getColumnDefinition", func(t *testing.T) {
		t.Run("accont table (no refs)", func(t *testing.T) {
			ctx := context.Background()
			defs, err := dumper.getColumnDefinition(ctx, "account")
			if err != nil {
				t.Fatalf("%+v", err)
			}

			if len(defs) != 5 {
				t.Errorf("should be 5 columns but %d", len(defs))
			}
		})

	})

}

func TestPGDump_Dump(t *testing.T) {
	conf := &PGConf{
		DBName:     "xmigrate_test",
		DBHost:     "127.0.0.1",
		DBPort:     "5432",
		DBPassword: "passw0rd",
		UserName:   "postgres",
	}

	dumper := NewPGDump(conf)
	ctx := context.Background()
	dumped, err := dumper.Dump(ctx)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	if len(dumped) != 5 {
		t.Fatalf("%+v", dumped)
	}

	for _, d := range dumped {
		if d.Name == "account" {

			exceptIdx, err := getParser("CREATE INDEX name_idx ON public.account using btree (name);").ParseStatement()
			if err != nil {
				t.Fatal(err)
			}

			if len(d.Indexes) != 1 {
				t.Fatal("must be index only name_idx")
			}

			idx := exceptIdx.(*sqlast.SQLCreateIndex)
			if diff := cmp.Diff(idx, d.Indexes["name_idx"], IgnoreMarker); diff != "" {
				t.Errorf("diff: %s", diff)
			}
		}

		if d.Name == "item" {
			exceptIdx, err := getParser("create index cat_name_idx on public.item using btree (category_id, name);").ParseStatement()
			if err != nil {
				t.Fatal(err)
			}

			if len(d.Indexes) != 1 {
				t.Fatalf("must be index only cate_name_idx but %+v", d.Indexes)
			}

			idx := exceptIdx.(*sqlast.SQLCreateIndex)
			if diff := cmp.Diff(idx, d.Indexes["cat_name_idx"], IgnoreMarker); diff != "" {
				t.Errorf("diff: %s", diff)
			}
		}
	}

}
