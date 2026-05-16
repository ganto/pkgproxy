// Copyright 2026 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultArgs(t *testing.T) {
	tests := []struct {
		name     string
		osArgs   []string
		wantArgs []string
	}{
		{
			name:     "zero args → serve inserted",
			osArgs:   []string{"pkgproxy"},
			wantArgs: []string{"serve"},
		},
		{
			name:     "--help → unchanged",
			osArgs:   []string{"pkgproxy", "--help"},
			wantArgs: []string{"--help"},
		},
		{
			name:     "version → unchanged",
			osArgs:   []string{"pkgproxy", "version"},
			wantArgs: []string{"version"},
		},
		{
			name:     "explicit serve → unchanged",
			osArgs:   []string{"pkgproxy", "serve"},
			wantArgs: []string{"serve"},
		},
		{
			name:     "bare flag → unchanged",
			osArgs:   []string{"pkgproxy", "--debug"},
			wantArgs: []string{"--debug"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantArgs, defaultArgs(tt.osArgs))
		})
	}
}
