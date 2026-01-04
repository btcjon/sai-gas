package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/mail"
	"github.com/steveyegge/gastown/internal/style"
	"github.com/steveyegge/gastown/internal/workspace"
)

// WorkerStatusReport represents a status update from a worker about their task.
type WorkerStatusReport struct {
	IssueID    string    `json:"issue_id"`
	Status     string    `json:"status"` // started|progress|blocked|completed|failed
	Progress   int       `json:"progress,omitempty"`
	Reason     string    `json:"reason,omitempty"`
	Message    string    `json:"message,omitempty"`
	ReportedAt time.Time `json:"reported_at"`
}

// Worker status flags
var (
	workerStatusMessage string
	workerProgressPct   int
)

var workerCmd = &cobra.Command{
	Use:     "worker",
	Aliases: []string{"w"},
	GroupID: GroupComm,
	Short:   "Report work status to refinery",
	Long: `Report work status from a polecat to the refinery.

The refinery uses these status updates to track work progress and detect
issues that need escalation.

COMMANDS:
  started     Report that work has started
  progress    Report percentage progress (0-100)
  blocked     Report that work is blocked with a reason
  completed   Report that work is completed
  failed      Report that work has failed with a reason

Each command sends a structured JSON status update to the refinery via mail.
The refinery processes these updates to maintain real-time convoy status.`,
	RunE: requireSubcommand,
}

var workerStartedCmd = &cobra.Command{
	Use:   "started <issue-id>",
	Short: "Report work started",
	Long: `Report that work on an issue has started.

This notifies the refinery that the worker has begun processing the issue.

Examples:
  gt worker started gt-abc123
  gt worker started gt-abc123 -m "Starting implementation phase"`,
	Args: cobra.ExactArgs(1),
	RunE: runWorkerStarted,
}

var workerProgressCmd = &cobra.Command{
	Use:   "progress <issue-id> <0-100>",
	Short: "Report work progress percentage",
	Long: `Report progress on work as a percentage (0-100).

This notifies the refinery of the current progress percentage.

Examples:
  gt worker progress gt-abc123 50
  gt worker progress gt-abc123 75 -m "Implementation 75% complete, testing next"`,
	Args: cobra.ExactArgs(2),
	RunE: runWorkerProgress,
}

var workerBlockedCmd = &cobra.Command{
	Use:   "blocked <issue-id> <reason>",
	Short: "Report work blocked",
	Long: `Report that work is blocked and cannot proceed.

The reason explains what is blocking the work. The refinery may escalate
this to witness or mayor depending on the reason.

Examples:
  gt worker blocked gt-abc123 "Waiting for API key from operations"
  gt worker blocked gt-abc123 "Test suite timing out, investigating"`,
	Args: cobra.ExactArgs(2),
	RunE: runWorkerBlocked,
}

var workerCompletedCmd = &cobra.Command{
	Use:   "completed <issue-id>",
	Short: "Report work completed",
	Long: `Report that work on an issue is completed.

This notifies the refinery that the worker has finished the task.
The work may still need review before being considered fully done.

Examples:
  gt worker completed gt-abc123
  gt worker completed gt-abc123 -m "All tests passing, code review ready"`,
	Args: cobra.ExactArgs(1),
	RunE: runWorkerCompleted,
}

var workerFailedCmd = &cobra.Command{
	Use:   "failed <issue-id> <reason>",
	Short: "Report work failed",
	Long: `Report that work has failed and cannot be completed.

The reason explains why the work could not be completed.
The refinery will escalate this to witness/mayor for reassignment.

Examples:
  gt worker failed gt-abc123 "Architecture incompatible with platform"
  gt worker failed gt-abc123 "Cannot reproduce the reported issue"`,
	Args: cobra.ExactArgs(2),
	RunE: runWorkerFailed,
}

func init() {
	// Progress flag (percentage)
	workerProgressCmd.Flags().StringVarP(&workerStatusMessage, "message", "m", "", "Optional status message")

	// Other commands: message flag
	workerStartedCmd.Flags().StringVarP(&workerStatusMessage, "message", "m", "", "Optional status message")
	workerCompletedCmd.Flags().StringVarP(&workerStatusMessage, "message", "m", "", "Optional status message")

	// Add subcommands to worker command
	workerCmd.AddCommand(workerStartedCmd)
	workerCmd.AddCommand(workerProgressCmd)
	workerCmd.AddCommand(workerBlockedCmd)
	workerCmd.AddCommand(workerCompletedCmd)
	workerCmd.AddCommand(workerFailedCmd)

	rootCmd.AddCommand(workerCmd)
}

