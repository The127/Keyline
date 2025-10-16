package postgres

import (
	"Keyline/internal/repositories"
)

type groupRoleRepository struct {
}

func NewGroupRoleRepository() repositories.GroupRoleRepository {
	return &groupRoleRepository{}
}
