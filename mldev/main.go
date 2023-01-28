package main

import (
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v2"
	"moslrn.net/ml-dev/lib/setup"
)

func main() {
	log.SetFlags(0)
	app := &cli.App{
		Name:  "ml-dev",
		Usage: "Utilities for Mosaic Learning developers",
		Commands: []*cli.Command{
			{
				Name:    "setup",
				Aliases: []string{"s"},
				Usage:   "Setup a repository for local development",
				Action: func(cCtx *cli.Context) error {
					setup.RunSetup()
					return nil
				},
			},
			{
				Name:    "init",
				Aliases: []string{"i"},
				Usage:   "Initialize a new project",
				Action: func(cCtx *cli.Context) error {
					fmt.Println("Initialize repository command called", cCtx.Args().First())
					return nil
				},
			},
			{
				Name:    "configure",
				Aliases: []string{"c"},
				Usage:   "Configure an exiting project",
				Action: func(cCtx *cli.Context) error {
					fmt.Println("completed task", cCtx.Args().First())
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
