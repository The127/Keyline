package events

import (
	"Keyline/ioc"
	"Keyline/middlewares"
	"Keyline/services"
	"context"
	"fmt"
	"github.com/google/uuid"
	gomail "gopkg.in/mail.v2"
)

type UserCreatedEvent struct {
	UserId uuid.UUID
}

func QueueEmailVerificationJobOnUserCreatedEvent(ctx context.Context, event UserCreatedEvent) error {
	// TODO: queue a job instead

	// Create a new message
	message := gomail.NewMessage()

	// Set email headers
	message.SetHeader("From", "youremail@email.com")
	message.SetHeader("To", "recipient1@email.com")
	message.SetHeader("Subject", "Hello from the Mailtrap team")

	// Set email body
	message.SetBody("text/plain", "This is the Test Body")

	scope := middlewares.GetScope(ctx)
	mailService := ioc.GetDependency[services.MailService](scope)
	err := mailService.Send(message)
	if err != nil {
		return fmt.Errorf("sending email verification mail: %w", err)
	}

	return nil
}
