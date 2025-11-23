package tools

import (
	"encoding/json"
	"fmt"

	"github.com/honganh1206/clue/schema"
)

var PlanReadDefinition = ToolDefinition{
	Name:        ToolNamePlanRead,
	Description: "Fetch evelopment plans. Use this tool to inspect and query the status of plans and their steps. Always specify the plan name.",
	InputSchema: PlanReadInputSchema,
	Function:    PlanRead,
}

type ReadAction string

const (
	ActionInspect     ReadAction = "inspect"
	ActionGetNextStep ReadAction = "get_next_step"
	ActionIsCompleted ReadAction = "is_completed"
)

type PlanReadInput struct {
	PlanName string     `json:"plan_name" jsonschema_description:"The name of the plan to manage (e.g., 'main', 'feature-x'). This corresponds to the unique ID in the plans database."`
	Action   ReadAction `json:"read_action" jsonschema_description:"The read operation to perform on the plan: 'inspect', 'get_next_step' or 'is_completed'."`
}

var PlanReadInputSchema = schema.Generate[PlanReadInput]()

func PlanRead(data *ToolData) (string, error) {
	planReadInput := PlanReadInput{}

	err := json.Unmarshal(data.Input, &planReadInput)
	if err != nil {
		return "", err
	}

	planName := planReadInput.PlanName

	if planName == "" {
		return "", fmt.Errorf("plan_read: missing or invalid plan_name")
	}

	switch planReadInput.Action {
	case ActionInspect:
		return handleInspect(data, planName)
	case ActionGetNextStep:
		return handleGetNextStep(data, planName)
	case ActionIsCompleted:
		return handleIsCompleted(data, planName)
	default:
		return "", fmt.Errorf("plan_read: unknown action '%s'", planReadInput.Action)
	}
}

func handleInspect(data *ToolData, planName string) (string, error) {
	// This means the agent in conversation A session can read the plan from conversation B given the opportunity.
	// Do we want this?
	plan, err := data.Client.GetPlanByName(planName)
	if err != nil {
		return "", fmt.Errorf("plan_read: failed to get plan '%s': %w", planName, err)
	}
	return plan.Inspect(), nil
}

func handleGetNextStep(data *ToolData, planName string) (string, error) {
	plan, err := data.GetPlanByName(planName)
	if err != nil {
		return "", fmt.Errorf("plan_read: failed to get plan '%s': %w", planName, err)
	}
	next := plan.NextStep()
	if next == nil {
		return "plan is completed", nil
	} else {
		// Are we going to format it to JSON or just string?
		resp := map[string]any{
			"next_step": map[string]any{
				"id":                  next.GetID(),
				"status":              next.GetStatus(),
				"description":         next.GetDescription(),
				"acceptance_criteria": next.GetAcceptanceCriteria(),
			},
		}

		b, err := json.Marshal(resp) // or json.MarshalIndent(resp, "", "  ") for pretty output
		if err != nil {
			return "", fmt.Errorf("plan_read: failed to marshal response to JSON: %w", err)
		}
		return string(b), nil
	}
}

func handleIsCompleted(data *ToolData, planName string) (string, error) {
	plan, err := data.GetPlanByName(planName)
	if err != nil {
		return "", fmt.Errorf("plan_read: failed to get plan '%s': %w", planName, err)
	}
	isCompleted := plan.IsCompleted()
	// Are we going to format it to JSON or just string?
	resp := map[string]any{
		"is_completed": isCompleted,
	}

	b, err := json.Marshal(resp) // or json.MarshalIndent(resp, "", "  ") for pretty output
	if err != nil {
		return "", fmt.Errorf("plan_read: failed to marshal response to JSON: %w", err)
	}
	return string(b), nil
}
