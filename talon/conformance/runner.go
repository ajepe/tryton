package conformance

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ajepe/talon/internal/client"
)

var defaultIgnored = []string{"create_date", "write_date", "_timestamp", "create_uid", "write_uid"}

type Runner struct {
	pythonURL  string
	goURL      string
	dbName     string
	results    []Result
	timeout    time.Duration
	pythonAuth string
	goAuth     string
}

type Result struct {
	Name      string        `json:"name"`
	Method    string        `json:"method"`
	Params    []interface{} `json:"params"`
	PythonRes interface{}   `json:"python_result"`
	GoRes     interface{}   `json:"go_result"`
	Match     bool          `json:"match"`
	Error     string        `json:"error,omitempty"`
}

func NewRunner(pythonURL, goURL, dbName string) *Runner {
	return &Runner{
		pythonURL: pythonURL,
		goURL:     goURL,
		dbName:    dbName,
		results:   []Result{},
		timeout:   30 * time.Second,
	}
}

func (r *Runner) SetTimeout(timeout time.Duration) {
	r.timeout = timeout
}

func (r *Runner) Run(ctx context.Context, cases []TestCase) error {
	for _, tc := range cases {
		if tc.Skip {
			continue
		}
		result := Result{
			Name:   tc.Name,
			Method: tc.Method,
			Params: tc.Params,
		}
		if result.Name == "" {
			result.Name = tc.Method
		}

		if err := r.applySetup(ctx, r.pythonURL, tc.Setup); err != nil {
			result.Error = fmt.Sprintf("python setup: %v", err)
			r.results = append(r.results, result)
			continue
		}
		if err := r.applySetup(ctx, r.goURL, tc.Setup); err != nil {
			result.Error = fmt.Sprintf("go setup: %v", err)
			r.results = append(r.results, result)
			continue
		}

		pyRes, err := r.callPython(ctx, tc.Method, tc.Params)
		if err != nil {
			result.Error = fmt.Sprintf("python: %v", err)
			r.results = append(r.results, result)
			continue
		}
		result.PythonRes = pyRes

		goRes, err := r.callGo(ctx, tc.Method, tc.Params)
		if err != nil {
			result.Error = fmt.Sprintf("go: %v", err)
			r.results = append(r.results, result)
			continue
		}
		result.GoRes = goRes

		ignore := append(append([]string{}, defaultIgnored...), tc.Ignore...)
		result.Match = r.compare(pyRes, goRes, ignore, tc.SortLists)
		r.results = append(r.results, result)
	}
	return nil
}

func (r *Runner) applySetup(ctx context.Context, baseURL string, setup []Call) error {
	for _, call := range setup {
		if _, err := r.call(ctx, baseURL, call.Method, call.Params); err != nil {
			return fmt.Errorf("%s: %w", call.Method, err)
		}
	}
	return nil
}

func (r *Runner) callPython(ctx context.Context, method string, params []interface{}) (interface{}, error) {
	return r.call(ctx, r.pythonURL, method, params)
}

func (r *Runner) callGo(ctx context.Context, method string, params []interface{}) (interface{}, error) {
	return r.call(ctx, r.goURL, method, params)
}

