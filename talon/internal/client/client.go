package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ajepe/talon/internal/rpc"
)

type Client struct {
	serverURL  string
	httpClient *http.Client
	codec      *rpc.Codec
	session    *Session
}

type Session struct {
	Token    string
	UserID   int
	Database string
	Context  map[string]interface{}
}

func NewClient(serverURL string) *Client {
	return &Client{
		serverURL:  serverURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		codec:      rpc.NewCodec(),
	}
}

func (c *Client) SetSession(s *Session) {
	c.session = s
}

func (c *Client) Session() *Session {
	return c.session
}

func (c *Client) Call(ctx context.Context, method string, params ...interface{}) (interface{}, error) {
	if c.session != nil && len(params) > 0 {
		if last, ok := params[len(params)-1].(map[string]interface{}); ok {
			for k, v := range c.session.Context {
				if _, exists := last[k]; !exists {
					last[k] = v
				}
			}
			params[len(params)-1] = last
		}
	}

	reqBody := map[string]interface{}{
		"id":     1,
		"method": method,
		"params": params,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/%s/rpc/", c.serverURL, c.session.Database)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.session != nil && c.session.Token != "" {
		req.Header.Set("Authorization", "session "+c.session.Token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("session expired")
	}

	if resp.StatusCode == 503 {
		return nil, fmt.Errorf("service unavailable")
	}

	var rpcResp struct {
		ID     int             `json:"id"`
		Result json.RawMessage `json:"result"`
		Error  json.RawMessage `json:"error"`
	}

	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, err
	}

	if len(rpcResp.Error) > 0 {
		var errValue interface{}
		if err := json.Unmarshal(rpcResp.Error, &errValue); err != nil {
			return nil, fmt.Errorf("invalid error response: %s", string(rpcResp.Error))
		}
		return nil, c.codec.DecodeError(errValue)
	}

	if len(rpcResp.Result) == 0 {
		return nil, nil
	}

	var result interface{}
	if err := json.Unmarshal(rpcResp.Result, &result); err != nil {
		return nil, err
	}

	return c.codec.DecodeResult(result), nil
}

type BatchCall struct {
	Method string
	Params []interface{}
}

func (c *Client) Batch(ctx context.Context, calls []BatchCall) ([]interface{}, error) {
	results := make([]interface{}, len(calls))
	for i, call := range calls {
		result, err := c.Call(ctx, call.Method, call.Params...)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

func (c *Client) Login(ctx context.Context, database, login, password string) error {
	result, err := c.Call(ctx, "common.db.login", login, map[string]interface{}{
		"password": password,
	}, map[string]interface{}{
		"language": "en",
	})
	if err != nil {
		return err
	}

	arr, ok := result.([]interface{})
	if !ok || len(arr) < 2 {
		return fmt.Errorf("invalid login response")
	}

	userID := 0
	if id, ok := arr[0].(float64); ok {
		userID = int(id)
	}

	c.session = &Session{
		Token:    fmt.Sprintf("%s:%d", login, userID),
		UserID:   userID,
		Database: database,
		Context:  map[string]interface{}{},
	}

	return nil
}

func (c *Client) Logout(ctx context.Context) error {
	if c.session == nil {
		return nil
	}
	_, err := c.Call(ctx, "common.db.logout")
	c.session = nil
	return err
}

func (c *Client) ServerVersion(ctx context.Context) (string, error) {
	result, err := c.Call(ctx, "common.server.version")
	if err != nil {
		return "", err
	}
	if s, ok := result.(string); ok {
		return s, nil
	}
	return "", fmt.Errorf("invalid version response")
}

func (c *Client) DBList(ctx context.Context) ([]string, error) {
	result, err := c.Call(ctx, "common.db.list")
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		dbs := make([]string, len(arr))
		for i, v := range arr {
			if s, ok := v.(string); ok {
				dbs[i] = s
			}
		}
		return dbs, nil
	}
	return nil, fmt.Errorf("invalid db list response")
}

func (c *Client) FieldsGet(ctx context.Context, model string, fields []string, context map[string]interface{}) (map[string]interface{}, error) {
	result, err := c.Call(ctx, fmt.Sprintf("model.%s.fields_get", model), fields, context)
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]interface{}); ok {
		return m, nil
	}
	return nil, fmt.Errorf("invalid fields_get response")
}

func (c *Client) SearchRead(ctx context.Context, model string, domain []interface{}, fields []string, limit, offset int, order string, context map[string]interface{}) ([]map[string]interface{}, error) {
	params := []interface{}{domain, offset, limit, orderParams(order), fields, context}
	result, err := c.Call(ctx, fmt.Sprintf("model.%s.search_read", model), params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		records := make([]map[string]interface{}, len(arr))
		for i, v := range arr {
			if m, ok := v.(map[string]interface{}); ok {
				records[i] = m
			}
		}
		return records, nil
	}
	return nil, fmt.Errorf("invalid search_read response")
}

