package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/akito0107/xsqlparser"
	"github.com/akito0107/xsqlparser/dialect"
	"github.com/akito0107/xsqlparser/sqlast"
	"github.com/jmoiron/sqlx"
	"github.com/urfave/cli"

	"github.com/akito0107/xmigrate"
)

func main() {
	app := cli.NewApp()
	app.Name = "xmigrate"
	app.Usage = "postgres db migration utility"
	app.UsageText = "xmigrate [GLOBAL OPTIONS] [COMMANDS] [sub command options]"

	app.Flags = []cli.Flag{
		cli.StringFlag{Name: "host", Value: "127.0.0.1", Usage: "db host"},
		cli.StringFlag{Name: "port,p", Value: "5432", Usage: "db host"},
		cli.StringFlag{Name: "dbname,d", Value: "", Usage: "dbname"},
		cli.StringFlag{Name: "password,W", Value: "", Usage: "password"},
		cli.StringFlag{Name: "username,U", Value: "postgres", Usage: "db user name"},
		cli.BoolFlag{Name: "verbose"},
	}

	app.Commands = []cli.Command{
		{
			Name: "sync",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "schema,f", Value: "schema.sql", Usage: "target schema file path"},
				cli.BoolFlag{Name: "apply", Usage: "applying query (default dry-run mode)"},
			},
			Action: syncAction,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalf("%+v", err)
	}
}

func getConf(c *cli.Context) *xmigrate.PGConf {
	host := c.GlobalString("host")
	port := c.GlobalString("port")
	dbname := c.GlobalString("dbname")
	password := c.GlobalString("password")
	username := c.GlobalString("username")
	return &xmigrate.PGConf{
		DBName:     dbname,
		DBHost:     host,
		DBPort:     port,
		DBPassword: password,
		UserName:   username,
	}
}

func getDiff(ctx context.Context, schemapath string, conf *xmigrate.PGConf) ([]*xmigrate.SchemaDiff, []*xmigrate.TableDef, error) {
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

	for _, s := range sqls {
		c, ok := s.(*sqlast.CreateTableStmt)
		if !ok {
			continue
		}
		createTables = append(createTables, c)
	}

	dumper := xmigrate.NewPGDump(conf)

	res, err := dumper.Dump(ctx)
	if err != nil {
		return nil, nil, err
	}

	diffs, err := xmigrate.Diff(&xmigrate.TargetTable{TableDef: createTables}, res)
	if err != nil {
		return nil, nil, err
	}

	return diffs, res, nil

}

func syncAction(c *cli.Context) error {
	ctx := context.Background()

	conf := getConf(c)
	schemapath := c.String("schema")

	diffs, _, err := getDiff(ctx, schemapath, conf)
	if err != nil {
		return err
	}

	apply := c.Bool("apply")
	if !apply {
		fmt.Println("dry-run mode (with --apply flag will be exec below queries)")
	}
	var db *sqlx.DB

	if apply {
		d, err := sqlx.Open("postgres", conf.Src())
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
