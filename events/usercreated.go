package events

import (
	"Keyline/config"
	"Keyline/ioc"
	"Keyline/messages"
	"Keyline/middlewares"
	"Keyline/repositories"
	"Keyline/services"
	"Keyline/templates"
	"context"
	"fmt"
	"github.com/google/uuid"
	"time"
)

type UserCreatedEvent struct {
	UserId uuid.UUID
}

func QueueEmailVerificationJobOnUserCreatedEvent(ctx context.Context, event UserCreatedEvent) error {
	scope := middlewares.GetScope(ctx)

	userRepository := ioc.GetDependency[repositories.UserRepository](scope)
	user, err := userRepository.First(ctx, repositories.NewUserFilter().Id(event.UserId))
	if err != nil {
		return fmt.Errorf("getting user: %w", err)
	}

	if user.EmailVerified() {
		return nil
	}

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServer, err := virtualServerRepository.First(ctx, repositories.NewVirtualServerFilter().Id(user.VirtualServerId()))
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

	outboxMessageRepository := ioc.GetDependency[repositories.OutboxMessageRepository](scope)
	err = outboxMessageRepository.Insert(ctx, repositories.NewOutboxMessage(message))
	if err != nil {
		return fmt.Errorf("creating email outbox message: %w", err)
	}

	return nil
}
