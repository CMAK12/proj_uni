package service

import (
	"context"
	"errors"
	"testing"

	"coursehub/internal/domain"
	"coursehub/internal/grading"
	"coursehub/internal/progress"
)

// --- fakes ---

type fakeStudentRepo struct {
	byID      map[string]*domain.Student
	createErr error
	listErr   error
}

func newFakeStudentRepo() *fakeStudentRepo {
	return &fakeStudentRepo{byID: map[string]*domain.Student{}}
}

func (r *fakeStudentRepo) Create(_ context.Context, s *domain.Student) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.byID[s.ID] = s
	return nil
}

func (r *fakeStudentRepo) GetByID(_ context.Context, id string) (*domain.Student, error) {
	s, ok := r.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return s, nil
}

func (r *fakeStudentRepo) List(_ context.Context) ([]*domain.Student, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	out := make([]*domain.Student, 0, len(r.byID))
	for _, s := range r.byID {
		out = append(out, s)
	}
	return out, nil
}

type fakeCourseRepo struct {
	recs      map[string]domain.CourseRecord
	order     []string
	createErr error
	listErr   error
}

func newFakeCourseRepo() *fakeCourseRepo {
	return &fakeCourseRepo{recs: map[string]domain.CourseRecord{}}
}

func (r *fakeCourseRepo) Create(_ context.Context, rec domain.CourseRecord) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.recs[rec.ID] = rec
	r.order = append(r.order, rec.ID)
	return nil
}

func (r *fakeCourseRepo) GetByID(_ context.Context, id string) (domain.CourseRecord, error) {
	rec, ok := r.recs[id]
	if !ok {
		return domain.CourseRecord{}, domain.ErrNotFound
	}
	return rec, nil
}

func (r *fakeCourseRepo) List(_ context.Context) ([]domain.CourseRecord, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	out := make([]domain.CourseRecord, 0, len(r.order))
	for _, id := range r.order {
		out = append(out, r.recs[id])
	}
	return out, nil
}

// fakeNotifier implements the notifier port and can be told to fail.
type fakeNotifier struct {
	err   error
	calls int
}

func (n *fakeNotifier) Notify(context.Context, progress.Event) error {
	n.calls++
	return n.err
}

// stubCourse is a minimal domain.Course returning an arbitrary grading name,
// used to exercise the unknown-strategy path in RecordGrade.
type stubCourse struct{ grading string }

func (stubCourse) ID() string                { return "stub" }
func (stubCourse) Code() string              { return "STUB" }
func (stubCourse) Title() string             { return "Stub" }
func (stubCourse) Credits() int              { return 0 }
func (s stubCourse) GradingStrategy() string { return s.grading }
func (stubCourse) Features() []string        { return nil }
func (stubCourse) Describe() string          { return "stub" }

// --- StudentService ---

func TestStudentService_Register(t *testing.T) {
	tests := []struct {
		name      string
		sName     string
		email     string
		createErr error
		wantErr   error
	}{
		{"valid", "Іван", "ivan@lnu.ua", nil, nil},
		{"empty name", "", "x@y.z", nil, domain.ErrValidation},
		{"bad email", "Bob", "no-at", nil, domain.ErrValidation},
		{"duplicate from repo", "Bob", "bob@x.io", domain.ErrDuplicate, domain.ErrDuplicate},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := newFakeStudentRepo()
			repo.createErr = tt.createErr
			svc := NewStudentService(repo)
			st, err := svc.Register(context.Background(), tt.sName, tt.email)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if st.ID == "" {
				t.Error("expected generated ID")
			}
			if st.CreatedAt.IsZero() {
				t.Error("expected CreatedAt to be set")
			}
		})
	}
}

