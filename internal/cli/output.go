package cli

import (
	"encoding/json"
	"fmt"
	"os"
)

// emit writes a command's result to stdout: indented JSON for OutputJSON, or a
// single compact line for OutputJSONL.
func emit[T any](format OutputFormat, item T) error {
	enc := json.NewEncoder(os.Stdout)
	if format != OutputJSONL {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(item); err != nil {
		return fmt.Errorf("encode result: %w", err)
	}
	return nil
}
