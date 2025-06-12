package main

import (
	"context"
	_ "embed"
	"os"

	"github.com/honganh1206/clue/app"
	"github.com/honganh1206/clue/cmd"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	app.Run(ctx, cancel)
	cli := cmd.NewCLI()
	err := cli.ExecuteContext(ctx)
	if err != nil {
		os.Exit(1)
	}
}
