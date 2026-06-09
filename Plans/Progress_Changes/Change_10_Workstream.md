# Change 10 Workstream - GBUS Behavioral Model

Date: 2026-06-08
Status: Complete
Scope: Implement the GBUS (Generalized Behavioral Understanding System) end-to-end: signal capture, feature store aggregation, training pipeline, baseline model, inference integration, and monitoring.

## Objective

Deliver the GBUS behavioral model that learns from user interactions with real resources (not stubs) to improve classification suggestions, search ranking, and reminder priority over time. GBUS must not be built until Changes 6 and 7 are complete — signals are only meaningful once content is real and classification is AI-driven.

## Guiding Constraints

- Changes 6 (content extraction) and 7 (AI intelligence) must be complete before this workstream begins.
- GBUS operates as a hidden backend system — no direct user-facing profile UI.
- Signals must be privacy-safe: no PII in signal payloads, signals are aggregated not raw-exported.
- The weighted scoring system (from the Outline) is the baseline to beat — GBUS must outperform it on offline metrics before replacing it.
- Weighted scoring remains as fallback when GBUS model is unavailable (cold start, retraining, error).
- Use the existing `events` table from Change 3 as the signal store: `aggregate_type = 'gbus_signal'`.
- Aggregation jobs must be bounded in execution time (30s per run) to not block the main runtime.
- Model versioning and rollback must be supported from day one.

## Signal Taxonomy

Signals follow the weighted scheme from the Outline:

| Signal | Weight | Notes |
|--------|--------|-------|
| User manual classification | 1.0 | Strongest — user chose the category |
| User correction / category move | 1.0 | Override signal |
| System auto-classification | 0.5 | Weaker — AI chose |
| Resource shared / saved (no confirm) | 0.3 | Mild interest |
| Resource deleted | 0.1 | Ambiguous |
| Resource revisited | +0.4 | Strong reaffirmation |
| Counter increment (re-save) | +0.2 | Mild reaffirmation |

Additional signals to capture:
- Search query issued (query text + result clicked)
- Resource marked favorite / priority / read
- Reminder snoozed or dismissed
- Deep processing result confirmed vs. overridden

## Workstream 1 — Signal Taxonomy and Instrumentation

Objective:
Define the full signal schema and instrument all emission points in services and UI interactions.

Key tasks:
- [x] Define `GBUSSignalPayload` schema: `signal_type`, `resource_id`, `category_id`, `weight`, `context` (search query, correction detail, etc.).
- [x] Add signal emission to: `ResourceService.Create`, `ResourceService.Update` (category change), `ResourceService.Delete`, `ResourceService.IncrementCounter`, search query handler, chat command handler.
- [x] All signals written as events: `aggregate_type = 'gbus_signal'`, `event_type = 'gbus.<signal_type>'`.
- [x] Signal emission must be async (fire-and-forget) — never blocks the primary operation.
- [x] Add `Plans/Progress/Phase_3_GBUS_Signals_Feature_Store.md` reference as the signal specification source.

Deliverables:
- [x] `internal/gbus/signals.go` — signal type constants and payload schema.
- [x] `internal/gbus/emitter.go` — async signal emitter backed by the event store.
- [x] Updated service files with signal emission calls.
- [x] `internal/gbus/emitter_test.go` — tests for each signal type emission.

Done criteria:
- [x] All 7 core signal types are emitted correctly from their respective services.
- [x] Signal emission is async and does not affect service response latency.
- [x] Signals are stored as events in the `events` table.

## Workstream 2 — Feature Store and Aggregation Jobs

Objective:
Aggregate raw signals into daily/weekly feature tables that the training pipeline and inference engine consume.

Key tasks:
- [x] Schema migrations: `gbus_user_category_features` (category interest scores per day), `gbus_resource_features` (per-resource interaction counts), `gbus_search_features` (query → category click patterns).
- [x] `internal/gbus/aggregator.go` — background job that tails `events` where `aggregate_type = 'gbus_signal'` and computes feature aggregates.
- [x] Run daily (cron-style, bounded to 30s execution).
- [x] Retention policy: raw signals retained 90 days; aggregated features retained indefinitely.
- [x] Privacy redaction: implement signal redaction using the existing event store redaction mechanism (Change 3 P6).

Deliverables:
- [x] Schema migrations for feature tables.
- [x] `internal/gbus/aggregator.go` with bounded daily job.
- [x] `internal/gbus/aggregator_test.go` — tests for aggregate correctness.
- [x] Config flags: `gbus.enabled`, `gbus.retention_days`.

Done criteria:
- [x] Feature tables are populated correctly after a daily aggregation run.
- [x] Aggregation job completes within 30s on 10K signal events.
- [x] Signal redaction removes raw payloads while preserving aggregate counts.

## Workstream 3 — Training Dataset and Baseline Model

Objective:
Build a reproducible training pipeline and train a baseline GBUS model that outperforms the weighted scoring baseline on offline metrics.

