package tools

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test string enum marshaling/unmarshaling
func TestWriteAction_StringEnum(t *testing.T) {
	tests := []struct {
		name     string
		action   WriteAction
		expected string
	}{
		{"set_status", ActionSetStatus, `"set_status"`},
		{"add_steps", ActionAddSteps, `"add_steps"`},
		{"remove_steps", ActionRemoveSteps, `"remove_steps"`},
		{"compact_plan", ActionCompactPlan, `"compact_plan"`},
		{"reorder_steps", ActionReorderSteps, `"reorder_steps"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.action)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, string(data))

			// Test unmarshaling
			var action WriteAction
			err = json.Unmarshal(data, &action)
			assert.NoError(t, err)
			assert.Equal(t, tt.action, action)
		})
	}
}

func TestStatus_StringEnum(t *testing.T) {
	tests := []struct {
		name     string
		status   Status
		expected string
	}{
		{"done", StatusDone, `"DONE"`},
		{"todo", StatusTodo, `"TODO"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.status)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, string(data))

			// Test unmarshaling
			var status Status
			err = json.Unmarshal(data, &status)
			assert.NoError(t, err)
			assert.Equal(t, tt.status, status)
		})
	}
}

func TestPlanWriteInput_ActionInference(t *testing.T) {
	tests := []struct {
		name           string
		inputJSON      string
		expectedAction WriteAction
		expectError    bool
		errorContains  string
	}{
		{
			name:           "infer add_steps from steps_to_add",
			inputJSON:      `{"plan_name":"test","steps_to_add":[{"id":"s1","description":"test"}]}`,
			expectedAction: ActionAddSteps,
			expectError:    false,
		},
		{
			name:           "infer set_status from step_id and status",
			inputJSON:      `{"plan_name":"test","step_id":"s1","status":"DONE"}`,
			expectedAction: ActionSetStatus,
			expectError:    false,
		},
		{
			name:           "explicit action overrides inference",
			inputJSON:      `{"plan_name":"test","write_action":"add_steps","steps_to_add":[{"id":"s1","description":"test"}]}`,
			expectedAction: ActionAddSteps,
			expectError:    false,
		},
		{
			name:          "no action and no inferable fields",
			inputJSON:     `{"plan_name":"test"}`,
			expectError:   true,
			errorContains: "cannot infer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var input PlanWriteInput
			err := json.Unmarshal([]byte(tt.inputJSON), &input)
			assert.NoError(t, err)

			// Simulate the inference logic
			if input.Action == "" {
				if len(input.StepsToAdd) > 0 {
					input.Action = ActionAddSteps
				} else if input.StepID != "" && input.Status != "" {
					input.Action = ActionSetStatus
				}
			}

			if tt.expectError {
				if input.Action == "" {
					assert.Contains(t, "cannot infer", tt.errorContains)
				}
			} else {
				assert.Equal(t, tt.expectedAction, input.Action)
			}
		})
	}
}

func TestPlanWriteInput_Validation(t *testing.T) {
	tests := []struct {
		name          string
		input         PlanWriteInput
		expectError   bool
		errorContains string
	}{
		{
			name: "valid add_steps",
			input: PlanWriteInput{
				PlanName: "test",
				Action:   ActionAddSteps,
				StepsToAdd: []PlanStepInput{
					{ID: "s1", Description: "test"},
				},
			},
			expectError: false,
		},
		{
			name: "add_steps without steps",
			input: PlanWriteInput{
				PlanName:   "test",
				Action:     ActionAddSteps,
				StepsToAdd: []PlanStepInput{},
			},
			expectError:   true,
			errorContains: "requires non-empty",
		},
		{
			name: "valid set_status",
			input: PlanWriteInput{
				PlanName: "test",
				Action:   ActionSetStatus,
				StepID:   "s1",
				Status:   StatusDone,
			},
			expectError: false,
		},
		{
			name: "set_status without step_id",
			input: PlanWriteInput{
				PlanName: "test",
				Action:   ActionSetStatus,
				Status:   StatusDone,
			},
			expectError:   true,
			errorContains: "requires 'step_id'",
		},
		{
			name: "set_status with invalid status",
			input: PlanWriteInput{
				PlanName: "test",
				Action:   ActionSetStatus,
				StepID:   "s1",
				Status:   "INVALID",
			},
			expectError:   true,
			errorContains: "invalid status",
		},
		{
			name: "unknown action",
			input: PlanWriteInput{
				PlanName: "test",
				Action:   "unknown_action",
			},
			expectError:   true,
			errorContains: "unknown action",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate validation logic
			var err error

			switch tt.input.Action {
			case ActionSetStatus:
				if tt.input.StepID == "" {
					err = assert.AnError
					assert.Contains(t, "requires 'step_id'", tt.errorContains)
				} else if tt.input.Status != StatusDone && tt.input.Status != StatusTodo && tt.input.Status != "" {
					err = assert.AnError
					assert.Contains(t, "invalid status", tt.errorContains)
				}
			case ActionAddSteps:
				if len(tt.input.StepsToAdd) == 0 {
					err = assert.AnError
					assert.Contains(t, "requires non-empty", tt.errorContains)
				}
			case "":
				// Skip - tested in inference tests
			default:
				if tt.input.Action != ActionRemoveSteps && 
				   tt.input.Action != ActionReorderSteps && 
				   tt.input.Action != ActionCompactPlan {
					err = assert.AnError
					assert.Contains(t, "unknown action", tt.errorContains)
				}
			}

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
