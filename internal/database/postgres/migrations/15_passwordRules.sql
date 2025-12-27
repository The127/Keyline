-- +migrate Up

create table password_rules(
    "id" uuid not null,
    "audit_created_at" timestamp not null,
    "audit_updated_at" timestamp not null,

    "virtual_server_id" uuid not null,

    "type" text not null,
    "details" jsonb not null,

    primary key ("id"),
    foreign key ("virtual_server_id") references "virtual_servers" ("id"),
    unique ("virtual_server_id", "type")
);

create trigger "trg_set_audit_updated_at"
    before update
    on "password_rules"
    for each row
    execute function update_audit_timestamp();

-- +migrate Down
