-- +migrate Up

create table files (
   "id" uuid not null,
   "audit_created_at" timestamp not null,
   "audit_updated_at" timestamp not null,

    "name" text not null,
    "mime_type" text not null,
    "content" bytea not null,

    primary key ("id")
);

create trigger "trg_set_audit_updated_at"
    before update
    on "files"
    for each row
    execute function update_audit_timestamp();

-- +migrate Down
