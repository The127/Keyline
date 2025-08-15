package events

import (
	"Keyline/config"
	"Keyline/ioc"
	"Keyline/middlewares"
	"Keyline/repositories"
	"Keyline/services"
	"Keyline/templates"
	"context"
	"fmt"
	"github.com/google/uuid"
	gomail "gopkg.in/mail.v2"
	"time"
)

type UserCreatedEvent struct {
	UserId uuid.UUID
}

func QueueEmailVerificationJobOnUserCreatedEvent(ctx context.Context, event UserCreatedEvent) error {
	scope := middlewares.GetScope(ctx)

	userRepository := ioc.GetDependency[*repositories.UserRepository](scope)
	user, err := userRepository.First(ctx, repositories.NewUserFilter().Id(event.UserId))
	if err != nil {
		return fmt.Errorf("getting user: %w", err)
	}

	if user.EmailVerified() {
		return nil
	}

	virtualServerRepository := ioc.GetDependency[*repositories.VirtualServerRepository](scope)
	virtualServer, err := virtualServerRepository.First(ctx, repositories.NewVirtualServerFilter().Id(user.VirtualServerId()))
	if err != nil {
		return fmt.Errorf("getting virtual server: %w", err)
	}

	tokenService := ioc.GetDependency[services.TokenService](scope)
	token, err := tokenService.StoreToken(ctx, services.GetEmailVerificationTokenKey(user.Id()), time.Minute*15)
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

	// TODO: queue a job instead

	// Create a new message
	message := gomail.NewMessage()

	// Set email headers
	message.SetHeader("From", "youremail@email.com")
	message.SetHeader("To", user.PrimaryEmail())
	message.SetHeader("Subject", "Email verification")

	// Set email body
	message.SetBody("text/plain", mailBody)

	mailService := ioc.GetDependency[services.MailService](scope)
	err = mailService.Send(message)
	if err != nil {
		return fmt.Errorf("sending email verification mail: %w", err)
	}

	// dummy outbox test
	outboxRepository := ioc.GetDependency[*repositories.OutboxMessageRepository](scope)
	err = outboxRepository.Insert(ctx, repositories.NewOutboxMessage(&repositories.DummyOutboxMessageDetails{
		Foo: "some test value",
	}))
	if err != nil {
		return fmt.Errorf("testing outbox repo: %w", err)
	}

	return nil
}
