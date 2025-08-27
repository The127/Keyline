package handlers

import (
	"Keyline/config"
	"Keyline/ioc"
	"Keyline/jsonTypes"
	"Keyline/middlewares"
	"Keyline/repositories"
	"Keyline/services"
	"Keyline/utils"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"time"
)

type GetLoginStateResponseDto struct {
	Step                     string  `json:"step"`
	ApplicationDisplayName   string  `json:"applicationDisplayName"`
	VirtualServerDisplayName string  `json:"virtualServerDisplayName"`
	SignUpUrl                *string `json:"signUpUrl"`
}

func GetLoginState(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vars := mux.Vars(r)
	loginToken := vars["loginToken"]

	tokenService := ioc.GetDependency[services.TokenService](scope)
	redisValueString, err := tokenService.GetToken(ctx, services.LoginSessionTokenType, loginToken)
	if err != nil {
		http.Error(w, "invalid login token", http.StatusBadRequest)
		return
	}

	var loginInfo jsonTypes.LoginInfo
	err = json.Unmarshal([]byte(redisValueString), &loginInfo)
	if err != nil {
		http.Error(w, "invalid login token", http.StatusBadRequest)
		return
	}

	var signUpUrl *string = nil
	if loginInfo.RegistrationEnabled {
		signUpUrl = utils.Ptr(fmt.Sprintf(
			"%s/%s/signup",
			config.C.Frontend.ExternalUrl,
			loginInfo.VirtualServerName,
		))
	}

	response := GetLoginStateResponseDto{
		Step:                     string(loginInfo.Step),
		ApplicationDisplayName:   loginInfo.ApplicationDisplayName,
		VirtualServerDisplayName: loginInfo.VirtualServerDisplayName,
		SignUpUrl:                signUpUrl,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "invalid login token", http.StatusBadRequest)
		return
	}
}

type VerifyPasswordRequestDto struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

func VerifyPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vars := mux.Vars(r)
	loginToken := vars["loginToken"]

	tokenService := ioc.GetDependency[services.TokenService](scope)
	redisValueString, err := tokenService.GetToken(ctx, services.LoginSessionTokenType, loginToken)
	if err != nil {
		http.Error(w, "invalid login token", http.StatusBadRequest)
		return
	}

	var loginInfo jsonTypes.LoginInfo
	err = json.Unmarshal([]byte(redisValueString), &loginInfo)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	if loginInfo.Step != jsonTypes.LoginStepPasswordVerification {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var dto VerifyPasswordRequestDto
	err = json.NewDecoder(r.Body).Decode(&dto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	err = utils.ValidateDto(dto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	userRepository := ioc.GetDependency[repositories.UserRepository](scope)
	userFilter := repositories.NewUserFilter().Username(dto.Username)
	user, err := userRepository.First(ctx, userFilter)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	credentialRepository := ioc.GetDependency[repositories.CredentialRepository](scope)
	credentialFilter := repositories.NewCredentialFilter().
		UserId(user.Id()).
		Type(repositories.CredentialTypePassword)
	credential, err := credentialRepository.Single(ctx, credentialFilter)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	passwordDetails, err := credential.PasswordDetails()
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if !utils.CompareHash(dto.Password, passwordDetails.HashedPassword) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	loginInfo.UserId = user.Id()

	switch {
	case passwordDetails.Temporary:
		loginInfo.Step = jsonTypes.LoginStepTemporaryPassword
		break

	case !user.EmailVerified():
		loginInfo.Step = jsonTypes.LoginStepEmailVerification
		break

	// TODO: check if totp onboarding is needed
	// TODO: check if totp verification is needed

	default:
		// TODO: set login to success (not sure)
		break
	}

	loginInfoString, err := json.Marshal(loginInfo)
	err = tokenService.UpdateToken(ctx, services.LoginSessionTokenType, loginToken, string(loginInfoString), time.Minute*15)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func VerifyEmailToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vars := mux.Vars(r)
	loginToken := vars["loginToken"]

	tokenService := ioc.GetDependency[services.TokenService](scope)
	redisValueString, err := tokenService.GetToken(ctx, services.LoginSessionTokenType, loginToken)
	if err != nil {
		http.Error(w, "invalid login token", http.StatusBadRequest)
		return
	}

	var loginInfo jsonTypes.LoginInfo
	err = json.Unmarshal([]byte(redisValueString), &loginInfo)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	if loginInfo.Step != jsonTypes.LoginStepEmailVerification {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// TODO: implement me
}

func FinishLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vars := mux.Vars(r)
	loginToken := vars["loginToken"]

	tokenService := ioc.GetDependency[services.TokenService](scope)
	redisValueString, err := tokenService.GetToken(ctx, services.LoginSessionTokenType, loginToken)
	if err != nil {
		http.Error(w, "invalid login token", http.StatusBadRequest)
		return
	}

	var loginInfo jsonTypes.LoginInfo
	err = json.Unmarshal([]byte(redisValueString), &loginInfo)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	if loginInfo.Step != jsonTypes.LoginStepFinish {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	err = middlewares.CreateSession(w, r, loginInfo.VirtualServerName, loginInfo.UserId)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	http.Redirect(w, r, loginInfo.OriginalUrl, http.StatusFound)
}
