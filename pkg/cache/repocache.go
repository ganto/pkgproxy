// Copyright 2022 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package cache

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/ganto/pkgproxy/pkg/utils"
)

type Cache interface {
	GetFilePath(string) string
	IsCacheCandidate(string) bool
	IsCached(string) bool
	SaveToDisk(string, *bytes.Buffer) error
}

type repoCache struct {
	config *CacheConfig
}

type CacheConfig struct {
	BasePath     string
	FileSuffixes []string
}

func New(cfg *CacheConfig) Cache {
	return &repoCache{
		config: cfg,
	}
}

// Returns the path to the cached file
func (rc *repoCache) GetFilePath(uri string) string {
	return path.Join(rc.config.BasePath, uri)
}

// Verifies if the given file URI is candidate to be cached
func (rc *repoCache) IsCacheCandidate(uri string) bool {
	c := false

	name := utils.FilenameFromUri(uri)
	for _, suffix := range rc.config.FileSuffixes {
		if strings.HasSuffix(name, suffix) {
			c = true
			break
		}
	}

	return c
}

// Verifies if the file is already cached
func (rc *repoCache) IsCached(uri string) bool {
	c := false

	if _, err := os.Stat(rc.GetFilePath(uri)); err == nil {
		c = true
	}

	return c
}

// Saves buffer to file
func (rc *repoCache) SaveToDisk(uri string, buffer *bytes.Buffer) error {
	cachePath := path.Join(rc.config.BasePath, uri)

	if _, err := os.Stat(path.Dir(cachePath)); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(path.Dir(cachePath), os.ModePerm); err != nil {
			return err
		}
	}

	fmt.Printf("<== writing file '%s': ", cachePath)
	cacheFile, err := os.Create(cachePath)
	if err != nil {
		return err
	}
	defer cacheFile.Close()

	size, err := cacheFile.ReadFrom(buffer)
	if err != nil {
		return err
	}
	fmt.Printf("%d bytes written\n", size)

	return nil
}
