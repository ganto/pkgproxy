// Copyright 2026 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolvePublicAddr(t *testing.T) {
	tests := []struct {
		name       string
		flagValue  string
		envValue   string
		listenAddr string
		listenPort uint16
		want       string
	}{
		{
			name:       "flag takes precedence over env var",
			flagValue:  "myproxy.lan",
			envValue:   "other.host",
			listenAddr: "localhost",
			listenPort: 8080,
			want:       "myproxy.lan",
		},
		{
			name:       "env var used when flag is empty",
			flagValue:  "",
			envValue:   "myproxy.lan",
			listenAddr: "localhost",
			listenPort: 8080,
			want:       "myproxy.lan",
		},
		{
			name:       "defaults to listen host:port when neither is set",
			flagValue:  "",
			envValue:   "",
			listenAddr: "localhost",
			listenPort: 8080,
			want:       "localhost:8080",
		},
		{
			name:       "flag with embedded port used verbatim",
			flagValue:  "myproxy.lan:9090",
			envValue:   "",
			listenAddr: "localhost",
			listenPort: 8080,
			want:       "myproxy.lan:9090",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(publicHostEnvVar, tt.envValue)
			}
			got := resolvePublicAddr(tt.flagValue, tt.listenAddr, tt.listenPort)
			assert.Equal(t, tt.want, got)
		})
	}
}