// sendWorkerStatus sends a worker status report to the refinery.
func sendWorkerStatus(issueID, status string, report *WorkerStatusReport) error {
	// Find mail work directory
	workDir, err := findMailWorkDir()
	if err != nil {
		return fmt.Errorf("not in a Gas Town workspace: %w", err)
	}

	// Detect sender (worker identity)
	from := detectSender()

	// Determine rig from context
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}
	townRoot, err := workspace.FindFromCwd()
	if err != nil || townRoot == "" {
		return fmt.Errorf("not in a Gas Town workspace")
	}
	roleInfo, err := GetRoleWithContext(cwd, townRoot)
	if err != nil {
		return fmt.Errorf("detecting role: %w", err)
	}

	// Build recipient address
	to := fmt.Sprintf("%s/refinery", roleInfo.Rig)

	// Encode the status report as JSON in the message body
	reportJSON, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding status report: %w", err)
	}

	// Create message
	msg := &mail.Message{
		From:      from,
		To:        to,
		Subject:   fmt.Sprintf("[%s] %s: %s", strings.ToUpper(status), issueID, status),
		Body:      string(reportJSON),
		Timestamp: time.Now(),
		Type:      mail.TypeNotification,
		Priority:  mail.PriorityNormal,
	}

	// Generate thread ID
	msg.ThreadID = generateThreadID()

	// Send via router
	router := mail.NewRouter(workDir)
	if err := router.Send(msg); err != nil {
		return fmt.Errorf("sending message: %w", err)
	}

	fmt.Printf("%s Status reported to %s\n", style.Bold.Render("✓"), to)
	fmt.Printf("  Issue: %s\n", issueID)
	fmt.Printf("  Status: %s\n", status)
	if report.Progress > 0 {
		fmt.Printf("  Progress: %d%%\n", report.Progress)
	}
	if report.Reason != "" {
		fmt.Printf("  Reason: %s\n", report.Reason)
	}
	if report.Message != "" {
		fmt.Printf("  Message: %s\n", report.Message)
	}

	return nil
}

func runWorkerStarted(cmd *cobra.Command, args []string) error {
	issueID := args[0]

	report := &WorkerStatusReport{
		IssueID:    issueID,
		Status:     "started",
		Message:    workerStatusMessage,
		ReportedAt: time.Now(),
	}

	return sendWorkerStatus(issueID, "started", report)
}

func runWorkerProgress(cmd *cobra.Command, args []string) error {
	issueID := args[0]
	pctStr := args[1]

	// Parse percentage (0-100)
	var pct int
	_, err := fmt.Sscanf(pctStr, "%d", &pct)
	if err != nil {
		return fmt.Errorf("invalid progress percentage (must be 0-100): %s", pctStr)
	}
	if pct < 0 || pct > 100 {
		return fmt.Errorf("progress must be between 0 and 100, got %d", pct)
	}

	report := &WorkerStatusReport{
		IssueID:    issueID,
		Status:     "progress",
		Progress:   pct,
		Message:    workerStatusMessage,
		ReportedAt: time.Now(),
	}

	return sendWorkerStatus(issueID, "progress", report)
}

func runWorkerBlocked(cmd *cobra.Command, args []string) error {
	issueID := args[0]
	reason := args[1]

	report := &WorkerStatusReport{
		IssueID:    issueID,
		Status:     "blocked",
		Reason:     reason,
		Message:    workerStatusMessage,
		ReportedAt: time.Now(),
	}

	return sendWorkerStatus(issueID, "blocked", report)
}

func runWorkerCompleted(cmd *cobra.Command, args []string) error {
	issueID := args[0]

	report := &WorkerStatusReport{
		IssueID:    issueID,
		Status:     "completed",
		Message:    workerStatusMessage,
		ReportedAt: time.Now(),
	}

	return sendWorkerStatus(issueID, "completed", report)
}

func runWorkerFailed(cmd *cobra.Command, args []string) error {
	issueID := args[0]
	reason := args[1]

	report := &WorkerStatusReport{
		IssueID:    issueID,
		Status:     "failed",
		Reason:     reason,
		Message:    workerStatusMessage,
		ReportedAt: time.Now(),
	}

	return sendWorkerStatus(issueID, "failed", report)
}
