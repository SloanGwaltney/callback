-- name: ListJobs :many
-- Every job, grouped by status and in board order within each status.
SELECT * FROM jobs
ORDER BY status, position;
