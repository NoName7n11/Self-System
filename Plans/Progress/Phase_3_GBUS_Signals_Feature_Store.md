# Phase 3 - GBUS Signal Taxonomy and Feature Store Spec

Purpose: Define the GBUS signal taxonomy and the feature store design needed for model training and inference.
Scope: Signals, event schema, aggregation layers, retention, and data governance. No model training details here.

## 1. Guiding Constraints

- Local-first behavior must remain fast and offline-tolerant.
- Keep domain logic loosely coupled from storage adapters.
- Use config-driven feature flags for opt-in or removable components.
- Store only what is necessary; avoid sensitive content in raw events.
- Preserve configuration precedence (config defaults, .env overrides, env overrides).

## 2. Signal Taxonomy

Signals are grouped by intent so the model can distinguish passive interest from strong intent.

### 2.1 Interaction Signals (Passive)

These indicate exposure and attention without explicit intent.

- resource_opened (resource_id, category_id, view_context, dwell_ms)
- resource_revisited (resource_id, category_id, revisit_gap_hours)
- resource_scrolled (resource_id, scroll_depth_pct, dwell_ms)
- graph_node_selected (entity_type, entity_id, category_id)
- search_result_clicked (query_id, resource_id, rank_position)

### 2.2 Action Signals (Explicit Intent)

These are the strongest indicators of interest and should carry higher weight.

- resource_saved (source_type, category_id, save_reason)
- resource_category_assigned (category_id, confidence, is_manual)
- resource_category_changed (from_category_id, to_category_id, is_manual)
- resource_favorite_toggled (is_favorite)
- resource_priority_set (priority_level)
- resource_read_status_changed (is_read)
- resource_deleted (delete_reason)
- resource_archived (archive_reason)
- resource_unarchived (restore_reason)

### 2.3 Reminder and Task Signals

These show downstream intent and follow-through.

- reminder_created (reminder_id, category_id, due_at)
- reminder_completed (reminder_id, completion_method)
- reminder_snoozed (reminder_id, snooze_minutes)
- reminder_dismissed (reminder_id)
- todo_created (todo_id, category_id, due_at)
- todo_completed (todo_id, completion_method)
- todo_rescheduled (todo_id, from_due_at, to_due_at)
- todo_linked_resource (todo_id, resource_id)

### 2.4 Search and Chat Signals

These represent active intent and should be modeled separately from browsing.

- search_query_executed (query_id, query_text, filters)
- search_filters_applied (query_id, filter_payload)
- chat_query_executed (query_id, intent_label, category_hint)
- chat_result_clicked (query_id, entity_type, entity_id)

### 2.5 System and AI Signals

These are system-derived indicators and should be used as lower-weight signals.

- ai_classification_applied (category_id, confidence, model_id)
- ai_suggestion_accepted (suggestion_type, category_id)
- ai_suggestion_rejected (suggestion_type, category_id)
- deep_processing_completed (resource_id, extraction_quality)
- embeddings_generated (resource_id, embedding_version)

### 2.6 Temporal and Derived Signals (Aggregates)

These are not raw events. They are derived during aggregation.

- time_of_day_preference (hour_bucket, activity_count)
- day_of_week_preference (weekday, activity_count)
- recency_weight (days_since_last_action)
- burstiness (events_per_day, rolling_window)

### 2.7 Cross-Category Signals (Aggregates)

Cross-category features capture co-occurrence and transitions.

- category_cooccurrence (category_id_a, category_id_b, session_count)
- category_transition (from_category_id, to_category_id, transition_count)

## 3. Event Schema (Raw Signal Event)

All signal events use a consistent schema for storage and aggregation.

Required fields:
- event_id (UUID)
- user_id (UUID)
- event_type (enum)
- event_ts (RFC3339)
- entity_type (resource|category|todo|reminder|search|chat|system)
- entity_id (UUID or query_id)
- source (ui|api|worker|sync)
- event_version (int)

Optional fields:
- device_id (UUID)
- session_id (UUID)
- category_id (UUID)
- related_category_ids (UUID[])
- operation_id (UUID, if derived from sync/offline replay)
- privacy_level (public|private|confidential)
- metadata (JSON object, event-specific payload)

## 4. Feature Store Design

### 4.1 Storage Layers

1) Raw signals (append-only)
2) Daily aggregates (user, category, resource)
3) Weekly aggregates (user, category)
4) Cross-category aggregates
5) Training extracts (materialized on demand)

### 4.2 Proposed Tables (Logical Model)

Raw:
- gbus_signal_events
  - event_id, user_id, event_type, event_ts, entity_type, entity_id,
    category_id, device_id, session_id, source, event_version,
    privacy_level, metadata_json, metadata_enc

