// Copyright 2022-2026 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package cache

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ganto/pkgproxy/pkg/utils"
)

type FileCache interface {
	// Remove cached file for given URL
	DeleteFile(string) error

	// Return file system path to cached file for given URL. If the URL points
	// to a path outside the cache directory return an error.
	GetFilePath(string) (string, error)

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
	p, err := c.resolvedFilePath(uri)
	if err != nil {
		return err
	}
	fmt.Printf("<== deleting file: %s\n", p)
	return os.Remove(p)
}

// Returns the local file system base path for storing the files
func (c *cache) getBasePath() string {
	return c.config.BasePath
}

// resolvedFilePath returns the filesystem path for the given URI,
// verifying it remains within the cache base directory.
func (c *cache) resolvedFilePath(uri string) (string, error) {
	base := filepath.Clean(c.getBasePath())
	// Trim any leading separators so filepath.Join always treats uri as relative to base.
	p := filepath.Clean(filepath.Join(base, strings.TrimLeft(uri, "/")))
	rel, err := filepath.Rel(base, p)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("URI %q resolves outside the cache directory", uri)
	}
	return p, nil
}

// Returns the path to the cached file
func (c *cache) GetFilePath(uri string) (string, error) {
	return c.resolvedFilePath(uri)
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
	p, err := c.resolvedFilePath(uri)
	if err != nil {
		return false
	}

	_, err = os.Stat(p)
	return err == nil
}

// Saves buffer to file
func (c *cache) SaveToDisk(uri string, buffer *bytes.Buffer, fileTime time.Time) error {
	filePath, err := c.resolvedFilePath(uri)
	if err != nil {
		return err
	}

	if _, err := os.Stat(filepath.Dir(filePath)); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(filePath), 0o750); err != nil {
			return err
		}
	}

	fmt.Printf("<== writing file '%s': ", filePath)
	tmpFile, err := os.CreateTemp(filepath.Dir(filePath), "*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()

	size, err := tmpFile.ReadFrom(buffer)
	closeErr := tmpFile.Close()
	if err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return closeErr
	}
	fmt.Printf("%d bytes written\n", size)

	// set modified time to given timestamp
	if err := os.Chtimes(tmpPath, time.Now().Local(), fileTime); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	// atomically move into place so IsCached never observes a partial file
	if err := os.Rename(tmpPath, filePath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	return nil
}
