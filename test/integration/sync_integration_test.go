package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"selfsystems/internal/ai"
	authapi "selfsystems/internal/auth"
	"selfsystems/internal/config"
	httpapi "selfsystems/internal/http"
	sqliterepo "selfsystems/internal/repository/sqlite"
	"selfsystems/internal/service"
	syncapi "selfsystems/internal/sync"
)

func TestSyncWebSocketDeliveryIntegration(t *testing.T) {
	server, _, jwtService, cfg := newSyncIntegrationServer(t, true)
	token := issueIntegrationToken(t, jwtService)

	conn, _, err := dialSyncWebSocket(server.URL, cfg.Sync.WebSocketPath, token)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	connected := readSyncEvent(t, conn)
	if connected.Type != "sync.connected" {
		t.Fatalf("expected sync.connected event, got %q", connected.Type)
	}

	response := publishSyncEvent(t, server.URL, token, "sync.update", map[string]any{"id": "res-1"})
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 from sync publish, got %d", response.StatusCode)
	}

	event := readSyncEvent(t, conn)
	if event.Type != "sync.update" {
		t.Fatalf("expected sync.update event, got %q", event.Type)
	}

	payload, ok := event.Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected map payload, got %T", event.Payload)
	}
	if payload["id"] != "res-1" {
		t.Fatalf("expected payload id res-1, got %v", payload["id"])
	}
}

func TestSyncReconnectBehaviorIntegration(t *testing.T) {
	server, hub, jwtService, cfg := newSyncIntegrationServer(t, true)
	token := issueIntegrationToken(t, jwtService)

	firstConn, _, err := dialSyncWebSocket(server.URL, cfg.Sync.WebSocketPath, token)
	if err != nil {
		t.Fatalf("dial first websocket: %v", err)
	}
	_ = readSyncEvent(t, firstConn)
	_ = firstConn.Close()

	waitForCondition(t, 2*time.Second, func() bool {
		return hub.ClientCount() == 0
	}, "expected first websocket client to disconnect")

	secondConn, _, err := dialSyncWebSocket(server.URL, cfg.Sync.WebSocketPath, token)
	if err != nil {
		t.Fatalf("dial second websocket: %v", err)
	}
	defer secondConn.Close()

	connected := readSyncEvent(t, secondConn)
	if connected.Type != "sync.connected" {
		t.Fatalf("expected sync.connected event after reconnect, got %q", connected.Type)
	}

	waitForCondition(t, 2*time.Second, func() bool {
		return hub.ClientCount() == 1
	}, "expected one connected websocket client after reconnect")

	response := publishSyncEvent(t, server.URL, token, "sync.reconnected", map[string]any{"status": "ok"})
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 from sync publish after reconnect, got %d", response.StatusCode)
	}

	event := readSyncEvent(t, secondConn)
	if event.Type != "sync.reconnected" {
		t.Fatalf("expected sync.reconnected event, got %q", event.Type)
	}
}

func TestSyncReconnectReplaySinceSequenceIntegration(t *testing.T) {
	server, hub, jwtService, cfg := newSyncIntegrationServer(t, true)
	token := issueIntegrationToken(t, jwtService)

	firstConn, _, err := dialSyncWebSocket(server.URL, cfg.Sync.WebSocketPath, token)
	if err != nil {
		t.Fatalf("dial first websocket: %v", err)
	}

	_ = readSyncEvent(t, firstConn)

	baselineResponse := publishSyncEvent(t, server.URL, token, syncapi.EventTypeResourceUpdated, map[string]any{
		"entity_id": "res-reconnect-replay",
		"title":     "v1",
	})
	if baselineResponse.StatusCode != http.StatusAccepted {
		defer baselineResponse.Body.Close()
		t.Fatalf("expected baseline publish status 202, got %d", baselineResponse.StatusCode)
	}
	_ = baselineResponse.Body.Close()

	baselineEvent := readSyncEventByType(t, firstConn, syncapi.EventTypeResourceUpdated)
	if baselineEvent.Sequence <= 0 {
		t.Fatalf("expected baseline event sequence > 0, got %d", baselineEvent.Sequence)
	}

	_ = firstConn.Close()
	waitForCondition(t, 2*time.Second, func() bool {
		return hub.ClientCount() == 0
	}, "expected websocket disconnect before replay reconnect")

	for _, title := range []string{"v2", "v3"} {
		response := publishSyncEvent(t, server.URL, token, syncapi.EventTypeResourceUpdated, map[string]any{
			"entity_id": "res-reconnect-replay",
			"title":     title,
		})
		if response.StatusCode != http.StatusAccepted {
			defer response.Body.Close()
			t.Fatalf("expected replay publish status 202 for %s, got %d", title, response.StatusCode)
		}
		_ = response.Body.Close()
	}

	reconnectPath := fmt.Sprintf("%s?since_sequence=%d&replay_limit=50", cfg.Sync.WebSocketPath, baselineEvent.Sequence)
	secondConn, _, err := dialSyncWebSocket(server.URL, reconnectPath, token)
	if err != nil {
		t.Fatalf("dial second websocket with replay query: %v", err)
	}
	defer secondConn.Close()

	reconnected := readSyncEvent(t, secondConn)
	if reconnected.Type != syncapi.EventTypeReconnected {
		t.Fatalf("expected %q event, got %q", syncapi.EventTypeReconnected, reconnected.Type)
	}
	reconnectedPayload, ok := reconnected.Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected reconnected payload map, got %T", reconnected.Payload)
	}
	replayedCount, ok := reconnectedPayload["replayed_count"].(float64)
	if !ok {
		t.Fatalf("expected replayed_count in reconnect payload, got %T", reconnectedPayload["replayed_count"])
	}
	if int(replayedCount) != 2 {
		t.Fatalf("expected replayed_count 2, got %d", int(replayedCount))
	}

	firstReplay := readSyncEventByType(t, secondConn, syncapi.EventTypeResourceUpdated)
	secondReplay := readSyncEventByType(t, secondConn, syncapi.EventTypeResourceUpdated)

	if firstReplay.Sequence != baselineEvent.Sequence+1 {
		t.Fatalf("expected first replay sequence %d, got %d", baselineEvent.Sequence+1, firstReplay.Sequence)
	}
	if secondReplay.Sequence != baselineEvent.Sequence+2 {
		t.Fatalf("expected second replay sequence %d, got %d", baselineEvent.Sequence+2, secondReplay.Sequence)
	}

	firstPayload, ok := firstReplay.Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected first replay payload map, got %T", firstReplay.Payload)
	}
	secondPayload, ok := secondReplay.Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected second replay payload map, got %T", secondReplay.Payload)
	}
	if firstPayload["title"] != "v2" {
		t.Fatalf("expected first replay title v2, got %v", firstPayload["title"])
	}
	if secondPayload["title"] != "v3" {
		t.Fatalf("expected second replay title v3, got %v", secondPayload["title"])
	}

	connected := readSyncEventByType(t, secondConn, syncapi.EventTypeConnected)
	if connected.Type != syncapi.EventTypeConnected {
		t.Fatalf("expected %q event after replay, got %q", syncapi.EventTypeConnected, connected.Type)
	}
}

