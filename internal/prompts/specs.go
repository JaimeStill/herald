package prompts

const classifySpec = `Respond with a JSON object matching this exact structure:

{
  "markings_found": ["<marking1>", "<marking2>"],
  "undetermined_markings": ["<description of an unreadable marking position>"],
  "confidence": "<HIGH|MEDIUM|LOW>",
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
  rationale. Record here only markings you can read confidently; a marking
  position you can see is present but cannot read goes in undetermined_markings,
  not here. Record a classification only when you can actually see and read the
  marking itself — never infer or assume one from the page's type, purpose,
  headers, or context (for example, do not assume a routing sheet, cover page, or
  transmittal is UNCLASSIFIED, and do not default an unmarked or heavily redacted
  page to any classification). If the page bears no legible classification
  marking, return an empty array ([]).
- undetermined_markings: Array describing each marking position that is clearly
  PRESENT on the page but whose value you cannot read confidently (e.g., "faint
  bottom banner", "faded stamp at bottom-center", "smeared portion mark on
  paragraph 3"). Check every banner position independently — a top AND a bottom
  banner position where text is present but cannot be resolved each belongs here,
  even when the other banner reads cleanly. Leave empty ([]) when every marking
  is legible. A non-empty list means a security marking
  exists but its value is undetermined — also set enhancements so the page can be
  re-examined. Only list something here when it is plausibly a SECURITY MARKING —
  a classification/control banner, portion mark, or classification/declassification
  stamp in a marking position. Do NOT list incidental handwriting, annotations,
  routing notes, signatures, or margin scribbles, even when they contain
  marking-like fragments (e.g., a handwritten "WN"); these are not security
  markings and do not belong here. Do not list redacted (blacked-out) regions;
  redaction is not an unread marking.
- confidence: Your certainty in READING this page's security markings — not the
  state of the page body, and not whether a marking is present.
  HIGH = every marking's value is directly determinable with no guessing. A
  degraded or partially obscured marking still counts as HIGH when its value is
  confirmed by a clearer instance of the same marking elsewhere on the page (e.g.,
  a garbled header banner matched by a clear footer banner), by unambiguous
  prior-page context, or by the closed marking vocabulary — resolving from a clear
  copy is reading, not guessing.
  MEDIUM = you had to make a genuine educated guess to read a marking whose value
  is NOT confirmed by any clearer instance or context, but you are reasonably
  confident the inferred value is correct.
  LOW = a marking is present but you cannot determine its value (it belongs in
  undetermined_markings), or this page's markings are largely illegible,
  contradictory, or missing where one is clearly expected.
  Redaction (blacked-out content) is a normal, valid state and never lowers
  confidence, and neither does illegible non-marking content — incidental
  handwriting, annotations, routing notes, signatures, smudges, or marginalia.
  Confidence reflects only the legibility of actual security markings; when those
  are legible, the page is HIGH even if other handwritten or faint marks on the
  page cannot be read. A page with no markings at all (blank, redacted, or
  routing/cover page) is HIGH — there is nothing illegible about it.
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
  by context; never record a garbled or impossible token (e.g., "NOFOP?Y" ->
  "NOFORN", "XI"/"Xl" -> "X1") as a marking, and never invent a nonexistent control
- Exclude any token that is not part of an actual security marking tied to a
  base classification
- If prior page findings are provided in the prompt, use them as context
  to identify consistency or conflicts, but do not repeat prior findings
  in markings_found — only include markings visible on the current page`

