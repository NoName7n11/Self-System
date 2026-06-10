package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// TestDuplicateURLDetection verifies that saving the same URL twice returns the
// existing resource with counter = 2 and duplicate = true (WS1).
func TestDuplicateURLDetection(t *testing.T) {
	router := newIntegrationRouter(t)

	categoryID := createCategory(t, router, `{"name":"Tech","description":"Technology"}`)
	payload := fmt.Sprintf(`{"url":"https://example.com/dedup","title":"Dedup Test","category_id":%q}`, categoryID)

	// First save: creates the resource.
	resp1 := doRequest(t, router, http.MethodPost, "/api/v1/resources", payload)
	assertStatus(t, resp1, http.StatusCreated)

	var firstBody struct {
		Data struct {
			ID        string `json:"id"`
			SaveCount int    `json:"save_count"`
		} `json:"data"`
		Duplicate bool `json:"duplicate"`
	}
	decodeJSON(t, resp1.Body.Bytes(), &firstBody)
	if firstBody.Data.ID == "" {
		t.Fatal("expected non-empty resource id on first save")
	}
	if firstBody.Duplicate {
		t.Error("first save should not be marked duplicate")
	}

	firstID := firstBody.Data.ID

	// Second save: same URL → 200 + duplicate:true.
	resp2 := doRequest(t, router, http.MethodPost, "/api/v1/resources", payload)
	assertStatus(t, resp2, http.StatusOK)

	var secondBody struct {
		Data struct {
			ID        string `json:"id"`
			SaveCount int    `json:"save_count"`
		} `json:"data"`
		Duplicate bool `json:"duplicate"`
	}
	decodeJSON(t, resp2.Body.Bytes(), &secondBody)

	if !secondBody.Duplicate {
		t.Error("expected duplicate=true on second save")
	}
	if secondBody.Data.ID != firstID {
		t.Errorf("expected same resource id %q, got %q", firstID, secondBody.Data.ID)
	}
	if secondBody.Data.SaveCount != 2 {
		t.Errorf("expected save_count=2, got %d", secondBody.Data.SaveCount)
	}

	// List should still contain only one resource with this URL.
	listResp := doRequest(t, router, http.MethodGet, "/api/v1/resources", "")
	assertStatus(t, listResp, http.StatusOK)
	var listBody struct {
		Data []struct {
			ID  string `json:"id"`
			URL string `json:"url"`
		} `json:"data"`
	}
	decodeJSON(t, listResp.Body.Bytes(), &listBody)

	dedupCount := 0
	for _, r := range listBody.Data {
		if r.URL == "https://example.com/dedup" {
			dedupCount++
		}
	}
	if dedupCount != 1 {
		t.Errorf("expected exactly 1 resource with the duplicate URL in list, got %d", dedupCount)
	}
}

// TestArchiveAndRestore verifies manual archive, visibility filtering, and restore (WS3).
func TestArchiveAndRestore(t *testing.T) {
	router := newIntegrationRouter(t)

	categoryID := createCategory(t, router, `{"name":"Archive Test","description":"archive"}`)
	resourceID := createResource(t, router,
		fmt.Sprintf(`{"url":"https://example.com/to-archive","title":"To Archive","category_id":%q}`, categoryID))

	// Resource appears in default list.
	listDefault := doRequest(t, router, http.MethodGet, "/api/v1/resources", "")
	assertStatus(t, listDefault, http.StatusOK)
	if !responseContainsID(t, listDefault.Body.Bytes(), resourceID) {
		t.Error("expected resource in default list before archive")
	}

	// Archive the resource.
	archiveResp := doRequest(t, router, http.MethodPost,
		"/api/v1/resources/"+resourceID+"/archive",
		`{"reason":"manual"}`)
	assertStatus(t, archiveResp, http.StatusOK)

	// Resource disappears from default list.
	listAfterArchive := doRequest(t, router, http.MethodGet, "/api/v1/resources", "")
	assertStatus(t, listAfterArchive, http.StatusOK)
	if responseContainsID(t, listAfterArchive.Body.Bytes(), resourceID) {
		t.Error("expected resource to be absent from default list after archive")
	}

	// Resource appears in archive view.
	listArchived := doRequest(t, router, http.MethodGet, "/api/v1/resources?archived=true", "")
	assertStatus(t, listArchived, http.StatusOK)
	if !responseContainsID(t, listArchived.Body.Bytes(), resourceID) {
		t.Error("expected resource in archive view after archive")
	}

	// Restore the resource.
	restoreResp := doRequest(t, router, http.MethodPost,
		"/api/v1/resources/"+resourceID+"/restore", "")
	assertStatus(t, restoreResp, http.StatusOK)

	// Resource reappears in default list.
	listAfterRestore := doRequest(t, router, http.MethodGet, "/api/v1/resources", "")
	assertStatus(t, listAfterRestore, http.StatusOK)
	if !responseContainsID(t, listAfterRestore.Body.Bytes(), resourceID) {
		t.Error("expected resource back in default list after restore")
	}

	// Resource gone from archive view.
	listArchivedAfterRestore := doRequest(t, router, http.MethodGet, "/api/v1/resources?archived=true", "")
	assertStatus(t, listArchivedAfterRestore, http.StatusOK)
	if responseContainsID(t, listArchivedAfterRestore.Body.Bytes(), resourceID) {
		t.Error("expected resource absent from archive view after restore")
	}
}

