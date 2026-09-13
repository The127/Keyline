-- +migrate Up
ALTER TABLE applications ADD COLUMN trusted_exchangers text[] NOT NULL DEFAULT '{}';

-- +migrate Down
ALTER TABLE applications DROP COLUMN trusted_exchangers;
