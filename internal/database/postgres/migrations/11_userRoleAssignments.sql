-- +migrate Up

create table "user_role_assignments" (
    "id" uuid not null,
    "audit_created_at" timestamp not null,
    "audit_updated_at" timestamp not null,

    "user_id" uuid not null,
    "role_id" uuid not null,
    "group_id" uuid,

    primary key ("id"),
    foreign key ("user_id") references "users" ("id"),
    foreign key ("role_id") references "roles" ("id"),
    foreign key ("group_id") references "groups" ("id"),
    unique ("user_id", "role_id", "group_id")
);

create trigger "trg_set_audit_updated_at"
    before update
    on "user_role_assignments"
    for each row
    execute function update_audit_timestamp();

-- +migrate Down
