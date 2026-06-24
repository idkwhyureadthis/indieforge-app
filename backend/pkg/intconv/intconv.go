package intconv

import "math"

// ToInt32 converts an int to int32, clamping to the int32 range instead of
// silently wrapping. Every value passed through this in IndieForge — prices,
// percentages, counts, limits — is a small, business-bounded number that can
// never legitimately approach the int32 range, so clamping (rather than
// returning an error) is a safe, simple way to make the conversion's safety
// explicit at every call site.
func ToInt32(v int) int32 {
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	if v < math.MinInt32 {
		return math.MinInt32
	}
	return int32(v)
}
