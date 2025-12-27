package handlers

import (
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/internal/services"
	"Keyline/utils"
	"fmt"
	"net/http"

	"github.com/The127/ioc"

	"github.com/google/uuid"
)

type Data struct {
	Link string
}

// Debug renders a test email template and returns 200.
// @Summary     Debug email template render
// @Tags        Debug
// @Produce     plain
// @Success     200 {string} string "OK"
// @Failure     500 {string} string
// @Router      /debug [get]
func Debug(w http.ResponseWriter, r *http.Request) {
	scope := middlewares.GetScope(r.Context())
	templateService := ioc.GetDependency[services.TemplateService](scope)

	data := Data{
		Link: "https://website.url/verifyemail/asodhrflaeawrhgawubhkawdf",
	}

	mailBody, err := templateService.Template(
		r.Context(),
		uuid.MustParse("3ed47d00-1cec-48c4-8be4-f258e91d7016"),
		repositories.EmailVerificationMailTemplate,
		data,
	)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	fmt.Printf("templated mail body: %v", mailBody)

	w.WriteHeader(http.StatusOK)
}
