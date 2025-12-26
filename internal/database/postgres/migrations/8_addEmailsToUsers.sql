-- +migrate Up

alter table users
    add column primary_email text not null;

alter table users
    add column email_verified bool not null;

-- +migrate Down
