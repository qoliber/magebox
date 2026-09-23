package bootstrap

import "testing"

// "docker info" fails for two very different reasons, and telling the user to
// start a daemon that is already running sends them the wrong way.
func TestClassifyDockerFailure(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		want       DockerIssue
		wantAdvice string
	}{
		{
			name:       "user is not in the docker group",
			output:     "permission denied while trying to connect to the Docker daemon socket at unix:///var/run/docker.sock",
			want:       DockerPermissionDenied,
			wantAdvice: "docker group",
		},
		{
			name:       "daemon is down",
			output:     "Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?",
			want:       DockerNotRunning,
			wantAdvice: "start Docker",
		},
		{
			name:   "anything else",
			output: "something unexpected",
			want:   DockerUnknownFailure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyDockerFailure(tt.output)
			if got != tt.want {
				t.Errorf("ClassifyDockerFailure() = %v, want %v", got, tt.want)
			}
			if tt.wantAdvice != "" && !containsFold(got.Advice(), tt.wantAdvice) {
				t.Errorf("advice %q does not mention %q", got.Advice(), tt.wantAdvice)
			}
		})
	}
}

func containsFold(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if equalFold(haystack[i:i+len(needle)], needle) {
			return true
		}
	}
	return false
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		x, y := a[i], b[i]
		if 'A' <= x && x <= 'Z' {
			x += 'a' - 'A'
		}
		if 'A' <= y && y <= 'Z' {
			y += 'a' - 'A'
		}
		if x != y {
			return false
		}
	}
	return true
}
