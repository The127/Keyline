-- +migrate Up

ALTER table applications
    ADD COLUMN access_token_header_type TEXT NOT NULL DEFAULT 'at+jwt';

-- +migrate Down

ALTER table applications
    DROP COLUMN access_token_header_type;
