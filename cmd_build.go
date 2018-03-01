package main

import (
	"github.com/urfave/cli"
)

func init() {
	app.Commands = append(app.Commands, cli.Command{
		Name:    "build",
		Aliases: []string{"b"},
		Usage:   "Build your site",
		Action: func(c *cli.Context) error {
			if err := cfg.load("_config.yml"); err != nil {
				return err
			}
			return cfg.Build()
		},
	})
}
