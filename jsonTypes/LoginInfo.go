package jsonTypes

type LoginStep string

const (
	LoginStepPassword LoginStep = "password"
)

type LoginInfo struct {
	Step                     LoginStep `json:"step"`
	ApplicationDisplayName   string    `json:"applicationDisplayName"`
	VirtualServerDisplayName string    `json:"virtualServerDisplayName"`
}

func NewLoginInfo(virtualServerDisplayName string, applicationDisplayName string) LoginInfo {
	return LoginInfo{
		Step:                     LoginStepPassword,
		VirtualServerDisplayName: virtualServerDisplayName,
		ApplicationDisplayName:   applicationDisplayName,
	}
}
