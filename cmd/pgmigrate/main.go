package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/akito0107/xsqlparser"
	"github.com/akito0107/xsqlparser/dialect"
	"github.com/akito0107/xsqlparser/sqlast"
	"github.com/jmoiron/sqlx"
	"github.com/urfave/cli"
	"github.com/xo/dburl"

	"github.com/akito0107/xmigrate"
)

func main() {
	app := cli.NewApp()
	app.Name = "pgmigrate"
	app.Usage = "postgres db migration utility"
	app.UsageText = "pgmigrate [db url] [OPTIONS]"
	app.Flags = []cli.Flag{
		cli.StringFlag{Name: "schemapath,f", Value: "schema.sql", Usage: "schema sql path"},
		cli.BoolFlag{Name: "apply,a", Usage: "apply migration"},
	}

	app.Action = func(c *cli.Context) error {
		dbsrc := c.Args().Get(0)
		if dbsrc == "" {
			return errors.New("db url is required")
		}

		u, err := dburl.Parse(dbsrc)
		if err != nil {
			return err
		}
		return syncAction(c, u)
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func getDiff(ctx context.Context, schemapath string, url *dburl.URL) ([]*xmigrate.SchemaDiff, []*xmigrate.TableDef, error) {
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

	var createTables []*sqlast.SQLCreateTable

	for _, s := range sqls {
		c, ok := s.(*sqlast.SQLCreateTable)
		if !ok {
			continue
		}
		createTables = append(createTables, c)
	}

	dumper := xmigrate.NewPGDumpFromURL(url)

	res, err := dumper.Dump(ctx)
	if err != nil {
		return nil, nil, err
	}

	diffs, err := xmigrate.Diff(createTables, res)
	if err != nil {
		return nil, nil, err
	}

	return diffs, res, nil

}

func syncAction(c *cli.Context, u *dburl.URL) error {
	ctx := context.Background()

	schemapath := c.String("schema")

	diffs, _, err := getDiff(ctx, schemapath, u)
	if err != nil {
		return err
	}

	apply := c.Bool("apply")
	if !apply {
		fmt.Println("dry-run mode (with --apply flag will be exec below queries)")
	}
	var db *sqlx.DB

	if apply {
		d, err := sqlx.Open(u.Driver, u.DSN)
		if err != nil {
			return err
		}
		db = d
		defer db.Close()
	}

	for _, d := range diffs {
		sql := d.Spec.ToSQLString()
		fmt.Printf("applying: %s\n", sql)
		if apply {
			if _, err := db.Exec(sql); err != nil {
				return err
			}
		}
	}

	return nil
}
