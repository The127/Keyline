-- +migrate Up
alter table "applications" add column "token_endpoint_auth_method" text;
update "applications" set "token_endpoint_auth_method" = 'client_secret' where "type" = 'confidential';

-- +migrate Down
alter table "applications" drop column "token_endpoint_auth_method";
