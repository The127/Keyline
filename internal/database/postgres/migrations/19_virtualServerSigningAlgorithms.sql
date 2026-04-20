-- +migrate Up
ALTER TABLE virtual_servers
    ADD COLUMN primary_signing_algorithm TEXT NOT NULL DEFAULT 'EdDSA',
    ADD COLUMN additional_signing_algorithms TEXT[] NOT NULL DEFAULT '{}';
UPDATE virtual_servers SET primary_signing_algorithm = signing_algorithm;
ALTER TABLE virtual_servers DROP COLUMN signing_algorithm;

-- +migrate Down
ALTER TABLE virtual_servers ADD COLUMN signing_algorithm TEXT NOT NULL DEFAULT 'EdDSA';
UPDATE virtual_servers SET signing_algorithm = primary_signing_algorithm;
ALTER TABLE virtual_servers
    DROP COLUMN primary_signing_algorithm,
    DROP COLUMN additional_signing_algorithms;
