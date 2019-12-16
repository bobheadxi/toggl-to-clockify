package transport

import (
	"net/http"
)

type ContentTypeTransport struct {
	next    http.RoundTripper
	content string
}

func NewContentTypeTransport(content string, next http.RoundTripper) http.RoundTripper {
	return &ContentTypeTransport{next, content}
}

func (c *ContentTypeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Content-Type", c.content)
	return c.next.RoundTrip(req)
}
