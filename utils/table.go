package utils

import (
	"os"

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