func TestStudentService_GetAndList(t *testing.T) {
	repo := newFakeStudentRepo()
	svc := NewStudentService(repo)
	ctx := context.Background()

	st, err := svc.Register(ctx, "Bob", "bob@x.io")
	if err != nil {
		t.Fatal(err)
	}

	got, err := svc.Get(ctx, st.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != st.ID {
		t.Errorf("Get id = %q, want %q", got.ID, st.ID)
	}

	if _, err := svc.Get(ctx, "missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Get(missing) = %v, want ErrNotFound", err)
	}

	list, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("List len = %d, want 1", len(list))
	}
}

func TestStudentService_List_RepoError(t *testing.T) {
	repo := newFakeStudentRepo()
	repo.listErr = errors.New("db down")
	svc := NewStudentService(repo)
	if _, err := svc.List(context.Background()); err == nil {
		t.Fatal("expected error from repo")
	}
}

// --- CourseService ---

func TestCourseService_Create(t *testing.T) {
	tests := []struct {
		name      string
		rec       domain.CourseRecord
		createErr error
		wantErr   error
	}{
		{"valid", domain.CourseRecord{Code: "CS1", Title: "Intro", Grading: grading.StrategyWeighted}, nil, nil},
		{"invalid title", domain.CourseRecord{Code: "CS1", Grading: grading.StrategyWeighted}, nil, domain.ErrValidation},
		{"invalid grading", domain.CourseRecord{Code: "CS1", Title: "Intro", Grading: "nope"}, nil, domain.ErrValidation},
		{"repo error", domain.CourseRecord{Code: "CS1", Title: "Intro", Grading: grading.StrategyWeighted}, errors.New("boom"), errors.New("boom")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := newFakeCourseRepo()
			repo.createErr = tt.createErr
			svc := NewCourseService(repo)
			c, err := svc.Create(context.Background(), tt.rec)
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}
				if errors.Is(tt.wantErr, domain.ErrValidation) && !errors.Is(err, domain.ErrValidation) {
					t.Fatalf("err = %v, want ErrValidation", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if c.ID() == "" {
				t.Error("expected generated ID")
			}
		})
	}
}

func TestCourseService_GetAndList(t *testing.T) {
	repo := newFakeCourseRepo()
	svc := NewCourseService(repo)
	ctx := context.Background()

	c1, err := svc.Create(ctx, domain.CourseRecord{Code: "CS1", Title: "Algo", Grading: grading.StrategyWeighted})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(ctx, domain.CourseRecord{Code: "CS2", Title: "Web", Type: domain.CourseTypeOnline, Grading: grading.StrategyLetter}); err != nil {
		t.Fatal(err)
	}

	got, err := svc.Get(ctx, c1.ID())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Code() != "CS1" {
		t.Errorf("Get code = %q", got.Code())
	}

	if _, err := svc.Get(ctx, "missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Get(missing) = %v, want ErrNotFound", err)
	}

	list, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("List len = %d, want 2", len(list))
	}
}

