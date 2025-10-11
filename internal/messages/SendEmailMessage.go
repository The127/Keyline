package messages

import (
	"Keyline/internal/repositories"
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type SendEmailMessage struct {
	VirtualServerId uuid.UUID `json:"virtualServerId"`
	To              string    `json:"to"`
	Subject         string    `json:"subject"`
	Body            string    `json:"body"`
}

func (m *SendEmailMessage) OutboxMessageType() repositories.OutboxMessageType {
	return repositories.SendMailOutboxMessageType
}

func (m *SendEmailMessage) Value() (driver.Value, error) {
	return json.Marshal(m)
}

func (m *SendEmailMessage) Scan(value any) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion for outbox message failed")
	}

	return json.Unmarshal(bytes, &m)
}
