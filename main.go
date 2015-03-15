package main

import (
	"fmt"
	"os"

	"github.com/kyleterry/shitbucket/server"
	"github.com/mitchellh/cli"
)

const (
	version = "0.1.1"
)

// Structs
type Meta struct {
	Color bool
	Ui    cli.Ui
}

func main() {
	ui := &cli.BasicUi{Writer: os.Stdout}

	meta := Meta{
		Color: true,
		Ui:    ui}

	c := cli.NewCLI("shitbucket", version)
	c.Args = os.Args[1:]
	c.Commands = map[string]cli.CommandFactory{
		"admin": func() (cli.Command, error) {
			return &AdminCommand{
				Meta: meta,
			}, nil
		},
		"setup": func() (cli.Command, error) {
			return &SetupCommand{
				Meta: meta,
			}, nil
		},
		"server": func() (cli.Command, error) {
			return &server.ServerCommand{
				Ui: ui,
			}, nil
		},
		"client": func() (cli.Command, error) {
			return &ClientCommand{
				Meta: meta,
			}, nil
		},
	}

	exitCode, err := c.Run()
	if err != nil {
		ui.Error(fmt.Sprintf("Error executing CLI: %s", err.Error()))
		os.Exit(1)
	}
	os.Exit(exitCode)
}
