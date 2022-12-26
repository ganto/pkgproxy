package pkgproxy

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ganto/pkgproxy/pkg/cache"
	"github.com/ganto/pkgproxy/pkg/utils"
)

type PkgProxyTransport struct {
	Rt    http.RoundTripper
	Cache cache.Cache
}

func (ppt PkgProxyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// save original request details for potential redirect and later caching
	reqUri := strings.Clone(req.RequestURI)
	reqHeaders := req.Header

	rsp, err := ppt.Rt.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// follow HTTP 301/302 redirects
	if rsp.StatusCode == 301 || rsp.StatusCode == 302 {
		if err := followRedirect(rsp, reqHeaders); err != nil {
			return nil, err
		}
	}

	// save payload in cache directory
	if ppt.Cache.IsCacheCandidate(reqUri) && !ppt.Cache.IsCached(reqUri) {
		if err = ppt.Cache.SaveToDisk(reqUri, rsp); err != nil {
			// don't fail request if we cannot write to cache
			fmt.Printf("Error: %s", err.Error())
		}
	}

	return rsp, nil
}

// Read redirect location from response, send request to new location and
// replace original response with response from new location
func followRedirect(rsp *http.Response, headers http.Header) error {
	location, err := rsp.Location()
	if err != nil {
		return err
	}

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, location.String(), nil)
	if err != nil {
		return err
	}
	updateHttpHeaders(&headers, &req.Header)

	r, err := client.Do(req)
	if err != nil {
		return err
	}
	updateResponse(r, rsp)

	return nil
}

// Overwrite existing response with another response
func updateResponse(src *http.Response, dst *http.Response) {
	dst.ProtoMajor = src.ProtoMajor
	dst.ProtoMinor = src.ProtoMinor
	dst.Request.Host = src.Request.Host
	dst.Request.RequestURI = src.Request.RequestURI
	dst.Status = src.Status
	dst.StatusCode = src.StatusCode
	dst.ContentLength = src.ContentLength
	dst.Body = src.Body

	updateHttpHeaders(&src.Header, &dst.Header)
}

// Overwrite existing HTTP headers with given headers
func updateHttpHeaders(src *http.Header, dst *http.Header) {
	srcHeaders := utils.KeysFromMap(*src)
	dstHeaders := utils.KeysFromMap(*dst)

	for _, header := range utils.ListIntersection(srcHeaders, dstHeaders) {
		dst.Set(header, src.Get(header))
	}
	for _, header := range utils.ListDifference(dstHeaders, srcHeaders) {
		dst.Del(header)
	}
	for _, header := range utils.ListDifference(srcHeaders, dstHeaders) {
		dst.Add(header, src.Get(header))
	}
}
