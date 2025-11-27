package data

import (
	"database/sql"
	"fmt"
	"reflect"
	"testing"
)

func createPlanTestModel(t *testing.T) PlanModel {
	testDB := createTestDB(t)
	return PlanModel{DB: testDB}
}

func createTestConversation(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	_, err := db.Exec("INSERT INTO conversations (id) VALUES (?)", id)
	if err != nil {
		t.Fatalf("Failed to create test conversation: %v", err)
	}
}

func TestNewPlanner(t *testing.T) {
	model := createPlanTestModel(t)

	// Check if tables were created (basic check by trying to query them)
	tables := []string{"plans", "steps", "step_acceptance_criteria"}
	for _, table := range tables {
		// Using QueryRow because we don't expect results, just no error
		err := model.DB.QueryRow(fmt.Sprintf("SELECT 1 FROM %s LIMIT 1", table)).Scan(new(int))
		// We expect sql.ErrNoRows if the table is empty, which is fine.
		// Any other error indicates a problem (e.g., table doesn't exist).
		if err != nil && err != sql.ErrNoRows {
			t.Errorf("Failed to query '%s' table, schema likely not initialized correctly: %v", table, err)
		}
	}
}

func TestPlanner_Create(t *testing.T) {
	planner := createPlanTestModel(t)

	conversationID := "test-conversation-id"
	createTestConversation(t, planner.DB, conversationID)

	plan, err := NewPlan(conversationID)
	if err != nil {
		t.Fatalf("NewPlan failed: %v", err)
	}

	err = planner.Create(plan)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if plan.ID == "" {
		t.Error("Create returned plan with empty ID")
	}
	if len(plan.Steps) != 0 {
		t.Errorf("Create returned plan with non-empty steps: got %d, want 0", len(plan.Steps))
	}

	// Verify in DB now
	var count int
	err = planner.DB.QueryRow("SELECT COUNT(*) FROM plans WHERE id = ?", plan.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query DB after Create: %v", err)
	}
	if count != 1 {
		t.Errorf("Plan count in DB is wrong after Create: got %d, want 1", count)
	}

	// Test creating a second plan with the same conversation ID (should fail)
	plan2, err := NewPlan(conversationID)
	if err != nil {
		t.Fatalf("NewPlan failed: %v", err)
	}
	err = planner.Create(plan2)
	if err == nil {
		t.Error("Creating a second plan with the same conversation ID should fail")
	}
}

func TestPlanner_Get_Basic(t *testing.T) {
	planner := createPlanTestModel(t)

	conversationID := "test-conversation-id"
	createTestConversation(t, planner.DB, conversationID)

	createdPlan, err := NewPlan(conversationID)
	if err != nil {
		t.Fatalf("NewPlan failed: %v", err)
	}

	err = planner.Create(createdPlan)
	if err != nil {
		t.Fatalf("Setup failed: Could not create plan: %v", err)
	}

	plan, err := planner.Get(conversationID)
	if err != nil {
		t.Fatalf("GetByConversationID failed: %v", err)
	}

	if plan == nil {
		t.Fatal("GetByConversationID returned a nil plan")
	}
	if plan.ID != createdPlan.ID {
		t.Errorf("GetByConversationID returned plan with wrong ID: got %s, want %s", plan.ID, createdPlan.ID)
	}
	if len(plan.Steps) != 0 {
		t.Errorf("GetByConversationID returned plan with non-empty steps initially: got %d, want 0", len(plan.Steps))
	}

	// Test getting non-existent plan
	_, err = planner.Get("non-existent-plan-id")
	if err == nil {
		t.Error("Expected error when getting non-existent plan, but got nil")
	}
}

