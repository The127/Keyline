-- +migrate Up
ALTER TABLE applications ADD COLUMN userinfo_in_access_token BOOLEAN NOT NULL DEFAULT FALSE;

-- +migrate Down
ALTER TABLE applications DROP COLUMN userinfo_in_access_token;
