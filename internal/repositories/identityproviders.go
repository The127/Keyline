package repositories

import (
	"context"
	"fmt"
	"github.com/The127/Keyline/internal/change"
	"github.com/The127/Keyline/utils"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

type IdentityProviderChange int

type IdentityProviderClaimMapping struct {
	Subject       string `json:"subject"`
	Email         string `json:"email"`
	EmailVerified string `json:"emailVerified"`
	Name          string `json:"name"`
	Username      string `json:"username"`
}

var DefaultIdentityProviderClaimMapping = IdentityProviderClaimMapping{
	Subject:       "sub",
	Email:         "email",
	EmailVerified: "email_verified",
	Name:          "name",
	Username:      "preferred_username",
}

func (m IdentityProviderClaimMapping) FilledFrom(defaults IdentityProviderClaimMapping) IdentityProviderClaimMapping {
	filled := m
	if filled.Subject == "" {
		filled.Subject = defaults.Subject
	}

	if filled.Email == "" {
		filled.Email = defaults.Email
	}

	if filled.EmailVerified == "" {
		filled.EmailVerified = defaults.EmailVerified
	}

	if filled.Name == "" {
		filled.Name = defaults.Name
	}

	if filled.Username == "" {
		filled.Username = defaults.Username
	}

	return filled
}

type IdentityProviderSettings struct {
	Issuer                string
	AuthorizationEndpoint string
	TokenEndpoint         string
	UserinfoEndpoint      string
	Scopes                []string
	ClientId              string
	ClientSecret          string `json:"-"`
	ClaimMapping          IdentityProviderClaimMapping
}

func (s IdentityProviderSettings) FilledFrom(defaults IdentityProviderSettings) IdentityProviderSettings {
	filled := s
	filled.ClaimMapping = filled.ClaimMapping.FilledFrom(defaults.ClaimMapping)
	if filled.Issuer == "" {
		filled.Issuer = defaults.Issuer
	}
	if filled.AuthorizationEndpoint == "" {
		filled.AuthorizationEndpoint = defaults.AuthorizationEndpoint
	}
	if filled.TokenEndpoint == "" {
		filled.TokenEndpoint = defaults.TokenEndpoint
	}
	if filled.UserinfoEndpoint == "" {
		filled.UserinfoEndpoint = defaults.UserinfoEndpoint
	}
	if filled.Scopes == nil {
		filled.Scopes = append([]string{}, defaults.Scopes...)
	}
	if filled.ClientId == "" {
		filled.ClientId = defaults.ClientId
	}
	if filled.ClientSecret == "" {
		filled.ClientSecret = defaults.ClientSecret
	}
	return filled
}

func (s IdentityProviderSettings) Validate() error {
	if s.AuthorizationEndpoint == "" {
		return fmt.Errorf("identity provider needs an authorization endpoint: %w", utils.ErrHttpBadRequest)
	}
	if s.TokenEndpoint == "" {
		return fmt.Errorf("identity provider needs a token endpoint: %w", utils.ErrHttpBadRequest)
	}
	if s.UserinfoEndpoint == "" {
		return fmt.Errorf("identity provider needs a userinfo endpoint: %w", utils.ErrHttpBadRequest)
	}
	for _, endpoint := range []string{s.Issuer, s.AuthorizationEndpoint, s.TokenEndpoint, s.UserinfoEndpoint} {
		if endpoint == "" {
			continue
		}
		parsed, err := url.Parse(endpoint)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return fmt.Errorf("identity provider url %s must be an absolute http or https url: %w", endpoint, utils.ErrHttpBadRequest)
		}
	}
	for _, scope := range s.Scopes {
		if scope == "" {
			return fmt.Errorf("identity provider scopes must not be empty: %w", utils.ErrHttpBadRequest)
		}
	}
	if s.ClientId == "" {
		return fmt.Errorf("identity provider needs a client id: %w", utils.ErrHttpBadRequest)
	}
	if s.ClientSecret == "" {
		return fmt.Errorf("identity provider needs a client secret: %w", utils.ErrHttpBadRequest)
	}
	return nil
}

type IdentityProvider struct {
	BaseModel
	change.List[IdentityProviderChange]

	virtualServerId uuid.UUID
	name            string
	displayName     string
	preset          string
	settings        IdentityProviderSettings
}

func NewIdentityProvider(virtualServerId uuid.UUID, name string, displayName string, preset string, settings IdentityProviderSettings) *IdentityProvider {
	return &IdentityProvider{
		BaseModel:       NewBaseModel(),
		List:            change.NewChanges[IdentityProviderChange](),
		virtualServerId: virtualServerId,
		name:            name,
		displayName:     displayName,
		preset:          preset,
		settings:        settings,
	}
}