const enhanceSpec = `Respond with a JSON object matching this exact structure:

{
  "resolved_markings": ["<marking now readable after enhancement>"],
  "undetermined_markings": ["<description of a marking still unreadable>"],
  "confidence": "<HIGH|MEDIUM|LOW>",
  "rationale": "<explanation>"
}

Field constraints:
- resolved_markings: Array of marking strings the enhanced image now lets you
  read confidently but the original pass could not — exactly as they appear, full
  text with any caveats, including declassification exemption category markings
  (e.g., X1-X8, 25X1, 50X1-HUM) and legacy dissemination controls (e.g., WNINTEL)
  as written. Only markings associated with a base classification; never headings,
  form names, organization names, titles, addresses, or dates. A specific
  declassification date (a calendar date or YYYYMMDD such as 20280901) is a
  declassification instruction, not part of the marking — do not append it. This
  list ADDS to the page: the markings the prior pass already read confidently are
  preserved automatically, so do NOT re-list them here. Leave empty ([]) when the
  enhancement reveals nothing new.
- undetermined_markings: Array describing each marking position still clearly
  PRESENT but whose value you cannot read confidently even after enhancement
  (e.g., "faded stamp at bottom-center"). Leave empty ([]) when every targeted
  marking is now legible OR when the targeted element turns out not to be a
  security marking at all. Only list plausible SECURITY MARKINGS here — never
  incidental handwriting, annotations, routing notes, signatures, or margin
  scribbles, even if they contain marking-like fragments. Do not list redacted
  (blacked-out) regions.
- confidence: Your certainty in READING this page's markings after enhancement.
  HIGH = the marking's value is directly determinable. A faint marking the
  enhancement clearly recovers is HIGH, and so is a degraded marking whose value
  is confirmed by a clearer instance of the same marking elsewhere on the page,
  unambiguous prior-page context, or the closed marking vocabulary — resolving
  from a clear copy is reading, not guessing.
  MEDIUM = you had to make a genuine educated guess to read a marking whose value
  is NOT confirmed by any clearer instance or context, but you are reasonably
  confident the value is correct.
  LOW = a marking is still present but undeterminable even after enhancement (it
  belongs in undetermined_markings), or this page's markings remain largely
  illegible or contradictory.
  Redaction (blacked-out content) never lowers confidence, and neither does
  illegible non-marking content (handwriting, annotations, signatures, smudges,
  marginalia). If enhancement shows the targeted faint element is not a security
  marking — so undetermined_markings is now empty — and the page's actual markings
  are legible, the page is HIGH; do not stay LOW over an element that was not a
  marking.
- rationale: Brief explanation of what the enhanced image reveals compared to the
  original assessment — which marking was recovered, which remains unreadable, and
  how you resolved any degraded reading to its valid value.

Behavioral constraints:
- Always respond with valid JSON, no markdown fencing
- Focus analysis on the enhanced image with improved rendering settings
- Compare findings against the prior page analysis provided in the prompt
- Report only what you observe on the current enhanced page
- resolved_markings ADDS to the page; the prior-confident markings are preserved
  by the workflow — do not re-list them, and never ask to remove a prior marking.
  If the enhancement shows a prior marking was a misread, put the corrected value
  in resolved_markings
- Report markings as found; do not modernize or convert legacy markings
- Resolve degraded or deformed markings to the nearest valid marking confirmed
  by context; never record a garbled or impossible token (e.g., "NOFOP?Y" ->
  "NOFORN", "XI"/"Xl" -> "X1") as a marking, and never invent a nonexistent control
- Exclude any token that is not part of an actual security marking tied to a
  base classification`

