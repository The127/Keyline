-- +migrate Up

alter table user_role_assignments drop column application_id;

-- +migrate Down

