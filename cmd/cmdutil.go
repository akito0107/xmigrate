package cmd

import (
	"context"
	"os"

	errors "golang.org/x/xerrors"

	"github.com/akito0107/xsqlparser"
	"github.com/akito0107/xsqlparser/dialect"
	"github.com/akito0107/xsqlparser/sqlast"
	"github.com/xo/dburl"

	"github.com/akito0107/xmigrate"
)

func GetDiff(ctx context.Context, schemapath string, url *dburl.URL) ([]*xmigrate.SchemaDiff, []*xmigrate.TableDef, error) {
	schemafile, err := os.Open(schemapath)
	if err != nil {
		return nil, nil, err
	}
	defer schemafile.Close()

	parser, err := xsqlparser.NewParser(schemafile, &dialect.PostgresqlDialect{})
	if err != nil {
		return nil, nil, err
	}
	sqls, err := parser.ParseSQL()
	if err != nil {
		return nil, nil, err
	}

	var createTables []*sqlast.CreateTableStmt
	var createIndexes []*sqlast.CreateIndexStmt

	for _, s := range sqls {

		switch sql := s.(type) {
		case *sqlast.CreateTableStmt:
			createTables = append(createTables, sql)
		case *sqlast.CreateIndexStmt:
			createIndexes = append(createIndexes, sql)
		default:
			return nil, nil, errors.Errorf("unsupported sql: %s", s.ToSQLString())
		}
	}

	dumper := xmigrate.NewPGDumpFromURL(url)

	res, err := dumper.Dump(ctx)
	if err != nil {
		return nil, nil, err
	}

	diffs, err := xmigrate.Diff(&xmigrate.TargetTable{TableDef: createTables, IndexDef: createIndexes}, res)
	if err != nil {
		return nil, nil, err
	}

	return diffs, res, nil

}
