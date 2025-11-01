package jsonTypes

import "github.com/google/uuid"

type PasskeyCreateChallenge struct {
	Id        uuid.UUID `json:"id"`
	UserId    uuid.UUID `json:"userId"`
	Challenge string    `json:"challenge"`
}

type PasskeyLoginChallenge struct {
	Id                uuid.UUID `json:"id"`
	Challenge         string    `json:"challenge"`
	LoginSessionToken string    `json:"loginSessionToken"`
}
