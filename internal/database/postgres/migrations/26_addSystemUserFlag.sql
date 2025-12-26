-- +migrate Up

alter table users add column service_user bool default false;

-- +migrate Down
