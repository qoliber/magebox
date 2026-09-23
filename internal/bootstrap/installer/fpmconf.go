package installer

import (
	"regexp"
	"strings"
)

// mageboxPoolInclude matches any MageBox pools include, whatever home it names.
var mageboxPoolInclude = regexp.MustCompile(`^\s*include\s*=\s*\S*/\.magebox/php/pools/\S*\.conf\s*$`)

// NormalizeFPMIncludes returns php-fpm.conf content holding exactly one MageBox
// include, the given one, and reports whether anything changed.
//
// Appending on every run left duplicates, which define the same pool twice, and
// a run as root added an include under /root that matches nothing. Either stops
// PHP-FPM from starting, and a PHP-FPM that cannot start breaks every later
// dpkg operation on the package.
func NormalizeFPMIncludes(content, want string) (string, bool) {
	lines := strings.Split(content, "\n")
	kept := make([]string, 0, len(lines)+1)
	seen := false
	changed := false

	for _, line := range lines {
		if !mageboxPoolInclude.MatchString(line) {
			kept = append(kept, line)
			continue
		}
		if strings.TrimSpace(line) == want && !seen {
			seen = true
			kept = append(kept, line)
			continue
		}
		// A duplicate, or an include naming a different home.
		changed = true
	}

	result := strings.Join(kept, "\n")
	if !seen {
		if !strings.HasSuffix(result, "\n") {
			result += "\n"
		}
		result += want + "\n"
		changed = true
	}
	return result, changed
}
