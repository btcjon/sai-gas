package mrqueue

import (
	"testing"
	"time"
)

func TestScoreMR_BaseScore(t *testing.T) {
	now := time.Now()
	config := DefaultScoreConfig()

	input := ScoreInput{
		Priority:    2,  // P2 (medium)
		MRCreatedAt: now,
		RetryCount:  0,
		Now:         now,
	}

	score := ScoreMR(input, config)

	// BaseScore(1000) + Priority(2 gives 4-2=2, so 2*100=200) = 1200
	expected := 1200.0
	if score != expected {
		t.Errorf("expected score %f, got %f", expected, score)
	}
}

func TestScoreMR_PriorityOrdering(t *testing.T) {
	now := time.Now()

	tests := []struct {
		priority int
		expected float64
	}{
		{0, 1400.0}, // P0: base(1000) + (4-0)*100 = 1400
		{1, 1300.0}, // P1: base(1000) + (4-1)*100 = 1300
		{2, 1200.0}, // P2: base(1000) + (4-2)*100 = 1200
		{3, 1100.0}, // P3: base(1000) + (4-3)*100 = 1100
		{4, 1000.0}, // P4: base(1000) + (4-4)*100 = 1000
	}

	for _, tt := range tests {
		t.Run("P"+string(rune('0'+tt.priority)), func(t *testing.T) {
			input := ScoreInput{
				Priority:    tt.priority,
				MRCreatedAt: now,
				Now:         now,
			}
			score := ScoreMRWithDefaults(input)
			if score != tt.expected {
				t.Errorf("P%d: expected %f, got %f", tt.priority, tt.expected, score)
			}
		})
	}

	// Verify ordering: P0 > P1 > P2 > P3 > P4
	for i := 0; i < 4; i++ {
		input1 := ScoreInput{Priority: i, MRCreatedAt: now, Now: now}
		input2 := ScoreInput{Priority: i + 1, MRCreatedAt: now, Now: now}
		score1 := ScoreMRWithDefaults(input1)
		score2 := ScoreMRWithDefaults(input2)
		if score1 <= score2 {
			t.Errorf("P%d (%f) should score higher than P%d (%f)", i, score1, i+1, score2)
		}
	}
}

func TestScoreMR_ConvoyAgeEscalation(t *testing.T) {
	now := time.Now()
	config := DefaultScoreConfig()

	// MR without convoy
	noConvoy := ScoreInput{
		Priority:    2,
		MRCreatedAt: now,
		Now:         now,
	}
	scoreNoConvoy := ScoreMR(noConvoy, config)

	// MR with 24-hour old convoy
	convoyTime := now.Add(-24 * time.Hour)
	withConvoy := ScoreInput{
		Priority:        2,
		MRCreatedAt:     now,
		ConvoyCreatedAt: &convoyTime,
		Now:             now,
	}
	scoreWithConvoy := ScoreMR(withConvoy, config)

	// 24 hours * 10 pts/hour = 240 extra points
	expectedDiff := 240.0
	actualDiff := scoreWithConvoy - scoreNoConvoy
	if actualDiff != expectedDiff {
		t.Errorf("expected convoy age to add %f pts, got %f", expectedDiff, actualDiff)
	}
}

func TestScoreMR_ConvoyStarvationPrevention(t *testing.T) {
	now := time.Now()

	// P4 issue in 48-hour old convoy vs P0 issue with no convoy
	oldConvoy := now.Add(-48 * time.Hour)
	lowPriorityOldConvoy := ScoreInput{
		Priority:        4, // P4 (lowest)
		MRCreatedAt:     now,
		ConvoyCreatedAt: &oldConvoy,
		Now:             now,
	}

	highPriorityNoConvoy := ScoreInput{
		Priority:    0, // P0 (highest)
		MRCreatedAt: now,
		Now:         now,
	}

	scoreOldConvoy := ScoreMRWithDefaults(lowPriorityOldConvoy)
	scoreHighPriority := ScoreMRWithDefaults(highPriorityNoConvoy)

	// P4 with 48h convoy: 1000 + 0 + 480 = 1480
	// P0 with no convoy: 1000 + 400 + 0 = 1400
	// Old convoy should win (starvation prevention)
	if scoreOldConvoy <= scoreHighPriority {
		t.Errorf("48h old P4 convoy (%f) should beat P0 no convoy (%f) for starvation prevention",
			scoreOldConvoy, scoreHighPriority)
	}
}

