package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/mergeoracle"
	"github.com/steveyegge/gastown/internal/style"
)

var (
	mergeOracleJSON bool
)

var mergeOracleCmd = &cobra.Command{
	Use:     "merge-oracle",
	Aliases: []string{"oracle"},
	GroupID: GroupServices,
	Short:   "Analyze MR queue before merge processing",
	Long: `Analyze changesets in the merge queue to identify conflicts and risks.

The merge-oracle plugin provides:
  - Overlap graph of file changes between MRs
  - Classification of parallel-safe vs conflict-prone MRs
  - Risk assessment for high-impact changes
  - Merge order recommendations

Examples:
  gt merge-oracle analyze greenplace
  gt merge-oracle recommend greenplace`,
	RunE: requireSubcommand,
}

var mergeOracleAnalyzeCmd = &cobra.Command{
	Use:   "analyze <rig>",
	Short: "Analyze MR queue for conflicts and risks",
	Long: `Analyze the merge queue for file overlaps, conflicts, and risks.

Output includes:
  - Summary of queue size
  - MRs that can merge in parallel (no file overlaps)
  - MRs with file conflicts that need sequencing
  - High-risk changes (large diffs, core files, migrations)
  - Risk scores and recommendations

Examples:
  gt merge-oracle analyze greenplace
  gt merge-oracle analyze greenplace --json`,
	Args: cobra.ExactArgs(1),
	RunE: runMergeOracleAnalyze,
}

var mergeOracleRecommendCmd = &cobra.Command{
	Use:   "recommend <rig>",
	Short: "Get merge order recommendations",
	Long: `Analyze the queue and recommend a merge order based on:
  - Database migrations (must come first)
  - High-risk changes (reviewed carefully)
  - Parallel-safe MRs (can merge in any order)
  - Remaining MRs (in priority order)

Examples:
  gt merge-oracle recommend greenplace
  gt merge-oracle recommend greenplace --json`,
	Args: cobra.ExactArgs(1),
	RunE: runMergeOracleRecommend,
}

func init() {
	mergeOracleAnalyzeCmd.Flags().BoolVar(&mergeOracleJSON, "json", false, "Output analysis in JSON format")
	mergeOracleRecommendCmd.Flags().BoolVar(&mergeOracleJSON, "json", false, "Output recommendations in JSON format")

	mergeOracleCmd.AddCommand(mergeOracleAnalyzeCmd)
	mergeOracleCmd.AddCommand(mergeOracleRecommendCmd)
	rootCmd.AddCommand(mergeOracleCmd)
}

func runMergeOracleAnalyze(cmd *cobra.Command, args []string) error {
	rigName := args[0]

	// Get rig
	_, r, err := getRig(rigName)
	if err != nil {
		return err
	}

	// Create analyzer
	analyzer, err := mergeoracle.New(r.Path)
	if err != nil {
		return fmt.Errorf("creating analyzer: %w", err)
	}

	// Run analysis
	result, err := analyzer.AnalyzeMRQueue()
	if err != nil {
		return fmt.Errorf("analyzing queue: %w", err)
	}

	// Format output
	if mergeOracleJSON {
		// Convert to JSON-serializable format
		jsonResult := map[string]interface{}{
			"queue_size":       result.QueueSize,
			"parallel_safe":    []interface{}{},
			"needs_sequencing": []map[string]interface{}{},
			"high_risk":        []map[string]interface{}{},
			"recommendations":  []string{},
		}

		// Build parallel safe list
		if len(result.ParallelSafe) > 0 {
			jsonResult["parallel_safe"] = result.ParallelSafe
		}

		// Build needs sequencing list
		if len(result.NeedsSequencing) > 0 {
			seqList := make([]map[string]interface{}, len(result.NeedsSequencing))
			for i, pair := range result.NeedsSequencing {
				seqList[i] = map[string]interface{}{
					"mr1":           pair.MR1.ID,
					"mr2":           pair.MR2.ID,
					"overlap_count": pair.OverlapCount,
					"shared_files":  pair.SharedFiles,
				}
			}
			jsonResult["needs_sequencing"] = seqList
		}

		// Build high risk list
		if len(result.HighRisk) > 0 {
			riskList := make([]map[string]interface{}, len(result.HighRisk))
			for i, mr := range result.HighRisk {
				assessment := result.RiskAssessments[mr.ID]
				riskList[i] = map[string]interface{}{
					"mr_id":           mr.ID,
					"title":           mr.Title,
					"files_changed":   assessment.FileCount,
					"risk_score":      assessment.RiskScore,
					"recommendations": assessment.Recommendations,
				}
			}
			jsonResult["high_risk"] = riskList
		}

		data, err := json.MarshalIndent(jsonResult, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling JSON: %w", err)
		}
		fmt.Println(string(data))
	} else {
		// Human-readable output
		fmt.Printf("%s MR Queue Analysis for %s %s\n\n", "📊", style.Bold.Render(rigName), style.Dim.Render(fmt.Sprintf("[%d mr(s)]", result.QueueSize)))

		if result.QueueSize == 0 {
			fmt.Println(style.Dim.Render("  Queue is empty"))
			return nil
		}

		// Parallel-safe section
		if len(result.ParallelSafe) > 0 {
			fmt.Println(style.Bold.Render("✓ Parallel-safe (no overlap):"))
			for _, mr := range result.ParallelSafe {
				fmt.Printf("  - %s: %s (%s)\n", style.Bold.Render(mr.ID), mr.Title, style.Dim.Render(mr.Branch))
			}
			fmt.Println()
		}

		// Needs sequencing section
		if len(result.NeedsSequencing) > 0 {
			fmt.Println(style.Bold.Render("⚠ Needs sequencing (file overlap):"))
			for _, pair := range result.NeedsSequencing {
				fmt.Printf("  - %s ↔ %s (%d files overlap)\n",
					style.Bold.Render(pair.MR1.ID), style.Bold.Render(pair.MR2.ID), pair.OverlapCount)
				if len(pair.SharedFiles) > 0 && len(pair.SharedFiles) <= 3 {
					for _, f := range pair.SharedFiles {
						fmt.Printf("      • %s\n", style.Dim.Render(f))
					}
				} else if len(pair.SharedFiles) > 3 {
					fmt.Printf("      • %s and %d more files\n", style.Dim.Render(pair.SharedFiles[0]), len(pair.SharedFiles)-1)
				}
			}
			fmt.Println()
		}

		// High-risk section
		if len(result.HighRisk) > 0 {
			fmt.Println(style.Bold.Render("🚨 High-risk:"))
			for _, mr := range result.HighRisk {
				assessment := result.RiskAssessments[mr.ID]
				riskLabel := "HIGH"
				if assessment.RiskScore > 5.0 {
					riskLabel = "CRITICAL"
				}
				fmt.Printf("  - %s: %s [%s] (%d files)\n",
					style.Bold.Render(mr.ID), mr.Title, riskLabel, assessment.FileCount)

				if len(assessment.Recommendations) > 0 {
					for _, rec := range assessment.Recommendations {
						fmt.Printf("      → %s\n", style.Dim.Render(rec))
					}
				}
			}
			fmt.Println()
		}

		// Summary
		fmt.Println(style.Dim.Render(fmt.Sprintf("Summary: %d parallel-safe, %d conflicts, %d high-risk",
			len(result.ParallelSafe), len(result.NeedsSequencing), len(result.HighRisk))))
	}

	return nil
}

