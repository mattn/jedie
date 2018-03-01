package main

import (
	"github.com/urfave/cli"
)

func init() {
	app.Commands = append(app.Commands, cli.Command{
		Name:    "newpost",
		Aliases: []string{"p"},
		Usage:   "Create new post",
		Action: func(c *cli.Context) error {
			if !c.Args().Present() {
				cli.ShowCommandHelp(c, "newpost")
				return nil
			}
			if err := cfg.load("_config.yml"); err != nil {
				return err
			}
			return cfg.NewPost(c.Args().First())
		},
	})
}
