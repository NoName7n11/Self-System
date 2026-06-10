package eventstore

// AggregateTypeGBUSSignal is the aggregate type for GBUS behavioural signals.
// These events flow through the unified events table with aggregate_type='gbus_signal'
// and are consumed by GBUS aggregation jobs (Phase 3 GBUS workstream).
// Signal event types follow the taxonomy in Plans/Progress/Phase_3_GBUS_Signals_Feature_Store.md.
const AggregateTypeGBUSSignal = "gbus_signal"

// GBUS signal event type constants.
// These are emitted as side effects of user interactions (resource opens,
// deep processing completion, category overrides, etc.) and consumed by
// the GBUS feature store aggregation pipeline.
const (
	EventTypeGBUSResourceOpened          = "gbus.resource_opened"
	EventTypeGBUSResourceDeepProcessed   = "gbus.resource_deep_processed"
	EventTypeGBUSCategoryOverridden      = "gbus.category_overridden"
	EventTypeGBUSCategoryAccepted        = "gbus.category_accepted"
	EventTypeGBUSSearchPerformed         = "gbus.search_performed"
	EventTypeGBUSChatCommandExecuted     = "gbus.chat_command_executed"
)
