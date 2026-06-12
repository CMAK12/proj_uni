package grading

// WeightedAverage computes the final mark as a weighted mean of component
// percentages. If no component carries a weight it falls back to an unweighted
// average, so it degrades gracefully. Pass threshold is 60/100.
type WeightedAverage struct{}

// Name implements Strategy.
func (WeightedAverage) Name() string { return StrategyWeighted }

// Evaluate implements Strategy.
func (WeightedAverage) Evaluate(components []Component) Result {
	if len(components) == 0 {
		return Result{}
	}

	var weightedSum, totalWeight float64
	for _, c := range components {
		weightedSum += c.Percent() * c.Weight
		totalWeight += c.Weight
	}

	var final float64
	if totalWeight > 0 {
		final = weightedSum / totalWeight * 100
	} else {
		final = average(components)
	}

	return Result{
		Final:  round2(final),
		Letter: letterFor(final),
		Passed: final >= 60,
	}
}

// round2 rounds to two decimal places to keep stored grades tidy.
func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}