const identityProviderNameForbiddenRunes = "/?#%"

func NewIdentityProviderFromSettings(virtualServerId uuid.UUID, name string, displayName string, preset string, settings IdentityProviderSettings) (*IdentityProvider, error) {
	if name == "" {
		return nil, fmt.Errorf("identity provider needs a name: %w", utils.ErrHttpBadRequest)
	}
	if strings.ContainsAny(name, identityProviderNameForbiddenRunes) {
		return nil, fmt.Errorf("identity provider name %s must not contain any of %q: %w", name, identityProviderNameForbiddenRunes, utils.ErrHttpBadRequest)
	}
	if displayName == "" {
		return nil, fmt.Errorf("identity provider %s needs a display name: %w", name, utils.ErrHttpBadRequest)
	}

	if preset != "" {
		defaults, ok := IdentityProviderPreset(preset)
		if !ok {
			return nil, fmt.Errorf("unknown identity provider preset %s: %w", preset, utils.ErrHttpBadRequest)
		}
		settings = settings.FilledFrom(defaults)
	}

	settings.Scopes = utils.EmptyIfNil(settings.Scopes)
	settings.ClaimMapping = settings.ClaimMapping.FilledFrom(DefaultIdentityProviderClaimMapping)

	err := settings.Validate()
	if err != nil {
		return nil, err
	}

	return NewIdentityProvider(virtualServerId, name, displayName, preset, settings), nil
}

func NewIdentityProviderFromDB(base BaseModel, virtualServerId uuid.UUID, name string, displayName string, preset string, settings IdentityProviderSettings) *IdentityProvider {
	settings.ClaimMapping = settings.ClaimMapping.FilledFrom(DefaultIdentityProviderClaimMapping)
	return &IdentityProvider{
		BaseModel:       base,
		List:            change.NewChanges[IdentityProviderChange](),
		virtualServerId: virtualServerId,
		name:            name,
		displayName:     displayName,
		preset:          preset,
		settings:        settings,
	}
}

func (p *IdentityProvider) VirtualServerId() uuid.UUID {
	return p.virtualServerId
}

func (p *IdentityProvider) Name() string {
	return p.name
}

func (p *IdentityProvider) DisplayName() string {
	return p.displayName
}

func (p *IdentityProvider) Preset() string {
	return p.preset
}

func (p *IdentityProvider) Settings() IdentityProviderSettings {
	return p.settings
}

type IdentityProviderFilter struct {
	id              *uuid.UUID
	virtualServerId *uuid.UUID
	name            *string
}

func NewIdentityProviderFilter() *IdentityProviderFilter {
	return &IdentityProviderFilter{}
}

func (f *IdentityProviderFilter) Clone() *IdentityProviderFilter {
	clone := *f
	return &clone
}

func (f *IdentityProviderFilter) Id(id uuid.UUID) *IdentityProviderFilter {
	filter := f.Clone()
	filter.id = &id
	return filter
}

func (f *IdentityProviderFilter) HasId() bool {
	return f.id != nil
}

func (f *IdentityProviderFilter) GetId() uuid.UUID {
	return utils.ZeroIfNil(f.id)
}

func (f *IdentityProviderFilter) VirtualServerId(virtualServerId uuid.UUID) *IdentityProviderFilter {
	filter := f.Clone()
	filter.virtualServerId = &virtualServerId
	return filter
}

func (f *IdentityProviderFilter) HasVirtualServerId() bool {
	return f.virtualServerId != nil
}

func (f *IdentityProviderFilter) GetVirtualServerId() uuid.UUID {
	return utils.ZeroIfNil(f.virtualServerId)
}

func (f *IdentityProviderFilter) Name(name string) *IdentityProviderFilter {
	filter := f.Clone()
	filter.name = &name
	return filter
}

func (f *IdentityProviderFilter) HasName() bool {
	return f.name != nil
}

func (f *IdentityProviderFilter) GetName() string {
	return utils.ZeroIfNil(f.name)
}

//go:generate mockgen -destination=./mocks/identity_provider_repository.go -package=mocks Keyline/internal/repositories IdentityProviderRepository
type IdentityProviderRepository interface {
	FirstOrErr(ctx context.Context, filter *IdentityProviderFilter) (*IdentityProvider, error)
	FirstOrNil(ctx context.Context, filter *IdentityProviderFilter) (*IdentityProvider, error)
	List(ctx context.Context, filter *IdentityProviderFilter) ([]*IdentityProvider, error)
	Insert(identityProvider *IdentityProvider)
}
