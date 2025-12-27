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

	"github.com/google/uuid"
)

type UserCreatedEvent struct {
	UserId uuid.UUID
}

func QueueEmailVerificationJobOnUserCreatedEvent(ctx context.Context, event UserCreatedEvent) error {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[db.Context](scope)

	user, err := dbContext.Users().FirstOrNil(ctx, repositories.NewUserFilter().Id(event.UserId))
	if err != nil {
		return fmt.Errorf("getting user: %w", err)
	}

	if user.EmailVerified() {
		return nil
	}

	virtualServer, err := dbContext.VirtualServers().FirstOrNil(ctx, repositories.NewVirtualServerFilter().Id(user.VirtualServerId()))
	if err != nil {
		return fmt.Errorf("getting virtual server: %w", err)
	}

	tokenService := ioc.GetDependency[services.TokenService](scope)
	token, err := tokenService.GenerateAndStoreToken(ctx, services.EmailVerificationTokenType, user.Id().String(), time.Minute*15)
	if err != nil {
		return fmt.Errorf("storing email verification token: %w", err)
	}

	templateService := ioc.GetDependency[services.TemplateService](scope)
	mailBody, err := templateService.Template(
		ctx,
		user.VirtualServerId(),
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
		VirtualServerId: user.VirtualServerId(),
		To:              user.PrimaryEmail(),
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
