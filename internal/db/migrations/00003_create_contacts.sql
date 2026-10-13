-- +goose Up

-- People the user deals with on a job. Each job keeps its own contacts, so the same person on two
-- jobs is two rows, edited separately. Optional text fields default to '' rather than NULL, like
-- the jobs table.
CREATE TABLE contacts (
    id           INTEGER PRIMARY KEY,
    job_id       INTEGER NOT NULL REFERENCES jobs (id) ON DELETE CASCADE,
    name         TEXT    NOT NULL CHECK (trim(name) <> ''),
    relationship TEXT    NOT NULL
                         CHECK (relationship IN ('recruiter', 'hiring_manager', 'referrer', 'interviewer', 'other')),
    email        TEXT    NOT NULL DEFAULT '',
    -- Free text, written however the user has it.
    phone        TEXT    NOT NULL DEFAULT '',
    profile_link TEXT    NOT NULL DEFAULT '',
    notes        TEXT    NOT NULL DEFAULT ''
);

CREATE INDEX contacts_job_id ON contacts (job_id);

-- +goose Down

DROP TABLE contacts;
