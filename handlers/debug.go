package handlers

import (
	"Keyline/ioc"
	"Keyline/middlewares"
	"Keyline/repositories"
	"Keyline/utils"
	"fmt"
	"net/http"
)

func Debug(w http.ResponseWriter, r *http.Request) {
	scope := middlewares.GetScope(r.Context())
	userRepository := ioc.GetDependency[*repositories.UserRepository](scope)

	filter := repositories.NewUserFilter()
	user, err := userRepository.First(r.Context(), filter)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	fmt.Printf("user: %v", user)

	w.WriteHeader(http.StatusOK)
}
