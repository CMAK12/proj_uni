package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"coursehub/internal/domain"
	"coursehub/internal/grading"
)

func TestOpen_BadDSN(t *testing.T) {
	if _, err := Open("/nonexistent-dir-xyz/does/not/exist.db"); err == nil {
		t.Fatal("expected error opening db in nonexistent directory")
	}
}

func TestStudentRepo_List(t *testing.T) {
	students, _, _ := newTestDB(t)
	ctx := context.Background()

	empty, err := students.List(ctx)
	if err != nil {
		t.Fatalf("list empty: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("empty list len = %d, want 0", len(empty))
	}

	base := time.Now().UTC()
	for i, id := range []string{"s1", "s2", "s3"} {
		err := students.Create(ctx, &domain.Student{
			ID: id, Name: id, Email: id + "@lnu.ua", CreatedAt: base.Add(time.Duration(i) * time.Second),
		})
		if err != nil {
			t.Fatalf("create %s: %v", id, err)
		}
	}

	list, err := students.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("list len = %d, want 3", len(list))
	}
	// Ordered by created_at.
	if list[0].ID != "s1" || list[2].ID != "s3" {
		t.Errorf("unexpected order: %s..%s", list[0].ID, list[2].ID)
	}
}

func TestCourseRepo_List_OrderedByCode(t *testing.T) {
	_, courses, _ := newTestDB(t)
	ctx := context.Background()

	for _, code := range []string{"CS300", "CS100", "CS200"} {
		err := courses.Create(ctx, domain.CourseRecord{
			ID: code, Code: code, Title: "T", Credits: 1,
			Type: domain.CourseTypeStandard, Grading: grading.StrategyWeighted, CreatedAt: time.Now().UTC(),
		})
		if err != nil {
			t.Fatalf("create %s: %v", code, err)
		}
	}
	list, err := courses.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 3 || list[0].Code != "CS100" || list[2].Code != "CS300" {
		t.Errorf("unexpected order: %+v", list)
	}
}

func TestCourseRepo_GetByID_NotFound(t *testing.T) {
	_, courses, _ := newTestDB(t)
	if _, err := courses.GetByID(context.Background(), "missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestCourseRepo_StandardNoFeatures(t *testing.T) {
	_, courses, _ := newTestDB(t)
	ctx := context.Background()
	rec := domain.CourseRecord{
		ID: "c1", Code: "CS1", Title: "Algo", Credits: 5,
		Type: domain.CourseTypeStandard, Grading: grading.StrategyWeighted, CreatedAt: time.Now().UTC(),
	}
	if err := courses.Create(ctx, rec); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := courses.GetByID(ctx, "c1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(got.Features) != 0 {
		t.Errorf("features = %v, want empty", got.Features)
	}
	if got.Type != domain.CourseTypeStandard {
		t.Errorf("type = %q, want standard", got.Type)
	}
}

func TestCourseRepo_DuplicateCode(t *testing.T) {
	_, courses, _ := newTestDB(t)
	ctx := context.Background()
	rec := domain.CourseRecord{
		ID: "c1", Code: "DUP", Title: "T", Credits: 1,
		Type: domain.CourseTypeStandard, Grading: grading.StrategyWeighted, CreatedAt: time.Now().UTC(),
	}
	if err := courses.Create(ctx, rec); err != nil {
		t.Fatalf("create: %v", err)
	}
	rec.ID = "c2"
	if err := courses.Create(ctx, rec); !errors.Is(err, domain.ErrDuplicate) {
		t.Errorf("duplicate code err = %v, want ErrDuplicate", err)
	}
}

func TestEnrollmentRepo_ListByStudent(t *testing.T) {
	students, courses, enrollments := newTestDB(t)
	ctx := context.Background()
	mustCreateStudent(t, students, "s1")
	mustCreateCourse(t, courses, "c1")
	mustCreateCourse(t, courses, "c2")

	empty, err := enrollments.ListByStudent(ctx, "s1")
	if err != nil {
		t.Fatalf("list empty: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("empty len = %d, want 0", len(empty))
	}

	base := time.Now().UTC()
	for i, e := range []*domain.Enrollment{
		{ID: "e1", StudentID: "s1", CourseID: "c1", Status: domain.StatusPending, EnrolledAt: base},
		{ID: "e2", StudentID: "s1", CourseID: "c2", Status: domain.StatusActive, EnrolledAt: base.Add(time.Second)},
	} {
		_ = i
		if err := enrollments.Create(ctx, e); err != nil {
			t.Fatalf("create %s: %v", e.ID, err)
		}
	}

	list, err := enrollments.ListByStudent(ctx, "s1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len = %d, want 2", len(list))
	}
	if list[0].ID != "e1" {
		t.Errorf("order: first = %q, want e1", list[0].ID)
	}
}

func TestEnrollmentRepo_GetByID_NotFound(t *testing.T) {
	_, _, enrollments := newTestDB(t)
	if _, err := enrollments.GetByID(context.Background(), "missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestEnrollmentRepo_SetProgress_NotFound(t *testing.T) {
	_, _, enrollments := newTestDB(t)
	if err := enrollments.SetProgress(context.Background(), "missing", 0.5); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestEnrollmentRepo_Update_NotFound(t *testing.T) {
	_, _, enrollments := newTestDB(t)
	e := &domain.Enrollment{ID: "missing", StudentID: "s1", CourseID: "c1", Status: domain.StatusActive, EnrolledAt: time.Now().UTC()}
	if err := enrollments.Update(context.Background(), e); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestEnrollmentRepo_Exists_False(t *testing.T) {
	_, _, enrollments := newTestDB(t)
	exists, err := enrollments.Exists(context.Background(), "s1", "c1")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if exists {
		t.Error("Exists = true, want false")
	}
}

func TestEnrollmentRepo_ProgressPersistedAndReloaded(t *testing.T) {
	students, courses, enrollments := newTestDB(t)
	ctx := context.Background()
	mustCreateStudent(t, students, "s1")
	mustCreateCourse(t, courses, "c1")

	e := &domain.Enrollment{ID: "e1", StudentID: "s1", CourseID: "c1", Status: domain.StatusPending, EnrolledAt: time.Now().UTC()}
	if err := enrollments.Create(ctx, e); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := enrollments.SetProgress(ctx, "e1", 0.75); err != nil {
		t.Fatalf("set progress: %v", err)
	}
	got, err := enrollments.GetByID(ctx, "e1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Progress != 0.75 {
		t.Errorf("progress = %v, want 0.75", got.Progress)
	}
	if len(got.Assessments) != 0 {
		t.Errorf("assessments = %v, want empty", got.Assessments)
	}
}

func TestBoolToInt(t *testing.T) {
	if boolToInt(true) != 1 {
		t.Errorf("boolToInt(true) = %d, want 1", boolToInt(true))
	}
	if boolToInt(false) != 0 {
		t.Errorf("boolToInt(false) = %d, want 0", boolToInt(false))
	}
}