func TestPlanner_SaveAndGet(t *testing.T) {
	planner := createPlanTestModel(t)
	conversationID := "test-conversation-id"
	createTestConversation(t, planner.DB, conversationID)

	// 1. Create the initial plan
	plan, err := NewPlan(conversationID)
	if err != nil {
		t.Fatalf("NewPlan failed: %v", err)
	}

	err = planner.Create(plan)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// 2. Add steps to the in-memory plan
	plan.AddStep("step1", "First step description", []string{"AC1.1", "AC1.2"})
	plan.AddStep("step2", "Second step", []string{"AC2.1"})

	// 3. Save the plan
	err = planner.Save(plan)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 4. Get the plan back
	retrievedPlan, err := planner.Get(conversationID)
	if err != nil {
		t.Fatalf("GetByConversationID after Save failed: %v", err)
	}

	// 5. Verify the retrieved plan
	if retrievedPlan.ID != plan.ID {
		t.Errorf("Retrieved plan ID mismatch: got %s, want %s", retrievedPlan.ID, plan.ID)
	}
	if len(retrievedPlan.Steps) != 2 {
		t.Fatalf("Retrieved plan step count mismatch: got %d, want 2", len(retrievedPlan.Steps))
	}

	// Verify step 1
	step1 := retrievedPlan.Steps[0]
	if step1.GetID() != "step1" {
		t.Errorf("Step 1 ID mismatch")
	}
	if step1.GetDescription() != "First step description" {
		t.Errorf("Step 1 Description mismatch")
	}
	if step1.GetStatus() != "TODO" {
		t.Errorf("Step 1 Status mismatch")
	}
	if !reflect.DeepEqual(step1.GetAcceptanceCriteria(), []string{"AC1.1", "AC1.2"}) {
		t.Errorf("Step 1 Acceptance Criteria mismatch: got %v", step1.GetAcceptanceCriteria())
	}

	// Verify step 2
	step2 := retrievedPlan.Steps[1]
	if step2.GetID() != "step2" {
		t.Errorf("Step 2 ID mismatch")
	}
	if step2.GetDescription() != "Second step" {
		t.Errorf("Step 2 Description mismatch")
	}
	if step2.GetStatus() != "TODO" {
		t.Errorf("Step 2 Status mismatch")
	}
	if !reflect.DeepEqual(step2.GetAcceptanceCriteria(), []string{"AC2.1"}) {
		t.Errorf("Step 2 Acceptance Criteria mismatch: got %v", step2.GetAcceptanceCriteria())
	}

	// 6. Modify the plan (e.g., remove step, change status, reorder)
	retrievedPlan.RemoveSteps([]string{"step1"})
	err = retrievedPlan.MarkStepAsCompleted("step2")
	if err != nil {
		t.Fatalf("MarkAsCompleted failed: %v", err)
	}
	retrievedPlan.AddStep("step3", "Third step", nil)

	// Reorder (step3, step2) - Note: step1 was removed
	retrievedPlan.ReorderSteps([]string{"step3", "step2"})

	// 7. Save again
	err = planner.Save(retrievedPlan)
	if err != nil {
		t.Fatalf("Second Save failed: %v", err)
	}

	// 8. Get again
	finalPlan, err := planner.Get(conversationID)
	if err != nil {
		t.Fatalf("Second GetByConversationID failed: %v", err)
	}

	// 9. Verify final state
	if len(finalPlan.Steps) != 2 {
		t.Fatalf("Final plan step count mismatch: got %d, want 2", len(finalPlan.Steps))
	}

	// Check order and content
	if finalPlan.Steps[0].GetID() != "step3" {
		t.Errorf("Final Step 1 ID mismatch (expected step3)")
	}
	if finalPlan.Steps[0].GetStatus() != "TODO" {
		t.Errorf("Final Step 1 Status mismatch (expected TODO)")
	}
	if finalPlan.Steps[1].GetID() != "step2" {
		t.Errorf("Final Step 2 ID mismatch (expected step2)")
	}
	if finalPlan.Steps[1].GetStatus() != "DONE" {
		t.Errorf("Final Step 2 Status mismatch (expected DONE)")
	}
}
