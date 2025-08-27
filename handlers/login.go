package handlers

import (
	"Keyline/commands"
	"Keyline/config"
	"Keyline/ioc"
	"Keyline/jsonTypes"
	"Keyline/mediator"
	"Keyline/messages"
	"Keyline/middlewares"
	"Keyline/repositories"
	"Keyline/services"
	"Keyline/templates"
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

	userRepository := ioc.GetDependency[repositories.UserRepository](scope)
	userFilter := repositories.NewUserFilter().Id(loginInfo.UserId)
	user, err := userRepository.Single(ctx, userFilter)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	if !user.EmailVerified() {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	loginInfo.Step = jsonTypes.LoginStepFinish
	loginInfoString, err := json.Marshal(loginInfo)
	err = tokenService.UpdateToken(ctx, services.LoginSessionTokenType, loginToken, string(loginInfoString), time.Minute*15)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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

	// TODO: delete login in redis

	http.Redirect(w, r, loginInfo.OriginalUrl, http.StatusFound)
}

type ResetTemporaryPasswordRequestDto struct {
	NewPassword string `json:"newPassword" validate:"required"`
}

func ResetTemporaryPassword(w http.ResponseWriter, r *http.Request) {
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

	if loginInfo.Step != jsonTypes.LoginStepTemporaryPassword {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var dto ResetTemporaryPasswordRequestDto
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

	m := ioc.GetDependency[*mediator.Mediator](scope)
	_, err = mediator.Send[*commands.ResetPasswordResponse](ctx, m, commands.ResetPassword{
		UserId:      loginInfo.UserId,
		NewPassword: dto.NewPassword,
		Temporary:   false,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	// TODO: Figure out next step to be done, prepare it if needed

	loginInfo.Step = jsonTypes.LoginStepFinish

	loginInfoString, err := json.Marshal(loginInfo)
	err = tokenService.UpdateToken(ctx, services.LoginSessionTokenType, loginToken, string(loginInfoString), time.Minute*15)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func ResendEmailVerification(w http.ResponseWriter, r *http.Request) {
	// TODO: add "cooldown" for sending emails

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

	// retrigger email verification sending
	token, err := tokenService.GenerateAndStoreToken(ctx, services.EmailVerificationTokenType, loginInfo.UserId.String(), time.Minute*15)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("storing email verification token: %w", err))
		return
	}

	templateService := ioc.GetDependency[services.TemplateService](scope)
	mailBody, err := templateService.Template(
		ctx,
		loginInfo.VirtualServerId,
		repositories.EmailVerificationMailTemplate,
		templates.EmailVerificationTemplateData{
			VerificationLink: fmt.Sprintf(
				"%s/api/virtual-servers/%s/users/verify-email?token=%s",
				config.C.Server.ExternalUrl,
				loginInfo.VirtualServerName,
				token,
			),
		},
	)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("templating email verification mail: %w", err))
		return
	}

	userRepository := ioc.GetDependency[repositories.UserRepository](scope)
	userFilter := repositories.NewUserFilter().Id(loginInfo.UserId)
	user, err := userRepository.Single(ctx, userFilter)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("getting user: %w", err))
		return
	}

	message := &messages.SendEmailMessage{
		VirtualServerId: loginInfo.VirtualServerId,
		To:              user.PrimaryEmail(),
		Subject:         "Email verification",
		Body:            mailBody,
	}

	outboxMessageRepository := ioc.GetDependency[repositories.OutboxMessageRepository](scope)
	err = outboxMessageRepository.Insert(ctx, repositories.NewOutboxMessage(message))
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("creating email outbox message: %w", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
