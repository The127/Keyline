package jsonTypes

import "Keyline/repositories"

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
	RegistrationEnabled      bool      `json:"registrationEnabled"`
}

func NewLoginInfo(virtualServer *repositories.VirtualServer, application *repositories.Application) LoginInfo {
	return LoginInfo{
		Step:                     LoginStepPasswordVerification,
		VirtualServerDisplayName: virtualServer.DisplayName(),
		VirtualServerName:        virtualServer.Name(),
		RegistrationEnabled:      virtualServer.EnableRegistration(),
		ApplicationDisplayName:   application.DisplayName(),
	}
}
