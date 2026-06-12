package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"coursehub/internal/domain"
	"coursehub/internal/grading"
)

func newTestDB(t *testing.T) (*StudentRepo, *CourseRepo, *EnrollmentRepo) {
	t.Helper()
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewStudentRepo(db), NewCourseRepo(db), NewEnrollmentRepo(db)
}

func TestStudentRepo_CRUDAndDuplicate(t *testing.T) {
	students, _, _ := newTestDB(t)
	ctx := context.Background()

	s := &domain.Student{ID: "s1", Name: "Іван", Email: "ivan@lnu.ua", CreatedAt: time.Now().UTC()}
	if err := students.Create(ctx, s); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := students.GetByID(ctx, "s1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Email != s.Email {
		t.Errorf("email = %q, want %q", got.Email, s.Email)
	}

	dup := &domain.Student{ID: "s2", Name: "Other", Email: "ivan@lnu.ua", CreatedAt: time.Now().UTC()}
	if err := students.Create(ctx, dup); !errors.Is(err, domain.ErrDuplicate) {
		t.Errorf("duplicate email err = %v, want ErrDuplicate", err)
	}

	if _, err := students.GetByID(ctx, "missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("missing err = %v, want ErrNotFound", err)
	}
}

func TestCourseRepo_RoundTripFeatures(t *testing.T) {
	_, courses, _ := newTestDB(t)
	ctx := context.Background()

	rec := domain.CourseRecord{
		ID: "c1", Code: "CS101", Title: "Algo", Credits: 5,
		Type: domain.CourseTypeOnline, Grading: grading.StrategyWeighted,
		Features: []string{"certified"}, Platform: "Moodle", CreatedAt: time.Now().UTC(),
	}
	if err := courses.Create(ctx, rec); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := courses.GetByID(ctx, "c1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Platform != "Moodle" || len(got.Features) != 1 || got.Features[0] != "certified" {
		t.Errorf("round trip mismatch: %+v", got)
	}
	if got.Type != domain.CourseTypeOnline {
		t.Errorf("type = %q, want online", got.Type)
	}
}

func TestEnrollmentRepo_UpdateAndAssessments(t *testing.T) {
	students, courses, enrollments := newTestDB(t)
	ctx := context.Background()

	mustCreateStudent(t, students, "s1")
	mustCreateCourse(t, courses, "c1")

	e := &domain.Enrollment{
		ID: "e1", StudentID: "s1", CourseID: "c1",
		Status: domain.StatusPending, EnrolledAt: time.Now().UTC(),
	}
	if err := enrollments.Create(ctx, e); err != nil {
		t.Fatalf("create enrollment: %v", err)
	}

	// Duplicate (same student+course) is rejected.
	dup := &domain.Enrollment{ID: "e2", StudentID: "s1", CourseID: "c1", Status: domain.StatusPending, EnrolledAt: time.Now().UTC()}
	if err := enrollments.Create(ctx, dup); !errors.Is(err, domain.ErrAlreadyEnrolled) {
		t.Errorf("duplicate enrollment err = %v, want ErrAlreadyEnrolled", err)
	}

	e.Status = domain.StatusCompleted
	e.FinalGrade = 88
	e.Letter = "B"
	e.Passed = true
	e.Assessments = []domain.Assessment{
		{ID: "a1", Name: "exam", Score: 88, MaxScore: 100, Weight: 1},
	}
	if err := enrollments.Update(ctx, e); err != nil {
		t.Fatalf("update: %v", err)
	}

	if err := enrollments.SetProgress(ctx, "e1", 1); err != nil {
		t.Fatalf("set progress: %v", err)
	}

	got, err := enrollments.GetByID(ctx, "e1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != domain.StatusCompleted || got.FinalGrade != 88 || got.Progress != 1 {
		t.Errorf("update not persisted: %+v", got)
	}
	if len(got.Assessments) != 1 || got.Assessments[0].Name != "exam" {
		t.Errorf("assessments not persisted: %+v", got.Assessments)
	}

	exists, err := enrollments.Exists(ctx, "s1", "c1")
	if err != nil || !exists {
		t.Errorf("Exists = %v, %v; want true, nil", exists, err)
	}
}

func mustCreateStudent(t *testing.T, repo *StudentRepo, id string) {
	t.Helper()
	err := repo.Create(context.Background(), &domain.Student{
		ID: id, Name: "N", Email: id + "@lnu.ua", CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("seed student: %v", err)
	}
}

func mustCreateCourse(t *testing.T, repo *CourseRepo, id string) {
	t.Helper()
	err := repo.Create(context.Background(), domain.CourseRecord{
		ID: id, Code: id, Title: "T", Credits: 1,
		Type: domain.CourseTypeStandard, Grading: grading.StrategyWeighted,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("seed course: %v", err)
	}
}
