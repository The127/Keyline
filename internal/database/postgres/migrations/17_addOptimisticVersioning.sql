-- +migrate Up

alter table "applications" add column "version" bigint not null default 1;
alter table "credentials" add column "version" bigint not null default 1;
alter table "files" add column "version" bigint not null default 1;
alter table "group_roles" add column "version" bigint not null default 1;
alter table "groups" add column "version" bigint not null default 1;
alter table "outbox_messages" add column "version" bigint not null default 1;
alter table "roles" add column "version" bigint not null default 1;
alter table "sessions" add column "version" bigint not null default 1;
alter table "templates" add column "version" bigint not null default 1;
alter table "user_role_assignments" add column "version" bigint not null default 1;
alter table "users" add column "version" bigint not null default 1;
alter table "virtual_servers" add column "version" bigint not null default 1;

-- +migrate Down
