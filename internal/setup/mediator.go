package setup

import (
	"Keyline/internal/behaviours"
	"Keyline/internal/commands"
	"Keyline/internal/events"
	"Keyline/internal/queries"
	"Keyline/ioc"
	"Keyline/mediator"
)

func Mediator(dc *ioc.DependencyCollection) {
	m := mediator.NewMediator()

	mediator.RegisterHandler(m, queries.HandleAnyVirtualServerExists)
	mediator.RegisterHandler(m, queries.HandleGetVirtualServerPublicInfo)
	mediator.RegisterHandler(m, queries.HandleGetVirtualServerQuery)
	mediator.RegisterHandler(m, commands.HandleCreateVirtualServer)

	mediator.RegisterHandler(m, queries.HandleListPasswordRules)
	mediator.RegisterHandler(m, commands.HandleCreatePasswordRule)
	mediator.RegisterHandler(m, commands.HandleUpdatePasswordRule)

	mediator.RegisterHandler(m, queries.HandleListTemplates)
	mediator.RegisterHandler(m, queries.HandleGetTemplate)

	mediator.RegisterHandler(m, commands.HandleRegisterUser)
	mediator.RegisterHandler(m, commands.HandleCreateUser)
	mediator.RegisterHandler(m, commands.HandleVerifyEmail)
	mediator.RegisterHandler(m, commands.HandleResetPassword)
	mediator.RegisterHandler(m, queries.HandleGetUserQuery)
	mediator.RegisterHandler(m, commands.HandlePatchUser)
	mediator.RegisterHandler(m, queries.HandleListUsers)
	mediator.RegisterHandler(m, commands.HandleCreateServiceUser)
	mediator.RegisterHandler(m, commands.HandleAssociateServiceUserPublicKey)
	mediator.RegisterHandler(m, queries.HandleGetUserMetadata)
	mediator.RegisterHandler(m, commands.HandleUpdateUserMetadata)
	mediator.RegisterHandler(m, commands.HandleUpdateUserAppMetadata)
	mediator.RegisterHandler(m, commands.HandlePatchUserMetadata)
	mediator.RegisterHandler(m, commands.HandlePatchUserAppMetadata)

	mediator.RegisterHandler(m, commands.HandleCreateApplication)
	mediator.RegisterHandler(m, queries.HandleListApplications)
	mediator.RegisterHandler(m, queries.HandleGetApplication)
	mediator.RegisterHandler(m, commands.HandlePatchApplication)
	mediator.RegisterHandler(m, commands.HandleDeleteApplication)

	mediator.RegisterHandler(m, commands.HandleCreateProject)
	mediator.RegisterHandler(m, queries.HandleListProjects)

	mediator.RegisterHandler(m, queries.HandleListRoles)
	mediator.RegisterHandler(m, queries.HandleGetRole)
	mediator.RegisterHandler(m, commands.HandleCreateRole)
	mediator.RegisterHandler(m, commands.HandleAssignRoleToUser)
	mediator.RegisterHandler(m, queries.HandleListUsersInRole)

	mediator.RegisterHandler(m, queries.HandleListGroups)

	mediator.RegisterHandler(m, queries.HandleListAuditEntries)

	mediator.RegisterEventHandler(m, events.QueueEmailVerificationJobOnUserCreatedEvent)

	mediator.RegisterBehaviour(m, behaviours.PolicyBehaviour)

	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) mediator.Mediator {
		return m
	})
}
