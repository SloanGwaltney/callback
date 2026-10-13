-- name: ListJobs :many
-- Every job with how many documents it has, grouped by status and in board order within each
-- status.
SELECT
    jobs.*,
    (SELECT COUNT(*) FROM documents WHERE documents.job_id = jobs.id) AS document_count
FROM jobs
ORDER BY status, position;

-- name: GetJob :one
SELECT * FROM jobs
WHERE id = ?;

-- name: ListJobIDsInStatus :many
-- The jobs in one status column, top to bottom.
SELECT id FROM jobs
WHERE status = ?
ORDER BY position;

-- name: CreateJob :one
INSERT INTO jobs (company, role, description, link, salary, location, notes, status, position, date_applied, interest)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id;

-- name: UpdateJobDetails :exec
UPDATE jobs
SET company = ?, role = ?, description = ?, link = ?, salary = ?, location = ?, notes = ?, date_applied = ?, interest = ?
WHERE id = ?;

-- name: UpdateJobPlacement :exec
-- Moves a job to a status and position, with the date applied and closing reason that go with it.
-- Status and closing reason change together so the schema's checks hold after the statement.
UPDATE jobs
SET status = ?, position = ?, date_applied = ?, closing_reason = ?, closing_reason_other = ?
WHERE id = ?;

-- name: SetJobPosition :exec
UPDATE jobs
SET position = ?
WHERE id = ?;

-- name: SetClosingReason :exec
UPDATE jobs
SET closing_reason = ?, closing_reason_other = ?
WHERE id = ?;

-- name: DeleteJob :exec
-- Deletes the job and, through ON DELETE CASCADE, its document, contact, and interview rows. The
-- document files are the caller's to delete.
-- @scenario [features/job-contacts.feature#contacts-deleted-with-job] Deleting a job deletes its contacts
-- @scenario [features/job-interviews.feature#interviews-deleted-with-job] Deleting a job deletes its interviews
DELETE FROM jobs
WHERE id = ?;

-- name: SetNextAction :exec
-- Sets what's next on a job and when it's expected, or NULL for no expected date. Nothing else on
-- the job changes, so its column and place in it stay the same.
-- @scenario [features/job-next-action.feature#set-next-action] Set a next action with an expected date
-- @scenario [features/job-next-action.feature#set-next-action-no-date] Set a next action without an expected date
-- @scenario [features/job-next-action.feature#edit-next-action] Change the next action and its expected date
-- @scenario [features/job-next-action.feature#clear-next-action-date] Clear only the expected date
UPDATE jobs
SET next_action = ?, next_action_date = ?
WHERE id = ?;

-- name: ClearNextAction :exec
-- @scenario [features/job-next-action.feature#clear-next-action] Clear the next action once it's done
UPDATE jobs
SET next_action = NULL, next_action_date = NULL
WHERE id = ?;
