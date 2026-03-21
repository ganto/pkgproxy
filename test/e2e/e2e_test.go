//go:build e2e

package e2e

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// containerRuntime holds the detected container runtime binary name.
var containerRuntime string

// hostGateway holds the hostname that containers use to reach the host.
var hostGateway string

func detectContainerRuntime(t *testing.T) {
	t.Helper()

	if v := os.Getenv("CONTAINER_RUNTIME"); v != "" {
		if _, err := exec.LookPath(v); err != nil {
			t.Skipf("CONTAINER_RUNTIME=%s not found on PATH", v)
		}
		containerRuntime = v
	} else if _, err := exec.LookPath("podman"); err == nil {
		containerRuntime = "podman"
	} else if _, err := exec.LookPath("docker"); err == nil {
		containerRuntime = "docker"
	} else {
		t.Skip("no container runtime (podman or docker) found on PATH")
	}

	if containerRuntime == "podman" {
		hostGateway = "host.containers.internal"
	} else {
		hostGateway = "host.docker.internal"
	}
	t.Logf("container runtime: %s, host gateway: %s", containerRuntime, hostGateway)
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "0.0.0.0:0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

func buildPkgproxy(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "pkgproxy")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = filepath.Join(projectRoot())
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "build failed: %s", out)
	return bin
}

func projectRoot() string {
	// test/e2e/ -> project root
	wd, _ := os.Getwd()
	return filepath.Join(wd, "..", "..")
}

func startPkgproxy(t *testing.T, bin string, port int, cacheDir string) *exec.Cmd {
	t.Helper()
	configPath := filepath.Join(projectRoot(), "configs", "pkgproxy.yaml")
	cmd := exec.Command(bin, "serve",
		"--host", "0.0.0.0",
		"--port", fmt.Sprintf("%d", port),
		"--config", configPath,
		"--cachedir", cacheDir,
		"--debug",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	// Wait for pkgproxy to be ready.
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			return cmd
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("pkgproxy did not become ready on %s", addr)
	return cmd
}

func runContainer(t *testing.T, image string, mounts []string, cmdArgs []string) {
	t.Helper()
	args := []string{"run", "--rm"}
	for _, m := range mounts {
		args = append(args, "-v", m)
	}
	args = append(args, image)
	args = append(args, cmdArgs...)

	t.Logf("running: %s %v", containerRuntime, args)
	cmd := exec.Command(containerRuntime, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	require.NoError(t, err, "container command failed")
}

func assertCachedFiles(t *testing.T, cacheDir string, repoPrefix string, suffix string) {
	t.Helper()
	var matches []string
	filepath.Walk(filepath.Join(cacheDir, repoPrefix), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(path, suffix) {
			matches = append(matches, path)
		}
		return nil
	})
	assert.NotEmpty(t, matches, "expected %s files under %s/%s", suffix, cacheDir, repoPrefix)
	if len(matches) > 0 {
		t.Logf("found %d %s files under %s/%s (first: %s)", len(matches), suffix, cacheDir, repoPrefix, matches[0])
	}
}

func TestE2E(t *testing.T) {
	detectContainerRuntime(t)

	bin := buildPkgproxy(t)
	port := freePort(t)
	cacheDir := t.TempDir()

	startPkgproxy(t, bin, port, cacheDir)

	proxyAddr := fmt.Sprintf("%s:%d", hostGateway, port)
	scriptDir := filepath.Join(projectRoot(), "test", "e2e")

	t.Run("Fedora", func(t *testing.T) {
		repoFile := filepath.Join(t.TempDir(), "pkgproxy-fedora.repo")
		repoContent := fmt.Sprintf(`[fedora]
# metalink=https://mirrors.fedoraproject.org/metalink?repo=fedora-$releasever&arch=$basearch
baseurl=http://%s/fedora/releases/$releasever/Everything/$basearch/os/
gpgcheck=1
gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-fedora-$releasever-$basearch
`, proxyAddr)
		require.NoError(t, os.WriteFile(repoFile, []byte(repoContent), 0644))

		runContainer(t, "fedora:43",
			[]string{
				filepath.Join(scriptDir, "test-dnf.sh") + ":/test-dnf.sh:ro,z",
				repoFile + ":/etc/yum.repos.d/pkgproxy-fedora.repo:ro,z",
			},
			[]string{"bash", "/test-dnf.sh", proxyAddr, "tree"},
		)

		assertCachedFiles(t, cacheDir, "fedora", ".rpm")
	})

	t.Run("COPR", func(t *testing.T) {
		repoFile := filepath.Join(t.TempDir(), "pkgproxy-copr.repo")
		repoContent := fmt.Sprintf(`[copr:copr.fedorainfracloud.org:ganto:jo]
baseurl=http://%s/copr/ganto/jo/fedora-$releasever-$basearch/
gpgcheck=0
`, proxyAddr)
		require.NoError(t, os.WriteFile(repoFile, []byte(repoContent), 0644))

		runContainer(t, "fedora:43",
			[]string{
				filepath.Join(scriptDir, "test-dnf.sh") + ":/test-dnf.sh:ro,z",
				repoFile + ":/etc/yum.repos.d/pkgproxy-copr.repo:ro,z",
			},
			[]string{"bash", "/test-dnf.sh", proxyAddr, "jo"},
		)

		assertCachedFiles(t, cacheDir, "copr", ".rpm")
	})

	t.Run("Debian", func(t *testing.T) {
		runContainer(t, "debian:trixie",
			[]string{
				filepath.Join(scriptDir, "test-apt.sh") + ":/test-apt.sh:ro,z",
			},
			[]string{"bash", "/test-apt.sh", proxyAddr, "trixie", "tree"},
		)

		assertCachedFiles(t, cacheDir, "debian", ".deb")
	})

	t.Run("Arch", func(t *testing.T) {
		runContainer(t, "archlinux:latest",
			[]string{
				filepath.Join(scriptDir, "test-pacman.sh") + ":/test-pacman.sh:ro,z",
			},
			[]string{"bash", "/test-pacman.sh", proxyAddr, "tree"},
		)

		assertCachedFiles(t, cacheDir, "archlinux", ".tar.zst")
	})
}
