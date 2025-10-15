package claimsMapping

import (
	"Keyline/internal/logging"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"context"
	"fmt"

	"github.com/dop251/goja"
	"github.com/google/uuid"
)

type Params struct {
	Roles            []string
	ApplicationRoles []string
}

//go:generate mockgen -destination=../mocks/claimsMapping.go -package=mocks Keyline/internal/services/claimsMapping ClaimsMapper
type ClaimsMapper interface {
	MapClaims(ctx context.Context, ApplicationId uuid.UUID, params Params) map[string]any
}

type claimsMapper struct {
}

func NewClaimsMapper() ClaimsMapper {
	return &claimsMapper{}
}

func defaultMapping(params Params) map[string]any {
	return map[string]any{
		"roles":             params.Roles,
		"application_roles": params.ApplicationRoles,
	}
}

func (c *claimsMapper) MapClaims(ctx context.Context, ApplicationId uuid.UUID, params Params) map[string]any {
	scope := middlewares.GetScope(ctx)

	applicationRepository := ioc.GetDependency[repositories.ApplicationRepository](scope)
	applicationFilter := repositories.NewApplicationFilter().
		Id(ApplicationId)
	application, err := applicationRepository.First(ctx, applicationFilter)
	if err != nil {
		logging.Logger.Error(fmt.Errorf("falling back to default mapping, failed getting application: %w", err))
		return defaultMapping(params)
	}
	if application == nil {
		logging.Logger.Error(fmt.Errorf("falling back to default mapping, application not found"))
		return defaultMapping(params)
	}

	claimsMappingScript := application.GetClaimsMappingScript()
	if claimsMappingScript == nil {
		// no need to log here, this is the default behaviour
		return defaultMapping(params)
	}

	vm := goja.New()

	err = vm.Set("roles", params.Roles)
	if err != nil {
		logging.Logger.Error(fmt.Errorf("falling back to default mapping, failed setting roles: %w", err))
		return defaultMapping(params)
	}

	err = vm.Set("applicationRoles", params.ApplicationRoles)
	if err != nil {
		logging.Logger.Error(fmt.Errorf("falling back to default mapping, failed setting applicationRoles: %w", err))
		return defaultMapping(params)
	}

	p, err := goja.Compile("mappingScript.js", *claimsMappingScript, true)
	if err != nil {
		logging.Logger.Error(fmt.Errorf("falling back to default mapping, failed compiling script: %w", err))
		return defaultMapping(params)
	}

	result, err := vm.RunProgram(p)
	if err != nil {
		logging.Logger.Error(fmt.Errorf("falling back to default mapping, failed running script: %w", err))
		return defaultMapping(params)
	}

	mappingResult, ok := result.Export().(map[string]any)
	if !ok {
		logging.Logger.Error(fmt.Errorf("falling back to default mapping, failed casting result"))
		return defaultMapping(params)
	}

	return mappingResult
}
