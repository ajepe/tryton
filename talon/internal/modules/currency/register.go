package currency

import (
	"context"

	"github.com/ajepe/talon/internal/orm"
	"github.com/ajepe/talon/internal/rpc"
)

func RegisterHandlers(registry *rpc.ModuleRegistry) {
	registry.Register("currency.currency", &rpc.ModuleHandler{
		Search:   SearchCurrency,
		Validate: ValidateCurrency,
	})

	registry.Register("currency.currency.rate", &rpc.ModuleHandler{
		Validate: ValidateRate,
	})
}

func RegisterRPC(srv *rpc.Server) {
	srv.Register("model.currency.currency.compute", func(ctx context.Context, params []interface{}) (interface{}, error) {
		if len(params) < 3 {
			return nil, &rpc.UserError{Message: "compute requires from currency, amount and to currency"}
		}
		fromID, ok := orm.AsInt(params[0])
		if !ok {
			return nil, &rpc.UserError{Message: "invalid source currency"}
		}
		amount, ok := orm.AsFloat(params[1])
		if !ok {
			return nil, &rpc.UserError{Message: "invalid amount"}
		}
		toID, ok := orm.AsInt(params[2])
		if !ok {
			return nil, &rpc.UserError{Message: "invalid target currency"}
		}
		rpcContext := map[string]interface{}{}
		if len(params) > 3 {
			if m, ok := params[3].(map[string]interface{}); ok {
				rpcContext = m
			}
		}
		group := orm.NewGroup(nil)
		group.SetDB(orm.DefaultPool().DB())
		result, err := Compute(ctx, group, fromID, amount, toID, rpcContext, true)
		if err != nil {
			return nil, &rpc.UserError{Message: err.Error()}
		}
		return result, nil
	})
}