Daily aggregates:
- gbus_feature_user_category_daily
  - user_id, category_id, day
  - saves_count, manual_moves_count, favorites_count, priority_count
  - views_count, revisits_count, dwell_ms_sum, search_clicks_count
  - reminders_created_count, todos_completed_count
  - deletes_count, archives_count
  - ai_auto_classify_count, ai_avg_confidence
  - last_action_at

- gbus_feature_resource_daily
  - resource_id, day
  - views_count, revisits_count, dwell_ms_sum
  - is_favorite, priority_level, counter_value
  - last_viewed_at

Weekly aggregates:
- gbus_feature_user_category_weekly
  - user_id, category_id, week_start
  - saves_count, views_count, revisits_count
  - favorites_count, priority_count
  - manual_moves_count, deletes_count, archives_count
  - last_action_at

Cross-category:
- gbus_feature_category_cooccurrence_daily
  - user_id, day, category_id_a, category_id_b
  - cooccurrence_sessions_count

Global user features:
- gbus_feature_user_daily
  - user_id, day
  - total_saves, total_views, active_days_30d
  - category_entropy, diversity_score

### 4.3 Feature Versioning

- Each aggregate table includes feature_version and aggregator_version.
- Model training stores the feature_version used in the model metadata.

## 5. Aggregation and Backfill

- Raw events are written in real time.
- Aggregation job runs hourly for rolling daily windows, and nightly for weekly rollups.
- Backfill job can recompute aggregates for a time range, keyed by user_id.
- Aggregation should be idempotent and safe to rerun.

## 6. Retention and Privacy

- Raw signal retention: 180 days (configurable).
- Aggregates retention: 24 months (configurable).
- If privacy_level is confidential, do not sync raw metadata; store only minimal counts.
- Redaction endpoint can delete raw events and recompute aggregates.

## 7. Emission Points (Implementation Map)

- Go services: CRUD events for resource, category, todo, reminder.
- UI events: open, dwell, search, graph interactions.
- Sync layer: operation_id and device_id attribution.
- Worker pipelines: AI classification and deep processing completion.

## 8. Decisions (Resolved Questions)

- Dwell capture: record `dwell_ms` only for in-app rendered content (HTML/PDF/image/text viewers). For external opens, emit `resource_opened` without `dwell_ms` and treat any return as a separate `resource_revisited` event.
- Cooccurrence windowing: compute cooccurrence by `session_id`. Define a session as app start or a 30-minute inactivity gap. If `session_id` is missing, bucket by a 30-minute rolling window keyed by `device_id`.
- E2E fields: when E2E sync is enabled, encrypt `entity_id`, `metadata`, and any free-text fields (query_text, intent_label, filters). Keep only `event_type`, a day-bucketed `event_ts`, `entity_type`, `category_id`, `source`, `event_version`, `device_id`, `session_id`, and `privacy_level` in plaintext for aggregation. Use `metadata_enc` for the encrypted payload and leave `metadata_json` empty on sync.
- Category entropy: keep `category_entropy` and `diversity_score` as derived daily user features stored in `gbus_feature_user_daily` and reused by training and runtime.
- Confidence/evidence tracking (Change 16): `gbus_category_features` rows now carry `evidence_count` (count of signals with weight >= `ExplicitIntentWeightThreshold` = 0.5, i.e. "explicit intent" signals) and `confidence` (`MIN(1.0, evidence_count / ConfidenceEvidenceThreshold)`, threshold = 10). Inference/training can use `confidence` to discount category weights backed by little evidence, without needing the full multi-axis signal decomposition or hourly aggregation cadence from sections 2.6/5 — both remain deferred as premature for a single-user, low-volume system.

## 9. Implementation Gap Note (Change 16)

This spec (section 3) has always required `user_id` (required) and `session_id` (optional) on signal events. The Phase 3 scaffold implementation (`internal/gbus`) initially omitted both from `GBUSSignalPayload` and from the `gbus_category_features` / `gbus_resource_features` tables — features were aggregated per-category/resource only, with no user scoping. Change 16 closes this gap:

- `GBUSSignalPayload` gained `user_id` and `session_id` fields, stamped by `SignalEmitter.Emit()` (defaulting to `gbus.DefaultUserID = "local"` and a per-process UUID respectively) when not explicitly set by the caller.
- `gbus_category_features` and `gbus_resource_features` gained a `user_id` column (default `'local'`), added via migration v4 rather than a primary-key rebuild — acceptable for the current single-user app; a real multi-user migration is a known follow-up if/when Phase 2+ multi-user/sync lands.
- The hourly/nightly aggregation cadence (section 5), full daily/weekly aggregate tables (section 4.2), and multi-axis signal decomposition remain **not implemented** — only the raw-signal-to-category/resource-feature aggregation path (`internal/gbus/aggregator.go`, daily ticker) exists today. These are deferred, not abandoned.
