package main

import (
	"github.com/urfave/cli"
)

func init() {
	app.Commands = append(app.Commands, cli.Command{
		Name:    "serve",
		Aliases: []string{"s"},
		Usage:   "Serve your site locally",
		Action: func(c *cli.Context) error {
			if err := cfg.load("_config.yml"); err != nil {
				return err
			}
			return cfg.Serve()
		},
	})
}
