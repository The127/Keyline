package handlers

import (
	"Keyline/ioc"
	"Keyline/middlewares"
	"Keyline/repositories"
	"Keyline/utils"
	"fmt"
	"github.com/google/uuid"
	"net/http"
)

func Debug(w http.ResponseWriter, r *http.Request) {
	scope := middlewares.GetScope(r.Context())
	userRepository := ioc.GetDependency[*repositories.UserRepository](scope)

	user := repositories.NewUser("foo", "bar", "", uuid.MustParse("3ed47d00-1cec-48c4-8be4-f258e91d7016"))
	err := userRepository.Insert(r.Context(), user)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	fmt.Printf("users: %v", user)

	w.WriteHeader(http.StatusOK)
}
