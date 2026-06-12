package progress

import (
	"context"
	"errors"
	"testing"

	"coursehub/internal/grading"
)

// fakeStore records the last progress written and can be told to fail.
type fakeStore struct {
	progress map[string]float64
	failWith error
}

func newFakeStore() *fakeStore {
	return &fakeStore{progress: map[string]float64{}}
}

func (f *fakeStore) SetProgress(_ context.Context, id string, p float64) error {
	if f.failWith != nil {
		return f.failWith
	}
	f.progress[id] = p
	return nil
}

// countingObserver records how many events it received.
type countingObserver struct{ calls int }

func (c *countingObserver) OnGraded(context.Context, Event) error {
	c.calls++
	return nil
}

func TestProgressTracker_computeProgress(t *testing.T) {
	tests := []struct {
		name  string
		event Event
		want  float64
	}{
		{"half of planned", Event{GradedCount: 2, PlannedCount: 4}, 0.5},
		{"all of planned", Event{GradedCount: 4, PlannedCount: 4}, 1},
		{"over planned clamps to 1", Event{GradedCount: 5, PlannedCount: 4}, 1},
		{"no plan, passed", Event{Result: grading.Result{Passed: true}}, 1},
		{"no plan, failed", Event{Result: grading.Result{Passed: false}}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := newFakeStore()
			tr := NewProgressTracker(store)
			if err := tr.OnGraded(context.Background(), withID(tt.event)); err != nil {
				t.Fatalf("OnGraded: %v", err)
			}
			if got := store.progress["e1"]; got != tt.want {
				t.Errorf("progress = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProgressTracker_storeError(t *testing.T) {
	store := newFakeStore()
	store.failWith = errors.New("db down")
	tr := NewProgressTracker(store)
	if err := tr.OnGraded(context.Background(), withID(Event{GradedCount: 1, PlannedCount: 1})); err == nil {
		t.Fatal("expected error from failing store, got nil")
	}
}

func TestPublisher_NotifyAll(t *testing.T) {
	a, b := &countingObserver{}, &countingObserver{}
	p := NewPublisher(a)
	p.Subscribe(b)

	if err := p.Notify(context.Background(), withID(Event{})); err != nil {
		t.Fatalf("Notify: %v", err)
	}
	if a.calls != 1 || b.calls != 1 {
		t.Errorf("calls a=%d b=%d, want 1 each", a.calls, b.calls)
	}
}

func TestPublisher_JoinsErrors(t *testing.T) {
	failing := &fakeStore{progress: map[string]float64{}, failWith: errors.New("boom")}
	tracker := NewProgressTracker(failing)
	ok := &countingObserver{}
	p := NewPublisher(tracker, ok)

	err := p.Notify(context.Background(), withID(Event{GradedCount: 1, PlannedCount: 1}))
	if err == nil {
		t.Fatal("expected joined error, got nil")
	}
	// The healthy observer still ran despite the tracker failing.
	if ok.calls != 1 {
		t.Errorf("healthy observer calls = %d, want 1", ok.calls)
	}
}

func withID(e Event) Event {
	e.EnrollmentID = "e1"
	return e
}
