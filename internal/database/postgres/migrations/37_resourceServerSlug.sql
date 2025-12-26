-- +migrate Up

alter table resource_servers add column slug text not null;
create unique index idx_unique_project_resource_server_slug on resource_servers (project_id, slug);

-- +migrate Down

