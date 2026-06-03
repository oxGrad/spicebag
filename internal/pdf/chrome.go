package pdf

import (
	"fmt"
	"io"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

func renderWithChrome(html, css string) ([]byte, error) {
	path, found := launcher.LookPath()
	if !found {
		return nil, fmt.Errorf("Chrome or Chromium not found — install Chrome, or point gotenberg_url at a running Gotenberg instance")
	}

	u, err := launcher.New().
		Bin(path).
		Headless(true).
		NoSandbox(true).
		Launch()
	if err != nil {
		return nil, fmt.Errorf("launch Chrome: %w", err)
	}

	browser := rod.New().ControlURL(u)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("connect to Chrome: %w", err)
	}
	defer browser.Close()

	page, err := browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, fmt.Errorf("open page: %w", err)
	}
	defer page.Close()

	fullHTML := fmt.Sprintf(`<!DOCTYPE html><html><head><style>%s</style></head><body>%s</body></html>`, css, html)
	if err := page.SetDocumentContent(fullHTML); err != nil {
		return nil, fmt.Errorf("set content: %w", err)
	}
	if err := page.WaitLoad(); err != nil {
		return nil, fmt.Errorf("wait load: %w", err)
	}

	zero := 0.0
	reader, err := page.PDF(&proto.PagePrintToPDF{
		PrintBackground:     true,
		DisplayHeaderFooter: false,
		MarginTop:           &zero,
		MarginBottom:        &zero,
		MarginLeft:          &zero,
		MarginRight:         &zero,
	})
	if err != nil {
		return nil, fmt.Errorf("print PDF: %w", err)
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// RenderChrome renders via a locally installed Chrome or Chromium binary.
func RenderChrome(html, css string) ([]byte, error) {
	return renderWithChrome(html, css)
}

// RenderWithFallback tries Gotenberg first; falls back to Chrome headless if
// Gotenberg is unavailable or gotenbergURL is empty.
func RenderWithFallback(gotenbergURL, html, css string) ([]byte, error) {
	if gotenbergURL != "" {
		b, err := NewClient(gotenbergURL).RenderPDF(html, css)
		if err == nil {
			return b, nil
		}
	}
	return renderWithChrome(html, css)
}