func (r *Runner) call(ctx context.Context, baseURL, method string, params []interface{}) (interface{}, error) {
	if params == nil {
		params = []interface{}{}
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

	url := fmt.Sprintf("%s/%s/rpc/", baseURL, r.dbName)
	if (strings.HasPrefix(method, "common.") || strings.HasPrefix(method, "system.")) &&
		method != "common.db.login" && method != "common.db.logout" {
		url = fmt.Sprintf("%s/rpc/", baseURL)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if auth := r.authFor(baseURL); auth != "" {
		req.Header.Set("Authorization", "session "+auth)
	}

	callClient := &http.Client{Timeout: r.timeout}
	resp, err := callClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rpcResp struct {
		ID     int             `json:"id"`
		Result json.RawMessage `json:"result"`
		Error  json.RawMessage `json:"error"`
	}

	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, fmt.Errorf("invalid response %q: %w", string(body), err)
	}

	if len(rpcResp.Error) > 0 {
		return nil, fmt.Errorf("rpc error: %s", string(rpcResp.Error))
	}

	if len(rpcResp.Result) == 0 {
		return nil, nil
	}

	var result interface{}
	if err := json.Unmarshal(rpcResp.Result, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *Runner) compare(a, b interface{}, ignore []string, sortLists bool) bool {
	ignoreSet := map[string]bool{}
	for _, key := range ignore {
		ignoreSet[key] = true
	}
	aJSON, _ := json.Marshal(normalize(a, ignoreSet, sortLists))
	bJSON, _ := json.Marshal(normalize(b, ignoreSet, sortLists))
	return string(aJSON) == string(bJSON)
}

func normalize(value interface{}, ignore map[string]bool, sortLists bool) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		out := map[string]interface{}{}
		for key, val := range v {
			if ignore[key] {
				continue
			}
			out[key] = normalize(val, ignore, sortLists)
		}
		return out
	case []interface{}:
		list := make([]interface{}, len(v))
		for i, item := range v {
			list[i] = normalize(item, ignore, sortLists)
		}
		if sortLists {
			sort.Slice(list, func(i, j int) bool {
				left, _ := json.Marshal(list[i])
				right, _ := json.Marshal(list[j])
				return string(left) < string(right)
			})
		}
		return list
	case float64:
		if v == math.Trunc(v) && math.Abs(v) < 1e15 {
			return int64(v)
		}
		return v
	default:
		return value
	}
}

func (r *Runner) Report() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Conformance Report: %d cases\n", len(r.results)))
	b.WriteString(strings.Repeat("=", 60) + "\n")

	passed := 0
	failed := 0
	for _, res := range r.results {
		if res.Match {
			passed++
			continue
		}
		failed++
		b.WriteString(fmt.Sprintf("FAIL: %s (%s)\n", res.Name, res.Method))
		if res.Error != "" {
			b.WriteString(fmt.Sprintf("  Error: %s\n", res.Error))
			continue
		}
		pyJSON, _ := json.Marshal(res.PythonRes)
		goJSON, _ := json.Marshal(res.GoRes)
		b.WriteString(fmt.Sprintf("  Python: %s\n", truncate(pyJSON, 500)))
		b.WriteString(fmt.Sprintf("  Go:     %s\n", truncate(goJSON, 500)))
	}

	b.WriteString(fmt.Sprintf("\nPassed: %d, Failed: %d\n", passed, failed))
	return b.String()
}

func (r *Runner) Failures() int {
	failed := 0
	for _, res := range r.results {
		if !res.Match {
			failed++
		}
	}
	return failed
}

func (r *Runner) Results() []Result {
	return r.results
}

func truncate(data []byte, limit int) string {
	if len(data) <= limit {
		return string(data)
	}
	return string(data[:limit]) + "..."
}

