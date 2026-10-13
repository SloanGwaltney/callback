-- name: CreateContact :one
INSERT INTO contacts (job_id, name, relationship, email, phone, profile_link, notes)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetContact :one
SELECT * FROM contacts
WHERE id = ?;

-- name: ListContactsForJob :many
-- A job's contacts in the order they were added.
-- @scenario [features/job-contacts.feature#view-contacts] See a job's contacts when opening its card
SELECT * FROM contacts
WHERE job_id = ?
ORDER BY id;

-- name: UpdateContact :exec
UPDATE contacts
SET name = ?, relationship = ?, email = ?, phone = ?, profile_link = ?, notes = ?
WHERE id = ?;

-- name: DeleteContact :exec
DELETE FROM contacts
WHERE id = ?;
