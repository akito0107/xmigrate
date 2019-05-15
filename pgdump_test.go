package xmigrate

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/k0kubun/pp"
)

func TestPGDump_DumpHelpers(t *testing.T) {

	conf := &PGConf{
		DBName:     "xmigrate_test",
		DBHost:     "127.0.0.1",
		DBPort:     "5432",
		DBPassword: "passw0rd",
		UserName:   "postgres",
	}

	dumper := NewPGDump(conf)

	t.Run("getTableNames", func(t *testing.T) {
		ctx := context.Background()

		tableNames, err := dumper.getTableNames(ctx)
		if err != nil {
			t.Fatalf("%+v", err)
		}

		expected := []string{"account", "category", "item"}

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
			pp.Println(defs)
		})

	})

}
