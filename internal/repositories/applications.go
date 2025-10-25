package repositories

import (
	"Keyline/utils"
	"context"
	"encoding/base64"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type ApplicationType string

const (
	ApplicationTypePublic       ApplicationType = "public"
	ApplicationTypeConfidential ApplicationType = "confidential"
)

type Application struct {
	ModelBase

	virtualServerId uuid.UUID

	name        string
	displayName string
	type_       ApplicationType

	hashedSecret           string
	redirectUris           []string
	postLogoutRedirectUris []string

	systemApplication bool

	claimsMappingScript   *string
	accessTokenHeaderType string
}

func NewApplication(virtualServerId uuid.UUID, name string, displayName string, type_ ApplicationType, redirectUris []string) *Application {
	return &Application{
		ModelBase:              NewModelBase(),
		virtualServerId:        virtualServerId,
		name:                   name,
		displayName:            displayName,
		type_:                  type_,
		redirectUris:           redirectUris,
		postLogoutRedirectUris: make([]string, 0),
		accessTokenHeaderType:  "at+jwt",
	}
}

func (a *Application) GetScanPointers() []any {
	return []any{
		&a.id,
		&a.auditCreatedAt,
		&a.auditUpdatedAt,
		&a.version,
		&a.virtualServerId,
		&a.name,
		&a.displayName,
		&a.type_,
		&a.hashedSecret,
		pq.Array(&a.redirectUris),
		pq.Array(&a.postLogoutRedirectUris),
		&a.systemApplication,
		&a.claimsMappingScript,
		&a.accessTokenHeaderType,
	}
}

func (a *Application) GenerateSecret() string {
	secretBytes := utils.GetSecureRandomBytes(16)
	secretBase64 := base64.RawURLEncoding.EncodeToString(secretBytes)
	a.hashedSecret = utils.CheapHash(secretBase64)
	a.TrackChange("hashed_secret", a.hashedSecret)
	return secretBase64
}

func (a *Application) VirtualServerId() uuid.UUID {
	return a.virtualServerId
}

func (a *Application) ClaimsMappingScript() *string {
	return a.claimsMappingScript
}

func (a *Application) SetClaimsMappingScript(script *string) {
	a.claimsMappingScript = script
	a.TrackChange("claims_mapping_script", script)
}

func (a *Application) SetAccessTokenHeaderType(accessTokenHeaderType string) {
	a.accessTokenHeaderType = accessTokenHeaderType
	a.TrackChange("access_token_header_type", accessTokenHeaderType)
}

func (a *Application) AccessTokenHeaderType() string {
	return a.accessTokenHeaderType
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

func (a *Application) Type() ApplicationType {
	return a.type_
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

func (a *Application) PostLogoutRedirectUris() []string {
	return a.postLogoutRedirectUris
}

func (a *Application) SetPostLogoutRedirectUris(postLogoutRedirectUris []string) {
	a.TrackChange("post_logout_redirect_uris", postLogoutRedirectUris)
	a.postLogoutRedirectUris = postLogoutRedirectUris
}

func (a *Application) SystemApplication() bool {
	return a.systemApplication
}

func (a *Application) SetSystemApplication(systemApplication bool) {
	a.TrackChange("system_application", systemApplication)
	a.systemApplication = systemApplication
}

type ApplicationFilter struct {
	PagingInfo
	OrderInfo
	name            *string
	id              *uuid.UUID
	ids             *[]uuid.UUID
	virtualServerId *uuid.UUID
	searchFilter    *SearchFilter
}

func NewApplicationFilter() ApplicationFilter {
	return ApplicationFilter{}
}

func (f ApplicationFilter) Clone() ApplicationFilter {
	return f
}

func (f ApplicationFilter) Pagination(page int, size int) ApplicationFilter {
	filter := f.Clone()
	filter.PagingInfo = PagingInfo{
		page: page,
		size: size,
	}
	return filter
}

func (f ApplicationFilter) HasPagination() bool {
	return !f.PagingInfo.IsZero()
}

func (f ApplicationFilter) GetPagingInfo() PagingInfo {
	return f.PagingInfo
}

func (f ApplicationFilter) Order(by string, direction string) ApplicationFilter {
	filter := f.Clone()
	filter.OrderInfo = OrderInfo{
		orderBy:  by,
		orderDir: direction,
	}
	return filter
}

func (f ApplicationFilter) HasOrder() bool {
	return !f.OrderInfo.IsZero()
}

func (f ApplicationFilter) GetOrderInfo() OrderInfo {
	return f.OrderInfo
}

func (f ApplicationFilter) Search(searchFilter SearchFilter) ApplicationFilter {
	filter := f.Clone()
	filter.searchFilter = &searchFilter
	return filter
}

func (f ApplicationFilter) HasSearch() bool {
	return f.searchFilter != nil
}

func (f ApplicationFilter) GetSearch() SearchFilter {
	return *f.searchFilter
}

func (f ApplicationFilter) Name(name string) ApplicationFilter {
	filter := f.Clone()
	filter.name = &name
	return filter
}

func (f ApplicationFilter) HasName() bool {
	return f.name != nil
}

func (f ApplicationFilter) GetName() string {
	return utils.ZeroIfNil(f.name)
}

func (f ApplicationFilter) Id(id uuid.UUID) ApplicationFilter {
	filter := f.Clone()
	filter.id = &id
	return filter
}

func (f ApplicationFilter) HasId() bool {
	return f.id != nil
}

func (f ApplicationFilter) GetId() uuid.UUID {
	return utils.ZeroIfNil(f.id)
}

func (f ApplicationFilter) VirtualServerId(virtualServerId uuid.UUID) ApplicationFilter {
	filter := f.Clone()
	filter.virtualServerId = &virtualServerId
	return filter
}

func (f ApplicationFilter) HasVirtualServerId() bool {
	return f.virtualServerId != nil
}

func (f ApplicationFilter) GetVirtualServerId() uuid.UUID {
	return utils.ZeroIfNil(f.virtualServerId)
}

func (f ApplicationFilter) Ids(ids []uuid.UUID) ApplicationFilter {
	fiter := f.Clone()
	fiter.ids = &ids
	return fiter
}

func (f ApplicationFilter) HasIds() bool {
	return f.ids != nil
}

func (f ApplicationFilter) GetIds() []uuid.UUID {
	if f.ids == nil {
		return []uuid.UUID{}
	}
	return *f.ids
}

//go:generate mockgen -destination=./mocks/application_repository.go -package=mocks Keyline/internal/repositories ApplicationRepository
type ApplicationRepository interface {
	Single(ctx context.Context, filter ApplicationFilter) (*Application, error)
	First(ctx context.Context, filter ApplicationFilter) (*Application, error)
	List(ctx context.Context, filter ApplicationFilter) ([]*Application, int, error)
	Insert(ctx context.Context, application *Application) error
	Update(ctx context.Context, application *Application) error
	Delete(ctx context.Context, id uuid.UUID) error
}
