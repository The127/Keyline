-- +migrate Up

alter table virtual_servers add column require_email_verification bool not null default true;

-- +migrate Down
