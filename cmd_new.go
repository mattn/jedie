package main

import (
	"github.com/urfave/cli"
)

func init() {
	app.Commands = append(app.Commands, cli.Command{
		Name:    "new",
		Aliases: []string{"n"},
		Usage:   "Creates a new jedie site scaffold in PATH",
		Action: func(c *cli.Context) error {
			if !c.Args().Present() {
				cli.ShowCommandHelp(c, "new")
				return nil
			}
			return cfg.New(c.Args().First())
		},
	})
}
