# Phase 3 Timeline

Purpose: Keep a short running log of implementation steps for Phase 3.
Style: Created or Updated file with reason highlighted.
Format: Created <file> ``because <reason>``

## Session 1 - Phase 3 Documentation Kickoff

- Step 01: Created Plans/Progress/Phase_3_Workstream.md ``because Phase 3 required a detailed implementation-level workstream plan aligned to the GBUS must-have and Phase 3+ scope.``
- Step 02: Created Plans/Progress/Phase_3_Completion_Checklist.md ``because Phase 3 needed explicit exit criteria and evidence tracking similar to Phase 2 governance.``
- Step 03: Created Plans/Progress/Phase_3_Timeline.md ``because Phase 3 execution needs a dedicated running log for implementation, validation, and debugging steps.``

## Session 2 - GBUS Signals and Feature Store Spec

- Step 01: Created Plans/Progress/Phase_3_GBUS_Signals_Feature_Store.md ``because Phase 3 needs a concrete GBUS signal taxonomy and feature store specification before instrumentation begins.``

## Session 3 - GBUS Spec Decisions and Workflow Update

- Step 01: Updated Plans/Project_Workflow_Guide.md ``because Phase 3 work now requires progress tracking against Phase 3 timeline and checklist files.``
- Step 02: Updated Plans/Progress/Phase_3_GBUS_Signals_Feature_Store.md ``because Phase 3 needed resolved decisions for dwell capture, cooccurrence windowing, E2E fields, and entropy features.``

## Session 4 - GBUS Training and Inference Plan

- Step 01: Created Plans/Progress/Phase_3_GBUS_Model_Training_Inference.md ``because Phase 3 needs a concrete training and inference plan for the GBUS behavioral model.``

## Session 5 - GBUS Training and Inference Decisions

- Step 01: Updated Plans/Progress/Phase_3_GBUS_Model_Training_Inference.md ``because Phase 3 required finalized decisions on inference runtime, scoring cadence, and labeling thresholds.``

## Session 6 - ADR Decision Log Initialization

- Step 01: Created Plans/ADR/README.md ``because Phase 3 needed a durable ADR process definition and usage rules.``
- Step 02: Created Plans/ADR/template.md ``because new ADRs need a shared, consistent template.``
- Step 03: Created Plans/ADR/0001-use-wails-for-desktop-shell.md ``because the desktop shell decision needed a durable rationale.``
- Step 04: Created Plans/ADR/0002-use-repository-interfaces-for-storage-abstraction.md ``because storage abstraction needed a durable architectural rationale.``
- Step 05: Created Plans/ADR/0003-use-sqlite-local-and-postgres-for-central-storage.md ``because the local and central storage split needed an explicit record.``
- Step 06: Created Plans/ADR/0004-adopt-last-write-wins-for-phase-2-sync-conflicts.md ``because sync conflict resolution needed an explicit decision log.``
- Step 07: Created Plans/ADR/0005-introduce-dgraph-for-graph-storage.md ``because the original graph storage decision needed historical capture.``
- Step 08: Created Plans/ADR/0006-split-ai-processing-into-skim-and-deep-stages.md ``because the two-tier processing decision needed an explicit record.``
- Step 09: Created Plans/ADR/0007-use-openapi-as-the-http-contract-source.md ``because the API contract source needed explicit documentation.``
- Step 10: Created Plans/ADR/0008-remove-dgraph-from-active-architecture.md ``because the graph storage reversal needed a clean superseding record.``
- Step 11: Created Plans/ADR/0009-adopt-adr-process-for-architectural-decisions.md ``because the ADR workflow itself needed to be formalized.``
- Step 12: Updated Plans/Progress/Phase_3_Workstream.md ``because the ADR initiative needed to be tracked as an important addition.``
- Step 13: Updated Plans/Progress/Phase_3_Completion_Checklist.md ``because evidence tracking should include the ADR decision log.``

## Session 7 - DGraph Removal Cleanup

- Step 01: Updated docker-compose.yml ``because unused DGraph services and volumes were removed from local infrastructure.``
- Step 02: Updated docker-compose.vps.yml ``because DGraph dependencies were removed from the VPS overlay.``
- Step 03: Updated README.md ``because architecture and prerequisites needed to reflect DGraph removal.``
- Step 04: Updated DEPLOYMENT.md ``because the VPS topology and port list no longer include DGraph.``
- Step 05: Updated Plans/Technical_Stack.md ``because stack diagrams, database sections, and tooling lists no longer include DGraph.``
- Step 06: Updated Plans/Outline.md ``because the data-store list and architecture diagrams no longer include DGraph.``
- Step 07: Updated Plans/Development_Workflow.md ``because configuration examples no longer include DGraph.``
- Step 08: Updated .github/copilot-instructions.md ``because architecture notes no longer include DGraph.``
- Step 09: Updated CHANGELOG.md ``because the DGraph service removal needed an Unreleased entry.``
- Step 10: Updated Plans/Progress/Phase_3_Completion_Checklist.md ``because evidence tracking should include the DGraph removal cleanup.``

## Session 8 - Progress Changes Log Initialization

- Step 01: Created Plans/Progress Changes/Changes.md ``because major architecture changes needed an objective/reasoning log.``
- Step 02: Created Plans/Progress Changes/Changes_log.md ``because each change session needs a concise timeline similar to Phase 2 tracking.``
