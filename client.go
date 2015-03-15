package main

type ClientCommand struct {
	Meta
}

func (c *ClientCommand) Help() string {
	return "todo"
}

func (c *ClientCommand) Synopsis() string {
	return "TODO"
}

func (c *ClientCommand) Run(args []string) int {
	return 0
}
