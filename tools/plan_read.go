package tools

import (
	"encoding/json"
	"fmt"

	"github.com/honganh1206/clue/schema"
)

var PlanReadDefinition = ToolDefinition{
	Name:        ToolNamePlanRead,
	Description: "Fetch evelopment plans. Use this tool to inspect and query the status of plans and their steps.",
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
	Action ReadAction `json:"read_action" jsonschema_description:"The read operation to perform on the plan: 'inspect', 'get_next_step' or 'is_completed'."`
}

var PlanReadInputSchema = schema.Generate[PlanReadInput]()

func PlanRead(input ToolInput) (string, error) {
	planReadInput := PlanReadInput{}

	err := json.Unmarshal(input.RawInput, &planReadInput)
	if err != nil {
		return "", err
	}

	switch planReadInput.Action {
	case ActionInspect:
		output, err := handleInspect(input)
		if err != nil {
			return "error when inspecting plan", err
		}
		return output, nil
	case ActionGetNextStep:
		output, err := handleGetNextStep(input)
		if err != nil {
			return "error when getting next step", err
		}
		return output, nil

	case ActionIsCompleted:
		output, err := handleIsCompleted(input)
		if err != nil {
			return "error when checking if step is completed", err
		}
		return output, nil

	default:
		return "", fmt.Errorf("plan_read: unknown action '%s'", planReadInput.Action)
	}
}

func handleInspect(input ToolInput) (string, error) {
	return input.Plan.Inspect(), nil
}

func handleGetNextStep(input ToolInput) (string, error) {
	next := input.Plan.NextStep()
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

func handleIsCompleted(input ToolInput) (string, error) {
	isCompleted := input.Plan.IsCompleted()
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