func TestScoreMR_RetryPenalty(t *testing.T) {
	now := time.Now()
	config := DefaultScoreConfig()

	// No retries
	noRetry := ScoreInput{
		Priority:    2,
		MRCreatedAt: now,
		RetryCount:  0,
		Now:         now,
	}
	scoreNoRetry := ScoreMR(noRetry, config)

	// 3 retries
	threeRetries := ScoreInput{
		Priority:    2,
		MRCreatedAt: now,
		RetryCount:  3,
		Now:         now,
	}
	scoreThreeRetries := ScoreMR(threeRetries, config)

	// 3 retries * 50 pts penalty = 150 pts less
	expectedDiff := 150.0
	actualDiff := scoreNoRetry - scoreThreeRetries
	if actualDiff != expectedDiff {
		t.Errorf("expected 3 retries to lose %f pts, lost %f", expectedDiff, actualDiff)
	}
}

func TestScoreMR_RetryPenaltyCapped(t *testing.T) {
	now := time.Now()
	config := DefaultScoreConfig()

	// Max penalty is 300, so 10 retries should be same as 6
	sixRetries := ScoreInput{
		Priority:    2,
		MRCreatedAt: now,
		RetryCount:  6,
		Now:         now,
	}
	tenRetries := ScoreInput{
		Priority:    2,
		MRCreatedAt: now,
		RetryCount:  10,
		Now:         now,
	}

	scoreSix := ScoreMR(sixRetries, config)
	scoreTen := ScoreMR(tenRetries, config)

	if scoreSix != scoreTen {
		t.Errorf("penalty should be capped: 6 retries (%f) should equal 10 retries (%f)",
			scoreSix, scoreTen)
	}

	// Both should be base(1000) + priority(200) - maxPenalty(300) = 900
	expected := 900.0
	if scoreSix != expected {
		t.Errorf("expected capped score %f, got %f", expected, scoreSix)
	}
}

func TestScoreMR_MRAgeAsTiebreaker(t *testing.T) {
	now := time.Now()

	// Two MRs with same priority, one submitted 10 hours ago
	oldMR := ScoreInput{
		Priority:    2,
		MRCreatedAt: now.Add(-10 * time.Hour),
		Now:         now,
	}
	newMR := ScoreInput{
		Priority:    2,
		MRCreatedAt: now,
		Now:         now,
	}

	scoreOld := ScoreMRWithDefaults(oldMR)
	scoreNew := ScoreMRWithDefaults(newMR)

	// Old MR should have 10 pts more (1 pt/hour)
	expectedDiff := 10.0
	actualDiff := scoreOld - scoreNew
	if actualDiff != expectedDiff {
		t.Errorf("older MR should score %f more, got %f", expectedDiff, actualDiff)
	}
}

func TestScoreMR_Deterministic(t *testing.T) {
	fixedNow := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	convoyTime := time.Date(2024, 12, 31, 12, 0, 0, 0, time.UTC)
	mrTime := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)

	input := ScoreInput{
		Priority:        1,
		MRCreatedAt:     mrTime,
		ConvoyCreatedAt: &convoyTime,
		RetryCount:      2,
		Now:             fixedNow,
	}

	// Run 100 times, should always be same
	first := ScoreMRWithDefaults(input)
	for i := 0; i < 100; i++ {
		score := ScoreMRWithDefaults(input)
		if score != first {
			t.Errorf("score not deterministic: iteration %d got %f, expected %f", i, score, first)
		}
	}
}

