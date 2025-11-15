package tools

import (
	"encoding/json"

	"github.com/honganh1206/clue/schema"
)

var PlanReadDefinition = ToolDefinition{
	Name:        ToolNamePlanRead,
	Description: "Fetch evelopment plans. Use this tool to inspect and query the status of plans and their steps. Always specify the plan name.",
	InputSchema: PlanReadInputSchema,
	Function:    PlanRead,
}

type ReadAction int

const (
	// Get Markdown representation of the plan
	ActionInspect ReadAction = iota
	ActionGetNextStep
	ActionIsCompleted
	ActionListPlans
)

type PlanReadInput struct {
	PlanName string     `json:"plan_name" jsonschema_description:"The name of the plan to manage (e.g., 'main', 'feature-x'). This corresponds to the unique ID in the plans database."`
	Action   ReadAction `json:"write_action" jsonschema_description:"The read operation to perform on the plan."`
}

var PlanReadInputSchema = schema.Generate[PlanReadInput]()

func PlanRead(input json.RawMessage, meta ToolMetadata) (string, error) {
	return "", nil // temp
}
