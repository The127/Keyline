-- +migrate Up

alter table projects add column system_project boolean not null default false;
update projects set system_project = true where name = 'system';

-- +migrate Down

