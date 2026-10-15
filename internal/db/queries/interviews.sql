-- name: CreateInterview :one
INSERT INTO interviews (job_id, date, time, kind, notes)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetInterview :one
SELECT * FROM interviews
WHERE id = ?;

-- name: ListInterviewsForJob :many
-- A job's interviews from earliest to latest. Interviews on the same day are in time order, with
-- the ones that have no time last, then in the order they were added.
-- @scenario [#interviews-in-date-order] [#interviews-same-day-by-time]
SELECT * FROM interviews
WHERE job_id = ?
ORDER BY date, time = '', time, id;

-- name: ListUpcomingInterviews :many
-- Every interview on or after a date, for every job, in the same order as ListInterviewsForJob
-- within each job. The board takes each job's first as its next interview.
-- @scenario [#card-shows-next-interview]
SELECT * FROM interviews
WHERE date >= ?
ORDER BY job_id, date, time = '', time, id;

-- name: UpdateInterview :exec
UPDATE interviews
SET date = ?, time = ?, kind = ?, notes = ?
WHERE id = ?;

-- name: DeleteInterview :exec
DELETE FROM interviews
WHERE id = ?;
