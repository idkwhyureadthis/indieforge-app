package dto

import "time"

// FormatTime renders a UTC RFC3339 timestamp for the wire format.
func FormatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// FormatTimePtr is FormatTime for an optional timestamp.
func FormatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := FormatTime(*t)
	return &s
}

// StrPtr returns nil for an empty string, otherwise a pointer to it —
// used so optional fields serialize as JSON null instead of "".
func StrPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// NonNilStrings turns a nil slice into an empty one for stable JSON output.
func NonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
