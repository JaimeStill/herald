package formatting

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ErrParseFailed is returned when content cannot be parsed as JSON,
// either directly or from a markdown code fence.
var ErrParseFailed = errors.New("failed to parse response")

var jsonBlockRegex = regexp.MustCompile(`(?s)` + "```" + `(?:json)?\s*\n?(.*?)\n?` + "```")

// Parse attempts to unmarshal content as JSON into T.
// If direct parsing fails, it extracts JSON from a markdown code fence
// and retries. Returns ErrParseFailed if both attempts fail.
func Parse[T any](content string) (T, error) {
	var result T
	content = strings.TrimSpace(content)

	if err := json.Unmarshal([]byte(content), &result); err == nil {
		return result, nil
	}

	matches := jsonBlockRegex.FindStringSubmatch(content)
	if len(matches) >= 2 {
		cleaned := strings.TrimSpace(matches[1])
		if err := json.Unmarshal([]byte(cleaned), &result); err == nil {
			return result, nil
		}
	}

	return result, fmt.Errorf("%w: %s", ErrParseFailed, content)
}
