package migration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"selfsystems/internal/eventstore"
)

// Divergence records a single field mismatch between the live projection
// and the state rebuilt from the event log.
type Divergence struct {
	ResourceID string
	Field      string
	LiveValue  string
	EventValue string
}

// ParityReport summarises a parity check run.
type ParityReport struct {
	Checked           int
	Divergences       []Divergence
	ExtraInProjection []string // IDs in projection with no events (not yet backfilled)
	ExtraInEvents     []string // IDs in events but missing from projection (e.g. deleted)
	Duration          time.Duration
}

// IsClean returns true when the report has no divergences and no extra IDs.
func (r ParityReport) IsClean() bool {
	return len(r.Divergences) == 0 && len(r.ExtraInProjection) == 0 && len(r.ExtraInEvents) == 0
}

// projectedState is the resource state as rebuilt from the event log.
type projectedState struct {
	id           string
	url          string
	host         string
	title        string
	summary      string
	categoryID   string
	categoryName string
	userOverride bool
	deleted      bool
}

// CheckResourceParity reads the live resources projection and the event log,
// rebuilds the projection from events, and returns a diff report.
func CheckResourceParity(ctx context.Context, db *sql.DB, store eventstore.Store) (ParityReport, error) {
	start := time.Now()

	live, err := readLiveProjection(ctx, db)
	if err != nil {
		return ParityReport{}, fmt.Errorf("read live projection: %w", err)
	}

	rebuilt, err := rebuildFromEvents(ctx, db)
	if err != nil {
		return ParityReport{}, fmt.Errorf("rebuild from events: %w", err)
	}

	report := compareProjections(live, rebuilt)
	report.Duration = time.Since(start)
	return report, nil
}

