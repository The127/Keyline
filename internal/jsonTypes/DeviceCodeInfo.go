package jsonTypes

type DeviceCodeStatus string

const (
	DeviceCodeStatusPending    DeviceCodeStatus = "pending"
	DeviceCodeStatusAuthorized DeviceCodeStatus = "authorized"
	DeviceCodeStatusDenied     DeviceCodeStatus = "denied"
)

type DeviceCodeInfo struct {
	VirtualServerName string
	ClientId          string
	GrantedScopes     []string
	Status            string
	UserId            string
	UserCode          string
}
