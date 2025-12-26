package repositories

import (
	"Keyline/internal/change"
	"Keyline/utils"
	"context"
	"encoding/base64"
	"slices"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type ApplicationType string

const (
	ApplicationTypePublic       ApplicationType = "public"
	ApplicationTypeConfidential ApplicationType = "confidential"
)

type ApplicationChange int

const (
	ApplicationChangeHashedSecret ApplicationChange = iota
	ApplicationChangeClaimsMappingScript
	ApplicationChangeAccessTokenHeaderType
	ApplicationChangeDisplayName
	ApplicationChangeRedirectUris
	ApplicationChangePostLogoutRedirectUris
	ApplicationChangeSystemApplication
)

type Application struct {
	BaseModel
	change.List[ApplicationChange]

	virtualServerId uuid.UUID
	projectId       uuid.UUID

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

func NewApplication(virtualServerId uuid.UUID, projectId uuid.UUID, name string, displayName string, type_ ApplicationType, redirectUris []string) *Application {
	return &Application{
		BaseModel:              NewModelBase(),
		List:                   change.NewChanges[ApplicationChange](),
		virtualServerId:        virtualServerId,
		projectId:              projectId,
		name:                   name,
		displayName:            displayName,
		type_:                  type_,
		redirectUris:           redirectUris,
		postLogoutRedirectUris: []string{},
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
		&a.projectId,
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

	a.SetHashedSecret(utils.CheapHash(secretBase64))
	return secretBase64
}

func (a *Application) VirtualServerId() uuid.UUID {
	return a.virtualServerId
}

func (a *Application) ProjectId() uuid.UUID {
	return a.projectId
}

func (a *Application) ClaimsMappingScript() *string {
	return a.claimsMappingScript
}

func (a *Application) SetClaimsMappingScript(script *string) {
	if a.claimsMappingScript == script {
		return
	}

	a.claimsMappingScript = script
	a.TrackChange(ApplicationChangeClaimsMappingScript)
}

func (a *Application) SetAccessTokenHeaderType(accessTokenHeaderType string) {
	if a.accessTokenHeaderType == accessTokenHeaderType {
		return
	}

	a.accessTokenHeaderType = accessTokenHeaderType
	a.TrackChange(ApplicationChangeAccessTokenHeaderType)
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
	if a.displayName == displayName {
		return
	}

	a.displayName = displayName
	a.TrackChange(ApplicationChangeDisplayName)
}

func (a *Application) Type() ApplicationType {
	return a.type_
}

func (a *Application) HashedSecret() string {
	return a.hashedSecret
}

func (a *Application) SetHashedSecret(hashedSecret string) {
	if a.hashedSecret == hashedSecret {
		return
	}

	a.hashedSecret = hashedSecret
	a.TrackChange(ApplicationChangeHashedSecret)
}

func (a *Application) RedirectUris() []string {
	return a.redirectUris
}

func (a *Application) SetRedirectUris(redirectUris []string) {
	if slices.Equal(a.redirectUris, redirectUris) {
		return
	}

	a.redirectUris = redirectUris
	a.TrackChange(ApplicationChangeRedirectUris)
}

func (a *Application) PostLogoutRedirectUris() []string {
	return a.postLogoutRedirectUris
}

func (a *Application) SetPostLogoutRedirectUris(postLogoutRedirectUris []string) {
	if slices.Equal(a.postLogoutRedirectUris, postLogoutRedirectUris) {
		return
	}

	a.postLogoutRedirectUris = postLogoutRedirectUris
	a.TrackChange(ApplicationChangePostLogoutRedirectUris)
}

func (a *Application) SystemApplication() bool {
	return a.systemApplication
}

func (a *Application) SetSystemApplication(systemApplication bool) {
	if a.systemApplication == systemApplication {
		return
	}

	a.systemApplication = systemApplication
	a.TrackChange(ApplicationChangeSystemApplication)
}

type ApplicationFilter struct {
	PagingInfo
	OrderInfo
	name            *string
	id              *uuid.UUID
	ids             *[]uuid.UUID
	virtualServerId *uuid.UUID
	projectId       *uuid.UUID
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

func (f ApplicationFilter) ProjectId(projectId uuid.UUID) ApplicationFilter {
	filter := f.Clone()
	filter.projectId = &projectId
	return filter
}

func (f ApplicationFilter) HasProjectId() bool {
	return f.projectId != nil
}

func (f ApplicationFilter) GetProjectId() uuid.UUID {
	return utils.ZeroIfNil(f.projectId)
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
	Insert(application *Application)
	Update(application *Application)
	Delete(id uuid.UUID)
}
