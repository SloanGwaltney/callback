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
-- @scenario [features/job-interviews.feature#interviews-in-date-order] Interviews are listed from earliest to latest
-- @scenario [features/job-interviews.feature#interviews-same-day-by-time] Interviews on the same day are listed by time, with untimed ones last
SELECT * FROM interviews
WHERE job_id = ?
ORDER BY date, time = '', time, id;

-- name: ListUpcomingInterviews :many
-- Every interview on or after a date, for every job, in the same order as ListInterviewsForJob
-- within each job. The board takes each job's first as its next interview.
-- @scenario [features/job-interviews.feature#card-shows-next-interview] The card shows the date of the next upcoming interview
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
