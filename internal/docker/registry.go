package docker

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// dockerHubAPIBase is the Docker Hub registry API base URL.
// It can be overridden in tests to point at a local httptest server.
var dockerHubAPIBase = "https://registry.hub.docker.com"

// dockerRegistryHTTPClient is the HTTP client used for Docker Hub tag queries.
// It can be overridden in tests.
var dockerRegistryHTTPClient = &http.Client{Timeout: 5 * time.Second}

// resolvedTags caches Docker Hub tag resolutions for the lifetime of the process,
// so each major.minor version is queried at most once.
var resolvedTags sync.Map

// hubTagsResponse mirrors the Docker Hub tags API response.
type hubTagsResponse struct {
	Results []struct {
		Name string `json:"name"`
	} `json:"results"`
}

// isFullVersion reports whether version already contains a patch component
// (i.e. has the form major.minor.patch). Such versions are returned unchanged
// without querying Docker Hub.
func isFullVersion(version string) bool {
	return len(strings.SplitN(version, ".", 3)) == 3
}

// resolveDockerTagVersion returns the latest full major.minor.patch image tag for the
// given namespace/image and version. If the version already contains a patch component
// (three dot-separated parts) it is returned unchanged. On any network or parse error
// the input version is returned unchanged so the caller – and ultimately Docker – can
// produce an actionable error message.
func resolveDockerTagVersion(namespace, image, version string) string {
	if isFullVersion(version) {
		return version
	}

	cacheKey := fmt.Sprintf("%s/%s:%s", namespace, image, version)
	if cached, ok := resolvedTags.Load(cacheKey); ok {
		if v, ok := cached.(string); ok {
			return v
		}
	}

	resolved := queryDockerHubLatestTag(namespace, image, version)
	resolvedTags.Store(cacheKey, resolved)
	return resolved
}

// queryDockerHubLatestTag calls the Docker Hub tags API and returns the highest
// major.minor.patch tag that exactly matches the given major.minor prefix.
// Returns majorMinor unchanged on any error.
func queryDockerHubLatestTag(namespace, image, majorMinor string) string {
	url := fmt.Sprintf(
		"%s/v2/repositories/%s/%s/tags?name=%s.&page_size=50&ordering=-last_updated",
		dockerHubAPIBase, namespace, image, majorMinor,
	)

	resp, err := dockerRegistryHTTPClient.Get(url)
	if err != nil {
		return majorMinor
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return majorMinor
	}

	var result hubTagsResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result); err != nil {
		return majorMinor
	}

	prefix := majorMinor + "."
	best := ""
	for _, tag := range result.Results {
		if !strings.HasPrefix(tag.Name, prefix) {
			continue
		}
		// Only accept tags of the form major.minor.patch with no extra suffix
		// (e.g. reject "8.11.4-alpine" or "8.11.4.1").
		patch := tag.Name[len(prefix):]
		if _, err := strconv.Atoi(patch); err != nil {
			continue
		}
		if best == "" || compareVersionStrings(tag.Name, best) > 0 {
			best = tag.Name
		}
	}

	if best != "" {
		return best
	}
	return majorMinor
}

// compareVersionStrings compares two dot-separated version strings numerically,
// component by component. Returns a positive number if a > b, negative if a < b,
// and zero if they are equal.
func compareVersionStrings(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")
	n := len(aParts)
	if len(bParts) > n {
		n = len(bParts)
	}
	for i := range n { // range-over-integer requires Go 1.22+; project targets Go 1.24
		var aNum, bNum int
		if i < len(aParts) {
			aNum, _ = strconv.Atoi(aParts[i])
		}
		if i < len(bParts) {
			bNum, _ = strconv.Atoi(bParts[i])
		}
		if aNum != bNum {
			return aNum - bNum
		}
	}
	return 0
}