func (c *Client) Read(ctx context.Context, model string, ids []int, fields []string, context map[string]interface{}) ([]map[string]interface{}, error) {
	params := []interface{}{ids, fields, context}
	result, err := c.Call(ctx, fmt.Sprintf("model.%s.read", model), params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		records := make([]map[string]interface{}, len(arr))
		for i, v := range arr {
			if m, ok := v.(map[string]interface{}); ok {
				records[i] = m
			}
		}
		return records, nil
	}
	return nil, fmt.Errorf("invalid read response")
}

func (c *Client) Create(ctx context.Context, model string, values []map[string]interface{}, context map[string]interface{}) ([]int, error) {
	params := []interface{}{values, context}
	result, err := c.Call(ctx, fmt.Sprintf("model.%s.create", model), params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		ids := make([]int, len(arr))
		for i, v := range arr {
			if f, ok := v.(float64); ok {
				ids[i] = int(f)
			}
		}
		return ids, nil
	}
	return nil, fmt.Errorf("invalid create response")
}

func (c *Client) Write(ctx context.Context, model string, ids []int, values map[string]interface{}, context map[string]interface{}) error {
	params := []interface{}{ids, values, context}
	_, err := c.Call(ctx, fmt.Sprintf("model.%s.write", model), params...)
	return err
}

func (c *Client) Delete(ctx context.Context, model string, ids []int, context map[string]interface{}) error {
	params := []interface{}{ids, context}
	_, err := c.Call(ctx, fmt.Sprintf("model.%s.delete", model), params...)
	return err
}

func (c *Client) OnChange(ctx context.Context, model string, values map[string]interface{}, fields []string, context map[string]interface{}) (map[string]interface{}, error) {
	params := []interface{}{values, fields, context}
	result, err := c.Call(ctx, fmt.Sprintf("model.%s.on_change", model), params...)
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]interface{}); ok {
		return m, nil
	}
	return nil, fmt.Errorf("invalid on_change response")
}

func (c *Client) OnChangeWith(ctx context.Context, model string, values map[string]interface{}, fields []string, context map[string]interface{}) (map[string]interface{}, error) {
	params := []interface{}{values, fields, context}
	result, err := c.Call(ctx, fmt.Sprintf("model.%s.on_change_with", model), params...)
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]interface{}); ok {
		return m, nil
	}
	return nil, fmt.Errorf("invalid on_change_with response")
}

func (c *Client) DefaultGet(ctx context.Context, model string, fields []string, context map[string]interface{}) (map[string]interface{}, error) {
	params := []interface{}{fields, context}
	result, err := c.Call(ctx, fmt.Sprintf("model.%s.default_get", model), params...)
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]interface{}); ok {
		return m, nil
	}
	return nil, fmt.Errorf("invalid default_get response")
}

func (c *Client) FieldsViewGet(ctx context.Context, model string, viewID int, viewType string, context map[string]interface{}) (map[string]interface{}, error) {
	params := []interface{}{viewID, viewType, context}
	result, err := c.Call(ctx, fmt.Sprintf("model.%s.fields_view_get", model), params...)
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]interface{}); ok {
		return m, nil
	}
	return nil, fmt.Errorf("invalid fields_view_get response")
}

func (c *Client) ViewToolbarGet(ctx context.Context, model string, context map[string]interface{}) (map[string]interface{}, error) {
	params := []interface{}{context}
	result, err := c.Call(ctx, fmt.Sprintf("model.%s.view_toolbar_get", model), params...)
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]interface{}); ok {
		return m, nil
	}
	return nil, fmt.Errorf("invalid view_toolbar_get response")
}

func (c *Client) GetPreferences(ctx context.Context, contextOnly bool, context map[string]interface{}) (map[string]interface{}, error) {
	params := []interface{}{contextOnly, context}
	result, err := c.Call(ctx, "model.res.user.get_preferences", params...)
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]interface{}); ok {
		return m, nil
	}
	return nil, fmt.Errorf("invalid get_preferences response")
}

func (c *Client) GetModelAccess(ctx context.Context, model string, context map[string]interface{}) (map[string]interface{}, error) {
	params := []interface{}{context}
	result, err := c.Call(ctx, fmt.Sprintf("model.ir.model.access.get_access"), params...)
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]interface{}); ok {
		return m, nil
	}
	return nil, fmt.Errorf("invalid get_access response")
}

func (c *Client) GetModelNames(ctx context.Context, context map[string]interface{}) ([]string, error) {
	params := []interface{}{context}
	result, err := c.Call(ctx, "model.ir.model.list_models", params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		names := make([]string, len(arr))
		for i, v := range arr {
			if s, ok := v.(string); ok {
				names[i] = s
			}
		}
		return names, nil
	}
	return nil, fmt.Errorf("invalid list_models response")
}

