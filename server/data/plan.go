package data

import (
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"strings"
)

//go:embed plan_schema.sql
var PlanSchema string

var ErrPlanNotFound = errors.New("plan not found")

type Plan struct {
	ID    string  `json:"id"`
	Steps []*Step `json:"steps"`
	isNew bool
}

type PlanModel struct {
	DB *sql.DB
}

// Hold summary of a plan. Used by List() method
type PlanInfo struct {
	Name           string `json:"name"`
	Status         string `json:"status"` // "DONE" or "TODO"
	TotalTasks     int    `json:"total_tasks"`
	CompletedTasks int    `json:"completed_tasks"`
}

type Step struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Status      string   `json:"status"` // "DONE" or "TODO"
	Acceptance  []string `json:"acceptance"`
	stepOrder   int
}

func (pm *PlanModel) Close() error {
	if pm.DB != nil {
		return pm.DB.Close()
	}
	return nil
}

// Create returns an in-memory Plan object.
// The ID of the plan is set to its name.
// The plan is not persisted to the database until Save is called.
func (pm *PlanModel) Create(plan *Plan) error {
	if plan.ID == "" {
		return fmt.Errorf("plan name cannot be empty")
	}

	query := `
	INSERT INTO plans (id) VALUES (?)
	RETURNING id
	`

	err := pm.DB.QueryRow(query, plan.ID).Scan(&plan.ID)
	// TODO: Is this the right way? Or should I handle it in server.go?
	if err != nil {
		// Check if the error is due to a unique constraint violation (plan already exists)
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("plan with name '%s' already exists in database, cannot save as new", plan.ID)
		}
		// Issue sql: no rows in result set
		return fmt.Errorf("failed to insert new plan '%s' into database: %w", plan.ID, err)
	}

	// Initialize Steps slice and mark as persisted (not new anymore)
	if plan.Steps == nil {
		plan.Steps = []*Step{}
	}
	plan.isNew = false

	return nil
}

func (pm *PlanModel) Get(name string) (*Plan, error) {
	var planID string
	err := pm.DB.QueryRow("SELECT id FROM plans WHERE id = ?", name).Scan(&planID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("plan with name '%s' not found", name)
		}
		return nil, fmt.Errorf("failed to query plan '%s': %w", name, err)
	}

	plan := &Plan{
		ID:    planID,
		Steps: []*Step{},
		isNew: false,
	}

	rows, err := pm.DB.Query("SELECT id, description, status, step_order FROM steps WHERE plan_id = ? ORDER BY step_order ASC", planID)
	if err != nil {
		return nil, fmt.Errorf("failed to query steps for plan '%s': %w", name, err)
	}
	defer rows.Close()

	// Temp store step by ID for lookup when adding acceptance criteria
	stepsByID := make(map[string]*Step)

	for rows.Next() {
		step := &Step{}
		err := rows.Scan(&step.ID, &step.Description, &step.Status, &step.stepOrder)
		if err != nil {
			return nil, fmt.Errorf("failed to scan step for plan '%s': %w", name, err)
		}
		// Store acceptance criteria
		step.Acceptance = []string{}
		plan.Steps = append(plan.Steps, step)
		stepsByID[step.ID] = step
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating steps for plan '%s': %w", name, err)
	}

	// Fetch acceptance criteria for each step.
	// Maintain order of steps from the query.
	for _, step := range plan.Steps {
		acRows, err := pm.DB.Query("SELECT criterion FROM step_acceptance_criteria WHERE step_id = ? AND plan_id = ? ORDER BY criterion_order ASC", step.ID, planID)
		if err != nil {
			return nil, fmt.Errorf("failed to query acceptance criteria for step '%s' in plan '%s': %w", step.ID, name, err)
		}
		// Close each acRow in each iteration to prevent resource leak,
		// especially for long running loop, where resources are kept open,
		// and will not be closed until the function contains the loop returns.
		for acRows.Next() {
			var acDescription string
			err := acRows.Scan(&acDescription)
			if err != nil {
				acRows.Close()
				return nil, fmt.Errorf("failed to scan acceptance criterion for step '%s' in plan '%s': %w", step.ID, name, err)
			}
			step.Acceptance = append(step.Acceptance, acDescription)
		}
		if err = acRows.Err(); err != nil {
			acRows.Close()
			return nil, fmt.Errorf("error iterating acceptance criteria for step '%s' in plan '%s': %w", step.ID, name, err)
		}
		acRows.Close() // Manual close instead of defer
	}

	return plan, nil
}

