package handlers

import (
	"Keyline/internal/commands"
	"Keyline/internal/middlewares"
	"Keyline/internal/queries"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"Keyline/utils"
	"encoding/json"
	"fmt"
	"github.com/The127/mediatr"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
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
	m := ioc.GetDependency[mediatr.Mediator](scope)

	rules, err := mediatr.Send[*queries.ListPasswordRulesResponse](ctx, m, queries.ListPasswordRules{
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

type CreatePasswordRuleRequestDto struct {
	Type    string                 `json:"type" validate:"required"`
	Details map[string]interface{} `json:"details" validate:"required"`
}

// CreatePasswordRule
// @summary     Create password rule
// @description Create a password rule for a virtual server.
// @tags        Password rules
// @accept      application/json
// @param       virtualServerName  path   string  true  "Virtual server name"  default(keyline)
// @param       body  body   CreatePasswordRuleRequestDto  true  "Password rule details"
// @success     204 "No Content"
// @failure     400  {string}  string "Bad Request"
// @failure     409  {string}  string "Conflict"
// @router      /api/virtual-servers/{virtualServerName}/password-policies/rules/{ruleType} [post]
func CreatePasswordRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	var requestDto CreatePasswordRuleRequestDto
	err = json.NewDecoder(r.Body).Decode(&requestDto)
	if err != nil {
		utils.HandleHttpError(w, err)
	}

	err = utils.ValidateDto(requestDto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[mediatr.Mediator](scope)

	_, err = mediatr.Send[*commands.CreatePasswordRuleResponse](ctx, m, commands.CreatePasswordRule{
		VirtualServerName: vsName,
		Type:              repositories.PasswordRuleType(requestDto.Type),
		Details:           requestDto.Details,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type PatchPasswordRuleRequestDto map[string]any

// UpdatePasswordRule
// @summary     Update a password rule
// @description Update a password rule for a virtual server.
// @tags        Password rules
// @accept      application/json
// @param       virtualServerName  path   string  true  "Virtual server name"  default(keyline)
// @param       body  body   PatchPasswordRuleRequestDto  true  "Password rule details"
// @success     204 "No Content"
// @failure     400  {string}  string "Bad Request"
// @failure     404  {string}  string "Not Found"
// @router      /api/virtual-servers/{virtualServerName}/password-policies/rules/{ruleType} [put]
func UpdatePasswordRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)
	ruleTypeString, ok := vars["ruleType"]
	if !ok {
		utils.HandleHttpError(w, fmt.Errorf("ruleType is required: %w", utils.ErrHttpBadRequest))
		return
	}

	ruleType := repositories.PasswordRuleType(ruleTypeString)

	var requestDto PatchPasswordRuleRequestDto
	err = json.NewDecoder(r.Body).Decode(&requestDto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[mediatr.Mediator](scope)

	_, err = mediatr.Send[*commands.UpdatePasswordRuleResponse](ctx, m, commands.UpdatePasswordRule{
		VirtualServerName: vsName,
		Type:              ruleType,
		Details:           requestDto,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
