package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
)

const (
	outputSchemaVersion    = 1
	supportedServerVersion = "0.0.40"
	maxResponseBytes       = 4 << 20
	maxTokenBytes          = 16 << 10
	requestTimeout         = 5 * time.Second
)

type output struct {
	SchemaVersion int          `json:"schemaVersion"`
	ObservedAt    string       `json:"observedAt"`
	State         string       `json:"state"`
	Environment   *environment `json:"environment"`
	Snapshot      *snapshot    `json:"snapshot"`
	Error         *outputError `json:"error"`
}

type outputError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type environment struct {
	ID            string `json:"id"`
	Label         string `json:"label"`
	ServerVersion string `json:"serverVersion"`
}

type snapshot struct {
	Sequence  int64     `json:"sequence"`
	UpdatedAt string    `json:"updatedAt"`
	Projects  []project `json:"projects"`
	Threads   []thread  `json:"threads"`
}

type project struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type thread struct {
	ID                        string          `json:"id"`
	ProjectID                 string          `json:"projectId"`
	Title                     string          `json:"title"`
	ProviderInstanceID        string          `json:"providerInstanceId"`
	CreatedAt                 string          `json:"createdAt"`
	UpdatedAt                 string          `json:"updatedAt"`
	ArchivedAt                *string         `json:"archivedAt"`
	SettledAt                 *string         `json:"settledAt"`
	LatestUserMessageAt       *string         `json:"latestUserMessageAt"`
	Session                   *sessionSummary `json:"session"`
	LatestTurn                *turnSummary    `json:"latestTurn"`
	HasPendingApprovals       bool            `json:"hasPendingApprovals"`
	HasPendingUserInput       bool            `json:"hasPendingUserInput"`
	HasActionableProposedPlan bool            `json:"hasActionableProposedPlan"`
	BackgroundLiveness        *string         `json:"backgroundLiveness"`
}

type sessionSummary struct {
	Status       string `json:"status"`
	ProviderName string `json:"providerName"`
	UpdatedAt    string `json:"updatedAt"`
}

type turnSummary struct {
	ID          string  `json:"id"`
	State       string  `json:"state"`
	RequestedAt string  `json:"requestedAt"`
	StartedAt   *string `json:"startedAt"`
	CompletedAt *string `json:"completedAt"`
}

type rawDescriptor struct {
	EnvironmentID string `json:"environmentId"`
	Label         string `json:"label"`
	ServerVersion string `json:"serverVersion"`
}

type rawSnapshot struct {
	Sequence  *int64        `json:"snapshotSequence"`
	Projects  *[]rawProject `json:"projects"`
	Threads   *[]rawThread  `json:"threads"`
	UpdatedAt string        `json:"updatedAt"`
}

type rawProject struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type rawThread struct {
	ID                        string             `json:"id"`
	ProjectID                 string             `json:"projectId"`
	Title                     string             `json:"title"`
	ModelSelection            *rawModelSelection `json:"modelSelection"`
	CreatedAt                 string             `json:"createdAt"`
	UpdatedAt                 string             `json:"updatedAt"`
	ArchivedAt                nullableString     `json:"archivedAt"`
	SettledAt                 nullableString     `json:"settledAt"`
	LatestUserMessageAt       nullableString     `json:"latestUserMessageAt"`
	Session                   nullableSession    `json:"session"`
	LatestTurn                nullableTurn       `json:"latestTurn"`
	HasPendingApprovals       *bool              `json:"hasPendingApprovals"`
	HasPendingUserInput       *bool              `json:"hasPendingUserInput"`
	HasActionableProposedPlan *bool              `json:"hasActionableProposedPlan"`
	BackgroundLiveness        nullableString     `json:"backgroundLiveness"`
}

type rawSession struct {
	ThreadID     string         `json:"threadId"`
	Status       string         `json:"status"`
	ProviderName nullableString `json:"providerName"`
	UpdatedAt    string         `json:"updatedAt"`
}

type rawModelSelection struct {
	InstanceID string `json:"instanceId"`
}

type rawTurn struct {
	TurnID      string         `json:"turnId"`
	State       string         `json:"state"`
	RequestedAt string         `json:"requestedAt"`
	StartedAt   nullableString `json:"startedAt"`
	CompletedAt nullableString `json:"completedAt"`
}