func TestSyncWebSocketBurstSequenceStabilityIntegration(t *testing.T) {
	server, _, jwtService, cfg := newSyncIntegrationServer(t, true)
	token := issueIntegrationToken(t, jwtService)

	conn, _, err := dialSyncWebSocket(server.URL, cfg.Sync.WebSocketPath, token)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	_ = readSyncEvent(t, conn)

	const burstSize = 12
	expectedEntityIDs := make([]string, 0, burstSize)
	for i := 0; i < burstSize; i++ {
		entityID := fmt.Sprintf("res-burst-%02d", i)
		expectedEntityIDs = append(expectedEntityIDs, entityID)

		response := publishSyncEvent(t, server.URL, token, syncapi.EventTypeResourceUpdated, map[string]any{
			"entity_id": entityID,
			"title":     fmt.Sprintf("burst-%02d", i),
		})
		if response.StatusCode != http.StatusAccepted {
			defer response.Body.Close()
			t.Fatalf("expected burst publish status 202 for %s, got %d", entityID, response.StatusCode)
		}
		_ = response.Body.Close()
	}

	lastSequence := int64(0)
	for i := 0; i < burstSize; i++ {
		event := readSyncEventByType(t, conn, syncapi.EventTypeResourceUpdated)
		if event.Sequence <= lastSequence {
			t.Fatalf("expected strictly increasing sequence, previous=%d current=%d", lastSequence, event.Sequence)
		}
		if lastSequence > 0 && event.Sequence != lastSequence+1 {
			t.Fatalf("expected contiguous sequence %d, got %d", lastSequence+1, event.Sequence)
		}

		payload, ok := event.Payload.(map[string]any)
		if !ok {
			t.Fatalf("expected payload map, got %T", event.Payload)
		}
		if payload[syncapi.PayloadKeyEntityID] != expectedEntityIDs[i] {
			t.Fatalf("expected entity_id %q at index %d, got %v", expectedEntityIDs[i], i, payload[syncapi.PayloadKeyEntityID])
		}

		lastSequence = event.Sequence
	}

	healthRequest, err := http.NewRequest(http.MethodGet, server.URL+"/api/v1/sync/health", nil)
	if err != nil {
		t.Fatalf("new sync health request: %v", err)
	}

	healthResponse, err := http.DefaultClient.Do(healthRequest)
	if err != nil {
		t.Fatalf("send sync health request: %v", err)
	}
	defer healthResponse.Body.Close()

	if healthResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected sync health status 200, got %d", healthResponse.StatusCode)
	}

	var envelope struct {
		Hub struct {
			PublishedTotal int64 `json:"published_total"`
			DroppedTotal   int64 `json:"dropped_total"`
			LastSequence   int64 `json:"last_sequence"`
			HistoryDepth   int64 `json:"history_depth"`
		} `json:"hub"`
	}
	decodeSyncResponse(t, healthResponse, &envelope)

	if envelope.Hub.PublishedTotal < burstSize {
		t.Fatalf("expected published_total >= %d, got %d", burstSize, envelope.Hub.PublishedTotal)
	}
	if envelope.Hub.DroppedTotal != 0 {
		t.Fatalf("expected dropped_total 0 for burst stability test, got %d", envelope.Hub.DroppedTotal)
	}
	if envelope.Hub.LastSequence < lastSequence {
		t.Fatalf("expected last_sequence >= %d, got %d", lastSequence, envelope.Hub.LastSequence)
	}
	if envelope.Hub.HistoryDepth < burstSize {
		t.Fatalf("expected history_depth >= %d, got %d", burstSize, envelope.Hub.HistoryDepth)
	}
}

func TestSyncUnauthorizedAccessIntegration(t *testing.T) {
	server, _, _, cfg := newSyncIntegrationServer(t, true)

	publishResponse := publishSyncEvent(t, server.URL, "", "sync.update", map[string]any{"id": "res-2"})
	defer publishResponse.Body.Close()
	if publishResponse.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthorized sync publish, got %d", publishResponse.StatusCode)
	}
	assertErrorEnvelope(t, readResponseBody(t, publishResponse), "unauthorized")

	_, wsResponse, err := dialSyncWebSocket(server.URL, cfg.Sync.WebSocketPath, "")
	if err == nil {
		t.Fatal("expected websocket unauthorized dial to fail")
	}
	if wsResponse == nil {
		t.Fatal("expected websocket response with unauthorized status")
	}
	defer wsResponse.Body.Close()
	if wsResponse.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected websocket unauthorized status 401, got %d", wsResponse.StatusCode)
	}
}

func TestSyncHealthIncludesHubTelemetryIntegration(t *testing.T) {
	server, _, jwtService, _ := newSyncIntegrationServer(t, true)
	token := issueIntegrationToken(t, jwtService)

	publishResponse := publishSyncEvent(t, server.URL, token, "sync.update", map[string]any{"id": "res-health"})
	defer publishResponse.Body.Close()
	if publishResponse.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 from sync publish, got %d", publishResponse.StatusCode)
	}

	healthRequest, err := http.NewRequest(http.MethodGet, server.URL+"/api/v1/sync/health", nil)
	if err != nil {
		t.Fatalf("new sync health request: %v", err)
	}

	healthResponse, err := http.DefaultClient.Do(healthRequest)
	if err != nil {
		t.Fatalf("send sync health request: %v", err)
	}
	defer healthResponse.Body.Close()

	if healthResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected sync health status 200, got %d", healthResponse.StatusCode)
	}

	var envelope struct {
		Status      string `json:"status"`
		SyncEnabled bool   `json:"sync_enabled"`
		Hub         struct {
			PublishedTotal int64 `json:"published_total"`
			DroppedTotal   int64 `json:"dropped_total"`
			LastSequence   int64 `json:"last_sequence"`
			HistoryDepth   int64 `json:"history_depth"`
		} `json:"hub"`
	}
	decodeSyncResponse(t, healthResponse, &envelope)

	if envelope.Status != "ok" {
		t.Fatalf("expected health status ok, got %q", envelope.Status)
	}
	if !envelope.SyncEnabled {
		t.Fatalf("expected sync_enabled true")
	}
	if envelope.Hub.PublishedTotal < 1 {
		t.Fatalf("expected published_total >= 1, got %d", envelope.Hub.PublishedTotal)
	}
	if envelope.Hub.LastSequence < 1 {
		t.Fatalf("expected last_sequence >= 1, got %d", envelope.Hub.LastSequence)
	}
	if envelope.Hub.DroppedTotal < 0 {
		t.Fatalf("expected dropped_total >= 0, got %d", envelope.Hub.DroppedTotal)
	}
	if envelope.Hub.HistoryDepth < 1 {
		t.Fatalf("expected history_depth >= 1, got %d", envelope.Hub.HistoryDepth)
	}
}

func TestSyncObservabilityMetricsIntegration(t *testing.T) {
	server, _, jwtService, cfg := newSyncIntegrationServer(t, true)
	token := issueIntegrationToken(t, jwtService)

	unauthorizedPublish := publishSyncEvent(t, server.URL, "", "sync.update", map[string]any{"id": "res-observe-unauthorized"})
	defer unauthorizedPublish.Body.Close()
	if unauthorizedPublish.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized publish status 401, got %d", unauthorizedPublish.StatusCode)
	}

	conn, _, err := dialSyncWebSocket(server.URL, cfg.Sync.WebSocketPath, token)
	if err != nil {
		t.Fatalf("dial websocket for observability test: %v", err)
	}
	_ = readSyncEvent(t, conn)
	_ = conn.Close()

	publishResponse := publishSyncEvent(t, server.URL, token, "sync.update", map[string]any{"id": "res-observe"})
	defer publishResponse.Body.Close()
	if publishResponse.StatusCode != http.StatusAccepted {
		t.Fatalf("expected authorized publish status 202, got %d", publishResponse.StatusCode)
	}

	enqueueOfflineMutation(t, server.URL, token, map[string]any{
		"operation_id": "op-observe",
		"type":         syncapi.EventTypeResourceUpdated,
		"occurred_at":  "2026-04-14T14:00:00Z",
		"payload": map[string]any{
			"entity_id": "res-observe",
			"title":     "Observed",
		},
	})

	replaySummary := replayOfflineQueue(t, server.URL, token, 50)
	if replaySummary.ReplayedCount != 1 {
		t.Fatalf("expected replayed_count 1, got %d", replaySummary.ReplayedCount)
	}

	waitForCondition(t, 2*time.Second, func() bool {
		metrics := getSyncMetrics(t, server.URL, token)
		return metrics.AuthFailuresTotal >= 1 &&
			metrics.WebSocketConnectionsTotal >= 1 &&
			metrics.WebSocketDisconnectionsTotal >= 1 &&
			metrics.SyncEventsPublishedTotal >= 1 &&
			metrics.ReplayEnqueueAcceptedTotal >= 1 &&
			metrics.ReplayRequestsSucceededTotal >= 1 &&
			metrics.ReplayMutationsReplayedTotal >= 1
	}, "expected sync observability metrics counters to be populated")

	metricsRequest, err := http.NewRequest(http.MethodGet, server.URL+"/api/v1/sync/metrics", nil)
	if err != nil {
		t.Fatalf("new metrics request: %v", err)
	}
	metricsResponse, err := http.DefaultClient.Do(metricsRequest)
	if err != nil {
		t.Fatalf("send unauthorized metrics request: %v", err)
	}
	defer metricsResponse.Body.Close()
	if metricsResponse.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized metrics status 401, got %d", metricsResponse.StatusCode)
	}
	assertErrorEnvelope(t, readResponseBody(t, metricsResponse), "unauthorized")
}

