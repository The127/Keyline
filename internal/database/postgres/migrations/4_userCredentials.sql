-- +migrate Up

create table "credentials"(
    "id" uuid not null,
    "audit_created_at" timestamp not null,
    "audit_updated_at" timestamp not null,

    "user_id" uuid not null,

    "type" text not null,
    "details" jsonb not null,

    primary key ("id"),
    foreign key ("user_id") references users ("id")
);

create trigger "trg_set_audit_updated_at"
    before update
    on "credentials"
    for each row
    execute function update_audit_timestamp();

-- +migrate Down
