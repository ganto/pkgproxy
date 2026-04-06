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

// pkgproxyBin holds the path to the pre-built pkgproxy binary (set in TestMain).
var pkgproxyBin string

// containerRuntime holds the detected container runtime binary name.
var containerRuntime string

// hostGateway holds the hostname that containers use to reach the host.
var hostGateway string

func TestMain(m *testing.M) {
	// Build pkgproxy binary once for all test functions.
	tmpDir, err := os.MkdirTemp("", "pkgproxy-e2e-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	bin := filepath.Join(tmpDir, "pkgproxy")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = projectRoot()
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "build failed: %s\n%v\n", out, err)
		os.RemoveAll(tmpDir)
		os.Exit(1)
	}
	pkgproxyBin = bin

	code := m.Run()
	os.RemoveAll(tmpDir)
	os.Exit(code)
}

func detectContainerRuntime(t *testing.T) {
	t.Helper()

	if containerRuntime != "" {
		return
	}

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

func releaseOrDefault(defaultRelease string) string {
	if v := os.Getenv("E2E_RELEASE"); v != "" {
		return v
	}
	return defaultRelease
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "0.0.0.0:0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

func projectRoot() string {
	// test/e2e/ -> project root
	wd, _ := os.Getwd()
	return filepath.Join(wd, "..", "..")
}

func startPkgproxy(t *testing.T, port int, cacheDir string) {
	t.Helper()
	configPath := filepath.Join(projectRoot(), "configs", "pkgproxy.yaml")
	cmd := exec.Command(pkgproxyBin, "serve",
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
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("pkgproxy did not become ready on %s", addr)
}

func runContainer(t *testing.T, image string, mounts []string, cmdArgs []string) {
	t.Helper()
	args := []string{"run", "--rm",
		"--add-host", hostGateway + ":host-gateway",
	}
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

func assertNotCached(t *testing.T, cacheDir string, repoPrefix string, name string) {
	t.Helper()
	var matches []string
	filepath.Walk(filepath.Join(cacheDir, repoPrefix), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && filepath.Base(path) == name {
			matches = append(matches, path)
		}
		return nil
	})
	assert.Empty(t, matches, "expected no %s files under %s/%s, but found: %v", name, cacheDir, repoPrefix, matches)
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

// setupPkgproxy is a convenience that detects the container runtime, allocates
// a free port, creates a temp cache dir, and starts a pkgproxy instance.
// It returns the proxy address and cache directory.
func setupPkgproxy(t *testing.T) (proxyAddr string, cacheDir string) {
	t.Helper()
	detectContainerRuntime(t)
	port := freePort(t)
	cacheDir = t.TempDir()
	startPkgproxy(t, port, cacheDir)
	proxyAddr = fmt.Sprintf("%s:%d", hostGateway, port)
	return proxyAddr, cacheDir
}

func scriptDir() string {
	return filepath.Join(projectRoot(), "test", "e2e")
}

// dnfRepoFile creates a .repo file in a temp directory and returns its path.
func dnfRepoFile(t *testing.T, name string, content string) string {
	t.Helper()
	repoFile := filepath.Join(t.TempDir(), "pkgproxy-"+name+".repo")
	require.NoError(t, os.WriteFile(repoFile, []byte(content), 0644))
	return repoFile
}

func TestFedora(t *testing.T) {
	release := releaseOrDefault("43")
	proxyAddr, cacheDir := setupPkgproxy(t)

	repoFile := dnfRepoFile(t, "fedora", fmt.Sprintf(`[fedora]
# metalink=https://mirrors.fedoraproject.org/metalink?repo=fedora-$releasever&arch=$basearch
baseurl=http://%s/fedora/releases/$releasever/Everything/$basearch/os/
gpgcheck=1
gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-fedora-$releasever-$basearch
`, proxyAddr))

	image := fmt.Sprintf("docker.io/library/fedora:%s", release)
	runContainer(t, image,
		[]string{
			filepath.Join(scriptDir(), "test-dnf.sh") + ":/test-dnf.sh:ro,z",
			repoFile + ":/etc/yum.repos.d/pkgproxy-fedora.repo:ro,z",
		},
		[]string{"bash", "/test-dnf.sh", proxyAddr, "tree"},
	)
	assertCachedFiles(t, cacheDir, "fedora", ".rpm")

	t.Run("COPR", func(t *testing.T) {
		coprFile := dnfRepoFile(t, "copr", fmt.Sprintf(`[copr:copr.fedorainfracloud.org:ganto:jo]
baseurl=http://%s/copr/ganto/jo/fedora-$releasever-$basearch/
gpgcheck=0
`, proxyAddr))

		runContainer(t, image,
			[]string{
				filepath.Join(scriptDir(), "test-dnf.sh") + ":/test-dnf.sh:ro,z",
				coprFile + ":/etc/yum.repos.d/pkgproxy-copr.repo:ro,z",
			},
			[]string{"bash", "/test-dnf.sh", proxyAddr, "jo"},
		)
		assertCachedFiles(t, cacheDir, "copr", ".rpm")
	})
}

func TestDebian(t *testing.T) {
	release := releaseOrDefault("trixie")
	proxyAddr, cacheDir := setupPkgproxy(t)

	sourcesList := filepath.Join(t.TempDir(), "sources.list")
	sourcesContent := fmt.Sprintf(`deb http://%s/debian          %s           main contrib non-free non-free-firmware
deb http://%s/debian          %s-updates   main contrib non-free non-free-firmware
deb http://%s/debian-security %s-security  main contrib non-free non-free-firmware
`, proxyAddr, release, proxyAddr, release, proxyAddr, release)
	require.NoError(t, os.WriteFile(sourcesList, []byte(sourcesContent), 0644))

	image := fmt.Sprintf("docker.io/library/debian:%s", release)
	runContainer(t, image,
		[]string{
			filepath.Join(scriptDir(), "test-apt.sh") + ":/test-apt.sh:ro,z",
			sourcesList + ":/etc/apt/sources.list:ro,z",
		},
		[]string{"bash", "/test-apt.sh", "tree"},
	)
	assertCachedFiles(t, cacheDir, "debian", ".deb")
}

func TestArch(t *testing.T) {
	proxyAddr, cacheDir := setupPkgproxy(t)

	runContainer(t, "docker.io/library/archlinux:latest",
		[]string{
			filepath.Join(scriptDir(), "test-pacman.sh") + ":/test-pacman.sh:ro,z",
		},
		[]string{"bash", "/test-pacman.sh", proxyAddr, "tree"},
	)
	assertCachedFiles(t, cacheDir, "archlinux", ".tar.zst")
}

func TestCentOSStream(t *testing.T) {
	release := releaseOrDefault("10")
	proxyAddr, cacheDir := setupPkgproxy(t)

	repoFile := dnfRepoFile(t, "centos-stream", fmt.Sprintf(`[baseos]
baseurl=http://%s/centos-stream/$releasever-stream/BaseOS/$basearch/os/
gpgcheck=1
gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-centosofficial-SHA256
`, proxyAddr))

	epelFile := dnfRepoFile(t, "epel", fmt.Sprintf(`[epel]
baseurl=http://%s/epel/$releasever/Everything/$basearch/
gpgcheck=1
gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-EPEL-$releasever
`, proxyAddr))

	image := fmt.Sprintf("quay.io/centos/centos:stream%s", release)
	runContainer(t, image,
		[]string{
			filepath.Join(scriptDir(), "test-dnf.sh") + ":/test-dnf.sh:ro,z",
			repoFile + ":/etc/yum.repos.d/pkgproxy-centos-stream.repo:ro,z",
			epelFile + ":/etc/yum.repos.d/pkgproxy-epel.repo:ro,z",
		},
		[]string{"bash", "/test-dnf.sh", proxyAddr, "tree"},
	)
	assertCachedFiles(t, cacheDir, "centos-stream", ".rpm")

	t.Run("COPR", func(t *testing.T) {
		coprFile := dnfRepoFile(t, "copr", fmt.Sprintf(`[copr:copr.fedorainfracloud.org:ganto:jo]
baseurl=http://%s/copr/ganto/jo/epel-$releasever-$basearch/
gpgcheck=0
`, proxyAddr))

		runContainer(t, image,
			[]string{
				filepath.Join(scriptDir(), "test-dnf.sh") + ":/test-dnf.sh:ro,z",
				coprFile + ":/etc/yum.repos.d/pkgproxy-copr.repo:ro,z",
			},
			[]string{"bash", "/test-dnf.sh", proxyAddr, "jo"},
		)
		assertCachedFiles(t, cacheDir, "copr", ".rpm")
	})
}

func TestAlmaLinux(t *testing.T) {
	release := releaseOrDefault("10")
	proxyAddr, cacheDir := setupPkgproxy(t)

	repoFile := dnfRepoFile(t, "almalinux", fmt.Sprintf(`[baseos]
baseurl=http://%s/almalinux/$releasever/BaseOS/$basearch/os/
gpgcheck=1
gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-AlmaLinux-$releasever
`, proxyAddr))

	epelFile := dnfRepoFile(t, "epel", fmt.Sprintf(`[epel]
baseurl=http://%s/epel/$releasever/Everything/$basearch/
gpgcheck=1
gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-EPEL-$releasever
`, proxyAddr))

	image := fmt.Sprintf("docker.io/library/almalinux:%s", release)
	runContainer(t, image,
		[]string{
			filepath.Join(scriptDir(), "test-dnf.sh") + ":/test-dnf.sh:ro,z",
			repoFile + ":/etc/yum.repos.d/pkgproxy-almalinux.repo:ro,z",
			epelFile + ":/etc/yum.repos.d/pkgproxy-epel.repo:ro,z",
		},
		[]string{"bash", "/test-dnf.sh", proxyAddr, "tree"},
	)
	assertCachedFiles(t, cacheDir, "almalinux", ".rpm")

	t.Run("COPR", func(t *testing.T) {
		coprFile := dnfRepoFile(t, "copr", fmt.Sprintf(`[copr:copr.fedorainfracloud.org:ganto:jo]
baseurl=http://%s/copr/ganto/jo/epel-$releasever-$basearch/
gpgcheck=0
`, proxyAddr))

		runContainer(t, image,
			[]string{
				filepath.Join(scriptDir(), "test-dnf.sh") + ":/test-dnf.sh:ro,z",
				coprFile + ":/etc/yum.repos.d/pkgproxy-copr.repo:ro,z",
			},
			[]string{"bash", "/test-dnf.sh", proxyAddr, "jo"},
		)
		assertCachedFiles(t, cacheDir, "copr", ".rpm")
	})
}

func TestRockyLinux(t *testing.T) {
	release := releaseOrDefault("10")
	proxyAddr, cacheDir := setupPkgproxy(t)

	repoFile := dnfRepoFile(t, "rockylinux", fmt.Sprintf(`[baseos]
baseurl=http://%s/rockylinux/$releasever/BaseOS/$basearch/os/
gpgcheck=1
gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-Rocky-$releasever
`, proxyAddr))

	epelFile := dnfRepoFile(t, "epel", fmt.Sprintf(`[epel]
baseurl=http://%s/epel/$releasever/Everything/$basearch/
gpgcheck=1
gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-EPEL-$releasever
`, proxyAddr))

	image := fmt.Sprintf("docker.io/rockylinux/rockylinux:%s", release)
	runContainer(t, image,
		[]string{
			filepath.Join(scriptDir(), "test-dnf.sh") + ":/test-dnf.sh:ro,z",
			repoFile + ":/etc/yum.repos.d/pkgproxy-rockylinux.repo:ro,z",
			epelFile + ":/etc/yum.repos.d/pkgproxy-epel.repo:ro,z",
		},
		[]string{"bash", "/test-dnf.sh", proxyAddr, "tree"},
	)
	assertCachedFiles(t, cacheDir, "rockylinux", ".rpm")

	t.Run("COPR", func(t *testing.T) {
		coprFile := dnfRepoFile(t, "copr", fmt.Sprintf(`[copr:copr.fedorainfracloud.org:ganto:jo]
baseurl=http://%s/copr/ganto/jo/epel-$releasever-$basearch/
gpgcheck=0
`, proxyAddr))

		runContainer(t, image,
			[]string{
				filepath.Join(scriptDir(), "test-dnf.sh") + ":/test-dnf.sh:ro,z",
				coprFile + ":/etc/yum.repos.d/pkgproxy-copr.repo:ro,z",
			},
			[]string{"bash", "/test-dnf.sh", proxyAddr, "jo"},
		)
		assertCachedFiles(t, cacheDir, "copr", ".rpm")
	})
}

func TestUbuntu(t *testing.T) {
	release := releaseOrDefault("noble")
	proxyAddr, cacheDir := setupPkgproxy(t)

	sourcesList := filepath.Join(t.TempDir(), "sources.list")
	sourcesContent := fmt.Sprintf(`deb http://%s/ubuntu           %s           main restricted universe multiverse
deb http://%s/ubuntu           %s-updates   main restricted universe multiverse
deb http://%s/ubuntu-security  %s-security  main restricted universe multiverse
`, proxyAddr, release, proxyAddr, release, proxyAddr, release)
	require.NoError(t, os.WriteFile(sourcesList, []byte(sourcesContent), 0644))

	image := fmt.Sprintf("docker.io/library/ubuntu:%s", release)
	runContainer(t, image,
		[]string{
			filepath.Join(scriptDir(), "test-apt.sh") + ":/test-apt.sh:ro,z",
			sourcesList + ":/etc/apt/sources.list:ro,z",
		},
		[]string{"bash", "/test-apt.sh", "tree"},
	)
	assertCachedFiles(t, cacheDir, "ubuntu", ".deb")
}

func TestGentoo(t *testing.T) {
	proxyAddr, cacheDir := setupPkgproxy(t)

	runContainer(t, "docker.io/gentoo/stage3:latest",
		[]string{
			filepath.Join(scriptDir(), "test-gentoo.sh") + ":/test-gentoo.sh:ro,z",
		},
		[]string{"bash", "/test-gentoo.sh", proxyAddr},
	)
	assertCachedFiles(t, cacheDir, "gentoo/distfiles", "")
	assertNotCached(t, cacheDir, "gentoo", "layout.conf")
}
