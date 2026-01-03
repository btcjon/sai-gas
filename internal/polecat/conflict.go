// Package polecat provides polecat lifecycle management.
package polecat

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/steveyegge/gastown/internal/beads"
	"github.com/steveyegge/gastown/internal/git"
	"github.com/steveyegge/gastown/internal/mrqueue"
)

// ConflictTaskPrefix is the title prefix that indicates a conflict resolution task.
const ConflictTaskPrefix = "Resolve merge conflicts:"

// ConflictResolutionTask represents metadata for a conflict resolution task.
type ConflictResolutionTask struct {
	// OriginalMRID is the merge request ID that had conflicts.
	OriginalMRID string

	// Branch is the source branch that needs rebasing.
	Branch string

	// TargetBranch is the target branch (usually "main").
	TargetBranch string

	// ConflictSHA is the SHA of the target when conflict was detected.
	ConflictSHA string

	// SourceIssue is the original work issue being merged.
	SourceIssue string

	// RetryCount tracks how many times this has been retried.
	RetryCount int
}

// IsConflictResolutionTask checks if a bead is a conflict resolution task.
// Conflict resolution tasks are identified by:
// 1. Title starting with "Resolve merge conflicts:"
// 2. Presence of conflict metadata in description
func IsConflictResolutionTask(issue *beads.Issue) bool {
	if issue == nil {
		return false
	}
	return strings.HasPrefix(issue.Title, ConflictTaskPrefix)
}

// ParseConflictTask extracts conflict resolution metadata from a bead.
// Returns nil if the issue is not a conflict resolution task.
func ParseConflictTask(issue *beads.Issue) *ConflictResolutionTask {
	if !IsConflictResolutionTask(issue) {
		return nil
	}

	task := &ConflictResolutionTask{}

	// Parse metadata from description
	// Expected format in description:
	// ## Metadata
	// - Original MR: mr-1234-abcd
	// - Branch: polecat/Toast-abc123
	// - Conflict with: main@abc12345
	// - Original issue: gt-xyz
	// - Retry count: 1

	lines := strings.Split(issue.Description, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "- Original MR:") {
			task.OriginalMRID = strings.TrimSpace(strings.TrimPrefix(line, "- Original MR:"))
		} else if strings.HasPrefix(line, "- Branch:") {
			task.Branch = strings.TrimSpace(strings.TrimPrefix(line, "- Branch:"))
		} else if strings.HasPrefix(line, "- Conflict with:") {
			// Format: "main@abc12345"
			conflict := strings.TrimSpace(strings.TrimPrefix(line, "- Conflict with:"))
			parts := strings.SplitN(conflict, "@", 2)
			if len(parts) == 2 {
				task.TargetBranch = parts[0]
				task.ConflictSHA = parts[1]
			} else {
				task.TargetBranch = conflict
			}
		} else if strings.HasPrefix(line, "- Original issue:") {
			task.SourceIssue = strings.TrimSpace(strings.TrimPrefix(line, "- Original issue:"))
		} else if strings.HasPrefix(line, "- Retry count:") {
			retryStr := strings.TrimSpace(strings.TrimPrefix(line, "- Retry count:"))
			fmt.Sscanf(retryStr, "%d", &task.RetryCount)
		}
	}

	return task
}

// ConflictResolver handles the git operations for resolving merge conflicts.
type ConflictResolver struct {
	git          *git.Git
	mrQueue      *mrqueue.Queue
	polecatPath  string
	rigName      string
	polecatName  string
}

// NewConflictResolver creates a new conflict resolver for the given polecat.
func NewConflictResolver(polecatPath, rigName, polecatName string) *ConflictResolver {
	return &ConflictResolver{
		git:         git.NewGit(polecatPath),
		polecatPath: polecatPath,
		rigName:     rigName,
		polecatName: polecatName,
	}
}

// SetMRQueue sets the MR queue for signaling re-queue after resolution.
func (r *ConflictResolver) SetMRQueue(q *mrqueue.Queue) {
	r.mrQueue = q
}

// RebaseResult contains the result of a rebase operation.
type RebaseResult struct {
	// Success indicates if the rebase completed successfully.
	Success bool

	// ConflictFiles lists files with conflicts (if Success is false).
	ConflictFiles []string

	// Error contains any error message.
	Error string

	// NewHeadSHA is the HEAD SHA after successful rebase.
	NewHeadSHA string
}

// FetchAndCheckout fetches the branch from origin and checks it out.
// This sets up the working directory for the rebase operation.
func (r *ConflictResolver) FetchAndCheckout(branch string) error {
	// Fetch the branch from origin
	if err := r.git.FetchBranch("origin", branch); err != nil {
		return fmt.Errorf("fetching branch %s: %w", branch, err)
	}

	// Create a local branch tracking origin/<branch>
	// First check if we need to create or reset the local branch
	localBranch := extractBranchName(branch)

	// Try to create or reset the local branch to origin/<branch>
	remoteBranch := "origin/" + branch
	exists, _ := r.git.BranchExists(localBranch)
	if exists {
		// Reset existing branch to origin
		if err := r.git.ResetBranch(localBranch, remoteBranch); err != nil {
			return fmt.Errorf("resetting branch %s to %s: %w", localBranch, remoteBranch, err)
		}
	} else {
		// Create new branch from origin
		if err := r.git.CreateBranchFrom(localBranch, remoteBranch); err != nil {
			return fmt.Errorf("creating branch %s from %s: %w", localBranch, remoteBranch, err)
		}
	}

	// Checkout the local branch
	if err := r.git.Checkout(localBranch); err != nil {
		return fmt.Errorf("checking out %s: %w", localBranch, err)
	}

	return nil
}

