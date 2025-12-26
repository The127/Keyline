-- +migrate Up

create table resource_servers (
    "id" uuid not null default gen_random_uuid(),
    "audit_created_at" timestamp not null default now(),
    "audit_updated_at" timestamp not null default now(),
    "version" bigint not null default 1,

    "virtual_server_id" uuid not null,
    "project_id" uuid not null,

    "name" text not null,
    "description" text not null,

    primary key ("id")
);

alter table "resource_servers"
    add constraint "fk_resource_servers_virtual_servers"
    foreign key ("virtual_server_id")
    references "virtual_servers" ("id");

alter table "resource_servers"
    add constraint "fk_resource_servers_projects"
    foreign key ("project_id")
    references "projects" ("id");

create trigger "trg_set_audit_updated_at"
    before update
    on "resource_servers"
    for each row
    execute function update_audit_timestamp();

-- +migrate Down

