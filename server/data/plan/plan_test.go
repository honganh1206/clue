package plan

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/honganh1206/clue/server/data/testutil"
)

func createTestModel(t *testing.T) PlanModel {
	testDB := testutil.CreateTestDB(t, Schema)
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
