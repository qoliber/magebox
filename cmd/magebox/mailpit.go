package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
)

const mailpitDefaultPort = "8025"

// getMailpitURL returns the Mailpit URL, reading the actual port from the running container.
// Falls back to the default port if the container is not running.
func getMailpitURL() string {
	portCmd := exec.Command("docker", "port", "magebox-mailpit", "8025")
	output, err := portCmd.Output()
	if err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
			line = strings.TrimSpace(line)
			if idx := strings.LastIndex(line, ":"); idx != -1 {
				port := line[idx+1:]
				return fmt.Sprintf("http://localhost:%s", port)
			}
		}
	}
	return fmt.Sprintf("http://localhost:%s", mailpitDefaultPort)
}

var mailpitCmd = &cobra.Command{
	Use:   "mailpit",
	Short: "Mailpit email testing UI",
	Long:  "Manage Mailpit email testing service",
}

var mailpitOpenCmd = &cobra.Command{
	Use:   "open",
	Short: "Open Mailpit in browser",
	Long:  "Opens the Mailpit web UI in the default browser",
	RunE:  runMailpitOpen,
}

var mailpitStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Mailpit status",
	Long:  "Shows Mailpit status and connection information",
	RunE:  runMailpitStatus,
}

func init() {
	mailpitCmd.AddCommand(mailpitOpenCmd)
	mailpitCmd.AddCommand(mailpitStatusCmd)
	rootCmd.AddCommand(mailpitCmd)
}

func runMailpitOpen(cmd *cobra.Command, args []string) error {
	// Check if container is running
	checkCmd := exec.Command("docker", "ps", "--filter", "name=magebox-mailpit", "--filter", "status=running", "-q")
	output, err := checkCmd.Output()
	if err != nil || len(strings.TrimSpace(string(output))) == 0 {
		cli.PrintError("Mailpit is not running")
		fmt.Println()
		cli.PrintInfo("Start global services with: magebox global start")
		return nil
	}

	url := getMailpitURL()
	cli.PrintInfo("Opening %s", cli.URL(url))

	var openCmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		openCmd = exec.Command("open", url)
	default:
		openCmd = exec.Command("xdg-open", url)
	}

	return openCmd.Start()
}

func runMailpitStatus(cmd *cobra.Command, args []string) error {
	cli.PrintTitle("Mailpit Status")
	fmt.Println()

	// Mailpit is always enabled
	fmt.Println("Enabled: " + cli.Success("yes (always on)"))

	// Check if container is running
	checkCmd := exec.Command("docker", "ps", "--filter", "name=magebox-mailpit", "--filter", "status=running", "-q")
	output, err := checkCmd.Output()
	if err == nil && len(strings.TrimSpace(string(output))) > 0 {
		fmt.Println("Status:  " + cli.Success("running"))
		fmt.Printf("Web UI:  %s\n", cli.Highlight(getMailpitURL()))
		fmt.Printf("SMTP:    %s\n", cli.Highlight("localhost:1025"))
	} else {
		fmt.Println("Status:  " + cli.Warning("stopped"))
		fmt.Println()
		cli.PrintInfo("Start global services with: magebox global start")
	}

	return nil
}