func (c *Client) GetNames(ctx context.Context, model string, ids []int, context map[string]interface{}) (map[int]string, error) {
	params := []interface{}{ids, context}
	result, err := c.Call(ctx, fmt.Sprintf("model.ir.model.get_names"), params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		names := make(map[int]string)
		for _, v := range arr {
			if m, ok := v.(map[string]interface{}); ok {
				if id, ok := m["id"].(float64); ok {
					if name, ok := m["rec_name"].(string); ok {
						names[int(id)] = name
					}
				}
			}
		}
		return names, nil
	}
	return nil, fmt.Errorf("invalid get_names response")
}

func (c *Client) GetActionValue(ctx context.Context, actionID int, context map[string]interface{}) (map[string]interface{}, error) {
	params := []interface{}{actionID, context}
	result, err := c.Call(ctx, "model.ir.action.get_action_value", params...)
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]interface{}); ok {
		return m, nil
	}
	return nil, fmt.Errorf("invalid get_action_value response")
}

func (c *Client) GetKeyword(ctx context.Context, keyword string, model string, recordID int, context map[string]interface{}) ([]map[string]interface{}, error) {
	params := []interface{}{keyword, model, recordID, context}
	result, err := c.Call(ctx, "model.ir.action.keyword.get_keyword", params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		actions := make([]map[string]interface{}, len(arr))
		for i, v := range arr {
			if m, ok := v.(map[string]interface{}); ok {
				actions[i] = m
			}
		}
		return actions, nil
	}
	return nil, fmt.Errorf("invalid get_keyword response")
}

func (c *Client) GetViewSearch(ctx context.Context, model string, context map[string]interface{}) ([]map[string]interface{}, error) {
	params := []interface{}{model, context}
	result, err := c.Call(ctx, "model.ir.ui.view_search.get", params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		searches := make([]map[string]interface{}, len(arr))
		for i, v := range arr {
			if m, ok := v.(map[string]interface{}); ok {
				searches[i] = m
			}
		}
		return searches, nil
	}
	return nil, fmt.Errorf("invalid view_search.get response")
}

func (c *Client) SetViewSearch(ctx context.Context, model string, name string, domain []interface{}, context map[string]interface{}) error {
	params := []interface{}{model, name, domain, context}
	_, err := c.Call(ctx, "model.ir.ui.view_search.set", params...)
	return err
}

func (c *Client) UnsetViewSearch(ctx context.Context, model string, name string, context map[string]interface{}) error {
	params := []interface{}{model, name, context}
	_, err := c.Call(ctx, "model.ir.ui.view_search.unset", params...)
	return err
}

func (c *Client) GetTreeState(ctx context.Context, model string, context map[string]interface{}) (map[string]interface{}, error) {
	params := []interface{}{model, context}
	result, err := c.Call(ctx, "model.ir.ui.view_tree_state.get", params...)
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]interface{}); ok {
		return m, nil
	}
	return nil, fmt.Errorf("invalid view_tree_state.get response")
}

func (c *Client) SetTreeState(ctx context.Context, model string, state map[string]interface{}, context map[string]interface{}) error {
	params := []interface{}{model, state, context}
	_, err := c.Call(ctx, "model.ir.ui.view_tree_state.set", params...)
	return err
}

func (c *Client) GetNotification(ctx context.Context, context map[string]interface{}) ([]map[string]interface{}, error) {
	params := []interface{}{context}
	result, err := c.Call(ctx, "model.res.notification.get", params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		notifications := make([]map[string]interface{}, len(arr))
		for i, v := range arr {
			if m, ok := v.(map[string]interface{}); ok {
				notifications[i] = m
			}
		}
		return notifications, nil
	}
	return nil, fmt.Errorf("invalid notification.get response")
}

func (c *Client) GetNotificationCount(ctx context.Context, context map[string]interface{}) (int, error) {
	params := []interface{}{context}
	result, err := c.Call(ctx, "model.res.notification.get_count", params...)
	if err != nil {
		return 0, err
	}
	if f, ok := result.(float64); ok {
		return int(f), nil
	}
	return 0, fmt.Errorf("invalid notification.get_count response")
}

func (c *Client) MarkRead(ctx context.Context, ids []int, context map[string]interface{}) error {
	params := []interface{}{ids, context}
	_, err := c.Call(ctx, "model.res.notification.mark_read", params...)
	return err
}

func (c *Client) GetAttachment(ctx context.Context, model string, id int, field string, context map[string]interface{}) ([]byte, error) {
	params := []interface{}{model, id, field, context}
	result, err := c.Call(ctx, "model.ir.attachment.read", params...)
	if err != nil {
		return nil, err
	}
	if s, ok := result.(string); ok {
		return []byte(s), nil
	}
	return nil, fmt.Errorf("invalid attachment.read response")
}

