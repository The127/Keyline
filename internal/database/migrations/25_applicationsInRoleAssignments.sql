-- +migrate Up

alter table user_role_assignments add column application_id uuid default null references applications(id);

-- +migrate Down
