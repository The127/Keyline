-- +migrate Up

alter table applications add column post_logout_redirect_uris text[] not null default '{}';

-- +migrate Down
