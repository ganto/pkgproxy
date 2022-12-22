package utils

import (
	"net/url"
	"testing"

	"github.com/bmizerany/assert"
)

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

func TestFilenameFromUrl(t *testing.T) {
	s := "http://foo.bar/foobar?foo#bar"
	u, _ := url.Parse(s)
	f := FilenameFromUrl(u)
	assert.Equal(t, f, "foobar")

	s = "http://foob.ar/foo.bar"
	u, _ = url.Parse(s)
	f = FilenameFromUrl(u)
	assert.Equal(t, f, "foo.bar")

	s = "http://foob.ar/foo.bar/"
	u, _ = url.Parse(s)
	f = FilenameFromUrl(u)
	assert.Equal(t, f, "/")

	s = "http://foo.bar/"
	u, _ = url.Parse(s)
	f = FilenameFromUrl(u)
	assert.Equal(t, f, "/")
}
