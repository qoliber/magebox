package installer

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

// PHPRepoKind identifies where PHP packages come from.
type PHPRepoKind int

const (
	// PHPRepoNone means no third-party repository covers this release, so only
	// the PHP version shipped by the distribution is available.
	PHPRepoNone PHPRepoKind = iota
	// PHPRepoPPA is Ondrej Sury's Launchpad PPA.
	PHPRepoPPA
	// PHPRepoSury is packages.sury.org, which the PPA is being merged into and
	// which covers releases the PPA does not.
	PHPRepoSury
)

// PHPRepository is the repository chosen for a release.
type PHPRepository struct {
	Kind  PHPRepoKind
	Suite string
}

const (
	ondrejPPADistsURL = "https://ppa.launchpadcontent.net/ondrej/php/ubuntu/dists/"
	suryDistsURL      = "https://packages.sury.org/php/dists/"
	// surySourceFile is where MageBox writes the sury apt source.
	surySourceFile = "/etc/apt/sources.list.d/magebox-php.list"
	// suryKeyring is installed by sury's own keyring package.
	suryKeyring = "/usr/share/keyrings/deb.sury.org-php.gpg"
)

// PickPHPRepository chooses where PHP packages should come from.
//
// The PPA is preferred while it covers the running release. It stops at Ubuntu
// 24.04 and is being folded into packages.sury.org, which does publish for
// newer releases. Pinning the PPA's older suite instead is worse than useless:
// those packages depend on library versions the newer release no longer has,
// so every install fails on unsatisfiable dependencies.
func PickPHPRepository(codename string, ppaPublished, suryPublished func(string) bool) PHPRepository {
	if codename == "" {
		return PHPRepository{Kind: PHPRepoNone}
	}
	if ppaPublished(codename) {
		return PHPRepository{Kind: PHPRepoPPA, Suite: codename}
	}
	if suryPublished(codename) {
		return PHPRepository{Kind: PHPRepoSury, Suite: codename}
	}
	return PHPRepository{Kind: PHPRepoNone}
}

// ShouldRefuseRootBootstrap reports whether bootstrap must stop.
//
// Run as root, bootstrap builds the environment under /root: the config,
// certificates, PHP-FPM pools and an nginx include pointing into a directory
// nginx cannot read, which then stops nginx from starting at all.
func ShouldRefuseRootBootstrap(uid int) bool {
	return uid == 0
}

// suiteAvailable reports whether a repository publishes a Release file.
func suiteAvailable(base, suite string) bool {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Head(base + suite + "/Release")
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode == http.StatusOK
}

// PPASuitePublished reports whether Ondrej's PPA covers a release.
func PPASuitePublished(suite string) bool { return suiteAvailable(ondrejPPADistsURL, suite) }

// SurySuitePublished reports whether packages.sury.org covers a release.
func SurySuitePublished(suite string) bool { return suiteAvailable(suryDistsURL, suite) }

// configureSuryRepository installs sury's keyring and apt source, and removes
// the PPA source so apt cannot pull packages built for an older release.
func (u *UbuntuInstaller) configureSuryRepository(codename string) error {
	keyringDeb := "/tmp/magebox-sury-keyring.deb"
	if err := u.RunCommand(fmt.Sprintf("curl -fsSL -o %s https://packages.sury.org/debsuryorg-archive-keyring.deb", keyringDeb)); err != nil {
		return fmt.Errorf("failed to download the sury keyring: %w", err)
	}
	defer func() { _ = os.Remove(keyringDeb) }()

	if err := u.RunSudo("dpkg", "-i", keyringDeb); err != nil {
		return fmt.Errorf("failed to install the sury keyring: %w", err)
	}

	source := fmt.Sprintf("deb [signed-by=%s] https://packages.sury.org/php/ %s main\n", suryKeyring, codename)
	if err := u.WriteFile(surySourceFile, source); err != nil {
		return fmt.Errorf("failed to write %s: %w", surySourceFile, err)
	}

	// A PPA source for a release it does not publish only produces 404s, and a
	// PPA pinned to an older suite offers packages that cannot be installed.
	for _, file := range ppaSourceFiles(codename) {
		if u.FileExists(file) {
			_ = u.RunSudo("rm", "-f", file)
		}
	}
	return nil
}
