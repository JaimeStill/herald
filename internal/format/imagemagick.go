package format

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"github.com/JaimeStill/herald/internal/state"
)

// normalizeArgs is the always-on contrast operator applied whenever
// RenderOptions.Normalize is set. It pulls a faint, low-contrast page toward
// the full tonal range so faded markings survive the vision model's downscale
// to ~768px, while leaving an already-crisp page largely untouched
// (self-limiting). Both the extract and enhance passes enable it so the
// enhance render builds on the same contrast baseline the model first saw;
// because each pass renders fresh from the source PDF, applying it does not
// compound across passes.
var normalizeArgs = []string{"-contrast-stretch", "0.5%x0.5%"}

// RenderOptions configures a single Render invocation. Density and Normalize
// describe the always-on extract pass; Settings carries the targeted enhance
// filters. Normalize and Settings are mutually exclusive in practice (Extract
// normalizes, Enhance applies Settings), but the operator order is fixed so
// any combination behaves predictably.
type RenderOptions struct {
	// Density passes `-density 300`, required for PDF rasterization. Image
	// sources leave this false since they already carry native resolution.
	Density bool
	// Normalize applies the always-on contrast operator (normalizeArgs) for
	// the extract pass. Enhance leaves this false.
	Normalize bool
	// Settings applies brightness/contrast and/or saturation filters when
	// non-nil. Used by Enhance; Extract leaves this nil.
	Settings *state.EnhanceSettings
}

// Render invokes the `magick` CLI to convert src → dst per opts. Operators are
// emitted in a fixed order: `-density` → src → contrast normalization →
// brightness/contrast → saturation → dst. Cancellation propagates via the
// context; errors wrap the magick stderr for diagnostics.
//
// PDF callers pass src as `<tempDir>/source.pdf[N-1]` (magick's native PDF
// page-selector syntax, zero-indexed). Image callers pass the direct file
// path. In both cases dst is the destination PNG path.
func Render(ctx context.Context, src, dst string, opts RenderOptions) error {
	args := make([]string, 0, 10)
	if opts.Density {
		args = append(args, "-density", "300")
	}
	args = append(args, src)

	if opts.Normalize {
		args = append(args, normalizeArgs...)
	}

	if opts.Settings != nil {
		if bc, ok := brightnessContrastArg(opts.Settings); ok {
			args = append(args, "-brightness-contrast", bc)
		}
		if opts.Settings.Saturation != nil {
			args = append(args, "-modulate", fmt.Sprintf("100,%d,100", *opts.Settings.Saturation))
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
