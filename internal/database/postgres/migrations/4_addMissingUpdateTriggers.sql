-- +migrate Up

create trigger "trg_set_audit_updated_at"
    before update
    on "virtual_servers"
    for each row
    execute function update_audit_timestamp();

create trigger "trg_set_audit_updated_at"
    before update
    on "users"
    for each row
    execute function update_audit_timestamp();

create trigger "trg_set_audit_updated_at"
    before update
    on "applications"
    for each row
    execute function update_audit_timestamp();


-- +migrate Down
