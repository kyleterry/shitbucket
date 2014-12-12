package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
	_"github.com/mxk/go-sqlite/sqlite3"
	"github.com/mitchellh/cli"
)

// Structs and Vars
const (
	version = "0.1.0"
)

// Structs
type Meta struct {
	Color bool
	Ui cli.Ui
}

type AdminCommand struct {
	Meta
}

type SetupCommand struct {
	Meta
}

type RunCommand struct {
	Meta
}

// Methods for commands
func (c *AdminCommand) Run(args []string) int {
	return 1
}

func (c *AdminCommand) Help() string {
	return "Not implemented yet"
}

func (c *AdminCommand) Synopsis() string {
	return "Not implemented yet"
}

func (c *SetupCommand) Run(args []string) int {
	return 1
}

func (c *SetupCommand) Synopsis() string {

	return "Not implemented yet"
}

func (c *SetupCommand) Help() string {

	return "Not implemented yet"
}

func (c *RunCommand) Run(args []string) int {
	var bind string
	cmdFlags := flag.NewFlagSet("Run", flag.ContinueOnError)
	cmdFlags.StringVar(&bind, "bind", ":8080", "bind")
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}
	err := wrappedrun(bind)
	if err != nil{
		return 1
	}
	return 0
}

func (c *RunCommand) Synopsis() string {
	return "Starts the shitbucket web server"
}

func (c *RunCommand) Help() string {
	help := `
Usage: shitbucket run [options]
	Start a shitbucket web server.

Options:
	-bind	Set how the server should bind. E.g. -bind "localhost:9090"
			Default is ":8080".
`
	return help
}

// "actions" or whatever you want to call it
func GetUrls() string {
	return "hello"
}

func GetUrl(params martini.Params) string {
	return fmt.Sprintf("sup %s", params["id"])
}

func NewUrl() string {
	return "sup submit"
}

func UpdateUrl(params martini.Params) string {
	return fmt.Sprintf("update %s", params["id"])
}

func DeleteUrl(params martini.Params) string {
	return fmt.Sprintf("delete %s", params["id"])
}

func Auth(res http.ResponseWriter, req *http.Request) {

}


func wrappedrun(bind string) error {
	m := martini.Classic()
	m.Use(Auth)
	m.Use(martini.Static("assets"))
	m.Use(render.Renderer(render.Options{
		Directory: "templates",
		Extensions: []string{".html"},
		Charset: "UTF-8",
	}))

	// Routes
	m.Get("/", GetUrls)

	m.Group("/url", func(r martini.Router) {
		r.Get("/:id", GetUrl)
		r.Post("/submit", NewUrl)
		r.Put("/update/:id", UpdateUrl)
		r.Delete("/delete/:id", DeleteUrl)
	})

	return http.ListenAndServe(bind, m)
}

func main() {
	ui := &cli.BasicUi{Writer: os.Stdout}

	meta := Meta{
		Color: true,
		Ui: ui,
	}

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
		"run": func() (cli.Command, error) {
			return &RunCommand{
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
