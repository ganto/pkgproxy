package utils

import (
	"net/url"
	"strings"
)

// Return a list of the keys of a map
func KeysFromMap[T comparable, S any](a map[T]S) []T {
	keys := make([]T, len(a))

	i := 0
	for k := range a {
		keys[i] = k
		i++
	}

	return keys
}

// Return the difference of 2 lists (items in `a` that don’t exist in `b`)
func ListDifference[T comparable](a []T, b []T) []T {
	lookup := make(map[T]bool)
	set := make([]T, 0)

	for _, i := range b {
		lookup[i] = true
	}
	for _, i := range a {
		if _, ok := lookup[i]; !ok {
			set = append(set, i)
		}
	}

	return set
}

// Return the intersection of 2 lists (unique list of all items in both)
func ListIntersection[T comparable](a []T, b []T) []T {
	lookup := make(map[T]bool)
	set := make([]T, 0)

	for _, i := range a {
		lookup[i] = true
	}
	for _, i := range b {
		if _, ok := lookup[i]; ok {
			set = append(set, i)
		}
	}

	return set
}

// Return the filename part of a URL
//
// Filename is specified as the last section of the path that is not followed
// by a `/`
func FilenameFromUrl(url *url.URL) string {
	file := "/"

	if url.Path[len(url.Path)-1:] == "/" {
		return file
	}
	path := strings.Split(url.Path, "/")
	if len(path) > 0 {
		file = path[len(path)-1]
	}

	return file
}
