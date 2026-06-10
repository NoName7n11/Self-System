package sqlite

import "time"

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func parseRFC3339(value string) time.Time {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return t
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func intToBool(value int) bool {
	return value == 1
}
