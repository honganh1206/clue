package tools

import (
	"encoding/json"
	"fmt"

	"github.com/honganh1206/tinker/schema"
	"github.com/honganh1206/tinker/server/data"
)

var PlanWriteDefinition = ToolDefinition{
	Name:        ToolNamePlanWrite,
	Description: "Update the plan for the current session. To be used proactively and often to track progress and pending steps.",
	InputSchema: PlanWriteInputSchema,
	Function:    PlanWrite,
}

type WriteAction string

const (
	ActionSetStatus    WriteAction = "set_status"
	ActionAddSteps     WriteAction = "add_steps"
	ActionRemoveSteps  WriteAction = "remove_steps"
	ActionReorderSteps WriteAction = "reorder_steps"
)

type PlanStepInput struct {
	ID                 string   `json:"id" jsonschema_description:"A short, unique identifier for the step (e.g., 'add-tests')."`
	Status             string   `json:"status" jsonschema_description:"The status to set: 'DONE' or 'TODO'."`
	Description        string   `json:"description" jsonschema_description:"A detailed description of the step's task."`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty" jsonschema_description:"A list of criteria that must be met for the step to be considered DONE."`
}

var PlanStepSchema = schema.Generate[PlanStepInput]()

type PlanWriteInput struct {
	Action          WriteAction     `json:"write_action" jsonschema_description:"The write operation to perform: 'add_steps', 'set_status', 'remove_steps', 'reorder_steps'."`
	StepID          string          `json:"step_id,omitempty" jsonschema_description:"The ID of the step to target (required for 'set_status')."`
	Status          string          `json:"status,omitempty" jsonschema_description:"The status to set: 'DONE' or 'TODO' (required for 'set_status')."`
	StepsToAdd      []PlanStepInput `json:"steps_to_add,omitempty" jsonschema_description:"A list of step objects to add to the plan (required for 'add_steps'), creating it if necessary."`
	StepIDsToRemove []string        `json:"step_ids_to_remove,omitempty" jsonschema_description:"A list of step IDs to remove from the plan (required for 'remove_steps')."`
	NewStepOrder    []string        `json:"new_step_order,omitempty" jsonschema_description:"A list of step IDs representing the desired new order (required for 'reorder_steps'). Steps not in this list are appended at the end."`
}

var PlanWriteInputSchema = schema.Generate[PlanWriteInput]()

func PlanWrite(input ToolInput) (string, error) {
	planWriteInput := PlanWriteInput{}

	err := json.Unmarshal(input.RawInput, &planWriteInput)
	if err != nil {
		return "", fmt.Errorf("error when unmarshalling raw input: %w", err)
	}

	switch planWriteInput.Action {
	case ActionSetStatus:
		output, err := handleSetStatus(&planWriteInput, input.ToolObject.Plan)
		if err != nil {
			return "", fmt.Errorf("error when setting status: %w", err)
		}
		return output, nil
	case ActionAddSteps:
		output, err := handleAddSteps(&planWriteInput, input.ToolObject.Plan)
		if err != nil {
			return "", fmt.Errorf("error when adding steps: %w", err)
		}
		return output, nil
	case ActionRemoveSteps:
		output, err := handleRemoveSteps(&planWriteInput, input.ToolObject.Plan)
		if err != nil {
			return "", fmt.Errorf("error when removing steps: %w", err)
		}
		return output, nil
	case ActionReorderSteps:
		output, err := handleReorderSteps(&planWriteInput, input.ToolObject.Plan)
		if err != nil {
			return "", fmt.Errorf("error when reordering steps: %w", err)
		}
		return output, nil

	default:
		return "", fmt.Errorf("plan_write: unknown action '%s'", planWriteInput.Action)
	}
}

func handleSetStatus(input *PlanWriteInput, plan *data.Plan) (string, error) {
	if plan == nil {
		return "", fmt.Errorf("plan is nil in handleSetStatus")
	}

	stepID := input.StepID
	status := input.Status

	var err error
	if status == "DONE" {
		err = plan.MarkStepAsCompleted(stepID)
	} else {
		err = plan.MarkStepAsIncomplete(stepID)
	}

	if err != nil {
		return "", fmt.Errorf("plan_write: failed to set status for step '%s' in plan '%s': %w", stepID, plan.ID, err)
	}

	return fmt.Sprintf("Step '%s' in plan '%s' set to '%s'.", stepID, plan.ID, status), nil
}

func handleAddSteps(input *PlanWriteInput, plan *data.Plan) (string, error) {
	stepsToAdd := input.StepsToAdd

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

		plan.AddStep(id, description, criteria)
		addedCount++
	}

	return fmt.Sprintf("Added %d steps to plan ID '%s'.", addedCount, plan.ID), nil
}

func handleRemoveSteps(input *PlanWriteInput, plan *data.Plan) (string, error) {
	stepIDsToRemove := input.StepIDsToRemove
	if len(stepIDsToRemove) == 0 {
		return "", fmt.Errorf("plan_write: 'remove_steps' action requires 'step_ids_to_remove' list")
	}

	removedCount := plan.RemoveSteps(stepIDsToRemove)

	return fmt.Sprintf("Removed %d steps from plan '%s'.", removedCount, plan.ID), nil
}

// TODO: This uses PlannerModel, not Plan struct
// so it must be made by the Client, not the tool
//
// func handleCompactPlans(input *PlanWriteInput, plan *data.Plan) (string, error) {
// 	err := plan.Compact() // This is the call to the method added in planner.go
// 	if err != nil {
// 		// The Compact method in planner.go logs warnings for individual file errors
// 		// but returns an error if the directory read fails.
// 		return "", fmt.Errorf("plan_write: 'compact_plans' action encountered an error: %w", err)
// 	}
//
// 	return fmt.Sprintf("Removed %d steps from plan '%s'.", removedCount, plan.ID), nil
// }

func handleReorderSteps(input *PlanWriteInput, plan *data.Plan) (string, error) {
	stepIDsToReorder := input.StepIDsToRemove
	if len(stepIDsToReorder) == 0 {
		return "", fmt.Errorf("plan_write: 'remove_steps' action requires 'step_ids_to_remove' list")
	}

	plan.ReorderSteps(stepIDsToReorder)

	return fmt.Sprintf("Removed steps from plan '%s'.", plan.ID), nil
}
