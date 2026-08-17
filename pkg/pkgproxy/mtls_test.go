// Copyright 2026 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package pkgproxy

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeClientKeyPair generates a self-signed client certificate, writes the PEM
// encoded certificate and private key into a temporary directory and returns
// their paths together with a pool trusting the certificate.
func writeClientKeyPair(t *testing.T) (certPath string, keyPath string, pool *x509.CertPool) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "pkgproxy-test-client"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	pool = x509.NewCertPool()
	pool.AddCert(cert)

	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)

	dir := t.TempDir()
	certPath = filepath.Join(dir, "client.pem")
	keyPath = filepath.Join(dir, "client-key.pem")
	require.NoError(t, os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600))
	require.NoError(t, os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600))

	return certPath, keyPath, pool
}

// newMTLSUpstream starts a TLS test server that requires a verified client
// certificate. The handler records the common name of the presented client
// certificate, or leaves it empty when none was sent.
func newMTLSUpstream(t *testing.T, clientCAs *x509.CertPool, seenCN *string) *httptest.Server {
	t.Helper()

	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(r.TLS.PeerCertificates) > 0 {
			*seenCN = r.TLS.PeerCertificates[0].Subject.CommonName
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte("rpm payload"))
	}))
	srv.TLS = &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  clientCAs,
		MinVersion: tls.VersionTLS12,
	}
	srv.StartTLS()
	t.Cleanup(srv.Close)
	return srv
}

// upstreamTrustingTransport returns a transport that trusts the test server's
// certificate, so it can serve as the base for the mTLS transport.
func upstreamTrustingTransport(t *testing.T, srv *httptest.Server) *http.Transport {
	t.Helper()
	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	return &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}}
}

func TestForwardProxyCDNWithMTLS(t *testing.T) {
	certPath, keyPath, clientCAs := writeClientKeyPair(t)
	var seenCN string
	srv := newMTLSUpstream(t, clientCAs, &seenCN)

	pp, cacheDir := newCDNTestProxy(t, srv.URL+"/", &MTLSConfig{Cert: certPath, Key: keyPath}, upstreamTrustingTransport(t, srv))
	app := newTestApp(pp)

	req := httptest.NewRequest(http.MethodGet, "/testrepo/content/dist/package.rpm", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "rpm payload", rec.Body.String())
	assert.Equal(t, "pkgproxy-test-client", seenCN, "upstream did not receive the client certificate")

	// The response must still be cached like any other repository.
	assert.FileExists(t, filepath.Join(cacheDir, "testrepo", "content", "dist", "package.rpm"))
}

func TestForwardProxyCDNWithoutMTLSIsRejected(t *testing.T) {
	_, _, clientCAs := writeClientKeyPair(t)
	var seenCN string
	srv := newMTLSUpstream(t, clientCAs, &seenCN)

	// No mTLS configured: the handshake fails and the proxy reports a bad gateway.
	pp, _ := newCDNTestProxy(t, srv.URL+"/", nil, upstreamTrustingTransport(t, srv))
	app := newTestApp(pp)

	req := httptest.NewRequest(http.MethodGet, "/testrepo/content/dist/package.rpm", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadGateway, rec.Code)
	assert.Empty(t, seenCN)
}

func TestForwardProxyCDNPathMapping(t *testing.T) {
	var receivedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	pp, _ := newCDNTestProxy(t, srv.URL+"/", nil, nil)
	app := newTestApp(pp)

	req := httptest.NewRequest(http.MethodGet, "/testrepo/content/dist/rhel9/9/x86_64/baseos/os/repodata/repomd.xml", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "/content/dist/rhel9/9/x86_64/baseos/os/repodata/repomd.xml", receivedPath)
}

func TestNewMTLSTransportIsRepositoryScoped(t *testing.T) {
	certPath, keyPath, _ := writeClientKeyPair(t)

	cacheDir := t.TempDir()
	pp, err := New(&PkgProxyConfig{
		CacheBasePath: cacheDir,
		RepositoryConfig: &RepoConfig{
			Repositories: map[string]Repository{
				"cdnrepo": {
					CacheSuffixes: []string{".rpm"},
					CDN:           "https://cdn.example.com/",
					MTLS:          &MTLSConfig{Cert: certPath, Key: keyPath},
				},
				"mirrorrepo": {
					CacheSuffixes: []string{".rpm"},
					Mirrors:       []string{"https://mirror.example.com/"},
				},
			},
		},
	})
	require.NoError(t, err)
	proxy := pp.(*pkgProxy)

	cdnTransport := proxy.transportFor("cdnrepo")
	require.NotSame(t, proxy.transport, cdnTransport)
	certs := cdnTransport.(*http.Transport).TLSClientConfig.Certificates
	require.Len(t, certs, 1)

	// Repositories without mTLS keep using the proxy-wide transport.
	assert.Same(t, proxy.transport, proxy.transportFor("mirrorrepo"))
}

