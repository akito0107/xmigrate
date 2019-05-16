package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/akito0107/xmigrate"
	"github.com/akito0107/xsqlparser"
	"github.com/akito0107/xsqlparser/dialect"
	"github.com/akito0107/xsqlparser/sqlast"
	"github.com/urfave/cli"
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
		cli.StringFlag{Name: "username,U", Value: "postgress", Usage: "db user name"},
		cli.BoolFlag{Name: "verbose"},
	}

	app.Commands = []cli.Command{
		{
			Name:    "diff",
			Aliases: []string{"g"},
			Usage:   "check diff between current table and schema.sql",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "schema,f", Value: "schema.sql", Usage: "target schema file path"},
			},
			Action: diffAction,
		},
	}

	app.Run(os.Args)
}

func diffAction(c *cli.Context) error {
	ctx := context.Background()

	host := c.GlobalString("host")
	port := c.GlobalString("port")
	dbname := c.GlobalString("dbname")
	password := c.GlobalString("password")
	username := c.GlobalString("username")

	v := c.GlobalBool("verbose")
	debug := func(format string, args ...interface{}) {
		if v {
			log.Printf(format, args)
		}
	}

	conf := &xmigrate.PGConf{
		DBName:     dbname,
		DBHost:     host,
		DBPort:     port,
		DBPassword: password,
		UserName:   username,
	}
	schemapath := c.String("schema")

	debug("%+v", schemapath)

	schemafile, err := os.Open(schemapath)
	if err != nil {
		return err
	}
	defer schemafile.Close()

	parser, err := xsqlparser.NewParser(schemafile, &dialect.PostgresqlDialect{})
	if err != nil {
		return err
	}
	sqls, err := parser.ParseSQL()
	if err != nil {
		return err
	}
	debug("%+v", sqls)

	var createTables []*sqlast.SQLCreateTable

	for _, s := range sqls {
		c, ok := s.(*sqlast.SQLCreateTable)
		if !ok {
			continue
		}
		createTables = append(createTables, c)
	}

	dumper := xmigrate.NewPGDump(conf)

	res, err := dumper.Dump(ctx)
	if err != nil {
		return err
	}

	diffs, err := xmigrate.Diff(createTables, res)
	if err != nil {
		return err
	}

	fmt.Println("diff between current and target state is...")
	for _, d := range diffs {
		fmt.Println(d.Spec.ToSQLString())
	}

	return nil
}
