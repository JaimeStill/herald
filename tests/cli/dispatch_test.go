package cli_test

import (
	"strings"
	"testing"

	"github.com/JaimeStill/herald/internal/cli"
)

func TestRunNoArgs(t *testing.T) {
	err := cli.Run(nil)
	if err == nil || !strings.Contains(err.Error(), "no command") {
		t.Errorf("got %v, want error containing 'no command'", err)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	err := cli.Run([]string{"frobnicate"})
	if err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("got %v, want error containing 'unknown command'", err)
	}
}

func TestRunVersion(t *testing.T) {
	for _, arg := range []string{"version", "--version", "-v"} {
		t.Run(arg, func(t *testing.T) {
			out := captureStdout(t, func() {
				if err := cli.Run([]string{arg}); err != nil {
					t.Fatalf("run %s: %v", arg, err)
				}
			})
			if !strings.Contains(out, "herald") {
				t.Errorf("output %q does not contain 'herald'", out)
			}
		})
	}
}

func TestRunHelp(t *testing.T) {
	for _, arg := range []string{"help", "--help", "-h"} {
		t.Run(arg, func(t *testing.T) {
			if err := cli.Run([]string{arg}); err != nil {
				t.Errorf("run %s: got %v, want nil", arg, err)
			}
		})
	}
}
