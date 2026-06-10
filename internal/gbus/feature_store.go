package gbus

import "selfsystems/internal/domain"

// FeatureStore aliases domain.GBUSFeatureStore so gbus-internal code can
// reference it without re-importing domain everywhere.
type FeatureStore = domain.GBUSFeatureStore

// Re-export domain types for convenience within the gbus package.
type CategoryFeature = domain.GBUSCategoryFeature
type ResourceFeature = domain.GBUSResourceFeature
