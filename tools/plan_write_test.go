package tools

import (
	"encoding/json"
	"testing"

	"github.com/honganh1206/clue/api"
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

// Tests for PlanWrite function - ActionAddSteps
func TestPlanWrite_AddSteps_Success(t *testing.T) {
	// t.Skip("Requires running API server")

	client := createTestAPIClient(t)

	input := PlanWriteInput{
		PlanName: "test-plan",
		Action:   ActionAddSteps,
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

	result, err := PlanWrite(inputJSON, client)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Added 1 steps to plan 'test-plan'")
}

func TestPlanWrite_AddSteps_MultipleSteps(t *testing.T) {
	t.Skip("Requires running API server")

	client := createTestAPIClient(t)

	input := PlanWriteInput{
		PlanName: "multi-step-plan",
		Action:   ActionAddSteps,
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

	result, err := PlanWrite(inputJSON, client)

	assert.NoError(t, err)
	assert.Contains(t, result, "Added 3 steps")
}

func TestPlanWrite_AddSteps_MissingStepID(t *testing.T) {
	t.Skip("Requires running API server")

	client := createTestAPIClient(t)

	input := PlanWriteInput{
		PlanName: "test-plan",
		Action:   ActionAddSteps,
		StepsToAdd: []PlanStepInput{
			{
				ID:          "",
				Description: "Step without ID",
			},
		},
	}
	inputJSON, _ := json.Marshal(input)

	result, err := PlanWrite(inputJSON, client)

	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "missing 'id' in step at index")
}

func TestPlanWrite_AddSteps_MissingDescription(t *testing.T) {
	t.Skip("Requires running API server")

	client := createTestAPIClient(t)

	input := PlanWriteInput{
		PlanName: "test-plan",
		Action:   ActionAddSteps,
		StepsToAdd: []PlanStepInput{
			{
				ID:          "step-1",
				Description: "",
			},
		},
	}
	inputJSON, _ := json.Marshal(input)

	result, err := PlanWrite(inputJSON, client)

	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "missing 'description'")
}

// Tests for PlanWrite function - ActionSetStatus
func TestPlanWrite_SetStatus_ToDone(t *testing.T) {
	t.Skip("Requires running API server")

	client := createTestAPIClient(t)

	input := PlanWriteInput{
		PlanName: "test-plan",
		Action:   ActionSetStatus,
		StepID:   "step-1",
		Status:   StatusDone,
	}
	inputJSON, _ := json.Marshal(input)

	result, err := PlanWrite(inputJSON, client)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Step 'step-1' in plan 'test-plan' set to")
}

func TestPlanWrite_SetStatus_ToTodo(t *testing.T) {
	t.Skip("Requires running API server")

	client := createTestAPIClient(t)

	input := PlanWriteInput{
		PlanName: "test-plan",
		Action:   ActionSetStatus,
		StepID:   "step-1",
		Status:   StatusTodo,
	}
	inputJSON, _ := json.Marshal(input)

	result, err := PlanWrite(inputJSON, client)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestPlanWrite_SetStatus_MissingStepID(t *testing.T) {
	t.Skip("Requires running API server")

	client := createTestAPIClient(t)

	input := PlanWriteInput{
		PlanName: "test-plan",
		Action:   ActionSetStatus,
		StepID:   "",
		Status:   StatusDone,
	}
	inputJSON, _ := json.Marshal(input)

	result, err := PlanWrite(inputJSON, client)

	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "'set_status' requires 'step_id'")
}

func TestPlanWrite_SetStatus_NonexistentPlan(t *testing.T) {
	t.Skip("Requires running API server")

	client := createTestAPIClient(t)

	input := PlanWriteInput{
		PlanName: "nonexistent-plan",
		Action:   ActionSetStatus,
		StepID:   "step-1",
		Status:   StatusDone,
	}
	inputJSON, _ := json.Marshal(input)

	result, err := PlanWrite(inputJSON, client)

	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "failed to get plan")
}

// Tests for error cases
func TestPlanWrite_EmptyPlanName(t *testing.T) {
	client := createTestAPIClient(t)

	input := PlanWriteInput{
		PlanName: "",
		Action:   ActionAddSteps,
	}
	inputJSON, _ := json.Marshal(input)

	result, err := PlanWrite(inputJSON, client)

	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "missing or invalid plan_name")
}

