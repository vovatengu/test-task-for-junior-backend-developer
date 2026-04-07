ALTER TABLE tasks ADD COLUMN scheduled_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;

ALTER TABLE tasks ADD COLUMN parent_id BIGINT REFERENCES tasks(id) ON DELETE CASCADE;

ALTER TABLE tasks ADD COLUMN recurrence_type VARCHAR;

ALTER TABLE tasks ADD CONSTRAINT chk_tasks_recurrence_type
CHECK (recurrence_type IN ('daily', 'monthly', 'specific_dates', 'even_odd') OR recurrence_type IS NULL);

ALTER TABLE tasks ADD COLUMN recurrence_params JSONB;