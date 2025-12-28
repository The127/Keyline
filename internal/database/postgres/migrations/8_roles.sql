-- +migrate Up

create table roles (
    "id" uuid not null,
    "audit_created_at" timestamp not null,
    "audit_updated_at" timestamp not null,

    "virtual_server_id" uuid not null,
    "project_id" uuid not null,

    "name" text not null,
    "description" text not null,

    primary key ("id"),
    foreign key ("virtual_server_id") references "virtual_servers" ("id"),
    foreign key ("project_id") references "projects" ("id")
);

create trigger "trg_set_audit_updated_at"
    before update
    on "roles"
    for each row
    execute function update_audit_timestamp();

-- +migrate Down
