package claimsMapping

import (
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"context"

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
		return defaultMapping(params)
	}
	if application == nil {
		return defaultMapping(params)
	}

	claimsMappingScript := application.GetClaimsMappingScript()
	if claimsMappingScript == nil {
		return defaultMapping(params)
	}

	vm := goja.New()

	err = vm.Set("roles", params.Roles)
	if err != nil {
		panic(err)
	}

	err = vm.Set("applicationRoles", params.ApplicationRoles)
	if err != nil {
		panic(err)
	}

	p, err := goja.Compile("mappingScript.js", *claimsMappingScript, true)
	if err != nil {
		panic(err)
	}

	result, err := vm.RunProgram(p)
	if err != nil {
		return defaultMapping(params)
	}

	mappingResult, ok := result.Export().(map[string]any)
	if !ok {
		return defaultMapping(params)
	}

	return mappingResult
}
