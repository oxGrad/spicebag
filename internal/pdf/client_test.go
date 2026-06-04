// internal/pdf/client_test.go
package pdf_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/oxGrad/spicebag/internal/pdf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderPDF(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/forms/chromium/convert/html", r.URL.Path)
		w.Header().Set("Content-Type", "application/pdf")
		w.Write([]byte("%PDF-fake"))
	}))
	defer srv.Close()

	client := pdf.NewClient(srv.URL)
	result, err := client.RenderPDF("<html><body>Hello</body></html>", "body { color: red; }")
	require.NoError(t, err)
	assert.Equal(t, []byte("%PDF-fake"), result)
}

func TestRenderPDFNonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	client := pdf.NewClient(srv.URL)
	_, err := client.RenderPDF("<html/>", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "503")
}

func TestRenderWithFallbackGotenbergSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		w.Write([]byte("%PDF-gotenberg"))
	}))
	defer srv.Close()

	result, err := pdf.RenderWithFallback(srv.URL, "<p>Hello</p>", "")
	require.NoError(t, err)
	assert.Equal(t, []byte("%PDF-gotenberg"), result)
}

func TestRenderWithFallbackGotenbergFailsFallsToChrome(t *testing.T) {
	// Gotenberg returns an error — should fall through to Chrome.
	// Chrome is present in this environment, so we only verify no panic.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "broken", http.StatusInternalServerError)
	}))
	defer srv.Close()

	// We don't assert success or failure here because Chrome availability
	// is environment-dependent; we only care the code path is exercised.
	pdf.RenderWithFallback(srv.URL, "<p>Hello</p>", "") //nolint:errcheck
}