type nullableString struct {
	set   bool
	value *string
}

func (n *nullableString) UnmarshalJSON(data []byte) error {
	n.set = true
	if string(data) == "null" {
		n.value = nil
		return nil
	}
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	n.value = &value
	return nil
}

type nullableSession struct {
	set   bool
	value *rawSession
}

func (n *nullableSession) UnmarshalJSON(data []byte) error {
	n.set = true
	if string(data) == "null" {
		n.value = nil
		return nil
	}
	var value rawSession
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	n.value = &value
	return nil
}

type nullableTurn struct {
	set   bool
	value *rawTurn
}

func (n *nullableTurn) UnmarshalJSON(data []byte) error {
	n.set = true
	if string(data) == "null" {
		n.value = nil
		return nil
	}
	var value rawTurn
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	n.value = &value
	return nil
}

type collectorError struct {
	code    string
	message string
}

func (e *collectorError) Error() string { return e.code }

func main() {
	result, exitCode := run(os.Args[1:], time.Now)
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(true)
	if err := encoder.Encode(result); err != nil {
		os.Exit(1)
	}
	os.Exit(exitCode)
}

func run(args []string, now func() time.Time) (result output, exitCode int) {
	result = output{
		SchemaVersion: outputSchemaVersion,
		State:         "error",
	}
	defer func() {
		result.ObservedAt = now().UTC().Format(time.RFC3339Nano)
	}()

	endpoint, tokenFile, err := parseArgs(args)
	if err != nil {
		result.Error = publicError(err)
		return result, 2
	}

	token, err := readToken(tokenFile)
	if err != nil {
		result.Error = publicError(err)
		return result, 1
	}

	client := newHTTPClient(endpoint)
	descriptor, err := fetchDescriptor(client, endpoint)
	if descriptor != nil {
		result.Environment = &environment{
			ID:            descriptor.EnvironmentID,
			Label:         descriptor.Label,
			ServerVersion: descriptor.ServerVersion,
		}
	}
	if err != nil {
		result.Error = publicError(err)
		return result, 1
	}
	if descriptor.ServerVersion != supportedServerVersion {
		result.Error = publicError(failure("incompatible_server", "The T3 Code server version is not supported."))
		return result, 1
	}

	snapshot, err := fetchSnapshot(client, endpoint, token)
	if err != nil {
		result.Error = publicError(err)
		return result, 1
	}

	result.State = "connected"
	result.Snapshot = snapshot
	return result, 0
}

func parseArgs(args []string) (*url.URL, string, error) {
	if len(args) == 0 || args[0] != "snapshot" {
		return nil, "", failure("invalid_arguments", "Usage: dankt3code snapshot --endpoint <origin> --token-file <file>.")
	}

	flags := flag.NewFlagSet("snapshot", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	endpointValue := flags.String("endpoint", "", "")
	tokenFile := flags.String("token-file", "", "")
	if err := flags.Parse(args[1:]); err != nil || flags.NArg() != 0 || *endpointValue == "" || *tokenFile == "" {
		return nil, "", failure("invalid_arguments", "Usage: dankt3code snapshot --endpoint <origin> --token-file <file>.")
	}

	endpoint, err := validateEndpoint(*endpointValue)
	if err != nil {
		return nil, "", err
	}
	return endpoint, *tokenFile, nil
}

func validateEndpoint(value string) (*url.URL, error) {
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, failure("invalid_arguments", "The endpoint must be an HTTP or HTTPS loopback origin.")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") || parsed.RawPath != "" || parsed.Opaque != "" {
		return nil, failure("invalid_arguments", "The endpoint must not contain credentials, a path, query, or fragment.")
	}
	if parsed.Hostname() == "" {
		return nil, failure("invalid_arguments", "The endpoint must be an HTTP or HTTPS loopback origin.")
	}
	if port := parsed.Port(); port != "" {
		portNumber, err := strconv.Atoi(port)
		if err != nil || portNumber < 1 || portNumber > 65535 {
			return nil, failure("invalid_arguments", "The endpoint port must be numeric and valid.")
		}
	} else if strings.HasSuffix(parsed.Host, ":") {
		return nil, failure("invalid_arguments", "The endpoint port must be numeric and valid.")
	}

	hostname := strings.ToLower(parsed.Hostname())
	if hostname != "localhost" {
		ip := net.ParseIP(hostname)
		if ip == nil || !ip.IsLoopback() {
			return nil, failure("invalid_arguments", "The endpoint must use localhost or a numeric loopback address.")
		}
	}
	parsed.Path = ""
	return parsed, nil
}

