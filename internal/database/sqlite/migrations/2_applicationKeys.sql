-- +migrate Up

create table "application_keys"
(
    "id"               text      not null,
    "audit_created_at" timestamp not null,
    "audit_updated_at" timestamp not null,
    "version"          integer   not null default 0,

    "application_id"   text      not null,

    "kid"              text      not null,
    "public_key"       text      not null,

    primary key ("id"),
    foreign key ("application_id") references "applications" ("id") on delete cascade,
    unique ("application_id", "kid")
);

-- +migrate StatementBegin
create trigger "trg_application_keys_audit_updated_at"
    after update
    on "application_keys"
    for each row
begin
    update "application_keys" set "audit_updated_at" = strftime('%Y-%m-%d %H:%M:%f+00:00', 'now') where "id" = new."id";
end;
-- +migrate StatementEnd

-- +migrate Down
drop table "application_keys";
