package handlers

import (
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/logging"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/utils"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/The127/ioc"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type activeSessionDto struct {
	VsName      string `json:"vsName"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
}

func ListActiveSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	sessionService := ioc.GetDependency[middlewares.SessionService](scope)
	dbContext := ioc.GetDependency[database.Context](scope)

	result := make([]activeSessionDto, 0)

	logging.Logger.Debugf("ListActiveSessions: %d cookies in request", len(r.Cookies()))

	for _, cookie := range r.Cookies() {
		if !strings.HasPrefix(cookie.Name, "keylineSession_") {
			continue
		}
		vsName := strings.TrimPrefix(cookie.Name, "keylineSession_")
		logging.Logger.Debugf("ListActiveSessions: found session cookie for vsName=%s", vsName)

		token, err := utils.DecodeSplitToken(cookie.Value)
		if err != nil {
			logging.Logger.Debugf("ListActiveSessions: failed to decode split token for vsName=%s: %v", vsName, err)
			continue
		}

		tokenId, err := uuid.Parse(token.Id())
		if err != nil {
			logging.Logger.Debugf("ListActiveSessions: failed to parse token id for vsName=%s: %v", vsName, err)
			continue
		}

		session, err := sessionService.GetSession(ctx, vsName, tokenId)
		if err != nil {
			logging.Logger.Debugf("ListActiveSessions: GetSession error for vsName=%s: %v", vsName, err)
			continue
		}
		if session == nil {
			logging.Logger.Debugf("ListActiveSessions: session not found for vsName=%s tokenId=%s", vsName, tokenId)
			continue
		}

		if !utils.CheapCompareHash(token.Secret(), session.HashedSecret()) {
			logging.Logger.Debugf("ListActiveSessions: hash mismatch for vsName=%s", vsName)
			continue
		}

		vsFilter := repositories.NewVirtualServerFilter().Name(vsName)
		vs, err := dbContext.VirtualServers().FirstOrNil(ctx, vsFilter)
		if err != nil {
			logging.Logger.Debugf("ListActiveSessions: VirtualServer lookup error for vsName=%s: %v", vsName, err)
			continue
		}
		if vs == nil {
			logging.Logger.Debugf("ListActiveSessions: VirtualServer not found for vsName=%s", vsName)
			continue
		}

		userFilter := repositories.NewUserFilter().VirtualServerId(vs.Id()).Id(session.UserId())
		user, err := dbContext.Users().FirstOrNil(ctx, userFilter)
		if err != nil {
			logging.Logger.Debugf("ListActiveSessions: User lookup error for vsName=%s userId=%s: %v", vsName, session.UserId(), err)
			continue
		}
		if user == nil {
			logging.Logger.Debugf("ListActiveSessions: User not found for vsName=%s userId=%s", vsName, session.UserId())
			continue
		}

		logging.Logger.Debugf("ListActiveSessions: resolved session for vsName=%s username=%s", vsName, user.Username())
		result = append(result, activeSessionDto{
			VsName:      vsName,
			Username:    user.Username(),
			DisplayName: user.DisplayName(),
		})
	}

	logging.Logger.Debugf("ListActiveSessions: returning %d sessions", len(result))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		logging.Logger.Warnf("ListActiveSessions: failed to encode response: %v", err)
	}
}

func DeleteActiveSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vsName := mux.Vars(r)["vsName"]
	if vsName == "" {
		http.Error(w, "missing vsName", http.StatusBadRequest)
		return
	}

	sessionService := ioc.GetDependency[middlewares.SessionService](scope)

	cookie, err := r.Cookie(middlewares.GetSessionCookieName(vsName))
	if err != nil {
		middlewares.ClearSessionCookie(w, vsName)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	token, err := utils.DecodeSplitToken(cookie.Value)
	if err != nil {
		middlewares.ClearSessionCookie(w, vsName)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	tokenId, err := uuid.Parse(token.Id())
	if err != nil {
		middlewares.ClearSessionCookie(w, vsName)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	session, err := sessionService.GetSession(ctx, vsName, tokenId)
	if err == nil && session != nil && utils.CheapCompareHash(token.Secret(), session.HashedSecret()) {
		if err := sessionService.DeleteSession(ctx, vsName, tokenId); err != nil {
			logging.Logger.Warnf("DeleteActiveSession: failed to delete session for vsName=%s: %v", vsName, err)
		}
	}

	middlewares.ClearSessionCookie(w, vsName)
	w.WriteHeader(http.StatusNoContent)
}
