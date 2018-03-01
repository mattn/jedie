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
			if c.String("s") != "" {
				cfg.Source = c.String("s")
			}
			if c.String("d") != "" {
				cfg.Destination = c.String("d")
			}
			return cfg.Build()
		},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "s",
				Usage: "source path",
			},
			cli.StringFlag{
				Name:  "d",
				Usage: "destination path",
			},
		},
	})
}
