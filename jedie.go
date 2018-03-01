package main

import (
	"os"

	"github.com/urfave/cli"
)

var (
	app = cli.NewApp()
	cfg config
)

func main() {
	app.Name = "jedie"
	app.Usage = "Static site generator written in golang"
	app.Version = "0.0.1"
	app.Run(os.Args)
}
