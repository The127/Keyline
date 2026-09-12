-- +migrate Up

create table identity_providers (
    "id" uuid not null,
    "audit_created_at" timestamp not null,
    "audit_updated_at" timestamp not null,

    "virtual_server_id" uuid not null,

    "name" text not null,
    "display_name" text not null,

    primary key ("id"),
    foreign key ("virtual_server_id") references "virtual_servers" ("id") on delete cascade,
    unique ("virtual_server_id", "name")
);

create trigger "trg_set_audit_updated_at"
    before update
    on "identity_providers"
    for each row
    execute function update_audit_timestamp();

-- +migrate Down
drop table identity_providers;
