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
  with any caveats (e.g., "SECRET//NOFORN" not just "SECRET"). Include
  declassification exemption category markings (e.g., X1-X8, 25X1, 50X1-HUM)
  and legacy dissemination controls (e.g., WNINTEL) exactly as written. Only
  record a marking when it is associated with a base classification; do not
  record document headings, form names, organization names, titles,
  addresses, or dates (e.g., "Defense Office Staff Routing Sheet") as
  markings. A specific declassification date (a calendar date or YYYYMMDD such
  as 20280901, or text like "JANUARY 2004") is a declassification instruction,
  not part of the marking — do not append it to the marking value (record
  "SECRET//NOFORN", not "SECRET//NOFORN//20280901"); you may note it in the
  rationale.
- rationale: Brief explanation of what security markings were found on
  this page and their significance. Note any conflicts or ambiguities
  with prior page findings if a classification state is provided. If you
  resolved a degraded or deformed marking to its valid value, note what you
  observed and what you resolved it to.
- enhancements: Set to null only when every marking region is clear enough to
  read confidently AND no marking position shows faint or partial text you
  cannot fully read. Marking quality varies between documents; where a document
  is degraded, markings may be faded, weak, or only partially legible and easy
  to miss — do not assume that reading one marking clearly means there are no
  others. When image quality prevents confident reading of any marking, or a
  marking position shows faint or degraded text that could be an unread marking,
  provide an object with rendering adjustments:
  {
    "brightness": <80-200, 100=neutral; raise for dark/underexposed pages,
                   lower to darken faint gray markings on a light background>,
    "contrast": <-50 to 50, 0=neutral, increase to sharpen faint markings>,
    "saturation": <80-200, 100=neutral, adjust for color-related issues>
  }
  Only include fields that need adjustment; omit fields that should stay
  at their neutral values. Redaction (intentionally blacked-out content) is
  normal and is NOT an image-quality problem — do not request enhancement to
  "recover" redacted regions. Set enhancements when faint, faded, low-contrast,
  or partial text in a marking position prevents you from confidently reading a
  marking.

Behavioral constraints:
- Always respond with valid JSON, no markdown fencing
- Process exactly one page per response
- Report only what you observe on this page
- Report markings as found; do not modernize or convert legacy markings
- Resolve degraded or deformed markings to the nearest valid marking confirmed
  by context; never record a garbled or impossible token (e.g., "NOFOP?Y") as a
  marking, and never invent a nonexistent control
- Exclude any token that is not part of an actual security marking tied to a
  base classification
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
  with any caveats. Include declassification exemption category markings
  (e.g., X1-X8, 25X1, 50X1-HUM) and legacy dissemination controls (e.g.,
  WNINTEL) exactly as written. Only record a marking when it is associated
  with a base classification; do not record document headings, form names,
  organization names, titles, addresses, or dates as markings. A specific
  declassification date (a calendar date or YYYYMMDD such as 20280901) is a
  declassification instruction, not part of the marking — do not append it to
  the marking value; you may note it in the rationale.
- rationale: Brief explanation of what the enhanced image reveals compared
  to the original assessment. Note any new markings discovered or prior
  findings confirmed by the improved image quality. If you resolved a degraded
  marking to its valid value, note what you observed and what you resolved it to.

Behavioral constraints:
- Always respond with valid JSON, no markdown fencing
- Focus analysis on the enhanced image with improved rendering settings
- Compare findings against the prior page analysis provided in the prompt
- Report only what you observe on the current enhanced page
- markings_found replaces the prior findings for this page — include every
  marking the prior pass read confidently plus any the enhancement newly reveals;
  never drop a previously-clear marking because the enhanced render obscured it
- Report markings as found; do not modernize or convert legacy markings
- Resolve degraded or deformed markings to the nearest valid marking confirmed
  by context; never record a garbled or impossible token (e.g., "NOFOP?Y") as a
  marking, and never invent a nonexistent control
