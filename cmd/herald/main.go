// Command herald is a stateless, Azure-CLI-style client for the Herald API:
// each subcommand maps to a single API call and emits the response as JSON.
package main

import (
	"os"

	"github.com/JaimeStill/herald/internal/cli"
)

func main() {
	os.Exit(cli.Main())
}
