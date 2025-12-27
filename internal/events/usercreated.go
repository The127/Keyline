package events

import (
	"Keyline/internal/config"
	db "Keyline/internal/database"
	"Keyline/internal/messages"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/internal/services"
	"Keyline/templates"
	"context"
	"fmt"
	"time"

	"github.com/The127/ioc"
)

type UserCreatedEvent struct {
	User *repositories.User
}

func QueueEmailVerificationJobOnUserCreatedEvent(ctx context.Context, event UserCreatedEvent) error {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[db.Context](scope)

	if event.User.EmailVerified() {
		return nil
	}

	virtualServer, err := dbContext.VirtualServers().FirstOrNil(ctx, repositories.NewVirtualServerFilter().Id(event.User.VirtualServerId()))
	if err != nil {
		return fmt.Errorf("getting virtual server: %w", err)
	}

	tokenService := ioc.GetDependency[services.TokenService](scope)
	token, err := tokenService.GenerateAndStoreToken(ctx, services.EmailVerificationTokenType, event.User.Id().String(), time.Minute*15)
	if err != nil {
		return fmt.Errorf("storing email verification token: %w", err)
	}

	templateService := ioc.GetDependency[services.TemplateService](scope)
	mailBody, err := templateService.Template(
		ctx,
		event.User.VirtualServerId(),
		repositories.EmailVerificationMailTemplate,
		templates.EmailVerificationTemplateData{
			VerificationLink: fmt.Sprintf(
				"%s/api/virtual-servers/%s/users/verify-email?token=%s",
				config.C.Server.ExternalUrl,
				virtualServer.Name(),
				token,
			),
		},
	)
	if err != nil {
		return fmt.Errorf("templating email verification mail: %w", err)
	}

	message := &messages.SendEmailMessage{
		VirtualServerId: event.User.VirtualServerId(),
		To:              event.User.PrimaryEmail(),
		Subject:         "Email verification",
		Body:            mailBody,
	}

	outboxMessage, err := repositories.NewOutboxMessage(message)
	if err != nil {
		return fmt.Errorf("creating email outbox message: %w", err)
	}

	dbContext.OutboxMessages().Insert(outboxMessage)
	return nil
}
