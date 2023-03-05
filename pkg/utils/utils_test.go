// Copyright 2022 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package utils

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContains(t *testing.T) {
	r := Contains([]int{1, 2, 3}, 1)
	assert.Equal(t, r, true)

	r = Contains([]int{1, 2, 3}, 0)
	assert.Equal(t, r, false)

	r = Contains([]string{"a", "b", "c"}, "a")
	assert.Equal(t, r, true)

	r = Contains([]string{}, "")
	assert.Equal(t, r, false)
}

func TestKeysFromMap(t *testing.T) {
	k := KeysFromMap(map[string]string{})
	assert.Equal(t, len(k), 0)
	assert.Equal(t, k, []string{})

	k = KeysFromMap(map[string]string{"a": "1", "b": "2"})
	assert.Equal(t, k, []string{"a", "b"})

	j := KeysFromMap(map[int]string{1: "a", 2: "b"})
	assert.Equal(t, j, []int{1, 2})
}

func TestListDifference(t *testing.T) {
	d := ListDifference([]int{}, []int{1, 2})
	assert.Equal(t, len(d), 0)
	assert.Equal(t, d, []int{})

	d = ListDifference([]int{1, 2}, []int{1, 2})
	assert.Equal(t, len(d), 0)
	assert.Equal(t, d, []int{})

	d = ListDifference([]int{1, 2, 3}, []int{2, 3, 4})
	assert.Equal(t, d, []int{1})
}

func TestListIntersection(t *testing.T) {
	i := ListIntersection([]int{1}, []int{2})
	assert.Equal(t, len(i), 0)
	assert.Equal(t, i, []int{})

	i = ListIntersection([]int{1, 2}, []int{2, 3})
	assert.Equal(t, i, []int{2})
}

func TestFilenameFromUri(t *testing.T) {
	s := "http://foo.bar/foobar?foo#bar"
	u, _ := url.Parse(s)
	f := FilenameFromUri(u.Path)
	assert.Equal(t, f, "foobar")

	s = "http://foob.ar/foo.bar"
	u, _ = url.Parse(s)
	f = FilenameFromUri(u.Path)
	assert.Equal(t, f, "foo.bar")

	s = "http://foob.ar/foo.bar/"
	u, _ = url.Parse(s)
	f = FilenameFromUri(u.Path)
	assert.Equal(t, f, "/")

	s = "http://foo.bar/"
	u, _ = url.Parse(s)
	f = FilenameFromUri(u.Path)
	assert.Equal(t, f, "/")

	f = FilenameFromUri("")
	assert.Equal(t, f, "/")
}

func TestFilepathFromUri(t *testing.T) {
	s := "http://foo.bar/foobaz/foobar?foo#bar"
	u, _ := url.Parse(s)
	f := FilepathFromUri(u.Path)
	assert.Equal(t, f, "/foobaz")

	s = "http://foo.bar/foobaz/foobar/baz?foo#bar"
	u, _ = url.Parse(s)
	f = FilepathFromUri(u.Path)
	assert.Equal(t, f, "/foobaz/foobar")

	s = "http://foob.ar/foo.bar"
	u, _ = url.Parse(s)
	f = FilepathFromUri(u.Path)
	assert.Equal(t, f, "/")

	s = "http://foob.ar/foo.bar/"
	u, _ = url.Parse(s)
	f = FilepathFromUri(u.Path)
	assert.Equal(t, f, "/foo.bar")

	s = "http://foo.bar/"
	u, _ = url.Parse(s)
	f = FilepathFromUri(u.Path)
	assert.Equal(t, f, "/")

	f = FilepathFromUri("")
	assert.Equal(t, f, "/")
}

func TestRouteFromUri(t *testing.T) {
	s := "http://foo.bar/foobaz/foobar?foo#bar"
	u, _ := url.Parse(s)
	f := RouteFromUri(u.Path)
	assert.Equal(t, f, "/foobaz")

	s = "http://foo.bar/foobaz/foobar/baz?foo#bar"
	u, _ = url.Parse(s)
	f = RouteFromUri(u.Path)
	assert.Equal(t, f, "/foobaz")

	s = "http://foob.ar/foo.bar"
	u, _ = url.Parse(s)
	f = RouteFromUri(u.Path)
	assert.Equal(t, f, "/foo.bar")

	s = "http://foo.bar/"
	u, _ = url.Parse(s)
	f = RouteFromUri(u.Path)
	assert.Equal(t, f, "/")

	f = RouteFromUri("")
	assert.Equal(t, f, "/")
}
