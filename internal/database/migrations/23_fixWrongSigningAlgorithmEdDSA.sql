-- +migrate Up

UPDATE virtual_servers
SET signing_algorithm = 'EdDSA'
WHERE signing_algorithm = 'ECDSA';

ALTER TABLE virtual_servers
    ALTER COLUMN signing_algorithm SET DEFAULT 'EdDSA';

-- +migrate Down

UPDATE virtual_servers
SET signing_algorithm = 'ECDSA'
WHERE signing_algorithm = 'EdDSA';

ALTER TABLE virtual_servers
    ALTER COLUMN signing_algorithm SET DEFAULT 'ECDSA';
