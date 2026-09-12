-- +migrate Up

create table "identity_providers"
(
    "id"                text      not null,
    "audit_created_at"  timestamp not null,
    "audit_updated_at"  timestamp not null,
    "version"           integer   not null default 0,

    "virtual_server_id" text      not null,

    "name"              text      not null,
    "display_name"      text      not null,
    "preset"            text      not null default '',
    "issuer"            text      not null default '',

    "authorization_endpoint" text not null,
    "token_endpoint"         text not null,
    "userinfo_endpoint"      text not null,
    "scopes"                 text not null default '[]',
    "client_id"              text not null,
    "client_secret"          text not null,
    "claim_mapping"          text not null default '{}',

    primary key ("id"),
    foreign key ("virtual_server_id") references "virtual_servers" ("id") on delete cascade,
    unique ("virtual_server_id", "name")
);

-- +migrate StatementBegin
create trigger "trg_identity_providers_audit_updated_at"
    after update
    on "identity_providers"
    for each row
begin
    update "identity_providers" set "audit_updated_at" = strftime('%Y-%m-%d %H:%M:%f+00:00', 'now') where "id" = new."id";
end;
-- +migrate StatementEnd

-- +migrate Down
drop table "identity_providers";
