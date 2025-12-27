package outbox

import (
	"Keyline/internal/logging"
	"Keyline/internal/messages"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/internal/services"
	"context"
	"encoding/json"
	"fmt"

	"github.com/The127/ioc"

	gomail "gopkg.in/mail.v2"
)

//go:generate mockgen -destination=./mocks/mock_deliveryService.go -package=mocks . DeliveryService
type DeliveryService interface {
	Deliver(ctx context.Context, message *repositories.OutboxMessage) error
}

//go:generate mockgen -destination=./mocks/mock_messageBroker.go -package=mocks . MessageBroker
type MessageBroker interface {
	Distribute(ctx context.Context, message *repositories.OutboxMessage) error
}

type messageBroker struct {
}

func NewMessageBroker() MessageBroker {
	return &messageBroker{}
}

func (m *messageBroker) Distribute(ctx context.Context, message *repositories.OutboxMessage) error {
	logging.Logger.Debug("Distributing message", "message_type", message.Type(), "message_id", message.Id())
	scope := middlewares.GetScope(ctx)

	switch message.Type() {
	case repositories.SendMailOutboxMessageType:
		var sendEmailDetails messages.SendEmailMessage
		err := json.Unmarshal(message.Details(), &sendEmailDetails)
		if err != nil {
			return fmt.Errorf("failed to unmarshal send email message details: %w", err)
		}

		mailService := ioc.GetDependency[services.MailService](scope)

		mail := gomail.NewMessage()
		mail.SetAddressHeader("To", sendEmailDetails.To, sendEmailDetails.DisplayName)
		mail.SetAddressHeader("From", "no-reply@keyline.home.arpa", "Keyline")
		mail.SetHeader("Subject", sendEmailDetails.Subject)
		mail.SetBody("text/plain", sendEmailDetails.Body)

		err = mailService.Send(mail)
		if err != nil {
			return fmt.Errorf("failed to send email: %w", err)
		}

		return nil

	default:
		return fmt.Errorf("unsupported message type: %s", message.Type())
	}
}
