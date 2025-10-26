-- +migrate Up

create table resource_server_scopes (
    "id" uuid not null default gen_random_uuid(),
    "audit_created_at" timestamp not null default now(),
    "audit_updated_at" timestamp not null default now(),
    "version" bigint not null default 1,

    "virtual_server_id" uuid not null,
    "project_id" uuid not null,
    "resource_server_id" uuid not null,

    "scope" text not null,
    "name" text not null,
    "description" text,

    primary key ("id")
);

alter table "resource_server_scopes"
    add constraint "fk_resource_server_scopes_virtual_server_id"
    foreign key ("virtual_server_id")
    references "virtual_servers" ("id");

alter table "resource_server_scopes"
    add constraint "fk_resource_server_scopes_project_id"
    foreign key ("project_id")
    references "projects" ("id");

alter table "resource_server_scopes"
    add constraint "fk_resource_server_scopes_resource_server_id"
    foreign key ("resource_server_id")
    references "resource_servers" ("id");

create unique index "idx_resource_server_scopes_unique"
    on "resource_server_scopes" ("resource_server_id", "scope");

create trigger "trg_set_audit_updated_at"
    before update
    on "resource_server_scopes"
    for each row
execute function update_audit_timestamp();

-- +migrate Down

