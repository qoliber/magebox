// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/config"
	"qoliber/magebox/internal/docker"
)

var queueCmd = &cobra.Command{
	Use:   "queue",
	Short: "Manage message queues",
	Long: `Manage RabbitMQ message queues for Magento.

Use 'magebox queue status' to view queue status.
Use 'magebox queue flush' to purge all queues.
Use 'magebox queue consumer <name>' to run a specific queue consumer.`,
	RunE: runQueueStatus,
}

var queueStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show queue status",
	Long:  `Shows the status of RabbitMQ queues including message counts and consumer information.`,
	RunE:  runQueueStatus,
}

var queueFlushCmd = &cobra.Command{
	Use:   "flush",
	Short: "Flush all queues",
	Long:  `Purges all messages from RabbitMQ queues. Use with caution - this cannot be undone.`,
	RunE:  runQueueFlush,
}

var queueConsumerCmd = &cobra.Command{
	Use:   "consumer [name]",
	Short: "Run a queue consumer",
	Long: `Runs a Magento queue consumer. If no name is provided, lists available consumers.

Examples:
  magebox queue consumer                    # List available consumers
  magebox queue consumer product_action_attribute.update
  magebox queue consumer --all             # Start all consumers`,
	RunE: runQueueConsumer,
}

var consumerAll bool
var consumerMaxMessages int

func init() {
	queueConsumerCmd.Flags().BoolVar(&consumerAll, "all", false, "Start all consumers")
	queueConsumerCmd.Flags().IntVar(&consumerMaxMessages, "max-messages", 0, "Maximum messages to process (0 = unlimited)")

	queueCmd.AddCommand(queueStatusCmd)
	queueCmd.AddCommand(queueFlushCmd)
	queueCmd.AddCommand(queueConsumerCmd)
	rootCmd.AddCommand(queueCmd)
}

// RabbitMQ API types
type rabbitQueue struct {
	Name      string `json:"name"`
	Messages  int    `json:"messages"`
	Consumers int    `json:"consumers"`
	State     string `json:"state"`
	Vhost     string `json:"vhost"`
}

func runQueueStatus(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	cfg, err := config.LoadFromPath(cwd)
	if err != nil {
		return err
	}

	cli.PrintTitle("Queue Status")
	fmt.Printf("Project: %s\n", cli.Highlight(cfg.Name))
	fmt.Println()

	if !cfg.Services.HasRabbitMQ() {
		cli.PrintInfo("RabbitMQ is not configured for this project")
		cli.PrintInfo("Add 'rabbitmq: true' to .magebox.yaml to enable")
		return nil
	}

	// Check if RabbitMQ is running
	p, err := getPlatform()
	if err != nil {
		return err
	}

	composeGen := docker.NewComposeGenerator(p)
	dockerCtrl := docker.NewDockerController(composeGen.ComposeFilePath())

	if !dockerCtrl.IsServiceRunning("rabbitmq") {
		cli.PrintWarning("RabbitMQ is not running")
		cli.PrintInfo("Start with: magebox start")
		return nil
	}

	// Get queue list from RabbitMQ Management API
	queues, err := getRabbitMQQueues()
	if err != nil {
		cli.PrintWarning("Could not fetch queue status: %v", err)
		cli.PrintInfo("RabbitMQ Management UI: http://localhost:15672")
		cli.PrintInfo("Default credentials: magebox / magebox")
		return nil
	}

	if len(queues) == 0 {
		cli.PrintInfo("No queues found")
		cli.PrintInfo("RabbitMQ Management UI: http://localhost:15672")
		return nil
	}

	// Display queues
	fmt.Println(cli.Header("RabbitMQ Queues"))
	fmt.Println()
	fmt.Printf("  %-50s %10s %10s %10s\n", "QUEUE", "MESSAGES", "CONSUMERS", "STATE")
	fmt.Printf("  %-50s %10s %10s %10s\n", strings.Repeat("-", 50), strings.Repeat("-", 10), strings.Repeat("-", 10), strings.Repeat("-", 10))

	totalMessages := 0
	for _, q := range queues {
		stateDisplay := cli.Success(q.State)
		if q.State != "running" {
			stateDisplay = cli.Warning(q.State)
		}

		msgDisplay := fmt.Sprintf("%d", q.Messages)
		if q.Messages > 0 {
			msgDisplay = cli.Warning(msgDisplay)
		}

		fmt.Printf("  %-50s %10s %10d %10s\n", truncate(q.Name, 50), msgDisplay, q.Consumers, stateDisplay)
		totalMessages += q.Messages
	}

	fmt.Println()
	fmt.Printf("Total queues: %d, Total messages: %s\n", len(queues), cli.Highlight(fmt.Sprintf("%d", totalMessages)))
	fmt.Println()

	cli.PrintInfo("RabbitMQ Management UI: http://localhost:15672")
	cli.PrintInfo("Default credentials: magebox / magebox")

	return nil
}

