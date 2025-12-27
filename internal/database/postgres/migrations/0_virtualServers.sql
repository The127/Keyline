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
    "id"                         uuid      not null,
    "audit_created_at"           timestamp not null,
    "audit_updated_at"           timestamp not null,

    "name"                       text      not null,
    "display_name"               text      not null,

    "enable_registration"        bool      not null default false,
    "require_2fa"                bool      not null default true,
    "require_email_verification" bool      not null default true,

    "signing_algorithm" text not null default 'EdDSA',

    "mail_host"                  text,
    "mail_port"                  integer,
    "mail_username"              text,
    "mail_encrypted_password"    text,

    primary key ("id"),
    unique ("name")
);

create trigger "trg_set_audit_updated_at"
    before update
    on "virtual_servers"
    for each row
execute function update_audit_timestamp();

-- +migrate Down