// Rebase attempts to rebase the current branch onto the target.
// Returns a RebaseResult indicating success or listing conflict files.
func (r *ConflictResolver) Rebase(target string) RebaseResult {
	// Make sure we have the latest target
	if err := r.git.Fetch("origin"); err != nil {
		return RebaseResult{
			Success: false,
			Error:   fmt.Sprintf("fetching origin: %v", err),
		}
	}

	// Attempt the rebase onto origin/<target>
	targetRef := "origin/" + target
	err := r.git.Rebase(targetRef)
	if err != nil {
		// Check if there are conflicts
		conflicts, _ := r.getRebaseConflicts()
		if len(conflicts) > 0 {
			return RebaseResult{
				Success:       false,
				ConflictFiles: conflicts,
				Error:         fmt.Sprintf("rebase conflicts in: %v", conflicts),
			}
		}

		// Some other rebase error
		return RebaseResult{
			Success: false,
			Error:   fmt.Sprintf("rebase failed: %v", err),
		}
	}

	// Rebase succeeded - get the new HEAD SHA
	headSHA, err := r.git.Rev("HEAD")
	if err != nil {
		headSHA = "unknown"
	}

	return RebaseResult{
		Success:    true,
		NewHeadSHA: headSHA,
	}
}

// getRebaseConflicts returns files with conflicts during rebase.
func (r *ConflictResolver) getRebaseConflicts() ([]string, error) {
	status, err := r.git.Status()
	if err != nil {
		return nil, err
	}

	// In a rebase conflict, files show up as modified
	// We also check for unmerged files
	var conflicts []string
	conflicts = append(conflicts, status.Modified...)

	return conflicts, nil
}

// AbortRebase aborts an in-progress rebase.
func (r *ConflictResolver) AbortRebase() error {
	return r.git.AbortRebase()
}

// ForcePushResult contains the result of a force push operation.
type ForcePushResult struct {
	Success bool
	Error   string
	PushedSHA string
}

// ForcePush force-pushes the current branch to origin.
// Includes safety checks:
// 1. Branch must be a polecat branch (starts with "polecat/")
// 2. Must not be main or master
// 3. Must have commits to push
func (r *ConflictResolver) ForcePush(branch string) ForcePushResult {
	// Safety check 1: Must be a polecat branch or similar working branch
	if !isWorkingBranch(branch) {
		return ForcePushResult{
			Success: false,
			Error:   fmt.Sprintf("refusing to force-push: %s is not a working branch", branch),
		}
	}

	// Safety check 2: Must not be main/master
	if isProtectedBranch(branch) {
		return ForcePushResult{
			Success: false,
			Error:   fmt.Sprintf("refusing to force-push to protected branch: %s", branch),
		}
	}

	// Get current HEAD SHA for tracking
	headSHA, err := r.git.Rev("HEAD")
	if err != nil {
		return ForcePushResult{
			Success: false,
			Error:   fmt.Sprintf("failed to get HEAD SHA: %v", err),
		}
	}

	// Force push to origin
	if err := r.git.Push("origin", branch, true); err != nil {
		return ForcePushResult{
			Success: false,
			Error:   fmt.Sprintf("force-push failed: %v", err),
		}
	}

	return ForcePushResult{
		Success:   true,
		PushedSHA: headSHA,
	}
}

// SignalMRRequeue updates the MR metadata to indicate it should be retried.
// This clears the conflict task ID and updates retry metadata.
func (r *ConflictResolver) SignalMRRequeue(mrID, resolvedSHA string) error {
	if r.mrQueue == nil {
		return fmt.Errorf("MR queue not configured")
	}

	// Get the MR from queue
	mr, err := r.mrQueue.Get(mrID)
	if err != nil {
		return fmt.Errorf("getting MR %s: %w", mrID, err)
	}

	// Clear conflict task ID (work is done)
	mr.ConflictTaskID = ""

	// Update the MR in queue
	// The refinery will automatically pick it up on its next poll
	if err := r.mrQueue.Update(mr); err != nil {
		return fmt.Errorf("updating MR %s: %w", mrID, err)
	}

	return nil
}

// extractBranchName extracts the simple branch name from a ref.
// "origin/polecat/Toast" -> "polecat/Toast"
// "polecat/Toast" -> "polecat/Toast"
func extractBranchName(ref string) string {
	if strings.HasPrefix(ref, "origin/") {
		return strings.TrimPrefix(ref, "origin/")
	}
	return ref
}

// isWorkingBranch checks if a branch is a working branch that can be force-pushed.
// Working branches include:
// - polecat/* branches
// - crew/* branches
// - feature/* branches
// - fix/* branches
var workingBranchPatterns = []string{
	`^polecat/`,
	`^crew/`,
	`^feature/`,
	`^fix/`,
	`^temp-resolve`,
}

func isWorkingBranch(branch string) bool {
	for _, pattern := range workingBranchPatterns {
		if matched, _ := regexp.MatchString(pattern, branch); matched {
			return true
		}
	}
	return false
}

// isProtectedBranch checks if a branch is protected (should never be force-pushed).
var protectedBranches = []string{"main", "master", "develop", "release"}

func isProtectedBranch(branch string) bool {
	branch = extractBranchName(branch)
	for _, protected := range protectedBranches {
		if branch == protected {
			return true
		}
	}
	return false
}
