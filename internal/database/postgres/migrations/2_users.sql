-- +migrate Up

create table "users"
(
    "id" uuid not null,
    "audit_created_at" timestamp not null,
    "audit_updated_at" timestamp not null,

    "virtual_server_id" uuid, -- null indicates system user

    "service_user" bool not null default false,

    "display_name" text not null,
    "username" text not null,

    primary_email text not null,
    email_verified bool not null,

    metadata jsonb not null default '{}',

    primary key ("id"),
    foreign key ("virtual_server_id") references virtual_servers("id"),
    unique ("username", "virtual_server_id")
);

create trigger "trg_set_audit_updated_at"
    before update
    on "users"
    for each row
execute function update_audit_timestamp();

-- +migrate Down