func TestSyncAuthGatesAllProtectedEndpointsIntegration(t *testing.T) {
	server, _, jwtService, _ := newSyncIntegrationServer(t, true)
	token := issueIntegrationToken(t, jwtService)

	unauthorizedCases := []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodPost, path: "/api/v1/sync/events", body: `{"type":"sync.update","payload":{"id":"res-auth"}}`},
		{method: http.MethodPost, path: "/api/v1/sync/offline-queue/enqueue", body: `{"operation_id":"op-auth","type":"sync.resource.updated","payload":{"entity_id":"res-auth","title":"auth"}}`},
		{method: http.MethodPost, path: "/api/v1/sync/offline-queue/replay", body: `{"limit":10}`},
		{method: http.MethodGet, path: "/api/v1/sync/conflicts?limit=10", body: ""},
		{method: http.MethodGet, path: "/api/v1/sync/metrics", body: ""},
	}

	for _, testCase := range unauthorizedCases {
		response := sendSyncRequest(t, server.URL, testCase.method, testCase.path, "", testCase.body)
		if response.StatusCode != http.StatusUnauthorized {
			responseBody := readResponseBody(t, response)
			_ = response.Body.Close()
			t.Fatalf("expected 401 for %s %s, got %d body=%s", testCase.method, testCase.path, response.StatusCode, string(responseBody))
		}
		assertErrorEnvelope(t, readResponseBody(t, response), "unauthorized")
		_ = response.Body.Close()
	}

	authorizedEvent := sendSyncRequest(t, server.URL, http.MethodPost, "/api/v1/sync/events", token, `{"type":"sync.update","payload":{"id":"res-auth"}}`)
	if authorizedEvent.StatusCode != http.StatusAccepted {
		responseBody := readResponseBody(t, authorizedEvent)
		_ = authorizedEvent.Body.Close()
		t.Fatalf("expected authorized events status 202, got %d body=%s", authorizedEvent.StatusCode, string(responseBody))
	}
	_ = authorizedEvent.Body.Close()

	authorizedEnqueue := sendSyncRequest(t, server.URL, http.MethodPost, "/api/v1/sync/offline-queue/enqueue", token, `{"operation_id":"op-auth-2","type":"sync.resource.updated","payload":{"entity_id":"res-auth","title":"auth"}}`)
	if authorizedEnqueue.StatusCode != http.StatusAccepted {
		responseBody := readResponseBody(t, authorizedEnqueue)
		_ = authorizedEnqueue.Body.Close()
		t.Fatalf("expected authorized enqueue status 202, got %d body=%s", authorizedEnqueue.StatusCode, string(responseBody))
	}
	_ = authorizedEnqueue.Body.Close()

	authorizedReplay := sendSyncRequest(t, server.URL, http.MethodPost, "/api/v1/sync/offline-queue/replay", token, `{"limit":10}`)
	if authorizedReplay.StatusCode != http.StatusOK {
		responseBody := readResponseBody(t, authorizedReplay)
		_ = authorizedReplay.Body.Close()
		t.Fatalf("expected authorized replay status 200, got %d body=%s", authorizedReplay.StatusCode, string(responseBody))
	}
	_ = authorizedReplay.Body.Close()

	authorizedConflicts := sendSyncRequest(t, server.URL, http.MethodGet, "/api/v1/sync/conflicts?limit=10", token, "")
	if authorizedConflicts.StatusCode != http.StatusOK {
		responseBody := readResponseBody(t, authorizedConflicts)
		_ = authorizedConflicts.Body.Close()
		t.Fatalf("expected authorized conflicts status 200, got %d body=%s", authorizedConflicts.StatusCode, string(responseBody))
	}
	_ = authorizedConflicts.Body.Close()

	authorizedMetrics := sendSyncRequest(t, server.URL, http.MethodGet, "/api/v1/sync/metrics", token, "")
	if authorizedMetrics.StatusCode != http.StatusOK {
		responseBody := readResponseBody(t, authorizedMetrics)
		_ = authorizedMetrics.Body.Close()
		t.Fatalf("expected authorized metrics status 200, got %d body=%s", authorizedMetrics.StatusCode, string(responseBody))
	}
	_ = authorizedMetrics.Body.Close()
}

func TestSyncEventProtocolValidationIntegration(t *testing.T) {
	server, _, jwtService, _ := newSyncIntegrationServer(t, true)
	token := issueIntegrationToken(t, jwtService)

	unsupportedType := publishSyncEvent(t, server.URL, token, "sync.unknown", map[string]any{"entity_id": "res-1"})
	defer unsupportedType.Body.Close()
	if unsupportedType.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for unsupported sync event type, got %d", unsupportedType.StatusCode)
	}
	assertErrorEnvelope(t, readResponseBody(t, unsupportedType), "validation_error")

	missingEntity := publishSyncEvent(t, server.URL, token, "sync.resource.updated", map[string]any{"title": "Updated"})
	defer missingEntity.Body.Close()
	if missingEntity.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing entity_id, got %d", missingEntity.StatusCode)
	}
	assertErrorEnvelope(t, readResponseBody(t, missingEntity), "validation_error")

	validEntity := publishSyncEvent(t, server.URL, token, "sync.resource.updated", map[string]any{"entity_id": "res-1", "title": "Updated"})
	defer validEntity.Body.Close()
	if validEntity.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 for valid entity event, got %d", validEntity.StatusCode)
	}
}

func TestSyncWebSocketReceivesResourceCreateMutationIntegration(t *testing.T) {
	server, _, jwtService, cfg := newSyncCRUDIntegrationServer(t, true)
	token := issueIntegrationToken(t, jwtService)

	conn, _, err := dialSyncWebSocket(server.URL, cfg.Sync.WebSocketPath, token)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	connected := readSyncEvent(t, conn)
	if connected.Type != syncapi.EventTypeConnected {
		t.Fatalf("expected %q event, got %q", syncapi.EventTypeConnected, connected.Type)
	}

	categoryID := createCategoryForSync(t, server.URL)
	_ = createResourceForSync(t, server.URL, categoryID)

	event := readSyncEventByType(t, conn, syncapi.EventTypeResourceCreated)
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected payload map, got %T", event.Payload)
	}

	entityID, _ := payload[syncapi.PayloadKeyEntityID].(string)
	if strings.TrimSpace(entityID) == "" {
		t.Fatalf("expected payload.entity_id to be present")
	}

	source, _ := payload[syncapi.PayloadKeyEventSource].(string)
	if source != syncapi.EventSourceHTTPMutation {
		t.Fatalf("expected event source %q, got %q", syncapi.EventSourceHTTPMutation, source)
	}

	version, ok := payload[syncapi.PayloadKeyEventVersion].(float64)
	if !ok {
		t.Fatalf("expected numeric event_version, got %T", payload[syncapi.PayloadKeyEventVersion])
	}
	if int(version) != syncapi.EventVersionCurrent {
		t.Fatalf("expected event_version %d, got %v", syncapi.EventVersionCurrent, version)
	}
}

func TestSyncOfflineQueueConflictResolutionIntegration(t *testing.T) {
	server, _, jwtService, cfg := newSyncReplayIntegrationServer(t, true)
	token := issueIntegrationToken(t, jwtService)

	conn, _, err := dialSyncWebSocket(server.URL, cfg.Sync.WebSocketPath, token)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	_ = readSyncEvent(t, conn)

	enqueueOfflineMutation(t, server.URL, token, map[string]any{
		"operation_id": "op-old",
		"type":         syncapi.EventTypeResourceUpdated,
		"occurred_at":  "2026-04-14T09:00:00Z",
		"payload": map[string]any{
			"entity_id": "res-1",
			"title":     "Old Title",
		},
	})

	enqueueOfflineMutation(t, server.URL, token, map[string]any{
		"operation_id": "op-new",
		"type":         syncapi.EventTypeResourceUpdated,
		"occurred_at":  "2026-04-14T09:05:00Z",
		"payload": map[string]any{
			"entity_id": "res-1",
			"title":     "New Title",
		},
	})

	summary := replayOfflineQueue(t, server.URL, token, 50)
	if summary.ReplayedCount != 2 {
		t.Fatalf("expected replayed_count 2, got %d", summary.ReplayedCount)
	}
	if summary.EmittedCount != 1 {
		t.Fatalf("expected emitted_count 1, got %d", summary.EmittedCount)
	}
	if summary.ConflictCount != 1 {
		t.Fatalf("expected conflict_count 1, got %d", summary.ConflictCount)
	}

	event := readSyncEventByType(t, conn, syncapi.EventTypeResourceUpdated)
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected payload map, got %T", event.Payload)
	}
	if payload["title"] != "New Title" {
		t.Fatalf("expected replay winner title New Title, got %v", payload["title"])
	}
	source, _ := payload[syncapi.PayloadKeyEventSource].(string)
	if source != syncapi.EventSourceSyncReplay {
		t.Fatalf("expected replay source %q, got %q", syncapi.EventSourceSyncReplay, source)
	}

	conflicts := listSyncConflicts(t, server.URL, token, "res-1", 10)
	if len(conflicts) == 0 {
		t.Fatalf("expected at least one conflict record")
	}
	if conflicts[0].WinnerOperationID != "op-new" {
		t.Fatalf("expected conflict winner op-new, got %q", conflicts[0].WinnerOperationID)
	}
}

