-- +migrate Up
ALTER TABLE applications ADD COLUMN signing_algorithm TEXT NULL;

-- +migrate Down
ALTER TABLE applications DROP COLUMN signing_algorithm;
