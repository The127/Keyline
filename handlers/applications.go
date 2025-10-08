package handlers

import (
	"Keyline/commands"
	"Keyline/ioc"
	"Keyline/mediator"
	"Keyline/middlewares"
	"Keyline/queries"
	"Keyline/repositories"
	"Keyline/utils"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type CreateApplicationRequestDto struct {
	Name           string   `json:"name" validate:"required,min=1,max=255"`
	DisplayName    string   `json:"displayName" validate:"required,min=1,max=255"`
	RedirectUris   []string `json:"redirectUris" validate:"required,dive,url,min=1"`
	PostLogoutUris []string `json:"postLogoutUris" validate:"dive,url"`
	Type           string   `json:"type" validate:"required,oneof=public confidential"`
}

type CreateApplicationResponseDto struct {
	Id     uuid.UUID `json:"id"`
	Secret *string   `json:"secret,omitempty"`
}

// CreateApplication creates a new application (OIDC client) in a virtual server
// @Summary Create application
// @Description Create a new OIDC application/client with redirect URIs and type
// @Tags applications
// @Accept json
// @Produce json
// @Param vsName path string true "Virtual server name"  default(keyline)
// @Param request body CreateApplicationRequestDto true "Application data"
// @Success 201 {object} CreateApplicationResponseDto
// @Failure 400
// @Failure 500
// @Router /api/virtual-servers/{vsName}/applications [post]
func CreateApplication(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vsName, err := middlewares.GetVirtualServerName(r.Context())
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	var dto CreateApplicationRequestDto
	err = json.NewDecoder(r.Body).Decode(&dto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	err = utils.ValidateDto(dto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[*mediator.Mediator](scope)

	response, err := mediator.Send[*commands.CreateApplicationResponse](ctx, m, commands.CreateApplication{
		VirtualServerName:      vsName,
		Name:                   dto.Name,
		DisplayName:            dto.DisplayName,
		Type:                   repositories.ApplicationType(dto.Type),
		RedirectUris:           dto.RedirectUris,
		PostLogoutRedirectUris: dto.PostLogoutUris,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	err = json.NewEncoder(w).Encode(CreateApplicationResponseDto{
		Id:     response.Id,
		Secret: response.Secret,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
	}
}

type GetApplicationResponseDto struct {
	Id          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"displayName"`
	Type        string    `json:"type"`

	RedirectUris           []string `json:"redirectUris"`
	PostLogoutRedirectUris []string `json:"postLogoutRedirectUris"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// GetApplication retrieves details of a specific application by ID
// @Summary Get application
// @Description Get an application by ID from a virtual server
// @Tags applications
// @Accept json
// @Produce json
// @Param vsName path string true "Virtual server name"  default(keyline)
// @Param appId path string true "Application ID (UUID)"
// @Success 200 {object} GetApplicationResponseDto
// @Failure 400
// @Failure 404 "Application not found"
// @Failure 500
// @Router /api/virtual-servers/{vsName}/applications/{appId} [get]
func GetApplication(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vsName, err := middlewares.GetVirtualServerName(r.Context())
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)
	appIdString := vars["appId"]
	appId, err := uuid.Parse(appIdString)
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[*mediator.Mediator](scope)

	application, err := mediator.Send[*queries.GetApplicationResult](ctx, m, queries.GetApplication{
		VirtualServerName: vsName,
		ApplicationId:     appId,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	if application == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(GetApplicationResponseDto{
		Id:                     application.Id,
		Name:                   application.Name,
		DisplayName:            application.DisplayName,
		Type:                   string(application.Type),
		RedirectUris:           application.RedirectUris,
		PostLogoutRedirectUris: application.PostLogoutUris,
		CreatedAt:              application.CreatedAt,
		UpdatedAt:              application.UpdatedAt,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
	}
}

type PagedApplicationsResponseDto = PagedResponseDto[ListApplicationsResponseDto]

type ListApplicationsResponseDto struct {
	Id          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"displayName"`
	Type        string    `json:"type"`
}

// ListApplications lists applications in a virtual server
// @Summary List applications
// @Description Retrieve a paginated list of applications (OIDC clients)
// @Tags applications
// @Accept json
// @Produce json
// @Param vsName path string true "Virtual server name"  default(keyline)
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Param orderBy query string false "Order by field"
// @Param orderDir query string false "Order direction (asc|desc)"
// @Param search query string false "Search term"
// @Success 200 {object} PagedApplicationsResponseDto
// @Failure 400
// @Failure 500
// @Router /api/virtual-servers/{vsName}/applications [get]
func ListApplications(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	queryOps, err := ParseQueryOps(r)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vsName, err := middlewares.GetVirtualServerName(r.Context())
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[*mediator.Mediator](scope)

	applications, err := mediator.Send[*queries.ListApplicationsResponse](ctx, m, queries.ListApplications{
		VirtualServerName: vsName,
		PagedQuery:        queryOps.ToPagedQuery(),
		OrderedQuery:      queryOps.ToOrderedQuery(),
		SearchText:        queryOps.Search,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	items := utils.MapSlice(applications.Items, func(x queries.ListApplicationsResponseItem) ListApplicationsResponseDto {
		return ListApplicationsResponseDto{
			Id:          x.Id,
			Name:        x.Name,
			DisplayName: x.DisplayName,
			Type:        string(x.Type),
		}
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(NewPagedResponseDto(
		items,
		queryOps,
		applications.TotalCount,
	))
	if err != nil {
		utils.HandleHttpError(w, err)
	}
}
