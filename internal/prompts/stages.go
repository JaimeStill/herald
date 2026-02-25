package prompts

import (
	"encoding/json"
	"slices"
)

// Stage represents a workflow stage that a prompt override targets.
type Stage string

// Valid workflow stages.
const (
	StageInit     Stage = "init"
	StageClassify Stage = "classify"
	StageEnhance  Stage = "enhance"
)

var stages = []Stage{
	StageInit,
	StageClassify,
	StageEnhance,
}

// Stages returns the list of valid workflow stages.
func Stages() []Stage {
	return stages
}

// UnmarshalJSON validates that the decoded string is a known stage value.
func (s *Stage) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	v := Stage(raw)
	if !slices.Contains(stages, v) {
		return ErrInvalidStage
	}
	*s = v
	return nil
}
