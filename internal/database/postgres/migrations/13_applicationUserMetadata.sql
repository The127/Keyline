-- +migrate Up

create table application_user_metadata(
    "id" uuid not null,
    "audit_created_at" timestamp not null,
    "audit_updated_at" timestamp not null,

    "application_id" uuid not null,
    "user_id" uuid not null,

    "metadata" jsonb not null,

    primary key ("id"),
    foreign key ("application_id") references "applications" ("id"),
    foreign key ("user_id") references "users" ("id"),
    unique ("application_id", "user_id")
);

create trigger "trg_set_audit_updated_at"
    before update
    on "application_user_metadata"
    for each row
    execute function update_audit_timestamp();

-- +migrate Down
