package main

import (
	"tfviz-agent/cmd"
)

func main() {
	// The main function is deliberately kept minimal. 
	// It simply delegates all execution to the Cobra root command.
	cmd.Execute()
}