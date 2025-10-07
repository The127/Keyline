-- +migrate Up

alter table virtual_servers add column signing_algorithm text;

-- +migrate Down