func (c *Client) SearchAttachment(ctx context.Context, domain []interface{}, context map[string]interface{}) ([]map[string]interface{}, error) {
	params := []interface{}{domain, context}
	result, err := c.Call(ctx, "model.ir.attachment.search_read", params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		attachments := make([]map[string]interface{}, len(arr))
		for i, v := range arr {
			if m, ok := v.(map[string]interface{}); ok {
				attachments[i] = m
			}
		}
		return attachments, nil
	}
	return nil, fmt.Errorf("invalid attachment.search_read response")
}

func (c *Client) GetIcon(ctx context.Context, name string, context map[string]interface{}) (string, error) {
	params := []interface{}{name, context}
	result, err := c.Call(ctx, "model.ir.ui.icon.read", params...)
	if err != nil {
		return "", err
	}
	if s, ok := result.(string); ok {
		return s, nil
	}
	return "", fmt.Errorf("invalid icon.read response")
}

func (c *Client) ListIcons(ctx context.Context, context map[string]interface{}) ([]string, error) {
	params := []interface{}{context}
	result, err := c.Call(ctx, "model.ir.ui.icon.list_icons", params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		icons := make([]string, len(arr))
		for i, v := range arr {
			if s, ok := v.(string); ok {
				icons[i] = s
			}
		}
		return icons, nil
	}
	return nil, fmt.Errorf("invalid list_icons response")
}

func (c *Client) GetChatModels(ctx context.Context, context map[string]interface{}) ([]string, error) {
	params := []interface{}{context}
	result, err := c.Call(ctx, "model.ir.chat.channel.get_models", params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		models := make([]string, len(arr))
		for i, v := range arr {
			if s, ok := v.(string); ok {
				models[i] = s
			}
		}
		return models, nil
	}
	return nil, fmt.Errorf("invalid get_models response")
}

func (c *Client) GetChatChannel(ctx context.Context, model string, id int, context map[string]interface{}) (map[string]interface{}, error) {
	params := []interface{}{model, id, context}
	result, err := c.Call(ctx, "model.ir.chat.channel.get", params...)
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]interface{}); ok {
		return m, nil
	}
	return nil, fmt.Errorf("invalid chat.channel.get response")
}

func (c *Client) PostChat(ctx context.Context, model string, id int, message string, context map[string]interface{}) error {
	params := []interface{}{model, id, message, context}
	_, err := c.Call(ctx, "model.ir.chat.channel.post", params...)
	return err
}

func (c *Client) GetChatFollowers(ctx context.Context, model string, id int, context map[string]interface{}) ([]int, error) {
	params := []interface{}{model, id, context}
	result, err := c.Call(ctx, "model.ir.chat.channel.get_followers", params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		followers := make([]int, len(arr))
		for i, v := range arr {
			if f, ok := v.(float64); ok {
				followers[i] = int(f)
			}
		}
		return followers, nil
	}
	return nil, fmt.Errorf("invalid get_followers response")
}

func (c *Client) SearchChatFollowers(ctx context.Context, model string, id int, text string, context map[string]interface{}) ([]int, error) {
	params := []interface{}{model, id, text, context}
	result, err := c.Call(ctx, "model.ir.chat.channel.search_followers", params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		followers := make([]int, len(arr))
		for i, v := range arr {
			if f, ok := v.(float64); ok {
				followers[i] = int(f)
			}
		}
		return followers, nil
	}
	return nil, fmt.Errorf("invalid search_followers response")
}

func (c *Client) SubscribeChat(ctx context.Context, model string, id int, context map[string]interface{}) error {
	params := []interface{}{model, id, context}
	_, err := c.Call(ctx, "model.ir.chat.channel.subscribe", params...)
	return err
}

func (c *Client) UnsubscribeChat(ctx context.Context, model string, id int, context map[string]interface{}) error {
	params := []interface{}{model, id, context}
	_, err := c.Call(ctx, "model.ir.chat.channel.unsubscribe", params...)
	return err
}

func (c *Client) ExportData(ctx context.Context, model string, ids []int, fields []string, context map[string]interface{}) ([][]interface{}, error) {
	params := []interface{}{ids, fields, context}
	result, err := c.Call(ctx, fmt.Sprintf("model.%s.export_data", model), params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		data := make([][]interface{}, len(arr))
		for i, v := range arr {
			if row, ok := v.([]interface{}); ok {
				data[i] = row
			}
		}
		return data, nil
	}
	return nil, fmt.Errorf("invalid export_data response")
}

