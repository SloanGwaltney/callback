-- name: ListJobs :many
-- Every job, grouped by status and in board order within each status.
SELECT * FROM jobs
-- @then [features/job-board.feature] the "Saved" column shows "Initech", "Acme", and "Globex" in that order
ORDER BY status, position;
