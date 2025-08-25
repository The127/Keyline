package handlers

import (
	"Keyline/ioc"
	"Keyline/jsonTypes"
	"Keyline/middlewares"
	"Keyline/services"
	"Keyline/utils"
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http"
)

type GetLoginStateResponseDto struct {
	Step                     string `json:"step"`
	ApplicationDisplayName   string `json:"applicationDisplayName"`
	VirtualServerDisplayName string `json:"virtualServerDisplayName"`
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

	response := GetLoginStateResponseDto{
		Step:                     string(loginInfo.Step),
		ApplicationDisplayName:   loginInfo.ApplicationDisplayName,
		VirtualServerDisplayName: loginInfo.VirtualServerDisplayName,
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

	// TODO: call business logic handler
}
