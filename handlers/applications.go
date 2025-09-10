package handlers

import (
	"Keyline/commands"
	"Keyline/ioc"
	"Keyline/mediator"
	"Keyline/middlewares"
	"Keyline/queries"
	"Keyline/utils"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"net/http"
	"time"
)

type CreateApplicationRequestDto struct {
	Name         string   `json:"name" validate:"required,min=1,max=255"`
	DisplayName  string   `json:"displayName" validate:"required,min=1,max=255"`
	RedirectUris []string `json:"redirectUris" validate:"required,dive,url,min=1"`
}

type CreateApplicationResponseDto struct {
	Id     uuid.UUID
	Secret string
}

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
		VirtualServerName: vsName,
		Name:              dto.Name,
		DisplayName:       dto.DisplayName,
		RedirectUris:      dto.RedirectUris,
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
	Id           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	DisplayName  string    `json:"displayName"`
	RedirectUris []string  `json:"redirectUris"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

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
		Id:           application.Id,
		Name:         application.Name,
		DisplayName:  application.DisplayName,
		RedirectUris: application.RedirectUris,
		CreatedAt:    application.CreatedAt,
		UpdatedAt:    application.UpdatedAt,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
	}
}

type ListApplicationsResponseDto struct {
	Id          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"displayName"`
}

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

	applications, err := mediator.Send[*queries.GetApplicationsResponse](ctx, m, queries.GetApplications{
		VirtualServerName: vsName,
		PagedQuery:        queryOps.ToPagedQuery(),
		OrderedQuery:      queryOps.ToOrderedQuery(),
		SearchText:        queryOps.Search,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	items := utils.MapSlice(applications.Items, func(x queries.GetApplicationsResponseItem) ListApplicationsResponseDto {
		return ListApplicationsResponseDto{
			Id:          x.Id,
			Name:        x.Name,
			DisplayName: x.DisplayName,
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
