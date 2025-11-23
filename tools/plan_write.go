package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/honganh1206/clue/schema"
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
	ActionCompactPlan  WriteAction = "compact_plan"
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
	PlanName        string          `json:"plan_name" jsonschema_description:"The name of the plan to manage (e.g., 'main', 'feature-x'). This corresponds to the unique ID in the plans database."`
	Action          WriteAction     `json:"write_action" jsonschema_description:"The write operation to perform: 'add_steps', 'set_status', 'remove_steps', 'reorder_steps', or 'compact_plan'."`
	StepID          string          `json:"step_id,omitempty" jsonschema_description:"The ID of the step to target (required for 'set_status')."`
	Status          string          `json:"status,omitempty" jsonschema_description:"The status to set: 'DONE' or 'TODO' (required for 'set_status')."`
	StepsToAdd      []PlanStepInput `json:"steps_to_add,omitempty" jsonschema_description:"A list of step objects to add to the plan (required for 'add_steps'), creating it if necessary."`
	StepIDsToRemove []string        `json:"step_ids_to_remove,omitempty" jsonschema_description:"A list of step IDs to remove from the plan (required for 'remove_steps')."`
	NewStepOrder    []string        `json:"new_step_order,omitempty" jsonschema_description:"A list of step IDs representing the desired new order (required for 'reorder_steps'). Steps not in this list are appended at the end."`
}

var PlanWriteInputSchema = schema.Generate[PlanWriteInput]()

// TODO: We might be using the same client with the agent here
// so when we create a transaction, the agent is already using the client to save conversation
func PlanWrite(data *ToolData) (string, error) {
	planWriteInput := PlanWriteInput{}

	err := json.Unmarshal(data.Input, &planWriteInput)
	if err != nil {
		return "", err
	}

	planName := planWriteInput.PlanName

	if planName == "" {
		return "", fmt.Errorf("plan_write: missing or invalid plan_name")
	}

	switch planWriteInput.Action {
	case ActionSetStatus:
		// TODO: Passing the entire data struct is kind dumb
		return handleSetStatus(&planWriteInput, data, planName)
	case ActionAddSteps:
		return handleAddSteps(&planWriteInput, data, planName)
	default:
		return "", fmt.Errorf("plan_write: unknown action '%s'", planWriteInput.Action)
	}
}

func handleSetStatus(input *PlanWriteInput, data *ToolData, planName string) (string, error) {
	// TODO: If we tell the agent to mark all steps as done here (meaning not tell it explicitly the names of the steps to mark)
	// then the agent will generate new step names.
	// This leads to mismatches between those names and step names from the DB, and eventually an error.
	// Can the context window of the agent handle that?
	stepID := input.StepID
	status := input.Status
	p, err := data.Client.GetPlanByName(planName)
	if err != nil {
		return "", fmt.Errorf("plan_write: failed to get plan '%s' for set_status: %w", planName, err)
	}

	if status == "DONE" {
		err = p.MarkStepAsCompleted(stepID)
	} else {
		err = p.MarkStepAsIncomplete(stepID)
	}

	if err != nil {
		return "", fmt.Errorf("plan_write: failed to set status for step '%s' in plan '%s': %w", stepID, planName, err)
	}

	// Persist the change to the plan (including the updated step status)
	if err = data.Client.SavePlan(p); err != nil {
		return "", fmt.Errorf("plan_write: failed to save plan '%s' after setting status: %w", planName, err)
	}

	return fmt.Sprintf("Step '%s' in plan '%s' set to '%s'.", stepID, planName, status), nil
}

func handleAddSteps(input *PlanWriteInput, data *ToolData, planName string) (string, error) {
	stepsToAdd := input.StepsToAdd

	p, err := data.Client.GetPlan(planName)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			// TODO: Fetch the current conversation from client object to write a plan to it
			// should we reuse the same client object?
			p, err = data.Client.CreatePlan(planName, data.ToolMetadata.ConversationID)
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

	if err := data.Client.SavePlan(p); err != nil {
		// 500 here? is it because of create plan and add steps in the same transaction??
		return "", fmt.Errorf("plan_write: failed to save updated plan '%s': %w", planName, err)
	}
	return fmt.Sprintf("Added %d steps to plan '%s'.", addedCount, planName), nil
}
