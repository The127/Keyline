package handlers

import (
	"Keyline/internal/clock"
	"Keyline/internal/commands"
	"Keyline/internal/config"
	"Keyline/internal/jsonTypes"
	"Keyline/internal/messages"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/internal/services"
	"Keyline/internal/services/keyValue"
	"Keyline/ioc"
	"Keyline/mediator"
	"Keyline/templates"
	"Keyline/utils"
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

func updateLoginStep(
	ctx context.Context,
	loginToken string,
	mutate func(info *jsonTypes.LoginInfo) error,
) error {
	scope := middlewares.GetScope(ctx)
	tokenService := ioc.GetDependency[services.TokenService](scope)

	redisValueString, err := tokenService.GetToken(ctx, services.LoginSessionTokenType, loginToken)
	if err != nil {
		return fmt.Errorf("getting token: %w", err)
	}

	var loginInfo jsonTypes.LoginInfo
	if err := json.Unmarshal([]byte(redisValueString), &loginInfo); err != nil {
		return fmt.Errorf("unmarshal login info: %w", err)
	}

	if mutate == nil {
		return fmt.Errorf("mutate function is nil")
	}

	if err := mutate(&loginInfo); err != nil {
		return fmt.Errorf("mutate login info: %w", err)
	}

	loginInfo.Step, err = DetermineNextLoginStep(ctx, &loginInfo)
	if err != nil {
		return fmt.Errorf("determine next login step: %w", err)
	}

	updated, err := json.Marshal(loginInfo)
	if err != nil {
		return fmt.Errorf("marshal login info: %w", err)
	}

	if err := tokenService.UpdateToken(ctx, services.LoginSessionTokenType, loginToken, string(updated), 15*time.Minute); err != nil {
		return fmt.Errorf("update token: %w", err)
	}

	return nil
}

// DetermineNextLoginStep decides what the next login step should be
// based on the current step, user state, and server configuration.
func DetermineNextLoginStep(
	ctx context.Context,
	loginInfo *jsonTypes.LoginInfo,
) (jsonTypes.LoginStep, error) {
	if loginInfo.Step == jsonTypes.LoginStepPasskey {
		return jsonTypes.LoginStepFinish, nil
	}

	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Id(loginInfo.VirtualServerId)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return "", err
	}

	userRepository := ioc.GetDependency[repositories.UserRepository](scope)
	userFilter := repositories.NewUserFilter().VirtualServerId(loginInfo.VirtualServerId).Id(loginInfo.UserId)
	user, err := userRepository.Single(ctx, userFilter)
	if err != nil {
		return "", err
	}

	credentialRepository := ioc.GetDependency[repositories.CredentialRepository](scope)
	passwordFilter := repositories.NewCredentialFilter().UserId(user.Id()).Type(repositories.CredentialTypePassword)
	passwordCredential, err := credentialRepository.Single(ctx, passwordFilter)
	if err != nil {
		return "", err
	}
	passwordDetails, err := passwordCredential.PasswordDetails()
	if err != nil {
		return "", err
	}

	totpFilter := repositories.NewCredentialFilter().UserId(user.Id()).Type(repositories.CredentialTypeTotp)
	totpCredentials, err := credentialRepository.List(ctx, totpFilter)
	if err != nil {
		return "", err
	}

	switch loginInfo.Step {
	case jsonTypes.LoginStepPasswordVerification:
		if passwordDetails.Temporary {
			return jsonTypes.LoginStepTemporaryPassword, nil
		}
		fallthrough

	case jsonTypes.LoginStepTemporaryPassword:
		if !user.EmailVerified() && virtualServer.RequireEmailVerification() {
			return jsonTypes.LoginStepEmailVerification, nil
		}
		fallthrough

	case jsonTypes.LoginStepEmailVerification:
		if len(totpCredentials) > 0 {
			return jsonTypes.LoginStepVerifyTotp, nil
		}
		if virtualServer.Require2fa() {
			loginInfo.TotpSecret = base32.StdEncoding.EncodeToString(utils.GetSecureRandomBytes(32))
			return jsonTypes.LoginStepOnboardTotp, nil
		}
		return jsonTypes.LoginStepFinish, nil

	case jsonTypes.LoginStepOnboardTotp:
		fallthrough

	case jsonTypes.LoginStepVerifyTotp:
		return jsonTypes.LoginStepFinish, nil

	default:
		return "", errors.New("invalid login step")
	}
}

