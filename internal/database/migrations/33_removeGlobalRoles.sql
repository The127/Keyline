-- +migrate Up

alter table user_role_assignments
    alter column application_id set not null;

-- +migrate Down
