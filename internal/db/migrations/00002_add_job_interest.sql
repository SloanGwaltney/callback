-- +goose Up

-- How interested the user is in a job, from 1 to 5 stars, or NULL when it isn't rated.
ALTER TABLE jobs ADD COLUMN interest INTEGER CHECK (interest IS NULL OR interest BETWEEN 1 AND 5);

-- +goose Down

ALTER TABLE jobs DROP COLUMN interest;
