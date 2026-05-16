// Copyright 2026 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"net/http"
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

func TestResolveTrustProxy(t *testing.T) {
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
			flagValue:   "loopback",
			envValue:    "private",
			want:        "loopback",
		},
		{
			name:        "flag changed wins even when value is empty",
			flagChanged: true,
			flagValue:   "",
			envValue:    "private",
			want:        "",
		},
		{
			name:        "env var used when flag unchanged",
			flagChanged: false,
			flagValue:   "",
			envValue:    "private",
			want:        "private",
		},
		{
			name:        "empty env var falls through to empty default",
			flagChanged: false,
			flagValue:   "",
			envValue:    "",
			want:        "",
		},
		{
			name:        "neither set returns empty default",
			flagChanged: false,
			flagValue:   "",
			envValue:    "",
			want:        "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveTrustProxy(tt.flagChanged, tt.flagValue, tt.envValue)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseTrustProxy(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		remoteAddr string
		xff        string
		wantIP     string
		wantErr    string
	}{
		{
			name:       "empty string ignores XFF",
			value:      "",
			remoteAddr: "127.0.0.1:1234",
			xff:        "1.2.3.4",
			wantIP:     "127.0.0.1",
		},
		{
			name:       "none ignores XFF",
			value:      "none",
			remoteAddr: "127.0.0.1:1234",
			xff:        "1.2.3.4",
			wantIP:     "127.0.0.1",
		},
		{
			name:       "loopback honors XFF from loopback source",
			value:      "loopback",
			remoteAddr: "127.0.0.1:1234",
			xff:        "203.0.113.5",
			wantIP:     "203.0.113.5",
		},
		{
			name:       "loopback does not trust XFF from private source",
			value:      "loopback",
			remoteAddr: "10.0.0.1:1234",
			xff:        "203.0.113.5",
			wantIP:     "10.0.0.1",
		},
		{
			name:       "private honors XFF from private source",
			value:      "private",
			remoteAddr: "10.0.0.1:1234",
			xff:        "203.0.113.5",
			wantIP:     "203.0.113.5",
		},
		{
			name:       "private does not trust XFF from loopback source",
			value:      "private",
			remoteAddr: "127.0.0.1:1234",
			xff:        "203.0.113.5",
			wantIP:     "127.0.0.1",
		},
		{
			name:       "CIDR trusts matching source",
			value:      "10.0.0.0/8",
			remoteAddr: "10.5.5.5:1234",
			xff:        "203.0.113.5",
			wantIP:     "203.0.113.5",
		},
		{
			name:       "CIDR does not trust non-matching source",
			value:      "10.0.0.0/8",
			remoteAddr: "192.168.1.1:1234",
			xff:        "203.0.113.5",
			wantIP:     "192.168.1.1",
		},
		{
			name:       "bare IPv4 trusted as /32",
			value:      "192.168.1.10",
			remoteAddr: "192.168.1.10:1234",
			xff:        "203.0.113.5",
			wantIP:     "203.0.113.5",
		},
		{
			name:       "bare IPv4 does not trust sibling in same subnet",
			value:      "192.168.1.10",
			remoteAddr: "192.168.1.11:1234",
			xff:        "203.0.113.5",
			wantIP:     "192.168.1.11",
		},
		{
			name:       "bare IPv6 trusted as /128",
			value:      "::1",
			remoteAddr: "[::1]:1234",
			xff:        "203.0.113.5",
			wantIP:     "203.0.113.5",
		},
		{
			name:       "combined loopback and CIDR trusts loopback",
			value:      "loopback,10.0.0.0/8",
			remoteAddr: "127.0.0.1:1234",
			xff:        "203.0.113.5",
			wantIP:     "203.0.113.5",
		},
		{
			name:       "combined loopback and CIDR trusts CIDR",
			value:      "loopback,10.0.0.0/8",
			remoteAddr: "10.5.5.5:1234",
			xff:        "203.0.113.5",
			wantIP:     "203.0.113.5",
		},
		{
			name:       "whitespace around entries is tolerated",
			value:      " loopback , 10.0.0.0/8 ",
			remoteAddr: "127.0.0.1:1234",
			xff:        "203.0.113.5",
			wantIP:     "203.0.113.5",
		},
		{
			name:       "LOOPBACK keyword is case-insensitive",
			value:      "LOOPBACK",
			remoteAddr: "127.0.0.1:1234",
			xff:        "203.0.113.5",
			wantIP:     "203.0.113.5",
		},
		{
			name:       "duplicate none is treated as none",
			value:      "none,none",
			remoteAddr: "127.0.0.1:1234",
			xff:        "1.2.3.4",
			wantIP:     "127.0.0.1",
		},
		{
			name:       "duplicate IP entries are deduplicated",
			value:      "192.168.1.10,192.168.1.10",
			remoteAddr: "192.168.1.10:1234",
			xff:        "203.0.113.5",
			wantIP:     "203.0.113.5",
		},
		{
			name:    "none combined with other keyword causes error",
			value:   "none,loopback",
			wantErr: "cannot be combined",
		},
		{
			name:    "unrecognized entry causes error naming the token",
			value:   "garbage",
			wantErr: "garbage",
		},
		{
			name:    "malformed entry causes error naming the token",
			value:   "10.0.0.0/8,not-an-ip",
			wantErr: "not-an-ip",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor, err := parseTrustProxy(tt.value)
			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			assert.NoError(t, err)
			req, _ := http.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			assert.Equal(t, tt.wantIP, extractor(req))
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
