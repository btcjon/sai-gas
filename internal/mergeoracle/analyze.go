// Package mergeoracle provides analysis of changesets before refinery processing.
package mergeoracle

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/steveyegge/gastown/internal/git"
	"github.com/steveyegge/gastown/internal/mrqueue"
)

// OverlapGraph represents file overlaps between MRs.
type OverlapGraph struct {
	Nodes map[string]*MRNode // mr.ID -> MRNode
	Edges map[string][]string // mr.ID -> list of MR IDs that overlap
}

// MRNode represents a single MR in the overlap graph.
type MRNode struct {
	MR        *mrqueue.MR
	Files     map[string]bool // Changed files in this MR
	FileCount int             // Number of files changed
	RiskScore float64         // Risk assessment score
}

// DisjointClassification categorizes MRs by merge safety.
type DisjointClassification struct {
	ParallelSafe   []*mrqueue.MR // MRs that can merge in any order
	NeedsSequencing []*MRPair    // MR pairs that have conflicts
	HighRisk       []*mrqueue.MR // MRs with high-risk patterns
}

// MRPair represents two MRs with file overlaps.
type MRPair struct {
	MR1       *mrqueue.MR
	MR2       *mrqueue.MR
	OverlapCount int
	SharedFiles  []string
}

// RiskAssessment analyzes a single MR for risk patterns.
type RiskAssessment struct {
	MR             *mrqueue.MR
	FileCount      int
	CoreFileChanges int  // Changes to core/critical files
	LargeDiffLines  bool // Large number of changed lines
	DBMigrations    bool // Contains database migrations
	APIChanges      bool // Changes to public APIs
	RiskScore      float64
	Recommendations []string
}

// Analyzer performs changeset analysis.
type Analyzer struct {
	rigPath string
	gitDir  *git.Git
	queue   *mrqueue.Queue
}

// New creates a new analyzer for a rig.
func New(rigPath string) (*Analyzer, error) {
	g := git.NewGit(rigPath)

	// Verify it's a git repo
	if _, err := g.CurrentBranch(); err != nil {
		return nil, fmt.Errorf("not a valid git repository: %w", err)
	}

	q := mrqueue.New(rigPath)

	return &Analyzer{
		rigPath: rigPath,
		gitDir:  g,
		queue:   q,
	}, nil
}

// AnalyzeMRQueue performs a full analysis of the MR queue.
func (a *Analyzer) AnalyzeMRQueue() (*AnalysisResult, error) {
	// Get all MRs in queue
	mrs, err := a.queue.ListByScore()
	if err != nil {
		return nil, fmt.Errorf("listing MR queue: %w", err)
	}

	if len(mrs) == 0 {
		return &AnalysisResult{
			QueueSize:             0,
			ParallelSafe:          []*mrqueue.MR{},
			NeedsSequencing:       []*MRPair{},
			HighRisk:              []*mrqueue.MR{},
			RiskAssessments:       map[string]*RiskAssessment{},
			RecommendedMergeOrder: []*mrqueue.MR{},
		}, nil
	}

	// Build overlap graph
	graph, err := a.buildOverlapGraph(mrs)
	if err != nil {
		return nil, fmt.Errorf("building overlap graph: %w", err)
	}

	// Classify MRs
	classification := a.classifyByDisjointness(graph)

	// Perform risk assessment on each MR
	riskAssessments := make(map[string]*RiskAssessment)
	for _, mr := range mrs {
		risk, err := a.assessMRRisk(mr)
		if err != nil {
			// Log error but continue
			fmt.Fprintf(os.Stderr, "warning: failed to assess risk for %s: %v\n", mr.ID, err)
			continue
		}
		riskAssessments[mr.ID] = risk
	}

	// Identify high-risk MRs
	highRisk := a.filterHighRisk(classification.HighRisk, riskAssessments)

	// Generate merge recommendations
	recommended := a.recommendMergeOrder(mrs, graph, riskAssessments)

	return &AnalysisResult{
		QueueSize:             len(mrs),
		ParallelSafe:          classification.ParallelSafe,
		NeedsSequencing:       classification.NeedsSequencing,
		HighRisk:              highRisk,
		RiskAssessments:       riskAssessments,
		RecommendedMergeOrder: recommended,
	}, nil
}

