-- +migrate Up

alter table public.users
    add column "username" text not null;

alter table "users" add constraint "idx_users_unique_name_per_vs" unique ("username", "virtual_server_id");

-- +migrate Down
