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
	"time"

	"github.com/ganto/pkgproxy/pkg/utils"
)

type FileCache interface {
	// Remove cached file for given URL
	DeleteFile(string) error

	// Return file system path to cached file for given URL
	GetFilePath(string) string

	// Returns a list of file suffixes that will be cached
	GetFileSuffixes() []string

	// Return if URL is supposed to be cached
	IsCacheCandidate(string) bool

	// Return if file exists in cache for given URL
	IsCached(string) bool

	// Save buffer as file in cache for given URL
	SaveToDisk(string, *bytes.Buffer, time.Time) error
}

type cache struct {
	config *CacheConfig
}

type CacheConfig struct {
	// Local file system base path for storing cached files
	BasePath string

	// List of file suffixes that will be cached
	FileSuffixes []string
}

func New(cfg *CacheConfig) FileCache {
	return &cache{
		config: cfg,
	}
}

// Delete file from cache
func (c *cache) DeleteFile(uri string) error {
	path := c.GetFilePath(uri)
	fmt.Printf("<== deleting file: %s\n", path)
	return os.Remove(path)
}

// Returns the local file system base path for storing the files
func (c *cache) getBasePath() string {
	return c.config.BasePath
}

// Returns the path to the cached file
func (c *cache) GetFilePath(uri string) string {
	return path.Join(c.getBasePath(), uri)
}

// Returns a list of file suffixes that will be cached
func (c *cache) GetFileSuffixes() []string {
	return c.config.FileSuffixes
}

// Verifies if the given file URI is candidate to be cached
func (c *cache) IsCacheCandidate(uri string) bool {
	ca := false

	name := utils.FilenameFromURI(uri)
	for _, suffix := range c.GetFileSuffixes() {
		if strings.HasSuffix(name, suffix) {
			ca = true
			break
		}
	}

	return ca
}

// Verifies if the file is already cached
func (c *cache) IsCached(uri string) bool {
	ca := false

	if _, err := os.Stat(c.GetFilePath(uri)); err == nil {
		ca = true
	}

	return ca
}

// Saves buffer to file
func (c *cache) SaveToDisk(uri string, buffer *bytes.Buffer, fileTime time.Time) error {
	filePath := path.Join(c.getBasePath(), uri)

	if _, err := os.Stat(path.Dir(filePath)); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(path.Dir(filePath), os.ModePerm); err != nil {
			return err
		}
	}

	fmt.Printf("<== writing file '%s': ", filePath)
	cacheFile, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer cacheFile.Close()

	size, err := cacheFile.ReadFrom(buffer)
	if err != nil {
		return err
	}
	fmt.Printf("%d bytes written\n", size)

	// set modified time to given timestamp
	if err := os.Chtimes(filePath, time.Now().Local(), fileTime); err != nil {
		return err
	}

	return nil
}
