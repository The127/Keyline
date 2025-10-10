package services

import (
	"Keyline/internal/config"
	"fmt"

	gomail "gopkg.in/mail.v2"
)

type MailService interface {
	Send(m ...*gomail.Message) error
}

type mailService struct {
}

func NewMailService() MailService {
	return &mailService{}
}

func (s *mailService) Send(m ...*gomail.Message) error {
	//TODO: per tenant credentials from database
	dialer := gomail.NewDialer(
		config.C.InitialVirtualServer.Mail.Host,
		config.C.InitialVirtualServer.Mail.Port,
		config.C.InitialVirtualServer.Mail.Username,
		config.C.InitialVirtualServer.Mail.Password,
	)

	if err := dialer.DialAndSend(m...); err != nil {
		return fmt.Errorf("sending mail: %w", err)
	}

	return nil
}
