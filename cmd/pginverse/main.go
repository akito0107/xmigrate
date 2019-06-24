package main

import (
	"context"
	"errors"
	"io"
	"log"
	"os"

	"github.com/akito0107/xsqlparser"
	"github.com/akito0107/xsqlparser/dialect"
	"github.com/urfave/cli"
	"github.com/xo/dburl"
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
		log.Fatal(err)
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

	return nil
}
