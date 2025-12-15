// Package testmode provides test mode detection for MageBox
// When MAGEBOX_TEST_MODE=1, certain operations are skipped:
// - Docker container management
// - DNS configuration (dnsmasq, hosts file)
package testmode

import "os"

// IsEnabled returns true if MageBox is running in test mode
func IsEnabled() bool {
	return os.Getenv("MAGEBOX_TEST_MODE") == "1"
}

// SkipDocker returns true if Docker operations should be skipped
func SkipDocker() bool {
	return IsEnabled()
}

// SkipDNS returns true if DNS operations should be skipped
func SkipDNS() bool {
	return IsEnabled()
}
