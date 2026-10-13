-- name: CreateLibraryDocument :one
INSERT INTO library_documents (file_name, description, stored_path)
VALUES (?, ?, ?)
RETURNING *;

-- name: GetLibraryDocument :one
SELECT * FROM library_documents
WHERE id = ?;

-- name: ListLibraryDocuments :many
-- The library, newest first. IDs only grow, so the highest is the latest added.
-- @scenario [features/document-library.feature#library-newest-first] The newest documents are listed first
SELECT * FROM library_documents
ORDER BY id DESC;

-- name: SetLibraryDocumentDescription :exec
UPDATE library_documents
SET description = ?
WHERE id = ?;

-- name: DeleteLibraryDocument :exec
DELETE FROM library_documents
WHERE id = ?;