func (c *Client) ExportDataDomain(ctx context.Context, model string, domain []interface{}, fields []string, context map[string]interface{}) ([][]interface{}, error) {
	params := []interface{}{domain, fields, context}
	result, err := c.Call(ctx, fmt.Sprintf("model.%s.export_data_domain", model), params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		data := make([][]interface{}, len(arr))
		for i, v := range arr {
			if row, ok := v.([]interface{}); ok {
				data[i] = row
			}
		}
		return data, nil
	}
	return nil, fmt.Errorf("invalid export_data_domain response")
}

func (c *Client) ImportData(ctx context.Context, model string, fields []string, data [][]interface{}, context map[string]interface{}) error {
	params := []interface{}{fields, data, context}
	_, err := c.Call(ctx, fmt.Sprintf("model.%s.import_data", model), params...)
	return err
}

func (c *Client) Copy(ctx context.Context, model string, ids []int, defaults map[string]interface{}, context map[string]interface{}) ([]int, error) {
	params := []interface{}{ids, defaults, context}
	result, err := c.Call(ctx, fmt.Sprintf("model.%s.copy", model), params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		newIDs := make([]int, len(arr))
		for i, v := range arr {
			if f, ok := v.(float64); ok {
				newIDs[i] = int(f)
			}
		}
		return newIDs, nil
	}
	return nil, fmt.Errorf("invalid copy response")
}

func (c *Client) PreValidate(ctx context.Context, model string, values map[string]interface{}, context map[string]interface{}) error {
	params := []interface{}{values, context}
	_, err := c.Call(ctx, fmt.Sprintf("model.%s.pre_validate", model), params...)
	return err
}

func (c *Client) Resources(ctx context.Context, model string, ids []int, context map[string]interface{}) (map[string]interface{}, error) {
	params := []interface{}{ids, context}
	result, err := c.Call(ctx, fmt.Sprintf("model.%s.resources", model), params...)
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]interface{}); ok {
		return m, nil
	}
	return nil, fmt.Errorf("invalid resources response")
}

func (c *Client) HistoryRevisions(ctx context.Context, model string, ids []int, context map[string]interface{}) ([]map[string]interface{}, error) {
	params := []interface{}{ids, context}
	result, err := c.Call(ctx, fmt.Sprintf("model.%s.history_revisions", model), params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		revisions := make([]map[string]interface{}, len(arr))
		for i, v := range arr {
			if m, ok := v.(map[string]interface{}); ok {
				revisions[i] = m
			}
		}
		return revisions, nil
	}
	return nil, fmt.Errorf("invalid history_revisions response")
}

func (c *Client) OnScanCode(ctx context.Context, model string, field string, code string, context map[string]interface{}) (map[string]interface{}, error) {
	params := []interface{}{field, code, context}
	result, err := c.Call(ctx, fmt.Sprintf("model.%s.on_scan_code", model), params...)
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]interface{}); ok {
		return m, nil
	}
	return nil, fmt.Errorf("invalid on_scan_code response")
}

func (c *Client) Autocomplete(ctx context.Context, model string, field string, text string, domain []interface{}, limit int, order string, context map[string]interface{}) ([]interface{}, error) {
	params := []interface{}{text, domain, limit, order, context}
	result, err := c.Call(ctx, fmt.Sprintf("model.%s.autocomplete_%s", model, field), params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		return arr, nil
	}
	return nil, fmt.Errorf("invalid autocomplete response")
}

func (c *Client) Search(ctx context.Context, model string, domain []interface{}, offset, limit int, order string, context map[string]interface{}) ([]int, error) {
	params := []interface{}{domain, offset, limit, orderParams(order), context}
	result, err := c.Call(ctx, fmt.Sprintf("model.%s.search", model), params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		ids := make([]int, len(arr))
		for i, v := range arr {
			if f, ok := v.(float64); ok {
				ids[i] = int(f)
			}
		}
		return ids, nil
	}
	return nil, fmt.Errorf("invalid search response")
}

func orderParams(order string) []interface{} {
	pairs := []interface{}{}
	for _, part := range strings.Split(order, ",") {
		fields := strings.Fields(strings.TrimSpace(part))
		if len(fields) == 0 {
			continue
		}
		direction := "ASC"
		if len(fields) > 1 {
			direction = strings.ToUpper(fields[1])
		}
		pairs = append(pairs, []interface{}{fields[0], direction})
	}
	return pairs
}

func (c *Client) SearchCount(ctx context.Context, model string, domain []interface{}, context map[string]interface{}) (int, error) {
	params := []interface{}{domain, context}
	result, err := c.Call(ctx, fmt.Sprintf("model.%s.search_count", model), params...)
	if err != nil {
		return 0, err
	}
	if f, ok := result.(float64); ok {
		return int(f), nil
	}
	return 0, fmt.Errorf("invalid search_count response")
}

func (c *Client) Get(ctx context.Context, model string, id int, fields []string, context map[string]interface{}) (map[string]interface{}, error) {
	result, err := c.Read(ctx, model, []int{id}, fields, context)
	if err != nil {
		return nil, err
	}
	if len(result) > 0 {
		return result[0], nil
	}
	return nil, fmt.Errorf("record not found")
}

