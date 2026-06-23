package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/config"
	"qoliber/magebox/internal/project"
)

var openCmd = &cobra.Command{
	Use:   "open [worktree]",
	Short: "Open project in browser",
	Long: "Starts the project if not already running, then opens the first domain from .magebox.yaml in the default browser.\n\n" +
		"With a worktree argument, MageBox targets .claude/worktrees/<worktree>: it derives a .magebox.local.yaml from " +
		"that worktree's .magebox.yaml (appending .<worktree> to the project name and inserting .<worktree> before the " +
		"TLD of each domain host), then starts and opens the worktree as its own isolated project.",
	Args: cobra.MaximumNArgs(1),
	RunE: runOpen,
}

func init() {
	rootCmd.AddCommand(openCmd)
}

func runOpen(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	if len(args) == 1 {
		worktreePath, err := prepareWorktree(cwd, args[0])
		if err != nil {
			cli.PrintError("%v", err)
			return nil
		}
		// A worktree is a freshly-derived project: always start it so its nginx
		// vhost, PHP-FPM pool, DNS entry and SSL certificate are created — even
		// when the shared global services are already running for the base
		// project (which would otherwise make it look "fully running").
		return openProject(worktreePath, true)
	}

	return openProject(cwd, false)
}

// openProject ensures the project at projectPath is running and opens its first
// domain in the default browser. When forceStart is true the project is started
// unconditionally, rather than only when its services appear stopped.
func openProject(projectPath string, forceStart bool) error {
	cfg, ok := loadProjectConfig(projectPath)
	if !ok {
		return nil
	}

	if len(cfg.Domains) == 0 {
		cli.PrintError("No domains configured in .magebox.yaml")
		return nil
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	mgr := project.NewManager(p)
	if forceStart || !projectFullyRunning(mgr, projectPath) {
		if err := startProject(mgr, projectPath, true); err != nil {
			return err
		}
		fmt.Println()
	}

	domain := cfg.Domains[0]
	scheme := "https"
	if !domain.IsSSLEnabled() {
		scheme = "http"
	}
	url := fmt.Sprintf("%s://%s", scheme, domain.Host)

	cli.PrintInfo("Opening %s", cli.URL(url))

	var browser *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		browser = exec.Command("open", url)
	default:
		browser = exec.Command("xdg-open", url)
	}

	return browser.Start()
}

// prepareWorktree resolves the worktree directory at
// <baseDir>/.claude/worktrees/<name>, derives a .magebox.local.yaml from its
// .magebox.yaml (appending ".<name>" to the project name and inserting ".<name>"
// before the TLD of each domain host), and returns the worktree path. The
// override is rewritten on every call, so the command is idempotent.
func prepareWorktree(baseDir, name string) (string, error) {
	worktreePath := filepath.Join(baseDir, ".claude", "worktrees", name)

	info, err := os.Stat(worktreePath)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("worktree not found: %s", worktreePath)
	}

	srcPath := filepath.Join(worktreePath, config.ConfigFileName)
	src, err := os.ReadFile(srcPath)
	if os.IsNotExist(err) {
		// Fall back to the legacy ".magebox" filename.
		srcPath = filepath.Join(worktreePath, config.ConfigFileNameLegacy)
		src, err = os.ReadFile(srcPath)
	}
	if err != nil {
		return "", fmt.Errorf("cannot read %s in worktree: %w", config.ConfigFileName, err)
	}

	out, err := worktreeLocalConfig(src, name)
	if err != nil {
		return "", err
	}

	destPath := filepath.Join(worktreePath, config.LocalConfigFileName)
	if err := os.WriteFile(destPath, out, 0644); err != nil {
		return "", fmt.Errorf("cannot write %s: %w", config.LocalConfigFileName, err)
	}

	cli.PrintSuccess("Prepared %s", cli.Path(destPath))
	return worktreePath, nil
}

// worktreeLocalConfig transforms a .magebox.yaml document into a worktree-specific
// .magebox.local.yaml: it appends ".<suffix>" to the project name and inserts
// ".<suffix>" before the TLD of every domain host, preserving the original
// comments and layout.
func worktreeLocalConfig(src []byte, suffix string) ([]byte, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(src, &root); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", config.ConfigFileName, err)
	}

	if len(root.Content) == 0 || root.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%s: unexpected document structure", config.ConfigFileName)
	}
	doc := root.Content[0]

	if nameNode := mappingValue(doc, "name"); nameNode != nil {
		nameNode.Value += "." + suffix
	}

	if domainsNode := mappingValue(doc, "domains"); domainsNode != nil && domainsNode.Kind == yaml.SequenceNode {
		for _, item := range domainsNode.Content {
			if item.Kind != yaml.MappingNode {
				continue
			}
			if hostNode := mappingValue(item, "host"); hostNode != nil {
				hostNode.Value = suffixHostBeforeTLD(hostNode.Value, suffix)
			}
		}
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&root); err != nil {
		return nil, fmt.Errorf("failed to encode local config: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("failed to encode local config: %w", err)
	}
	return buf.Bytes(), nil
}

// mappingValue returns the value node for key in a YAML mapping node, or nil.
func mappingValue(m *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

// suffixHostBeforeTLD inserts ".<suffix>" before the final label (the TLD) of host:
// "shop.localhost" + "b2b-case" -> "shop.b2b-case.localhost". A host with no dot is
// treated as a bare TLD, so the suffix is prepended: "localhost" -> "b2b-case.localhost".
func suffixHostBeforeTLD(host, suffix string) string {
	idx := strings.LastIndex(host, ".")
	if idx == -1 {
		return suffix + "." + host
	}
	return host[:idx] + "." + suffix + host[idx:]
}

// projectFullyRunning reports whether all essential services for the project
// are running. Optional debug tooling (xdebug, blackfire) is ignored: those
// being disabled is a normal state and should not trigger a start.
func projectFullyRunning(mgr *project.Manager, projectPath string) bool {
	status, err := mgr.Status(projectPath)
	if err != nil {
		return false
	}
	for name, svc := range status.Services {
		if name == "xdebug" || name == "blackfire" {
			continue
		}
		if !svc.IsRunning {
			return false
		}
	}
	return true
}
