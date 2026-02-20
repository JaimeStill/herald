package openapi

import (
	"encoding/json"
	"os"
)

// MarshalJSON serializes the spec to indented JSON bytes.
func MarshalJSON(spec *Spec) ([]byte, error) {
	return json.MarshalIndent(spec, "", "  ")
}

// WriteJSON serializes the spec to indented JSON and writes it to the given file.
func WriteJSON(spec *Spec, filename string) error {
	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}