func runQueueFlush(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	cfg, err := config.LoadFromPath(cwd)
	if err != nil {
		return err
	}

	if !cfg.Services.HasRabbitMQ() {
		cli.PrintInfo("RabbitMQ is not configured for this project")
		return nil
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	composeGen := docker.NewComposeGenerator(p)
	dockerCtrl := docker.NewDockerController(composeGen.ComposeFilePath())

	if !dockerCtrl.IsServiceRunning("rabbitmq") {
		cli.PrintWarning("RabbitMQ is not running")
		return nil
	}

	cli.PrintTitle("Flushing Queues")
	fmt.Println()

	// Get queue list
	queues, err := getRabbitMQQueues()
	if err != nil {
		return fmt.Errorf("failed to get queue list: %w", err)
	}

	if len(queues) == 0 {
		cli.PrintInfo("No queues to flush")
		return nil
	}

	// Purge each queue
	purged := 0
	for _, q := range queues {
		if q.Messages > 0 {
			fmt.Printf("Purging %s (%d messages)... ", q.Name, q.Messages)
			if err := purgeRabbitMQQueue(q.Vhost, q.Name); err != nil {
				fmt.Println(cli.Warning("failed"))
			} else {
				fmt.Println(cli.Success("done"))
				purged++
			}
		}
	}

	fmt.Println()
	if purged > 0 {
		cli.PrintSuccess("Purged %d queue(s)", purged)
	} else {
		cli.PrintInfo("No queues had messages to purge")
	}

	return nil
}

func runQueueConsumer(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	cfg, err := config.LoadFromPath(cwd)
	if err != nil {
		return err
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	// Check if this is a Magento project
	binMagento := filepath.Join(cwd, "bin", "magento")
	if _, err := os.Stat(binMagento); os.IsNotExist(err) {
		cli.PrintWarning("bin/magento not found - is this a Magento project?")
		return nil
	}

	phpBinary := p.PHPBinary(cfg.PHP)

	// If --all flag, start all consumers
	if consumerAll {
		return runAllConsumers(cwd, phpBinary)
	}

	// If no args, list available consumers
	if len(args) == 0 {
		return listConsumers(cwd, phpBinary)
	}

	// Run specific consumer
	consumerName := args[0]

	cli.PrintTitle("Running Queue Consumer")
	fmt.Printf("Consumer: %s\n", cli.Highlight(consumerName))
	fmt.Println()

	cmdArgs := []string{filepath.Join(cwd, "bin", "magento"), "queue:consumers:start", consumerName}
	if consumerMaxMessages > 0 {
		cmdArgs = append(cmdArgs, fmt.Sprintf("--max-messages=%d", consumerMaxMessages))
	}

	consumer := exec.Command(phpBinary, cmdArgs...)
	consumer.Dir = cwd
	consumer.Stdout = os.Stdout
	consumer.Stderr = os.Stderr
	consumer.Stdin = os.Stdin

	cli.PrintInfo("Press Ctrl+C to stop the consumer")
	fmt.Println()

	return consumer.Run()
}

func listConsumers(cwd, phpBinary string) error {
	cli.PrintTitle("Available Queue Consumers")
	fmt.Println()

	cmd := exec.Command(phpBinary, filepath.Join(cwd, "bin", "magento"), "queue:consumers:list")
	cmd.Dir = cwd
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list consumers: %w", err)
	}

	consumers := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(consumers) == 0 || (len(consumers) == 1 && consumers[0] == "") {
		cli.PrintInfo("No consumers found")
		return nil
	}

	for _, c := range consumers {
		c = strings.TrimSpace(c)
		if c != "" {
			fmt.Printf("  %s %s\n", cli.Success(""), c)
		}
	}

	fmt.Println()
	cli.PrintInfo("Run a consumer with: magebox queue consumer <name>")
	cli.PrintInfo("Run all consumers with: magebox queue consumer --all")

	return nil
}

func runAllConsumers(cwd, phpBinary string) error {
	cli.PrintTitle("Starting All Queue Consumers")
	fmt.Println()

	// Get list of consumers
	cmd := exec.Command(phpBinary, filepath.Join(cwd, "bin", "magento"), "queue:consumers:list")
	cmd.Dir = cwd
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list consumers: %w", err)
	}

	consumers := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(consumers) == 0 {
		cli.PrintInfo("No consumers found")
		return nil
	}

	// Start each consumer using Magento's cron-style runner
	cmdArgs := []string{filepath.Join(cwd, "bin", "magento"), "cron:run", "--group=consumers"}

	cli.PrintInfo("Starting consumers via cron group...")
	fmt.Println()

	consumer := exec.Command(phpBinary, cmdArgs...)
	consumer.Dir = cwd
	consumer.Stdout = os.Stdout
	consumer.Stderr = os.Stderr

	if err := consumer.Run(); err != nil {
		return fmt.Errorf("failed to start consumers: %w", err)
	}

	cli.PrintSuccess("Consumer cron group executed")
	cli.PrintInfo("For continuous processing, set up cron or run individual consumers")

	return nil
}

func getRabbitMQQueues() ([]rabbitQueue, error) {
	// Use RabbitMQ Management API
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://localhost:15672/api/queues", nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(docker.DefaultRabbitMQUser, docker.DefaultRabbitMQPass)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var queues []rabbitQueue
	if err := json.Unmarshal(body, &queues); err != nil {
		return nil, err
	}

	return queues, nil
}

func purgeRabbitMQQueue(vhost, queueName string) error {
	// Use RabbitMQ Management API to purge queue
	client := &http.Client{}

	// URL encode vhost (/ becomes %2F)
	encodedVhost := strings.ReplaceAll(vhost, "/", "%2F")
	url := fmt.Sprintf("http://localhost:15672/api/queues/%s/%s/contents", encodedVhost, queueName)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(docker.DefaultRabbitMQUser, docker.DefaultRabbitMQPass)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	return nil
}
