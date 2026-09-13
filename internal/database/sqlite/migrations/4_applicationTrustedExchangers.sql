-- +migrate Up
ALTER TABLE "applications" ADD COLUMN "trusted_exchangers" text not null default '[]';

-- +migrate Down
ALTER TABLE "applications" DROP COLUMN "trusted_exchangers";
