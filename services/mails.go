package services

import (
	"Keyline/config"
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
	dialer := gomail.NewDialer(
		config.C.Mail.Host,
		config.C.Mail.Port,
		config.C.Mail.Username,
		config.C.Mail.Password,
	)

	if err := dialer.DialAndSend(m...); err != nil {
		return fmt.Errorf("sending mail: %w", err)
	}

	return nil
}
