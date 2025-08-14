-- +migrate Up

create table "outbox_messages"
(
    "id" uuid not null default gen_random_uuid(),
    "audit_created_at" timestamp not null default now(),
    "audit_updated_at" timestamp not null default now(),

    "type" text not null,
    "details" jsonb not null,

    primary key ("id")
);

create trigger "trg_set_audit_updated_at"
    before update
    on "outbox_messages"
    for each row
    execute function update_audit_timestamp();

-- +migrate Down
