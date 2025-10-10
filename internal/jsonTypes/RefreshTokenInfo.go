package jsonTypes

import "github.com/google/uuid"

type RefreshTokenInfo struct {
	VirtualServerName string
	UserId            uuid.UUID
	GrantedScopes     []string
	ClientId          string
}

func NewRefreshTokenInfo(
	virtualServerName string,
	clientId string,
	userId uuid.UUID,
	grantedScopes []string,
) RefreshTokenInfo {
	return RefreshTokenInfo{
		VirtualServerName: virtualServerName,
		ClientId:          clientId,
		UserId:            userId,
		GrantedScopes:     grantedScopes,
	}
}
