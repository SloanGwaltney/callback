-- +goose Up

-- What the user is waiting for or needs to do next on a job, and when they expect it, as
-- YYYY-MM-DD. Both are NULL when the job has no next action. An expected date needs a next action,
-- and a next action can't be blank.
ALTER TABLE jobs ADD COLUMN next_action TEXT CHECK (next_action IS NULL OR trim(next_action) <> '');
ALTER TABLE jobs ADD COLUMN next_action_date TEXT CHECK (
    next_action_date IS NULL OR (next_action IS NOT NULL AND date(next_action_date) IS next_action_date)
);

-- +goose Down

ALTER TABLE jobs DROP COLUMN next_action_date;
ALTER TABLE jobs DROP COLUMN next_action;
