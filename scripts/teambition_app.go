package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	defaultBaseURL  = "https://open.teambition.com/api"
	maxResponseSize = 16 << 20
	exitUsage       = 2
	exitConfig      = 3
	exitRequest     = 4
	exitAPI         = 5
)

type config struct {
	appID      string
	appSecret  string
	orgID      string
	operatorID string
	baseURL    string
}

type apiError struct {
	statusCode int
	body       string
}

func (e *apiError) Error() string {
	if e.statusCode != 0 {
		return fmt.Sprintf("Teambition API request failed (HTTP %d): %s", e.statusCode, e.body)
	}
	return "Teambition API request failed: " + e.body
}

func main() {
	client := &http.Client{Timeout: 30 * time.Second}
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, client, time.Now))
}

func run(args []string, stdout, stderr io.Writer, client *http.Client, now func() time.Time) int {
	if len(args) == 0 {
		printUsage(stderr)
		return exitUsage
	}

	switch args[0] {
	case "help", "--help", "-h":
		printUsage(stdout)
		return 0
	case "project":
		if len(args) >= 2 {
			switch args[1] {
			case "sprint-query":
				return runSprintQuery(args[2:], stdout, stderr, client, now)
			case "task-query":
				return runProjectTaskQuery(args[2:], stdout, stderr, client, now)
			case "priority-query":
				return runProjectPriorityQuery(args[2:], stdout, stderr, client, now)
			}
		}
	case "task":
		if len(args) >= 2 && args[1] == "update" {
			return runTaskUpdate(args[2:], stdout, stderr, client, now)
		}
	case "member":
		if len(args) >= 2 && args[1] == "query" {
			return runMemberQuery(args[2:], stdout, stderr, client, now)
		}
	}

	fmt.Fprintln(stderr, "unsupported command")
	printUsage(stderr)
	return exitUsage
}

func runMemberQuery(args []string, stdout, stderr io.Writer, client *http.Client, now func() time.Time) int {
	flags := flag.NewFlagSet("member query", flag.ContinueOnError)
	flags.SetOutput(stderr)
	userIDs := flags.String("user-id", "", "comma-separated user IDs (required)")
	pageSize := flags.Int("page-size", 0, "page size")
	pageToken := flags.String("page-token", "", "page token")
	baseURL := flags.String("base-url", "", "API base URL")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: teambition_app member query --user-id IDS [flags]")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return exitUsage
	}
	if flags.NArg() != 0 || *userIDs == "" {
		flags.Usage()
		return exitUsage
	}

	query := make(url.Values)
	query.Set("userIds", *userIDs)
	if *pageSize > 0 {
		query.Set("pageSize", fmt.Sprint(*pageSize))
	}
	if *pageToken != "" {
		query.Set("pageToken", *pageToken)
	}

	cfg := loadConfig(*baseURL, "")
	if err := validateConfig(cfg, false); err != nil {
		fmt.Fprintln(stderr, err)
		return exitConfig
	}
	response, err := request(client, cfg, now(), http.MethodGet, "/v3/member/query", query, nil, "")
	if err != nil {
		return reportRequestError(stderr, err)
	}
	writeResponse(stdout, response)
	return 0
}

func runProjectPriorityQuery(args []string, stdout, stderr io.Writer, client *http.Client, now func() time.Time) int {
	flags := flag.NewFlagSet("project priority-query", flag.ContinueOnError)
	flags.SetOutput(stderr)
	baseURL := flags.String("base-url", "", "API base URL")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: teambition_app project priority-query [flags]")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return exitUsage
	}
	if flags.NArg() != 0 {
		flags.Usage()
		return exitUsage
	}

	cfg := loadConfig(*baseURL, "")
	if err := validateConfig(cfg, false); err != nil {
		fmt.Fprintln(stderr, err)
		return exitConfig
	}
	response, err := request(client, cfg, now(), http.MethodGet, "/v3/project/priority/list", nil, nil, "")
	if err != nil {
		return reportRequestError(stderr, err)
	}
	writeResponse(stdout, response)
	return 0
}

