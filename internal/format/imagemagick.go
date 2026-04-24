package format

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"github.com/JaimeStill/herald/internal/state"
)

// Render invokes the `magick` CLI to convert src → dst. When density is
// true, passes `-density 300` (required for PDF rasterization — image
// sources should pass false since they already have native resolution).
// When settings is non-nil, applies brightness/contrast and/or saturation
// filters in a single pass. Cancellation propagates via the context; errors
// wrap the magick stderr for diagnostics.
//
// PDF callers pass src as `<tempDir>/source.pdf[N-1]` (magick's native PDF
// page-selector syntax, zero-indexed). Image callers pass the direct file
// path. In both cases dst is the destination PNG path.
func Render(
	ctx context.Context,
	src, dst string,
	density bool,
	settings *state.EnhanceSettings,
) error {
	args := make([]string, 0, 8)
	if density {
		args = append(args, "-density", "300")
	}
	args = append(args, src)

	if settings != nil {
		if bc, ok := brightnessContrastArg(settings); ok {
			args = append(args, "-brightness-contrast", bc)
		}
		if settings.Saturation != nil {
			args = append(args, "-modulate", fmt.Sprintf("100,%d,100", *settings.Saturation))
		}
	}

	args = append(args, dst)

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "magick", args...)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("magick %s: %w: %s", src, err, stderr.String())
	}
	return nil
}

// brightnessContrastArg assembles the paired `brightness,contrast` argument
// that magick's -brightness-contrast operator expects. Either component
// being set is enough to emit the argument; the unset side defaults to 0
// (no change). Returns ("", false) when neither is set, signaling the
// caller to omit the operator entirely.
func brightnessContrastArg(s *state.EnhanceSettings) (string, bool) {
	if s.Brightness == nil && s.Contrast == nil {
		return "", false
	}
	b, c := 0, 0
	if s.Brightness != nil {
		b = *s.Brightness
	}
	if s.Contrast != nil {
		c = *s.Contrast
	}
	return fmt.Sprintf("%d,%d", b, c), true
}
