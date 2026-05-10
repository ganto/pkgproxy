// Copyright 2026 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveListenHost(t *testing.T) {
	tests := []struct {
		name        string
		flagChanged bool
		flagValue   string
		envValue    string
		want        string
	}{
		{
			name:        "flag changed wins over env var",
			flagChanged: true,
			flagValue:   "192.168.10.4",
			envValue:    "10.0.0.1",
			want:        "192.168.10.4",
		},
		{
			name:        "flag changed wins even when value equals default",
			flagChanged: true,
			flagValue:   "localhost",
			envValue:    "0.0.0.0",
			want:        "localhost",
		},
		{
			name:        "env var used when flag unchanged",
			flagChanged: false,
			flagValue:   "localhost",
			envValue:    "0.0.0.0",
			want:        "0.0.0.0",
		},
		{
			name:        "empty env var falls through to default",
			flagChanged: false,
			flagValue:   "localhost",
			envValue:    "",
			want:        "localhost",
		},
		{
			name:        "neither set returns default",
			flagChanged: false,
			flagValue:   "localhost",
			envValue:    "",
			want:        "localhost",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveListenHost(tt.flagChanged, tt.flagValue, tt.envValue)
			assert.Equal(t, tt.want, got)
		})
	}
}

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
			t.Setenv(publicHostEnvVar, tt.envValue)
			got := resolvePublicAddr(tt.flagValue, tt.listenAddr, tt.listenPort)
			assert.Equal(t, tt.want, got)
		})
	}
}
