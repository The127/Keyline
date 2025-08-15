package messages

import (
	"Keyline/repositories"
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

func (s SendEmailMessage) OutboxMessageType() repositories.OutboxMessageType {
	return repositories.SendMailOutboxMessageType
}

func (d *SendEmailMessage) Value() (driver.Value, error) {
	return json.Marshal(d)
}

func (d *SendEmailMessage) Scan(value any) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion for outbox message failed")
	}

	return json.Unmarshal(bytes, &d)
}
