-- +migrate Up

alter table applications add column type text not null default 'confidential';

-- +migrate Down
