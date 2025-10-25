package handlers

import (
	"Keyline/internal/middlewares"
	"Keyline/internal/queries"
	"Keyline/ioc"
	"Keyline/mediator"
	"Keyline/utils"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

type PagedPasswordRuleResponseDto struct {
	Items []ListPasswordRulesResponseDto `json:"items"`
}

type ListPasswordRulesResponseDto struct {
	Id      uuid.UUID      `json:"id"`
	Type    string         `json:"type"`
	Details map[string]any `json:"details"`
}

// ListPasswordRules
// @summary     List password rules
// @description Retrieve all password rules of a virtual server.
// @tags        Password rules
// @produce     application/json
// @param       virtualServerName  path   string  true  "Virtual server name"  default(keyline)
// @param       page query int true "Page number"  default(1)
// @param       pageSize query int true "Page size"  default(10)
// @success     200 {object} PagedPasswordRuleResponseDto
// @failure     400  {string}  string "Bad Request"
// @router      /api/virtual-servers/{virtualServerName}/password-policies/rules [get]
func ListPasswordRules(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[mediator.Mediator](scope)

	rules, err := mediator.Send[*queries.ListPasswordRulesResponse](ctx, m, queries.ListPasswordRules{
		VirtualServerName: vsName,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	response := PagedPasswordRuleResponseDto{}

	for _, rule := range rules.Items {
		details := make(map[string]any)
		err = json.Unmarshal(rule.Details, &details)
		if err != nil {
			utils.HandleHttpError(w, err)
			return
		}

		response.Items = append(response.Items, ListPasswordRulesResponseDto{
			Id:      rule.Id,
			Type:    string(rule.Type),
			Details: details,
		})
	}

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}
}

func CreatePasswordRule(w http.ResponseWriter, r *http.Request) {

}

func GetPasswordRule(w http.ResponseWriter, r *http.Request) {

}

func PatchPasswordRule(w http.ResponseWriter, r *http.Request) {

}
