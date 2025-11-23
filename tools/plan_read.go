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

func PlanRead(data *ToolData) (string, error) {
	planReadInput := PlanReadInput{}

	err := json.Unmarshal(data.Input, &planReadInput)
	if err != nil {
		return "", err
	}

	switch planReadInput.Action {
	case ActionInspect:
		return handleInspect(data)
	case ActionGetNextStep:
		return handleGetNextStep(data)
	case ActionIsCompleted:
		return handleIsCompleted(data)
	default:
		return "", fmt.Errorf("plan_read: unknown action '%s'", planReadInput.Action)
	}
}

func handleInspect(data *ToolData) (string, error) {
	// This means the agent in conversation A session can read the plan from conversation B given the opportunity.
	// Do we want this?
	plan, err := data.Client.GetPlan(data.ConversationID)
	if err != nil {
		return "", fmt.Errorf("plan_read: failed to get plan with conversation ID '%s': %w", data.ConversationID, err)
	}
	return plan.Inspect(), nil
}

func handleGetNextStep(data *ToolData) (string, error) {
	plan, err := data.GetPlan(data.ConversationID)
	if err != nil {
		return "", fmt.Errorf("plan_read: failed to get plan for conversation with ID '%s': %w", data.ConversationID, err)
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

func handleIsCompleted(data *ToolData) (string, error) {
	plan, err := data.GetPlan(data.ConversationID)
	if err != nil {
		return "", fmt.Errorf("plan_read: failed to get plan for conversation with ID '%s': %w", data.ConversationID, err)
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
