package tools

import (
	"encoding/json"
	"testing"

	"github.com/honganh1206/clue/server/api"
	"github.com/honganh1206/clue/server/data"
	"github.com/stretchr/testify/assert"
)

// Helper functions for plan_write tests

func createTestAPIClient(t *testing.T) *api.Client {
	t.Helper()

	client := api.NewClient("")
	// if err != nil {
	// 	t.Fatalf("Failed to create test API client: %v", err)
	// }

	return client
}

func createToolInput(inputJSON []byte) ToolInput {
	return ToolInput{
		RawInput: inputJSON,
		ToolObject: &ToolObject{
			Plan: &data.Plan{
				ID:             "test-plan",
				ConversationID: "test-conversation",
				Steps:          []*data.Step{},
			},
		},
	}
}

// Tests for PlanWrite function - ActionAddSteps
func TestPlanWrite_AddSteps_Success(t *testing.T) {
	t.Skip("Requires running API server")

	input := PlanWriteInput{
		Action: ActionAddSteps,
		StepsToAdd: []PlanStepInput{
			{
				ID:          "step-1",
				Description: "First test step",
				AcceptanceCriteria: []string{
					"Criterion 1",
					"Criterion 2",
				},
			},
		},
	}
	inputJSON, _ := json.Marshal(input)

	result, err := PlanWrite(createToolInput(inputJSON))

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Added 1 steps to plan 'test-plan'")
}

func TestPlanWrite_AddSteps_MultipleSteps(t *testing.T) {
	t.Skip("Requires running API server")

	input := PlanWriteInput{
		Action: ActionAddSteps,
		StepsToAdd: []PlanStepInput{
			{
				ID:          "step-1",
				Description: "First step",
			},
			{
				ID:          "step-2",
				Description: "Second step",
			},
			{
				ID:          "step-3",
				Description: "Third step",
			},
		},
	}
	inputJSON, _ := json.Marshal(input)

	result, err := PlanWrite(createToolInput(inputJSON))

	assert.NoError(t, err)
	assert.Contains(t, result, "Added 3 steps")
}

func TestPlanWrite_AddSteps_MissingStepID(t *testing.T) {
	t.Skip("Requires running API server")

	input := PlanWriteInput{
		Action: ActionAddSteps,
		StepsToAdd: []PlanStepInput{
			{
				ID:          "",
				Description: "Step without ID",
			},
		},
	}
	inputJSON, _ := json.Marshal(input)

	result, err := PlanWrite(createToolInput(inputJSON))

	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "missing 'id' in step at index")
}

func TestPlanWrite_AddSteps_MissingDescription(t *testing.T) {
	t.Skip("Requires running API server")

	input := PlanWriteInput{
		Action: ActionAddSteps,
		StepsToAdd: []PlanStepInput{
			{
				ID:          "step-1",
				Description: "",
			},
		},
	}
	inputJSON, _ := json.Marshal(input)

	result, err := PlanWrite(createToolInput(inputJSON))

	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "missing 'description'")
}

// Tests for PlanWrite function - ActionSetStatus
func TestPlanWrite_SetStatus_ToDone(t *testing.T) {
	t.Skip("Requires running API server")

	input := PlanWriteInput{
		Action: ActionSetStatus,
		StepID: "step-1",
		Status: "DONE",
	}
	inputJSON, _ := json.Marshal(input)

	result, err := PlanWrite(createToolInput(inputJSON))

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Step 'step-1' in plan 'test-plan' set to")
}

func TestPlanWrite_SetStatus_ToTodo(t *testing.T) {
	t.Skip("Requires running API server")

	input := PlanWriteInput{
		Action: ActionSetStatus,
		StepID: "step-1",
		Status: "TODO",
	}
	inputJSON, _ := json.Marshal(input)

	result, err := PlanWrite(createToolInput(inputJSON))

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestPlanWrite_SetStatus_MissingStepID(t *testing.T) {
	t.Skip("Requires running API server")

	input := PlanWriteInput{
		Action: ActionSetStatus,
		StepID: "",
		Status: "DONE",
	}
	inputJSON, _ := json.Marshal(input)

	result, err := PlanWrite(createToolInput(inputJSON))

	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "'set_status' requires 'step_id'")
}

func TestPlanWrite_SetStatus_NonexistentPlan(t *testing.T) {
	t.Skip("Requires running API server")

	input := PlanWriteInput{
		Action: ActionSetStatus,
		StepID: "step-1",
		Status: "DONE",
	}
	inputJSON, _ := json.Marshal(input)

	result, err := PlanWrite(createToolInput(inputJSON))

	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "failed to get plan")
}

// Tests for error cases
func TestPlanWrite_EmptyPlanName(t *testing.T) {
	t.Skip("Requires running API server")
	input := PlanWriteInput{
		Action: ActionAddSteps,
	}
	inputJSON, _ := json.Marshal(input)

	result, err := PlanWrite(createToolInput(inputJSON))

	assert.Error(t, err)
	assert.Empty(t, result)
	// Error message depends on implementation
}

func TestPlanWrite_InvalidJSON(t *testing.T) {
	invalidJSON := []byte(`{"plan_name": invalid json}`)

	result, err := PlanWrite(ToolInput{RawInput: invalidJSON})

	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestPlanWrite_UnknownAction(t *testing.T) {
	input := PlanWriteInput{
		Action: WriteAction("invalid_action"), // Invalid action
	}
	inputJSON, _ := json.Marshal(input)

	result, err := PlanWrite(createToolInput(inputJSON))

	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "unknown action")
}