func (p *Plan) Inspect() string {
	var builder strings.Builder

	for i, step := range p.Steps {
		// Headline: includes step number, status, and ID.
		header := fmt.Sprintf("## %d. [%s] %s\n", i+1, strings.ToUpper(step.Status), step.ID)
		builder.WriteString(header)

		if step.Description != "" {
			builder.WriteString("\n" + step.Description + "\n") // Add blank lines around description
		}
		builder.WriteString("\n") // Ensure a blank line after header or description

		// Acceptance criteria numbered list
		if len(step.Acceptance) > 0 {
			builder.WriteString("Acceptance Criteria:\n")
			for j, criterion := range step.Acceptance {
				builder.WriteString(fmt.Sprintf("%d. %s\n", j+1, criterion))
			}
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

func (p *Plan) NextStep() *Step {
	for _, step := range p.Steps {
		// Case-insensitive comparison just in case
		if strings.ToUpper(step.Status) != "DONE" {
			return step
		}
	}
	return nil
}

func (p *Plan) RemoveSteps(stepIDs []string) int {
	if len(stepIDs) == 0 {
		return 0 // Nothing to remove
	}
	if len(p.Steps) == 0 {
		return 0 // No steps in the plan to remove from
	}

	// Create a set of IDs to remove for efficient lookup
	idsToRemove := make(map[string]struct{})
	for _, id := range stepIDs {
		idsToRemove[id] = struct{}{}
	}

	var newSteps []*Step
	removedCount := 0
	for _, step := range p.Steps {
		if _, found := idsToRemove[step.ID]; found {
			removedCount++
		} else {
			newSteps = append(newSteps, step)
		}
	}

	p.Steps = newSteps
	return removedCount
}

func (s *Step) GetID() string {
	return s.ID
}

func (s *Step) GetStatus() string {
	return strings.ToUpper(s.Status)
}

func (s *Step) GetDescription() string {
	return s.Description
}

func (s *Step) GetAcceptanceCriteria() []string {
	// Just a return, no need for a copy
	return s.Acceptance
}

// Set the status of the step with the given stepID to "DONE" in-memory.
func (p *Plan) MarkStepAsCompleted(stepID string) error {
	for _, step := range p.Steps {
		if step.ID == stepID {
			step.Status = "DONE"
			return nil
		}
	}
	return fmt.Errorf("step with ID '%s' not found in plan '%s'", stepID, p.ID)
}

// Sets the status of the step with the given stepID to "TODO" in-memory.
func (p *Plan) MarkStepAsIncomplete(stepID string) error {
	for _, step := range p.Steps {
		if step.ID == stepID {
			step.Status = "TODO"
			return nil
		}
	}
	return fmt.Errorf("step with ID '%s' not found in plan '%s'", stepID, p.ID)
}

// Appends a new step to the plan.
// The new step is initialized with status "TODO".
func (p *Plan) AddStep(id, description string, acceptanceCriteria []string) {
	newStep := &Step{
		ID:          id,
		Description: description,
		Status:      "TODO", // Default status for new steps
		Acceptance:  acceptanceCriteria,
	}
	p.Steps = append(p.Steps, newStep)
}

// Rearranges the steps in the plan.
// Steps whose IDs are in newStepOrder are placed first, in the specified order.
// Any remaining steps from the original plan are appended afterwards,
// maintaining their original relative order.
// If a step ID in newStepOrder does not exist in the plan, it is ignored.
// Duplicate step IDs in newStepOrder are also effectively ignored after the first placement.
func (p *Plan) ReorderSteps(newStepOrder []string) {
	if len(p.Steps) == 0 {
		return
	}

	originalStepMap := make(map[string]*Step, len(p.Steps))
	for _, step := range p.Steps {
		originalStepMap[step.ID] = step
	}

	var reorderedSteps []*Step

	// Keep track of steps set by newStepOrder
	// to correctly append remaining steps and handle potential duplicates in newStepOrder (do we really need this?).
	placedStepIDs := make(map[string]struct{})

	// Place steps into newStepOrder
	for _, sID := range newStepOrder {
		s, exists := originalStepMap[sID]
		if !exists {
			continue
		}

		if _, alreadyPlaced := placedStepIDs[sID]; alreadyPlaced {
			// Duplicate in newStepOrder, ignore
			continue
		}
		reorderedSteps = append(reorderedSteps, s)
		// Marked as placed
		placedStepIDs[sID] = struct{}{}
	}

	// Append remaining steps from original step order
	// Remaining = not part of or duplicated with newStepOrder
	for _, originalStep := range p.Steps {
		if _, wasPlaced := placedStepIDs[originalStep.ID]; !wasPlaced {
			reorderedSteps = append(reorderedSteps, originalStep)
			// Marked as placed
			placedStepIDs[originalStep.ID] = struct{}{}
		}
	}

	p.Steps = reorderedSteps
}

func (p *Plan) IsStepCompleted() bool {
	return p.NextStep() == nil
}

// Retrieve summary information for all plans from the database
func (pm *PlanModel) List() ([]PlanInfo, error) {
	rows, err := pm.DB.Query(
		`SELECT
				p.id,
				COUNT(s.id),
				SUM(CASE WHEN s.status = 'DONE' THEN 1 ELSE 0 END)
		FROM plans p
		LEFT JOIN steps s ON p.id = s.plan_id
		GROUP BY p.id`)
	if err != nil {
		return nil, fmt.Errorf("failed to query plan summaries: %w", err)
	}
	defer rows.Close()

	var plansInfo []PlanInfo

	for rows.Next() {
		var info PlanInfo
		var totalTasks sql.NullInt64 // For COUNT which can be 0 -> NULL
		var completedTasks sql.NullInt64

		if err := rows.Scan(&info.Name, &totalTasks, &completedTasks); err != nil {
			return nil, fmt.Errorf("failed to scan plan summary: %w", err)
		}

		info.TotalTasks = int(totalTasks.Int64)
		info.CompletedTasks = int(completedTasks.Int64)

		if info.TotalTasks > 0 && info.CompletedTasks == info.TotalTasks {
			info.Status = "DONE"
		} else {
			info.Status = "TODO"
		}
		plansInfo = append(plansInfo, info)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating plan summaries: %w", err)
	}

	return plansInfo, nil
}

func (pm *PlanModel) Save(plan *Plan) error {
	tx, err := pm.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if plan.isNew {
		_, err := tx.Exec("INSERT INTO plans (id) VALUES (?)", plan.ID)
		if err != nil {
			// Check if the error is due to a unique constraint violation (plan already exists)
			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				return fmt.Errorf("plan with name '%s' already exists in database, cannot save as new", plan.ID)
			}
			return fmt.Errorf("failed to insert new plan '%s' into database: %w", plan.ID, err)
		}
	} else {
		// Even if it isn't a new plan, we still verify it to get a cleaner error
		// than what might come from step synchronization?
		var checkID string
		err := tx.QueryRow("SELECT id FROM plans WHERE id = ?", plan.ID).Scan(&checkID)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("plan with name '%s' not found in database, cannot update", plan.ID)
			}
			return fmt.Errorf("failed to verify existence of plan '%s': %w", plan.ID, err)
		}
	}

	/* Synchronize steps from the DB (if exist) with input steps*/

	// Get existing step IDs for the current plan
	rows, err := tx.Query("SELECT id FROM steps WHERE plan_id = ?", plan.ID)
	if err != nil {
		return fmt.Errorf("failed to query existing steps for plan '%s': %w", plan.ID, err)
	}

	dbStepIDs := make(map[string]bool)
	for rows.Next() {
		var stepID string
		if err := rows.Scan(&stepID); err != nil {
			rows.Close()
			return fmt.Errorf("failed to scan existing step ID: %w", err)
		}
		dbStepIDs[stepID] = true
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return fmt.Errorf("error iterating existing step IDs: %w", err)
	}

	// Compare incoming plan with plan from DB
	planStepIDs := make(map[string]bool)
	for _, s := range plan.Steps {
		planStepIDs[s.ID] = true
	}

	for dbStepID := range dbStepIDs {
		if !planStepIDs[dbStepID] {
			// Deprecated steps from DB, remove by constraints
			_, err := tx.Exec("DELETE FROM step_acceptance_criteria WHERE plan_id = ? AND step_id = ?", plan.ID, dbStepID)
			if err != nil {
				return fmt.Errorf("failed to delete old acceptance criteria for step '%s' in plan '%s': %w", dbStepID, plan.ID, err)
			}
		}
	}

	for i, s := range plan.Steps {
		s.stepOrder = i

		// Validate step before persisting
		if s.ID == "" {
			return fmt.Errorf("step ID cannot be empty")
		}
		if s.Status == "" {
			s.Status = "TODO" // Default to TODO if not set
		}
		s.Status = strings.ToUpper(s.Status)
		if s.Status != "TODO" && s.Status != "DONE" {
			return fmt.Errorf("step '%s' has invalid status '%s': must be TODO or DONE", s.ID, s.Status)
		}

		// Update or create step
		if dbStepIDs[s.ID] {
			_, err := tx.Exec("UPDATE steps SET description = ?, status = ?, step_order = ? WHERE plan_id = ? AND id = ?", s.Description, s.Status, s.stepOrder, plan.ID, s.ID)
			if err != nil {
				return fmt.Errorf("failed to update step '%s' in plan '%s': %w", s.ID, plan.ID, err)
			}
		} else {
			_, err := tx.Exec("INSERT INTO steps(id, plan_id, description, status, step_order) VALUES(?, ?, ?, ?, ?)", s.ID, plan.ID, s.Description, s.Status, s.stepOrder)
			if err != nil {
				return fmt.Errorf("failed to insert step '%s' into plan '%s': %w", s.ID, plan.ID, err)
			}
		}
		// Delete ACs here just to make sure clean ACs when we update a plan?
		_, err = tx.Exec("DELETE FROM step_acceptance_criteria WHERE plan_id = ? AND step_id = ?", plan.ID, s.ID)
		if err != nil {
			return fmt.Errorf("failed to delete old acceptance criteria for step '%s' in plan '%s': %w", s.ID, plan.ID, err)
		}

		for j, acText := range s.Acceptance {
			_, err = tx.Exec("INSERT INTO step_acceptance_criteria (plan_id, step_id, criterion_order, criterion) VALUES (?, ?, ?, ?)", plan.ID, s.ID, j, acText)
			if err != nil {
				return fmt.Errorf("failed to insert acceptance criterion for step '%s' in plan '%s': %w", s.ID, plan.ID, err)
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction for plan '%s': %w", plan.ID, err)
	}

	// Update in-mem status
	if plan.isNew {
		plan.isNew = false
	}

	return nil
}

func (p *Plan) IsCompleted() bool {
	return p.NextStep() == nil // If NextStep is nil, all steps are DONE
}

// Remove deletes plans from the database by their names (IDs).
// It relies on "ON DELETE CASCADE" foreign key constraints to remove associated steps and criteria.
// It returns a map where keys are plan names and values are errors encountered during deletion (nil on success).
func (pm *PlanModel) Remove(planNames []string) map[string]error {
	results := make(map[string]error)
	tx, err := pm.DB.Begin()
	if err != nil {
		// Return a general error
		results["_"] = fmt.Errorf("failed to begin transaction for remove: %w", err)
		return results
	}

	defer tx.Rollback()

	stmt, err := tx.Prepare("DELETE FROM plans WHERE id = ?")
	if err != nil {
		results["_"] = fmt.Errorf("failed to prepare delete statement: %w", err)
		return results
	}
	defer stmt.Close()

	for _, name := range planNames {
		result, err := stmt.Exec(name)
		if err != nil {
			results[name] = fmt.Errorf("failed to execute delete for plan '%s': %w", name, err)
			continue
		}
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			// Report this either as an error or warning
			results[name] = fmt.Errorf("plan '%s' not found for deletion", name)
		} else {
			// Success
			results[name] = nil
		}
	}

	hasErrors := false
	for _, err := range results {
		if err != nil {
			hasErrors = true
			break
		}
	}

	if !hasErrors {
		if err := tx.Commit(); err != nil {
			results["_"] = fmt.Errorf("failed to commit transaction for remove: %w", err)
			for name, resErr := range results {
				if resErr == nil {
					results[name] = fmt.Errorf("transaction commit failed after successful delete prep: %w", err)
				}
			}
		}
	} else {
		// Rollback happens automatically with defer
		// so we return the map with errors now
	}

	return results
}

// Remove all completed plans from the DB i.e., plans that have their steps all marked as DONE.
func (pm *PlanModel) Compact() error {
	query := `
		SELECT p.id
		FROM plans p
		LEFT JOIN steps s ON p.id = s.plan_id
		GROUP BY p.id
		HAVING COUNT(s.id) = 0 OR SUM(CASE WHEN s.status = 'DONE' THEN 1 ELSE 0 END) = COUNT(s.id);
	`
	rows, err := pm.DB.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query completed plans for compaction: %w", err)
	}
	defer rows.Close()

	var completedPlanIDs []string
	for rows.Next() {
		var planID string
		if err := rows.Scan(&planID); err != nil {
			return fmt.Errorf("failed to scan completed plan ID: %w", err)
		}
		completedPlanIDs = append(completedPlanIDs, planID)
	}
	if err = rows.Err(); err != nil {
		return fmt.Errorf("error iterating completed plan IDs: %w", err)
	}
	rows.Close()

	if len(completedPlanIDs) == 0 {
		// Nothing to compact
		return nil
	}

	// While Remove returns a map of errors, Compact just returns a single error
	// so we check the map for any errors
	removeResults := pm.Remove(completedPlanIDs)

	var firstErr error
	var errorCount int
	for planID, err := range removeResults {
		if err != nil {
			errorCount++
			if firstErr == nil {
				if planID == "_" {
					// Check transaction error from Remove
					firstErr = err
				} else {
					firstErr = fmt.Errorf("failed to remove plan '%s': '%w'", planID, err)
				}
			}
		}
	}

	if firstErr != nil {
		return fmt.Errorf("encountered %d error(s) during compaction, first error: %w", errorCount, firstErr)
	}

	return nil
}