// TestForwardProxyCDNWithPrivateCA covers the Red Hat CDN case: the upstream
// server certificate is issued by a private CA that is absent from the system
// trust store, so it has to be supplied via 'mtls.ca'.
func TestForwardProxyCDNWithPrivateCA(t *testing.T) {
	certPath, keyPath, clientCAs := writeClientKeyPair(t)
	var seenCN string
	srv := newMTLSUpstream(t, clientCAs, &seenCN)

	// Without the CA bundle the handshake fails on server verification.
	pp, _ := newCDNTestProxy(t, srv.URL+"/", &MTLSConfig{Cert: certPath, Key: keyPath}, nil)
	rec := httptest.NewRecorder()
	newTestApp(pp).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/testrepo/package.rpm", nil))
	require.Equal(t, http.StatusBadGateway, rec.Code)
	assert.Contains(t, rec.Body.String(), "certificate")

	// Pointing 'ca' at the upstream's CA makes the same request succeed.
	caPath := filepath.Join(t.TempDir(), "ca.pem")
	require.NoError(t, os.WriteFile(caPath,
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw}), 0o600))

	pp, _ = newCDNTestProxy(t, srv.URL+"/", &MTLSConfig{Cert: certPath, Key: keyPath, CA: caPath}, nil)
	rec = httptest.NewRecorder()
	newTestApp(pp).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/testrepo/package.rpm", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "rpm payload", rec.Body.String())
	assert.Equal(t, "pkgproxy-test-client", seenCN)
}

func TestNewMTLSCAErrors(t *testing.T) {
	certPath, keyPath, _ := writeClientKeyPair(t)

	garbagePath := filepath.Join(t.TempDir(), "garbage.pem")
	require.NoError(t, os.WriteFile(garbagePath, []byte("not a certificate"), 0o600))

	tests := []struct {
		name    string
		ca      string
		wantErr string
	}{
		{name: "missing file", ca: "/nonexistent/ca.pem", wantErr: "unable to read mTLS CA bundle"},
		{name: "no certificates", ca: garbagePath, wantErr: "no certificates found in mTLS CA bundle"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(&PkgProxyConfig{
				CacheBasePath: t.TempDir(),
				RepositoryConfig: &RepoConfig{
					Repositories: map[string]Repository{
						"cdnrepo": {
							CacheSuffixes: []string{".rpm"},
							CDN:           "https://cdn.example.com/",
							MTLS:          &MTLSConfig{Cert: certPath, Key: keyPath, CA: tt.ca},
						},
					},
				},
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
			assert.Contains(t, err.Error(), "cdnrepo")
		})
	}
}

func TestNewMTLSMissingFileFails(t *testing.T) {
	_, err := New(&PkgProxyConfig{
		CacheBasePath: t.TempDir(),
		RepositoryConfig: &RepoConfig{
			Repositories: map[string]Repository{
				"cdnrepo": {
					CacheSuffixes: []string{".rpm"},
					CDN:           "https://cdn.example.com/",
					MTLS:          &MTLSConfig{Cert: "/nonexistent/cert.pem", Key: "/nonexistent/key.pem"},
				},
			},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cdnrepo")
	assert.Contains(t, err.Error(), "mTLS key pair")
}

func TestForwardProxyCrossHostRedirectDropsClientCertificate(t *testing.T) {
	certPath, keyPath, clientCAs := writeClientKeyPair(t)

	// The redirect target is a plain HTTP server, which the client certificate
	// must never reach.
	var redirectTargetTLS bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectTargetTLS = r.TLS != nil
		_, _ = w.Write([]byte("redirected payload"))
	}))
	defer target.Close()

	var seenCN string
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(r.TLS.PeerCertificates) > 0 {
			seenCN = r.TLS.PeerCertificates[0].Subject.CommonName
		}
		http.Redirect(w, r, target.URL+"/moved/package.rpm", http.StatusFound)
	}))
	srv.TLS = &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  clientCAs,
		MinVersion: tls.VersionTLS12,
	}
	srv.StartTLS()
	defer srv.Close()

	pp, _ := newCDNTestProxy(t, srv.URL+"/", &MTLSConfig{Cert: certPath, Key: keyPath}, upstreamTrustingTransport(t, srv))
	app := newTestApp(pp)

	req := httptest.NewRequest(http.MethodGet, "/testrepo/package.rpm", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "redirected payload", rec.Body.String())
	assert.Equal(t, "pkgproxy-test-client", seenCN, "the CDN itself must still see the client certificate")
	assert.False(t, redirectTargetTLS)
}
