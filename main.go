package main

import (
	_ "embed"
	"os"

	"github.com/honganh1206/clue/cmd"
)

func main() {
	cli := cmd.NewCLI()
	err := cli.Execute()
	if err != nil {
		os.Exit(1)
	}
}
