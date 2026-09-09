-- +migrate Up

create table "virtual_servers"
(
    "id"                            text      not null,
    "audit_created_at"              timestamp not null,
    "audit_updated_at"              timestamp not null,
    "version"                       integer   not null default 0,

    "name"                          text      not null,
    "display_name"                  text      not null,

    "enable_registration"           boolean   not null default false,
    "require_2fa"                   boolean   not null default true,
    "require_email_verification"    boolean   not null default true,

    "primary_signing_algorithm"     text      not null default 'EdDSA',
    "additional_signing_algorithms" text      not null default '[]',

    "mail_host"                     text,
    "mail_port"                     integer,
    "mail_username"                 text,
    "mail_encrypted_password"       text,

    primary key ("id"),
    unique ("name")
);

-- +migrate StatementBegin
create trigger "trg_virtual_servers_audit_updated_at"
    after update
    on "virtual_servers"
    for each row
begin
    update "virtual_servers" set "audit_updated_at" = strftime('%Y-%m-%d %H:%M:%f+00:00', 'now') where "id" = new."id";
end;
-- +migrate StatementEnd

create table "projects"
(
    "id"                text      not null,
    "audit_created_at"  timestamp not null,
    "audit_updated_at"  timestamp not null,
    "version"           integer   not null default 0,

    "virtual_server_id" text      not null,

    "system_project"    boolean   not null default false,

    "slug"              text      not null,
    "name"              text      not null,
    "description"       text      not null,

    primary key ("id"),
    foreign key ("virtual_server_id") references "virtual_servers" ("id"),
    unique ("virtual_server_id", "slug")
);

-- +migrate StatementBegin
create trigger "trg_projects_audit_updated_at"
    after update
    on "projects"
    for each row
begin
    update "projects" set "audit_updated_at" = strftime('%Y-%m-%d %H:%M:%f+00:00', 'now') where "id" = new."id";
end;
-- +migrate StatementEnd

create table "users"
(
    "id"                text      not null,
    "audit_created_at"  timestamp not null,
    "audit_updated_at"  timestamp not null,
    "version"           integer   not null default 0,

    "virtual_server_id" text, -- null indicates system user

    "service_user"      boolean   not null default false,

    "display_name"      text      not null,
    "username"          text      not null,

    "primary_email"     text      not null,
    "email_verified"    boolean   not null,

    "metadata"          text      not null default '{}',

    primary key ("id"),
    foreign key ("virtual_server_id") references "virtual_servers" ("id"),
    unique ("username", "virtual_server_id")
);

-- +migrate StatementBegin
create trigger "trg_users_audit_updated_at"
    after update
    on "users"
    for each row
begin
    update "users" set "audit_updated_at" = strftime('%Y-%m-%d %H:%M:%f+00:00', 'now') where "id" = new."id";
end;
-- +migrate StatementEnd

create table "applications"
(
    "id"                        text      not null,
    "audit_created_at"          timestamp not null,
    "audit_updated_at"          timestamp not null,
    "version"                   integer   not null default 0,

    "virtual_server_id"         text      not null,
    "project_id"                text      not null,

    "display_name"              text      not null,
    "name"                      text      not null,

    "system_application"        boolean   not null default false,

    "type"                      text      not null default 'confidential',
    "hashed_secret"             text      not null,

    "redirect_uris"             text      not null,
    "post_logout_redirect_uris" text      not null default '[]',

    "claims_mapping_script"     text,
    "access_token_header_type"  text      not null default 'at+jwt',

    "device_flow_enabled"       boolean   not null default false,
    "signing_algorithm"         text,

    primary key ("id"),
    foreign key ("virtual_server_id") references "virtual_servers" ("id"),
    foreign key ("project_id") references "projects" ("id"),
    unique ("name", "virtual_server_id")
);

-- +migrate StatementBegin
create trigger "trg_applications_audit_updated_at"
    after update
    on "applications"
    for each row
begin
    update "applications" set "audit_updated_at" = strftime('%Y-%m-%d %H:%M:%f+00:00', 'now') where "id" = new."id";
end;
-- +migrate StatementEnd

create table "credentials"
(
    "id"               text      not null,
    "audit_created_at" timestamp not null,
    "audit_updated_at" timestamp not null,
    "version"          integer   not null default 0,

    "user_id"          text      not null,

    "type"             text      not null,
    "details"          text      not null,

    primary key ("id"),
    foreign key ("user_id") references "users" ("id")
);

-- +migrate StatementBegin
create trigger "trg_credentials_audit_updated_at"
    after update
    on "credentials"
    for each row
begin
    update "credentials" set "audit_updated_at" = strftime('%Y-%m-%d %H:%M:%f+00:00', 'now') where "id" = new."id";
end;
-- +migrate StatementEnd