func (r *Runner) WriteReport(path string) error {
	data, err := json.MarshalIndent(r.results, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

type Call struct {
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
}

type TestCase struct {
	Name      string        `json:"name"`
	Method    string        `json:"method"`
	Params    []interface{} `json:"params"`
	Setup     []Call        `json:"setup"`
	Ignore    []string      `json:"ignore"`
	SortLists bool          `json:"sort_lists"`
	Skip      bool          `json:"skip"`
}

type Fixture struct {
	Name  string `json:"name"`
	Setup []Call `json:"setup"`
}

func LoadCases(dir string) ([]TestCase, error) {
	cases := []TestCase{}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		loaded, err := parseCases(data)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		cases = append(cases, loaded...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Logout must run last because it invalidates the shared session.
	sort.SliceStable(cases, func(i, j int) bool {
		return cases[i].Method != "common.db.logout" && cases[j].Method == "common.db.logout"
	})
	return cases, nil
}

func parseCases(data []byte) ([]TestCase, error) {
	var list []TestCase
	if err := json.Unmarshal(data, &list); err == nil {
		return list, nil
	}
	var wrapper struct {
		Cases []TestCase `json:"cases"`
	}
	if err := json.Unmarshal(data, &wrapper); err == nil && len(wrapper.Cases) > 0 {
		return wrapper.Cases, nil
	}
	var single TestCase
	if err := json.Unmarshal(data, &single); err != nil {
		return nil, err
	}
	return []TestCase{single}, nil
}

func LoadFixtures(dir string) ([]Fixture, error) {
	fixtures := []Fixture{}
	if dir == "" {
		return fixtures, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return fixtures, nil
		}
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		var fixture Fixture
		if err := json.Unmarshal(data, &fixture); err != nil {
			return nil, fmt.Errorf("%s: %w", entry.Name(), err)
		}
		fixtures = append(fixtures, fixture)
	}
	return fixtures, nil
}

func (r *Runner) ApplyFixtures(ctx context.Context, fixtures []Fixture) error {
	for _, fixture := range fixtures {
		if err := r.applySetup(ctx, r.pythonURL, fixture.Setup); err != nil {
			return fmt.Errorf("fixture %s (python): %w", fixture.Name, err)
		}
		if err := r.applySetup(ctx, r.goURL, fixture.Setup); err != nil {
			return fmt.Errorf("fixture %s (go): %w", fixture.Name, err)
		}
	}
	return nil
}

func (r *Runner) authFor(baseURL string) string {
	if baseURL == r.pythonURL {
		return r.pythonAuth
	}
	return r.goAuth
}

func (r *Runner) Authenticate(ctx context.Context, login, password string) error {
	pyAuth, err := r.login(ctx, r.pythonURL, login, password)
	if err != nil {
		return fmt.Errorf("python login: %w", err)
	}
	goAuth, err := r.login(ctx, r.goURL, login, password)
	if err != nil {
		return fmt.Errorf("go login: %w", err)
	}
	r.pythonAuth = pyAuth
	r.goAuth = goAuth
	return nil
}

func (r *Runner) login(ctx context.Context, baseURL, login, password string) (string, error) {
	body, err := json.Marshal(map[string]interface{}{
		"id":     0,
		"method": "common.db.login",
		"params": []interface{}{login, map[string]interface{}{"password": password}, "en"},
	})
	if err != nil {
		return "", err
	}
	url := fmt.Sprintf("%s/%s/session/login", baseURL, r.dbName)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	callClient := &http.Client{Timeout: r.timeout}
	resp, err := callClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	for _, cookie := range resp.Cookies() {
		if cookie.Name == "tryton_session" {
			return base64.StdEncoding.EncodeToString([]byte(cookie.Value)), nil
		}
	}
	return "", fmt.Errorf("login did not return a session cookie (status %d)", resp.StatusCode)
}

func StartPythonTrytond(dbName, dbPath string) (*exec.Cmd, error) {
	cmd := exec.Command("trytond", "-d", dbName, "-c", dbPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	time.Sleep(2 * time.Second)
	return cmd, nil
}

func StartGoServer(dbName, uri string) (*exec.Cmd, error) {
	cmd := exec.Command("go", "run", "./cmd/talon", "-database", dbName, "-uri", uri)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	time.Sleep(2 * time.Second)
	return cmd, nil
}

func (r *Runner) Login(ctx context.Context, username, password string) error {
	c := client.NewClient(r.goURL)
	return c.Login(ctx, r.dbName, username, password)
}

func (r *Runner) LoginPython(ctx context.Context, username, password string) error {
	c := client.NewClient(r.pythonURL)
	return c.Login(ctx, r.dbName, username, password)
}
