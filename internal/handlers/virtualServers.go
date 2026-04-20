package handlers

import (
	"encoding/json"
	"github.com/The127/Keyline/api"
	"github.com/The127/Keyline/config"
	"github.com/The127/Keyline/internal/commands"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/queries"
	"github.com/The127/Keyline/utils"
	"net/http"

	"github.com/The127/ioc"
	"github.com/The127/mediatr"
)

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

	var dto api.CreateVirtualServerRequestDto
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

	primaryAlgorithm := config.SigningAlgorithmEdDSA
	if dto.PrimarySigningAlgorithm != nil {
		primaryAlgorithm = config.SigningAlgorithm(*dto.PrimarySigningAlgorithm)
	}

	additionalAlgorithms := make([]config.SigningAlgorithm, len(dto.AdditionalSigningAlgorithms))
	for i, a := range dto.AdditionalSigningAlgorithms {
		additionalAlgorithms[i] = config.SigningAlgorithm(a)
	}

	_, err = mediatr.Send[*commands.CreateVirtualServerResponse](ctx, m, commands.CreateVirtualServer{
		Name:                        dto.Name,
		DisplayName:                 dto.DisplayName,
		EnableRegistration:          dto.EnableRegistration,
		PrimarySigningAlgorithm:     primaryAlgorithm,
		AdditionalSigningAlgorithms: additionalAlgorithms,
		Require2fa:                  dto.Require2fa,
		Admin: utils.MapPtr(dto.Admin, func(admin api.CreateVirtualServerRequestDtoAdminDto) commands.CreateVirtualServerAdmin {
			return commands.CreateVirtualServerAdmin{
				Username:     admin.Username,
				DisplayName:  admin.DisplayName,
				PrimaryEmail: admin.PrimaryEmail,
				PasswordHash: admin.PasswordHash,
				Roles:        admin.Roles,
			}
		}),
		ServiceUsers: utils.MapSlice(dto.ServiceUsers, func(x api.CreateVirtualServerRequestDtoServiceUserDto) commands.CreateVirtualServerServiceUser {
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
		Projects: utils.MapSlice(dto.Projects, func(project api.CreateVirtualServerRequestDtoProjectDto) commands.CreateVirtualServerProject {
			return commands.CreateVirtualServerProject{
				Slug:        project.Slug,
				Name:        project.Name,
				Description: project.Description,
				Applications: utils.MapSlice(project.Applications, func(app api.CreateVirtualServerRequestDtoProjectDtoApplicationDto) commands.CreateVirtualServerProjectApplication {
					return commands.CreateVirtualServerProjectApplication{
						Name:           app.Name,
						DisplayName:    app.DisplayName,
						Type:           app.Type,
						HashedSecret:   app.HashedSecret,
						RedirectUris:   app.RedirectUris,
						PostLogoutUris: app.PostLogoutUris,
					}
				}),
				Roles: utils.MapSlice(project.Roles, func(role api.CreateVirtualServerRequestDtoProjectDtoRoleDto) commands.CreateVirtualServerProjectRole {
					return commands.CreateVirtualServerProjectRole{
						Name:        role.Name,
						Description: role.Description,
					}
				}),
				ResourceServers: utils.MapSlice(project.ResourceServers, func(rs api.CreateVirtualServerRequestDtoProjectDtoResourceServerDto) commands.CreateVirtualServerProjectResourceServer {
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

	additionalAlgorithms := make([]string, len(response.AdditionalSigningAlgorithms))
	for i, a := range response.AdditionalSigningAlgorithms {
		additionalAlgorithms[i] = string(a)
	}

	err = json.NewEncoder(w).Encode(api.GetVirtualServerResponseDto{
		Id:                          response.Id,
		Name:                        response.Name,
		DisplayName:                 response.DisplayName,
		RegistrationEnabled:         response.RegistrationEnabled,
		Require2fa:                  response.Require2fa,
		RequireEmailVerification:    response.RequireEmailVerification,
		PrimarySigningAlgorithm:     string(response.PrimarySigningAlgorithm),
		AdditionalSigningAlgorithms: additionalAlgorithms,
		CreatedAt:                   response.CreatedAt,
		UpdatedAt:                   response.UpdatedAt,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
	}
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

	err = json.NewEncoder(w).Encode(api.GetVirtualServerListResponseDto{
		Name:                response.Name,
		DisplayName:         response.DisplayName,
		RegistrationEnabled: response.RegistrationEnabled,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
	}
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

	var dto api.PatchVirtualServerRequestDto
	err = json.NewDecoder(r.Body).Decode(&dto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	var additionalAlgorithms *[]config.SigningAlgorithm
	if dto.AdditionalSigningAlgorithms != nil {
		converted := make([]config.SigningAlgorithm, len(*dto.AdditionalSigningAlgorithms))
		for i, a := range *dto.AdditionalSigningAlgorithms {
			converted[i] = config.SigningAlgorithm(a)
		}
		additionalAlgorithms = &converted
	}

	m := ioc.GetDependency[mediatr.Mediator](scope)
	command := commands.PatchVirtualServer{
		VirtualServerName:           vsName,
		DisplayName:                 utils.TrimSpace(dto.DisplayName),
		EnableRegistration:          dto.EnableRegistration,
		Require2fa:                  dto.Require2fa,
		RequireEmailVerification:    dto.RequireEmailVerification,
		PrimarySigningAlgorithm:     (*config.SigningAlgorithm)(dto.PrimarySigningAlgorithm),
		AdditionalSigningAlgorithms: additionalAlgorithms,
	}
	_, err = mediatr.Send[*commands.PatchVirtualServerResponse](ctx, m, command)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
