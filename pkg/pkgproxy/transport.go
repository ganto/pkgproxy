// Copyright 2022 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package pkgproxy

import (
	"context"
	"net/http"
	"time"

	"github.com/ganto/pkgproxy/pkg/utils"
)

type transport struct {
	RT http.RoundTripper
}

// Custom RountTrip that follows redirects
func (t transport) RoundTrip(req *http.Request) (*http.Response, error) {
	reqHeaders := req.Header

	rsp, err := t.RT.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// follow HTTP redirects
	if utils.Contains([]int{301, 302, 303, 307, 308}, rsp.StatusCode) {
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

	client := &http.Client{
		Timeout: time.Second * 10,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, location.String(), nil)
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
