-- +migrate Up
ALTER TABLE resource_server_scopes DROP CONSTRAINT resource_server_scopes_virtual_server_id_scope_key;
ALTER TABLE resource_server_scopes ADD CONSTRAINT resource_server_scopes_project_id_scope_key UNIQUE (project_id, scope);

-- +migrate Down
ALTER TABLE resource_server_scopes DROP CONSTRAINT resource_server_scopes_project_id_scope_key;
ALTER TABLE resource_server_scopes ADD CONSTRAINT resource_server_scopes_virtual_server_id_scope_key UNIQUE (virtual_server_id, scope);
