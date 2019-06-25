package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/akito0107/xsqlparser"
	"github.com/akito0107/xsqlparser/dialect"
	"github.com/urfave/cli"
	"github.com/xo/dburl"

	"github.com/akito0107/xmigrate"
)

func main() {
	app := cli.NewApp()
	app.Name = "pginverse"
	app.Usage = "postgres db migration utility"
	app.UsageText = "pginverse [db url] [OPTIONS]"
	app.Flags = []cli.Flag{
		cli.StringFlag{Name: "in,i", Value: "stdin", Usage: "input file name (default = stdin)"},
		cli.StringFlag{Name: "out,o", Value: "stdout", Usage: "output file name (default = stdout)"},
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
		return diffAction(c, u)
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalf("%+v", err)
	}
}

func diffAction(c *cli.Context, u *dburl.URL) error {
	ctx := context.Background()

	insrc := c.GlobalString("in")
	outsrc := c.GlobalString("out")

	var in io.Reader

	if insrc == "stdin" {
		in = os.Stdin
	} else {
		f, err := os.Open(insrc)
		if err != nil {
			return err
		}
		defer f.Close()
		in = f
	}

	var out io.Writer

	if outsrc == "stdout" {
		out = os.Stdout
	} else {
		f, err := os.Open(outsrc)
		if err != nil {
			return err
		}
		defer f.Close()
		out = f
	}

	parser, err := xsqlparser.NewParser(in, &dialect.PostgresqlDialect{})
	if err != nil {
		return err
	}

	stmts, err := parser.ParseSQL()
	if err != nil {
		return err
	}

	diffs, err := xmigrate.DSLToDiff(stmts)
	if err != nil {
		return err
	}

	dumper := xmigrate.NewPGDumpFromURL(u)
	current, err := dumper.Dump(ctx)
	if err != nil {
		return err
	}

	for _, d := range diffs {
		inv, err := xmigrate.Inverse(d, current)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%s\n", inv.Spec.ToSQLString())
	}

	return nil
}