func TestSyncOfflineQueueReplayFIFOIntegration(t *testing.T) {
	server, _, jwtService, cfg := newSyncReplayIntegrationServer(t, true)
	token := issueIntegrationToken(t, jwtService)

	conn, _, err := dialSyncWebSocket(server.URL, cfg.Sync.WebSocketPath, token)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	_ = readSyncEvent(t, conn)

	enqueueOfflineMutation(t, server.URL, token, map[string]any{
		"operation_id": "op-1",
		"type":         syncapi.EventTypeResourceUpdated,
		"occurred_at":  "2026-04-14T10:00:00Z",
		"payload": map[string]any{
			"entity_id": "res-a",
			"title":     "A",
		},
	})
	enqueueOfflineMutation(t, server.URL, token, map[string]any{
		"operation_id": "op-2",
		"type":         syncapi.EventTypeResourceUpdated,
		"occurred_at":  "2026-04-14T10:01:00Z",
		"payload": map[string]any{
			"entity_id": "res-b",
			"title":     "B",
		},
	})
	enqueueOfflineMutation(t, server.URL, token, map[string]any{
		"operation_id": "op-3",
		"type":         syncapi.EventTypeResourceUpdated,
		"occurred_at":  "2026-04-14T10:02:00Z",
		"payload": map[string]any{
			"entity_id": "res-c",
			"title":     "C",
		},
	})

	summary := replayOfflineQueue(t, server.URL, token, 50)
	if summary.ReplayedCount != 3 {
		t.Fatalf("expected replayed_count 3, got %d", summary.ReplayedCount)
	}
	if summary.EmittedCount != 3 {
		t.Fatalf("expected emitted_count 3, got %d", summary.EmittedCount)
	}
	if summary.ConflictCount != 0 {
		t.Fatalf("expected conflict_count 0, got %d", summary.ConflictCount)
	}

	expectedOrder := []string{"res-a", "res-b", "res-c"}
	for _, expectedEntityID := range expectedOrder {
		event := readSyncEventByType(t, conn, syncapi.EventTypeResourceUpdated)
		payload, ok := event.Payload.(map[string]any)
		if !ok {
			t.Fatalf("expected payload map, got %T", event.Payload)
		}
		entityID, _ := payload[syncapi.PayloadKeyEntityID].(string)
		if entityID != expectedEntityID {
			t.Fatalf("expected replay order entity %q, got %q", expectedEntityID, entityID)
		}
	}
}

func TestSyncOfflineQueueEnqueueIdempotentIntegration(t *testing.T) {
	server, _, jwtService, cfg := newSyncReplayIntegrationServer(t, true)
	token := issueIntegrationToken(t, jwtService)

	conn, _, err := dialSyncWebSocket(server.URL, cfg.Sync.WebSocketPath, token)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	_ = readSyncEvent(t, conn)

	enqueueOfflineMutation(t, server.URL, token, map[string]any{
		"operation_id": "op-idempotent",
		"type":         syncapi.EventTypeResourceUpdated,
		"occurred_at":  "2026-04-14T11:00:00Z",
		"payload": map[string]any{
			"entity_id": "res-idempotent",
			"title":     "first",
		},
	})
	enqueueOfflineMutation(t, server.URL, token, map[string]any{
		"operation_id": "op-idempotent",
		"type":         syncapi.EventTypeResourceUpdated,
		"occurred_at":  "2026-04-14T11:01:00Z",
		"payload": map[string]any{
			"entity_id": "res-idempotent",
			"title":     "second",
		},
	})

	summary := replayOfflineQueue(t, server.URL, token, 50)
	if summary.QueuedCount != 1 {
		t.Fatalf("expected queued_count 1 for duplicate operation_id, got %d", summary.QueuedCount)
	}
	if summary.ReplayedCount != 1 {
		t.Fatalf("expected replayed_count 1 for duplicate operation_id, got %d", summary.ReplayedCount)
	}
	if summary.EmittedCount != 1 {
		t.Fatalf("expected emitted_count 1 for duplicate operation_id, got %d", summary.EmittedCount)
	}
	if summary.ConflictCount != 0 {
		t.Fatalf("expected conflict_count 0 for duplicate operation_id, got %d", summary.ConflictCount)
	}

	event := readSyncEventByType(t, conn, syncapi.EventTypeResourceUpdated)
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected payload map, got %T", event.Payload)
	}
	if payload["title"] != "first" {
		t.Fatalf("expected duplicate operation to preserve first payload title, got %v", payload["title"])
	}

	nextSummary := replayOfflineQueue(t, server.URL, token, 50)
	if nextSummary.QueuedCount != 0 {
		t.Fatalf("expected queued_count 0 after idempotent replay drain, got %d", nextSummary.QueuedCount)
	}
}

func TestSyncOfflineQueueReplayAppliesResourceMutationIntegration(t *testing.T) {
	server, _, jwtService, cfg := newSyncReplayCRUDIntegrationServer(t, true)
	token := issueIntegrationToken(t, jwtService)

	conn, _, err := dialSyncWebSocket(server.URL, cfg.Sync.WebSocketPath, token)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	_ = readSyncEvent(t, conn)

	categoryID := createCategoryForSync(t, server.URL)
	resourceID := createResourceForSync(t, server.URL, categoryID)

	enqueueOfflineMutation(t, server.URL, token, map[string]any{
		"operation_id": "op-apply-resource-update",
		"type":         syncapi.EventTypeResourceUpdated,
		"occurred_at":  "2026-04-14T12:00:00Z",
		"payload": map[string]any{
			"entity_id": resourceID,
			"title":     "Replay Updated Title",
		},
	})

	summary := replayOfflineQueue(t, server.URL, token, 50)
	if summary.ReplayedCount != 1 {
		t.Fatalf("expected replayed_count 1, got %d", summary.ReplayedCount)
	}
	if summary.EmittedCount != 1 {
		t.Fatalf("expected emitted_count 1, got %d", summary.EmittedCount)
	}

	event := readSyncEventByType(t, conn, syncapi.EventTypeResourceUpdated)
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected payload map, got %T", event.Payload)
	}
	if payload[syncapi.PayloadKeyEntityID] != resourceID {
		t.Fatalf("expected replay event for resource %q, got %v", resourceID, payload[syncapi.PayloadKeyEntityID])
	}
	if payload["title"] != "Replay Updated Title" {
		t.Fatalf("expected replay event payload title Replay Updated Title, got %v", payload["title"])
	}

	resource := getResourceForSync(t, server.URL, resourceID)
	if resource.Title != "Replay Updated Title" {
		t.Fatalf("expected replay to apply updated resource title, got %q", resource.Title)
	}
}

func TestSyncOfflineQueueReplayAppliesCategoryMutationIntegration(t *testing.T) {
	server, _, jwtService, cfg := newSyncReplayCRUDIntegrationServer(t, true)
	token := issueIntegrationToken(t, jwtService)

	categoryID := createCategoryForSync(t, server.URL)

	conn, _, err := dialSyncWebSocket(server.URL, cfg.Sync.WebSocketPath, token)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	_ = readSyncEvent(t, conn)

	enqueueOfflineMutation(t, server.URL, token, map[string]any{
		"operation_id": "op-apply-category-update",
		"type":         syncapi.EventTypeCategoryUpdated,
		"occurred_at":  "2026-04-14T12:15:00Z",
		"payload": map[string]any{
			"entity_id":   categoryID,
			"name":        "Replay Category Updated",
			"description": "Category updated by replay",
		},
	})

	summary := replayOfflineQueue(t, server.URL, token, 50)
	if summary.ReplayedCount != 1 {
		t.Fatalf("expected replayed_count 1, got %d", summary.ReplayedCount)
	}
	if summary.EmittedCount != 1 {
		t.Fatalf("expected emitted_count 1, got %d", summary.EmittedCount)
	}

	event := readSyncEventByType(t, conn, syncapi.EventTypeCategoryUpdated)
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected payload map, got %T", event.Payload)
	}
	if payload[syncapi.PayloadKeyEntityID] != categoryID {
		t.Fatalf("expected replay event for category %q, got %v", categoryID, payload[syncapi.PayloadKeyEntityID])
	}
	if payload[syncapi.PayloadKeyEventSource] != syncapi.EventSourceSyncReplay {
		t.Fatalf("expected replay source %q, got %v", syncapi.EventSourceSyncReplay, payload[syncapi.PayloadKeyEventSource])
	}

	category := getCategoryForSync(t, server.URL, categoryID)
	if category.Name != "Replay Category Updated" {
		t.Fatalf("expected replay to apply updated category name, got %q", category.Name)
	}
	if category.Description != "Category updated by replay" {
		t.Fatalf("expected replay to apply updated category description, got %q", category.Description)
	}
}

