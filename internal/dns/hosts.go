package dns

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/qoliber/magebox/internal/platform"
)

const (
	// MageBoxStartMarker marks the beginning of MageBox entries
	MageBoxStartMarker = "# >>> MageBox managed hosts - do not edit manually >>>"
	// MageBoxEndMarker marks the end of MageBox entries
	MageBoxEndMarker = "# <<< MageBox managed hosts <<<"
)

// HostsManager manages /etc/hosts entries
type HostsManager struct {
	platform  *platform.Platform
	hostsFile string
}

// HostEntry represents a single hosts file entry
type HostEntry struct {
	IP     string
	Domain string
}

// NewHostsManager creates a new hosts manager
func NewHostsManager(p *platform.Platform) *HostsManager {
	return &HostsManager{
		platform:  p,
		hostsFile: p.HostsFilePath(),
	}
}

// AddDomains adds domains to /etc/hosts
func (m *HostsManager) AddDomains(domains []string) error {
	// Read current hosts file
	content, err := os.ReadFile(m.hostsFile)
	if err != nil {
		return fmt.Errorf("failed to read hosts file: %w", err)
	}

	// Get existing MageBox domains
	existingDomains := m.extractMageBoxDomains(string(content))

	// Merge with new domains
	allDomains := make(map[string]bool)
	for _, d := range existingDomains {
		allDomains[d] = true
	}
	for _, d := range domains {
		allDomains[d] = true
	}

	// Convert back to slice
	domainList := make([]string, 0, len(allDomains))
	for d := range allDomains {
		domainList = append(domainList, d)
	}

	return m.writeDomains(string(content), domainList)
}

// RemoveDomains removes domains from /etc/hosts
func (m *HostsManager) RemoveDomains(domains []string) error {
	// Read current hosts file
	content, err := os.ReadFile(m.hostsFile)
	if err != nil {
		return fmt.Errorf("failed to read hosts file: %w", err)
	}

	// Get existing MageBox domains
	existingDomains := m.extractMageBoxDomains(string(content))

	// Remove specified domains
	toRemove := make(map[string]bool)
	for _, d := range domains {
		toRemove[d] = true
	}

	remainingDomains := make([]string, 0)
	for _, d := range existingDomains {
		if !toRemove[d] {
			remainingDomains = append(remainingDomains, d)
		}
	}

	return m.writeDomains(string(content), remainingDomains)
}

// RemoveAllDomains removes all MageBox-managed domains
func (m *HostsManager) RemoveAllDomains() error {
	content, err := os.ReadFile(m.hostsFile)
	if err != nil {
		return fmt.Errorf("failed to read hosts file: %w", err)
	}

	return m.writeDomains(string(content), nil)
}

// ListDomains returns all MageBox-managed domains
func (m *HostsManager) ListDomains() ([]string, error) {
	content, err := os.ReadFile(m.hostsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read hosts file: %w", err)
	}

	return m.extractMageBoxDomains(string(content)), nil
}

// DomainExists checks if a domain is in /etc/hosts
func (m *HostsManager) DomainExists(domain string) bool {
	domains, err := m.ListDomains()
	if err != nil {
		return false
	}

	for _, d := range domains {
		if d == domain {
			return true
		}
	}
	return false
}

// extractMageBoxDomains extracts MageBox-managed domains from hosts content
func (m *HostsManager) extractMageBoxDomains(content string) []string {
	domains := make([]string, 0)
	inMageBoxSection := false

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == MageBoxStartMarker {
			inMageBoxSection = true
			continue
		}
		if line == MageBoxEndMarker {
			inMageBoxSection = false
			continue
		}

		if inMageBoxSection && !strings.HasPrefix(line, "#") && line != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				// Add all domains from the line (excluding IP)
				for _, domain := range parts[1:] {
					domains = append(domains, domain)
				}
			}
		}
	}

	return domains
}

// writeDomains writes domains to the hosts file
func (m *HostsManager) writeDomains(currentContent string, domains []string) error {
	// Remove existing MageBox section
	newContent := m.removeMageBoxSection(currentContent)

	// Add new MageBox section if we have domains
	if len(domains) > 0 {
		newContent = strings.TrimRight(newContent, "\n") + "\n\n"
		newContent += MageBoxStartMarker + "\n"
		for _, domain := range domains {
			newContent += fmt.Sprintf("127.0.0.1 %s\n", domain)
		}
		newContent += MageBoxEndMarker + "\n"
	}

	// Write to temp file first
	tmpFile, err := os.CreateTemp("", "hosts-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.WriteString(newContent); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	// Copy to /etc/hosts with sudo
	cmd := exec.Command("sudo", "cp", tmpPath, m.hostsFile)
	if err := cmd.Run(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to update hosts file (sudo required): %w", err)
	}

	os.Remove(tmpPath)
	return nil
}

// removeMageBoxSection removes the MageBox section from hosts content
func (m *HostsManager) removeMageBoxSection(content string) string {
	var result strings.Builder
	inMageBoxSection := false

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == MageBoxStartMarker {
			inMageBoxSection = true
			continue
		}
		if trimmed == MageBoxEndMarker {
			inMageBoxSection = false
			continue
		}

		if !inMageBoxSection {
			result.WriteString(line)
			result.WriteString("\n")
		}
	}

	return result.String()
}

// GenerateMageBoxSection generates the MageBox section for hosts file
func GenerateMageBoxSection(domains []string) string {
	if len(domains) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(MageBoxStartMarker + "\n")
	for _, domain := range domains {
		sb.WriteString(fmt.Sprintf("127.0.0.1 %s\n", domain))
	}
	sb.WriteString(MageBoxEndMarker + "\n")
	return sb.String()
}

// ParseHostsLine parses a single hosts file line
func ParseHostsLine(line string) *HostEntry {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return nil
	}

	parts := strings.Fields(line)
	if len(parts) < 2 {
		return nil
	}

	return &HostEntry{
		IP:     parts[0],
		Domain: parts[1],
	}
}

// FormatHostsLine formats a hosts entry as a line
func FormatHostsLine(entry HostEntry) string {
	return fmt.Sprintf("%s %s", entry.IP, entry.Domain)
}
