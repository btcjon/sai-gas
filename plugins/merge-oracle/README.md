# Merge Oracle Plugin (gt-pio)

## Overview

The Merge Oracle plugin provides intelligent analysis of the merge queue before Refinery processes changes. It identifies conflicts, assesses risks, and recommends optimal merge ordering.

## Features

### 1. Overlap Graph Analysis
- Builds a dependency graph of file changes between all MRs
- Identifies which MRs modify the same files
- Detects parallel-safe vs conflict-prone changesets

### 2. Disjointness Classification
MRs are categorized into three groups:

- **Parallel-safe**: MRs with no file overlaps that can merge in any order
- **Needs sequencing**: MRs with file conflicts that require careful ordering
- **High-risk**: Changes that pose merge or stability risks

### 3. Risk Assessment
Each MR is evaluated for:

- **File count**: Large changesets with many files
- **Core file changes**: Modifications to critical infrastructure
- **Database migrations**: Schema or migration changes
- **API changes**: Public API or interface modifications
- **Risk score**: Composite score combining all factors

### 4. Merge Order Recommendations
Smart recommendations based on:

1. Database migrations (must merge first)
2. High-risk changes (reviewed before merge)
3. Parallel-safe MRs (any order)
4. Remaining MRs (priority order)

## Usage

### Analyze the Queue

```bash
gt merge-oracle analyze <rig>
```

Output shows:
- Queue size and summary
- Parallel-safe MRs (can merge simultaneously)
- MR pairs with file overlaps (need sequencing)
- High-risk changes with recommendations

Example:

```bash
$ gt merge-oracle analyze greenplace

📊 MR Queue Analysis for greenplace [3 mr(s)]

✓ Parallel-safe (no overlap):
  - mr-abc123: Add user auth feature (polecat/auth)
  - mr-def456: Update documentation (docs/update-readme)

⚠ Needs sequencing (file overlap):
  - mr-ghi789 ↔ mr-jkl012 (2 files overlap)
      • internal/api/handler.go
      • go.mod

🚨 High-risk:
  - mr-mno345: Core database refactor [HIGH-RISK] (15 files)
      → Large number of core file changes - requires careful review
      → Core files modified - ensure changes are backward compatible

Summary: 2 parallel-safe, 1 conflict, 1 high-risk
```

### Get Merge Recommendations

```bash
gt merge-oracle recommend <rig>
```

Output shows recommended merge order with:
- Order number
- MR ID and title
- Priority and risk score
- Number of files changed
- Special flags (migrations, etc.)

Example:

```bash
$ gt merge-oracle recommend greenplace

📋 Merge Order Recommendations for greenplace

 1. mr-mno345: Core database refactor [HIGH-RISK]
    Priority: 0 | Risk: 4.2 | Files: 15
    ⚠ Database migrations - must merge first
    → Large number of core file changes - requires careful review

 2. mr-ghi789: Implement new API version
    Priority: 1 | Risk: 3.5 | Files: 8
    → Core files modified - ensure changes are backward compatible

 3. mr-abc123: Add user auth feature
    Priority: 2 | Risk: 1.0 | Files: 4

 4. mr-def456: Update documentation
    Priority: 3 | Risk: 0.0 | Files: 2
```

### JSON Output

Both commands support `--json` for machine-readable output:

```bash
gt merge-oracle analyze greenplace --json
gt merge-oracle recommend greenplace --json
```

## Risk Scoring

Risk scores are calculated from:

| Factor | Weight | Threshold |
|--------|--------|-----------|
| File count (5-10 files) | +1.0 | - |
| File count (10-20 files) | +2.0 | - |
| File count (>20 files) | +3.0 | - |
| Core file changes (1-5) | +2.0 | - |
| Core file changes (>5) | +3.0 | - |
| Database migrations | +4.0 | Critical |
| API changes | +2.0 | High |

Scores above 3.0 are flagged as high-risk.

## Architecture

### Analyzer

The `Analyzer` type performs the analysis:

```go
analyzer, err := mergeoracle.New(rigPath)
result, err := analyzer.AnalyzeMRQueue()
```

### Analysis Result

Returns:
- `QueueSize`: Number of MRs in queue
- `ParallelSafe`: MRs with no conflicts
- `NeedsSequencing`: MR pairs with overlaps
- `HighRisk`: MRs above risk threshold
- `RiskAssessments`: Detailed risk data per MR
- `RecommendedMergeOrder`: Suggested merge sequence

### Graph Structure

The overlap graph identifies:
- **Nodes**: Each MR with its changed files
- **Edges**: Connections between MRs that share files

## Integration with Refinery

The merge-oracle plugin is designed to be called by Refinery before processing the queue:

```go
// In refinery processing loop
analysis, err := analyzer.AnalyzeMRQueue()
if len(analysis.HighRisk) > 0 {
    log.Printf("Warning: %d high-risk MRs in queue", len(analysis.HighRisk))
}
recommendedOrder := analysis.RecommendedMergeOrder
// Process in recommended order
```

## Use Cases

### Pre-merge Safety Check

Before Refinery starts processing, check if there are any critical risks:

```bash
gt merge-oracle analyze rig && echo "Safe to merge"
```

### Resolve Merge Conflicts

When conflicts occur, analyze the queue to see which MRs should have merged in a different order:

```bash
gt merge-oracle analyze rig --json > queue-analysis.json
# Review overlap graph and adjust sequencing
```

### Optimize CI/CD Pipeline

Use recommendations to merge MRs in order that maximizes parallelism while respecting dependencies:

```bash
gt merge-oracle recommend rig | grep "Parallel-safe"
# Merge these in parallel, others sequentially
```

## Implementation Notes

- Diff detection uses `git diff base...head --name-only` (three-dot syntax)
- Risk patterns are heuristic-based and may need tuning per project
- Analysis is read-only - doesn't modify repository state
- Can be run multiple times safely (idempotent)

## Future Enhancements

Potential improvements:

1. **Conflict prediction**: Test-merge branches to predict actual conflicts
2. **Custom risk patterns**: User-defined risk patterns per project
3. **Historical analysis**: Track merge patterns and success rates
4. **Dependency inference**: Extract dependencies from code (imports, etc.)
5. **CI integration**: Hook results into GitHub checks/statuses
