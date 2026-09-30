package company

import (
	"github.com/ajepe/talon/internal/rpc"
)

func RegisterHandlers(registry *rpc.ModuleRegistry) {
	registry.Register("company.company", &rpc.ModuleHandler{
		Search:   SearchCompany,
		Validate: ValidateCompany,
	})

	registry.Register("company.employee", &rpc.ModuleHandler{
		OnChange: OnChangeEmployee,
		Search:   SearchEmployee,
		Validate: ValidateEmployee,
	})
}