// buildOverlapGraph constructs a file-overlap graph of all MRs.
func (a *Analyzer) buildOverlapGraph(mrs []*mrqueue.MR) (*OverlapGraph, error) {
	graph := &OverlapGraph{
		Nodes: make(map[string]*MRNode),
		Edges: make(map[string][]string),
	}

	// Build nodes - get changed files for each MR
	for _, mr := range mrs {
		files, err := a.getChangedFiles(mr)
		if err != nil {
			return nil, fmt.Errorf("getting changed files for %s: %w", mr.ID, err)
		}

		fileMap := make(map[string]bool)
		for _, f := range files {
			fileMap[f] = true
		}

		node := &MRNode{
			MR:        mr,
			Files:     fileMap,
			FileCount: len(files),
		}

		graph.Nodes[mr.ID] = node
	}

	// Build edges - find overlapping files
	mrIDs := make([]string, 0, len(graph.Nodes))
	for id := range graph.Nodes {
		mrIDs = append(mrIDs, id)
	}

	for i := 0; i < len(mrIDs); i++ {
		for j := i + 1; j < len(mrIDs); j++ {
			id1, id2 := mrIDs[i], mrIDs[j]
			node1, node2 := graph.Nodes[id1], graph.Nodes[id2]

			// Count overlaps
			overlapCount := 0
			for file := range node1.Files {
				if node2.Files[file] {
					overlapCount++
				}
			}

			if overlapCount > 0 {
				graph.Edges[id1] = append(graph.Edges[id1], id2)
				graph.Edges[id2] = append(graph.Edges[id2], id1)
			}
		}
	}

	return graph, nil
}

// getChangedFiles returns the list of files changed in an MR.
func (a *Analyzer) getChangedFiles(mr *mrqueue.MR) ([]string, error) {
	// Get diff between base and source branch
	files, err := a.gitDir.DiffFiles(mr.Target, mr.Branch)
	if err != nil {
		// If we can't get the diff (e.g., branch doesn't exist yet), return empty
		return []string{}, nil
	}
	return files, nil
}

// classifyByDisjointness categorizes MRs as parallel-safe or needing sequencing.
func (a *Analyzer) classifyByDisjointness(graph *OverlapGraph) *DisjointClassification {
	classification := &DisjointClassification{
		ParallelSafe:    []*mrqueue.MR{},
		NeedsSequencing: []*MRPair{},
	}

	// MRs with no edges are parallel-safe
	for id, node := range graph.Nodes {
		if len(graph.Edges[id]) == 0 {
			classification.ParallelSafe = append(classification.ParallelSafe, node.MR)
		}
	}

	// Pairs with edges need sequencing
	seenPairs := make(map[string]bool)
	for id1, edges := range graph.Edges {
		for _, id2 := range edges {
			// Create canonical pair key to avoid duplicates
			pair := []string{id1, id2}
			sort.Strings(pair)
			pairKey := strings.Join(pair, "|")

			if !seenPairs[pairKey] {
				seenPairs[pairKey] = true

				node1 := graph.Nodes[id1]
				node2 := graph.Nodes[id2]

				// Find shared files
				shared := []string{}
				for file := range node1.Files {
					if node2.Files[file] {
						shared = append(shared, file)
					}
				}
				sort.Strings(shared)

				classification.NeedsSequencing = append(classification.NeedsSequencing, &MRPair{
					MR1:           node1.MR,
					MR2:           node2.MR,
					OverlapCount:  len(shared),
					SharedFiles:   shared,
				})
			}
		}
	}

	return classification
}

