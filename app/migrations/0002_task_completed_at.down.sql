ALTER TABLE tasks
    DROP KEY idx_tasks_team_completed,
    ADD KEY idx_tasks_status_updated (status, updated_at),
    DROP COLUMN completed_at;
