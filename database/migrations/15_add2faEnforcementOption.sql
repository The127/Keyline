-- +migrate Up

alter table virtual_servers
    add column require_2fa bool default true;

-- +migrate Down