func runProjectTaskQuery(args []string, stdout, stderr io.Writer, client *http.Client, now func() time.Time) int {
	flags := flag.NewFlagSet("project task-query", flag.ContinueOnError)
	flags.SetOutput(stderr)
	projectID := flags.String("project-id", "", "project ID (required)")
	tql := flags.String("tql", "", "TQL query")
	pageSize := flags.Int("page-size", 0, "page size")
	pageToken := flags.String("page-token", "", "page token")
	baseURL := flags.String("base-url", "", "API base URL")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: teambition_app project task-query --project-id ID [flags]")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return exitUsage
	}
	if flags.NArg() != 0 || *projectID == "" {
		flags.Usage()
		return exitUsage
	}

	query := make(url.Values)
	if *tql != "" {
		query.Set("tql", *tql)
	}
	if *pageSize > 0 {
		query.Set("pageSize", fmt.Sprint(*pageSize))
	}
	if *pageToken != "" {
		query.Set("pageToken", *pageToken)
	}

	cfg := loadConfig(*baseURL, "")
	if err := validateConfig(cfg, false); err != nil {
		fmt.Fprintln(stderr, err)
		return exitConfig
	}
	path := "/v3/project/" + url.PathEscape(*projectID) + "/task/query"
	response, err := request(client, cfg, now(), http.MethodGet, path, query, nil, "")
	if err != nil {
		return reportRequestError(stderr, err)
	}
	writeResponse(stdout, response)
	return 0
}

func runSprintQuery(args []string, stdout, stderr io.Writer, client *http.Client, now func() time.Time) int {
	flags := flag.NewFlagSet("project sprint-query", flag.ContinueOnError)
	flags.SetOutput(stderr)
	projectID := flags.String("project-id", "", "project ID (required)")
	name := flags.String("name", "", "sprint name query")
	sprintIDs := flags.String("id", "", "comma-separated sprint IDs")
	pageSize := flags.Int("page-size", 0, "page size")
	pageToken := flags.String("page-token", "", "page token")
	baseURL := flags.String("base-url", "", "API base URL")
	dryRun := flags.Bool("dry-run", false, "print the request without sending it")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: teambition_app project sprint-query --project-id ID [flags]")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return exitUsage
	}
	if flags.NArg() != 0 || *projectID == "" {
		flags.Usage()
		return exitUsage
	}

	query := make(url.Values)
	if *name != "" {
		query.Set("q", *name)
	}
	if *sprintIDs != "" {
		query.Set("sprintIds", *sprintIDs)
	}
	if *pageSize > 0 {
		query.Set("pageSize", fmt.Sprint(*pageSize))
	}
	if *pageToken != "" {
		query.Set("pageToken", *pageToken)
	}

	cfg := loadConfig(*baseURL, "")
	path := "/v3/project/" + url.PathEscape(*projectID) + "/sprint/search"
	if *dryRun {
		return writeDryRun(stdout, stderr, cfg, http.MethodGet, path, query, nil)
	}
	if err := validateConfig(cfg, false); err != nil {
		fmt.Fprintln(stderr, err)
		return exitConfig
	}

	response, err := request(client, cfg, now(), http.MethodGet, path, query, nil, "")
	if err != nil {
		return reportRequestError(stderr, err)
	}
	writeResponse(stdout, response)
	return 0
}

