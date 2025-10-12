-- +migrate Up

alter table users alter column virtual_server_id drop not null;

-- +migrate Down
