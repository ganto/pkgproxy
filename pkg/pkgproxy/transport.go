package pkgproxy

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ganto/pkgproxy/pkg/cache"
	"github.com/ganto/pkgproxy/pkg/utils"
)

type pkgProxyTransport struct {
	host  string
	rt    http.RoundTripper
	cache cache.Cache
}

func (ppt pkgProxyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	reqUri := strings.Clone(req.RequestURI)

	if ppt.cache.IsCacheCandidate(reqUri) {
		if ppt.cache.IsCached(reqUri) {
			fmt.Println("read from cache: needs implementation!")
		}
	}

	// either the file must not be cached or it's not in the cache yet
	// therefore send out request
	setRequestHeaders(req, ppt.host)
	rsp, err := ppt.rt.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// follow HTTP 301/302 redirects
	if rsp.StatusCode == 301 || rsp.StatusCode == 302 {
		if err := followRedirect(rsp); err != nil {
			return nil, err
		}
	}

	// save payload in cache directory
	if ppt.cache.IsCacheCandidate(reqUri) && !ppt.cache.IsCached(reqUri) {
		if err = ppt.cache.SaveToDisk(reqUri, rsp); err != nil {
			// don't fail request if we cannot write to cache
			fmt.Printf("Error: %s", err.Error())
		}
	}

	return rsp, nil
}

// Adjust HTTP request headers
// - Add correct `Host` header for upstream destination
// - Remove proxy headers to avoid giving off network internals
func setRequestHeaders(req *http.Request, host string) {
	req.Host = host
	req.Header.Del("X-Real-Ip")
	req.Header.Del("X-Forwarded-For")
	req.Header.Del("X-Forwarded-Proto")
}

// Read redirect location from response, send request to new location and
// replace original response with response from new location
func followRedirect(rsp *http.Response) error {
	location, err := rsp.Location()
	if err != nil {
		return err
	}

	r, err := http.Get(location.String())
	if err != nil {
		return err
	}
	updateResponse(r, rsp)

	return nil
}

// Overwrite existing response with another response
func updateResponse(src *http.Response, dst *http.Response) error {
	dst.ProtoMajor = src.ProtoMajor
	dst.ProtoMinor = src.ProtoMinor
	dst.Request.Host = src.Request.Host
	dst.Request.RequestURI = src.Request.RequestURI
	dst.Status = src.Status
	dst.StatusCode = src.StatusCode
	dst.ContentLength = src.ContentLength
	dst.Body = src.Body

	srcHeaders := utils.KeysFromMap(src.Header)
	dstHeaders := utils.KeysFromMap(dst.Header)

	for _, header := range utils.ListIntersection(srcHeaders, dstHeaders) {
		dst.Header.Set(header, src.Header.Get(header))
	}
	for _, header := range utils.ListDifference(dstHeaders, srcHeaders) {
		dst.Header.Del(header)
	}
	for _, header := range utils.ListDifference(srcHeaders, dstHeaders) {
		dst.Header.Add(header, src.Header.Get(header))
	}

	return nil
}
