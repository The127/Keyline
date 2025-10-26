-- +migrate Up

create table projects (
    "id" uuid not null default gen_random_uuid(),
    "audit_created_at" timestamp not null default now(),
    "audit_updated_at" timestamp not null default now(),
    "version" bigint not null default 1,

    "virtual_server_id" uuid not null,

    "slug" text not null,
    "name" text not null,
    "description" text not null,

    primary key ("id")
);

alter table "projects"
    add constraint "fk_projects_virtual_server_id"
    foreign key ("virtual_server_id")
    references "virtual_servers" ("id");

create unique index "idx_projects_virtual_server_id_slug" on "projects" ("virtual_server_id", "slug");

create trigger "trg_set_audit_updated_at"
    before update
    on "projects"
    for each row
    execute function update_audit_timestamp();

alter table applications add column "project_id" uuid not null;
alter table applications
    add constraint "fk_applications_projects"
    foreign key ("project_id")
    references "projects" ("id");

alter table roles add column "project_id" uuid not null;
alter table roles
    add constraint "fk_roles_projects"
    foreign key ("project_id")
    references "projects" ("id");

-- +migrate Down

