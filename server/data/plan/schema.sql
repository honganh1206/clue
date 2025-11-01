
-- Foreign key constraint to ensure referential integrity
-- that is a value in one's table column must exist in another table's primary key column.
-- E.g., user_id of orders table must exist in users table primary key.
PRAGMA foreign_keys=ON;

CREATE TABLE IF NOT EXISTS plans (
		id TEXT PRIMARY KEY NOT NULL,  -- "active", "feature-x" - Are we sure using text as primary key is fine?...
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER IF NOT EXISTS plans_updated_at
AFTER UPDATE ON plans
FOR EACH ROW
BEGIN
		UPDATE plans SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.id;
END;

CREATE TABLE IF NOT EXISTS steps (
		id TEXT NOT NULL, -- e.g., "add-tests"
		plan_id TEXT NOT NULL, 
		description TEXT,
		status TEXT NOT NULL CHECK (status IN ('TODO', 'DONE')),
		step_order INTEGER NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (plan_id, id), -- Two primary keys?
		FOREIGN KEY (plan_id) REFERENCES plans(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_steps_plan_id ON steps(plan_id);

CREATE TRIGGER IF NOT EXISTS steps_updated_at
AFTER UPDATE ON steps
FOR EACH ROW
BEGIN
		UPDATE steps SET updated_at = CURRENT_TIMESTAMP WHERE plan_id = OLD.plan_id AND id = OLD.id;
		UPDATE plans SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.plan_id;
END;

-- Join table between steps and plans
CREATE TABLE IF NOT EXISTS step_acceptance_criteria (
		plan_id TEXT NOT NULL,
		step_id TEXT NOT NULL,
		criterion TEXT NOT NULL,
		criterion_order INTEGER NOT NULL, -- Order of criteria for a step
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY(plan_id, step_id, criterion_order), -- Why criterion_order as primary?
		FOREIGN KEY(plan_id, step_id) REFERENCES steps(plan_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_step_acceptance_criteria_plan_step ON step_acceptance_criteria(plan_id, step_id);
