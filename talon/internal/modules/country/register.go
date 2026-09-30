package country

import (
	"github.com/ajepe/talon/internal/rpc"
)

func RegisterHandlers(registry *rpc.ModuleRegistry) {
	registry.Register("country.country", &rpc.ModuleHandler{
		OnChange:   OnChangeCountry,
		Search:     SearchCountry,
		Validate:   ValidateCountry,
		Preprocess: PreprocessCountry,
	})

	registry.Register("country.subdivision", &rpc.ModuleHandler{
		OnChange:   OnChangeSubdivision,
		Validate:   ValidateSubdivision,
		Preprocess: PreprocessSubdivision,
	})

	registry.Register("country.postal_code", &rpc.ModuleHandler{
		Search:   SearchPostalCode,
		Validate: ValidatePostalCode,
	})
}
