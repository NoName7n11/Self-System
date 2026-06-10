package http

import (
	"fmt"
	"strings"
)

const (
	maxURLLength          = 2048
	maxIDLength           = 128
	maxTitleLength        = 512
	maxSummaryLength      = 10000
	maxDescriptionLength  = 4000
	maxDetailsLength      = 10000
	maxMessageLength      = 10000
	maxCategoryNameLength = 128
)

func validateLength(fieldName, value string, maxLen int) error {
	if len(strings.TrimSpace(value)) > maxLen {
		return fmt.Errorf("%s is too long (max %d chars)", fieldName, maxLen)
	}
	return nil
}
