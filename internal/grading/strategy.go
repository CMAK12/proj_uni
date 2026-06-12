// Package grading implements the Strategy pattern for evaluating a student's
// assessments. Each course chooses a Strategy by name; the service layer applies
// it without knowing which concrete algorithm runs. Adding a new grading scheme
// means adding a type here — no caller changes required.
package grading

import "fmt"

// Component is one graded item fed to a Strategy. It is deliberately decoupled
// from domain.Assessment so the grading package depends on nothing else.
type Component struct {
	Name     string
	Score    float64 // points earned
	MaxScore float64 // points possible (> 0)
	Weight   float64 // relative weight; used only by weighted strategies
}

// Percent returns the component score as a fraction in [0,1]. A non-positive
// MaxScore yields 0 to avoid division by zero.
func (c Component) Percent() float64 {
	if c.MaxScore <= 0 {
		return 0
	}
	return c.Score / c.MaxScore
}

// Result is the outcome of evaluating a set of components.
type Result struct {
	Final  float64 // final mark on a 0..100 scale
	Letter string  // letter grade, when the strategy assigns one
	Passed bool    // whether the student passed
}

// Strategy evaluates assessment components into a Result. Implementations must
// be safe for concurrent use (they hold no mutable state).
type Strategy interface {
	// Name is the stable identifier persisted with a course.
	Name() string
	// Evaluate computes the result for the given components.
	Evaluate(components []Component) Result
}

// Registry of available strategies keyed by Name. Lookup keeps the service layer
// free of switch statements over concrete types.
var registry = map[string]Strategy{
	StrategyWeighted: WeightedAverage{},
	StrategyPassFail: NewPassFail(60),
	StrategyLetter:   LetterGrade{},
}

// Strategy name constants.
const (
	StrategyWeighted = "weighted"
	StrategyPassFail = "passfail"
	StrategyLetter   = "letter"
)

// Get returns the Strategy registered under name, or an error if unknown.
func Get(name string) (Strategy, error) {
	s, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("grading: unknown strategy %q", name)
	}
	return s, nil
}

// Names returns the registered strategy names (unordered).
func Names() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	return names
}

// average returns the unweighted mean percentage (0..100) of the components.
func average(components []Component) float64 {
	if len(components) == 0 {
		return 0
	}
	var sum float64
	for _, c := range components {
		sum += c.Percent()
	}
	return sum / float64(len(components)) * 100
}
