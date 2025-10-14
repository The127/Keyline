package claimsMapping

type Params struct {
	Roles            []string
	ApplicationRoles []string
}

//go:generate mockgen -destination=../mocks/claimsMapping.go -package=mocks Keyline/internal/services/claimsMapping ClaimsMapper
type ClaimsMapper interface {
	MapClaims(params Params) map[string]any
}

type claimsMapper struct {
}

func NewClaimsMapper() ClaimsMapper {
	return &claimsMapper{}
}

func (c *claimsMapper) MapClaims(params Params) map[string]any {
	return map[string]any{
		"roles":             params.Roles,
		"application_roles": params.ApplicationRoles,
	}
}