func TestSyncOfflineQueueReplayAppliesTodoMutationIntegration(t *testing.T) {
	server, _, jwtService, cfg := newSyncReplayCRUDIntegrationServer(t, true)
	token := issueIntegrationToken(t, jwtService)

	todoID := createTodoForSync(t, server.URL)

	conn, _, err := dialSyncWebSocket(server.URL, cfg.Sync.WebSocketPath, token)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	_ = readSyncEvent(t, conn)

	enqueueOfflineMutation(t, server.URL, token, map[string]any{
		"operation_id": "op-apply-todo-update",
		"type":         syncapi.EventTypeTodoUpdated,
		"occurred_at":  "2026-04-14T12:30:00Z",
		"payload": map[string]any{
			"entity_id": todoID,
			"title":     "Replay Todo Updated",
			"details":   "Todo updated by replay",
			"status":    "done",
			"due_at":    "2026-04-30T09:00:00Z",
		},
	})

	summary := replayOfflineQueue(t, server.URL, token, 50)
	if summary.ReplayedCount != 1 {
		t.Fatalf("expected replayed_count 1, got %d", summary.ReplayedCount)
	}
	if summary.EmittedCount != 1 {
		t.Fatalf("expected emitted_count 1, got %d", summary.EmittedCount)
	}

	event := readSyncEventByType(t, conn, syncapi.EventTypeTodoUpdated)
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected payload map, got %T", event.Payload)
	}
	if payload[syncapi.PayloadKeyEntityID] != todoID {
		t.Fatalf("expected replay event for todo %q, got %v", todoID, payload[syncapi.PayloadKeyEntityID])
	}
	if payload[syncapi.PayloadKeyEventSource] != syncapi.EventSourceSyncReplay {
		t.Fatalf("expected replay source %q, got %v", syncapi.EventSourceSyncReplay, payload[syncapi.PayloadKeyEventSource])
	}

	todo := getTodoForSync(t, server.URL, todoID)
	if todo.Title != "Replay Todo Updated" {
		t.Fatalf("expected replay to apply updated todo title, got %q", todo.Title)
	}
	if todo.Details != "Todo updated by replay" {
		t.Fatalf("expected replay to apply updated todo details, got %q", todo.Details)
	}
	if todo.Status != "done" {
		t.Fatalf("expected replay to apply updated todo status done, got %q", todo.Status)
	}
}

func TestSyncOfflineQueueReplayAppliesReminderMutationIntegration(t *testing.T) {
	server, _, jwtService, cfg := newSyncReplayCRUDIntegrationServer(t, true)
	token := issueIntegrationToken(t, jwtService)

	reminderID := createReminderForSync(t, server.URL)

	conn, _, err := dialSyncWebSocket(server.URL, cfg.Sync.WebSocketPath, token)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	_ = readSyncEvent(t, conn)

	enqueueOfflineMutation(t, server.URL, token, map[string]any{
		"operation_id": "op-apply-reminder-update",
		"type":         syncapi.EventTypeReminderUpdated,
		"occurred_at":  "2026-04-14T12:45:00Z",
		"payload": map[string]any{
			"entity_id": reminderID,
			"title":     "Replay Reminder Updated",
			"message":   "Reminder updated by replay",
			"status":    "sent",
			"remind_at": "2026-04-30T10:00:00Z",
		},
	})

	summary := replayOfflineQueue(t, server.URL, token, 50)
	if summary.ReplayedCount != 1 {
		t.Fatalf("expected replayed_count 1, got %d", summary.ReplayedCount)
	}
	if summary.EmittedCount != 1 {
		t.Fatalf("expected emitted_count 1, got %d", summary.EmittedCount)
	}

	event := readSyncEventByType(t, conn, syncapi.EventTypeReminderUpdated)
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected payload map, got %T", event.Payload)
	}
	if payload[syncapi.PayloadKeyEntityID] != reminderID {
		t.Fatalf("expected replay event for reminder %q, got %v", reminderID, payload[syncapi.PayloadKeyEntityID])
	}
	if payload[syncapi.PayloadKeyEventSource] != syncapi.EventSourceSyncReplay {
		t.Fatalf("expected replay source %q, got %v", syncapi.EventSourceSyncReplay, payload[syncapi.PayloadKeyEventSource])
	}

	reminder := getReminderForSync(t, server.URL, reminderID)
	if reminder.Title != "Replay Reminder Updated" {
		t.Fatalf("expected replay to apply updated reminder title, got %q", reminder.Title)
	}
	if reminder.Message != "Reminder updated by replay" {
		t.Fatalf("expected replay to apply updated reminder message, got %q", reminder.Message)
	}
	if reminder.Status != "sent" {
		t.Fatalf("expected replay to apply updated reminder status sent, got %q", reminder.Status)
	}
}

func TestSyncOfflineQueueReplayApplyFailurePersistsPendingMutationIntegration(t *testing.T) {
	server, _, jwtService, _ := newSyncReplayCRUDIntegrationServer(t, true)
	token := issueIntegrationToken(t, jwtService)

	enqueueOfflineMutation(t, server.URL, token, map[string]any{
		"operation_id": "op-apply-failure",
		"type":         syncapi.EventTypeResourceUpdated,
		"occurred_at":  "2026-04-14T13:00:00Z",
		"payload": map[string]any{
			"entity_id": "resource-does-not-exist",
			"title":     "Will Fail",
		},
	})

	firstReplay := replayOfflineQueueExpectStatus(t, server.URL, token, 50, http.StatusInternalServerError)
	assertErrorEnvelope(t, firstReplay, "internal_error")

	secondReplay := replayOfflineQueueExpectStatus(t, server.URL, token, 50, http.StatusInternalServerError)
	assertErrorEnvelope(t, secondReplay, "internal_error")
}

func TestSyncOfflineQueueReplayInvalidTodoStatusPersistsPendingMutationIntegration(t *testing.T) {
	server, _, jwtService, _ := newSyncReplayCRUDIntegrationServer(t, true)
	token := issueIntegrationToken(t, jwtService)

	todoID := createTodoForSync(t, server.URL)

	enqueueOfflineMutation(t, server.URL, token, map[string]any{
		"operation_id": "op-invalid-todo-status",
		"type":         syncapi.EventTypeTodoUpdated,
		"occurred_at":  "2026-04-14T13:10:00Z",
		"payload": map[string]any{
			"entity_id": todoID,
			"title":     "Replay Todo Invalid",
			"status":    "invalid_status",
		},
	})

	firstReplay := replayOfflineQueueExpectStatus(t, server.URL, token, 50, http.StatusInternalServerError)
	assertErrorEnvelope(t, firstReplay, "internal_error")

	secondReplay := replayOfflineQueueExpectStatus(t, server.URL, token, 50, http.StatusInternalServerError)
	assertErrorEnvelope(t, secondReplay, "internal_error")

	todo := getTodoForSync(t, server.URL, todoID)
	if todo.Title != "Sync Todo" {
		t.Fatalf("expected todo title to remain unchanged after failed replay, got %q", todo.Title)
	}
	if todo.Status != "open" {
		t.Fatalf("expected todo status to remain open after failed replay, got %q", todo.Status)
	}
}

