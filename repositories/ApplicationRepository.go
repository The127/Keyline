package repositories

import "github.com/google/uuid"

type Application struct {
	ModelBase

	virtualServerId uuid.UUID

	name        string
	displayName string

	hashedSecret string
	redirectUris []string
}

func NewApplication(
	virtualServerId uuid.UUID,
	name string,
	displayName string,
	hashedSecret string,
	redirectUris []string,
) *Application {
	return &Application{
		ModelBase:       NewModelBase(),
		virtualServerId: virtualServerId,
		name:            name,
		displayName:     displayName,
		hashedSecret:    hashedSecret,
		redirectUris:    redirectUris,
	}
}

func (a *Application) VirtualServerId() uuid.UUID {
	return a.virtualServerId
}

func (a *Application) Name() string {
	return a.name
}

func (a *Application) DisplayName() string {
	return a.displayName
}

func (a *Application) SetDisplayName(displayName string) {
	a.TrackChange("display_name", displayName)
	a.displayName = displayName
}

func (a *Application) HashedSecret() string {
	return a.hashedSecret
}

func (a *Application) SetHashedSecret(hashedSecret string) {
	a.TrackChange("hashed_secret", hashedSecret)
	a.hashedSecret = hashedSecret
}

func (a *Application) RedirectUris() []string {
	return a.redirectUris
}

func (a *Application) SetRedirectUris(redirectUris []string) {
	a.TrackChange("redirect_uris", redirectUris)
	a.redirectUris = redirectUris
}

type ApplicationFilter struct {
	name            *string
	id              *uuid.UUID
	virtualServerId *uuid.UUID
}

func NewApplicationFilter() ApplicationFilter {
	return ApplicationFilter{}
}

func (f ApplicationFilter) Clone() ApplicationFilter {
	return f
}

func (f ApplicationFilter) Name(name string) ApplicationFilter {
	filter := f.Clone()
	filter.name = &name
	return filter
}

func (f ApplicationFilter) Id(id uuid.UUID) ApplicationFilter {
	filter := f.Clone()
	filter.id = &id
	return filter
}

func (f ApplicationFilter) VirtualServerId(virtualServerId uuid.UUID) ApplicationFilter {
	filter := f.Clone()
	filter.virtualServerId = &virtualServerId
	return filter
}

type ApplicationRepository struct{}
