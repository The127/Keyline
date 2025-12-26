-- +migrate Up

alter table applications add column claims_mapping_script text;

-- +migrate Down
