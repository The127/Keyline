-- +migrate Up

alter table virtual_servers add column signing_algorithm text not null default 'ECDSA';

-- +migrate Down
