-- +migrate Up

create table "group_roles" (
    "id" uuid not null,
    "audit_created_at" timestamp not null,
    "audit_updated_at" timestamp not null,
    "group_id" uuid not null,
    "role_id" uuid not null,

    primary key ("id"),
    foreign key ("group_id") references "groups" ("id"),
    foreign key ("role_id") references "roles" ("id"),
    unique ("group_id", "role_id")
);

create trigger "trg_set_audit_updated_at"
    before update
    on "group_roles"
    for each row
    execute function update_audit_timestamp();

-- +migrate Down
