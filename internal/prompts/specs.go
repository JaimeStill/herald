package prompts

const classifySpec = `Respond with a JSON object matching this exact structure:

{
  "markings_found": ["<marking1>", "<marking2>"],
  "rationale": "<explanation>",
  "enhancements": null
}

Field constraints:
- markings_found: Array of distinct marking strings found on this page,
  exactly as they appear in the document. Include the full marking text
  with any caveats (e.g., "SECRET//NOFORN" not just "SECRET").
- rationale: Brief explanation of what security markings were found on
  this page and their significance. Note any conflicts or ambiguities
  with prior page findings if a classification state is provided.
- enhancements: Set to null when the image is clear enough to read all
  markings confidently. When image quality prevents confident reading of
  any markings, provide an object with rendering adjustments:
  {
    "brightness": <80-200, 100=neutral, increase for faded/dark pages>,
    "contrast": <-50 to 50, 0=neutral, increase to sharpen faded markings>,
    "saturation": <80-200, 100=neutral, adjust for color-related issues>
  }
  Only include fields that need adjustment; omit fields that should stay
  at their neutral values.

Behavioral constraints:
- Always respond with valid JSON, no markdown fencing
- Process exactly one page per response
- Report only what you observe on this page
- If prior page findings are provided in the prompt, use them as context
  to identify consistency or conflicts, but do not repeat prior findings
  in markings_found — only include markings visible on the current page`

const enhanceSpec = `Respond with a JSON object matching this exact structure:

{
  "markings_found": ["<marking1>", "<marking2>"],
  "rationale": "<explanation>"
}

Field constraints:
- markings_found: Array of distinct marking strings found on this
  enhanced page, exactly as they appear. Include the full marking text
  with any caveats.
- rationale: Brief explanation of what the enhanced image reveals compared
  to the original assessment. Note any new markings discovered or prior
  findings confirmed by the improved image quality.

Behavioral constraints:
- Always respond with valid JSON, no markdown fencing
- Focus analysis on the enhanced image with improved rendering settings
- Compare findings against the prior page analysis provided in the prompt
- Report only what you observe on the current enhanced page`

const finalizeSpec = `Respond with a JSON object matching this exact structure:

{
  "classification": "<marking>",
  "confidence": "<HIGH|MEDIUM|LOW>",
  "rationale": "<explanation>"
}

Field constraints:
- classification: The overall security classification marking for the
  document, synthesized from all page findings (e.g., UNCLASSIFIED,
  CONFIDENTIAL, SECRET, TOP SECRET, or with caveats like SECRET//NOFORN).
  Apply the highest classification encountered across all pages.
- confidence: Categorical assessment of classification certainty.
  HIGH = markings are clear, consistent, and unambiguous across pages.
  MEDIUM = markings are present but partially obscured or inconsistent.
  LOW = markings are unclear, missing, or contradictory.
- rationale: Comprehensive explanation of the document classification
  synthesized from all page findings. Reference specific page evidence,
  note any cross-page conflicts, and explain how the final classification
  was determined.

Behavioral constraints:
- Always respond with valid JSON, no markdown fencing
- Consider all page findings holistically when determining classification
- Apply the highest classification encountered across all pages
- Never downgrade based on pages with lower or missing markings
- Confidence reflects the overall clarity and consistency across all pages,
  not just the most recent page analyzed`

var specs = map[Stage]string{
	StageClassify: classifySpec,
	StageEnhance:  enhanceSpec,
	StageFinalize: finalizeSpec,
}

// Spec returns the hardcoded specification for a workflow stage.
// Specifications define the expected output format and behavioral constraints.
// Returns ErrInvalidStage if the stage is not recognized.
func Spec(stage Stage) (string, error) {
	text, ok := specs[stage]
	if !ok {
		return "", ErrInvalidStage
	}
	return text, nil
}
