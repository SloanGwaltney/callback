-- name: AddStatusChange :exec
-- Records a job moving to a status on a date, with the closing reason when the status is Closed.
INSERT INTO status_changes (job_id, status, date, closing_reason, closing_reason_other)
VALUES (?, ?, ?, ?, ?);

-- name: ListStatusChangesForJob :many
-- A job's status history from oldest to newest. Rows are only ever added, so ID order is the order
-- the changes happened in, even on the same day.
-- @scenario [features/job-status-history.feature#history-oldest-first] The history lists every change from oldest to newest
-- @scenario [features/job-status-history.feature#history-same-day-order] Several changes on the same day keep the order they happened in
SELECT * FROM status_changes
WHERE job_id = ?
ORDER BY id;

-- name: ListCurrentStatusDates :many
-- The date each job moved to the status it's in now: the date of its latest status change.
SELECT job_id, date FROM status_changes
WHERE id IN (SELECT MAX(id) FROM status_changes GROUP BY job_id);
