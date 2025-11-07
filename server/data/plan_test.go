package data

import (
	"database/sql"
	"fmt"
	"reflect"
	"testing"
)

func createTestModel(t *testing.T) PlanModel {
	testDB := testutil.CreateTestDB(t, PlanSchema)
	return PlanModel{DB: testDB}
}

func TestNewPlanner(t *testing.T) {
	model := createTestModel(t)

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
	planner := createTestModel(t)

	planName := "test-plan-create"
	plan, err := planner.Create(planName)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if plan == nil {
		t.Fatal("Create returned a nil plan")
	}
	if plan.ID != planName {
		t.Errorf("Create returned plan with wrong ID: got %s, want %s", plan.ID, planName)
	}
	if len(plan.Steps) != 0 {
		t.Errorf("Create returned plan with non-empty steps: got %d, want 0", len(plan.Steps))
	}
	if !plan.isNew { // Verify isNew flag is true
		t.Errorf("Create returned plan with isNew = false, want true")
	}

	// Verify NOT in DB yet
	var count int
	err = planner.DB.QueryRow("SELECT COUNT(*) FROM plans WHERE id = ?", planName).Scan(&count)
	if err != nil && err != sql.ErrNoRows { // sql.ErrNoRows is expected if not found, other errors are DB issues
		t.Fatalf("Failed to query DB after Create (expected no rows or 0 count): %v", err)
	}
	if count != 0 { // Should be 0 as it's not saved yet
		t.Errorf("Plan count in DB is wrong after Create: got %d, want 0", count)
	}

	// Test creating a plan with the same name (should not error, as it's in-memory only until save)
	// The old test expected an error because Create also saved to DB and hit a UNIQUE constraint.
	// Now, Create only makes an in-memory object.
	// The responsibility of checking for existing plans shifts to the Save method (or a pre-check if desired).
	_, err = planner.Create(planName)
	if err != nil {
		t.Errorf("Creating a second in-memory plan with the same name should not error: %v", err)
	}
}

func TestPlanner_Get_Basic(t *testing.T) {
	planner := createTestModel(t)

	planName := "test-plan-get"
	createdPlan, err := planner.Create(planName)
	if err != nil {
		t.Fatalf("Setup failed: Could not create plan: %v", err)
	}
	err = planner.Save(createdPlan)
	if err != nil {
		t.Fatalf("Setup failed: Could not save plan: %v", err)
	}

	plan, err := planner.Get(planName)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if plan == nil {
		t.Fatal("Get returned a nil plan")
	}
	if plan.ID != planName {
		t.Errorf("Get returned plan with wrong ID: got %s, want %s", plan.ID, planName)
	}
	if len(plan.Steps) != 0 {
		t.Errorf("Get returned plan with non-empty steps initially: got %d, want 0", len(plan.Steps))
	}

	// Test getting non-existent plan
	_, err = planner.Get("non-existent-plan")
	if err == nil {
		t.Error("Expected error when getting non-existent plan, but got nil")
	}
}

func TestPlanner_SaveAndGet(t *testing.T) {
	planner := createTestModel(t)
	planName := "test-plan-save-get"

	// 1. Create the initial plan
	plan, err := planner.Create(planName)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if !plan.isNew {
		t.Fatal("Newly created plan should have isNew = true")
	}

	// 2. Add steps to the in-memory plan
	plan.AddStep("step1", "First step description", []string{"AC1.1", "AC1.2"})
	plan.AddStep("step2", "Second step", []string{"AC2.1"})

	// 3. Save the plan
	err = planner.Save(plan)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if plan.isNew { // isNew should be false after a successful save
		t.Errorf("plan.isNew is true after Save, want false")
	}

	// 4. Get the plan back
	retrievedPlan, err := planner.Get(planName)
	if err != nil {
		t.Fatalf("Get after Save failed: %v", err)
	}

	// 5. Verify the retrieved plan
	if retrievedPlan.ID != planName {
		t.Errorf("Retrieved plan ID mismatch: got %s, want %s", retrievedPlan.ID, planName)
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
	// retrievedPlan.Steps[0].status = "DONE" // Mark step2 as DONE (it's now at index 0)
	err = retrievedPlan.MarkStepAsCompleted("step2") // Mark step2 as DONE (it's now at index 0)
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
	finalPlan, err := planner.Get(planName)
	if err != nil {
		t.Fatalf("Second Get failed: %v", err)
	}

	// 9. Verify final state
	if len(finalPlan.Steps) != 2 {
		t.Fatalf("Final plan step count mismatch: got %d, want 2", len(finalPlan.Steps))
	}

	// Check order and content
	if finalPlan.Steps[0].ID() != "step3" {
		t.Errorf("Final Step 1 ID mismatch (expected step3)")
	}
	if finalPlan.Steps[0].Status() != "TODO" {
		t.Errorf("Final Step 1 Status mismatch (expected TODO)")
	}
	if finalPlan.Steps[1].ID() != "step2" {
		t.Errorf("Final Step 2 ID mismatch (expected step2)")
	}
	if finalPlan.Steps[1].Status() != "DONE" {
		t.Errorf("Final Step 2 Status mismatch (expected DONE)")
	}
	if finalPlan.isNew { // Should be false as it was retrieved from DB
		t.Errorf("finalPlan.isNew is true after Get, want false")
	}
}
