-- +migrate Up

create table sessions (
    "id" uuid not null,
    "audit_created_at" timestamp not null,
    "audit_updated_at" timestamp not null,

    "virtual_server_id" uuid not null,
    "user_id" uuid not null,
    "hashed_token" text not null,
    "expires_at" timestamp not null,
    "last_used_at" timestamp,

    primary key ("id"),
    foreign key ("virtual_server_id") references "virtual_servers" ("id"),
    foreign key ("user_id") references "users" ("id")
);

create trigger "trg_set_audit_updated_at"
    before update
    on "sessions"
    for each row
    execute function update_audit_timestamp();

-- +migrate Down
