package tools

import (
	"encoding/json"
	"fmt"

	"github.com/honganh1206/clue/schema"
	"github.com/honganh1206/clue/server/api"
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
	Action   ReadAction `json:"read_action" jsonschema_description:"The read operation to perform on the plan."`
}

var PlanReadInputSchema = schema.Generate[PlanReadInput]()

func PlanRead(input json.RawMessage, meta ToolMetadata) (string, error) {
	client := api.NewClient("") // TODO: Very temp
	planReadInput := PlanReadInput{}

	err := json.Unmarshal(input, &planReadInput)
	if err != nil {
		return "", err
	}

	planName := planReadInput.PlanName

	if planName == "" {
		return "", fmt.Errorf("plan_read: missing or invalid plan_name")
	}

	switch planReadInput.Action {
	case ActionInspect:
		return handleInspect(client, planName)
	default:
		return "", fmt.Errorf("plan_read: unknown action '%s'", planReadInput.Action)
	}
}

func handleInspect(client *api.Client, planName string) (string, error) {
	// TODO: This means the agent in conversation A session can read the plan from conversation B given the opportunity.
	// Do we want this?
	plan, err := client.GetPlanByName(planName)
	if err != nil {
		return "", fmt.Errorf("plan_read: failed to get plan '%s': %w", planName, err)
	}
	return plan.Inspect(), nil
}
