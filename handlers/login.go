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
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type GetLoginStateResponseDto struct {
	// Step is one of: password_verification | temporary_password | email_verification | finish
	Step                     string `json:"step"`
	ApplicationDisplayName   string `json:"applicationDisplayName"`
	VirtualServerDisplayName string `json:"virtualServerDisplayName"`
	VirtualServerName        string `json:"virtualServerName"`
	SignupEnabled            bool   `json:"signupEnabled"`
}

// GetLoginState returns the current step of the login session.
// @Summary      Get login state
// @Tags         Logins
// @Produce      json
// @Param        loginToken  path   string true  "Login session token"
// @Success      200         {object}  handlers.GetLoginStateResponseDto
// @Failure      400         {string}  string "Bad Request"
// @Failure      401         {string}  string "Unknown/invalid token"
// @Router       /logins/{loginToken} [get]
func GetLoginState(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vars := mux.Vars(r)
	loginToken := vars["loginToken"]

	tokenService := ioc.GetDependency[services.TokenService](scope)
	redisValueString, err := tokenService.GetToken(ctx, services.LoginSessionTokenType, loginToken)
	switch {
	case errors.Is(err, services.ErrTokenNotFound):
		http.Error(w, "unknown token", http.StatusUnauthorized)
		return

	case err != nil:
		http.Error(w, "error getting token", http.StatusBadRequest)
		return
	}

	var loginInfo jsonTypes.LoginInfo
	err = json.Unmarshal([]byte(redisValueString), &loginInfo)
	if err != nil {
		http.Error(w, "invalid login token", http.StatusBadRequest)
		return
	}

	response := GetLoginStateResponseDto{
		Step:                     string(loginInfo.Step),
		ApplicationDisplayName:   loginInfo.ApplicationDisplayName,
		VirtualServerDisplayName: loginInfo.VirtualServerDisplayName,
		VirtualServerName:        loginInfo.VirtualServerName,
		SignupEnabled:            loginInfo.RegistrationEnabled,
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

// VerifyPassword verifies user credentials for the login session.
// @Summary      Verify password
// @Tags         Logins
// @Accept       json
// @Produce      plain
// @Param        loginToken  path   string true  "Login session token"
// @Param        body        body   handlers.VerifyPasswordRequestDto true "Credentials"
// @Success      204         {string} string "No Content"
// @Failure      400         {string} string "Bad Request"
// @Failure      401         {string} string "Unauthorized or wrong step"
// @Router       /logins/{loginToken}/verify-password [post]
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

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Id(user.VirtualServerId())
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	switch {
	case passwordDetails.Temporary:
		loginInfo.Step = jsonTypes.LoginStepTemporaryPassword

	case !user.EmailVerified() && virtualServer.RequireEmailVerification():
		loginInfo.Step = jsonTypes.LoginStepEmailVerification

	// TODO: check if totp onboarding is needed
	// TODO: check if totp verification is needed

	default:
		loginInfo.Step = jsonTypes.LoginStepFinish
	}

	loginInfoString, err := json.Marshal(loginInfo)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	err = tokenService.UpdateToken(ctx, services.LoginSessionTokenType, loginToken, string(loginInfoString), time.Minute*15)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// VerifyEmailToken advances the login after the user's email is verified.
// @Summary      Verify email token (advance state)
// @Tags         Logins
// @Produce      plain
// @Param        loginToken  path   string true  "Login session token"
// @Success      204         {string} string "No Content"
// @Failure      400         {string} string "Bad Request"
// @Failure      401         {string} string "Unauthorized or wrong step"
// @Router       /logins/{loginToken}/verify-email [post]
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
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	err = tokenService.UpdateToken(ctx, services.LoginSessionTokenType, loginToken, string(loginInfoString), time.Minute*15)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// FinishLogin creates a session and redirects to the original URL.
// @Summary      Finish login
// @Tags         Logins
// @Produce      plain
// @Param        loginToken  path   string true  "Login session token"
// @Success      302         {string} string "Redirect to original URL"
// @Failure      400         {string} string "Bad Request"
// @Failure      401         {string} string "Unauthorized or wrong step"
// @Router       /logins/{loginToken}/finish-login [post]
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

// ResetTemporaryPassword sets a new password when the current one is temporary.
// @Summary      Reset temporary password
// @Tags         Logins
// @Accept       json
// @Produce      plain
// @Param        loginToken  path   string true  "Login session token"
// @Param        body        body   handlers.ResetTemporaryPasswordRequestDto true "New password"
// @Success      204         {string} string "No Content"
// @Failure      400         {string} string "Bad Request"
// @Failure      401         {string} string "Unauthorized or wrong step"
// @Router       /logins/{loginToken}/reset-temporary-password [post]
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

	m := ioc.GetDependency[mediator.MediatorInterface](scope)
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
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	err = tokenService.UpdateToken(ctx, services.LoginSessionTokenType, loginToken, string(loginInfoString), time.Minute*15)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ResendEmailVerification sends a new email verification message.
// @Summary      Resend email verification
// @Tags         Logins
// @Produce      plain
// @Param        loginToken  path   string true  "Login session token"
// @Success      204         {string} string "No Content"
// @Failure      400         {string} string "Bad Request"
// @Failure      401         {string} string "Unauthorized or wrong step"
// @Router       /logins/{loginToken}/resend-email-verification [post]
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
