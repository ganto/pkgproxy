package cache

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/ganto/pkgproxy/pkg/utils"
)

type Cache interface {
	IsCacheCandidate(string) bool
	IsCached(string) bool
	SaveToDisk(string, *http.Response) error
}

type pkgCache struct {
	name   string
	config *PkgCacheConfig
}

type PkgCacheConfig struct {
	FileSuffixes []string
	BasePath     string
}

func NewPkgCache(name string, cfg *PkgCacheConfig) Cache {
	return &pkgCache{
		name:   name,
		config: cfg,
	}
}

// Verifies if the given file URI is candidate to be cached
func (pc *pkgCache) IsCacheCandidate(uri string) bool {
	c := false

	name := utils.FilenameFromUri(uri)
	for _, suffix := range pc.config.FileSuffixes {
		if strings.HasSuffix(name, suffix) {
			c = true
			break
		}
	}

	return c
}

// Verifies if the file is already cached
func (pc *pkgCache) IsCached(uri string) bool {
	c := false

	cachePath := path.Join(pc.config.BasePath, uri)
	if _, err := os.Stat(cachePath); err == nil {
		c = true
	}

	return c
}

// Saves response body to cache
func (pc *pkgCache) SaveToDisk(uri string, rsp *http.Response) error {
	cachePath := path.Join(pc.config.BasePath, uri)

	if _, err := os.Stat(path.Dir(cachePath)); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(path.Dir(cachePath), os.ModePerm); err != nil {
			return err
		}
	}

	payload, err := io.ReadAll(rsp.Body)
	if err != nil {
		return err
	}
	if err := rsp.Body.Close(); err != nil {
		return err
	}
	rsp.Body = io.NopCloser(bytes.NewReader(payload))

	fmt.Printf("writing file '%s': ", cachePath)
	cacheFile, err := os.Create(cachePath)
	if err != nil {
		return err
	}
	defer cacheFile.Close()

	size, err := cacheFile.ReadFrom(bytes.NewReader(payload))
	if err != nil {
		return err
	}
	fmt.Printf("%d bytes written\n", size)

	return nil
}
