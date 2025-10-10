-- +migrate Up

create table "user_role_assignments" (
    "id" uuid not null default gen_random_uuid(),
    "audit_created_at" timestamp not null default now(),
    "audit_updated_at" timestamp not null default now(),

    "user_id" uuid not null,
    "role_id" uuid not null,
    "group_id" uuid,

    primary key ("id")
);

alter table "user_role_assignments" add constraint "fk_user_role_assignments_users" foreign key ("user_id") references "users" ("id");
alter table "user_role_assignments" add constraint "fk_user_role_assignments_roles" foreign key ("role_id") references "roles" ("id");
alter table "user_role_assignments" add constraint "fk_user_role_assignments_groups" foreign key ("group_id") references "groups" ("id");

create unique index "idx_user_role_assignments_user_id_role_id_group_id" on "user_role_assignments" ("user_id", "role_id", "group_id");

create trigger "trg_set_audit_updated_at"
    before update
    on "user_role_assignments"
    for each row
    execute function update_audit_timestamp();

-- +migrate Down
