-- +goose Up

-- Each job's status history: one row for the status it was added to, then one for each move to
-- another status, with the date as YYYY-MM-DD. Rows are only ever added, so IDs run oldest to
-- newest, including several changes on the same day. A move to Closed keeps the closing reason
-- chosen then, which later changes to the job's reason don't rewrite. The checks match the jobs
-- table's.
CREATE TABLE status_changes (
    id                   INTEGER PRIMARY KEY,
    job_id               INTEGER NOT NULL REFERENCES jobs (id) ON DELETE CASCADE,
    status               TEXT    NOT NULL
                                 CHECK (status IN ('saved', 'applied', 'interviewing', 'offer', 'closed')),
    date                 TEXT    NOT NULL CHECK (date(date) IS date),
    closing_reason       TEXT    CHECK (closing_reason IN
                                        ('rejected', 'withdrew', 'no_response', 'declined_offer', 'position_filled', 'other')),
    closing_reason_other TEXT,

    CHECK ((status = 'closed') = (closing_reason IS NOT NULL)),
    CHECK ((closing_reason IS 'other') = (closing_reason_other IS NOT NULL AND trim(closing_reason_other) <> ''))
);

CREATE INDEX status_changes_job_id ON status_changes (job_id);

-- Jobs added before the history existed start it with the status they're in now. When they got
-- there wasn't recorded, so the date is the day this migration runs.
INSERT INTO status_changes (job_id, status, date, closing_reason, closing_reason_other)
SELECT id, status, date('now', 'localtime'), closing_reason, closing_reason_other
FROM jobs
ORDER BY id;

-- +goose Down

DROP TABLE status_changes;