Key tasks:
- [x] `scripts/gbus_train/main.go` — dataset extraction: reads feature tables, assembles labeled training data (positive = user confirmed / corrected; negative = user deleted / ignored).
- [x] Define evaluation metric: category suggestion accuracy (top-1 and top-3) vs. weighted scoring baseline.
- [x] Train baseline model: gradient boosting (XGBoost via Go binding or Python subprocess).
- [x] Evaluate: accuracy, precision, recall on held-out validation set.
- [x] Store model artifact: serialized model file + metadata (version, training date, validation metrics).
- [x] Document: minimum accuracy threshold to promote a model to production (default: must beat weighted scoring baseline by ≥ 5%).

Deliverables:
- [x] `scripts/gbus_train/main.go` — training pipeline.
- [x] `models/gbus/` — model artifact storage directory.
- [x] `models/gbus/model_registry.json` — version metadata.
- [x] Training and evaluation report template.

Done criteria:
- [x] Training pipeline is reproducible from the feature tables.
- [x] Baseline model achieves > 5% lift over weighted scoring on validation set.
- [x] Model artifact and metadata are versioned and stored.

## Workstream 4 — Inference Integration

Objective:
Serve GBUS scores at runtime and integrate them into classification suggestions, search ranking, and reminder priority.

Key tasks:
- [x] `internal/gbus/inference.go` — loads model artifact, computes category scores for a resource given its features.
- [x] Integrate into classification: when AI classification returns < 0.85 confidence, consult GBUS scores to bias category suggestion.
- [x] Integrate into search ranking: `SemanticSearch` applies GBUS interest boost (weight 0.5) to results from categories with high user interest scores.
- [x] Integrate into reminder priority: reminders for categories with high GBUS interest score surface first.
- [x] Safe fallback: if model file missing or load fails → fall back to weighted scoring silently.
- [x] Feature flag: `gbus.inference_enabled` — off by default until baseline model passes promotion threshold.

Deliverables:
- [x] `internal/gbus/inference.go` — inference engine with model loading and scoring.
- [x] Updated `internal/service/classifier.go` with GBUS bias integration.
- [x] Updated `internal/service/resource_service.go` (search ranking) with GBUS boost.
- [x] Updated reminder priority sorting with GBUS interest score.
- [x] Tests for fallback behavior when model is unavailable.

Done criteria:
- [x] Classification suggestions are influenced by GBUS scores when model is loaded.
- [x] Search results are re-ranked by GBUS interest scores.
- [x] Safe fallback to weighted scoring works when GBUS is unavailable.
- [x] Feature flag controls activation.

## Workstream 5 — Monitoring and Governance

Objective:
Make GBUS reliable, observable, and rollback-safe.

Key tasks:
- [x] Model registry: `models/gbus/model_registry.json` tracks version, training date, validation accuracy, promotion status (candidate / production / retired).
- [x] Drift detection: daily job compares current model's top-1 accuracy on recent signals against the baseline — alert if drift > 10%.
- [x] Model rollback procedure: documented runbook for reverting to previous model version.
- [x] GBUS metrics endpoint: `GET /api/v1/gbus/health` (auth-gated) — returns active model version, inference latency p50/p99, signal ingestion rate, feature freshness.
- [x] Retraining cadence: document monthly retraining schedule (manual trigger via `go run ./scripts/gbus_train`).

Deliverables:
- [x] Updated `models/gbus/model_registry.json` schema.
- [x] `internal/gbus/monitor.go` — drift detection job.
- [x] `GET /api/v1/gbus/health` endpoint.
- [x] GBUS rollback runbook in `DEPLOYMENT.md` (Section 10).
- [x] Updated `api/openapi.yaml` with GBUS health route.

Done criteria:
- [x] Model version and promotion status are tracked in the registry.
- [x] Drift detection alerts when model accuracy degrades beyond threshold.
- [x] Rollback to prior model version is documented and executable.
- [x] GBUS health endpoint returns current model state.

## Planned Milestones

- [x] Milestone 10A: Signal taxonomy defined and all emission points instrumented (WS1 complete).
- [x] Milestone 10B: Feature store aggregation jobs running and tables populated (WS2 complete).
- [x] Milestone 10C: Baseline model trained and beating weighted scoring (WS3 complete).
- [x] Milestone 10D: Inference integrated into classification, search, and reminders (WS4 complete).
- [x] Milestone 10E: Model registry, drift monitoring, and rollback runbook in place (WS5 complete).

## Change 10 Definition of Done

- [x] All core GBUS signal types are emitted from services and stored as events.
- [x] Daily aggregation jobs populate feature tables from raw signals.
- [x] Baseline model is trained, evaluated, and achieves > 5% lift over weighted scoring.
- [x] GBUS scores influence classification suggestions, search ranking, and reminder priority.
- [x] Safe fallback to weighted scoring works when GBUS is unavailable.
- [x] Model versioning, drift monitoring, and rollback runbook are operational.
- [x] `go test ./...` passes with no regressions.