create table "outbox_messages"
(
    "id"               text      not null,
    "audit_created_at" timestamp not null,
    "audit_updated_at" timestamp not null,
    "version"          integer   not null default 0,

    "type"             text      not null,
    "details"          text      not null,

    primary key ("id")
);

-- +migrate StatementBegin
create trigger "trg_outbox_messages_audit_updated_at"
    after update
    on "outbox_messages"
    for each row
begin
    update "outbox_messages" set "audit_updated_at" = strftime('%Y-%m-%d %H:%M:%f+00:00', 'now') where "id" = new."id";
end;
-- +migrate StatementEnd

create table "files"
(
    "id"               text      not null,
    "audit_created_at" timestamp not null,
    "audit_updated_at" timestamp not null,
    "version"          integer   not null default 0,

    "name"             text      not null,
    "mime_type"        text      not null,
    "content"          blob      not null,

    primary key ("id")
);

-- +migrate StatementBegin
create trigger "trg_files_audit_updated_at"
    after update
    on "files"
    for each row
begin
    update "files" set "audit_updated_at" = strftime('%Y-%m-%d %H:%M:%f+00:00', 'now') where "id" = new."id";
end;
-- +migrate StatementEnd

create table "templates"
(
    "id"                text      not null,
    "audit_created_at"  timestamp not null,
    "audit_updated_at"  timestamp not null,
    "version"           integer   not null default 0,

    "virtual_server_id" text      not null,
    "file_id"           text      not null,

    "type"              text      not null,

    primary key ("id"),
    foreign key ("virtual_server_id") references "virtual_servers" ("id"),
    foreign key ("file_id") references "files" ("id")
);

-- +migrate StatementBegin
create trigger "trg_templates_audit_updated_at"
    after update
    on "templates"
    for each row
begin
    update "templates" set "audit_updated_at" = strftime('%Y-%m-%d %H:%M:%f+00:00', 'now') where "id" = new."id";
end;
-- +migrate StatementEnd

create table "roles"
(
    "id"                text      not null,
    "audit_created_at"  timestamp not null,
    "audit_updated_at"  timestamp not null,
    "version"           integer   not null default 0,

    "virtual_server_id" text      not null,
    "project_id"        text      not null,

    "name"              text      not null,
    "description"       text      not null,

    primary key ("id"),
    foreign key ("virtual_server_id") references "virtual_servers" ("id"),
    foreign key ("project_id") references "projects" ("id")
);

-- +migrate StatementBegin
create trigger "trg_roles_audit_updated_at"
    after update
    on "roles"
    for each row
begin
    update "roles" set "audit_updated_at" = strftime('%Y-%m-%d %H:%M:%f+00:00', 'now') where "id" = new."id";
end;
-- +migrate StatementEnd

create table "groups"
(
    "id"                text      not null,
    "audit_created_at"  timestamp not null,
    "audit_updated_at"  timestamp not null,
    "version"           integer   not null default 0,

    "virtual_server_id" text      not null,

    "name"              text      not null,
    "description"       text      not null,

    primary key ("id"),
    foreign key ("virtual_server_id") references "virtual_servers" ("id")
);

-- +migrate StatementBegin
create trigger "trg_groups_audit_updated_at"
    after update
    on "groups"
    for each row
begin
    update "groups" set "audit_updated_at" = strftime('%Y-%m-%d %H:%M:%f+00:00', 'now') where "id" = new."id";
end;
-- +migrate StatementEnd

create table "group_roles"
(
    "id"               text      not null,
    "audit_created_at" timestamp not null,
    "audit_updated_at" timestamp not null,
    "version"          integer   not null default 0,

    "group_id"         text      not null,
    "role_id"          text      not null,

    primary key ("id"),
    foreign key ("group_id") references "groups" ("id"),
    foreign key ("role_id") references "roles" ("id"),
    unique ("group_id", "role_id")
);

-- +migrate StatementBegin
create trigger "trg_group_roles_audit_updated_at"
    after update
    on "group_roles"
    for each row
begin
    update "group_roles" set "audit_updated_at" = strftime('%Y-%m-%d %H:%M:%f+00:00', 'now') where "id" = new."id";
end;
-- +migrate StatementEnd

create table "user_role_assignments"
(
    "id"               text      not null,
    "audit_created_at" timestamp not null,
    "audit_updated_at" timestamp not null,
    "version"          integer   not null default 0,

    "user_id"          text      not null,
    "role_id"          text      not null,
    "group_id"         text,

    primary key ("id"),
    foreign key ("user_id") references "users" ("id"),
    foreign key ("role_id") references "roles" ("id"),
    foreign key ("group_id") references "groups" ("id"),
    unique ("user_id", "role_id", "group_id")
);

-- +migrate StatementBegin
create trigger "trg_user_role_assignments_audit_updated_at"
    after update
    on "user_role_assignments"
    for each row
