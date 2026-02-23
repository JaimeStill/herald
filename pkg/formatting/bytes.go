// Package formatting provides human-readable formatting and parsing utilities
// for common value types such as byte sizes.
package formatting

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var units = []string{
	"B", "KB", "MB",
	"GB", "TB", "PB",
	"EB", "ZB", "YB",
}

var bytesPattern = regexp.MustCompile(`^(\d+\.?\d*)\s*([A-Za-z]*)$`)

// FormatBytes converts a byte count to a human-readable string using base-1024 units.
// Negative precision values are clamped to zero.
func FormatBytes(n int64, precision int) string {
	if n == 0 {
		return "0 B"
	}

	if precision < 0 {
		precision = 0
	}

	f := float64(n)
	k := 1024.0
	i := int(math.Floor(math.Log(f) / math.Log(k)))

	if i >= len(units) {
		i = len(units) - 1
	}

	size := f / math.Pow(k, float64(i))
	formatted := strconv.FormatFloat(size, 'f', precision, 64)

	return formatted + " " + units[i]
}

// ParseBytes parses a human-readable byte size string (e.g., "50MB") into a byte count.
// Supports units B through YB (base-1024). A bare number with no unit is treated as bytes.
// Unit matching is case-insensitive and an optional space between number and unit is allowed.
func ParseBytes(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty byte size string")
	}

	matches := bytesPattern.FindStringSubmatch(s)
	if matches == nil {
		return 0, fmt.Errorf("invalid byte size: %q", s)
	}

	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid byte size number: %w", err)
	}

	unit := strings.ToUpper(matches[2])

	// bare number with no unit means bytes
	if unit == "" {
		return int64(value), nil
	}

	idx := slices.Index(units, unit)
	if idx == -1 {
		return 0, fmt.Errorf("unknown byte size unit: %q", unit)
	}

	return int64(value * math.Pow(1024, float64(idx))), nil
}
