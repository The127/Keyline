package setup

import (
	"Keyline/internal/behaviours"
	"Keyline/internal/commands"
	"Keyline/internal/events"
	"Keyline/internal/queries"

	"github.com/The127/ioc"
	"github.com/The127/mediatr"
)

func Mediator(dc *ioc.DependencyCollection) {
	m := mediatr.NewMediator()

	mediatr.RegisterHandler(m, queries.HandleAnyVirtualServerExists)
	mediatr.RegisterHandler(m, queries.HandleGetVirtualServerPublicInfo)
	mediatr.RegisterHandler(m, queries.HandleGetVirtualServerQuery)
	mediatr.RegisterHandler(m, commands.HandleCreateVirtualServer)
	mediatr.RegisterHandler(m, commands.HandlePatchVirtualServer)

	mediatr.RegisterHandler(m, queries.HandleListPasswordRules)
	mediatr.RegisterHandler(m, commands.HandleCreatePasswordRule)
	mediatr.RegisterHandler(m, commands.HandleUpdatePasswordRule)

	mediatr.RegisterHandler(m, queries.HandleListTemplates)
	mediatr.RegisterHandler(m, queries.HandleGetTemplate)

	mediatr.RegisterHandler(m, commands.HandleRegisterUser)
	mediatr.RegisterHandler(m, commands.HandleCreateUser)
	mediatr.RegisterHandler(m, commands.HandleVerifyEmail)
	mediatr.RegisterHandler(m, commands.HandleSetPassword)
	mediatr.RegisterHandler(m, queries.HandleGetUserQuery)
	mediatr.RegisterHandler(m, commands.HandlePatchUser)
	mediatr.RegisterHandler(m, queries.HandleListUsers)
	mediatr.RegisterHandler(m, commands.HandleCreateServiceUser)
	mediatr.RegisterHandler(m, commands.HandleAssociateServiceUserPublicKey)
	mediatr.RegisterHandler(m, commands.HandleRemoveServiceUserPublicKey)
	mediatr.RegisterHandler(m, queries.HandleGetUserMetadata)
	mediatr.RegisterHandler(m, commands.HandleUpdateUserMetadata)
	mediatr.RegisterHandler(m, commands.HandleUpdateUserAppMetadata)
	mediatr.RegisterHandler(m, commands.HandlePatchUserMetadata)
	mediatr.RegisterHandler(m, commands.HandlePatchUserAppMetadata)
	mediatr.RegisterHandler(m, queries.HandleListPasskeys)

	mediatr.RegisterHandler(m, commands.HandleCreateResourceServer)
	mediatr.RegisterHandler(m, commands.HandlePatchResourceServer)
	mediatr.RegisterHandler(m, queries.HandleListResourceServers)
	mediatr.RegisterHandler(m, queries.HandleGetResourceServer)

	mediatr.RegisterHandler(m, commands.HandleCreateResourceServerScope)
	mediatr.RegisterHandler(m, queries.HandleListResourceServerScopes)
	mediatr.RegisterHandler(m, queries.HandleGetResourceServerScope)

	mediatr.RegisterHandler(m, commands.HandleCreateApplication)
	mediatr.RegisterHandler(m, queries.HandleListApplications)
	mediatr.RegisterHandler(m, queries.HandleGetApplication)
	mediatr.RegisterHandler(m, commands.HandlePatchApplication)
	mediatr.RegisterHandler(m, commands.HandleDeleteApplication)

	mediatr.RegisterHandler(m, commands.HandleCreateProject)
	mediatr.RegisterHandler(m, queries.HandleListProjects)
	mediatr.RegisterHandler(m, queries.HandleGetProject)
	mediatr.RegisterHandler(m, commands.HandlePatchProject)

	mediatr.RegisterHandler(m, queries.HandleListRoles)
	mediatr.RegisterHandler(m, queries.HandleGetRole)
	mediatr.RegisterHandler(m, commands.HandleCreateRole)
	mediatr.RegisterHandler(m, commands.HandlePatchRole)
	mediatr.RegisterHandler(m, commands.HandleAssignRoleToUser)
	mediatr.RegisterHandler(m, queries.HandleListUsersInRole)

	mediatr.RegisterHandler(m, queries.HandleListGroups)

	mediatr.RegisterHandler(m, queries.HandleListAuditEntries)

	mediatr.RegisterEventHandler(m, events.QueueEmailVerificationJobOnUserCreatedEvent)

	mediatr.RegisterBehaviour(m, behaviours.PolicyBehaviour)

	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) mediatr.Mediator {
		return m
	})
}