func TestSyncOfflineQueueReplayInvalidReminderTimestampPersistsPendingMutationIntegration(t *testing.T) {
	server, _, jwtService, _ := newSyncReplayCRUDIntegrationServer(t, true)
	token := issueIntegrationToken(t, jwtService)

	reminderID := createReminderForSync(t, server.URL)

	enqueueOfflineMutation(t, server.URL, token, map[string]any{
		"operation_id": "op-invalid-reminder-time",
		"type":         syncapi.EventTypeReminderUpdated,
		"occurred_at":  "2026-04-14T13:15:00Z",
		"payload": map[string]any{
			"entity_id": reminderID,
			"title":     "Replay Reminder Invalid",
			"remind_at": "not-a-timestamp",
		},
	})

	firstReplay := replayOfflineQueueExpectStatus(t, server.URL, token, 50, http.StatusInternalServerError)
	assertErrorEnvelope(t, firstReplay, "internal_error")

	secondReplay := replayOfflineQueueExpectStatus(t, server.URL, token, 50, http.StatusInternalServerError)
	assertErrorEnvelope(t, secondReplay, "internal_error")

	reminder := getReminderForSync(t, server.URL, reminderID)
	if reminder.Title != "Sync Reminder" {
		t.Fatalf("expected reminder title to remain unchanged after failed replay, got %q", reminder.Title)
	}
	if reminder.Status != "scheduled" {
		t.Fatalf("expected reminder status to remain scheduled after failed replay, got %q", reminder.Status)
	}
}

func TestSyncOfflineQueueReplayInvalidResourceCategoryPersistsPendingMutationIntegration(t *testing.T) {
	server, _, jwtService, _ := newSyncReplayCRUDIntegrationServer(t, true)
	token := issueIntegrationToken(t, jwtService)

	categoryID := createCategoryForSync(t, server.URL)
	resourceID := createResourceForSync(t, server.URL, categoryID)

	enqueueOfflineMutation(t, server.URL, token, map[string]any{
		"operation_id": "op-invalid-resource-category",
		"type":         syncapi.EventTypeResourceUpdated,
		"occurred_at":  "2026-04-14T13:20:00Z",
		"payload": map[string]any{
			"entity_id":   resourceID,
			"title":       "Replay Resource Invalid",
			"category_id": "missing-category-id",
		},
	})

	firstReplay := replayOfflineQueueExpectStatus(t, server.URL, token, 50, http.StatusInternalServerError)
	assertErrorEnvelope(t, firstReplay, "internal_error")

	secondReplay := replayOfflineQueueExpectStatus(t, server.URL, token, 50, http.StatusInternalServerError)
	assertErrorEnvelope(t, secondReplay, "internal_error")

	resource := getResourceForSync(t, server.URL, resourceID)
	if resource.Title != "Sync Fanout" {
		t.Fatalf("expected resource title to remain unchanged after failed replay, got %q", resource.Title)
	}
	if resource.CategoryID != categoryID {
		t.Fatalf("expected resource category to remain %q after failed replay, got %q", categoryID, resource.CategoryID)
	}
}

func TestSyncOfflineQueueReplayInvalidCategoryPayloadPersistsPendingMutationIntegration(t *testing.T) {
	server, _, jwtService, _ := newSyncReplayCRUDIntegrationServer(t, true)
	token := issueIntegrationToken(t, jwtService)

	baselineCategoryID := createCategoryForSync(t, server.URL)

	enqueueOfflineMutation(t, server.URL, token, map[string]any{
		"operation_id": "op-invalid-category-payload",
		"type":         syncapi.EventTypeCategoryUpdated,
		"occurred_at":  "2026-04-14T13:25:00Z",
		"payload": map[string]any{
			"entity_id":   "missing-category-id",
			"description": "missing name should fail",
		},
	})

	firstReplay := replayOfflineQueueExpectStatus(t, server.URL, token, 50, http.StatusInternalServerError)
	assertErrorEnvelope(t, firstReplay, "internal_error")

	secondReplay := replayOfflineQueueExpectStatus(t, server.URL, token, 50, http.StatusInternalServerError)
	assertErrorEnvelope(t, secondReplay, "internal_error")

	baselineCategory := getCategoryForSync(t, server.URL, baselineCategoryID)
	if baselineCategory.Name != "Sync Test Category" {
		t.Fatalf("expected baseline category name to remain unchanged, got %q", baselineCategory.Name)
	}
	if baselineCategory.Description != "For websocket fanout test" {
		t.Fatalf("expected baseline category description to remain unchanged, got %q", baselineCategory.Description)
	}
}

func TestSyncOfflineQueueReplayInvalidResourceCreatePayloadPersistsPendingMutationIntegration(t *testing.T) {
	server, _, jwtService, _ := newSyncReplayCRUDIntegrationServer(t, true)
	token := issueIntegrationToken(t, jwtService)

	categoryID := createCategoryForSync(t, server.URL)
	baselineResourceID := createResourceForSync(t, server.URL, categoryID)

	enqueueOfflineMutation(t, server.URL, token, map[string]any{
		"operation_id": "op-invalid-resource-create-payload",
		"type":         syncapi.EventTypeResourceCreated,
		"occurred_at":  "2026-04-14T13:30:00Z",
		"payload": map[string]any{
			"entity_id":   "res-should-not-create",
			"url":         "not a valid url",
			"title":       "Should Not Create",
			"category_id": categoryID,
		},
	})

	firstReplay := replayOfflineQueueExpectStatus(t, server.URL, token, 50, http.StatusInternalServerError)
	assertErrorEnvelope(t, firstReplay, "internal_error")

	secondReplay := replayOfflineQueueExpectStatus(t, server.URL, token, 50, http.StatusInternalServerError)
	assertErrorEnvelope(t, secondReplay, "internal_error")

	baselineResource := getResourceForSync(t, server.URL, baselineResourceID)
	if baselineResource.Title != "Sync Fanout" {
		t.Fatalf("expected baseline resource title to remain unchanged, got %q", baselineResource.Title)
	}
	if baselineResource.CategoryID != categoryID {
		t.Fatalf("expected baseline resource category to remain %q, got %q", categoryID, baselineResource.CategoryID)
	}
}

func TestSyncOfflineQueueReplayResourceCreateMissingCategoryPersistsPendingMutationIntegration(t *testing.T) {
	server, _, jwtService, _ := newSyncReplayCRUDIntegrationServer(t, true)
	token := issueIntegrationToken(t, jwtService)

	categoryID := createCategoryForSync(t, server.URL)
	baselineResourceID := createResourceForSync(t, server.URL, categoryID)

	enqueueOfflineMutation(t, server.URL, token, map[string]any{
		"operation_id": "op-resource-create-missing-category",
		"type":         syncapi.EventTypeResourceCreated,
		"occurred_at":  "2026-04-14T13:35:00Z",
		"payload": map[string]any{
			"entity_id":   "res-missing-category",
			"url":         "https://example.com/should-not-create",
			"title":       "Should Not Create Missing Category",
			"category_id": "missing-category-id",
		},
	})

	firstReplay := replayOfflineQueueExpectStatus(t, server.URL, token, 50, http.StatusInternalServerError)
	assertErrorEnvelope(t, firstReplay, "internal_error")

	secondReplay := replayOfflineQueueExpectStatus(t, server.URL, token, 50, http.StatusInternalServerError)
	assertErrorEnvelope(t, secondReplay, "internal_error")

	missingRequest, err := http.NewRequest(http.MethodGet, server.URL+"/api/v1/resources/res-missing-category", nil)
	if err != nil {
		t.Fatalf("new missing resource request: %v", err)
	}
	missingResponse, err := http.DefaultClient.Do(missingRequest)
	if err != nil {
		t.Fatalf("send missing resource request: %v", err)
	}
	defer missingResponse.Body.Close()
	if missingResponse.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for resource not created by failed replay, got %d", missingResponse.StatusCode)
	}
	assertErrorEnvelope(t, readResponseBody(t, missingResponse), "not_found")

	baselineResource := getResourceForSync(t, server.URL, baselineResourceID)
	if baselineResource.Title != "Sync Fanout" {
		t.Fatalf("expected baseline resource title to remain unchanged, got %q", baselineResource.Title)
	}
	if baselineResource.CategoryID != categoryID {
		t.Fatalf("expected baseline resource category to remain %q, got %q", categoryID, baselineResource.CategoryID)
	}
}

func newSyncIntegrationServer(t *testing.T, authEnabled bool) (*httptest.Server, *syncapi.Hub, *authapi.JWTService, config.Config) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	router := gin.New()
	hub := syncapi.NewHub()
	cfg := config.Config{
		Sync: config.SyncConfig{
			Enabled:          true,
			WebSocketPath:    "/api/v1/sync/ws",
			HeartbeatSeconds: 1,
		},
		Auth: config.AuthConfig{
			Enabled:         authEnabled,
			JWTSecret:       "integration-secret",
			JWTIssuer:       "self-systems",
			JWTAudience:     "self-systems-clients",
			TokenTTLMinutes: 5,
		},
	}

	jwtService := authapi.NewJWTService(cfg.Auth)
	syncapi.RegisterBootstrapRoutes(router, cfg, hub, jwtService.Middleware())

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return server, hub, jwtService, cfg
}

