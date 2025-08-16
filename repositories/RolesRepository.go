package repositories

import "github.com/google/uuid"

type Role struct {
	ModelBase

	applicationId *uuid.UUID

	name        string
	description string

	requireMfa         bool
	maxTokenAgeSeconds int
}

func NewRole(applicationId *uuid.UUID, name string, description string) *Role {
	return &Role{
		ModelBase:     NewModelBase(),
		applicationId: applicationId,
		name:          name,
		description:   description,
	}
}

func (r *Role) Name() string {
	return r.name
}

func (r *Role) SetName(name string) {
	r.TrackChange("name", name)
	r.name = name
}

func (r *Role) Description() string {
	return r.description
}

func (r *Role) SetDescription(description string) {
	r.TrackChange("description", description)
	r.description = description
}

func (r *Role) ApplicationId() *uuid.UUID {
	return r.applicationId
}

func (r *Role) RequireMfa() bool {
	return r.requireMfa
}

func (r *Role) SetRequireMfa(requireMfa bool) {
	r.TrackChange("require_mfa", requireMfa)
	r.requireMfa = requireMfa
}

func (r *Role) MaxTokenAgeSeconds() int {
	return r.maxTokenAgeSeconds
}

func (r *Role) SetMaxTokenAgeSeconds(maxTokenAgeSeconds int) {
	r.TrackChange("max_token_age_seconds", maxTokenAgeSeconds)
	r.maxTokenAgeSeconds = maxTokenAgeSeconds
}

type RoleFilter struct {
	name *string
	id   *uuid.UUID
}

func NewRoleFilter() RoleFilter {
	return RoleFilter{}
}

func (f RoleFilter) Clone() RoleFilter {
	return f
}

func (f RoleFilter) Name(name string) RoleFilter {
	filter := f.Clone()
	filter.name = &name
	return filter
}

func (f RoleFilter) Id(id uuid.UUID) RoleFilter {
	filter := f.Clone()
	filter.id = &id
	return filter
}

type RoleRepository struct {
}
