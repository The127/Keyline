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
	"net/http"
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

type GetApplicationListResponseDto struct {
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

	items := utils.MapSlice(applications.Items, func(x queries.GetApplicationsResponseItem) GetApplicationListResponseDto {
		return GetApplicationListResponseDto{
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
