-- +migrate Up

create table sessions (
    "id" uuid not null default gen_random_uuid(),
    "audit_created_at" timestamp not null default now(),
    "audit_updated_at" timestamp not null default now(),

    "virtual_server_id" uuid not null,
    "user_id" uuid not null,
    "hashed_token" text not null,
    "expires_at" timestamp not null,
    "last_used_at" timestamp,

    primary key ("id")
);

alter table "sessions"
    add constraint "fk_sessions_virtual_server_id"
    foreign key ("virtual_server_id")
    references "virtual_servers" ("id");

alter table "sessions"
    add constraint "fk_sessions_user_id"
    foreign key ("user_id")
    references "users" ("id");

create trigger "trg_set_audit_updated_at"
    before update
    on "sessions"
    for each row
    execute function update_audit_timestamp();

-- +migrate Down
