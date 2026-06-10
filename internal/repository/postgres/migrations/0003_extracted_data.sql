-- Change 6: Add extracted_data column to resources for content extraction pipeline.
ALTER TABLE resources ADD COLUMN IF NOT EXISTS extracted_data TEXT NOT NULL DEFAULT '{}';