type GetLoginStateResponseDto struct {
	// Step is one of: password_verification | temporary_password | email_verification | finish
	Step                     string `json:"step"`
	ApplicationDisplayName   string `json:"applicationDisplayName"`
	VirtualServerDisplayName string `json:"virtualServerDisplayName"`
	VirtualServerName        string `json:"virtualServerName"`
	SignupEnabled            bool   `json:"signupEnabled"`
	TotpSecret               string `json:"totpSecret"`
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
	rawTokenData, err := tokenService.GetToken(ctx, services.LoginSessionTokenType, loginToken)
	switch {
	case errors.Is(err, services.ErrTokenNotFound):
		http.Error(w, "unknown token", http.StatusUnauthorized)
		return

	case err != nil:
		http.Error(w, "error getting token", http.StatusBadRequest)
		return
	}

	var loginInfo jsonTypes.LoginInfo
	err = json.Unmarshal([]byte(rawTokenData), &loginInfo)
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
		TotpSecret:               loginInfo.TotpSecret,
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

	var dto VerifyPasswordRequestDto
	err := json.NewDecoder(r.Body).Decode(&dto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	err = utils.ValidateDto(dto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	err = updateLoginStep(ctx, loginToken, func(loginInfo *jsonTypes.LoginInfo) error {
		if loginInfo.Step != jsonTypes.LoginStepPasswordVerification {
			return utils.ErrHttpUnauthorized
		}

		userRepository := ioc.GetDependency[repositories.UserRepository](scope)
		userFilter := repositories.NewUserFilter().Username(dto.Username)
		user, err := userRepository.First(ctx, userFilter)
		if err != nil {
			return err
		}
		if user == nil {
			return utils.ErrHttpUnauthorized
		}

		credentialRepository := ioc.GetDependency[repositories.CredentialRepository](scope)
		credentialFilter := repositories.NewCredentialFilter().
			UserId(user.Id()).
			Type(repositories.CredentialTypePassword)
		credential, err := credentialRepository.Single(ctx, credentialFilter)
		if err != nil {
			return utils.ErrHttpUnauthorized
		}

		passwordDetails, err := credential.PasswordDetails()
		if err != nil {
			return utils.ErrHttpUnauthorized
		}

		if !utils.CompareHash(dto.Password, passwordDetails.HashedPassword) {
			return utils.ErrHttpUnauthorized
		}

		loginInfo.UserId = user.Id()
		return nil
	})
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

	err := updateLoginStep(ctx, loginToken, func(loginInfo *jsonTypes.LoginInfo) error {
		if loginInfo.Step != jsonTypes.LoginStepEmailVerification {
			return utils.ErrHttpUnauthorized
		}

		userRepository := ioc.GetDependency[repositories.UserRepository](scope)
		userFilter := repositories.NewUserFilter().Id(loginInfo.UserId)
		user, err := userRepository.Single(ctx, userFilter)
		if err != nil {
			return err
		}

		if !user.EmailVerified() {
			return utils.ErrHttpUnauthorized
		}

		return nil
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type OnboardTotpRequestDto struct {
	TotpCode string `json:"totpCode" validate:"required"`
}

// OnboardTotp advances the login after the user has onboarded TOTP.
// @Summary      Onboard TOTP (advance state)
// @Tags         Logins
// @Accept       json
// @Produce      plain
// @Param        loginToken  path   string true  "Login session token"
// @Param        body        body   handlers.OnboardTotpRequestDto true "TOTP code"
// @Success      204         {string} string "No Content"
// @Failure      400         {string} string "Bad Request"
// @Failure      401         {string} string "Unauthorized or wrong step"
// @Router       /logins/{loginToken}/onboard-totp [post]
func OnboardTotp(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vars := mux.Vars(r)
	loginToken := vars["loginToken"]

	var dto OnboardTotpRequestDto
	err := json.NewDecoder(r.Body).Decode(&dto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	err = utils.ValidateDto(dto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	err = updateLoginStep(ctx, loginToken, func(loginInfo *jsonTypes.LoginInfo) error {
		if loginInfo.Step != jsonTypes.LoginStepOnboardTotp {
			return utils.ErrHttpUnauthorized
		}

		isValid := totp.Validate(dto.TotpCode, loginInfo.TotpSecret)
		if !isValid {
			return fmt.Errorf("invalid totp code: %w", utils.ErrHttpBadRequest)
		}

		totpCredential := repositories.NewCredential(loginInfo.UserId, &repositories.CredentialTotpDetails{
			Secret:    loginInfo.TotpSecret,
			Digits:    int(otp.DigitsSix),
			Algorithm: int(otp.AlgorithmSHA1),
		})
		credentialRepository := ioc.GetDependency[repositories.CredentialRepository](scope)
		err := credentialRepository.Insert(ctx, totpCredential)
		if err != nil {
			return err
		}

		loginInfo.TotpSecret = ""

		return nil
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type VerifyTotpRequestDto struct {
	TotpCode string `json:"totpCode" validate:"required"`
}

// VerifyTotp advances the login after the user has verified TOTP.
// @Summary      Verify TOTP (advance state)
// @Tags         Logins
// @Produce      plain
// @Param        loginToken  path   string true  "Login session token"
// @Param        body        body   handlers.VerifyTotpRequestDto true "TOTP code"
// @Success      204         {string} string "No Content"
// @Failure      400         {string} string "Bad Request"
// @Failure      401         {string} string "Unauthorized or wrong step"
// @Router       /logins/{loginToken}/verify-totp [post]
func VerifyTotp(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vars := mux.Vars(r)
	loginToken := vars["loginToken"]

	var dto VerifyTotpRequestDto
	err := json.NewDecoder(r.Body).Decode(&dto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	err = utils.ValidateDto(dto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	err = updateLoginStep(ctx, loginToken, func(loginInfo *jsonTypes.LoginInfo) error {
		if loginInfo.Step != jsonTypes.LoginStepVerifyTotp {
			return utils.ErrHttpUnauthorized
		}

		credentialRepository := ioc.GetDependency[repositories.CredentialRepository](scope)
		totpCredentialFilter := repositories.NewCredentialFilter().
			UserId(loginInfo.UserId).
			Type(repositories.CredentialTypeTotp)
		totpCredentials, err := credentialRepository.List(ctx, totpCredentialFilter)
		if err != nil {
			return fmt.Errorf("failed to get totp credentials: %w", err)
		}

		clockService := ioc.GetDependency[clock.Service](scope)
		now := clockService.Now()

		var isValid = false
		for _, credential := range totpCredentials {
			details, err := credential.TotpDetails()
			if err != nil {
				return fmt.Errorf("failed to get totp details: %w", err)
			}
			opts := totp.ValidateOpts{
				Period:    30,
				Skew:      1,
				Digits:    otp.Digits(details.Digits),
				Algorithm: otp.Algorithm(details.Algorithm),
			}
			valid, err := totp.ValidateCustom(dto.TotpCode, details.Secret, now, opts)
			if err != nil {
				return fmt.Errorf("failed to validate totp code: %w", err)
			}
			if valid {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("invalid totp code: %w", utils.ErrHttpBadRequest)
		}

		return nil
	})
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
	rawTokenData, err := tokenService.GetToken(ctx, services.LoginSessionTokenType, loginToken)
	if err != nil {
		http.Error(w, "invalid login token", http.StatusBadRequest)
		return
	}

	var loginInfo jsonTypes.LoginInfo
	err = json.Unmarshal([]byte(rawTokenData), &loginInfo)
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

	err = tokenService.DeleteToken(ctx, services.LoginSessionTokenType, loginToken)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

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

	err := updateLoginStep(ctx, loginToken, func(loginInfo *jsonTypes.LoginInfo) error {
		if loginInfo.Step != jsonTypes.LoginStepTemporaryPassword {
			return utils.ErrHttpUnauthorized
		}

		var dto ResetTemporaryPasswordRequestDto
		err := json.NewDecoder(r.Body).Decode(&dto)
		if err != nil {
			return err
		}

		err = utils.ValidateDto(dto)
		if err != nil {
			return err
		}

		m := ioc.GetDependency[mediator.Mediator](scope)
		_, err = mediator.Send[*commands.ResetPasswordResponse](ctx, m, commands.ResetPassword{
			UserId:      loginInfo.UserId,
			NewPassword: dto.NewPassword,
			Temporary:   false,
		})
		if err != nil {
			return err
		}

		return nil
	})
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
	rawTokenData, err := tokenService.GetToken(ctx, services.LoginSessionTokenType, loginToken)
	if err != nil {
		http.Error(w, "invalid login token", http.StatusBadRequest)
		return
	}

	var loginInfo jsonTypes.LoginInfo
	err = json.Unmarshal([]byte(rawTokenData), &loginInfo)
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
	outboxMessage, err := repositories.NewOutboxMessage(message)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("creating email outbox message: %w", err))
		return
	}

	err = outboxMessageRepository.Insert(ctx, outboxMessage)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("creating email outbox message: %w", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type StartPasskeyLoginResponseDto struct {
	Id        uuid.UUID `json:"id"`
	Challenge string    `json:"challenge"`
}

func StartPasskeyLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vars := mux.Vars(r)
	loginToken := vars["loginToken"]

	challengeBytes := utils.GetSecureRandomBytes(64)

	challenge := jsonTypes.PasskeyLoginChallenge{
		Id:                uuid.New(),
		Challenge:         base64.StdEncoding.EncodeToString(challengeBytes),
		LoginSessionToken: loginToken,
	}

	challengeJson, err := json.Marshal(challenge)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	kvStore := ioc.GetDependency[keyValue.Store](scope)
	err = kvStore.Set(ctx, "passkey_login:"+challenge.Id.String(), string(challengeJson), keyValue.WithExpiration(time.Minute*5))
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(StartPasskeyLoginResponseDto{
		Id:        challenge.Id,
		Challenge: challenge.Challenge,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}
}

type FinishPasskeyLoginRequestDto struct {
	Id               uuid.UUID `json:"id" validate:"required"`
	WebauthnResponse struct {
		Id       string `json:"id"`
		RawId    string `json:"rawId"`
		Response struct {
			ClientDataJSON    string `json:"clientDataJSON"`
			AuthenticatorData string `json:"authenticatorData"`
			Signature         string `json:"signature"`
			UserHandle        string `json:"userHandle"`
		} `json:"response"`
		AuthenticatorAttachment string `json:"authenticatorAttachment"`
		Type                    string `json:"type"`
	} `json:"webauthnResponse" validate:"required"`
}

func FinishPasskeyLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vars := mux.Vars(r)
	loginToken := vars["loginToken"]

	// decode request
	var dto FinishPasskeyLoginRequestDto
	err := json.NewDecoder(r.Body).Decode(&dto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	err = utils.ValidateDto(dto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	// get from kv store
	kvStore := ioc.GetDependency[keyValue.Store](scope)
	challengeKey := "passkey_login:" + dto.Id.String()
	challengeJson, err := kvStore.Get(ctx, challengeKey)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("challenge expired or missing"))
		return
	}

	var challenge jsonTypes.PasskeyLoginChallenge
	if err := json.Unmarshal([]byte(challengeJson), &challenge); err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	// validate challenge
	if challenge.LoginSessionToken != loginToken {
		utils.HandleHttpError(w, fmt.Errorf("challenge mismatch"))
		return
	}

	clientDataBytes, err := base64.StdEncoding.DecodeString(dto.WebauthnResponse.Response.ClientDataJSON)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	var clientData struct {
		Type      string `json:"type"`
		Challenge string `json:"challenge"`
		Origin    string `json:"origin"`
	}
	if err := json.Unmarshal(clientDataBytes, &clientData); err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	if clientData.Type != "webauthn.get" {
		utils.HandleHttpError(w, fmt.Errorf("invalid clientData type"))
		return
	}

	challengeFromClient, err := base64.RawURLEncoding.DecodeString(clientData.Challenge)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	actualChallenge, err := base64.RawURLEncoding.DecodeString(challenge.Challenge)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	if !bytes.Equal(challengeFromClient, actualChallenge) {
		utils.HandleHttpError(w, fmt.Errorf("challenge mismatch: %w", utils.ErrHttpUnauthorized))
		return
	}

	authData, err := base64.RawURLEncoding.DecodeString(dto.WebauthnResponse.Response.AuthenticatorData)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	// get credential from database
	credentialRepository := ioc.GetDependency[repositories.CredentialRepository](scope)
	credentialFilter := repositories.NewCredentialFilter().
		DetailsId(dto.WebauthnResponse.RawId).
		Type(repositories.CredentialTypeWebauthn)
	credential, err := credentialRepository.Single(ctx, credentialFilter)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	credentialDetails, err := credential.WebauthnDetails()
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	clientHash := sha256.Sum256(clientDataBytes)
	signedData := make([]byte, len(authData)+len(clientHash))
	copy(signedData, authData)
	copy(signedData[len(authData):], clientHash[:])

	sigBytes, err := base64.RawURLEncoding.DecodeString(dto.WebauthnResponse.Response.Signature)
	if err != nil {
		utils.HandleHttpError(w, fmt.Errorf("invalid signature encoding: %w", err))
		return
	}

	pubKey, err := x509.ParsePKIXPublicKey(credentialDetails.PublicKey)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	err = validateSignature(pubKey, credentialDetails.PublicKeyAlgorithm, signedData, sigBytes)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	err = updateLoginStep(ctx, loginToken, func(loginInfo *jsonTypes.LoginInfo) error {
		loginInfo.UserId = credential.UserId()
		loginInfo.Step = jsonTypes.LoginStepPasskey
		return nil
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

var ErrSignatureInvalid = fmt.Errorf("signature verification failed: %w", utils.ErrHttpUnauthorized)
var ErrSignatureInvalidAlgorithm = errors.New("invalid public key algorithm")

const (
	CoseAlgorithmES256   = -7
	CoseAlgorithmEd25519 = -8 // COSE calls this EdDSA and marks it as deprecated, but implementations seem to use it for Ed25519 instead of -19 (which is what COSE uses for Ed25519)
	CoseAlgorithmPS256   = -37
	CoseAlgorithmRS256   = -257
)

func validateSignature(pubKey any, pubKeyAlgorithm int, message, sigBytes []byte) error {
	switch k := pubKey.(type) {
	case *rsa.PublicKey:
		switch pubKeyAlgorithm {
		case CoseAlgorithmRS256:
			err := rsa.VerifyPKCS1v15(k, crypto.SHA256, message, sigBytes)
			if err != nil {
				return ErrSignatureInvalid
			}

		case CoseAlgorithmPS256:
			err := rsa.VerifyPSS(k, crypto.SHA256, message, sigBytes, nil)
			if err != nil {
				return ErrSignatureInvalid
			}

		default:
			return ErrSignatureInvalidAlgorithm
		}
	case *ecdsa.PublicKey:
		if pubKeyAlgorithm != CoseAlgorithmES256 {
			return ErrSignatureInvalidAlgorithm
		}

		hash := sha256.Sum256(message)

		if !ecdsa.VerifyASN1(k, hash[:], sigBytes) {
			return ErrSignatureInvalid
		}

	case ed25519.PublicKey:
		if pubKeyAlgorithm != CoseAlgorithmEd25519 {
			return ErrSignatureInvalidAlgorithm
		}

		if !ed25519.Verify(k, message, sigBytes) {
			return ErrSignatureInvalid
		}

	default:
		return ErrSignatureInvalidAlgorithm
	}

	return nil
}
