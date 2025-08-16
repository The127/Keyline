-- +migrate Up

create table roles (
    "id" uuid not null default gen_random_uuid(),
    "audit_created_at" timestamp not null default now(),
    "audit_updated_at" timestamp not null default now(),

    "virtual_server_id" uuid not null,
    "application_id" uuid,

    "name" text not null,
    "description" text not null,

    "require_mfa" bool not null,
    "max_token_age" interval,

    primary key ("id")
);

alter table "roles" add constraint "fk_roles_virtual_servers"
    foreign key ("virtual_server_id")
    references "virtual_servers" ("id");

alter table "roles" add constraint "fk_roles_applications"
    foreign key ("application_id")
    references "applications" ("id");

create trigger "trg_set_audit_updated_at"
    before update
    on "roles"
    for each row
    execute function update_audit_timestamp();

create table groups (
    "id" uuid not null default gen_random_uuid(),
    "audit_created_at" timestamp not null default now(),
    "audit_updated_at" timestamp not null default now(),

    "virtual_server_id" uuid not null,

    "name" text not null,
    "description" text not null,

    primary key ("id")
);

alter table "groups" add constraint "fk_groups_virtual_servers"
    foreign key ("virtual_server_id")
    references "virtual_servers" ("id");

create trigger "trg_set_audit_updated_at"
    before update
    on "groups"
    for each row
    execute function update_audit_timestamp();

create table "group_roles" (
    "id" uuid not null default gen_random_uuid(),
    "audit_created_at" timestamp not null default now(),
    "audit_updated_at" timestamp not null default now(),
    "group_id" uuid not null,
    "role_id" uuid not null
);

alter table "group_roles" add constraint "fk_group_roles_groups"
    foreign key ("group_id")
    references "groups" ("id");

alter table "group_roles" add constraint "fk_group_roles_roles"
    foreign key ("role_id")
    references "roles" ("id");

create unique index "idx_group_roles_group_id_role_id"
    on "group_roles" ("group_id", "role_id");

create trigger "trg_set_audit_updated_at"
    before update
    on "group_roles"
    for each row
    execute function update_audit_timestamp();

-- +migrate Down