func runTaskUpdate(args []string, stdout, stderr io.Writer, client *http.Client, now func() time.Time) int {
	flags := flag.NewFlagSet("task update", flag.ContinueOnError)
	flags.SetOutput(stderr)
	taskID := flags.String("task-id", "", "task ID (required)")
	sprintID := flags.String("sprint-id", "", "sprint ID; pass an empty value to remove the task from its sprint")
	operatorID := flags.String("operator-id", "", "operator user ID; defaults to TEAMBITION_OPERATOR_ID")
	baseURL := flags.String("base-url", "", "API base URL")
	dryRun := flags.Bool("dry-run", false, "print the request without sending it")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: teambition_app task update --task-id ID --sprint-id ID [flags]")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return exitUsage
	}
	sprintFlagSet := false
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "sprint-id" {
			sprintFlagSet = true
		}
	})
	if flags.NArg() != 0 || *taskID == "" || !sprintFlagSet {
		flags.Usage()
		return exitUsage
	}

	cfg := loadConfig(*baseURL, *operatorID)
	if cfg.operatorID == "" {
		fmt.Fprintln(stderr, "missing operator ID: pass --operator-id or set TEAMBITION_OPERATOR_ID")
		return exitConfig
	}
	path := "/v3/task/" + url.PathEscape(*taskID) + "/sprint"
	body := map[string]string{"sprintId": *sprintID}
	if *dryRun {
		return writeDryRun(stdout, stderr, cfg, http.MethodPut, path, nil, body)
	}
	if err := validateConfig(cfg, true); err != nil {
		fmt.Fprintln(stderr, err)
		return exitConfig
	}

	response, err := request(client, cfg, now(), http.MethodPut, path, nil, body, cfg.operatorID)
	if err != nil {
		return reportRequestError(stderr, err)
	}
	writeResponse(stdout, response)
	return 0
}