func newSyncCRUDIntegrationServer(t *testing.T, authEnabled bool) (*httptest.Server, *syncapi.Hub, *authapi.JWTService, config.Config) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dbPath := filepath.Join(t.TempDir(), "self_systems_sync_integration.db")
	db, err := sqliterepo.Open(dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	categoryRepo := sqliterepo.NewCategoryRepository(db)
	resourceRepo := sqliterepo.NewResourceRepository(db)
	todoRepo := sqliterepo.NewTodoRepository(db)
	reminderRepo := sqliterepo.NewReminderRepository(db)

	aiManager := ai.NewManager("heuristic")
	heuristicProvider := ai.NewHeuristicProvider()
	aiManager.Register(heuristicProvider)
	aiManager.SetFallback(heuristicProvider.Name())

	categorySvc := service.NewCategoryService(categoryRepo)
	classifier := service.NewCategoryClassifier(categoryRepo, aiManager)
	resourceSvc := service.NewResourceService(resourceRepo, categoryRepo, classifier, categorySvc)
	todoSvc := service.NewTodoService(todoRepo)
	reminderSvc := service.NewReminderService(reminderRepo)
	graphSvc := service.NewGraphService(categoryRepo, resourceRepo)
	chatSvc := service.NewChatService(categorySvc, resourceSvc, todoSvc, reminderSvc, graphSvc)

	hub := syncapi.NewHub()
	router := gin.New()
	handler := httpapi.NewHandlerWithOptions(resourceSvc, categorySvc, todoSvc, reminderSvc, graphSvc, chatSvc, httpapi.WithSyncHub(hub))
	handler.RegisterRoutes(router)

	cfg := config.Config{
		Sync: config.SyncConfig{
			Enabled:          true,
			WebSocketPath:    "/api/v1/sync/ws",
			HeartbeatSeconds: 1,
		},
		Auth: config.AuthConfig{
			Enabled:         authEnabled,
			JWTSecret:       "integration-secret",
			JWTIssuer:       "self-systems",
			JWTAudience:     "self-systems-clients",
			TokenTTLMinutes: 5,
		},
	}

	jwtService := authapi.NewJWTService(cfg.Auth)
	syncapi.RegisterBootstrapRoutes(router, cfg, hub, jwtService.Middleware())

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return server, hub, jwtService, cfg
}

func newSyncReplayIntegrationServer(t *testing.T, authEnabled bool) (*httptest.Server, *syncapi.Hub, *authapi.JWTService, config.Config) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dbPath := filepath.Join(t.TempDir(), "self_systems_sync_replay_integration.db")
	db, err := sqliterepo.Open(dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	replayStore, err := syncapi.NewSQLiteReplayStore(db)
	if err != nil {
		t.Fatalf("open sqlite replay store: %v", err)
	}

	router := gin.New()
	hub := syncapi.NewHub()
	cfg := config.Config{
		Sync: config.SyncConfig{
			Enabled:          true,
			WebSocketPath:    "/api/v1/sync/ws",
			HeartbeatSeconds: 1,
		},
		Auth: config.AuthConfig{
			Enabled:         authEnabled,
			JWTSecret:       "integration-secret",
			JWTIssuer:       "self-systems",
			JWTAudience:     "self-systems-clients",
			TokenTTLMinutes: 5,
		},
	}

	jwtService := authapi.NewJWTService(cfg.Auth)
	replayManager := syncapi.NewOfflineReplayManager(replayStore, nil, hub)
	syncapi.RegisterBootstrapRoutes(router, cfg, hub, jwtService.Middleware(), syncapi.WithOfflineReplayManager(replayManager))

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return server, hub, jwtService, cfg
}

func newSyncReplayCRUDIntegrationServer(t *testing.T, authEnabled bool) (*httptest.Server, *syncapi.Hub, *authapi.JWTService, config.Config) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dbPath := filepath.Join(t.TempDir(), "self_systems_sync_replay_crud_integration.db")
	db, err := sqliterepo.Open(dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	categoryRepo := sqliterepo.NewCategoryRepository(db)
	resourceRepo := sqliterepo.NewResourceRepository(db)
	todoRepo := sqliterepo.NewTodoRepository(db)
	reminderRepo := sqliterepo.NewReminderRepository(db)
	replayStore, err := syncapi.NewSQLiteReplayStore(db)
	if err != nil {
		t.Fatalf("open sqlite replay store: %v", err)
	}

	aiManager := ai.NewManager("heuristic")
	heuristicProvider := ai.NewHeuristicProvider()
	aiManager.Register(heuristicProvider)
	aiManager.SetFallback(heuristicProvider.Name())

	categorySvc := service.NewCategoryService(categoryRepo)
	classifier := service.NewCategoryClassifier(categoryRepo, aiManager)
	resourceSvc := service.NewResourceService(resourceRepo, categoryRepo, classifier, categorySvc)
	todoSvc := service.NewTodoService(todoRepo)
	reminderSvc := service.NewReminderService(reminderRepo)
	graphSvc := service.NewGraphService(categoryRepo, resourceRepo)
	chatSvc := service.NewChatService(categorySvc, resourceSvc, todoSvc, reminderSvc, graphSvc)

	hub := syncapi.NewHub()
	router := gin.New()
	handler := httpapi.NewHandlerWithOptions(resourceSvc, categorySvc, todoSvc, reminderSvc, graphSvc, chatSvc, httpapi.WithSyncHub(hub))
	handler.RegisterRoutes(router)

	cfg := config.Config{
		Sync: config.SyncConfig{
			Enabled:          true,
			WebSocketPath:    "/api/v1/sync/ws",
			HeartbeatSeconds: 1,
		},
		Auth: config.AuthConfig{
			Enabled:         authEnabled,
			JWTSecret:       "integration-secret",
			JWTIssuer:       "self-systems",
			JWTAudience:     "self-systems-clients",
			TokenTTLMinutes: 5,
		},
	}

	jwtService := authapi.NewJWTService(cfg.Auth)
	replayApplier := syncapi.NewServiceMutationApplier(resourceSvc, categorySvc, todoSvc, reminderSvc)
	replayManager := syncapi.NewOfflineReplayManagerWithApplier(replayStore, nil, hub, replayApplier)
	syncapi.RegisterBootstrapRoutes(router, cfg, hub, jwtService.Middleware(), syncapi.WithOfflineReplayManager(replayManager))

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return server, hub, jwtService, cfg
}

func issueIntegrationToken(t *testing.T, jwtService *authapi.JWTService) string {
	t.Helper()
	token, err := jwtService.IssueToken("integration-user")
	if err != nil {
		t.Fatalf("issue jwt token: %v", err)
	}
	return token
}

func dialSyncWebSocket(baseURL, path, token string) (*websocket.Conn, *http.Response, error) {
	headers := http.Header{}
	if strings.TrimSpace(token) != "" {
		headers.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	}

	wsURL := "ws" + strings.TrimPrefix(baseURL, "http") + path
	return websocket.DefaultDialer.Dial(wsURL, headers)
}

func readSyncEvent(t *testing.T, conn *websocket.Conn) syncapi.Event {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	var event syncapi.Event
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatalf("read websocket event: %v", err)
	}
	return event
}

func publishSyncEvent(t *testing.T, baseURL, token, eventType string, payload map[string]any) *http.Response {
	t.Helper()

	requestBody, err := json.Marshal(map[string]any{
		"type":    eventType,
		"payload": payload,
	})
	if err != nil {
		t.Fatalf("marshal sync event request: %v", err)
	}

	request, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/sync/events", bytes.NewReader(requestBody))
	if err != nil {
		t.Fatalf("new sync publish request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(token) != "" {
		request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("send sync publish request: %v", err)
	}
	return response
}

func enqueueOfflineMutation(t *testing.T, baseURL, token string, payload map[string]any) {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal offline queue payload: %v", err)
	}

	request, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/sync/offline-queue/enqueue", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new offline enqueue request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("send offline enqueue request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted {
		t.Fatalf("expected offline enqueue status 202, got %d", response.StatusCode)
	}
}

func replayOfflineQueue(t *testing.T, baseURL, token string, limit int) syncapi.ReplaySummary {
	t.Helper()

	body, err := json.Marshal(map[string]any{"limit": limit})
	if err != nil {
		t.Fatalf("marshal replay request payload: %v", err)
	}

	request, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/sync/offline-queue/replay", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new replay request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("send replay request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected replay status 200, got %d", response.StatusCode)
	}

	var envelope struct {
		Data syncapi.ReplaySummary `json:"data"`
	}
	decodeSyncResponse(t, response, &envelope)
	return envelope.Data
}