func TestCourseService_List_Errors(t *testing.T) {
	t.Run("repo error", func(t *testing.T) {
		repo := newFakeCourseRepo()
		repo.listErr = errors.New("db down")
		if _, err := NewCourseService(repo).List(context.Background()); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("rebuild failure on bad record", func(t *testing.T) {
		repo := newFakeCourseRepo()
		// Inject a record the factory cannot rebuild (unknown grading).
		repo.recs["bad"] = domain.CourseRecord{ID: "bad", Code: "X", Title: "T", Grading: "bogus"}
		repo.order = []string{"bad"}
		if _, err := NewCourseService(repo).List(context.Background()); err == nil {
			t.Fatal("expected rebuild error")
		}
	})
}

// --- EnrollmentService extra paths ---

func TestEnrollmentService_Enroll_CourseNotFound(t *testing.T) {
	repo := newFakeEnrollmentRepo()
	pub := progress.NewPublisher(progress.NewProgressTracker(repo))
	svc := NewEnrollmentService(repo, fakeCourses{c: nil}, pub)
	_, err := svc.Enroll(context.Background(), "s1", "missing")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestEnrollmentService_RecordGrade_EnrollmentNotFound(t *testing.T) {
	svc, _ := newEnrollmentSvc(t, grading.StrategyWeighted)
	_, err := svc.RecordGrade(context.Background(), "missing", []GradeInput{{Name: "x", Score: 1, MaxScore: 1, Weight: 1}}, 1)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestEnrollmentService_RecordGrade_UnknownStrategy(t *testing.T) {
	repo := newFakeEnrollmentRepo()
	repo.byID["e1"] = &domain.Enrollment{ID: "e1", StudentID: "s1", CourseID: "c1"}
	pub := progress.NewPublisher(progress.NewProgressTracker(repo))
	svc := NewEnrollmentService(repo, fakeCourses{c: stubCourse{grading: "bogus"}}, pub)

	_, err := svc.RecordGrade(context.Background(), "e1", []GradeInput{{Name: "x", Score: 1, MaxScore: 1, Weight: 1}}, 1)
	if err == nil {
		t.Fatal("expected error for unknown strategy")
	}
}

func TestEnrollmentService_RecordGrade_NotifyError(t *testing.T) {
	repo := newFakeEnrollmentRepo()
	repo.byID["e1"] = &domain.Enrollment{ID: "e1", StudentID: "s1", CourseID: "c1"}
	notifier := &fakeNotifier{err: errors.New("observer boom")}
	svc := NewEnrollmentService(repo, fakeCourses{c: buildCourse(t, grading.StrategyWeighted)}, notifier)

	_, err := svc.RecordGrade(context.Background(), "e1", []GradeInput{{Name: "x", Score: 80, MaxScore: 100, Weight: 1}}, 1)
	if err == nil {
		t.Fatal("expected notify error")
	}
	if notifier.calls != 1 {
		t.Errorf("notifier calls = %d, want 1", notifier.calls)
	}
}

func TestEnrollmentService_StudentProgress(t *testing.T) {
	repo := newFakeEnrollmentRepo()
	repo.byID["e1"] = &domain.Enrollment{ID: "e1", StudentID: "s1", CourseID: "c1", Status: domain.StatusActive}
	repo.byID["e2"] = &domain.Enrollment{ID: "e2", StudentID: "s1", CourseID: "c1", Status: domain.StatusPending}
	repo.byID["e3"] = &domain.Enrollment{ID: "e3", StudentID: "other", CourseID: "c1"}

	pub := progress.NewPublisher(progress.NewProgressTracker(repo))
	svc := NewEnrollmentService(repo, fakeCourses{c: buildCourse(t, grading.StrategyWeighted)}, pub)

	views, err := svc.StudentProgress(context.Background(), "s1")
	if err != nil {
		t.Fatalf("StudentProgress: %v", err)
	}
	if len(views) != 2 {
		t.Fatalf("views len = %d, want 2", len(views))
	}
	for _, v := range views {
		if v.CourseCode != "CS101" || v.CourseTitle != "Algo" {
			t.Errorf("view course = {%q %q}, want {CS101 Algo}", v.CourseCode, v.CourseTitle)
		}
	}
}

func TestEnrollmentService_StudentProgress_CourseLookupFails(t *testing.T) {
	repo := newFakeEnrollmentRepo()
	repo.byID["e1"] = &domain.Enrollment{ID: "e1", StudentID: "s1", CourseID: "c1"}
	pub := progress.NewPublisher(progress.NewProgressTracker(repo))
	// nil course => lookup returns ErrNotFound, but the row is still included.
	svc := NewEnrollmentService(repo, fakeCourses{c: nil}, pub)

	views, err := svc.StudentProgress(context.Background(), "s1")
	if err != nil {
		t.Fatalf("StudentProgress: %v", err)
	}
	if len(views) != 1 || views[0].CourseCode != "" {
		t.Errorf("views = %+v, want one row with empty course code", views)
	}
}

func TestNewID_UniqueAndHex(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		id := newID()
		if len(id) != 32 {
			t.Fatalf("id length = %d, want 32", len(id))
		}
		if seen[id] {
			t.Fatalf("duplicate id %q", id)
		}
		seen[id] = true
	}
}
