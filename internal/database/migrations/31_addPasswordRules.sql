-- +migrate Up

create table password_rules(
    "id" uuid not null default gen_random_uuid(),
    "audit_created_at" timestamp not null default now(),
    "audit_updated_at" timestamp not null default now(),
    "version" bigint not null default 1,

    "virtual_server_id" uuid not null,

    "type" text not null,
    "details" jsonb not null,

    primary key ("id")
);

alter table "password_rules"
    add constraint "fk_password_rules_virtual_server"
    foreign key ("virtual_server_id")
    references "virtual_servers" ("id");

create unique index idx_unique_rule_per_virtual_sever on password_rules (virtual_server_id, type);

create trigger "trg_set_audit_updated_at"
    before update
    on "password_rules"
    for each row
    execute function update_audit_timestamp();

-- +migrate Down
