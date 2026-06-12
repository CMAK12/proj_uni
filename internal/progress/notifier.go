package progress

import (
	"context"
	"log/slog"
)

// LogNotifier is an Observer that emits a structured log line whenever a grade
// is recorded. It stands in for any out-of-band notification (e-mail, push) and
// demonstrates that observers are independent: it neither reads nor writes the
// progress the tracker maintains.
type LogNotifier struct {
	log *slog.Logger
}

// NewLogNotifier returns a notifier writing to the given logger. A nil logger
// defaults to slog.Default so the notifier is always safe to use.
func NewLogNotifier(log *slog.Logger) *LogNotifier {
	if log == nil {
		log = slog.Default()
	}
	return &LogNotifier{log: log}
}

// OnGraded implements Observer.
func (n *LogNotifier) OnGraded(ctx context.Context, e Event) error {
	n.log.InfoContext(ctx, "grade recorded",
		"enrollment", e.EnrollmentID,
		"student", e.StudentID,
		"course", e.CourseID,
		"final", e.Result.Final,
		"letter", e.Result.Letter,
		"passed", e.Result.Passed,
	)
	return nil
}
