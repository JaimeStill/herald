package formatting_test

import (
	"testing"

	"github.com/JaimeStill/herald/pkg/formatting"
)

func TestParseBytes(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{"bare bytes", "1024", 1024, false},
		{"bytes unit", "512B", 512, false},
		{"kilobytes", "1KB", 1024, false},
		{"megabytes", "50MB", 50 * 1024 * 1024, false},
		{"gigabytes", "2GB", 2 * 1024 * 1024 * 1024, false},
		{"terabytes", "1TB", 1024 * 1024 * 1024 * 1024, false},
		{"lowercase unit", "10mb", 10 * 1024 * 1024, false},
		{"mixed case", "5Gb", 5 * 1024 * 1024 * 1024, false},
		{"with space", "100 MB", 100 * 1024 * 1024, false},
		{"leading whitespace", "  50MB", 50 * 1024 * 1024, false},
		{"trailing whitespace", "50MB  ", 50 * 1024 * 1024, false},
		{"zero", "0", 0, false},
		{"empty string", "", 0, true},
		{"unknown unit", "50XX", 0, true},
		{"no number", "MB", 0, true},
		{"negative", "-5MB", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := formatting.ParseBytes(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseBytes(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseBytes(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name      string
		n         int64
		precision int
		want      string
	}{
		{"zero", 0, 2, "0 B"},
		{"bytes", 500, 0, "500 B"},
		{"one KB", 1024, 0, "1 KB"},
		{"one MB", 1024 * 1024, 0, "1 MB"},
		{"one GB", 1024 * 1024 * 1024, 0, "1 GB"},
		{"50 MB", 50 * 1024 * 1024, 0, "50 MB"},
		{"fractional MB", 1536 * 1024, 1, "1.5 MB"},
		{"negative precision clamped to zero", 1024, -1, "1 KB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatting.FormatBytes(tt.n, tt.precision)
			if got != tt.want {
				t.Errorf("FormatBytes(%d, %d) = %q, want %q", tt.n, tt.precision, got, tt.want)
			}
		})
	}
}

func TestParseBytesFormatBytesRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input int64
	}{
		{"1 KB", 1024},
		{"50 MB", 50 * 1024 * 1024},
		{"1 GB", 1024 * 1024 * 1024},
		{"1 TB", 1024 * 1024 * 1024 * 1024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := formatting.FormatBytes(tt.input, 0)
			parsed, err := formatting.ParseBytes(formatted)
			if err != nil {
				t.Fatalf("round-trip failed: FormatBytes(%d) = %q, ParseBytes error: %v", tt.input, formatted, err)
			}
			if parsed != tt.input {
				t.Errorf("round-trip mismatch: %d → %q → %d", tt.input, formatted, parsed)
			}
		})
	}
}