func readToken(path string) (string, error) {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return "", failure("token_unavailable", "The bearer token is unavailable.")
	}
	file := os.NewFile(uintptr(fd), "token")
	if file == nil {
		syscall.Close(fd)
		return "", failure("token_unavailable", "The bearer token is unavailable.")
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		return "", failure("token_unavailable", "The bearer token must be an owner-only regular file.")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Geteuid()) {
		return "", failure("token_unavailable", "The bearer token must be owned by the current user.")
	}

	data, err := io.ReadAll(io.LimitReader(file, maxTokenBytes+1))
	if err != nil || len(data) > maxTokenBytes {
		return "", failure("token_unavailable", "The bearer token is unavailable.")
	}
	token := strings.TrimSpace(string(data))
	if token == "" {
		return "", failure("token_unavailable", "The bearer token is unavailable.")
	}
	for i := 0; i < len(token); i++ {
		if token[i] < 0x21 || token[i] > 0x7e {
			return "", failure("token_unavailable", "The bearer token is malformed.")
		}
	}
	return token, nil
}

func newHTTPClient(endpoint *url.URL) *http.Client {
	hostname := strings.ToLower(endpoint.Hostname())
	dialer := &net.Dialer{Timeout: 2 * time.Second, KeepAlive: 15 * time.Second}
	transport := &http.Transport{
		Proxy:                 nil,
		ForceAttemptHTTP2:     true,
		DisableKeepAlives:     true,
		TLSHandshakeTimeout:   2 * time.Second,
		ResponseHeaderTimeout: 3 * time.Second,
	}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		_, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}

		if hostname != "localhost" {
			return dialer.DialContext(ctx, network, net.JoinHostPort(endpoint.Hostname(), port))
		}

		addresses, err := net.DefaultResolver.LookupIPAddr(ctx, "localhost")
		if err != nil || len(addresses) == 0 {
			return nil, fmt.Errorf("localhost resolution failed")
		}
		for _, address := range addresses {
			if !address.IP.IsLoopback() {
				return nil, fmt.Errorf("localhost resolved outside loopback")
			}
		}
		var lastErr error
		for _, address := range addresses {
			connection, err := dialer.DialContext(ctx, network, net.JoinHostPort(address.IP.String(), port))
			if err == nil {
				return connection, nil
			}
			lastErr = err
		}
		return nil, lastErr
	}
	return &http.Client{
		Transport: transport,
		Timeout:   requestTimeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func fetchDescriptor(client *http.Client, endpoint *url.URL) (*rawDescriptor, error) {
	body, status, err := get(client, endpoint, "/.well-known/t3/environment", "")
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		// Discovery is unauthenticated; a failure here cannot diagnose the bearer.
		return nil, failure("unexpected_response", "The T3 Code environment descriptor returned an unexpected response.")
	}

	var descriptor rawDescriptor
	if err := decodeOne(body, &descriptor); err != nil || !validText(descriptor.EnvironmentID, 1024) || !validText(descriptor.Label, 4096) || !validText(descriptor.ServerVersion, 128) {
		return nil, failure("malformed_descriptor", "The T3 Code environment descriptor is malformed.")
	}
	return &descriptor, nil
}

func fetchSnapshot(client *http.Client, endpoint *url.URL, token string) (*snapshot, error) {
	body, status, err := get(client, endpoint, "/api/orchestration/shell", token)
	if err != nil {
		return nil, err
	}
	if err := statusError(status); err != nil {
		return nil, err
	}

	var raw rawSnapshot
	if err := decodeOne(body, &raw); err != nil {
		return nil, failure("malformed_snapshot", "The T3 Code activity snapshot is malformed.")
	}
	return minimizeSnapshot(&raw)
}