func replayOfflineQueueExpectStatus(t *testing.T, baseURL, token string, limit int, expectedStatus int) []byte {
	t.Helper()

	body, err := json.Marshal(map[string]any{"limit": limit})
	if err != nil {
		t.Fatalf("marshal replay request payload: %v", err)
	}

	request, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/sync/offline-queue/replay", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new replay request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("send replay request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != expectedStatus {
		t.Fatalf("expected replay status %d, got %d", expectedStatus, response.StatusCode)
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read replay response body: %v", err)
	}
	return responseBody
}

func listSyncConflicts(t *testing.T, baseURL, token, entityID string, limit int) []syncapi.ConflictRecord {
	t.Helper()

	request, err := http.NewRequest(http.MethodGet, baseURL+"/api/v1/sync/conflicts?entity_id="+entityID+"&limit="+strconv.Itoa(limit), nil)
	if err != nil {
		t.Fatalf("new conflict list request: %v", err)
	}
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("send conflict list request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected conflict list status 200, got %d", response.StatusCode)
	}

	var envelope struct {
		Data []syncapi.ConflictRecord `json:"data"`
	}
	decodeSyncResponse(t, response, &envelope)
	return envelope.Data
}

func sendSyncRequest(t *testing.T, baseURL, method, path, token, body string) *http.Response {
	t.Helper()

	request, err := http.NewRequest(method, baseURL+path, strings.NewReader(body))
	if err != nil {
		t.Fatalf("new sync request %s %s: %v", method, path, err)
	}
	if strings.TrimSpace(body) != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if strings.TrimSpace(token) != "" {
		request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("send sync request %s %s: %v", method, path, err)
	}
	return response
}

func getSyncMetrics(t *testing.T, baseURL, token string) syncapi.ObservabilitySnapshot {
	t.Helper()

	request, err := http.NewRequest(http.MethodGet, baseURL+"/api/v1/sync/metrics", nil)
	if err != nil {
		t.Fatalf("new sync metrics request: %v", err)
	}
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("send sync metrics request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected sync metrics status 200, got %d", response.StatusCode)
	}

	var envelope struct {
		Data syncapi.ObservabilitySnapshot `json:"data"`
	}
	decodeSyncResponse(t, response, &envelope)
	return envelope.Data
}

func createCategoryForSync(t *testing.T, baseURL string) string {
	t.Helper()

	requestBody := []byte(`{"name":"Sync Test Category","description":"For websocket fanout test"}`)
	request, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/categories", bytes.NewReader(requestBody))
	if err != nil {
		t.Fatalf("new category request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("send category request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating category, got %d", response.StatusCode)
	}

	var payload struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decodeSyncResponse(t, response, &payload)
	return payload.Data.ID
}

func createResourceForSync(t *testing.T, baseURL, categoryID string) string {
	t.Helper()

	requestBody, err := json.Marshal(map[string]any{
		"url":         "https://example.com/sync-fanout",
		"title":       "Sync Fanout",
		"summary":     "resource for sync websocket test",
		"category_id": categoryID,
	})
	if err != nil {
		t.Fatalf("marshal resource payload: %v", err)
	}

	request, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/resources", bytes.NewReader(requestBody))
	if err != nil {
		t.Fatalf("new resource request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("send resource request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating resource, got %d", response.StatusCode)
	}

	var payload struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decodeSyncResponse(t, response, &payload)
	return payload.Data.ID
}

func getResourceForSync(t *testing.T, baseURL, resourceID string) struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	CategoryID string `json:"category_id"`
} {
	t.Helper()

	request, err := http.NewRequest(http.MethodGet, baseURL+"/api/v1/resources/"+resourceID, nil)
	if err != nil {
		t.Fatalf("new get resource request: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("send get resource request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting resource, got %d", response.StatusCode)
	}

	var payload struct {
		Data map[string]any `json:"data"`
	}
	decodeSyncResponse(t, response, &payload)

	resource := struct {
		ID         string `json:"id"`
		Title      string `json:"title"`
		CategoryID string `json:"category_id"`
	}{
		ID:    strings.TrimSpace(anyToString(payload.Data["id"], payload.Data["ID"])),
		Title: strings.TrimSpace(anyToString(payload.Data["title"], payload.Data["Title"])),
	}

	categoryID := strings.TrimSpace(anyToString(payload.Data["category_id"], payload.Data["CategoryID"]))
	resource.CategoryID = categoryID

	return resource
}

func getCategoryForSync(t *testing.T, baseURL, categoryID string) struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
} {
	t.Helper()

	request, err := http.NewRequest(http.MethodGet, baseURL+"/api/v1/categories/"+categoryID, nil)
	if err != nil {
		t.Fatalf("new get category request: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("send get category request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting category, got %d", response.StatusCode)
	}

	var payload struct {
		Data struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"data"`
	}
	decodeSyncResponse(t, response, &payload)
	return payload.Data
}

func createTodoForSync(t *testing.T, baseURL string) string {
	t.Helper()

	requestBody, err := json.Marshal(map[string]any{
		"title":   "Sync Todo",
		"details": "todo for replay integration",
	})
	if err != nil {
		t.Fatalf("marshal todo payload: %v", err)
	}

	request, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/todos", bytes.NewReader(requestBody))
	if err != nil {
		t.Fatalf("new todo request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("send todo request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating todo, got %d", response.StatusCode)
	}

	var payload struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decodeSyncResponse(t, response, &payload)
	return payload.Data.ID
}

func getTodoForSync(t *testing.T, baseURL, todoID string) struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Details string `json:"details"`
	Status  string `json:"status"`
} {
	t.Helper()

	request, err := http.NewRequest(http.MethodGet, baseURL+"/api/v1/todos/"+todoID, nil)
	if err != nil {
		t.Fatalf("new get todo request: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("send get todo request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting todo, got %d", response.StatusCode)
	}

	var payload struct {
		Data struct {
			ID      string `json:"id"`
			Title   string `json:"title"`
			Details string `json:"details"`
			Status  string `json:"status"`
		} `json:"data"`
	}
	decodeSyncResponse(t, response, &payload)
	return payload.Data
}

func createReminderForSync(t *testing.T, baseURL string) string {
	t.Helper()

	requestBody, err := json.Marshal(map[string]any{
		"title":     "Sync Reminder",
		"message":   "reminder for replay integration",
		"remind_at": "2026-04-29T10:00:00Z",
	})
	if err != nil {
		t.Fatalf("marshal reminder payload: %v", err)
	}

	request, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/reminders", bytes.NewReader(requestBody))
	if err != nil {
		t.Fatalf("new reminder request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("send reminder request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating reminder, got %d", response.StatusCode)
	}

	var payload struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decodeSyncResponse(t, response, &payload)
	return payload.Data.ID
}

func getReminderForSync(t *testing.T, baseURL, reminderID string) struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Message string `json:"message"`
	Status  string `json:"status"`
} {
	t.Helper()

	request, err := http.NewRequest(http.MethodGet, baseURL+"/api/v1/reminders/"+reminderID, nil)
	if err != nil {
		t.Fatalf("new get reminder request: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("send get reminder request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting reminder, got %d", response.StatusCode)
	}

	var payload struct {
		Data struct {
			ID      string `json:"id"`
			Title   string `json:"title"`
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"data"`
	}
	decodeSyncResponse(t, response, &payload)
	return payload.Data
}

func readSyncEventByType(t *testing.T, conn *websocket.Conn, eventType string) syncapi.Event {
	t.Helper()

	for i := 0; i < 6; i++ {
		event := readSyncEvent(t, conn)
		if event.Type == eventType {
			return event
		}
	}

	t.Fatalf("expected to receive event type %q", eventType)
	return syncapi.Event{}
}

func decodeSyncResponse(t *testing.T, response *http.Response, target any) {
	t.Helper()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("decode response json: %v, body=%s", err, string(body))
	}
}

func waitForCondition(t *testing.T, timeout time.Duration, predicate func() bool, failureMessage string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if predicate() {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal(failureMessage)
}

func readResponseBody(t *testing.T, response *http.Response) []byte {
	t.Helper()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	return body
}

func anyToString(values ...any) string {
	for _, value := range values {
		if value == nil {
			continue
		}
		switch typed := value.(type) {
		case string:
			return strings.TrimSpace(typed)
		default:
			return strings.TrimSpace(fmt.Sprint(typed))
		}
	}
	return ""
}
