package prompts

import (
	_ "embed"
	"os"
	"path/filepath"
)

func System() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return filepath.Join(wd, "prompts", "system.txt")
}
