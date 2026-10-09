-- name: CreateDocument :one
INSERT INTO documents (job_id, kind, file_name, stored_path)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetDocument :one
SELECT * FROM documents
WHERE id = ?;

-- name: ListDocumentsForJob :many
-- A job's documents in the order they were attached.
SELECT * FROM documents
WHERE job_id = ?
ORDER BY id;

-- name: SetDocumentKind :exec
UPDATE documents
SET kind = ?
WHERE id = ?;

-- name: DeleteDocument :exec
DELETE FROM documents
WHERE id = ?;