func (c *Client) CreateOne(ctx context.Context, model string, values map[string]interface{}, context map[string]interface{}) (int, error) {
	ids, err := c.Create(ctx, model, []map[string]interface{}{values}, context)
	if err != nil {
		return 0, err
	}
	if len(ids) > 0 {
		return ids[0], nil
	}
	return 0, fmt.Errorf("create failed")
}

func (c *Client) WriteOne(ctx context.Context, model string, id int, values map[string]interface{}, context map[string]interface{}) error {
	return c.Write(ctx, model, []int{id}, values, context)
}

func (c *Client) DeleteOne(ctx context.Context, model string, id int, context map[string]interface{}) error {
	return c.Delete(ctx, model, []int{id}, context)
}

func (c *Client) GetButtonClicks(ctx context.Context, model string, ids []int, context map[string]interface{}) (map[string]interface{}, error) {
	params := []interface{}{ids, context}
	result, err := c.Call(ctx, "model.ir.model.button.click.get_click", params...)
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]interface{}); ok {
		return m, nil
	}
	return nil, fmt.Errorf("invalid get_click response")
}

func (c *Client) GetGlobalSearch(ctx context.Context, text string, limit int, context map[string]interface{}) ([]map[string]interface{}, error) {
	params := []interface{}{text, limit, context}
	result, err := c.Call(ctx, "model.ir.model.global_search", params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		results := make([]map[string]interface{}, len(arr))
		for i, v := range arr {
			if m, ok := v.(map[string]interface{}); ok {
				results[i] = m
			}
		}
		return results, nil
	}
	return nil, fmt.Errorf("invalid global_search response")
}

func (c *Client) GetLanguages(ctx context.Context, context map[string]interface{}) ([]map[string]interface{}, error) {
	params := []interface{}{context}
	result, err := c.Call(ctx, "model.ir.lang.search_read", params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		languages := make([]map[string]interface{}, len(arr))
		for i, v := range arr {
			if m, ok := v.(map[string]interface{}); ok {
				languages[i] = m
			}
		}
		return languages, nil
	}
	return nil, fmt.Errorf("invalid lang.search_read response")
}

func (c *Client) GetExport(ctx context.Context, context map[string]interface{}) ([]map[string]interface{}, error) {
	params := []interface{}{context}
	result, err := c.Call(ctx, "model.ir.export.get", params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		exports := make([]map[string]interface{}, len(arr))
		for i, v := range arr {
			if m, ok := v.(map[string]interface{}); ok {
				exports[i] = m
			}
		}
		return exports, nil
	}
	return nil, fmt.Errorf("invalid export.get response")
}

func (c *Client) SetExport(ctx context.Context, name string, fields []string, context map[string]interface{}) error {
	params := []interface{}{name, fields, context}
	_, err := c.Call(ctx, "model.ir.export.set", params...)
	return err
}

func (c *Client) UpdateExport(ctx context.Context, name string, fields []string, context map[string]interface{}) error {
	params := []interface{}{name, fields, context}
	_, err := c.Call(ctx, "model.ir.export.update", params...)
	return err
}

func (c *Client) UnsetExport(ctx context.Context, name string, context map[string]interface{}) error {
	params := []interface{}{name, context}
	_, err := c.Call(ctx, "model.ir.export.unset", params...)
	return err
}

func (c *Client) GetMenuFavorites(ctx context.Context, context map[string]interface{}) ([]map[string]interface{}, error) {
	params := []interface{}{context}
	result, err := c.Call(ctx, "model.ir.ui.menu.favorite.get", params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		favorites := make([]map[string]interface{}, len(arr))
		for i, v := range arr {
			if m, ok := v.(map[string]interface{}); ok {
				favorites[i] = m
			}
		}
		return favorites, nil
	}
	return nil, fmt.Errorf("invalid menu.favorite.get response")
}

func (c *Client) SetMenuFavorite(ctx context.Context, menuID int, context map[string]interface{}) error {
	params := []interface{}{menuID, context}
	_, err := c.Call(ctx, "model.ir.ui.menu.favorite.set", params...)
	return err
}

func (c *Client) UnsetMenuFavorite(ctx context.Context, menuID int, context map[string]interface{}) error {
	params := []interface{}{menuID, context}
	_, err := c.Call(ctx, "model.ir.ui.menu.favorite.unset", params...)
	return err
}

func (c *Client) GetTreeWidth(ctx context.Context, model string, context map[string]interface{}) (map[string]interface{}, error) {
	params := []interface{}{model, context}
	result, err := c.Call(ctx, "model.ir.ui.view_tree_width.get_width", params...)
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]interface{}); ok {
		return m, nil
	}
	return nil, fmt.Errorf("invalid view_tree_width.get_width response")
}