func TestPlanWrite_InvalidJSON(t *testing.T) {
	client := createTestAPIClient(t)

	invalidJSON := []byte(`{"plan_name": invalid json}`)

	result, err := PlanWrite(invalidJSON, client)

	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestPlanWrite_UnknownAction(t *testing.T) {
	client := createTestAPIClient(t)

	input := PlanWriteInput{
		PlanName: "test-plan",
		Action:   WriteAction(999), // Invalid action
	}
	inputJSON, _ := json.Marshal(input)

	result, err := PlanWrite(inputJSON, client)

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
		PlanName: "test-plan",
		Action:   ActionAddSteps,
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
	assert.Contains(t, jsonStr, `"plan_name":"test-plan"`)
	assert.Contains(t, jsonStr, `"write_action":1`)
}

func TestPlanWriteInput_JSONUnmarshaling(t *testing.T) {
	jsonData := `{
		"plan_name":"test-plan",
		"write_action":1,
		"steps_to_add":[
			{"id":"step-1","description":"Test step"}
		]
	}`

	var input PlanWriteInput
	err := json.Unmarshal([]byte(jsonData), &input)

	assert.NoError(t, err)
	assert.Equal(t, "test-plan", input.PlanName)
	assert.Equal(t, ActionAddSteps, input.Action)
	assert.Len(t, input.StepsToAdd, 1)
	assert.Equal(t, "step-1", input.StepsToAdd[0].ID)
}

// Tests for WriteAction enum
func TestWriteAction_Values(t *testing.T) {
	assert.Equal(t, WriteAction(0), ActionSetStatus)
	assert.Equal(t, WriteAction(1), ActionAddSteps)
	assert.Equal(t, WriteAction(2), ActionRemoveSteps)
	assert.Equal(t, WriteAction(3), ActionCompactPlan)
	assert.Equal(t, WriteAction(4), ActionReorderSteps)
}

// Tests for Status enum
func TestStatus_Values(t *testing.T) {
	assert.Equal(t, Status(0), StatusDone)
	assert.Equal(t, Status(1), StatusTodo)
}

func TestStatus_Names(t *testing.T) {
	assert.Equal(t, "DONE", statusName[StatusDone])
	assert.Equal(t, "TODO", statusName[StatusTodo])
}

// Table-driven tests
func TestPlanWrite_VariousInputs(t *testing.T) {
	t.Skip("Requires running API server")

	client := createTestAPIClient(t)

	tests := []struct {
		name        string
		input       PlanWriteInput
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid add steps",
			input: PlanWriteInput{
				PlanName: "test-plan-1",
				Action:   ActionAddSteps,
				StepsToAdd: []PlanStepInput{
					{ID: "step-1", Description: "Test step"},
				},
			},
			expectError: false,
		},
		{
			name: "missing plan name",
			input: PlanWriteInput{
				PlanName: "",
				Action:   ActionAddSteps,
			},
			expectError: true,
			errorMsg:    "missing or invalid plan_name",
		},
		{
			name: "set status without step id",
			input: PlanWriteInput{
				PlanName: "test-plan",
				Action:   ActionSetStatus,
				StepID:   "",
				Status:   StatusDone,
			},
			expectError: true,
			errorMsg:    "requires 'step_id'",
		},
		{
			name: "add step without id",
			input: PlanWriteInput{
				PlanName: "test-plan",
				Action:   ActionAddSteps,
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
				PlanName: "test-plan",
				Action:   ActionAddSteps,
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

			result, err := PlanWrite(inputJSON, client)

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

	client, _ := api.NewClient("http://localhost:8080")
	input := PlanWriteInput{
		PlanName: "bench-plan",
		Action:   ActionAddSteps,
		StepsToAdd: []PlanStepInput{
			{ID: "step-1", Description: "Benchmark step"},
		},
	}
	inputJSON, _ := json.Marshal(input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		PlanWrite(inputJSON, client)
	}
}

func BenchmarkPlanWrite_SetStatus(b *testing.B) {
	b.Skip("Requires running API server")

	client, _ := api.NewClient("http://localhost:8080")
	input := PlanWriteInput{
		PlanName: "bench-plan",
		Action:   ActionSetStatus,
		StepID:   "step-1",
		Status:   StatusDone,
	}
	inputJSON, _ := json.Marshal(input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		PlanWrite(inputJSON, client)
	}
}
