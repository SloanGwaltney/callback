-- +goose Up

-- The document library: documents kept for the whole job search, not attached to any job. Like a
-- job's documents, the file itself is a copy in the data directory, at stored_path relative to it.
-- See docs/adr/0008-store-attached-documents-as-files-in-the-app-data-directory.md. There's no
-- job_id, so deleting a job never reaches these rows. The description defaults to '' rather than
-- NULL, like the jobs table.
CREATE TABLE library_documents (
    id          INTEGER PRIMARY KEY,
    -- The name of the file the user added, as shown in the app.
    file_name   TEXT    NOT NULL CHECK (file_name <> ''),
    description TEXT    NOT NULL DEFAULT '',
    stored_path TEXT    NOT NULL UNIQUE
);

-- +goose Down

DROP TABLE library_documents;
