package grading

// PassFail evaluates components against a single percentage threshold. The final
// mark is the unweighted average; the student passes when it meets the
// threshold. Letter is "P" (pass) or "F" (fail).
type PassFail struct {
	// Threshold is the minimum final mark (0..100) required to pass.
	Threshold float64
}

// NewPassFail returns a PassFail strategy with the given pass threshold.
func NewPassFail(threshold float64) PassFail {
	return PassFail{Threshold: threshold}
}

// Name implements Strategy.
func (PassFail) Name() string { return StrategyPassFail }

// Evaluate implements Strategy.
func (p PassFail) Evaluate(components []Component) Result {
	final := round2(average(components))
	passed := final >= p.Threshold
	letter := "F"
	if passed {
		letter = "P"
	}
	return Result{Final: final, Letter: letter, Passed: passed}
}
