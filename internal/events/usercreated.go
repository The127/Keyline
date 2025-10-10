package events

import (
	"Keyline/internal/config"
	"Keyline/internal/messages"
	"Keyline/internal/middlewares"
	repositories2 "Keyline/internal/repositories"
	services2 "Keyline/internal/services"
	"Keyline/ioc"
	"Keyline/templates"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type UserCreatedEvent struct {
	UserId uuid.UUID
}

func QueueEmailVerificationJobOnUserCreatedEvent(ctx context.Context, event UserCreatedEvent) error {
	scope := middlewares.GetScope(ctx)

	userRepository := ioc.GetDependency[repositories2.UserRepository](scope)
	user, err := userRepository.First(ctx, repositories2.NewUserFilter().Id(event.UserId))
	if err != nil {
		return fmt.Errorf("getting user: %w", err)
	}

	if user.EmailVerified() {
		return nil
	}

	virtualServerRepository := ioc.GetDependency[repositories2.VirtualServerRepository](scope)
	virtualServer, err := virtualServerRepository.First(ctx, repositories2.NewVirtualServerFilter().Id(user.VirtualServerId()))
	if err != nil {
		return fmt.Errorf("getting virtual server: %w", err)
	}

	tokenService := ioc.GetDependency[services2.TokenService](scope)
	token, err := tokenService.GenerateAndStoreToken(ctx, services2.EmailVerificationTokenType, user.Id().String(), time.Minute*15)
	if err != nil {
		return fmt.Errorf("storing email verification token: %w", err)
	}

	templateService := ioc.GetDependency[services2.TemplateService](scope)
	mailBody, err := templateService.Template(
		ctx,
		user.VirtualServerId(),
		repositories2.EmailVerificationMailTemplate,
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

	outboxMessageRepository := ioc.GetDependency[repositories2.OutboxMessageRepository](scope)
	err = outboxMessageRepository.Insert(ctx, repositories2.NewOutboxMessage(message))
	if err != nil {
		return fmt.Errorf("creating email outbox message: %w", err)
	}

	return nil
}
