package grading

// LetterGrade maps the unweighted average percentage onto an A–F scale.
// Anything D or above (>= 60) is a pass.
type LetterGrade struct{}

// Name implements Strategy.
func (LetterGrade) Name() string { return StrategyLetter }

// Evaluate implements Strategy.
func (LetterGrade) Evaluate(components []Component) Result {
	final := round2(average(components))
	return Result{
		Final:  final,
		Letter: letterFor(final),
		Passed: final >= 60,
	}
}

// letterFor converts a 0..100 mark to a letter grade. Shared by the strategies
// that assign letters.
func letterFor(final float64) string {
	switch {
	case final >= 90:
		return "A"
	case final >= 80:
		return "B"
	case final >= 70:
		return "C"
	case final >= 60:
		return "D"
	default:
		return "F"
	}
}
