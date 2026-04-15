package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type verificationReport struct {
	TimestampUTC         string   `json:"timestamp_utc"`
	BaseURL              string   `json:"base_url"`
	HealthStatusCode     int      `json:"health_status_code"`
	SyncHealthStatusCode int      `json:"sync_health_status_code"`
	SyncEnabled          bool     `json:"sync_enabled"`
	WebSocketPath        string   `json:"websocket_path"`
	SyncHubHistoryDepth  int      `json:"sync_hub_history_depth,omitempty"`
	SyncHubPublished     int64    `json:"sync_hub_published_total,omitempty"`
	SyncHubDropped       int64    `json:"sync_hub_dropped_total,omitempty"`
	SyncHubLastSequence  int64    `json:"sync_hub_last_sequence,omitempty"`
	WebSocketConnected   bool     `json:"websocket_connected"`
	FirstEventType       string   `json:"first_event_type,omitempty"`
	Errors               []string `json:"errors,omitempty"`
}

type syncHubHealth struct {
	ConnectedClients int   `json:"connected_clients"`
	PublishedTotal   int64 `json:"published_total"`
	DroppedTotal     int64 `json:"dropped_total"`
	LastSequence     int64 `json:"last_sequence"`
	HistoryDepth     int   `json:"history_depth"`
}

type syncHealthEnvelope struct {
	Status           string        `json:"status"`
	SyncEnabled      bool          `json:"sync_enabled"`
	WebSocketPath    string        `json:"websocket_path"`
	ConnectedClients int           `json:"connected_clients"`
	Hub              syncHubHealth `json:"hub"`
}

func main() {
	baseURL := flag.String("base-url", "http://127.0.0.1:8080", "Base URL for the sync runtime")
	webSocketPath := flag.String("websocket-path", "/api/v1/sync/ws", "Sync websocket path")
	timeoutSeconds := flag.Int("timeout-seconds", 10, "Timeout per request in seconds")
	reportFile := flag.String("report-file", "", "Optional output file path for JSON verification report")
	bearerToken := flag.String("bearer-token", strings.TrimSpace(os.Getenv("SS_SYNC_RUNTIME_BEARER_TOKEN")), "Optional bearer token for protected websocket runtime")
	flag.Parse()

	report := verificationReport{
		TimestampUTC: time.Now().UTC().Format(time.RFC3339),
		BaseURL:      strings.TrimRight(strings.TrimSpace(*baseURL), "/"),
	}

	timeout := time.Duration(*timeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	client := &http.Client{Timeout: timeout}

	report.HealthStatusCode = verifyHTTPStatus(client, report.BaseURL+"/health", http.StatusOK, &report.Errors)
	report.SyncHealthStatusCode = verifyHTTPStatus(client, report.BaseURL+"/api/v1/sync/health", http.StatusOK, &report.Errors)

	syncHealth := fetchSyncHealth(client, report.BaseURL+"/api/v1/sync/health", &report.Errors)
	report.SyncEnabled = syncHealth.SyncEnabled
	report.SyncHubHistoryDepth = syncHealth.Hub.HistoryDepth
	report.SyncHubPublished = syncHealth.Hub.PublishedTotal
	report.SyncHubDropped = syncHealth.Hub.DroppedTotal
	report.SyncHubLastSequence = syncHealth.Hub.LastSequence
	resolvedWebSocketPath := strings.TrimSpace(*webSocketPath)
	if strings.TrimSpace(syncHealth.WebSocketPath) != "" {
		resolvedWebSocketPath = strings.TrimSpace(syncHealth.WebSocketPath)
	}
	if !strings.HasPrefix(resolvedWebSocketPath, "/") {
		resolvedWebSocketPath = "/" + resolvedWebSocketPath
	}
	report.WebSocketPath = resolvedWebSocketPath

	if report.SyncEnabled {
		connected, eventType, err := verifyWebSocket(report.BaseURL, resolvedWebSocketPath, strings.TrimSpace(*bearerToken), timeout)
		report.WebSocketConnected = connected
		report.FirstEventType = eventType
		if err != nil {
			report.Errors = append(report.Errors, err.Error())
		}
	}

	if strings.TrimSpace(*reportFile) != "" {
		if err := writeReport(*reportFile, report); err != nil {
			report.Errors = append(report.Errors, err.Error())
		}
	}

	output, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println(string(output))

	if len(report.Errors) > 0 {
		os.Exit(1)
	}
}

func verifyHTTPStatus(client *http.Client, url string, expected int, errorsList *[]string) int {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		*errorsList = append(*errorsList, fmt.Sprintf("create request %s: %v", url, err))
		return 0
	}

	response, err := client.Do(request)
	if err != nil {
		*errorsList = append(*errorsList, fmt.Sprintf("request %s failed: %v", url, err))
		return 0
	}
	defer response.Body.Close()

	if response.StatusCode != expected {
		*errorsList = append(*errorsList, fmt.Sprintf("expected %s status %d, got %d", url, expected, response.StatusCode))
	}

	return response.StatusCode
}

func fetchSyncHealth(client *http.Client, url string, errorsList *[]string) syncHealthEnvelope {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		*errorsList = append(*errorsList, fmt.Sprintf("create sync health request: %v", err))
		return syncHealthEnvelope{}
	}

	response, err := client.Do(request)
	if err != nil {
		*errorsList = append(*errorsList, fmt.Sprintf("sync health request failed: %v", err))
		return syncHealthEnvelope{}
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return syncHealthEnvelope{}
	}

	var envelope syncHealthEnvelope
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		*errorsList = append(*errorsList, fmt.Sprintf("decode sync health response failed: %v", err))
		return syncHealthEnvelope{}
	}

	return envelope
}

func verifyWebSocket(baseURL, path, bearerToken string, timeout time.Duration) (bool, string, error) {
	wsURL := strings.TrimRight(baseURL, "/")
	wsURL = "ws" + strings.TrimPrefix(wsURL, "http") + path

	headers := http.Header{}
	if strings.TrimSpace(bearerToken) != "" {
		headers.Set("Authorization", "Bearer "+strings.TrimSpace(bearerToken))
	}

	dialer := websocket.Dialer{HandshakeTimeout: timeout}
	conn, response, err := dialer.Dial(wsURL, headers)
	if err != nil {
		if response != nil {
			return false, "", fmt.Errorf("websocket dial failed with status %d at %s: %w", response.StatusCode, wsURL, err)
		}
		return false, "", fmt.Errorf("websocket dial failed at %s: %w", wsURL, err)
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	var event struct {
		Type string `json:"type"`
	}
	if err := conn.ReadJSON(&event); err != nil {
		return false, "", fmt.Errorf("read websocket event failed: %w", err)
	}

	if strings.TrimSpace(event.Type) == "" {
		return false, "", fmt.Errorf("websocket connected but first event type was empty")
	}

	if event.Type != "sync.connected" {
		return true, event.Type, fmt.Errorf("expected first websocket event sync.connected, got %s", event.Type)
	}

	return true, event.Type, nil
}

func writeReport(filePath string, report verificationReport) error {
	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal verification report: %w", err)
	}

	if err := os.WriteFile(filePath, payload, 0o644); err != nil {
		return fmt.Errorf("write verification report: %w", err)
	}
	return nil
}
