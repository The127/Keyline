-- +migrate Up

create table application_keys (
    "id" uuid not null,
    "audit_created_at" timestamp not null,
    "audit_updated_at" timestamp not null,

    "application_id" uuid not null,

    "kid" text not null,
    "public_key" text not null,

    primary key ("id"),
    foreign key ("application_id") references "applications" ("id") on delete cascade,
    unique ("application_id", "kid")
);

create trigger "trg_set_audit_updated_at"
    before update
    on "application_keys"
    for each row
    execute function update_audit_timestamp();

-- +migrate Down
drop table application_keys;
