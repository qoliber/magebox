/**
 * Created by Qoliber
 *
 * @category    Qoliber
 * @package     MageBox
 * @author      Jakub Winkler <jwinkler@qoliber.com>
 */

package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
	"github.com/qoliber/magebox/internal/teamserver"
)

var (
	auditFrom   string
	auditTo     string
	auditUser   string
	auditAction string
	auditFormat string
	auditLimit  int
)

var serverAuditCmd = &cobra.Command{
	Use:   "audit",
	Short: "View audit log",
	Long: `View the team server audit log.

The audit log records all security-relevant actions including user creation,
environment access, key deployments, and authentication events.

Examples:
  magebox server audit
  magebox server audit --user alice
  magebox server audit --action USER_CREATE
  magebox server audit --from 2024-01-01 --to 2024-12-31
  magebox server audit --format csv > audit.csv`,
	RunE: runServerAudit,
}

var serverAuditVerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify audit log integrity",
	Long: `Verify the integrity of the audit log hash chain.

This checks that no entries have been tampered with.`,
	RunE: runServerAuditVerify,
}

func init() {
	serverAuditCmd.Flags().StringVar(&auditFrom, "from", "", "Start date (YYYY-MM-DD)")
	serverAuditCmd.Flags().StringVar(&auditTo, "to", "", "End date (YYYY-MM-DD)")
	serverAuditCmd.Flags().StringVar(&auditUser, "user", "", "Filter by username")
	serverAuditCmd.Flags().StringVar(&auditAction, "action", "", "Filter by action (USER_CREATE, USER_REMOVE, ENV_ACCESS, etc.)")
	serverAuditCmd.Flags().StringVar(&auditFormat, "format", "table", "Output format: table, json, csv")
	serverAuditCmd.Flags().IntVar(&auditLimit, "limit", 100, "Maximum entries to return")

	serverAuditCmd.AddCommand(serverAuditVerifyCmd)
	serverCmd.AddCommand(serverAuditCmd)
}

func runServerAudit(cmd *cobra.Command, args []string) error {
	adminToken, err := getAdminToken()
	if err != nil {
		return err
	}

	// Build query parameters
	params := url.Values{}
	if auditFrom != "" {
		// Parse and format as RFC3339
		t, err := time.Parse("2006-01-02", auditFrom)
		if err != nil {
			return fmt.Errorf("invalid from date format (use YYYY-MM-DD): %w", err)
		}
		params.Set("from", t.Format(time.RFC3339))
	}
	if auditTo != "" {
		t, err := time.Parse("2006-01-02", auditTo)
		if err != nil {
			return fmt.Errorf("invalid to date format (use YYYY-MM-DD): %w", err)
		}
		// Set to end of day
		t = t.Add(24*time.Hour - time.Second)
		params.Set("to", t.Format(time.RFC3339))
	}
	if auditUser != "" {
		params.Set("user", auditUser)
	}
	if auditAction != "" {
		params.Set("action", auditAction)
	}

	endpoint := "/api/admin/audit"
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	resp, err := apiRequest("GET", endpoint, nil, adminToken)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp teamserver.ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("failed to get audit log: %s", errResp.Error)
	}

	var entries []auditEntryDisplay

	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Apply limit
	if auditLimit > 0 && len(entries) > auditLimit {
		entries = entries[:auditLimit]
	}

	switch auditFormat {
	case "json":
		return outputAuditJSON(entries)
	case "csv":
		return outputAuditCSV(entries)
	default:
		return outputAuditTable(entries)
	}
}

type auditEntryDisplay struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	UserName  string    `json:"user_name"`
	Action    string    `json:"action"`
	Details   string    `json:"details"`
	IPAddress string    `json:"ip_address"`
}

func outputAuditTable(entries []auditEntryDisplay) error {
	if len(entries) == 0 {
		cli.PrintInfo("No audit entries found")
		return nil
	}

	cli.PrintTitle("Audit Log")
	fmt.Println()

	for _, entry := range entries {
		timestamp := entry.Timestamp.Local().Format("2006-01-02 15:04:05")

		// Color-code by action type
		actionDisplay := entry.Action
		switch {
		case entry.Action == "AUTH_FAILED":
			actionDisplay = cli.Error(entry.Action)
		case entry.Action == "USER_REMOVE" || entry.Action == "ENV_REMOVE":
			actionDisplay = cli.Warning(entry.Action)
		case entry.Action == "USER_CREATE" || entry.Action == "USER_JOIN":
			actionDisplay = cli.Success(entry.Action)
		}

		fmt.Printf("  [%s] %s\n", timestamp, actionDisplay)
		if entry.UserName != "" {
			fmt.Printf("      User: %s\n", entry.UserName)
		}
		if entry.Details != "" {
			fmt.Printf("      %s\n", entry.Details)
		}
		if entry.IPAddress != "" {
			fmt.Printf("      IP: %s\n", entry.IPAddress)
		}
		fmt.Println()
	}

	fmt.Printf("Showing %d entries\n", len(entries))

	return nil
}

func outputAuditJSON(entries []auditEntryDisplay) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(entries)
}

func outputAuditCSV(entries []auditEntryDisplay) error {
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	// Write header
	_ = writer.Write([]string{"ID", "Timestamp", "User", "Action", "Details", "IP Address"})

	for _, entry := range entries {
		_ = writer.Write([]string{
			fmt.Sprintf("%d", entry.ID),
			entry.Timestamp.Format(time.RFC3339),
			entry.UserName,
			entry.Action,
			entry.Details,
			entry.IPAddress,
		})
	}

	return nil
}

func runServerAuditVerify(cmd *cobra.Command, args []string) error {
	adminToken, err := getAdminToken()
	if err != nil {
		return err
	}

	cli.PrintInfo("Verifying audit log integrity...")
	fmt.Println()

	// Get all audit entries
	resp, err := apiRequest("GET", "/api/admin/audit?limit=10000", nil, adminToken)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp teamserver.ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("failed to get audit log: %s", errResp.Error)
	}

	var entries []teamserver.AuditEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if len(entries) == 0 {
		cli.PrintInfo("Audit log is empty")
		return nil
	}

	// Reverse to chronological order
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	// Verify chain
	valid, idx := teamserver.VerifyAuditChain(entries)

	if valid {
		cli.PrintSuccess("Audit log integrity verified!")
		cli.PrintInfo("Total entries: %d", len(entries))
		cli.PrintInfo("First entry:   %s", entries[0].Timestamp.Format("2006-01-02 15:04:05"))
		cli.PrintInfo("Last entry:    %s", entries[len(entries)-1].Timestamp.Format("2006-01-02 15:04:05"))
	} else {
		cli.PrintError("Audit log integrity check FAILED!")
		cli.PrintError("Tampered entry at index: %d (ID: %d)", idx, entries[idx].ID)
		cli.PrintError("Timestamp: %s", entries[idx].Timestamp.Format("2006-01-02 15:04:05"))
		cli.PrintError("Action: %s", entries[idx].Action)
		return fmt.Errorf("audit log integrity check failed")
	}

	return nil
}