func get(client *http.Client, endpoint *url.URL, path, token string) ([]byte, int, error) {
	requestURL := *endpoint
	requestURL.Path = path
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return nil, 0, failure("server_unavailable", "The T3 Code server is unavailable.")
	}
	request.Header.Set("Accept", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, 0, failure("server_unavailable", "The T3 Code server is unavailable.")
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return nil, 0, failure("server_unavailable", "The T3 Code server response could not be read.")
	}
	if len(body) > maxResponseBytes {
		return nil, 0, failure("response_too_large", "The T3 Code server response exceeded the size limit.")
	}
	return body, response.StatusCode, nil
}

func statusError(status int) error {
	switch status {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized:
		return failure("authentication_failed", "The T3 Code bearer token was not accepted.")
	case http.StatusForbidden:
		return failure("authorization_failed", "The T3 Code bearer token lacks the required read scope.")
	default:
		return failure("unexpected_response", "The T3 Code server returned an unexpected response.")
	}
}

func decodeOne(data []byte, target any) error {
	if !utf8.Valid(data) {
		return fmt.Errorf("invalid UTF-8")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("trailing JSON value")
	}
	return nil
}

func minimizeSnapshot(raw *rawSnapshot) (*snapshot, error) {
	if raw.Sequence == nil || *raw.Sequence < 0 || raw.Projects == nil || raw.Threads == nil {
		return nil, failure("malformed_snapshot", "The T3 Code activity snapshot is missing required fields.")
	}
	updatedAt, ok := validTimestamp(raw.UpdatedAt)
	if !ok {
		return nil, failure("malformed_snapshot", "The T3 Code activity snapshot contains an invalid timestamp.")
	}

	result := &snapshot{
		Sequence:  *raw.Sequence,
		UpdatedAt: updatedAt,
		Projects:  make([]project, 0, len(*raw.Projects)),
		Threads:   make([]thread, 0, len(*raw.Threads)),
	}
	projectIDs := make(map[string]bool, len(*raw.Projects))
	for _, item := range *raw.Projects {
		createdAt, createdOK := validTimestamp(item.CreatedAt)
		itemUpdatedAt, updatedOK := validTimestamp(item.UpdatedAt)
		if !validText(item.ID, 1024) || !validText(item.Title, 4096) || !createdOK || !updatedOK || projectIDs[item.ID] {
			return nil, failure("malformed_snapshot", "The T3 Code activity snapshot contains an invalid project.")
		}
		projectIDs[item.ID] = true
		result.Projects = append(result.Projects, project{ID: item.ID, Title: item.Title, CreatedAt: createdAt, UpdatedAt: itemUpdatedAt})
	}

	threadIDs := make(map[string]bool, len(*raw.Threads))
	for _, item := range *raw.Threads {
		minimized, err := minimizeThread(item, projectIDs)
		if err != nil || threadIDs[item.ID] {
			return nil, failure("malformed_snapshot", "The T3 Code activity snapshot contains an invalid thread.")
		}
		threadIDs[item.ID] = true
		result.Threads = append(result.Threads, minimized)
	}
	return result, nil
}

