# Phase 3 - GBUS Model Training and Inference Plan

Purpose: Define the GBUS model training pipeline and inference plan for Phase 3.
Scope: Data labeling, feature assembly, baseline model choice, evaluation, model artifacts, and inference integration. Monitoring and retraining cadence are covered in the GBUS governance slice.

## 1. Goals and Assumptions

- Produce a category-level interest profile per user that improves classification, search ranking, and reminder prioritization.
- Start with a reproducible baseline model that beats weighted scoring on offline metrics.
- Prefer batch inference with cached scores for stability; add real-time scoring only if needed.
- Keep a safe fallback to weighted scoring when the model or features are unavailable.

## 2. Prediction Targets

Primary target: user-category affinity score.

- Each training row represents a (user_id, category_id, anchor_day).
- Label is derived from observed high-intent actions in a future window.
- Output score is continuous [0,1] and later calibrated to tiers (low, medium, high).

High-intent actions for positive label:
- manual category assignment or change
- favorite toggle on
- priority set high
- reminder or todo created and completed
- repeated revisits in the window

Negative label examples:
- no actions in the future window
- only passive views without follow-through

## 3. Labeling Strategy

- Use a 30-day future window for positive labeling.
- High-intent score is weighted by action type and recency with exponential decay.
- Maintain a neutral band to avoid noisy negatives (label = null for training).

Initial weights (tuned later):
- 1.0: manual category assign or change
- 0.8: reminder or todo completion
- 0.5: favorite toggle on, priority set high, chat query with category hint
- 0.2: revisits and search result clicks
- 0.05: passive opens without revisit

Recency decay:
- Exponential decay with a 7-day half-life over the 30-day window.

Initial thresholds (tuned after 4 weeks of data):
- high_intent_score >= 0.6 => label 1
- high_intent_score < 0.15 and zero manual actions => label 0
- Otherwise => label null

Calibration plan:
- Recalibrate thresholds each retrain to keep positives in the 15-25% range.
- Reject retrain if label distribution shifts more than +/-5 percentage points across splits.
- Persist thresholds and weights in model metadata for auditability.

## 4. Feature Assembly

Sources:
- gbus_feature_user_category_daily
- gbus_feature_user_category_weekly
- gbus_feature_user_daily
- gbus_feature_category_cooccurrence_daily

Feature groups:
- Activity: saves_count, views_count, revisits_count, dwell_ms_sum
- Intent: manual_moves_count, favorites_count, priority_count
- Outcomes: reminders_created_count, todos_completed_count
- Recency: last_action_at age buckets
- Diversity: category_entropy, diversity_score
- Cooccurrence: top-N cooccurring categories in recent sessions

Feature windows:
- 7d, 30d, 90d rolling windows
- Weekly aggregates for stability

## 5. Baseline Model Choice

- Use gradient boosting (XGBoost) as the baseline model.
- Train and evaluate in a Python pipeline for reproducibility.
- Export model to a JSON/text dump and score with a pure-Go tree model runtime.
- Do not use ONNX or a Python sidecar in Phase 3.

## 6. Training Pipeline

Steps:
1) Extract features and labels from the feature store for a time range.
2) Build time-based train, validation, and test splits.
3) Train baseline model and calibrate probabilities.
4) Evaluate against the weighted scoring baseline.
5) Persist model artifact and metadata.

Data split:
- Train: oldest 70 percent
- Validation: middle 15 percent
- Test: most recent 15 percent

## 7. Evaluation Criteria

Primary metrics:
- PR-AUC (handles sparse positive labels)
- ROC-AUC
- Calibration error (Brier score)

Ranking checks:
- Top-K recall for categories a user later manually assigns
- Lift over weighted scoring baseline in top 5 recommendations

Acceptance threshold (baseline):
- At least +5 percent PR-AUC over weighted scoring
- Non-regression on top-K recall

## 8. Model Artifacts and Metadata

Store artifacts in a model registry table and a local artifacts folder.

Registry fields:
- model_id, model_version, feature_version
- training_window_start, training_window_end
- metrics_json, artifact_path, created_at
- status (active, deprecated, staged)
- labeling_thresholds_json, labeling_weights_json

## 9. Inference Plan

Phase 3 inference uses batch scoring with cached results and event-triggered refresh.

Batch scoring:
- Hourly scoring job computes all active (user_id, category_id) pairs.
- Nightly full rebuild recomputes everything as an idempotent safety net.
- Write to gbus_user_category_scores with a score_ts.

Event-triggered scoring:
- On high-intent signals (manual move, favorite, priority, reminder/todo completion), enqueue an Asynq task to rescore the affected pair.

Staleness guard:
- If score_ts is older than 90 minutes, fall back to weighted scoring.

Fallback behavior:
- If model or features are missing, fall back to weighted scoring.
- Feature flags: gbus.enabled, gbus.training.enabled, gbus.inference.enabled.

## 10. Inference Integration Points

- Classification suggestions: add gbus_affinity to category ranking.
- Search ranking: apply gbus_affinity boost when filters are not strict.
- Reminders: prioritize suggestions in high-affinity categories.

## 11. Open Questions for Next Slice

Open items are limited to tuning and validation, not architecture:

- Confirm threshold tuning after 4 weeks of signals.
- Validate staleness guard and hourly cadence impact on UX.
