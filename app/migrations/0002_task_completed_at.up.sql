ALTER TABLE tasks
    ADD COLUMN completed_at TIMESTAMP NULL AFTER updated_at,
    DROP KEY idx_tasks_status_updated,
    ADD KEY idx_tasks_team_completed (team_id, completed_at);

-- Best-effort backfill: updated_at is the closest known completion moment.
UPDATE tasks SET completed_at = updated_at WHERE status = 'done';
