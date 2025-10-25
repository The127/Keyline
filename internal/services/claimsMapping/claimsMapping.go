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
	Roles          []string
	GlobalMetadata map[string]interface{}
	AppMetadata    map[string]interface{}
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
		"roles": params.Roles,
	}
}

func (c *claimsMapper) MapClaims(ctx context.Context, applicationId uuid.UUID, params Params) map[string]any {
	scope := middlewares.GetScope(ctx)

	applicationRepository := ioc.GetDependency[repositories.ApplicationRepository](scope)
	applicationFilter := repositories.NewApplicationFilter().
		Id(applicationId)
	application, err := applicationRepository.First(ctx, applicationFilter)
	if err != nil {
		logging.Logger.Error(fmt.Errorf("falling back to default mapping, failed getting application: %w", err))
		return defaultMapping(params)
	}
	if application == nil {
		logging.Logger.Error(fmt.Errorf("falling back to default mapping, application not found"))
		return defaultMapping(params)
	}

	claimsMappingScript := application.ClaimsMappingScript()
	if claimsMappingScript == nil {
		// no need to log here, this is the default behaviour
		return defaultMapping(params)
	}

	mappedClaims, err := c.runCustomClaimsMappingScript(claimsMappingScript, params)
	if err != nil {
		logging.Logger.Error(fmt.Errorf("falling back to default mapping, failed running custom claims mapping script: %w", err))
		return defaultMapping(params)
	}

	return mappedClaims
}

func (c *claimsMapper) runCustomClaimsMappingScript(claimsMappingScript *string, params Params) (map[string]any, error) {
	vm := goja.New()

	err := vm.Set("roles", params.Roles)
	if err != nil {
		return nil, fmt.Errorf("failed setting roles: %w", err)
	}

	err = vm.Set("globalMetadata", params.GlobalMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed setting globalMetadata: %w", err)
	}

	err = vm.Set("appMetadata", params.AppMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed setting appMetadata: %w", err)
	}

	p, err := goja.Compile("mappingScript.js", *claimsMappingScript, true)
	if err != nil {
		return nil, fmt.Errorf("failed compiling script: %w", err)
	}

	result, err := vm.RunProgram(p)
	if err != nil {
		return nil, fmt.Errorf("failed running script: %w", err)
	}

	mappingResult, ok := result.Export().(map[string]any)
	if !ok {
		return nil, fmt.Errorf("failed casting result to map[string]any")
	}

	return mappingResult, nil
}
