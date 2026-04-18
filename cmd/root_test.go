// Copyright 2026 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInjectServeDefault(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantArgs []string
	}{
		{
			name:     "zero args → serve inserted",
			args:     []string{"pkgproxy"},
			wantArgs: []string{"pkgproxy", "serve"},
		},
		{
			name:     "--help → unchanged",
			args:     []string{"pkgproxy", "--help"},
			wantArgs: []string{"pkgproxy", "--help"},
		},
		{
			name:     "version → unchanged",
			args:     []string{"pkgproxy", "version"},
			wantArgs: []string{"pkgproxy", "version"},
		},
		{
			name:     "explicit serve → unchanged",
			args:     []string{"pkgproxy", "serve"},
			wantArgs: []string{"pkgproxy", "serve"},
		},
		{
			name:     "bare flag → unchanged",
			args:     []string{"pkgproxy", "--debug"},
			wantArgs: []string{"pkgproxy", "--debug"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := os.Args
			t.Cleanup(func() { os.Args = original })

			os.Args = tt.args
			injectServeDefault()
			assert.Equal(t, tt.wantArgs, os.Args)
		})
	}
}
