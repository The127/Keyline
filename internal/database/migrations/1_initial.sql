-- +migrate Up

-- +migrate StatementBegin
create or replace function "update_audit_timestamp"()
    returns trigger as
$$
begin
    new."audit_updated_at" = now();
    return new;
end;
$$ language plpgsql;
-- +migrate StatementEnd

create table "virtual_servers"
(
    "id" uuid not null default gen_random_uuid(),
    "audit_created_at" timestamp not null default now(),
    "audit_updated_at" timestamp not null default now(),

    "name" text not null,
    "display_name" text not null,

    primary key ("id")
);

create unique index "idx_unique_virtual_server_name" on "virtual_servers" ("name");

-- +migrate Down
