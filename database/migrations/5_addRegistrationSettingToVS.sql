-- +migrate Up

alter table public.virtual_servers
    add "enable_registration" bool default false not null;

-- +migrate Down