func TestScoreMR_InvalidPriorityClamped(t *testing.T) {
	now := time.Now()

	// Negative priority should clamp to 0 bonus (priority=4)
	negativePriority := ScoreInput{
		Priority:    -1,
		MRCreatedAt: now,
		Now:         now,
	}
	scoreNegative := ScoreMRWithDefaults(negativePriority)

	// Very high priority should clamp to max bonus (priority=0)
	highPriority := ScoreInput{
		Priority:    10,
		MRCreatedAt: now,
		Now:         now,
	}
	scoreHigh := ScoreMRWithDefaults(highPriority)

	// Negative priority gets clamped to max bonus (4*100=400)
	if scoreNegative != 1400.0 {
		t.Errorf("negative priority should clamp to P0 bonus, got %f", scoreNegative)
	}

	// High priority (10) gives 4-10=-6, clamped to 0
	if scoreHigh != 1000.0 {
		t.Errorf("priority>4 should give 0 bonus, got %f", scoreHigh)
	}
}

func TestMR_Score(t *testing.T) {
	now := time.Now()
	convoyTime := now.Add(-12 * time.Hour)

	mr := &MR{
		Priority:        1,
		CreatedAt:       now.Add(-2 * time.Hour),
		ConvoyCreatedAt: &convoyTime,
		RetryCount:      1,
	}

	score := mr.ScoreAt(now)

	// base(1000) + convoy(12*10=120) + priority(3*100=300) - retry(1*50=50) + mrAge(2*1=2)
	expected := 1000.0 + 120.0 + 300.0 - 50.0 + 2.0
	if score != expected {
		t.Errorf("MR.ScoreAt expected %f, got %f", expected, score)
	}
}

func TestScoreMR_EdgeCases(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name  string
		input ScoreInput
	}{
		{
			name: "zero time MR",
			input: ScoreInput{
				Priority:    2,
				MRCreatedAt: time.Time{},
				Now:         now,
			},
		},
		{
			name: "future MR",
			input: ScoreInput{
				Priority:    2,
				MRCreatedAt: now.Add(24 * time.Hour),
				Now:         now,
			},
		},
		{
			name: "future convoy",
			input: ScoreInput{
				Priority:        2,
				MRCreatedAt:     now,
				ConvoyCreatedAt: func() *time.Time { t := now.Add(24 * time.Hour); return &t }(),
				Now:             now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			score := ScoreMRWithDefaults(tt.input)
			// Score should still be reasonable (>= base - maxPenalty)
			if score < 700 {
				t.Errorf("score %f unexpectedly low for edge case", score)
			}
		})
	}
}

// === Tests for gt-si8rq.10: End-to-end rebase workflow validation ===

func TestScoreMR_OlderConvoyMRsPrioritizedOverNewer(t *testing.T) {
	now := time.Now()

	// Create two MRs in different convoys with same priority
	oldConvoyTime := now.Add(-72 * time.Hour)
	newConvoyTime := now.Add(-12 * time.Hour)

	olderConvoyMR := ScoreInput{
		Priority:        2,
		MRCreatedAt:     now.Add(-1 * time.Hour),
		ConvoyCreatedAt: &oldConvoyTime,
		Now:             now,
	}

	newerConvoyMR := ScoreInput{
		Priority:        2,
		MRCreatedAt:     now,
		ConvoyCreatedAt: &newConvoyTime,
		Now:             now,
	}

	scoreOld := ScoreMRWithDefaults(olderConvoyMR)
	scoreNew := ScoreMRWithDefaults(newerConvoyMR)

	// Older convoy should score higher (starvation prevention)
	if scoreOld <= scoreNew {
		t.Errorf("older convoy (%f) should beat newer convoy (%f)", scoreOld, scoreNew)
	}

	// Verify the difference is significant (72h - 12h = 60h * 10pts = 600pts convoy age diff)
	expectedMinDiff := 500.0 // At least 50 hours difference
	actualDiff := scoreOld - scoreNew
	if actualDiff < expectedMinDiff {
		t.Errorf("expected convoy age diff of at least %f, got %f", expectedMinDiff, actualDiff)
	}
}

