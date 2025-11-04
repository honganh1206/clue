package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/honganh1206/clue/schema"
	"github.com/honganh1206/clue/server/api"
)

var PlanWriteDefinition = ToolDefinition{
	Name:        "plan_write",
	Description: "Update development plans. Use this tool to create, modify, and query the status of plans and their steps. Always specify the plan name.",
	InputSchema: PlanWriteInputSchema,
	Function:    PlanWrite,
}

type WriteAction string

const (
	ActionSetStatus    WriteAction = "set_status"
	ActionAddSteps     WriteAction = "add_steps"
	ActionRemoveSteps  WriteAction = "remove_steps"
	ActionCompactPlan  WriteAction = "compact_plan"
	ActionReorderSteps WriteAction = "reorder_steps"
)

type Status string

const (
	StatusDone Status = "DONE"
	StatusTodo Status = "TODO"
)

type PlanStepInput struct {
	ID                 string   `json:"id" jsonschema_description:"A short, unique identifier for the step (e.g., 'add-tests')."`
	Description        string   `json:"description" jsonschema_description:"A detailed description of the step's task."`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty" jsonschema_description:"A list of criteria that must be met for the step to be considered DONE."`
}

var PlanStepSchema = schema.Generate[PlanStepInput]()

type PlanWriteInput struct {
	PlanName        string          `json:"plan_name" jsonschema_description:"The name of the plan to manage (e.g., 'main', 'feature-x'). This corresponds to the unique ID in the plans database."`
	Action          WriteAction     `json:"write_action" jsonschema_description:"The write operation to perform: 'add_steps', 'set_status', 'remove_steps', 'reorder_steps', or 'compact_plan'."`
	StepID          string          `json:"step_id,omitempty" jsonschema_description:"The ID of the step to target (required for 'set_status')."`
	Status          Status          `json:"status,omitempty" jsonschema_description:"The status to set: 'DONE' or 'TODO' (required for 'set_status')."`
	StepsToAdd      []PlanStepInput `json:"steps_to_add,omitempty" jsonschema_description:"A list of step objects to add to the plan (required for 'add_steps'), creating it if necessary."`
	StepIDsToRemove []string        `json:"step_ids_to_remove,omitempty" jsonschema_description:"A list of step IDs to remove from the plan (required for 'remove_steps')."`
	NewStepOrder    []string        `json:"new_step_order,omitempty" jsonschema_description:"A list of step IDs representing the desired new order (required for 'reorder_steps'). Steps not in this list are appended at the end."`
}

var PlanWriteInputSchema = schema.Generate[PlanWriteInput]()

// TODO: We might be using the same client with the agent here
// so when we create a transaction, the agent is already using the client to save conversation
func PlanWrite(input json.RawMessage) (string, error) {
	client := api.NewClient("") // TODO: Very temp
	planWriteInput := PlanWriteInput{}

	err := json.Unmarshal(input, &planWriteInput)
	if err != nil {
		return "", err
	}

	planName := planWriteInput.PlanName

	if planName == "" {
		return "", fmt.Errorf("plan_write: missing or invalid plan_name")
	}

	switch planWriteInput.Action {
	case ActionSetStatus:
		stepID := planWriteInput.StepID
		status := planWriteInput.Status

		plan, err := client.GetPlan(planName)
		if err != nil {
			return "", fmt.Errorf("plan_write: failed to get plan '%s' for set_status: %w", planName, err)
		}

		if status == StatusDone {
			err = plan.MarkStepAsCompleted(stepID)
		} else {
			err = plan.MarkStepAsIncomplete(stepID)
		}

		if err != nil {
			return "", fmt.Errorf("plan_write: failed to set status for step '%s' in plan '%s': %w", stepID, planName, err)
		}

		// Persist the change to the plan (including the updated step status)
		if err = client.SavePlan(plan); err != nil {
			return "", fmt.Errorf("plan_write: failed to save plan '%s' after setting status: %w", planName, err)
		}

		return fmt.Sprintf("Step '%s' in plan '%s' set to '%s'.", stepID, planName, string(status)), nil
	case ActionAddSteps:
		stepsToAdd := planWriteInput.StepsToAdd

		p, err := client.GetPlan(planName)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "not found") {
				// We need to reuse the same PlanModel object here??
				p, err = client.CreatePlan(planName)
				if err != nil {
					return "", fmt.Errorf("plan_write: failed to create new plan '%s' for adding steps: %w", planName, err)
				}
			} else {
				return "", fmt.Errorf("plan_write: failed to get plan '%s': %w", planName, err)
			}
		}

		addedCount := 0
		for i, s := range stepsToAdd {
			id := s.ID
			if id == "" {
				return "", fmt.Errorf("plan_write: missing 'id' in step at index %d", i)
			}

			description := s.Description
			if description == "" {
				return "", fmt.Errorf("plan_write: missing 'description' in step '%s' at index %d", id, i)
			}

			var criteria []string
			for _, criterion := range s.AcceptanceCriteria {
				criteria = append(criteria, criterion)
			}

			p.AddStep(id, description, criteria)
			addedCount++
		}

		if err := client.SavePlan(p); err != nil {
			// 500 here? is it because of create plan and add steps in the same transaction??
			return "", fmt.Errorf("plan_write: failed to save updated plan '%s': %w", planName, err)
		}
		return fmt.Sprintf("Added %d steps to plan '%s'.", addedCount, planName), nil
	default:
		return "", fmt.Errorf("plan_write: unknown action '%s'", planWriteInput.Action)
	}
}