func (c *Client) GetTreeOptional(ctx context.Context, model string, context map[string]interface{}) (map[string]interface{}, error) {
	params := []interface{}{model, context}
	result, err := c.Call(ctx, "model.ir.ui.view_tree_optional.get_optional", params...)
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]interface{}); ok {
		return m, nil
	}
	return nil, fmt.Errorf("invalid view_tree_optional.get_optional response")
}

func (c *Client) GetView(ctx context.Context, viewID int, context map[string]interface{}) (map[string]interface{}, error) {
	params := []interface{}{viewID, context}
	result, err := c.Call(ctx, "model.ir.ui.view.view_get", params...)
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]interface{}); ok {
		return m, nil
	}
	return nil, fmt.Errorf("invalid view.view_get response")
}

func (c *Client) GetCron(ctx context.Context, context map[string]interface{}) ([]map[string]interface{}, error) {
	params := []interface{}{context}
	result, err := c.Call(ctx, "model.ir.cron.search_read", params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		crons := make([]map[string]interface{}, len(arr))
		for i, v := range arr {
			if m, ok := v.(map[string]interface{}); ok {
				crons[i] = m
			}
		}
		return crons, nil
	}
	return nil, fmt.Errorf("invalid cron.search_read response")
}

func (c *Client) GetAuthenticationServices(ctx context.Context) ([]map[string]interface{}, error) {
	result, err := c.Call(context.Background(), "common.authentication.services")
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		services := make([]map[string]interface{}, len(arr))
		for i, v := range arr {
			if m, ok := v.(map[string]interface{}); ok {
				services[i] = m
			}
		}
		return services, nil
	}
	return nil, fmt.Errorf("invalid authentication.services response")
}

func (c *Client) ResetPassword(ctx context.Context, database, login string) error {
	params := []interface{}{login}
	_, err := c.Call(ctx, "common.db.reset_password", params...)
	return err
}

func (c *Client) GetPreferencesFieldsView(ctx context.Context, context map[string]interface{}) (map[string]interface{}, error) {
	params := []interface{}{context}
	result, err := c.Call(ctx, "model.res.user.get_preferences_fields_view", params...)
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]interface{}); ok {
		return m, nil
	}
	return nil, fmt.Errorf("invalid get_preferences_fields_view response")
}

func (c *Client) SetPreferences(ctx context.Context, values map[string]interface{}, context map[string]interface{}) error {
	params := []interface{}{values, context}
	_, err := c.Call(ctx, "model.res.user.set_preferences", params...)
	return err
}

func (c *Client) GetWarning(ctx context.Context, name string, context map[string]interface{}) (map[string]interface{}, error) {
	params := []interface{}{name, context}
	result, err := c.Call(ctx, "model.res.user.warning", params...)
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]interface{}); ok {
		return m, nil
	}
	return nil, fmt.Errorf("invalid warning response")
}

func (c *Client) SkipWarning(ctx context.Context, name string, always bool, context map[string]interface{}) error {
	params := []interface{}{name, always, context}
	_, err := c.Call(ctx, "model.res.user.warning.skip", params...)
	return err
}

func (c *Client) GetModelHistory(ctx context.Context, model string, context map[string]interface{}) ([]map[string]interface{}, error) {
	params := []interface{}{model, context}
	result, err := c.Call(ctx, "model.ir.model.list_history", params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		history := make([]map[string]interface{}, len(arr))
		for i, v := range arr {
			if m, ok := v.(map[string]interface{}); ok {
				history[i] = m
			}
		}
		return history, nil
	}
	return nil, fmt.Errorf("invalid list_history response")
}

func (c *Client) GetModelNotification(ctx context.Context, model string, context map[string]interface{}) ([]map[string]interface{}, error) {
	params := []interface{}{model, context}
	result, err := c.Call(ctx, "model.ir.model.get_notification", params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		notifications := make([]map[string]interface{}, len(arr))
		for i, v := range arr {
			if m, ok := v.(map[string]interface{}); ok {
				notifications[i] = m
			}
		}
		return notifications, nil
	}
	return nil, fmt.Errorf("invalid get_notification response")
}

func (c *Client) GetDictKeys(ctx context.Context, model string, context map[string]interface{}) ([]string, error) {
	params := []interface{}{model, context}
	result, err := c.Call(ctx, fmt.Sprintf("model.%s.get_keys", model), params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		keys := make([]string, len(arr))
		for i, v := range arr {
			if s, ok := v.(string); ok {
				keys[i] = s
			}
		}
		return keys, nil
	}
	return nil, fmt.Errorf("invalid get_keys response")
}

