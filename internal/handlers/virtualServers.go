package handlers

import (
	"Keyline/internal/commands"
	"Keyline/internal/config"
	"Keyline/internal/middlewares"
	"Keyline/internal/queries"
	"Keyline/utils"
	"encoding/json"
	"net/http"
	"time"

	"github.com/The127/ioc"
	"github.com/The127/mediatr"

	"github.com/google/uuid"
)

type CreateVirtualServerRequestDtoAdminDto struct {
	Username     string   `json:"username" validate:"required,min=1,max=255"`
	DisplayName  string   `json:"displayName" validate:"required,min=1,max=255"`
	PrimaryEmail string   `json:"primaryEmail" validate:"required,email"`
	PasswordHash string   `json:"passwordHash" validate:"required"`
	Roles        []string `json:"roles"`
}

type CreateVirtualServerRequestDtoServiceUserDto struct {
	Username  string   `json:"username" validate:"required,min=1,max=255"`
	Roles     []string `json:"roles"`
	PublicKey struct {
		Pem string `json:"pem" validate:"required"`
		Kid string `json:"kid" validate:"required"`
	} `json:"publicKey" validate:"required"`
}

type CreateVirtualServerRequestDtoProjectDtoRoleDto struct {
	Name        string `json:"name" validate:"required,min=1,max=255"`
	Description string `json:"description"`
}

type CreateVirtualServerRequestDtoProjectDtoApplicationDto struct {
	Name           string   `json:"name" validate:"required,min=1,max=255"`
	DisplayName    string   `json:"displayName" validate:"required,min=1,max=255"`
	Type           string   `json:"type" validate:"required,oneof=public confidential"`
	HashedSecret   *string  `json:"hashedSecret"`
	RedirectUris   []string `json:"redirectUris" validate:"required,dive,url,min=1"`
	PostLogoutUris []string `json:"postLogoutUris" validate:"dive,url"`
}

type CreateVirtualServerRequestDtoProjectDtoResourceServerDto struct {
	Slug        string `json:"slug" validate:"required,min=1,max=255"`
	Name        string `json:"name" validate:"required,min=1,max=255"`
	Description string `json:"description"`
}

type CreateVirtualServerRequestDtoProjectDto struct {
	Slug        string `json:"slug" validate:"required,min=1,max=255"`
	Name        string `json:"name" validate:"required,min=1,max=255"`
	Description string `json:"description"`

	Roles           []CreateVirtualServerRequestDtoProjectDtoRoleDto           `json:"roles"`
	Applications    []CreateVirtualServerRequestDtoProjectDtoApplicationDto    `json:"applications"`
	ResourceServers []CreateVirtualServerRequestDtoProjectDtoResourceServerDto `json:"resourceServers"`
}

type CreateVirtualServerRequestDto struct {
	Name               string  `json:"name" validate:"required,min=1,max=255,alphanum"`
	DisplayName        string  `json:"displayName" validate:"required,min=1,max=255"`
	EnableRegistration bool    `json:"enableRegistration"`
	SigningAlgorithm   *string `json:"signingAlgorithm" validate:"oneof=RS256 EdDSA"`
	Require2fa         bool    `json:"require2fa"`

	Admin        *CreateVirtualServerRequestDtoAdminDto        `json:"admin"`
	ServiceUsers []CreateVirtualServerRequestDtoServiceUserDto `json:"serviceUsers"`
	Projects     []CreateVirtualServerRequestDtoProjectDto     `json:"projects"`
}

