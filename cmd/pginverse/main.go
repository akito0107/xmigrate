package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/urfave/cli"
	"github.com/xo/dburl"

	"github.com/akito0107/xmigrate"
	"github.com/akito0107/xmigrate/cmd"
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

	schemapath := c.String("schema")

	diffs, current, err := cmd.GetDiff(ctx, schemapath, u)

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
