package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

// version is the CLI version, overridable at build time via
// -ldflags "-X github.com/JaimeStill/herald/internal/cli.version=vX.Y.Z".
var version = "dev"

// command handles one CLI subcommand, receiving the args that follow the command
// tokens.
type command func(ctx context.Context, args []string) error

// commands maps a command path ("group command", or a bare command) to its
// handler.
var commands = map[string]command{
	"documents upload":            documentsUpload,
	"documents list":              documentsList,
	"documents get":               documentsGet,
	"classify":                    classify,
	"classifications list":        classificationsList,
	"classifications get":         classificationsGet,
	"classifications by-document": classificationsByDocument,
	"settings show":               settingsShow,
	"settings secret":             settingsSecret,
}

// Main runs the CLI and returns a process exit code. A help request (-h) exits
// 0 without an error message; any other error is printed to stderr.
func Main() int {
	err := Run(os.Args[1:])
	switch {
	case err == nil, errors.Is(err, flag.ErrHelp):
		return 0
	default:
		fmt.Fprintln(os.Stderr, "herald: "+err.Error())
		return 1
	}
}

// Run dispatches args to a command. It installs a signal-cancelled context so
// in-flight requests abort on SIGINT/SIGTERM. Commands are matched first as a
// two-token "group command" path, then as a bare command.
func Run(args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if len(args) == 0 {
		usage()
		return fmt.Errorf("no command given")
	}

	switch args[0] {
	case "version", "--version", "-v":
		fmt.Println("herald " + version)
		return nil
	case "help", "--help", "-h":
		usage()
		return nil
	}

	if len(args) >= 2 {
		if cmd, ok := commands[args[0]+" "+args[1]]; ok {
			return cmd(ctx, args[2:])
		}
	}
	if cmd, ok := commands[args[0]]; ok {
		return cmd(ctx, args[1:])
	}

	usage()
	return fmt.Errorf("unknown command: %s", strings.Join(args, " "))
}

func usage() {
	fmt.Fprint(os.Stderr, `herald — Herald API client

Usage:
  herald <command> [flags] [args]

Commands:
  documents upload                            upload a file (--file --external-id --platform)
  documents list                              list documents (filters, --page, --page-size)
  documents get <id>                          get a document
  classify <document-id>                      classify a document (streams to completion)
  classifications list                        list classifications (filters, --page, --page-size)
  classifications get <id>                    get a classification
  classifications by-document <document-id>   get a document's classification
  settings show                               print the fully resolved settings (--show-secrets to reveal)
  settings secret [<secret>|-]                write the client secret to ~/.herald/secrets.json (stdin if omitted)
  version                                     print the CLI version

Global flags (also HERALD_CLI_* env):
  --api <url>          Herald API base URL
  --scope <scope>      Entra token scope
  --timeout <dur>      per-request timeout
  --output json|jsonl  output format
  auth via HERALD_CLI_AUTH_* env / settings.json

Config is resolved from ~/.herald/ then the working directory, then env, then flags.

Run 'herald <command> -h' for command-specific flags.
`)
}
