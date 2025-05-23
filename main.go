package main

import (
	_ "embed"

	"github.com/honganh1206/clue/commands"
)

func main() {
	cmd := commands.NewRootCmd()
	cmd.Execute()
}