// CreateVirtualServer creates a new virtual server.
// @Summary      Create virtual server
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        body  body  handlers.CreateVirtualServerRequestDto  true  "Virtual server"
// @Success      204   {string}  string  "No Content"
// @Failure      400   {string}  string
// @Router       /api/virtual-servers [post]
func CreateVirtualServer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	var dto CreateVirtualServerRequestDto
	err := json.NewDecoder(r.Body).Decode(&dto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	err = utils.ValidateDto(dto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}
	m := ioc.GetDependency[mediatr.Mediator](scope)

	signingAlgorithm := config.SigningAlgorithmEdDSA
	if dto.SigningAlgorithm != nil {
		signingAlgorithm = config.SigningAlgorithm(*dto.SigningAlgorithm)
	}

	_, err = mediatr.Send[*commands.CreateVirtualServerResponse](ctx, m, commands.CreateVirtualServer{
		Name:               dto.Name,
		DisplayName:        dto.DisplayName,
		EnableRegistration: dto.EnableRegistration,
		SigningAlgorithm:   signingAlgorithm,
		Require2fa:         dto.Require2fa,
		Admin: utils.MapPtr(dto.Admin, func(admin CreateVirtualServerRequestDtoAdminDto) commands.CreateVirtualServerAdmin {
			return commands.CreateVirtualServerAdmin{
				Username:     admin.Username,
				DisplayName:  admin.DisplayName,
				PrimaryEmail: admin.PrimaryEmail,
				PasswordHash: admin.PasswordHash,
				Roles:        admin.Roles,
			}
		}),
		ServiceUsers: utils.MapSlice(dto.ServiceUsers, func(x CreateVirtualServerRequestDtoServiceUserDto) commands.CreateVirtualServerServiceUser {
			return commands.CreateVirtualServerServiceUser{
				Username: x.Username,
				Roles:    x.Roles,
				PublicKey: struct {
					Pem string
					Kid string
				}{
					Pem: x.PublicKey.Pem,
					Kid: x.PublicKey.Kid,
				},
			}
		}),
		Projects: utils.MapSlice(dto.Projects, func(project CreateVirtualServerRequestDtoProjectDto) commands.CreateVirtualServerProject {
			return commands.CreateVirtualServerProject{
				Slug:        project.Slug,
				Name:        project.Name,
				Description: project.Description,
				Applications: utils.MapSlice(project.Applications, func(app CreateVirtualServerRequestDtoProjectDtoApplicationDto) commands.CreateVirtualServerProjectApplication {
					return commands.CreateVirtualServerProjectApplication{
						Name:           app.Name,
						DisplayName:    app.DisplayName,
						Type:           app.Type,
						HashedSecret:   app.HashedSecret,
						RedirectUris:   app.RedirectUris,
						PostLogoutUris: app.PostLogoutUris,
					}
				}),
				Roles: utils.MapSlice(project.Roles, func(role CreateVirtualServerRequestDtoProjectDtoRoleDto) commands.CreateVirtualServerProjectRole {
					return commands.CreateVirtualServerProjectRole{
						Name:        role.Name,
						Description: role.Description,
					}
				}),
				ResourceServers: utils.MapSlice(project.ResourceServers, func(rs CreateVirtualServerRequestDtoProjectDtoResourceServerDto) commands.CreateVirtualServerProjectResourceServer {
					return commands.CreateVirtualServerProjectResourceServer{
						Slug:        rs.Slug,
						Name:        rs.Name,
						Description: rs.Description,
					}
				}),
			}
		}),
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type GetVirtualServerResponseDto struct {
	Id                       uuid.UUID `json:"id"`
	Name                     string    `json:"name"`
	DisplayName              string    `json:"displayName"`
	RegistrationEnabled      bool      `json:"registrationEnabled"`
	Require2fa               bool      `json:"require2fa"`
	RequireEmailVerification bool      `json:"requireEmailVerification"`
	SigningAlgorithm         string    `json:"signingAlgorithm"`
	CreatedAt                time.Time `json:"createdAt"`
	UpdatedAt                time.Time `json:"updatedAt"`
}

// GetVirtualServer returns details of a virtual server.
// @Summary      Get virtual server
// @Tags         Admin
// @Produce      json
// @Param        virtualServerName  path  string  true  "Virtual server name"  default(keyline)
// @Success      200  {object}  handlers.GetVirtualServerResponseDto
// @Failure      404  {string}  string
// @Router       /api/virtual-servers/{virtualServerName} [get]
func GetVirtualServer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vsName, err := middlewares.GetVirtualServerName(r.Context())
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	m := ioc.GetDependency[mediatr.Mediator](scope)
	response, err := mediatr.Send[*queries.GetVirtualServerResponse](ctx, m, queries.GetVirtualServerQuery{
		VirtualServerName: vsName,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(GetVirtualServerResponseDto{
		Id:                       response.Id,
		Name:                     response.Name,
		DisplayName:              response.DisplayName,
		RegistrationEnabled:      response.RegistrationEnabled,
		Require2fa:               response.Require2fa,
		RequireEmailVerification: response.RequireEmailVerification,
		SigningAlgorithm:         string(response.SigningAlgorithm),
		CreatedAt:                response.CreatedAt,
		UpdatedAt:                response.UpdatedAt,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
	}
}

type GetVirtualServerListResponseDto struct {
	Name                string `json:"name"`
	DisplayName         string `json:"displayName"`
	RegistrationEnabled bool   `json:"registrationEnabled"`
}

// GetVirtualServerPublicInfo returns public info of a virtual server.
// @Summary      Get virtual server public info
// @Tags         Admin
// @Produce      json
// @Param        virtualServerName  path  string  true  "Virtual server name"  default(keyline)
// @Success      200  {object}  handlers.GetVirtualServerListResponseDto
// @Failure      404  {string}  string
// @Router       /api/virtual-servers/{virtualServerName}/public-info [get]
func GetVirtualServerPublicInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vsName, err := middlewares.GetVirtualServerName(r.Context())
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	m := ioc.GetDependency[mediatr.Mediator](scope)

	response, err := mediatr.Send[*queries.GetVirtualServerPublicInfoResponse](ctx, m, queries.GetVirtualServerPublicInfo{
		VirtualServerName: vsName,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(GetVirtualServerListResponseDto{
		Name:                response.Name,
		DisplayName:         response.DisplayName,
		RegistrationEnabled: response.RegistrationEnabled,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
	}
}

type PatchVirtualServerRequestDto struct {
	DisplayName *string `json:"displayName"`

	EnableRegistration       *bool `json:"enableRegistration"`
	Require2fa               *bool `json:"require2fa"`
	RequireEmailVerification *bool `json:"requireEmailVerification"`
}

// PatchVirtualServer patches a virtual server.
// @Summary      Patch virtual server
// @Tags         Admin
// @Accept       json
// @Produce      plain
// @Param        virtualServerName  path  string  true  "Virtual server name"  default(keyline)
// @Param        body  body  PatchVirtualServerRequestDto  true  "Patch document"
// @Success      204  {string} string "No Content"
// @Failure      404  {string}  string
// @Router       /api/virtual-servers/{virtualServerName} [patch]
func PatchVirtualServer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	var dto PatchVirtualServerRequestDto
	err = json.NewDecoder(r.Body).Decode(&dto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	m := ioc.GetDependency[mediatr.Mediator](scope)
	command := commands.PatchVirtualServer{
		VirtualServerName: vsName,
		DisplayName:       utils.TrimSpace(dto.DisplayName),
	}
	_, err = mediatr.Send[*commands.PatchVirtualServerResponse](ctx, m, command)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