begin
    update "user_role_assignments" set "audit_updated_at" = strftime('%Y-%m-%d %H:%M:%f+00:00', 'now') where "id" = new."id";
end;
-- +migrate StatementEnd

create table "sessions"
(
    "id"                text      not null,
    "audit_created_at"  timestamp not null,
    "audit_updated_at"  timestamp not null,
    "version"           integer   not null default 0,

    "virtual_server_id" text      not null,
    "user_id"           text      not null,

    "hashed_token"      text      not null,

    "expires_at"        timestamp not null,
    "last_used_at"      timestamp,

    primary key ("id"),
    foreign key ("virtual_server_id") references "virtual_servers" ("id"),
    foreign key ("user_id") references "users" ("id")
);

-- +migrate StatementBegin
create trigger "trg_sessions_audit_updated_at"
    after update
    on "sessions"
    for each row
begin
    update "sessions" set "audit_updated_at" = strftime('%Y-%m-%d %H:%M:%f+00:00', 'now') where "id" = new."id";
end;
-- +migrate StatementEnd

create table "application_user_metadata"
(
    "id"               text      not null,
    "audit_created_at" timestamp not null,
    "audit_updated_at" timestamp not null,
    "version"          integer   not null default 0,

    "application_id"   text      not null,
    "user_id"          text      not null,

    "metadata"         text      not null,

    primary key ("id"),
    foreign key ("application_id") references "applications" ("id"),
    foreign key ("user_id") references "users" ("id"),
    unique ("application_id", "user_id")
);

-- +migrate StatementBegin
create trigger "trg_application_user_metadata_audit_updated_at"
    after update
    on "application_user_metadata"
    for each row
begin
    update "application_user_metadata" set "audit_updated_at" = strftime('%Y-%m-%d %H:%M:%f+00:00', 'now') where "id" = new."id";
end;
-- +migrate StatementEnd

create table "audit_logs"
(
    "id"                text      not null,
    "audit_created_at"  timestamp not null,
    "audit_updated_at"  timestamp not null,
    "version"           integer   not null default 0,

    "virtual_server_id" text      not null,
    "user_id"           text,

    "request_type"      text      not null,
    "request"           text      not null,
    "response"          text,

    "allowed"           boolean   not null,
    "allow_reason_type" text,
    "allow_reason"      text,

    primary key ("id"),
    foreign key ("user_id") references "users" ("id") on delete set null
);

-- +migrate StatementBegin
create trigger "trg_audit_logs_audit_updated_at"
    after update
    on "audit_logs"
    for each row
begin
    update "audit_logs" set "audit_updated_at" = strftime('%Y-%m-%d %H:%M:%f+00:00', 'now') where "id" = new."id";
end;
-- +migrate StatementEnd

create table "password_rules"
(
    "id"                text      not null,
    "audit_created_at"  timestamp not null,
    "audit_updated_at"  timestamp not null,
    "version"           integer   not null default 0,

    "virtual_server_id" text      not null,

    "type"              text      not null,
    "details"           text      not null,

    primary key ("id"),
    foreign key ("virtual_server_id") references "virtual_servers" ("id"),
    unique ("virtual_server_id", "type")
);

-- +migrate StatementBegin
create trigger "trg_password_rules_audit_updated_at"
    after update
    on "password_rules"
    for each row
begin
    update "password_rules" set "audit_updated_at" = strftime('%Y-%m-%d %H:%M:%f+00:00', 'now') where "id" = new."id";
end;
-- +migrate StatementEnd

create table "resource_servers"
(
    "id"                text      not null,
    "audit_created_at"  timestamp not null,
    "audit_updated_at"  timestamp not null,
    "version"           integer   not null default 0,

    "virtual_server_id" text      not null,
    "project_id"        text      not null,

    "slug"              text      not null,
    "name"              text      not null,
    "description"       text      not null,

    primary key ("id"),
    foreign key ("virtual_server_id") references "virtual_servers" ("id"),
    foreign key ("project_id") references "projects" ("id"),
    unique ("project_id", "slug")
);

-- +migrate StatementBegin
create trigger "trg_resource_servers_audit_updated_at"
    after update
    on "resource_servers"
    for each row
begin
    update "resource_servers" set "audit_updated_at" = strftime('%Y-%m-%d %H:%M:%f+00:00', 'now') where "id" = new."id";
end;
-- +migrate StatementEnd

create table "resource_server_scopes"
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

drop table "resource_server_scopes";
drop table "resource_servers";
drop table "password_rules";
drop table "audit_logs";
drop table "application_user_metadata";
drop table "sessions";
drop table "user_role_assignments";
drop table "group_roles";
drop table "groups";
drop table "roles";
drop table "templates";
drop table "files";
drop table "outbox_messages";
drop table "credentials";
drop table "applications";
drop table "users";
drop table "projects";
drop table "virtual_servers";
