package ui

import (
	"fmt"
)

const (
	SuccessSymbol = "✓"
	ErrorSymbol   = "✗"
)

type ToolResultFormat struct {
	Name    string
	Detail  string
	IsError bool
}

func FormatToolResult(f ToolResultFormat) string {
	if f.IsError {
		if f.Detail != "" {
			return fmt.Sprintf("[red]%s [white::-]%s [blue]%s[white::-]\n\n", ErrorSymbol, f.Name, f.Detail)
		}
		return fmt.Sprintf("[red]%s [white::-]%s\n\n", ErrorSymbol, f.Name)
	}

	if f.Detail != "" {
		return fmt.Sprintf("[green]%s [white::-]%s [blue]%s[white::-]\n\n", SuccessSymbol, f.Name, f.Detail)
	}
	return fmt.Sprintf("[green]%s [white::-]%s\n\n", SuccessSymbol, f.Name)
}