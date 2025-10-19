package jsonTypes

import (
	"time"

	"github.com/google/uuid"
)

type CodeInfo struct {
	VirtualServerName string
	GrantedScopes     []string
	UserId            uuid.UUID
	Nonce             string
	AuthenticatedAt   time.Time
}

func NewCodeInfo(virtualServerName string, grantedScopes []string, userId uuid.UUID, nonce string, authenticatedAt time.Time) CodeInfo {
	return CodeInfo{
		VirtualServerName: virtualServerName,
		GrantedScopes:     grantedScopes,
		UserId:            userId,
		Nonce:             nonce,
		AuthenticatedAt:   authenticatedAt,
	}
}
