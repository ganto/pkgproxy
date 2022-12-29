// Copyright 2022 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package pkgproxy

import (
	"net/http"

	"github.com/ganto/pkgproxy/pkg/utils"
)

type PkgProxyTransport struct {
	Rt http.RoundTripper
}

func (ppt PkgProxyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
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