func loadConfig(baseURL, operatorID string) config {
	if baseURL == "" {
		baseURL = os.Getenv("TEAMBITION_API_BASE_URL")
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if operatorID == "" {
		operatorID = os.Getenv("TEAMBITION_OPERATOR_ID")
	}
	return config{
		appID:      os.Getenv("TEAMBITION_APP_ID"),
		appSecret:  os.Getenv("TEAMBITION_APP_SECRET"),
		orgID:      os.Getenv("TEAMBITION_ORG_ID"),
		operatorID: operatorID,
		baseURL:    strings.TrimRight(baseURL, "/"),
	}
}

func validateConfig(cfg config, requireOperator bool) error {
	missing := make([]string, 0, 4)
	if cfg.appID == "" {
		missing = append(missing, "TEAMBITION_APP_ID")
	}
	if cfg.appSecret == "" {
		missing = append(missing, "TEAMBITION_APP_SECRET")
	}
	if cfg.orgID == "" {
		missing = append(missing, "TEAMBITION_ORG_ID")
	}
	if requireOperator && cfg.operatorID == "" {
		missing = append(missing, "TEAMBITION_OPERATOR_ID")
	}
	if len(missing) != 0 {
		return fmt.Errorf("missing AppToken configuration: %s", strings.Join(missing, ", "))
	}
	if _, err := url.ParseRequestURI(cfg.baseURL); err != nil {
		return fmt.Errorf("invalid API base URL: %w", err)
	}
	return nil
}

func signAppToken(appID, appSecret string, now time.Time) (string, error) {
	if appID == "" || appSecret == "" {
		return "", errors.New("app ID and app secret are required")
	}
	const period int64 = 1800
	iat := now.Unix() / period * period
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	claims := struct {
		IssuedAt  int64  `json:"iat"`
		ExpiresAt int64  `json:"exp"`
		AppID     string `json:"_appId"`
	}{IssuedAt: iat, ExpiresAt: iat + 1980, AppID: appID}

	encodedHeader, err := encodeJWTPart(header)
	if err != nil {
		return "", err
	}
	encodedClaims, err := encodeJWTPart(claims)
	if err != nil {
		return "", err
	}
	unsigned := encodedHeader + "." + encodedClaims
	mac := hmac.New(sha256.New, []byte(appSecret))
	_, _ = mac.Write([]byte(unsigned))
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func encodeJWTPart(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func request(client *http.Client, cfg config, now time.Time, method, path string, query url.Values, body any, operatorID string) ([]byte, error) {
	token, err := signAppToken(cfg.appID, cfg.appSecret, now)
	if err != nil {
		return nil, err
	}
	endpoint, err := buildURL(cfg.baseURL, path, query)
	if err != nil {
		return nil, err
	}

	var requestBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		requestBody = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, endpoint, requestBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-Id", cfg.orgID)
	req.Header.Set("X-Tenant-Type", "organization")
	if operatorID != "" {
		req.Header.Set("X-Operator-Id", operatorID)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxResponseSize {
		return nil, errors.New("Teambition API response exceeds 16 MiB")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &apiError{statusCode: resp.StatusCode, body: strings.TrimSpace(string(data))}
	}
	if message := responseError(data); message != "" {
		return nil, &apiError{body: message}
	}
	return data, nil
}

func buildURL(baseURL, path string, query url.Values) (string, error) {
	endpoint, err := url.Parse(strings.TrimRight(baseURL, "/") + path)
	if err != nil {
		return "", err
	}
	if endpoint.Scheme != "http" && endpoint.Scheme != "https" {
		return "", errors.New("API base URL must use http or https")
	}
	if endpoint.Host == "" {
		return "", errors.New("API base URL must include a host")
	}
	endpoint.RawQuery = query.Encode()
	return endpoint.String(), nil
}

func responseError(data []byte) string {
	var value map[string]any
	if json.Unmarshal(data, &value) != nil {
		return ""
	}
	if success, ok := value["success"].(bool); ok && !success {
		return firstMessage(value)
	}
	for _, key := range []string{"error", "errorCode"} {
		if text := fmt.Sprint(value[key]); value[key] != nil && text != "" && text != "0" {
			return firstMessage(value)
		}
	}
	return ""
}

func firstMessage(value map[string]any) string {
	parts := make([]string, 0, 3)
	for _, key := range []string{"error", "errorCode", "errorMessage", "message"} {
		if raw, ok := value[key]; ok {
			text := strings.TrimSpace(fmt.Sprint(raw))
			if text != "" && text != "<nil>" {
				parts = append(parts, text)
			}
		}
	}
	if len(parts) == 0 {
		return "unknown API error"
	}
	return strings.Join(parts, ": ")
}

func writeDryRun(stdout, stderr io.Writer, cfg config, method, path string, query url.Values, body any) int {
	endpoint, err := buildURL(cfg.baseURL, path, query)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitConfig
	}
	headers := map[string]string{
		"X-Tenant-Id":   cfg.orgID,
		"X-Tenant-Type": "organization",
	}
	if cfg.operatorID != "" {
		headers["X-Operator-Id"] = cfg.operatorID
	}
	preview := map[string]any{
		"dryRun":  true,
		"method":  method,
		"url":     endpoint,
		"headers": headers,
	}
	if body != nil {
		preview["body"] = body
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(preview); err != nil {
		fmt.Fprintln(stderr, err)
		return exitRequest
	}
	return 0
}

func reportRequestError(stderr io.Writer, err error) int {
	fmt.Fprintln(stderr, err)
	var apiErr *apiError
	if errors.As(err, &apiErr) {
		return exitAPI
	}
	return exitRequest
}

func writeResponse(stdout io.Writer, response []byte) {
	_, _ = stdout.Write(response)
	if len(response) == 0 || response[len(response)-1] != '\n' {
		_, _ = fmt.Fprintln(stdout)
	}
}

func printUsage(output io.Writer) {
	fmt.Fprintln(output, `Direct Teambition OpenAPI client for AppToken-only sprint operations.

Usage:
  teambition_app project sprint-query --project-id ID [--name NAME] [--id IDS]
  teambition_app project task-query --project-id ID [--tql TQL]
  teambition_app project priority-query
  teambition_app member query --user-id IDS
  teambition_app task update --task-id ID --sprint-id ID --operator-id USER_ID [--dry-run]

Environment:
  TEAMBITION_APP_ID
  TEAMBITION_APP_SECRET
  TEAMBITION_ORG_ID
  TEAMBITION_OPERATOR_ID       required for writes unless --operator-id is used
  TEAMBITION_API_BASE_URL      optional; defaults to https://open.teambition.com/api`)
}
