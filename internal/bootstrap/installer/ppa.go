package installer

import (
	"fmt"
	"os"
	"strings"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/verbose"
)

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

// configurePHPRepository points apt at a PHP repository that covers this
// release, so php8.1 … php8.4 can actually be installed.
func (u *UbuntuInstaller) configurePHPRepository() error {
	codename := currentUbuntuCodename()
	repo := PickPHPRepository(codename, PPASuitePublished, SurySuitePublished)

	switch repo.Kind {
	case PHPRepoPPA:
		if err := u.RunCommand("sudo add-apt-repository -y ppa:ondrej/php"); err != nil {
			verbose.Debug("add-apt-repository reported an error: %v", err)
		}
	case PHPRepoSury:
		fmt.Printf("  The PHP PPA does not cover %s; using packages.sury.org instead.\n", codename)
		if err := u.configureSuryRepository(repo.Suite); err != nil {
			return err
		}
	default:
		cli.PrintWarning("No PHP repository covers %s yet; only the PHP version shipped by Ubuntu can be installed.", codename)
	}
	return nil
}
