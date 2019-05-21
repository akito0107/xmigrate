package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/akito0107/xmigrate"
	"github.com/akito0107/xsqlparser"
	"github.com/akito0107/xsqlparser/dialect"
	"github.com/akito0107/xsqlparser/sqlast"
	"github.com/jmoiron/sqlx"
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
		cli.StringFlag{Name: "username,U", Value: "postgres", Usage: "db user name"},
		cli.BoolFlag{Name: "verbose"},
	}

	app.Commands = []cli.Command{
		{
			Name:    "diff",
			Aliases: []string{"d"},
			Usage:   "check diff between current table and schema.sql",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "schema,f", Value: "schema.sql", Usage: "target schema file path"},
				cli.BoolFlag{Name: "preview"},
				cli.StringFlag{Name: "migrations,m", Value: "migrations", Usage: "migrations file dir"},
			},
			Action: diffAction,
		},
		{
			Name: "sync",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "schema,f", Value: "schema.sql", Usage: "target schema file path"},
				cli.BoolFlag{Name: "apply", Usage: "applying query (default dry-run mode)"},
			},
			Action: syncAction,
		},
		{
			Name: "new",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "migrations,m", Value: "migrations", Usage: "migrations file dir"},
			},
			Action: newAction,
		},
		{
			Name: "up",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "migrations,m", Value: "migrations", Usage: "migrations file dir"},
			},
			Action: upAction,
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
		return nil, nil, err
	}

	diffs, err := xmigrate.Diff(createTables, res)
	if err != nil {
		return nil, nil, err
	}

	return diffs, res, nil

}

func diffAction(c *cli.Context) error {
	ctx := context.Background()

	conf := getConf(c)
	schemapath := c.String("schema")

	diffs, current, err := getDiff(ctx, schemapath, conf)

	preview := c.Bool("preview")
	migdir := c.String("migrations")

	for _, d := range diffs {
		err = func() error {
			var upout io.Writer
			var downout io.Writer

			if preview {
				fmt.Println("diff between current and target state is...")
				upout = os.Stdout
				downout = os.Stdout
			} else {
				t := time.Now()
				id := fmt.Sprintf("%d", t.UnixNano())

				upf, err := os.Create(fmt.Sprintf("%s/%s.up.sql", migdir, id))
				if err != nil {
					return err
				}
				defer upf.Close()
				upout = upf

				downf, err := os.Create(fmt.Sprintf("%s/%s.down.sql", migdir, id))
				if err != nil {
					return err
				}
				defer downf.Close()
				downout = downf
			}

			fmt.Fprintln(upout, d.Spec.ToSQLString())
			inv, err := xmigrate.Inverse(d, current)
			if err != nil {
				return err
			}
			if preview {
				fmt.Println("inverse: ")
			}
			fmt.Fprintln(downout, inv.Spec.ToSQLString())

			return nil
		}()
		if err != nil {
			return err
		}
	}
	return nil
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

func newAction(c *cli.Context) error {
	migdir := c.String("migrations")

	t := time.Now()
	id := fmt.Sprintf("%d", t.UnixNano())

	mes := "-- created by xmigrate"

	upf, err := os.Create(fmt.Sprintf("%s/%s.up.sql", migdir, id))
	if err != nil {
		return err
	}
	defer upf.Close()
	fmt.Fprintf(upf, mes)

	downf, err := os.Create(fmt.Sprintf("%s/%s.down.sql", migdir, id))
	if err != nil {
		return err
	}
	defer downf.Close()
	fmt.Fprintf(downf, mes)

	return nil
}

func upAction(c *cli.Context) error {
	conf := getConf(c)
	ctx := context.Background()

	migdir := c.String("migrations")

	db, err := sqlx.Connect("postgres", conf.Src())
	if err != nil {
		return err
	}
	defer db.Close()

	currentId, err := xmigrate.CheckCurrent(ctx, db)
	if err != nil {
		return err
	}

	files, err := ioutil.ReadDir(migdir)
	if err != nil {
		return err
	}

	var upmigrates []string

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".up.sql") {
			continue
		}
	}

	return nil
}
