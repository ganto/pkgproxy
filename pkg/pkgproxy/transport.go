package pkgproxy

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/ganto/pkgproxy/pkg/utils"
)

type pkgProxyTransport struct {
	host             string
	rt               http.RoundTripper
	cachedFileSuffix []string
}

func (ppt pkgProxyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
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

	if err := handleResponse(rsp, ppt.cachedFileSuffix); err != nil {
		return nil, err
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

func handleResponse(rsp *http.Response, saveSuffixes []string) error {
	urlPath := "pub/fedora"

	reqFile := utils.FilenameFromUrl(rsp.Request.URL)
	mustCache := false
	for _, suffix := range saveSuffixes {
		if strings.HasSuffix(reqFile, suffix) && rsp.ContentLength > 0 {
			mustCache = true
			break
		}
	}
	if mustCache {
		cacheBasePath := "cache"
		cacheRepoPath := cacheBasePath + "/" + urlPath

		filename := cacheRepoPath + "/" + reqFile
		skipSave := false
		if _, err := os.Stat(cacheRepoPath); errors.Is(err, os.ErrNotExist) {
			err := os.MkdirAll(cacheRepoPath, os.ModePerm)
			if err != nil {
				skipSave = true
			}
		}
		if _, err := os.Stat(filename); err == nil {
			skipSave = true
		}

		// write file to disk
		if !skipSave {

			payload, err := io.ReadAll(rsp.Body)
			if err != nil {
				return err
			}
			err = rsp.Body.Close()
			if err != nil {
				return err
			}
			body := io.NopCloser(bytes.NewReader(payload))
			rsp.Body = body

			fmt.Printf("writing file '%s': ", filename)
			cacheFile, err := os.Create(filename)
			if err == nil {
				defer cacheFile.Close()
				size, err := cacheFile.ReadFrom(bytes.NewReader(payload))
				if err != nil {
					fmt.Printf("\nerror when writing file: %s\n", err.Error())
				} else {
					if size != rsp.ContentLength {
						fmt.Printf("\nerror: could not write entire file size: %d != %d\n", size, rsp.ContentLength)
						os.Remove(filename)
					} else {
						fmt.Printf("%d bytes written\n", size)
					}
				}
			}
		}
	}

	return nil
}
