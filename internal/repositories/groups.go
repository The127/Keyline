package repositories

import "github.com/google/uuid"

type Group struct {
	ModelBase

	virtualServerId uuid.UUID

	name        string
	description string
}

func NewGroup(virtualServerId uuid.UUID, name string, description string) *Group {
	return &Group{
		ModelBase:       NewModelBase(),
		virtualServerId: virtualServerId,
		name:            name,
		description:     description,
	}
}

func (g *Group) Name() string {
	return g.name
}

func (g *Group) SetName(name string) {
	g.TrackChange("name", name)
	g.name = name
}

func (g *Group) Description() string {
	return g.description
}

func (g *Group) SetDescription(description string) {
	g.TrackChange("description", description)
	g.description = description
}

func (g *Group) VirtualServerId() uuid.UUID {
	return g.virtualServerId
}

type GroupFilter struct {
	name *string
	id   *uuid.UUID
}

func NewGroupFilter() GroupFilter {
	return GroupFilter{}
}

func (f GroupFilter) Clone() GroupFilter {
	return f
}

func (f GroupFilter) Name(name string) GroupFilter {
	filter := f.Clone()
	filter.name = &name
	return filter
}

func (f GroupFilter) Id(id uuid.UUID) GroupFilter {
	filter := f.Clone()
	filter.id = &id
	return filter
}

//go:generate mockgen -destination=./mocks/group_repository.go -package=mocks Keyline/repositories GroupRepository
type GroupRepository interface {
}

type groupRepository struct {
}

func NewGroupRepository() GroupRepository {
	return &groupRepository{}
}