const finalizeSpec = `Respond with a JSON object matching this exact structure:

{
  "classification": "<overall banner marking>",
  "rationale": "<explanation>"
}

The "classification" value is the document's overall BANNER LINE, assembled by combining the
markings across ALL pages per DoD marking policy (DoDM 5200.01, Volume 2). Build it from
components — never copy a single page's banner verbatim. Escalation across pages (markings
building up page to page) is expected and is not a conflict.

Banner structure and category order (omit any category that is absent):
  CLASSIFICATION//SCI//SAP//AEA//FGI//DISSEMINATION CONTROLS//OTHER DISSEMINATION CONTROLS
Separators:
  - "//" separates marking categories
  - "/"  separates multiple markings within the same category
  - "-"  links a marking to its sub-control/compartment (e.g., SI-G, RD-N, SAR-BP, ACCM-<NICKNAME>)
  - a space separates multiple sub-markings; ", " separates REL TO / DISPLAY ONLY country codes

1. Base classification level: the SINGLE HIGHEST level on any page
   (TOP SECRET > SECRET > CONFIDENTIAL > UNCLASSIFIED), spelled out in uppercase, never
   abbreviated, exactly one level. A document whose pages bear no classified marking is
   UNCLASSIFIED.

2. Category contents, each listing multiple entries in the order shown:
   - SCI: control systems (e.g., HCS, SI, TK), alphabetical; compartments via "-". If HCS or
     TK appears, NOFORN must also appear.
   - SAP: SAR-<NICKNAME>; multiple programs -> SAR-MULTIPLE PROGRAMS; add WAIVED if marked.
   - AEA: RESTRICTED DATA (or RD) / FORMERLY RESTRICTED DATA (or FRD) — these are AEA categories,
     NOT dissemination controls. CNWDI is RD-N; SIGMA is RD-SIGMA <n>. If any portion is RD/FRD it
     MUST appear here, and the document has no automatic declassification.
   - FGI: FGI <country/org codes, alphabetical> (or FGI NATO).
   - DISSEMINATION CONTROLS, in this order: ORCON, IMCON, NOFORN, PROPIN, REL TO, RELIDO,
     DISPLAY ONLY, FISA.
   - OTHER DISSEMINATION CONTROLS, trailing, in this order: SPECAT, NC2-ESI, ACCM-<NICKNAME>,
     EXDIS, NODIS.

Combination rules (apply when assembling — a naive union is WRONG):
   - NOFORN and REL TO/RELIDO are MUTUALLY EXCLUSIVE. If NOFORN appears anywhere, the banner uses
     NOFORN and REL TO/RELIDO are dropped from it (NOFORN takes precedence).
   - REL TO appears ONLY if the ENTIRE document is releasable to the listed partners — every
     classified portion must carry REL TO and the country lists must intersect; use the common
     set. If any classified portion is uncaveated (no REL TO/RELIDO/NOFORN) or the lists do not
     intersect, drop REL TO. Country order: USA first, then remaining countries alphabetical, then
     coalition/organization codes alphabetical.
   - FOUO/CUI is NOT carried into a classified banner; it appears only when the overall document
     is UNCLASSIFIED (UNCLASSIFIED//FOUO).
   - DISPLAY ONLY may not co-occur with RELIDO or NOFORN.
   - Level limits: IMCON is SECRET-level; ORCON, RELIDO, RD, and FRD apply only to TS/S/C.

Excluded from the banner:
   - Declassification instructions and exemption markings — specific dates (e.g., YYYYMMDD),
     events, exemption categories (25X1-25X9, 50X1-HUM, 50X2-WMD), and legacy declass values
     (X1-X8, OADR, MR, "DCI Only") — belong to the classification AUTHORITY BLOCK, NOT the banner.
     Do NOT place any of them in "classification". Record the document's declassification/exemption
     instructions in the rationale instead, transcribed verbatim as found (never modernize or
     recalculate them).
   - Any token that is not a security marking tied to a classification (headings, form/org names,
     titles, addresses, plain dates) is excluded.

Place legacy or unrecognized but plausible controls (e.g., WNINTEL) in the dissemination-controls
category, in the position they appear relative to recognized controls; transcribe them verbatim.

Example: pages marked (S), SECRET//NOFORN, and SECRET NOFORN WNINTEL, where one page's authority
block reads "Declassify On: X1", combine to the banner SECRET//NOFORN/WNINTEL — the X1 is a
declassification exemption noted in the rationale, not part of the banner. Had another page instead
been marked SECRET//REL TO USA, GBR, the banner would still be SECRET//NOFORN, because NOFORN takes
precedence over REL TO.

- rationale: Comprehensive explanation citing specific page evidence — which marking came from
  which page, how the banner was assembled, which combination rules applied (e.g., NOFORN over
  REL TO, REL TO dropped because a portion was uncaveated), and the document's
  declassification/exemption instructions transcribed as found.

Behavioral constraints:
- Always respond with valid JSON, no markdown fencing
- Take the union of all valid page markings, THEN apply the combination rules and exclusions above;
  never copy a single page's banner as the final answer
- Every caveat/control must be associated with the base classification; discard anything that is
  not an actual security marking
- Report markings as found; transcribe legacy markings verbatim, never modernizing or converting
- Do not union corrupted or invalid tokens; if a page lists a degraded reading of a valid marking
  present elsewhere (e.g., "NOFOPI" vs "NOFORN", "XI" vs "X1"), use the valid marking
- Keep declassification dates and exemption markings (X1-X8, 25X1, 50X1-HUM, OADR, MR) OUT of the
  classification string; record them in the rationale`

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
