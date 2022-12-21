package pkgproxy

import "net/http"

type headerUpdate struct {
	host string
	r    http.RoundTripper
}

// Adjust HTTP request headers
// - Add correct `Host` header for upstream destination
// - Remove proxy headers to avoid giving off network internals
func (hu headerUpdate) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Host = hu.host
	r.Header.Del("X-Real-Ip")
	r.Header.Del("X-Forwarded-For")
	r.Header.Del("X-Forwarded-Proto")
	return hu.r.RoundTrip(r)
}
