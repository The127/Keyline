-- +migrate Up

ALTER table applications
    ADD COLUMN access_token_header_type TEXT;
UPDATE applications
SET access_token_header_type = 'at+jwt'
WHERE true;

-- +migrate Down

ALTER table applications
    DROP COLUMN access_token_header_type;