- Exclude any token that is not part of an actual security marking tied to a
  base classification`

const finalizeSpec = `Respond with a JSON object matching this exact structure:

{
  "classification": "<overall banner marking>",
  "confidence": "<HIGH|MEDIUM|LOW>",
  "rationale": "<explanation>"
}

Field constraints:
- classification: The document's highest cumulative classification, assembled by
  combining markings across ALL pages — never copied from a single page:
    1. Base level: the single highest classification level found on any page,
       spelled out in uppercase (UNCLASSIFIED, CONFIDENTIAL, SECRET, TOP SECRET).
    2. Controls: the union of every caveat, dissemination/handling control, and
       declassification/exemption marking that is associated with a classification
       marking on any page. Preserve each exactly as marked, including legacy
       markings (e.g., WNINTEL, X1-X8); do not convert, modernize, or drop them.
  Every control must be tied to the base classification — never include a token
  that is not part of an actual security marking (headings, form/org names,
  titles, addresses, and dates are not markings).
  Order the components using this structure (omit categories that are absent):
    CLASSIFICATION//SCI//SAP//AEA//FGI//DISSEMINATION CONTROLS//DECLASS-OR-OTHER
  Separators:
    - "//" separates marking categories
    - "/"  separates multiple controls within the same category
    - "-"  links a control to its sub-control (e.g., SI-G, RD-N)
    - a space separates multiple sub-markings; ", " separates REL TO country codes
  Dissemination controls order: FOUO, ORCON, IMCON, NOFORN, PROPIN, REL TO,
  RELIDO, FISA, DISPLAY ONLY. Declassification exemption category markings
  (e.g., X1-X8, 25X1, 50X1-HUM) trail as the final category after "//". Do NOT
  include specific declassification dates (calendar dates or YYYYMMDD such as
  20280901) in the classification — a date is a declassification instruction
  that belongs to the authority block, not the banner; note it in the rationale
  if relevant. Place legacy/unrecognized controls in the position they appear
  relative to recognized controls.
  Example: pages marked SECRET, SECRET//NOFORN, SECRET NOFORN WNINTEL, and
  SECRET//NOFORN//X1 combine to: SECRET//NOFORN/WNINTEL//X1
- confidence: Certainty in READING the security markings themselves (banner lines
  and portion marks) — not cross-page uniformity, and not the state of the page body.
  HIGH = the markings are legible and the classification is directly determinable
  with no guessing — even if markings differ page to page (escalation is normal)
  and even if the page body is heavily redacted.
  MEDIUM = some markings are partially obscured and require an educated guess to
  read, but you are reasonably confident the inferred value is correct.
  LOW = substantial degradation — markings are largely illegible, missing, or
  genuinely contradictory, and you are not confident any inferred value is correct.
  Redaction (blacked-out content) is a normal, valid state for these documents and
  is NOT illegibility — it never lowers confidence and never affects the
  classification. Pages with no markings (blank, redacted, or routing/cover pages)
  and pages that escalate the markings do not by themselves lower confidence.
- rationale: Comprehensive explanation citing specific page evidence — which
  control came from which page and how the cumulative banner was assembled.

Behavioral constraints:
- Always respond with valid JSON, no markdown fencing
- Assemble the banner from the union of all valid page markings; never copy a
  single page's banner as the final answer
- Every caveat/control/exemption must be associated with a base classification;
  discard anything that is not an actual security marking
- Never drop a control or exemption that appears on any page, including legacy
  markings; report markings as found without modernizing them
- Do not union corrupted or invalid control tokens; if a page lists a degraded
  reading of a valid marking present elsewhere (e.g., "NOFOPI" vs "NOFORN"), use
  the valid marking
- Exclude specific declassification dates from the classification string; keep
  exemption category markings (e.g., X1) and note any dropped dates in the
  rationale
- Confidence reflects legibility of the markings, not whether pages match`

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
