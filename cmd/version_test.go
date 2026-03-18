// Copyright 2026 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"bytes"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildDateInjected(t *testing.T) {
	orig := BuildDate
	t.Cleanup(func() { BuildDate = orig })

	BuildDate = "2026-03-17T10:00:00Z"
	assert.Equal(t, "2026-03-17T10:00:00Z", buildDate())
}

func TestBuildDateFallback(t *testing.T) {
	orig := BuildDate
	t.Cleanup(func() { BuildDate = orig })

	BuildDate = ""
	before := time.Now().UTC()
	got := buildDate()
	after := time.Now().UTC()

	parsed, err := time.Parse(time.RFC3339, got)
	require.NoError(t, err, "buildDate() should return RFC3339 format")
	assert.True(t, !parsed.Before(before.Truncate(time.Second)), "buildDate() should be >= test start")
	assert.True(t, !parsed.After(after.Add(time.Second)), "buildDate() should be <= test end")
}

func TestVersionCommandOutput(t *testing.T) {
	origVersion := Version
	origCommit := GitCommit
	origDate := BuildDate
	t.Cleanup(func() {
		Version = origVersion
		GitCommit = origCommit
		BuildDate = origDate
	})

	Version = "v0.1.0"
	GitCommit = "abc1234"
	BuildDate = "2026-03-17T10:00:00Z"

	cmd := newVersionCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{})

	require.NoError(t, cmd.Execute())

	output := buf.String()
	assert.Contains(t, output, "Version:    v0.1.0\n")
	assert.Contains(t, output, "GitCommit:  abc1234\n")
	assert.Contains(t, output, "GoVersion:  "+runtime.Version()+"\n")
	assert.Contains(t, output, "BuildDate:  2026-03-17T10:00:00Z\n")

	// Verify field order: Version before GitCommit before GoVersion before BuildDate
	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	require.Len(t, lines, 4)
	assert.True(t, bytes.HasPrefix(lines[0], []byte("Version:")))
	assert.True(t, bytes.HasPrefix(lines[1], []byte("GitCommit:")))
	assert.True(t, bytes.HasPrefix(lines[2], []byte("GoVersion:")))
	assert.True(t, bytes.HasPrefix(lines[3], []byte("BuildDate:")))
}