// Tests for PlanWriteDefinition global variable
func TestPlanWriteDefinition_Structure(t *testing.T) {
	assert.Equal(t, "plan_write", PlanWriteDefinition.Name)
	assert.NotEmpty(t, PlanWriteDefinition.Description)
	assert.NotNil(t, PlanWriteDefinition.InputSchema)
	assert.NotNil(t, PlanWriteDefinition.Function)
}

// Tests for PlanStepInput struct
func TestPlanStepInput_JSONMarshaling(t *testing.T) {
	input := PlanStepInput{
		ID:          "test-step",
		Description: "Test description",
		AcceptanceCriteria: []string{
			"Criterion 1",
			"Criterion 2",
		},
	}

	data, err := json.Marshal(input)
	assert.NoError(t, err)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"id":"test-step"`)
	assert.Contains(t, jsonStr, `"description":"Test description"`)
	assert.Contains(t, jsonStr, `"acceptance_criteria"`)
}

func TestPlanStepInput_JSONMarshalingWithoutCriteria(t *testing.T) {
	input := PlanStepInput{
		ID:          "test-step",
		Description: "Test description",
	}

	data, err := json.Marshal(input)
	assert.NoError(t, err)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"id":"test-step"`)
	assert.Contains(t, jsonStr, `"description":"Test description"`)
	// AcceptanceCriteria should be omitted when empty due to omitempty tag
	assert.NotContains(t, jsonStr, `"acceptance_criteria"`)
}

func TestPlanStepInput_JSONUnmarshaling(t *testing.T) {
	jsonData := `{"id":"step-1","description":"Test step","acceptance_criteria":["Criterion 1"]}`

	var input PlanStepInput
	err := json.Unmarshal([]byte(jsonData), &input)

	assert.NoError(t, err)
	assert.Equal(t, "step-1", input.ID)
	assert.Equal(t, "Test step", input.Description)
	assert.Len(t, input.AcceptanceCriteria, 1)
	assert.Equal(t, "Criterion 1", input.AcceptanceCriteria[0])
}

// Tests for PlanWriteInput struct
func TestPlanWriteInput_JSONMarshaling(t *testing.T) {
	input := PlanWriteInput{
		Action: ActionAddSteps,
		StepsToAdd: []PlanStepInput{
			{
				ID:          "step-1",
				Description: "First step",
			},
		},
	}

	data, err := json.Marshal(input)
	assert.NoError(t, err)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"write_action":"add_steps"`)
}

func TestPlanWriteInput_JSONUnmarshaling(t *testing.T) {
	jsonData := `{
		"write_action":"add_steps",
		"steps_to_add":[
			{"id":"step-1","description":"Test step"}
		]
	}`

	var input PlanWriteInput
	err := json.Unmarshal([]byte(jsonData), &input)

	assert.NoError(t, err)
	assert.Equal(t, ActionAddSteps, input.Action)
	assert.Len(t, input.StepsToAdd, 1)
	assert.Equal(t, "step-1", input.StepsToAdd[0].ID)
}

// Tests for WriteAction values
func TestWriteAction_Values(t *testing.T) {
	assert.Equal(t, "set_status", string(ActionSetStatus))
	assert.Equal(t, "add_steps", string(ActionAddSteps))
	assert.Equal(t, "remove_steps", string(ActionRemoveSteps))
	assert.Equal(t, "reorder_steps", string(ActionReorderSteps))
}

// Table-driven tests
func TestPlanWrite_VariousInputs(t *testing.T) {
	t.Skip("Requires running API server")

	tests := []struct {
		name        string
		input       PlanWriteInput
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid add steps",
			input: PlanWriteInput{
				Action: ActionAddSteps,
				StepsToAdd: []PlanStepInput{
					{ID: "step-1", Description: "Test step"},
				},
			},
			expectError: false,
		},
		{
			name: "missing plan name",
			input: PlanWriteInput{
				Action: ActionAddSteps,
			},
			expectError: true,
			errorMsg:    "missing or invalid plan_name",
		},
		{
			name: "set status without step id",
			input: PlanWriteInput{
				Action: ActionSetStatus,
				StepID: "",
				Status: "DONE",
			},
			expectError: true,
			errorMsg:    "requires 'step_id'",
		},
		{
			name: "add step without id",
			input: PlanWriteInput{
				Action: ActionAddSteps,
				StepsToAdd: []PlanStepInput{
					{ID: "", Description: "Test"},
				},
			},
			expectError: true,
			errorMsg:    "missing 'id'",
		},
		{
			name: "add step without description",
			input: PlanWriteInput{
				Action: ActionAddSteps,
				StepsToAdd: []PlanStepInput{
					{ID: "step-1", Description: ""},
				},
			},
			expectError: true,
			errorMsg:    "missing 'description'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputJSON, _ := json.Marshal(tt.input)

			result, err := PlanWrite(createToolInput(inputJSON))

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, result)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, result)
			}
		})
	}
}

// Benchmark tests
func BenchmarkPlanWrite_AddSteps(b *testing.B) {
	b.Skip("Requires running API server")

	input := PlanWriteInput{
		Action: ActionAddSteps,
		StepsToAdd: []PlanStepInput{
			{ID: "step-1", Description: "Benchmark step"},
		},
	}
	inputJSON, _ := json.Marshal(input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		PlanWrite(createToolInput(inputJSON))
	}
}

func BenchmarkPlanWrite_SetStatus(b *testing.B) {
	b.Skip("Requires running API server")

	input := PlanWriteInput{
		Action: ActionSetStatus,
		StepID: "step-1",
		Status: "DONE",
	}
	inputJSON, _ := json.Marshal(input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		PlanWrite(createToolInput(inputJSON))
	}
}

