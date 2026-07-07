package services

// RoundStats is the single source of truth for round/vote statistics
// (consensus, average, agreement, and value distribution). It replaces the
// three previously-divergent inline implementations that used to live in
// RoomManager.RevealVotes, RoomManager.CreateNextRound and
// handlers.calculateStats.
type RoundStats struct {
	// Total is the number of votes considered (including non-numeric ones).
	Total int
	// ValueBreakdown counts every raw vote value (numeric and non-numeric).
	ValueBreakdown map[string]int
	// Average is the mean of all numeric votes (0 IS included as a valid
	// estimate). Only meaningful when HasAverage is true.
	Average float64
	// HasAverage is true when at least one vote parsed as numeric.
	HasAverage bool
	// Consensus is true when there are at least 2 votes and they are all
	// identical. A single vote never counts as consensus - this mirrors the
	// original RevealVotes/CreateNextRound rule, which exists so a single
	// early voter can't trivially trigger (and infinitely streak) consensus.
	Consensus bool
	// AgreementPercentage is the share (0-100) of votes matching
	// MostCommonValue. 0 when there are no votes.
	AgreementPercentage float64
	// MostCommonValue is the value with the highest count. Deterministic tie
	// -break: first value (in input order) to reach the current max count.
	MostCommonValue string
}

// ComputeRoundStats computes consensus/average/agreement statistics from a
// slice of raw vote values. It is the single implementation used by every
// call site that previously computed these statistics independently.
func ComputeRoundStats(values []string) RoundStats {
	stats := RoundStats{
		ValueBreakdown: make(map[string]int),
	}

	if len(values) == 0 {
		return stats
	}

	validator := NewVoteValidator()

	var sum float64
	var numericCount int
	var mostCommonCount int

	for _, value := range values {
		stats.ValueBreakdown[value]++

		// Track most common value using a stable "first value to reach the
		// current max count" tie-break.
		if stats.ValueBreakdown[value] > mostCommonCount {
			mostCommonCount = stats.ValueBreakdown[value]
			stats.MostCommonValue = value
		}

		if num, ok := validator.ParseNumericValue(value); ok {
			sum += num
			numericCount++
		}
	}

	stats.Total = len(values)

	if numericCount > 0 {
		stats.Average = sum / float64(numericCount)
		stats.HasAverage = true
	}

	stats.AgreementPercentage = (float64(mostCommonCount) / float64(stats.Total)) * 100
	stats.Consensus = stats.Total >= 2 && mostCommonCount == stats.Total

	return stats
}
