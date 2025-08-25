package jsonTypes

import "Keyline/repositories"

type LoginStep string

const (
	LoginStepPassword LoginStep = "password"
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
		Step:                     LoginStepPassword,
		VirtualServerDisplayName: virtualServer.DisplayName(),
		VirtualServerName:        virtualServer.Name(),
		RegistrationEnabled:      virtualServer.EnableRegistration(),
		ApplicationDisplayName:   application.DisplayName(),
	}
}
