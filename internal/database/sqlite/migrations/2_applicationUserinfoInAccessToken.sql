-- +migrate Up
ALTER TABLE "applications" ADD COLUMN "userinfo_in_access_token" boolean not null default false;

-- +migrate Down
ALTER TABLE "applications" DROP COLUMN "userinfo_in_access_token";
