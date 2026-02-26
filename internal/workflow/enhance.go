package workflow

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/JaimeStill/document-context/pkg/config"
	"github.com/JaimeStill/document-context/pkg/document"
	"github.com/JaimeStill/document-context/pkg/image"

	"github.com/JaimeStill/go-agents/pkg/agent"

	"github.com/JaimeStill/go-agents-orchestration/pkg/state"

	"github.com/JaimeStill/herald/internal/prompts"
	"github.com/JaimeStill/herald/pkg/formatting"
)

type enhanceResponse struct {
	MarkingsFound []string `json:"markings_found"`
	Rationale     string   `json:"rationale"`
}

// EnhanceNode returns a state node that re-renders flagged pages with adjusted
// ImageMagick settings and reclassifies them via vision. For each page with
// non-nil Enhancements, it re-renders from the source PDF using the specified
// brightness/contrast/saturation adjustments, sends the enhanced image to the
// vision model, and updates the per-page findings. Clears Enhancements after
// processing so the page is no longer flagged.
func EnhanceNode(rt *Runtime) state.StateNode {
	return state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		cs, err := extractClassState(s)
		if err != nil {
			return s, fmt.Errorf("enhance: %w", err)
		}

		tempDir, err := extractTempDir(s)
		if err != nil {
			return s, fmt.Errorf("enhance: %w", err)
		}

		enhanced := cs.EnhancePages()

		if err := enhancePages(ctx, rt, cs, tempDir); err != nil {
			return s, fmt.Errorf("enhance: %w", err)
		}

		rt.Logger.InfoContext(
			ctx, "enhance node complete",
			"pages_enhanced", len(enhanced),
		)

		s = s.Set(KeyClassState, *cs)
		return s, nil
	})
}

func buildEnhanceConfig(settings *EnhanceSettings) config.ImageConfig {
	opts := map[string]any{
		"background": "white",
	}

	if settings.Brightness != nil {
		opts["brightness"] = *settings.Brightness
	}

	if settings.Contrast != nil {
		opts["contrast"] = *settings.Contrast
	}

	if settings.Saturation != nil {
		opts["saturation"] = *settings.Saturation
	}

	return config.ImageConfig{
		Format:  "png",
		DPI:     300,
		Options: opts,
	}
}

func enhancePage(
	ctx context.Context,
	a agent.Agent,
	rt *Runtime,
	pdfDoc document.Document,
	cs *ClassificationState,
	pageIdx int,
	tempDir string,
) error {
	page := &cs.Pages[pageIdx]

	// Re-render with adjusted settings
	imgPath, err := rerender(pdfDoc, page, tempDir)
	if err != nil {
		return err
	}
	page.ImagePath = imgPath

	// Encode enhanced image for vision call
	dataURI, err := encodePageImage(imgPath)
	if err != nil {
		return err
	}

	// Compose prompt with current classification state as context
	prompt, err := ComposePrompt(ctx, rt.Prompts, prompts.StageEnhance, cs)
	if err != nil {
		return err
	}

	resp, err := a.Vision(ctx, prompt, []string{dataURI})
	if err != nil {
		return fmt.Errorf("vision call: %w", err)
	}

	parsed, err := formatting.Parse[enhanceResponse](resp.Content())
	if err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	// Update page findings and clear enhancement flag
	page.MarkingsFound = parsed.MarkingsFound
	page.Rationale = parsed.Rationale
	page.Enhancements = nil

	return nil
}

func enhancePages(ctx context.Context, rt *Runtime, cs *ClassificationState, tempDir string) error {
	pdfPath := filepath.Join(tempDir, sourcePDF)
	pdfDoc, err := document.OpenPDF(pdfPath)
	if err != nil {
		return fmt.Errorf("%w: open pdf: %w", ErrEnhanceFailed, err)
	}
	defer pdfDoc.Close()

	a, err := agent.New(&rt.Agent)
	if err != nil {
		return fmt.Errorf("%w: create agent: %w", ErrEnhanceFailed, err)
	}

	for _, i := range cs.EnhancePages() {
		if err := enhancePage(ctx, a, rt, pdfDoc, cs, i, tempDir); err != nil {
			return fmt.Errorf("%w: page %d: %w", ErrEnhanceFailed, cs.Pages[i].PageNumber, err)
		}

		rt.Logger.InfoContext(
			ctx, "page enhanced",
			"page", cs.Pages[i].PageNumber,
			"markings", cs.Pages[i].MarkingsFound,
		)
	}

	return nil
}

func extractTempDir(s state.State) (string, error) {
	val, ok := s.Get(KeyTempDir)
	if !ok {
		return "", fmt.Errorf("%w: missing %s in state", ErrEnhanceFailed, KeyTempDir)
	}

	tempDir, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("%w: %s is not string", ErrEnhanceFailed, KeyTempDir)
	}

	return tempDir, nil
}

func rerender(pdfDoc document.Document, page *ClassificationPage, tempDir string) (string, error) {
	p, err := pdfDoc.ExtractPage(page.PageNumber)
	if err != nil {
		return "", fmt.Errorf("extract page %d: %w", page.PageNumber, err)
	}

	cfg := buildEnhanceConfig(page.Enhancements)
	renderer, err := image.NewImageMagickRenderer(cfg)
	if err != nil {
		return "", fmt.Errorf("create renderer: %w", err)
	}

	data, err := p.ToImage(renderer, nil)
	if err != nil {
		return "", fmt.Errorf("render page %d: %w", page.PageNumber, err)
	}

	imgPath := filepath.Join(tempDir, fmt.Sprintf("page-%d-enhanced.png", page.PageNumber))
	if err := os.WriteFile(imgPath, data, 0600); err != nil {
		return "", fmt.Errorf("write enhanced page %d: %w", page.PageNumber, err)
	}

	return imgPath, nil
}
