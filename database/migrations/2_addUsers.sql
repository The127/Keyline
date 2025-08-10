-- +migrate Up

create table "users"
(
    "id" uuid not null default gen_random_uuid(),
    "audit_created_at" timestamp not null default now(),
    "audit_updated_at" timestamp not null default now(),

    "virtual_server_id" uuid not null,

    "display_name" text not null,

    primary key ("id")
);

alter table "users" add constraint "fk_users_virtual_servers" foreign key ("virtual_server_id") references virtual_servers ("id");

-- +migrate Down
