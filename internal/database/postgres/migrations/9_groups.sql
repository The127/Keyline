-- +migrate Up

create table groups (
    "id" uuid not null,
    "audit_created_at" timestamp not null,
    "audit_updated_at" timestamp not null,

    "virtual_server_id" uuid not null,

    "name" text not null,
    "description" text not null,

    primary key ("id"),
    foreign key ("virtual_server_id") references "virtual_servers" ("id")
);

create trigger "trg_set_audit_updated_at"
    before update
    on "groups"
    for each row
    execute function update_audit_timestamp();

-- +migrate Down
