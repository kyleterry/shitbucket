package server

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mitchellh/cli"
)

type ServerCommand struct {
	Ui cli.Ui
}

func (c *ServerCommand) Help() string {
	return "todo"
}

func (c *ServerCommand) Synopsis() string {
	return "TODO"
}

func (c *ServerCommand) Run(args []string) int {

	return 0
}
