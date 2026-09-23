package bootstrap

import (
	"os/exec"
	"strings"
)

// DockerIssue explains why talking to Docker failed.
type DockerIssue int

const (
	// DockerOK means the daemon answered.
	DockerOK DockerIssue = iota
	// DockerNotRunning means the daemon is not up.
	DockerNotRunning
	// DockerPermissionDenied means the daemon is up but this user may not
	// reach its socket — usually a missing membership of the docker group.
	DockerPermissionDenied
	// DockerUnknownFailure means the command failed for another reason.
	DockerUnknownFailure
)

// Advice returns what the user should do about the issue.
func (d DockerIssue) Advice() string {
	switch d {
	case DockerNotRunning:
		return "start Docker and run bootstrap again"
	case DockerPermissionDenied:
		return "add yourself to the docker group, then log out and back in: sudo usermod -aG docker $USER"
	case DockerUnknownFailure:
		return "run 'docker info' to see what Docker reports"
	default:
		return ""
	}
}

// ClassifyDockerFailure reads the output of a failed "docker info".
//
// A permission error means the daemon is running and reachable by others;
// telling the user to start Docker would send them the wrong way.
func ClassifyDockerFailure(output string) DockerIssue {
	lower := strings.ToLower(output)
	switch {
	case strings.Contains(lower, "permission denied"):
		return DockerPermissionDenied
	case strings.Contains(lower, "cannot connect to the docker daemon"),
		strings.Contains(lower, "is the docker daemon running"):
		return DockerNotRunning
	default:
		return DockerUnknownFailure
	}
}

// CheckDocker reports whether Docker answers, and why it does not.
func (b *Bootstrapper) CheckDocker() DockerIssue {
	output, err := exec.Command("docker", "info").CombinedOutput()
	if err == nil {
		return DockerOK
	}
	return ClassifyDockerFailure(string(output))
}
