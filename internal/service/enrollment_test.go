package service

import (
	"context"
	"errors"
	"testing"

	"coursehub/internal/course"
	"coursehub/internal/domain"
	"coursehub/internal/grading"
	"coursehub/internal/progress"
)

// fakeEnrollmentRepo is an in-memory EnrollmentRepository and progress.Store.
type fakeEnrollmentRepo struct {
	byID map[string]*domain.Enrollment
}

func newFakeEnrollmentRepo() *fakeEnrollmentRepo {
	return &fakeEnrollmentRepo{byID: map[string]*domain.Enrollment{}}
}

func (r *fakeEnrollmentRepo) Create(_ context.Context, e *domain.Enrollment) error {
	r.byID[e.ID] = e
	return nil
}

func (r *fakeEnrollmentRepo) GetByID(_ context.Context, id string) (*domain.Enrollment, error) {
	e, ok := r.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	clone := *e
	return &clone, nil
}

func (r *fakeEnrollmentRepo) ListByStudent(_ context.Context, studentID string) ([]*domain.Enrollment, error) {
	var out []*domain.Enrollment
	for _, e := range r.byID {
		if e.StudentID == studentID {
			clone := *e
			out = append(out, &clone)
		}
	}
	return out, nil
}

func (r *fakeEnrollmentRepo) Exists(_ context.Context, studentID, courseID string) (bool, error) {
	for _, e := range r.byID {
		if e.StudentID == studentID && e.CourseID == courseID {
			return true, nil
		}
	}
	return false, nil
}

func (r *fakeEnrollmentRepo) Update(_ context.Context, e *domain.Enrollment) error {
	if _, ok := r.byID[e.ID]; !ok {
		return domain.ErrNotFound
	}
	clone := *e
	r.byID[e.ID] = &clone
	return nil
}

func (r *fakeEnrollmentRepo) SetProgress(_ context.Context, id string, p float64) error {
	e, ok := r.byID[id]
	if !ok {
		return domain.ErrNotFound
	}
	e.Progress = p
	return nil
}

// fakeCourses returns a single decorated course for any ID.
type fakeCourses struct{ c domain.Course }

func (f fakeCourses) Get(context.Context, string) (domain.Course, error) {
	if f.c == nil {
		return nil, domain.ErrNotFound
	}
	return f.c, nil
}

func buildCourse(t *testing.T, strategy string) domain.Course {
	t.Helper()
	c, err := course.NewFactory().Build(domain.CourseRecord{
		ID: "c1", Code: "CS101", Title: "Algo", Grading: strategy,
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func newEnrollmentSvc(t *testing.T, strategy string) (*EnrollmentService, *fakeEnrollmentRepo) {
	t.Helper()
	repo := newFakeEnrollmentRepo()
	pub := progress.NewPublisher(progress.NewProgressTracker(repo))
	svc := NewEnrollmentService(repo, fakeCourses{c: buildCourse(t, strategy)}, pub)
	return svc, repo
}

func TestEnrollmentService_Enroll_RejectsDuplicate(t *testing.T) {
	svc, _ := newEnrollmentSvc(t, grading.StrategyWeighted)

	if _, err := svc.Enroll(context.Background(), "s1", "c1"); err != nil {
		t.Fatalf("first enroll: %v", err)
	}
	_, err := svc.Enroll(context.Background(), "s1", "c1")
	if !errors.Is(err, domain.ErrAlreadyEnrolled) {
		t.Fatalf("second enroll err = %v, want ErrAlreadyEnrolled", err)
	}
}

func TestEnrollmentService_RecordGrade_AppliesStrategyAndProgress(t *testing.T) {
	svc, _ := newEnrollmentSvc(t, grading.StrategyWeighted)
	ctx := context.Background()

	e, err := svc.Enroll(ctx, "s1", "c1")
	if err != nil {
		t.Fatal(err)
	}

	// Grade 1 of 2 planned assessments: status stays active, progress 0.5.
	got, err := svc.RecordGrade(ctx, e.ID, []GradeInput{
		{Name: "exam", Score: 90, MaxScore: 100, Weight: 1},
	}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got.FinalGrade != 90 || got.Letter != "A" || !got.Passed {
		t.Errorf("result = {%.0f %q %v}, want {90 A true}", got.FinalGrade, got.Letter, got.Passed)
	}
	if got.Status != domain.StatusActive {
		t.Errorf("status = %q, want active (partial)", got.Status)
	}
	if got.Progress != 0.5 {
		t.Errorf("progress = %v, want 0.5", got.Progress)
	}

	// Grade the full set (default planned): status completed, progress 1.
	got, err = svc.RecordGrade(ctx, e.ID, []GradeInput{
		{Name: "exam", Score: 90, MaxScore: 100, Weight: 1},
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.StatusCompleted {
		t.Errorf("status = %q, want completed", got.Status)
	}
	if got.Progress != 1 {
		t.Errorf("progress = %v, want 1", got.Progress)
	}
}

func TestEnrollmentService_RecordGrade_RequiresAssessments(t *testing.T) {
	svc, repo := newEnrollmentSvc(t, grading.StrategyWeighted)
	repo.byID["e1"] = &domain.Enrollment{ID: "e1", StudentID: "s1", CourseID: "c1"}

	_, err := svc.RecordGrade(context.Background(), "e1", nil, 0)
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
}
