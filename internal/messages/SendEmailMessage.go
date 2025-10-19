package messages

import (
	"Keyline/internal/repositories"
	"encoding/json"

	"github.com/google/uuid"
)

type SendEmailMessage struct {
	VirtualServerId uuid.UUID `json:"virtualServerId"`
	DisplayName     string    `json:"displayName"`
	To              string    `json:"to"`
	Subject         string    `json:"subject"`
	Body            string    `json:"body"`
}

func (m *SendEmailMessage) OutboxMessageType() repositories.OutboxMessageType {
	return repositories.SendMailOutboxMessageType
}

func (m *SendEmailMessage) Serialize() ([]byte, error) {
	return json.Marshal(m)
}
