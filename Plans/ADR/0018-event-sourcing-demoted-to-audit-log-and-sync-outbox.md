# ADR 0018: Event Sourcing Demoted to Audit Log and Sync Outbox

## Status
Accepted

## Date
2026-06-03

## Decision Date
2026-06-03

## Context

Changes 3–7 implemented an event sourcing infrastructure (append-only `events` table,
projectors, outbox worker, snapshot worker, dual-write feature flag) on the premise
that the event log would become the sole source of truth for all resource state.

A correctness review against the implementation identified a structural gap: the system
appends events for domain mutations (create/update/delete/category-assign) but does not
emit events for AI-derived state — extracted content, embeddings, classification scores,
key points, entities, thumbnails, promoted titles. These are written directly to the
`resources` and `resource_embeddings` tables and to the `extracted_data` JSON column.

This means a replay-from-events rebuild would silently drop everything that makes the
product useful. The event log does not represent the full truth of a resource's state;
the mutable tables do. The documentation and ADRs 0010–0017 overstate the system's
guarantees.

Separately, the review identified five no-regret correctness bugs (Findings 2, 3, 6,
7, 9) that exist regardless of the architecture direction. Four of those (Findings 2,
6, 7, 9) are fixed in the same commit as this ADR.

### The fork

**Option A — Make event sourcing the true source of truth.**
Every extractor result, enrichment output, embedding, classification score, and
thumbnail would become a versioned event with a sync projector. The log becomes
complete and rebuildable. This is architecturally pure but multiplies the cost of every
future feature and is justified only if there is a real requirement for audit,
time-travel, or multi-source replay. For a single-user local KMS, no such requirement
exists yet.

**Option B — Demote event sourcing to audit log and sync outbox.**
The mutable tables (`resources`, `resource_embeddings`, `extracted_data`) are the
declared source of truth. The `events` table records domain mutations (create/update/
delete/category-assign) for sync propagation and audit. AI-derived state is written
directly to the projection tables, as it already is. No rebuild-from-events guarantee
is made or implied.

## Decision

**Option B.** The event log is an audit trail and sync outbox, not a complete source
of truth. The mutable tables are authoritative.

Concretely:

- The `events` table retains all existing mutations (ResourceCreated, ResourceUpdated,
  ResourceDeleted, ResourceCategoryAssigned, and the domain events for categories,
  todos, reminders). These are correct and continue to be used for sync fanout.
- AI-derived state (extracted content, embeddings, classification metadata, key points,
  entities, thumbnails) is written directly to the projection tables. This is not a bug;
  it is the declared behaviour.
- ADRs 0010–0017 remain valid for what they describe (OCC, idempotency, outbox, etc.)
  but the rebuild-from-events guarantee implied by ADR 0013 and the snapshot ADR 0014
  is narrowed: rebuilds restore domain mutations only, not AI-enrichment state.
- The snapshot worker and replay machinery remain in place for the domain event types
  they already handle.
- Finding 1 (AI state absent from the log) is resolved by this ADR — it is now by
  design, not an oversight.
- Finding 5 (snapshot worker that starts but does nothing useful, plus a latent
  Postgres placeholder bug in its candidate query) is resolved by deletion: the
  snapshot worker and its test are removed, the startup wiring is gone, and the
  orphaned `snapshots_created` observability counter is dropped from the events_health
  endpoint. The `projection_snapshots` table and `Store.Snapshot` method remain unused
  but in place (see ADR 0014, now superseded). This is less code than making the
  snapshot path correct for a guarantee we no longer make.
- Finding 3 (two independent sequence spaces in outbox + hub) is addressed: the
  reconnect merge in `ws_handler.go` now dedupes hub events by origin
  (`mergeDurableAndHubReplay` skips only `outbox.worker`-sourced hub events) instead
  of by raw sequence, so a directly-published hub event whose minted sequence collides
  with an unrelated events-table sequence is no longer silently dropped.

## Consequences

Positive:
- Every future feature (content ingestion, embedding, enrichment, GBUS signals) stays
  at its natural complexity — no event + projector + upcaster overhead for each new
  state field.
- The documentation gap closes immediately: the system is now honestly described as
  "audit log + sync outbox" with mutable tables authoritative, which matches the code.
- Findings 2, 6, 7, 9 (correctness bugs in the existing event path) are fixed alongside
  this decision.

Negative:
- Replay-from-events cannot reconstruct AI-enriched state. If the database is lost,
  domain structure (categories, resource URLs, titles, user overrides) is recoverable
  from the log; AI-derived annotations (summaries, embeddings, key points) must be
  re-derived by re-running the extraction and enrichment pipeline.
- If a future requirement for full audit or time-travel emerges, Option A becomes
  necessary and the migration cost will be larger than if it had been done incrementally.

## Alternatives Considered

### Option A — Full event sourcing
Pros:
- Complete audit trail and deterministic rebuild.
- Sync guarantees are stronger: any device can replay to any point in time.

Cons:
- Every AI-derived state field (extracted text, thumbnail, embedding, classification
  confidence, key points, entities) requires a dedicated event type, payload schema,
  upcaster, and sync projector.
- The daily token budget and async enrichment pipeline become significantly more complex
  to model as events (enrichment is non-deterministic and may be retried with different
  models).
- No real requirement justifies this cost for a single-user local application.

## Notes
- This ADR supersedes the rebuild-from-events implication in ADR 0013 and ADR 0014.
  Those ADRs remain valid for domain mutation events; the scope is narrowed.
- This decision was reached after a structured correctness review in session 24.
