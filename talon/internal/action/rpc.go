package action

import (
	"context"

	"github.com/ajepe/talon/internal/orm"
	"github.com/ajepe/talon/internal/rpc"
)

func RegisterRPC(s *rpc.Server, mgr *Manager) {
	s.Register("model.ir.action.get_action_value", func(ctx context.Context, params []interface{}) (interface{}, error) {
		if len(params) < 1 {
			return nil, &rpc.UserError{Message: "get_action_value requires an action"}
		}
		actionID, ok := orm.AsInt(params[0])
		if !ok {
			switch value := params[0].(type) {
			case string:
				if _, id, ok := ParseReference(value); ok {
					actionID = id
				} else {
					return nil, &rpc.UserError{Message: "invalid action reference"}
				}
			default:
				return nil, &rpc.UserError{Message: "invalid action"}
			}
		}
		rpcContext := map[string]interface{}{}
		if len(params) > 1 {
			if m, ok := params[1].(map[string]interface{}); ok {
				rpcContext = m
			}
		}
		return mgr.GetActionValue(ctx, actionID, rpcContext)
	})

	s.Register("model.ir.action.keyword.get_keyword", func(ctx context.Context, params []interface{}) (interface{}, error) {
		if len(params) < 3 {
			return nil, &rpc.UserError{Message: "get_keyword requires keyword, model and id"}
		}
		keyword, _ := params[0].(string)
		model, _ := params[1].(string)
		recordID, _ := orm.AsInt(params[2])
		return mgr.GetKeyword(ctx, keyword, model, recordID)
	})
}
