-- +goose Up

-- Interviews for a job, past and upcoming. Dates are YYYY-MM-DD and times HH:MM on a 24-hour
-- clock, so both sort as text. An interview with no time has time ''. Notes default to '' rather
-- than NULL, like the jobs table.
CREATE TABLE interviews (
    id     INTEGER PRIMARY KEY,
    job_id INTEGER NOT NULL REFERENCES jobs (id) ON DELETE CASCADE,
    date   TEXT    NOT NULL CHECK (date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
    time   TEXT    NOT NULL DEFAULT '' CHECK (time = '' OR time GLOB '[0-2][0-9]:[0-5][0-9]'),
    kind   TEXT    NOT NULL
                   CHECK (kind IN ('phone_screen', 'technical', 'behavioral', 'panel', 'onsite', 'other')),
    notes  TEXT    NOT NULL DEFAULT ''
);

CREATE INDEX interviews_job_id ON interviews (job_id);

-- +goose Down

DROP TABLE interviews;
