package handlers

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/The127/Keyline/config"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/internal/services"
	"github.com/The127/Keyline/utils"

	"github.com/The127/ioc"
	"github.com/google/uuid"
)

func handleJwtBearer(w http.ResponseWriter, r *http.Request) {
	assertion := r.Form.Get("assertion")
	if assertion == "" {
		writeOAuthError(w, "invalid_request", "assertion is required")
		return
	}

	applicationName := r.Form.Get("client_id")
	if applicationName == "" {
		writeOAuthError(w, "invalid_grant", "client_id is required")
		return
	}

	scopeParameter := r.Form.Get("scope")
	if scopeParameter == "" {
		writeOAuthError(w, "invalid_grant", "scope is required")
		return
	}
	scopes := strings.Fields(scopeParameter)
	if !slices.Contains(scopes, "openid") {
		writeOAuthError(w, "invalid_grant", "scope must contain openid")
		return
	}

	if asksForResourceServerScope(scopes) {
		writeOAuthError(w, invalidScope.Error, invalidScope.ErrorDescription)
		return
	}

	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("getting virtual server name: %w", err))
		return
	}

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(virtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrNil(ctx, virtualServerFilter)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("getting virtual server: %w", err))
		return
	}
	if virtualServer == nil {
		utils.HandleHttpError(w, fmt.Errorf("virtual server not found"))
		return
	}

	var user *repositories.User
	verified, err := verifySignedAssertion(ctx, assertion, services.ServiceUserAssertionJtiTokenType, resolveServiceUserKey(dbContext, virtualServer, &user))
	if err != nil {
		writeOAuthError(w, "invalid_grant", "assertion is invalid")
		return
	}

	issuer := fmt.Sprintf("%s/oidc/%s", config.C.Server.ExternalUrl, virtualServer.Name())
	audience, err := verified.Claims.GetAudience()
	if err != nil || len(audience) != 1 {
		writeOAuthError(w, "invalid_grant", "assertion aud must be a single value")
		return
	}
	if audience[0] != issuer {
		writeOAuthError(w, "invalid_grant", "assertion aud must be this server's issuer")
		return
	}

	applicationFilter := repositories.NewApplicationFilter().
		VirtualServerId(virtualServer.Id()).
		Name(applicationName)
	application, err := dbContext.Applications().FirstOrNil(ctx, applicationFilter)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("getting application: %w", err))
		return
	}
	if application == nil {
		writeOAuthError(w, "invalid_grant", "client_id names an unknown application")
		return
	}

	issueServiceUserAccessToken(w, r, virtualServer, application, user, scopes)
}

func resolveServiceUserKey(dbContext database.Context, virtualServer *repositories.VirtualServer, user **repositories.User) assertionKeyResolver {
	return func(ctx context.Context, subject string, kid string) (uuid.UUID, any, error) {
		userFilter := repositories.NewUserFilter().
			VirtualServerId(virtualServer.Id()).
			Username(subject)
		found, err := dbContext.Users().FirstOrNil(ctx, userFilter)
		if err != nil {
			return uuid.Nil, nil, fmt.Errorf("getting user: %w", err)
		}
		if found == nil {
			return uuid.Nil, nil, fmt.Errorf("user not found")
		}
		if !found.IsServiceUser() {
			return uuid.Nil, nil, fmt.Errorf("user is not a service user")
		}

		credentialFilter := repositories.NewCredentialFilter().
			Type(repositories.CredentialTypeServiceUserKey).
			UserId(found.Id()).
			DetailKid(kid)
		credential, err := dbContext.Credentials().FirstOrNil(ctx, credentialFilter)
		if err != nil {
			return uuid.Nil, nil, fmt.Errorf("getting credential: %w", err)
		}
		if credential == nil {
			return uuid.Nil, nil, fmt.Errorf("unknown kid")
		}

		serviceUserKeyDetails, err := credential.ServiceUserKeyDetails()
		if err != nil {
			return uuid.Nil, nil, fmt.Errorf("getting service user key details: %w", err)
		}

		publicKey, err := utils.ParsePublicKeyPem(serviceUserKeyDetails.PublicKey)
		if err != nil {
			return uuid.Nil, nil, fmt.Errorf("parsing service user key: %w", err)
		}

		*user = found
		return found.Id(), publicKey, nil
	}
}
