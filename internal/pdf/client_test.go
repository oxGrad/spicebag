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
	// mock Gotenberg server
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
