package installer

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// ubuntuCodenames lists Ubuntu releases, newest first. It is used to find the
// newest suite a PPA publishes when it has nothing for the running release.
var ubuntuCodenames = []string{
	"resolute", // 26.04 LTS
	"questing", // 25.10
	"plucky",   // 25.04
	"oracular", // 24.10
	"noble",    // 24.04 LTS
	"mantic",   // 23.10
	"lunar",    // 23.04
	"kinetic",  // 22.10
	"jammy",    // 22.04 LTS
	"focal",    // 20.04 LTS
}

// ondrejPPADists is where the PHP PPA publishes its suites.
const ondrejPPADists = "https://ppa.launchpadcontent.net/ondrej/php/ubuntu/dists/"

// pickPPASuite returns the suite to configure and whether it is a fallback.
//
// Ondrej's PHP PPA lags new Ubuntu releases by months, and on a release it does
// not cover there is no php8.1 … php8.4 at all. Pinning the newest published
// suite keeps those versions installable; packages are built against an older
// but compatible Ubuntu.
func pickPPASuite(current string, published func(string) bool) (suite string, fallback bool) {
	if published(current) {
		return current, false
	}

	start := 0
	for i, codename := range ubuntuCodenames {
		if codename == current {
			start = i + 1
			break
		}
	}

	for _, codename := range ubuntuCodenames[start:] {
		if codename == current {
			continue
		}
		if published(codename) {
			return codename, true
		}
	}

	// Nothing reachable (offline, or the PPA moved): leave the release as is.
	return current, false
}

// ppaSuitePublished reports whether the PHP PPA has a Release file for a suite.
func ppaSuitePublished(suite string) bool {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Head(ondrejPPADists + suite + "/Release")
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode == http.StatusOK
}

// currentUbuntuCodename reads the running release's codename, "" if unknown.
func currentUbuntuCodename() string {
	content, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(content), "\n") {
		if value, ok := strings.CutPrefix(line, "VERSION_CODENAME="); ok {
			return strings.Trim(strings.TrimSpace(value), `"`)
		}
	}
	return ""
}

// ppaSourceFiles returns the files add-apt-repository may have written for a
// codename, newest apt format first.
func ppaSourceFiles(codename string) []string {
	return []string{
		fmt.Sprintf("/etc/apt/sources.list.d/ondrej-ubuntu-php-%s.sources", codename),
		fmt.Sprintf("/etc/apt/sources.list.d/ondrej-ubuntu-php-%s.list", codename),
	}
}

// pinPPASuite rewrites the suite in the PPA source file, leaving the signing
// key and every other line untouched.
func (u *UbuntuInstaller) pinPPASuite(from, to string) error {
	for _, file := range ppaSourceFiles(from) {
		if !u.FileExists(file) {
			continue
		}
		// deb822 keeps the suite on its own "Suites:" line; the older one-line
		// format has it between the URI and the component.
		expression := fmt.Sprintf("/^Suites:/s/%s/%s/; /^deb /s/ %s / %s /", from, to, from, to)
		if err := u.RunSudo("sed", "-i", "-e", expression, file); err != nil {
			return fmt.Errorf("failed to pin the PHP PPA to %s in %s: %w", to, file, err)
		}
		return nil
	}
	return fmt.Errorf("could not find the PHP PPA source file for %s", from)
}

// configurePHPRepository adds Ondrej's PHP PPA, falling back to the newest
// suite it publishes when the running release is not covered yet.
func (u *UbuntuInstaller) configurePHPRepository() error {
	if err := u.RunCommand("sudo add-apt-repository -y ppa:ondrej/php"); err != nil {
		return fmt.Errorf("failed to add Ondrej PPA: %w", err)
	}

	codename := currentUbuntuCodename()
	if codename == "" {
		return nil
	}

	suite, fallback := pickPPASuite(codename, ppaSuitePublished)
	if !fallback {
		return nil
	}

	fmt.Printf("  The PHP PPA does not publish packages for %s yet; using its %s packages instead.\n", codename, suite)
	return u.pinPPASuite(codename, suite)
}