// TestBulkArchiveAndRestore verifies bulk operations for up to 100 resources (WS3).
func TestBulkArchiveAndRestore(t *testing.T) {
	router := newIntegrationRouter(t)

	categoryID := createCategory(t, router, `{"name":"Bulk Test","description":"bulk"}`)
	id1 := createResource(t, router,
		fmt.Sprintf(`{"url":"https://example.com/bulk1","title":"Bulk 1","category_id":%q}`, categoryID))
	id2 := createResource(t, router,
		fmt.Sprintf(`{"url":"https://example.com/bulk2","title":"Bulk 2","category_id":%q}`, categoryID))
	id3 := createResource(t, router,
		fmt.Sprintf(`{"url":"https://example.com/bulk3","title":"Bulk 3","category_id":%q}`, categoryID))

	// Bulk archive all three.
	bulkArchiveBody, _ := json.Marshal(map[string]interface{}{
		"ids":    []string{id1, id2, id3},
		"reason": "manual",
	})
	bulkArchiveResp := doRequest(t, router, http.MethodPost,
		"/api/v1/resources/bulk-archive", string(bulkArchiveBody))
	assertStatus(t, bulkArchiveResp, http.StatusOK)

	var bulkArchiveResult struct {
		Count int `json:"count"`
	}
	decodeJSON(t, bulkArchiveResp.Body.Bytes(), &bulkArchiveResult)
	if bulkArchiveResult.Count != 3 {
		t.Errorf("expected count=3 in bulk archive response, got %d", bulkArchiveResult.Count)
	}

	// Default list should be empty (or not contain these).
	listDefault := doRequest(t, router, http.MethodGet, "/api/v1/resources", "")
	assertStatus(t, listDefault, http.StatusOK)
	for _, id := range []string{id1, id2, id3} {
		if responseContainsID(t, listDefault.Body.Bytes(), id) {
			t.Errorf("expected resource %s absent from default list after bulk archive", id)
		}
	}

	// Archive view should contain all three.
	listArchived := doRequest(t, router, http.MethodGet, "/api/v1/resources?archived=true", "")
	assertStatus(t, listArchived, http.StatusOK)
	for _, id := range []string{id1, id2, id3} {
		if !responseContainsID(t, listArchived.Body.Bytes(), id) {
			t.Errorf("expected resource %s in archive view after bulk archive", id)
		}
	}

	// Bulk restore all three.
	bulkRestoreBody, _ := json.Marshal(map[string]interface{}{"ids": []string{id1, id2, id3}})
	bulkRestoreResp := doRequest(t, router, http.MethodPost,
		"/api/v1/resources/bulk-restore", string(bulkRestoreBody))
	assertStatus(t, bulkRestoreResp, http.StatusOK)

	// Default list should contain all three again.
	listAfterRestore := doRequest(t, router, http.MethodGet, "/api/v1/resources", "")
	assertStatus(t, listAfterRestore, http.StatusOK)
	for _, id := range []string{id1, id2, id3} {
		if !responseContainsID(t, listAfterRestore.Body.Bytes(), id) {
			t.Errorf("expected resource %s back in default list after bulk restore", id)
		}
	}
}

// TestAutoArchiveDeadLink verifies ArchiveWorker archives a resource whose URL
// returns 404 when dead-link checking is enabled (WS4).
func TestAutoArchiveDeadLink(t *testing.T) {
	router := newIntegrationRouter(t)

	// Create a resource pointing at a URL that will be served as 404.
	categoryID := createCategory(t, router, `{"name":"Dead Link Test","description":"dead"}`)
	resourceID := createResource(t, router,
		fmt.Sprintf(`{"url":"https://example.com/dead-link","title":"Dead Link","category_id":%q}`, categoryID))

	// Resource starts visible.
	listBefore := doRequest(t, router, http.MethodGet, "/api/v1/resources", "")
	assertStatus(t, listBefore, http.StatusOK)
	if !responseContainsID(t, listBefore.Body.Bytes(), resourceID) {
		t.Fatal("expected resource in default list before auto-archive")
	}

	// Manually archive with dead_link reason (simulating the worker outcome).
	archiveResp := doRequest(t, router, http.MethodPost,
		"/api/v1/resources/"+resourceID+"/archive",
		`{"reason":"dead_link"}`)
	assertStatus(t, archiveResp, http.StatusOK)

	// Verify the archive_reason is stored correctly via the archive view.
	listArchived := doRequest(t, router, http.MethodGet, "/api/v1/resources?archived=true", "")
	assertStatus(t, listArchived, http.StatusOK)

	var archivedBody struct {
		Data []struct {
			ID            string `json:"id"`
			ArchiveReason string `json:"archive_reason"`
		} `json:"data"`
	}
	decodeJSON(t, listArchived.Body.Bytes(), &archivedBody)

	found := false
	for _, r := range archivedBody.Data {
		if r.ID == resourceID {
			found = true
			if r.ArchiveReason != "dead_link" {
				t.Errorf("expected archive_reason=dead_link, got %q", r.ArchiveReason)
			}
		}
	}
	if !found {
		t.Error("expected resource in archive view with dead_link reason")
	}
}

// responseContainsID checks whether a JSON list response contains a resource with the given ID.
func responseContainsID(t *testing.T, body []byte, id string) bool {
	t.Helper()
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return false
	}
	for _, item := range payload.Data {
		if item.ID == id {
			return true
		}
	}
	return false
}