func (c *Client) SearchDictKeys(ctx context.Context, model string, text string, context map[string]interface{}) ([]string, error) {
	params := []interface{}{text, context}
	result, err := c.Call(ctx, fmt.Sprintf("model.%s.search_get_keys", model), params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		keys := make([]string, len(arr))
		for i, v := range arr {
			if s, ok := v.(string); ok {
				keys[i] = s
			}
		}
		return keys, nil
	}
	return nil, fmt.Errorf("invalid search_get_keys response")
}

func (c *Client) GetDictSchema(ctx context.Context, model string, context map[string]interface{}) (map[string]interface{}, error) {
	params := []interface{}{model, context}
	result, err := c.Call(ctx, fmt.Sprintf("model.%s.get_schema", model), params...)
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]interface{}); ok {
		return m, nil
	}
	return nil, fmt.Errorf("invalid get_schema response")
}

func (c *Client) GetSelection(ctx context.Context, model string, field string, context map[string]interface{}) ([]interface{}, error) {
	params := []interface{}{context}
	result, err := c.Call(ctx, fmt.Sprintf("model.%s.%s.selection", model, field), params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		return arr, nil
	}
	return nil, fmt.Errorf("invalid selection response")
}

func (c *Client) GetSelectionChangeWith(ctx context.Context, model string, field string, context map[string]interface{}) ([]interface{}, error) {
	params := []interface{}{context}
	result, err := c.Call(ctx, fmt.Sprintf("model.%s.%s.selection_change_with", model, field), params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		return arr, nil
	}
	return nil, fmt.Errorf("invalid selection_change_with response")
}

func (c *Client) GetReferenceModels(ctx context.Context, model string, field string, context map[string]interface{}) ([]string, error) {
	params := []interface{}{context}
	result, err := c.Call(ctx, fmt.Sprintf("model.%s.%s.get_models", model, field), params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		models := make([]string, len(arr))
		for i, v := range arr {
			if s, ok := v.(string); ok {
				models[i] = s
			}
		}
		return models, nil
	}
	return nil, fmt.Errorf("invalid get_models response")
}

func (c *Client) GetReferenceDomain(ctx context.Context, model string, field string, context map[string]interface{}) ([]interface{}, error) {
	params := []interface{}{context}
	result, err := c.Call(ctx, fmt.Sprintf("model.%s.%s.get_domain", model, field), params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		return arr, nil
	}
	return nil, fmt.Errorf("invalid get_domain response")
}

func (c *Client) GetReferenceOrder(ctx context.Context, model string, field string, context map[string]interface{}) (string, error) {
	params := []interface{}{context}
	result, err := c.Call(ctx, fmt.Sprintf("model.%s.%s.get_order", model, field), params...)
	if err != nil {
		return "", err
	}
	if s, ok := result.(string); ok {
		return s, nil
	}
	return "", fmt.Errorf("invalid get_order response")
}

func (c *Client) GetFieldDomain(ctx context.Context, model string, field string, context map[string]interface{}) ([]interface{}, error) {
	params := []interface{}{context}
	result, err := c.Call(ctx, fmt.Sprintf("model.%s.%s.get_domain", model, field), params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		return arr, nil
	}
	return nil, fmt.Errorf("invalid get_domain response")
}

func (c *Client) GetFieldContext(ctx context.Context, model string, field string, context map[string]interface{}) (map[string]interface{}, error) {
	params := []interface{}{context}
	result, err := c.Call(ctx, fmt.Sprintf("model.%s.%s.get_context", model, field), params...)
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]interface{}); ok {
		return m, nil
	}
	return nil, fmt.Errorf("invalid get_context response")
}

func (c *Client) GetFieldSearchContext(ctx context.Context, model string, field string, context map[string]interface{}) (map[string]interface{}, error) {
	params := []interface{}{context}
	result, err := c.Call(ctx, fmt.Sprintf("model.%s.%s.get_search_context", model, field), params...)
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]interface{}); ok {
		return m, nil
	}
	return nil, fmt.Errorf("invalid get_search_context response")
}

func (c *Client) GetFieldSearchOrder(ctx context.Context, model string, field string, context map[string]interface{}) (string, error) {
	params := []interface{}{context}
	result, err := c.Call(ctx, fmt.Sprintf("model.%s.%s.get_search_order", model, field), params...)
	if err != nil {
		return "", err
	}
	if s, ok := result.(string); ok {
		return s, nil
	}
	return "", fmt.Errorf("invalid get_search_order response")
}

func (c *Client) GetFieldSearchDomain(ctx context.Context, model string, field string, context map[string]interface{}) ([]interface{}, error) {
	params := []interface{}{context}
	result, err := c.Call(ctx, fmt.Sprintf("model.%s.%s.get_search_domain", model, field), params...)
	if err != nil {
		return nil, err
	}
	if arr, ok := result.([]interface{}); ok {
		return arr, nil
	}
	return nil, fmt.Errorf("invalid get_search_domain response")
}
