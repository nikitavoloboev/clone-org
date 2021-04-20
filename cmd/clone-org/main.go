package main

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/caarlos0/clone-org/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli"
)

var version = "master"

func main() {
	app := cli.NewApp()
	app.Name = "clone-org"
	app.Usage = "Clone all repos of a github organization"
	app.Version = version
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name: "org, o",
			Usage: "organization to clone",
		},
		cli.StringFlag{
			Name:   "token, t",
			EnvVar: "GITHUB_TOKEN",
			Usage: "github token to use to authenticate and gather the repository list",
		},
		cli.StringFlag{
			Name: "destination, d",
			Usage: "path to clone the repositories into",
		},
		cli.BoolFlag{
			Name: "no-tui",
			Usage: "disable the TUI and use plain text output only",
		},
	}
	app.Action = func(c *cli.Context) error {
		log.SetFlags(0)

		token := c.String("token")
		if token == "" {
			return cli.NewExitError("missing github token", 1)
		}

		org := c.String("org")
		if org == "" {
			return cli.NewExitError("missing organization name", 1)
		}

		destination := c.String("destination")
		if destination == "" {
			destination = filepath.Join(os.TempDir(), org)
		}

		var opts []tea.ProgramOption
		isTUI := isatty.IsTerminal(os.Stdout.Fd()) && !c.Bool("no-tui")
		if isTUI {
			log.SetOutput(io.Discard)
		} else {
			opts = []tea.ProgramOption{tea.WithoutRenderer()}
		}

		p := tea.NewProgram(ui.NewInitialModel(token, org, destination,isTUI) , opts...)
		if isTUI {
			p.EnterAltScreen()
			defer p.ExitAltScreen()
		}
		if err := p.Start(); err != nil {
			return cli.NewExitError(err.Error(), 1)
		}
		return nil
	}
	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}
