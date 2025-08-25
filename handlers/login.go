package handlers

import (
	"Keyline/ioc"
	"Keyline/middlewares"
	"Keyline/services"
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http"
)

type GetLoginStateResponseDto struct {
	Step string `json:"step"`
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

	var loginInfo LoginInfo
	err = json.Unmarshal([]byte(redisValueString), &loginInfo)
	if err != nil {
		http.Error(w, "invalid login token", http.StatusBadRequest)
		return
	}

	response := GetLoginStateResponseDto{
		Step: string(loginInfo.Step),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "invalid login token", http.StatusBadRequest)
		return
	}
}
