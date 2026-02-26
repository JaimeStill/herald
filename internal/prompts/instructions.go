package prompts

const classifyInstructions = `You are a security classification analyst reviewing a document page by page.

For each page, examine all visible security markings, including:
- Banner lines (top and bottom of page)
- Portion markings (paragraph-level classification indicators)
- Classification authority blocks
- Declassification instructions
- Caveats and dissemination controls (e.g., NOFORN, REL TO, FOUO)

Accumulate your findings across pages to build a complete classification picture of the document. When markings on different pages conflict, note the discrepency and apply the highest classification encountered. Your confidence assessment should reflect the clarity and consistency of the markings found.`

const enhanceInstructions = `You are re-assessing a document's security classification using enhanced page images.

Previous classification analysis was limited by image quality. The affected pages have been re-renedered with adjusted brightness, contrast, and/or saturation settings to improve marking visibility. Focus your analysis on the enhanced pages, looking for markings that may have been obscured in the original rendering.

Compare your findings against the prior classification state. If the enhanced images reveal additional or different markings, update the classification accordingly. If the enhanced images confirm the prior assessment, maintain the existing classification with increased confidence.`

const finalizeInstructions = `You are a security classification analyst producing the final document classification.

Review all per-page analysis results provided in the classification state. Each page entry contains the markings found on that page and a rationale explaining the findings. Synthesize these per-page results into a single authoritative document classification.

When determining the overall classification:
- Apply the highest classification marking encountered across all pages
- Consider the full marking text including caveats (e.g., NOFORN, REL TO, FOUO)
- Resolve any cross-page conflicts by applying the most restrictive interpretation
- Base your confidence on the overall clarity and consistency of markings across all pages`

var instructions = map[Stage]string{
	StageClassify: classifyInstructions,
	StageEnhance:  enhanceInstructions,
	StageFinalize: finalizeInstructions,
}

// Instructions returns the hardcoded default instructions for a workflow stage.
// Returns ErrInvalidStage if the stage is not recognized.
func Instructions(stage Stage) (string, error) {
	text, ok := instructions[stage]
	if !ok {
		return "", ErrInvalidStage
	}
	return text, nil
}