func runMergeOracleRecommend(cmd *cobra.Command, args []string) error {
	rigName := args[0]

	// Get rig
	_, r, err := getRig(rigName)
	if err != nil {
		return err
	}

	// Create analyzer
	analyzer, err := mergeoracle.New(r.Path)
	if err != nil {
		return fmt.Errorf("creating analyzer: %w", err)
	}

	// Run analysis
	result, err := analyzer.AnalyzeMRQueue()
	if err != nil {
		return fmt.Errorf("analyzing queue: %w", err)
	}

	if mergeOracleJSON {
		// JSON output
		jsonResult := map[string]interface{}{
			"recommended_order": []map[string]interface{}{},
			"rationale":         []string{},
		}

		for i, mr := range result.RecommendedMergeOrder {
			assessment := result.RiskAssessments[mr.ID]
			jsonResult["recommended_order"] = append(
				jsonResult["recommended_order"].([]map[string]interface{}),
				map[string]interface{}{
					"order":      i + 1,
					"mr_id":      mr.ID,
					"title":      mr.Title,
					"priority":   mr.Priority,
					"risk_score": assessment.RiskScore,
				},
			)
		}

		data, err := json.MarshalIndent(jsonResult, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling JSON: %w", err)
		}
		fmt.Println(string(data))
	} else {
		// Human-readable output
		fmt.Printf("%s Merge Order Recommendations for %s\n\n", "📋", style.Bold.Render(rigName))

		if result.QueueSize == 0 {
			fmt.Println(style.Dim.Render("  Queue is empty"))
			return nil
		}

		for i, mr := range result.RecommendedMergeOrder {
			assessment := result.RiskAssessments[mr.ID]
			riskBadge := ""
			if assessment.RiskScore > 5.0 {
				riskBadge = " [CRITICAL]"
			} else if assessment.RiskScore > 3.0 {
				riskBadge = " [HIGH-RISK]"
			}

			fmt.Printf("%2d. %s: %s%s\n", i+1, style.Bold.Render(mr.ID), mr.Title, riskBadge)
			fmt.Printf("    Priority: %d | Risk: %.1f | Files: %d\n",
				mr.Priority, assessment.RiskScore, assessment.FileCount)

			if assessment.DBMigrations {
				fmt.Printf("    %s Database migrations - must merge first\n", "⚠")
			}

			if len(assessment.Recommendations) > 0 && len(assessment.Recommendations) <= 2 {
				for _, rec := range assessment.Recommendations {
					fmt.Printf("    → %s\n", rec)
				}
			}
			fmt.Println()
		}
	}

	return nil
}
