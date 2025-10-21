package utils

import (
	"os"
	"strings"
	"unicode/utf8"

	"github.com/olekukonko/tablewriter"
)

// Render output into an ASCII table
func RenderTable(headers []string, data [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header(headers)
	table.Bulk(data)
	table.Render()
	// TODO: Add configs for different data displays e.g., models, conversation, code blocks
}

func RenderBox(title string, lines []string) string {
	// Determine max line width using visual character count (runes), not byte count
	titleWidth := utf8.RuneCountInString(title)
	maxWidth := titleWidth + 4 // for padding
	for _, line := range lines {
		lineWidth := utf8.RuneCountInString(line)
		if lineWidth+2 > maxWidth {
			maxWidth = lineWidth + 2
		}
	}

	var b strings.Builder

	// Top border with title
	b.WriteString("┌─ " + title + " " + strings.Repeat("─", maxWidth-titleWidth-3) + "┐\n")

	// Content lines
	for _, line := range lines {
		lineWidth := utf8.RuneCountInString(line)
		padding := maxWidth - lineWidth - 2
		b.WriteString("│ " + line + strings.Repeat(" ", padding) + " │\n")
	}

	// Bottom border
	b.WriteString("└" + strings.Repeat("─", maxWidth) + "┘\n")

	return b.String()
}
