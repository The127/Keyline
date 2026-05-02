package jsonTypes

import (
	"time"

	"github.com/google/uuid"
)

type CodeInfo struct {
	VirtualServerName string
	ApplicationId     uuid.UUID
	GrantedScopes     []string
	UserId            uuid.UUID
	Nonce             string
	AuthenticatedAt   time.Time
	RedirectUri       string
	// CodeChallenge and CodeChallengeMethod are populated when the client
	// initiated the flow with PKCE. If CodeChallenge is empty, no PKCE
	// verification is required at /token.
	CodeChallenge       string
	CodeChallengeMethod string
}

func NewCodeInfo(
	virtualServerName string,
	applicationId uuid.UUID,
	grantedScopes []string,
	userId uuid.UUID,
	nonce string,
	authenticatedAt time.Time,
	redirectUri string,
	codeChallenge string,
	codeChallengeMethod string,
) CodeInfo {
	return CodeInfo{
		VirtualServerName:   virtualServerName,
		ApplicationId:       applicationId,
		GrantedScopes:       grantedScopes,
		UserId:              userId,
		Nonce:               nonce,
		AuthenticatedAt:     authenticatedAt,
		RedirectUri:         redirectUri,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
	}
}
