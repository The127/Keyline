-- +migrate Up

alter table applications add column system_application boolean not null default false;
update applications set system_application = true where name = 'admin-ui';

-- +migrate Down
