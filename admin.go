package main

type AdminCommand struct {
	Meta
}

func (c *AdminCommand) Help() string {
	return "todo"
}

func (c *AdminCommand) Synopsis() string {
	return "TODO"
}

func (c *AdminCommand) Run(args []string) int {
	return 0
}
