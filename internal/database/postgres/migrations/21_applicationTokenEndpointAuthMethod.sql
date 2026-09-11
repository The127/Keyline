-- +migrate Up
ALTER TABLE applications ADD COLUMN token_endpoint_auth_method TEXT NULL;
UPDATE applications SET token_endpoint_auth_method = 'client_secret' WHERE type = 'confidential';

-- +migrate Down
ALTER TABLE applications DROP COLUMN token_endpoint_auth_method;
