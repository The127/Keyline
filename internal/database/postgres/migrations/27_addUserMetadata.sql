-- +migrate Up

alter table users add column metadata jsonb default '{}';

create table application_user_metadata(
    "id" uuid not null default gen_random_uuid(),
    "audit_created_at" timestamp not null default now(),
    "audit_updated_at" timestamp not null default now(),
    "version" bigint not null default 1,

    "application_id" uuid not null,
    "user_id" uuid not null,

    "metadata" jsonb not null,

    primary key ("id")
);

alter table "application_user_metadata"
    add constraint "fk_app_user_metadata_app_id"
    foreign key ("application_id")
    references "applications" ("id");

alter table "application_user_metadata"
    add constraint "fk_app_user_metadata_user_id"
    foreign key ("user_id")
    references "users" ("id");

create unique index idx_app_user_metadata_unique_user_per_app on application_user_metadata (application_id, user_id);

create trigger "trg_set_audit_updated_at"
    before update
    on "application_user_metadata"
    for each row
    execute function update_audit_timestamp();

-- +migrate Down
