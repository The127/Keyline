package api

import "github.com/google/uuid"

type PagedPasswordRuleResponseDto struct {
	Items []ListPasswordRulesResponseDto `json:"items"`
}

type ListPasswordRulesResponseDto struct {
	Id      uuid.UUID      `json:"id"`
	Type    string         `json:"type"`
	Details map[string]any `json:"details"`
}

type CreatePasswordRuleRequestDto struct {
	Type    string                 `json:"type" validate:"required"`
	Details map[string]interface{} `json:"details" validate:"required"`
}

type PatchPasswordRuleRequestDto map[string]any
