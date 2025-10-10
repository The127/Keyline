package jsonTypes

import "github.com/google/uuid"

type CodeInfo struct {
	VirtualServerName string
	GrantedScopes     []string
	UserId            uuid.UUID
}

func NewCodeInfo(
	virtualServerName string,
	grantedScopes []string,
	userId uuid.UUID,
) CodeInfo {
	return CodeInfo{
		VirtualServerName: virtualServerName,
		GrantedScopes:     grantedScopes,
		UserId:            userId,
	}
}
