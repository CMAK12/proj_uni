package progress

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"coursehub/internal/grading"
)

func TestComputeProgress_Table(t *testing.T) {
	tests := []struct {
		name  string
		event Event
		want  float64
	}{
		{"quarter", Event{GradedCount: 1, PlannedCount: 4}, 0.25},
		{"half", Event{GradedCount: 2, PlannedCount: 4}, 0.5},
		{"complete", Event{GradedCount: 4, PlannedCount: 4}, 1},
		{"over planned clamps", Event{GradedCount: 9, PlannedCount: 4}, 1},
		{"zero graded", Event{GradedCount: 0, PlannedCount: 4}, 0},
		{"no plan passed", Event{Result: grading.Result{Passed: true}}, 1},
		{"no plan failed", Event{Result: grading.Result{Passed: false}}, 0},
		{"negative plan treated as no plan, passed", Event{PlannedCount: -1, Result: grading.Result{Passed: true}}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := computeProgress(tt.event); got != tt.want {
				t.Errorf("computeProgress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLogNotifier_OnGraded(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	n := NewLogNotifier(log)

	err := n.OnGraded(context.Background(), Event{
		EnrollmentID: "e1", StudentID: "s1", CourseID: "c1",
		Result: grading.Result{Final: 90, Letter: "A", Passed: true},
	})
	if err != nil {
		t.Fatalf("OnGraded: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"grade recorded", "e1", "s1", "c1"} {
		if !bytes.Contains([]byte(out), []byte(want)) {
			t.Errorf("log output missing %q: %s", want, out)
		}
	}
}

func TestNewLogNotifier_NilLoggerDefaults(t *testing.T) {
	n := NewLogNotifier(nil)
	if n.log == nil {
		t.Fatal("expected default logger, got nil")
	}
	if err := n.OnGraded(context.Background(), Event{EnrollmentID: "e1"}); err != nil {
		t.Errorf("OnGraded with default logger: %v", err)
	}
}

func TestPublisher_NoObservers(t *testing.T) {
	p := NewPublisher()
	if err := p.Notify(context.Background(), Event{EnrollmentID: "e1"}); err != nil {
		t.Errorf("Notify with no observers = %v, want nil", err)
	}
}

func TestPublisher_SubscribeAddsObserver(t *testing.T) {
	p := NewPublisher()
	c := &countingObserver{}
	p.Subscribe(c)
	if err := p.Notify(context.Background(), Event{EnrollmentID: "e1"}); err != nil {
		t.Fatalf("Notify: %v", err)
	}
	if c.calls != 1 {
		t.Errorf("calls = %d, want 1", c.calls)
	}
}

func TestProgressTracker_PersistsComputedValue(t *testing.T) {
	store := newFakeStore()
	tr := NewProgressTracker(store)
	if err := tr.OnGraded(context.Background(), Event{EnrollmentID: "e7", GradedCount: 3, PlannedCount: 6}); err != nil {
		t.Fatalf("OnGraded: %v", err)
	}
	if got := store.progress["e7"]; got != 0.5 {
		t.Errorf("stored progress = %v, want 0.5", got)
	}
}
