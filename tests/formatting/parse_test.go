package formatting_test

import (
	"errors"
	"testing"

	"github.com/JaimeStill/herald/pkg/formatting"
)

type sample struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestParse(t *testing.T) {
	t.Run("direct JSON", func(t *testing.T) {
		got, err := formatting.Parse[sample](`{"name":"test","value":42}`)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if got.Name != "test" || got.Value != 42 {
			t.Errorf("Parse = %+v, want {Name:test Value:42}", got)
		}
	})

	t.Run("direct JSON with whitespace", func(t *testing.T) {
		got, err := formatting.Parse[sample](`  {"name":"padded","value":1}  `)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if got.Name != "padded" {
			t.Errorf("Name = %q, want padded", got.Name)
		}
	})

	t.Run("markdown fenced JSON", func(t *testing.T) {
		input := "```json\n{\"name\":\"fenced\",\"value\":7}\n```"
		got, err := formatting.Parse[sample](input)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if got.Name != "fenced" || got.Value != 7 {
			t.Errorf("Parse = %+v, want {Name:fenced Value:7}", got)
		}
	})

	t.Run("markdown fenced without language tag", func(t *testing.T) {
		input := "```\n{\"name\":\"bare\",\"value\":3}\n```"
		got, err := formatting.Parse[sample](input)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if got.Name != "bare" || got.Value != 3 {
			t.Errorf("Parse = %+v, want {Name:bare Value:3}", got)
		}
	})

	t.Run("markdown fenced with surrounding text", func(t *testing.T) {
		input := "Here is the result:\n```json\n{\"name\":\"wrapped\",\"value\":5}\n```\nDone."
		got, err := formatting.Parse[sample](input)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if got.Name != "wrapped" || got.Value != 5 {
			t.Errorf("Parse = %+v, want {Name:wrapped Value:5}", got)
		}
	})

	t.Run("invalid content returns ErrParseFailed", func(t *testing.T) {
		_, err := formatting.Parse[sample]("not json at all")
		if !errors.Is(err, formatting.ErrParseFailed) {
			t.Errorf("error = %v, want ErrParseFailed", err)
		}
	})

	t.Run("empty string returns ErrParseFailed", func(t *testing.T) {
		_, err := formatting.Parse[sample]("")
		if !errors.Is(err, formatting.ErrParseFailed) {
			t.Errorf("error = %v, want ErrParseFailed", err)
		}
	})

	t.Run("invalid JSON in code fence returns ErrParseFailed", func(t *testing.T) {
		input := "```json\n{broken\n```"
		_, err := formatting.Parse[sample](input)
		if !errors.Is(err, formatting.ErrParseFailed) {
			t.Errorf("error = %v, want ErrParseFailed", err)
		}
	})

	t.Run("parses into map type", func(t *testing.T) {
		got, err := formatting.Parse[map[string]any](`{"key":"value"}`)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if got["key"] != "value" {
			t.Errorf("got[key] = %v, want value", got["key"])
		}
	})

	t.Run("parses into slice type", func(t *testing.T) {
		got, err := formatting.Parse[[]int](`[1,2,3]`)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if len(got) != 3 || got[0] != 1 || got[2] != 3 {
			t.Errorf("got = %v, want [1 2 3]", got)
		}
	})
}
