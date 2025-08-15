package events

import (
	"Keyline/ioc"
	"Keyline/middlewares"
	"Keyline/repositories"
	"Keyline/services"
	"Keyline/templates"
	"context"
	"fmt"
	"github.com/google/uuid"
	gomail "gopkg.in/mail.v2"
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

	templateService := ioc.GetDependency[services.TemplateService](scope)
	templateService.Template(
		ctx,
		user.VirtualServerId(),
		repositories.EmailVerificationMailTemplate,
		templates.EmailVerificationTemplateData{
			VerificationLink: "",
		},
	)

	// TODO: queue a job instead

	// Create a new message
	message := gomail.NewMessage()

	// Set email headers
	message.SetHeader("From", "youremail@email.com")
	message.SetHeader("To", "recipient1@email.com")
	message.SetHeader("Subject", "Hello from the Mailtrap team")

	// Set email body
	message.SetBody("text/plain", "This is the Test Body")

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
