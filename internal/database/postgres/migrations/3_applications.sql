-- +migrate Up

create table "applications"
(
    "id" uuid not null,
    "audit_created_at" timestamp not null,
    "audit_updated_at" timestamp not null,

    "virtual_server_id" uuid not null,
    "project_id" uuid not null,

    "display_name" text not null,
    "name" text not null,

    "system_application" boolean not null default false,

    "type" text not null default 'confidential',
    "hashed_secret" text not null,

    "redirect_uris" text[] not null,
    "post_logout_redirect_uris" text[] not null default '{}',

    "claims_mapping_script" text,
    "access_token_header_type" TEXT NOT NULL DEFAULT 'at+jwt',

    primary key ("id"),
    foreign key ("virtual_server_id") references virtual_servers ("id"),
    foreign key ("project_id") references "projects" ("id"),
    unique ("name", "virtual_server_id")
);

create trigger "trg_set_audit_updated_at"
    before update
    on "applications"
    for each row
execute function update_audit_timestamp();

-- +migrate Down
