// Package progress implements the Observer pattern for reacting to grading
// events. When the service records a grade it publishes an Event; subscribed
// observers (a progress tracker, a notifier, and any future addition such as an
// e-mail sender) react independently. The publisher is the Subject; observers
// never know about each other, so new reactions are added without touching the
// service.
package progress

import (
	"context"
	"errors"
	"time"

	"coursehub/internal/grading"
)

// Event describes a grade that has just been recorded for an enrollment.
type Event struct {
	EnrollmentID string
	StudentID    string
	CourseID     string
	Result       grading.Result
	GradedCount  int // assessments recorded so far
	PlannedCount int // assessments planned for the course (0 = unknown)
	OccurredAt   time.Time
}

// Observer reacts to grading events. Implementations must not assume they are
// the only observer and should be independent of one another.
type Observer interface {
	OnGraded(ctx context.Context, e Event) error
}

// Publisher is the Subject side of the pattern: it keeps a list of observers and
// fans an Event out to all of them. It is safe to construct with zero observers.
type Publisher struct {
	observers []Observer
}

// NewPublisher returns a Publisher subscribed to the given observers.
func NewPublisher(observers ...Observer) *Publisher {
	return &Publisher{observers: observers}
}

// Subscribe registers an additional observer.
func (p *Publisher) Subscribe(o Observer) {
	p.observers = append(p.observers, o)
}

// Notify delivers the event to every observer. All observers run even if some
// fail; their errors are joined so a single misbehaving observer neither stops
// the others nor is silently dropped.
func (p *Publisher) Notify(ctx context.Context, e Event) error {
	var errs []error
	for _, o := range p.observers {
		if err := o.OnGraded(ctx, e); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
