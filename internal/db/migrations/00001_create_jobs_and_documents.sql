-- +goose Up

-- One row per job on the board. Optional text fields default to '' rather than NULL, so "empty"
-- has one meaning.
CREATE TABLE jobs (
    id                   INTEGER PRIMARY KEY,
    company              TEXT    NOT NULL CHECK (trim(company) <> ''),
    role                 TEXT    NOT NULL CHECK (trim(role) <> ''),
    description          TEXT    NOT NULL DEFAULT '',
    link                 TEXT    NOT NULL DEFAULT '',
    salary               TEXT    NOT NULL DEFAULT '',
    location             TEXT    NOT NULL DEFAULT '',
    notes                TEXT    NOT NULL DEFAULT '',
    status               TEXT    NOT NULL DEFAULT 'saved'
                                 CHECK (status IN ('saved', 'applied', 'interviewing', 'offer', 'closed')),
    -- Order within the status column, from 0 at the top. See
    -- docs/adr/0015-store-card-order-as-an-integer-position.md.
    position             INTEGER NOT NULL CHECK (position >= 0),
    -- YYYY-MM-DD, or NULL when no date applied is set. date() returns NULL for text it can't
    -- parse, and a CHECK that evaluates to NULL passes, so compare with IS, which is NULL-safe.
    date_applied         TEXT    CHECK (date_applied IS NULL OR date(date_applied) IS date_applied),
    -- Set exactly when the job is closed. 'other' uses closing_reason_other for its text.
    closing_reason       TEXT    CHECK (closing_reason IN
                                        ('rejected', 'withdrew', 'no_response', 'declined_offer', 'position_filled', 'other')),
    closing_reason_other TEXT,

    CHECK ((status = 'closed') = (closing_reason IS NOT NULL)),
    CHECK ((closing_reason IS 'other') = (closing_reason_other IS NOT NULL AND trim(closing_reason_other) <> ''))
);

-- Not unique: renumbering a column shifts positions one row at a time, which would trip a unique
-- constraint partway through the update.
CREATE INDEX jobs_status_position ON jobs (status, position);

-- Documents attached to a job. The file itself is a copy in the data directory, at stored_path
-- relative to it. See docs/adr/0008-store-attached-documents-as-files-in-the-app-data-directory.md.
-- Deleting a job deletes these rows, but the code must also delete the files.
CREATE TABLE documents (
    id          INTEGER PRIMARY KEY,
    job_id      INTEGER NOT NULL REFERENCES jobs (id) ON DELETE CASCADE,
    kind        TEXT    NOT NULL CHECK (kind IN ('resume', 'cover_letter', 'other')),
    -- The name of the file the user attached, as shown in the app.
    file_name   TEXT    NOT NULL CHECK (file_name <> ''),
    stored_path TEXT    NOT NULL UNIQUE
);

CREATE INDEX documents_job_id ON documents (job_id);

-- +goose Down

DROP TABLE documents;
DROP TABLE jobs;
