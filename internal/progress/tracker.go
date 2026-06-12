package progress

import (
	"context"
	"fmt"
)

// Store is the narrow port the tracker needs to persist computed progress. It is
// defined here, where it is consumed; the storage layer supplies an adapter.
type Store interface {
	SetProgress(ctx context.Context, enrollmentID string, progress float64) error
}

// ProgressTracker is an Observer that converts a grading Event into a progress
// fraction (0..1) and persists it. Progress is the share of planned assessments
// that have been graded; with an unknown plan it falls back to pass/fail.
type ProgressTracker struct {
	store Store
}

// NewProgressTracker returns a tracker that writes through store.
func NewProgressTracker(store Store) *ProgressTracker {
	return &ProgressTracker{store: store}
}

// OnGraded implements Observer.
func (t *ProgressTracker) OnGraded(ctx context.Context, e Event) error {
	progress := computeProgress(e)
	if err := t.store.SetProgress(ctx, e.EnrollmentID, progress); err != nil {
		return fmt.Errorf("track progress for enrollment %s: %w", e.EnrollmentID, err)
	}
	return nil
}

// computeProgress derives a 0..1 completion fraction from the event.
func computeProgress(e Event) float64 {
	if e.PlannedCount > 0 {
		p := float64(e.GradedCount) / float64(e.PlannedCount)
		if p > 1 {
			return 1
		}
		return p
	}
	// No plan known: treat a passing result as fully complete.
	if e.Result.Passed {
		return 1
	}
	return 0
}
