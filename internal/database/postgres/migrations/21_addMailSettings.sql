-- +migrate Up

alter table virtual_servers add column mail_host text;
alter table virtual_servers add column mail_port integer;
alter table virtual_servers add column mail_username text;
alter table virtual_servers add column mail_encrypted_password text;

-- +migrate Down