// readLiveProjection reads all rows from resources JOIN categories.
func readLiveProjection(ctx context.Context, db *sql.DB) (map[string]projectedState, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT r.id, r.url, r.host, r.title, r.summary, r.category_id, c.name, r.user_override
		FROM resources r
		JOIN categories c ON c.id = r.category_id
	`)
	if err != nil {
		return nil, fmt.Errorf("query live projection: %w", err)
	}
	defer rows.Close()

	result := make(map[string]projectedState)
	for rows.Next() {
		var s projectedState
		var userOverrideRaw interface{}
		if err := rows.Scan(
			&s.id, &s.url, &s.host, &s.title, &s.summary,
			&s.categoryID, &s.categoryName, &userOverrideRaw,
		); err != nil {
			return nil, fmt.Errorf("scan live row: %w", err)
		}
		s.userOverride = scanBool(userOverrideRaw)
		result[s.id] = s
	}
	return result, rows.Err()
}

// rebuildFromEvents applies resource events in version order to derive
// the expected projection state for every resource aggregate.
func rebuildFromEvents(ctx context.Context, db *sql.DB) (map[string]projectedState, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT aggregate_id, event_type, event_version, payload
		FROM events
		WHERE aggregate_type = 'resource'
		ORDER BY aggregate_id, event_version ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query resource events: %w", err)
	}
	defer rows.Close()

	states := make(map[string]projectedState)
	for rows.Next() {
		var aggregateID, eventType string
		var version int
		var payloadBytes []byte
		if err := rows.Scan(&aggregateID, &eventType, &version, &payloadBytes); err != nil {
			return nil, fmt.Errorf("scan event row: %w", err)
		}
		applyEvent(states, aggregateID, eventType, payloadBytes)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return states, nil
}

// applyEvent updates states by applying a single event.
func applyEvent(states map[string]projectedState, aggregateID, eventType string, payloadBytes []byte) {
	switch eventType {
	case eventstore.EventTypeResourceCreated, eventstore.EventTypeResourceImported:
		var p eventstore.ResourceCreatedPayload
		if err := json.Unmarshal(payloadBytes, &p); err != nil {
			return
		}
		states[aggregateID] = projectedState{
			id:           aggregateID,
			url:          p.URL,
			host:         p.Host,
			title:        p.Title,
			summary:      p.Summary,
			categoryID:   p.CategoryID,
			categoryName: p.CategoryName,
			userOverride: p.UserOverride,
		}

	case eventstore.EventTypeResourceUpdated:
		var p eventstore.ResourceUpdatedPayload
		if err := json.Unmarshal(payloadBytes, &p); err != nil {
			return
		}
		s := states[aggregateID]
		s.url = p.URL
		s.host = p.Host
		s.title = p.Title
		s.summary = p.Summary
		s.categoryID = p.CategoryID
		s.categoryName = p.CategoryName
		s.userOverride = p.UserOverride
		states[aggregateID] = s

	case eventstore.EventTypeResourceCategoryAssigned:
		var p eventstore.ResourceCategoryAssignedPayload
		if err := json.Unmarshal(payloadBytes, &p); err != nil {
			return
		}
		s := states[aggregateID]
		s.categoryID = p.CategoryID
		s.categoryName = p.CategoryName
		s.userOverride = p.UserOverride
		states[aggregateID] = s

	case eventstore.EventTypeResourceDeleted:
		s := states[aggregateID]
		s.deleted = true
		states[aggregateID] = s
	}
}

// compareProjections builds the ParityReport from live vs rebuilt maps.
func compareProjections(live, rebuilt map[string]projectedState) ParityReport {
	var report ParityReport

	// Find IDs in live that are missing from events or have divergences.
	liveIDs := make([]string, 0, len(live))
	for id := range live {
		liveIDs = append(liveIDs, id)
	}
	sort.Strings(liveIDs)

	for _, id := range liveIDs {
		liveR := live[id]
		rebuiltR, ok := rebuilt[id]
		if !ok || rebuiltR.deleted {
			report.ExtraInProjection = append(report.ExtraInProjection, id)
			continue
		}
		report.Checked++
		report.Divergences = append(report.Divergences, diffStates(liveR, rebuiltR)...)
	}

	// Find IDs with events that are NOT deleted but missing from the projection.
	rebuiltIDs := make([]string, 0, len(rebuilt))
	for id := range rebuilt {
		rebuiltIDs = append(rebuiltIDs, id)
	}
	sort.Strings(rebuiltIDs)

	for _, id := range rebuiltIDs {
		s := rebuilt[id]
		if s.deleted {
			continue // deleted resources are expected to be absent from projection
		}
		if _, ok := live[id]; !ok {
			report.ExtraInEvents = append(report.ExtraInEvents, id)
		}
	}

	return report
}

// diffStates returns field-level divergences between live and rebuilt states.
func diffStates(live, rebuilt projectedState) []Divergence {
	var divs []Divergence
	check := func(field, lv, rv string) {
		if lv != rv {
			divs = append(divs, Divergence{
				ResourceID: live.id,
				Field:      field,
				LiveValue:  lv,
				EventValue: rv,
			})
		}
	}
	boolStr := func(b bool) string {
		if b {
			return "true"
		}
		return "false"
	}
	check("url", live.url, rebuilt.url)
	check("host", live.host, rebuilt.host)
	check("title", live.title, rebuilt.title)
	check("summary", live.summary, rebuilt.summary)
	check("category_id", live.categoryID, rebuilt.categoryID)
	check("category_name", live.categoryName, rebuilt.categoryName)
	check("user_override", boolStr(live.userOverride), boolStr(rebuilt.userOverride))
	return divs
}

// FormatReport returns a human-readable summary of a ParityReport.
func FormatReport(r ParityReport) string {
	if r.IsClean() {
		return fmt.Sprintf("PASS: %d resources checked, no divergences (%.2fs)",
			r.Checked, r.Duration.Seconds())
	}
	result := fmt.Sprintf("FAIL: %d checked, %d divergences, %d extra-in-projection, %d extra-in-events (%.2fs)\n",
		r.Checked, len(r.Divergences), len(r.ExtraInProjection), len(r.ExtraInEvents),
		r.Duration.Seconds())
	for _, d := range r.Divergences {
		result += fmt.Sprintf("  [%s] %s: live=%q event=%q\n", d.ResourceID, d.Field, d.LiveValue, d.EventValue)
	}
	for _, id := range r.ExtraInProjection {
		result += fmt.Sprintf("  [%s] in projection but no events (needs backfill)\n", id)
	}
	for _, id := range r.ExtraInEvents {
		result += fmt.Sprintf("  [%s] in events but missing from projection\n", id)
	}
	return result
}
