-- +migrate Up
create table "resource_server_scopes_new"
(
    "id"                 text      not null,
    "audit_created_at"   timestamp not null,
    "audit_updated_at"   timestamp not null,
    "version"            integer   not null default 0,

    "virtual_server_id"  text      not null,
    "project_id"         text      not null,
    "resource_server_id" text      not null,

    "scope"              text      not null,
    "name"               text      not null,
    "description"        text,

    primary key ("id"),
    foreign key ("virtual_server_id") references "virtual_servers" ("id"),
    foreign key ("project_id") references "projects" ("id"),
    foreign key ("resource_server_id") references "resource_servers" ("id"),
    unique ("project_id", "scope")
);

insert into "resource_server_scopes_new"
select "id", "audit_created_at", "audit_updated_at", "version",
       "virtual_server_id", "project_id", "resource_server_id",
       "scope", "name", "description"
from "resource_server_scopes";

drop table "resource_server_scopes";

alter table "resource_server_scopes_new" rename to "resource_server_scopes";

-- +migrate StatementBegin
create trigger "trg_resource_server_scopes_audit_updated_at"
    after update
    on "resource_server_scopes"
    for each row
begin
    update "resource_server_scopes" set "audit_updated_at" = strftime('%Y-%m-%d %H:%M:%f+00:00', 'now') where "id" = new."id";
end;
-- +migrate StatementEnd

-- +migrate Down
create table "resource_server_scopes_old"
(
    "id"                 text      not null,
    "audit_created_at"   timestamp not null,
    "audit_updated_at"   timestamp not null,
    "version"            integer   not null default 0,

    "virtual_server_id"  text      not null,
    "project_id"         text      not null,
    "resource_server_id" text      not null,

    "scope"              text      not null,
    "name"               text      not null,
    "description"        text,

    primary key ("id"),
    foreign key ("virtual_server_id") references "virtual_servers" ("id"),
    foreign key ("project_id") references "projects" ("id"),
    foreign key ("resource_server_id") references "resource_servers" ("id"),
    unique ("virtual_server_id", "scope")
);

insert into "resource_server_scopes_old"
select "id", "audit_created_at", "audit_updated_at", "version",
       "virtual_server_id", "project_id", "resource_server_id",
       "scope", "name", "description"
from "resource_server_scopes";

drop table "resource_server_scopes";

alter table "resource_server_scopes_old" rename to "resource_server_scopes";

-- +migrate StatementBegin
create trigger "trg_resource_server_scopes_audit_updated_at"
    after update
    on "resource_server_scopes"
    for each row
begin
    update "resource_server_scopes" set "audit_updated_at" = strftime('%Y-%m-%d %H:%M:%f+00:00', 'now') where "id" = new."id";
end;
-- +migrate StatementEnd
