package domain

import "time"

// Course is the abstraction shared by the base course and every decorator
// (see package course). Modelling it as an interface is what makes the
// Decorator pattern possible: a decorator both implements Course and wraps a
// Course, so wrapped courses are indistinguishable from plain ones to callers.
type Course interface {
	ID() string
	Code() string
	Title() string
	Credits() int
	// GradingStrategy is the name of the evaluation strategy this course uses
	// (resolved to a grading.Strategy by the service layer).
	GradingStrategy() string
	// Features lists the extra capabilities contributed by decorators
	// (e.g. "certified", "online"). A bare course returns its own base set.
	Features() []string
	// Describe returns a one-line human description; decorators extend it.
	Describe() string
}

// CourseType enumerates the kinds of course the factory can build.
type CourseType string

const (
	// CourseTypeStandard is an ordinary in-person course.
	CourseTypeStandard CourseType = "standard"
	// CourseTypeOnline is delivered through an online platform.
	CourseTypeOnline CourseType = "online"
)

// CourseRecord is the flat, persistable representation of a course. The factory
// turns a record into a decorated Course; repositories store a Course back into
// a record. Keeping persistence flat avoids serialising the decorator chain.
type CourseRecord struct {
	ID        string
	Code      string
	Title     string
	Credits   int
	Type      CourseType
	Grading   string
	Features  []string // requested decorations, e.g. ["certified"]
	Platform  string   // used when Type == online
	CreatedAt time.Time
}
