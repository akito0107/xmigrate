package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/urfave/cli"
	"github.com/xo/dburl"

	"github.com/akito0107/xmigrate"
	"github.com/akito0107/xmigrate/cmd"
	"github.com/akito0107/xmigrate/toposort"
)

func main() {
	app := cli.NewApp()
	app.Name = "pgmigrate"
	app.Usage = "postgres db migration utility"
	app.UsageText = "pgmigrate [db url] [OPTIONS]"
	app.Flags = []cli.Flag{
		cli.StringFlag{Name: "schemapath, f", Value: "schema.sql", Usage: "schema sql path"},
		cli.BoolFlag{Name: "apply, a", Usage: "apply migration"},
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

func syncAction(c *cli.Context, u *dburl.URL) error {
	ctx := context.Background()

	schemapath := c.String("schemapath")

	diffs, _, err := cmd.GetDiff(ctx, schemapath, u)
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
	graph := xmigrate.CalcGraph(diffs)
	resolved, err := toposort.ResolveGraph(graph)
	if err != nil {
		return err
	}

	for _, n := range resolved.Nodes {
		d := n.(*xmigrate.DiffNode)
		sql := d.Diff.Spec.ToSQLString()

		fmt.Printf("applying: %s\n", sql)

		if apply {
			if _, err := db.Exec(sql); err != nil {
				return err
			}
		}
	}

	return nil
}
