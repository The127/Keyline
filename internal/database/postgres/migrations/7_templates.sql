-- +migrate Up

create table templates (
    "id" uuid not null,
    "audit_created_at" timestamp not null,
    "audit_updated_at" timestamp not null,

    "virtual_server_id" uuid not null,
    "file_id" uuid not null,
    "type" text not null,

    primary key ("id"),
    foreign key ("virtual_server_id") references virtual_servers ("id"),
    foreign key ("file_id") references files ("id")
);

create trigger "trg_set_audit_updated_at"
    before update
    on "templates"
    for each row
    execute function update_audit_timestamp();

-- +migrate Down
