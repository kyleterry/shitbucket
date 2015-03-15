package main

type SetupCommand struct {
	Meta
}

func (c *SetupCommand) Help() string {
	return "todo"
}

func (c *SetupCommand) Synopsis() string {
	return "TODO"
}

func (c *SetupCommand) Run(args []string) int {
	return 0
}
