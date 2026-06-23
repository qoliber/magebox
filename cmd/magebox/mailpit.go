package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
)

const (
	mailpitDefaultPort = "8025"
	mailpitContainer   = "magebox-mailpit"
	mailpitService     = "mailpit"
)

// getMailpitURL returns the Mailpit URL, reading the actual port from the running container.
// Falls back to the default port if the container is not running.
func getMailpitURL() string {
	portCmd := exec.Command("docker", "port", mailpitContainer, "8025")
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
	Long:  "Opens the Mailpit web UI in the default browser, starting it if needed",
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
	if !isContainerRunning(mailpitContainer) {
		p, err := getPlatform()
		if err != nil {
			return err
		}
		fmt.Print("Mailpit is not running, starting... ")
		if err := ensureGlobalServiceRunning(p, mailpitService); err != nil {
			fmt.Println(cli.Error("failed"))
			cli.PrintError("%v", err)
			return nil
		}
		fmt.Println(cli.Success("done"))
	}

	url := getMailpitURL()
	cli.PrintInfo("Opening %s", cli.URL(url))
	return openInBrowser(url)
}

func runMailpitStatus(cmd *cobra.Command, args []string) error {
	cli.PrintTitle("Mailpit Status")
	fmt.Println()

	// Mailpit is always enabled
	fmt.Println("Enabled: " + cli.Success("yes (always on)"))

	if isContainerRunning(mailpitContainer) {
		fmt.Println("Status:  " + cli.Success("running"))
		fmt.Printf("Web UI:  %s\n", cli.Highlight(getMailpitURL()))
		fmt.Printf("SMTP:    %s\n", cli.Highlight("localhost:1025"))
	} else {
		fmt.Println("Status:  " + cli.Warning("stopped"))
		fmt.Println()
		cli.PrintInfo("Start with: magebox mailpit open")
	}

	return nil
}
