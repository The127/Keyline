-- +migrate Up

create table "audit_logs" (
    "id" uuid not null,
    "audit_created_at" timestamp not null,
    "audit_updated_at" timestamp not null,

    "virtual_server_id" uuid not null,
    "user_id" uuid,

    "request_type" text not null,
    "request" jsonb not null,
    "response" jsonb,

    "allowed" boolean not null,
    "allow_reason_type" text,
    "allow_reason" jsonb,

    primary key ("id"),
    foreign key ("user_id") references "users" ("id") on delete set null
);

create trigger "trg_set_audit_updated_at"
    before update
    on "audit_logs"
    for each row
execute function update_audit_timestamp();

-- +migrate Down
