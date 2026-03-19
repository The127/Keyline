-- +migrate Up

alter table applications add column device_flow_enabled boolean not null default false;

-- +migrate Down