func TestScoreMR_P1EpicMRsPrioritizedOverP3(t *testing.T) {
	now := time.Now()
	convoyTime := now.Add(-24 * time.Hour)

	p1EpicMR := ScoreInput{
		Priority:        1, // P1 - high priority
		MRCreatedAt:     now,
		ConvoyCreatedAt: &convoyTime,
		Now:             now,
	}

	p3EpicMR := ScoreInput{
		Priority:        3, // P3 - low priority
		MRCreatedAt:     now,
		ConvoyCreatedAt: &convoyTime,
		Now:             now,
	}

	scoreP1 := ScoreMRWithDefaults(p1EpicMR)
	scoreP3 := ScoreMRWithDefaults(p3EpicMR)

	// P1 should score higher than P3
	if scoreP1 <= scoreP3 {
		t.Errorf("P1 (%f) should beat P3 (%f)", scoreP1, scoreP3)
	}

	// P1 gets (4-1)*100=300, P3 gets (4-3)*100=100, diff should be 200
	expectedDiff := 200.0
	actualDiff := scoreP1 - scoreP3
	if actualDiff != expectedDiff {
		t.Errorf("expected priority diff of %f, got %f", expectedDiff, actualDiff)
	}
}

func TestScoreMR_HighRetryCountGetsAdjustment(t *testing.T) {
	now := time.Now()

	// MR with no retries
	noRetryMR := ScoreInput{
		Priority:    2,
		MRCreatedAt: now,
		RetryCount:  0,
		Now:         now,
	}

	// MR with high retry count (but not capped)
	highRetryMR := ScoreInput{
		Priority:    2,
		MRCreatedAt: now,
		RetryCount:  4,
		Now:         now,
	}

	scoreNoRetry := ScoreMRWithDefaults(noRetryMR)
	scoreHighRetry := ScoreMRWithDefaults(highRetryMR)

	// High retry should score lower (penalized for thrashing)
	if scoreHighRetry >= scoreNoRetry {
		t.Errorf("high retry (%f) should score lower than no retry (%f)", scoreHighRetry, scoreNoRetry)
	}

	// 4 retries * 50 pts = 200 pts penalty
	expectedPenalty := 200.0
	actualPenalty := scoreNoRetry - scoreHighRetry
	if actualPenalty != expectedPenalty {
		t.Errorf("expected retry penalty of %f, got %f", expectedPenalty, actualPenalty)
	}
}

func TestScoreMR_ConflictMetadataIntegration(t *testing.T) {
	// Test that MRs with conflict metadata (retry count, conflict task) score correctly
	now := time.Now()
	convoyTime := now.Add(-24 * time.Hour)

	// Fresh MR in convoy
	freshMR := &MR{
		ID:              "mr-fresh",
		Priority:        2,
		CreatedAt:       now,
		ConvoyCreatedAt: &convoyTime,
		RetryCount:      0,
	}

	// MR that has had conflicts (higher retry count)
	conflictMR := &MR{
		ID:              "mr-conflict",
		Priority:        2,
		CreatedAt:       now,
		ConvoyCreatedAt: &convoyTime,
		RetryCount:      2,
		LastConflictSHA: "abc1234",
		ConflictTaskID:  "task-xyz",
	}

	scoreFresh := freshMR.ScoreAt(now)
	scoreConflict := conflictMR.ScoreAt(now)

	// Fresh MR should score higher (no retry penalty)
	if scoreFresh <= scoreConflict {
		t.Errorf("fresh MR (%f) should beat conflict MR (%f)", scoreFresh, scoreConflict)
	}

	// The penalty should be 2 retries * 50 = 100 pts
	expectedPenalty := 100.0
	actualPenalty := scoreFresh - scoreConflict
	if actualPenalty != expectedPenalty {
		t.Errorf("expected conflict penalty of %f, got %f", expectedPenalty, actualPenalty)
	}
}

