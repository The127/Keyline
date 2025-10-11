package jsonTypes

import (
	"Keyline/internal/repositories"

	"github.com/google/uuid"
)

type LoginStep string

const (
	LoginStepPasswordVerification LoginStep = "passwordVerification"
	LoginStepTemporaryPassword    LoginStep = "temporaryPassword"
	LoginStepEmailVerification    LoginStep = "emailVerification"
	LoginStepTotpOnboarding       LoginStep = "totpOnboarding"
	LoginStepTotpVerification     LoginStep = "totpVerification"
	LoginStepFinish               LoginStep = "finish"
)

type LoginInfo struct {
	Step                     LoginStep `json:"step"`
	ApplicationDisplayName   string    `json:"applicationDisplayName"`
	VirtualServerDisplayName string    `json:"virtualServerDisplayName"`
	VirtualServerName        string    `json:"virtualServerName"`
	VirtualServerId          uuid.UUID `json:"virtualServerId"`
	RegistrationEnabled      bool      `json:"registrationEnabled"`
	UserId                   uuid.UUID `json:"userId"`
	OriginalUrl              string    `json:"originalUrl"`
}

func NewLoginInfo(virtualServer *repositories.VirtualServer, application *repositories.Application, originalUrl string) LoginInfo {
	return LoginInfo{
		Step:                     LoginStepPasswordVerification,
		VirtualServerDisplayName: virtualServer.DisplayName(),
		VirtualServerName:        virtualServer.Name(),
		VirtualServerId:          virtualServer.Id(),
		RegistrationEnabled:      virtualServer.EnableRegistration(),
		ApplicationDisplayName:   application.DisplayName(),
		OriginalUrl:              originalUrl,
	}
}