func minimizeThread(raw rawThread, projectIDs map[string]bool) (thread, error) {
	if !validText(raw.ID, 1024) || !validText(raw.ProjectID, 1024) || !projectIDs[raw.ProjectID] || !validText(raw.Title, 4096) || raw.ModelSelection == nil || !validProviderInstanceID(raw.ModelSelection.InstanceID) || raw.HasPendingApprovals == nil || raw.HasPendingUserInput == nil || raw.HasActionableProposedPlan == nil || !raw.LatestUserMessageAt.set || !raw.Session.set || !raw.LatestTurn.set {
		return thread{}, failure("malformed_snapshot", "invalid thread")
	}
	createdAt, createdOK := validTimestamp(raw.CreatedAt)
	updatedAt, updatedOK := validTimestamp(raw.UpdatedAt)
	latestUserMessageAt, latestUserOK := optionalTimestamp(raw.LatestUserMessageAt)
	archivedAt, archivedOK := optionalDefaultTimestamp(raw.ArchivedAt)
	settledAt, settledOK := optionalDefaultTimestamp(raw.SettledAt)
	if !createdOK || !updatedOK || !latestUserOK || !archivedOK || !settledOK {
		return thread{}, failure("malformed_snapshot", "invalid thread timestamp")
	}

	result := thread{
		ID:                        raw.ID,
		ProjectID:                 raw.ProjectID,
		Title:                     raw.Title,
		ProviderInstanceID:        raw.ModelSelection.InstanceID,
		CreatedAt:                 createdAt,
		UpdatedAt:                 updatedAt,
		ArchivedAt:                archivedAt,
		SettledAt:                 settledAt,
		LatestUserMessageAt:       latestUserMessageAt,
		HasPendingApprovals:       *raw.HasPendingApprovals,
		HasPendingUserInput:       *raw.HasPendingUserInput,
		HasActionableProposedPlan: *raw.HasActionableProposedPlan,
	}

	if raw.BackgroundLiveness.value != nil {
		value := normalizeStatus(*raw.BackgroundLiveness.value, []string{"working", "monitoring"})
		result.BackgroundLiveness = &value
	}
	if raw.Session.value != nil {
		if raw.Session.value.ThreadID != raw.ID || !validText(raw.Session.value.Status, 128) || !raw.Session.value.ProviderName.set {
			return thread{}, failure("malformed_snapshot", "invalid session")
		}
		providerName := "unknown"
		if raw.Session.value.ProviderName.value != nil {
			if !validText(*raw.Session.value.ProviderName.value, 128) {
				return thread{}, failure("malformed_snapshot", "invalid session provider")
			}
			providerName = *raw.Session.value.ProviderName.value
		}
		sessionUpdatedAt, ok := validTimestamp(raw.Session.value.UpdatedAt)
		if !ok {
			return thread{}, failure("malformed_snapshot", "invalid session timestamp")
		}
		result.Session = &sessionSummary{
			Status:       normalizeStatus(raw.Session.value.Status, []string{"idle", "starting", "running", "ready", "interrupted", "stopped", "error"}),
			ProviderName: providerName,
			UpdatedAt:    sessionUpdatedAt,
		}
	}
	if raw.LatestTurn.value != nil {
		turn := raw.LatestTurn.value
		if !validText(turn.TurnID, 1024) || !validText(turn.State, 128) || !turn.StartedAt.set || !turn.CompletedAt.set {
			return thread{}, failure("malformed_snapshot", "invalid latest turn")
		}
		requestedAt, requestedOK := validTimestamp(turn.RequestedAt)
		startedAt, startedOK := optionalTimestamp(turn.StartedAt)
		completedAt, completedOK := optionalTimestamp(turn.CompletedAt)
		if !requestedOK || !startedOK || !completedOK {
			return thread{}, failure("malformed_snapshot", "invalid latest turn timestamp")
		}
		result.LatestTurn = &turnSummary{
			ID:          turn.TurnID,
			State:       normalizeStatus(turn.State, []string{"running", "interrupted", "completed", "error"}),
			RequestedAt: requestedAt,
			StartedAt:   startedAt,
			CompletedAt: completedAt,
		}
	}
	return result, nil
}

func optionalDefaultTimestamp(value nullableString) (*string, bool) {
	if !value.set {
		return nil, true
	}
	return optionalTimestamp(value)
}

func optionalTimestamp(value nullableString) (*string, bool) {
	if value.value == nil {
		return nil, true
	}
	parsed, ok := validTimestamp(*value.value)
	if !ok {
		return nil, false
	}
	return &parsed, true
}

func validTimestamp(value string) (string, bool) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return "", false
	}
	return parsed.UTC().Format(time.RFC3339Nano), true
}

func validText(value string, max int) bool {
	return value != "" && value == strings.TrimSpace(value) && len(value) <= max
}

func validProviderInstanceID(value string) bool {
	if len(value) == 0 || len(value) > 64 || !isASCIIAlpha(value[0]) {
		return false
	}
	for i := 1; i < len(value); i++ {
		if !isASCIIAlpha(value[i]) && (value[i] < '0' || value[i] > '9') && value[i] != '-' && value[i] != '_' {
			return false
		}
	}
	return true
}

func isASCIIAlpha(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z'
}

func normalizeStatus(value string, known []string) string {
	for _, candidate := range known {
		if value == candidate {
			return value
		}
	}
	return "unknown"
}

func failure(code, message string) error {
	return &collectorError{code: code, message: message}
}

func publicError(err error) *outputError {
	if known, ok := err.(*collectorError); ok {
		return &outputError{Code: known.code, Message: known.message}
	}
	return &outputError{Code: "internal_error", Message: "The snapshot could not be collected."}
}