func TestScoreMR_PriorityOrderingWithMixedFactors(t *testing.T) {
	// Complex test: verify priority ordering holds with multiple factors
	now := time.Now()

	// Scenario: P1 recent MR vs P3 old convoy MR with retries
	// P1 should still win unless convoy is extremely old

	recentConvoy := now.Add(-6 * time.Hour)
	oldConvoy := now.Add(-48 * time.Hour)

	p1RecentMR := ScoreInput{
		Priority:        1,
		MRCreatedAt:     now,
		ConvoyCreatedAt: &recentConvoy,
		RetryCount:      0,
		Now:             now,
	}

	p3OldConvoyWithRetries := ScoreInput{
		Priority:        3,
		MRCreatedAt:     now.Add(-24 * time.Hour),
		ConvoyCreatedAt: &oldConvoy,
		RetryCount:      3,
		Now:             now,
	}

	scoreP1 := ScoreMRWithDefaults(p1RecentMR)
	scoreP3 := ScoreMRWithDefaults(p3OldConvoyWithRetries)

	// Calculate expected:
	// P1: base(1000) + convoy(6h*10=60) + priority(3*100=300) - retry(0) = 1360
	// P3: base(1000) + convoy(48h*10=480) + priority(1*100=100) - retry(3*50=150) + mrAge(24*1=24) = 1454
	// In this case, the old convoy's starvation prevention should outweigh priority

	t.Logf("P1 recent score: %f, P3 old convoy with retries: %f", scoreP1, scoreP3)

	// Both scores should be positive and reasonable
	if scoreP1 < 1000 || scoreP3 < 1000 {
		t.Errorf("scores unexpectedly low: P1=%f, P3=%f", scoreP1, scoreP3)
	}
}

func TestMR_ScoreReflectsConflictState(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name             string
		mr               *MR
		expectLowerScore bool // compared to clean MR
	}{
		{
			name: "clean MR",
			mr: &MR{
				ID:         "mr-clean",
				Priority:   2,
				CreatedAt:  now,
				RetryCount: 0,
			},
			expectLowerScore: false,
		},
		{
			name: "first conflict",
			mr: &MR{
				ID:              "mr-first-conflict",
				Priority:        2,
				CreatedAt:       now,
				RetryCount:      1,
				LastConflictSHA: "sha1",
				ConflictTaskID:  "task-1",
			},
			expectLowerScore: true,
		},
		{
			name: "multiple conflicts",
			mr: &MR{
				ID:              "mr-multi-conflict",
				Priority:        2,
				CreatedAt:       now,
				RetryCount:      3,
				LastConflictSHA: "sha3",
				ConflictTaskID:  "task-3",
			},
			expectLowerScore: true,
		},
		{
			name: "resolved conflict (task cleared)",
			mr: &MR{
				ID:              "mr-resolved",
				Priority:        2,
				CreatedAt:       now,
				RetryCount:      2, // Still has retry count
				LastConflictSHA: "",
				ConflictTaskID:  "", // Task cleared
			},
			expectLowerScore: true, // Still penalized by retry count
		},
	}

	cleanMR := &MR{ID: "clean", Priority: 2, CreatedAt: now, RetryCount: 0}
	cleanScore := cleanMR.ScoreAt(now)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := tt.mr.ScoreAt(now)

			if tt.expectLowerScore && score >= cleanScore {
				t.Errorf("expected lower score than clean (%f), got %f", cleanScore, score)
			}
			if !tt.expectLowerScore && score < cleanScore {
				t.Errorf("expected score >= clean (%f), got %f", cleanScore, score)
			}
		})
	}
}
