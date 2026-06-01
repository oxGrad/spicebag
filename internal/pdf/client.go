// internal/pdf/client.go
package pdf

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{baseURL: baseURL, http: &http.Client{}}
}

func (c *Client) RenderPDF(html, css string) ([]byte, error) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	htmlPart, err := w.CreateFormFile("files", "index.html")
	if err != nil {
		return nil, err
	}
	// inject CSS into HTML
	fullHTML := fmt.Sprintf(`<!DOCTYPE html><html><head><style>%s</style></head><body>%s</body></html>`, css, html)
	if _, err := io.WriteString(htmlPart, fullHTML); err != nil {
		return nil, err
	}
	w.Close()

	resp, err := c.http.Post(c.baseURL+"/forms/chromium/convert/html", w.FormDataContentType(), &body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gotenberg returned %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
