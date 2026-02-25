package prompts

const classifySpec = `Respond with a JSON object matching this exact structure:

{
  "classification": "<marking>",
  "confidence": "<HIGH|MEDIUM|LOW>",
  "markings_found": ["<marking1>", "<marking2>"],
  "rationale": "<explanation>",
  "page_number": <number>,
  "image_quality_limiting": <true|false>
}

Field constraints:
- classification: The overall security classification marking for the document
  as assessed through this page (e.g., UNCLASSIFIED, CONFIDENTIAL, SECRET,
  TOP SECRET, or with caveats like SECRET//NOFORN).
- confidence: Categorical assessment of classification certainty.
  HIGH = markings are clear, consistent, and unambiguous.
  MEDIUM = markings are present but partially obscured or inconsistent.
  LOW = markings are unclear, missing, or contradictory.
- markings_found: Array of distinct marking strings found on this page,
  exactly as they appear in the document.
- rationale: Brief explanation of how the classification was determined
  from the visible markings. Note any conflicts or ambiguities.
- page_number: The 1-indexed page number being analyzed.
- image_quality_limiting: Whether image quality prevented confident
  reading of any markings on this page. true triggers potential
  enhancement processing.

Behavioral constraints:
- Always respond with valid JSON, no markdown fencing
- Process exactly one page per response
- Accumulate classification state: if a prior state is provided in the
  prompt, update it based on this page's findings
- Apply the highest classification encountered across all pages
- Never downgrade a classification based on a subsequent page`

const enhanceSpec = `Respond with a JSON object matching this exact structure:

{
  "classification": "<marking>",
  "confidence": "<HIGH|MEDIUM|LOW>",
  "markings_found": ["<marking1>", "<marking2>"],
  "rationale": "<explanation>",
  "page_number": <number>,
  "image_quality_limiting": <true|false>,
  "enhanced": true,
  "prior_confidence": "<HIGH|MEDIUM|LOW>"
}

Field constraints:
- All fields from the classify specification apply, plus:
- enhanced: Always true in enhancement responses.
- prior_confidence: The confidence level from the classify stage that
  triggered enhancement. Used to track whether enhancement improved
  the assessment.

Behavioral constraints:
- Always respond with valid JSON, no markdown fencing
- Focus analysis on re-rendered pages with improved image settings
- Compare findings against the prior classification state
- Maintain or upgrade classification; never downgrade
- If enhanced images reveal no new information, preserve the prior
  classification and note this in the rationale`

var specs = map[Stage]string{
	StageClassify: classifySpec,
	StageEnhance:  enhanceSpec,
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