// assessMRRisk performs risk assessment on a single MR.
func (a *Analyzer) assessMRRisk(mr *mrqueue.MR) (*RiskAssessment, error) {
	assessment := &RiskAssessment{
		MR:              mr,
		Recommendations: []string{},
	}

	// Get changed files
	files, err := a.getChangedFiles(mr)
	if err != nil {
		return nil, err
	}

	assessment.FileCount = len(files)

	// Check for core file changes
	corePatterns := []string{"internal/", "cmd/", "main.go", "go.mod"}
	for _, file := range files {
		for _, pattern := range corePatterns {
			if strings.Contains(file, pattern) {
				assessment.CoreFileChanges++
				break
			}
		}

		// Check for database migrations
		if strings.Contains(file, "migration") || strings.Contains(file, "migrate") {
			assessment.DBMigrations = true
		}

		// Check for API changes
		if strings.Contains(file, "api") || strings.Contains(file, "handler") || strings.Contains(file, "endpoint") {
			assessment.APIChanges = true
		}
	}

	// Calculate risk score
	assessment.RiskScore = 0.0

	// File count risk (more files = higher risk)
	if assessment.FileCount > 20 {
		assessment.RiskScore += 3.0
		assessment.LargeDiffLines = true
	} else if assessment.FileCount > 10 {
		assessment.RiskScore += 2.0
	} else if assessment.FileCount > 5 {
		assessment.RiskScore += 1.0
	}

	// Core file changes risk
	if assessment.CoreFileChanges > 5 {
		assessment.RiskScore += 3.0
		assessment.Recommendations = append(assessment.Recommendations,
			"High number of core file changes - requires careful review before merge")
	} else if assessment.CoreFileChanges > 0 {
		assessment.RiskScore += 2.0
		assessment.Recommendations = append(assessment.Recommendations,
			"Core files modified - ensure changes are backward compatible")
	}

	// Database migration risk
	if assessment.DBMigrations {
		assessment.RiskScore += 4.0
		assessment.Recommendations = append(assessment.Recommendations,
			"Contains database migrations - must merge before dependent code")
	}

	// API changes risk
	if assessment.APIChanges {
		assessment.RiskScore += 2.0
		assessment.Recommendations = append(assessment.Recommendations,
			"Public API changes - verify backward compatibility")
	}

	return assessment, nil
}

// filterHighRisk returns MRs with risk scores above threshold.
func (a *Analyzer) filterHighRisk(baseHighRisk []*mrqueue.MR, assessments map[string]*RiskAssessment) []*mrqueue.MR {
	highRiskMap := make(map[string]bool)
	for _, mr := range baseHighRisk {
		highRiskMap[mr.ID] = true
	}

	// Add any MRs with risk score > 3.0
	for id, assessment := range assessments {
		if assessment.RiskScore > 3.0 {
			highRiskMap[id] = true
		}
	}

	// Find MR objects for high-risk IDs
	result := make([]*mrqueue.MR, 0)
	for id := range highRiskMap {
		for _, assessment := range assessments {
			if assessment.MR.ID == id {
				result = append(result, assessment.MR)
				break
			}
		}
	}

	return result
}

// recommendMergeOrder suggests merge order based on analysis.
func (a *Analyzer) recommendMergeOrder(mrs []*mrqueue.MR, graph *OverlapGraph, assessments map[string]*RiskAssessment) []*mrqueue.MR {
	// Strategy: merge high-risk items first (especially DB migrations), then parallel-safe, then others

	recommended := make([]*mrqueue.MR, 0)
	merged := make(map[string]bool)

	// First: DB migrations (must come first)
	for _, mr := range mrs {
		if merged[mr.ID] {
			continue
		}
		if assessment, ok := assessments[mr.ID]; ok && assessment.DBMigrations {
			recommended = append(recommended, mr)
			merged[mr.ID] = true
		}
	}

	// Second: High-risk items (review before merge but after migrations)
	for _, mr := range mrs {
		if merged[mr.ID] {
			continue
		}
		if assessment, ok := assessments[mr.ID]; ok && assessment.RiskScore > 3.0 {
			recommended = append(recommended, mr)
			merged[mr.ID] = true
		}
	}

	// Third: Parallel-safe items
	for _, mr := range mrs {
		if merged[mr.ID] {
			continue
		}
		if len(graph.Edges[mr.ID]) == 0 {
			recommended = append(recommended, mr)
			merged[mr.ID] = true
		}
	}

	// Fourth: Everything else, using existing queue order
	for _, mr := range mrs {
		if !merged[mr.ID] {
			recommended = append(recommended, mr)
		}
	}

	return recommended
}

// AnalysisResult contains the full analysis output.
type AnalysisResult struct {
	QueueSize             int
	ParallelSafe          []*mrqueue.MR
	NeedsSequencing       []*MRPair
	HighRisk              []*mrqueue.MR
	RiskAssessments       map[string]*RiskAssessment
	RecommendedMergeOrder []*mrqueue.MR
}
