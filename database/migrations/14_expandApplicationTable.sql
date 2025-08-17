-- +migrate Up

alter table applications add column name text not null;
alter table applications add column redirect_uris text[] not null;
alter table applications add column hashed_secret text not null;

create unique index idx_application_unique_name on applications (name, virtual_server_id);

-- +migrate Down
