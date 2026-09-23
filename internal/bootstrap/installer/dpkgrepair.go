package installer

import (
	"os/exec"
	"strings"

	"qoliber/magebox/internal/verbose"
)

// dpkgAuditReportsBroken reports whether dpkg found packages left in a broken
// state. Its output is empty when everything is configured.
func dpkgAuditReportsBroken(auditOutput string) bool {
	return strings.TrimSpace(auditOutput) != ""
}

// repairBrokenPackages finishes configuring packages left half-installed.
//
// One such package makes every apt install exit 100, whatever it was asked to
// install, so bootstrap could not install anything and could not repair the
// repository that would have let it.
func (u *UbuntuInstaller) repairBrokenPackages() {
	output, err := exec.Command("dpkg", "--audit").Output()
	if err != nil || !dpkgAuditReportsBroken(string(output)) {
		return
	}

	verbose.Debug("dpkg reports broken packages, running dpkg --configure -a")
	if err := u.RunSudo("dpkg", "--configure", "-a"); err != nil {
		verbose.Debug("dpkg --configure -a did not succeed: %v", err)
	}
}
