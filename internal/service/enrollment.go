package service

import (
	"context"
	"fmt"
	"time"

	"coursehub/internal/domain"
	"coursehub/internal/grading"
	"coursehub/internal/progress"
)

// courseLookup is the slice of CourseService that EnrollmentService needs.
// Declaring it here (consumer side) keeps the dependency minimal and testable.
type courseLookup interface {
	Get(ctx context.Context, id string) (domain.Course, error)
}

// notifier publishes grading events to observers (the Subject side of Observer).
type notifier interface {
	Notify(ctx context.Context, e progress.Event) error
}

// EnrollmentService registers students on courses and records grades. Recording
// a grade is where the patterns meet: the course's grading Strategy evaluates
// the assessments, the enrollment is updated, and the result is published to
// Observers that track progress and send notifications.
type EnrollmentService struct {
	repo      EnrollmentRepository
	courses   courseLookup
	publisher notifier
	now       func() time.Time
}

// NewEnrollmentService wires the enrollment repository, a course lookup and the
// event publisher.
func NewEnrollmentService(repo EnrollmentRepository, courses courseLookup, publisher notifier) *EnrollmentService {
	return &EnrollmentService{repo: repo, courses: courses, publisher: publisher, now: time.Now}
}

// Enroll registers a student on a course. It rejects duplicate enrollments and
// verifies the course exists.
func (s *EnrollmentService) Enroll(ctx context.Context, studentID, courseID string) (*domain.Enrollment, error) {
	if _, err := s.courses.Get(ctx, courseID); err != nil {
		return nil, fmt.Errorf("enroll: %w", err)
	}
	exists, err := s.repo.Exists(ctx, studentID, courseID)
	if err != nil {
		return nil, fmt.Errorf("enroll: %w", err)
	}
	if exists {
		return nil, domain.ErrAlreadyEnrolled
	}

	e := &domain.Enrollment{
		ID:         newID(),
		StudentID:  studentID,
		CourseID:   courseID,
		Status:     domain.StatusPending,
		EnrolledAt: s.now().UTC(),
	}
	if err := s.repo.Create(ctx, e); err != nil {
		return nil, fmt.Errorf("enroll: %w", err)
	}
	return e, nil
}

// GradeInput is one assessment submitted for grading.
type GradeInput struct {
	Name     string
	Score    float64
	MaxScore float64
	Weight   float64
}

// RecordGrade evaluates the submitted assessments with the course's strategy,
// updates the enrollment, and notifies observers. planned is the total number
// of assessments expected for the course (used for progress); when <= 0 it
// defaults to the number submitted, treating this call as the full set.
func (s *EnrollmentService) RecordGrade(ctx context.Context, enrollmentID string, inputs []GradeInput, planned int) (*domain.Enrollment, error) {
	if len(inputs) == 0 {
		return nil, fmt.Errorf("%w: at least one assessment is required", domain.ErrValidation)
	}

	e, err := s.repo.GetByID(ctx, enrollmentID)
	if err != nil {
		return nil, err
	}
	c, err := s.courses.Get(ctx, e.CourseID)
	if err != nil {
		return nil, fmt.Errorf("record grade: %w", err)
	}
	strategy, err := grading.Get(c.GradingStrategy())
	if err != nil {
		return nil, fmt.Errorf("record grade: %w", err)
	}

	// Strategy pattern: evaluate without knowing the concrete algorithm.
	components := toComponents(inputs)
	result := strategy.Evaluate(components)

	graded := len(inputs)
	if planned <= 0 {
		planned = graded
	}

	e.Assessments = toAssessments(inputs)
	e.FinalGrade = result.Final
	e.Letter = result.Letter
	e.Passed = result.Passed
	if graded >= planned {
		e.Status = domain.StatusCompleted
	} else {
		e.Status = domain.StatusActive
	}
	if err := s.repo.Update(ctx, e); err != nil {
		return nil, fmt.Errorf("record grade: %w", err)
	}

	// Observer pattern: publish the event. The ProgressTracker persists the
	// progress fraction; the notifier logs. Failures are surfaced but the grade
	// is already saved.
	event := progress.Event{
		EnrollmentID: e.ID,
		StudentID:    e.StudentID,
		CourseID:     e.CourseID,
		Result:       result,
		GradedCount:  graded,
		PlannedCount: planned,
		OccurredAt:   s.now().UTC(),
	}
	if err := s.publisher.Notify(ctx, event); err != nil {
		return nil, fmt.Errorf("record grade: notify observers: %w", err)
	}

	// Reload so the returned enrollment reflects the progress the tracker wrote.
	return s.repo.GetByID(ctx, enrollmentID)
}

// ProgressView pairs an enrollment with its course's display fields.
type ProgressView struct {
	Enrollment  *domain.Enrollment
	CourseCode  string
	CourseTitle string
}

// StudentProgress returns every enrollment for a student with course context,
// the read model behind the progress page and the /progress endpoint.
func (s *EnrollmentService) StudentProgress(ctx context.Context, studentID string) ([]ProgressView, error) {
	enrollments, err := s.repo.ListByStudent(ctx, studentID)
	if err != nil {
		return nil, err
	}
	views := make([]ProgressView, 0, len(enrollments))
	for _, e := range enrollments {
		view := ProgressView{Enrollment: e}
		if c, err := s.courses.Get(ctx, e.CourseID); err == nil {
			view.CourseCode = c.Code()
			view.CourseTitle = c.Title()
		}
		views = append(views, view)
	}
	return views, nil
}

func toComponents(inputs []GradeInput) []grading.Component {
	out := make([]grading.Component, len(inputs))
	for i, in := range inputs {
		out[i] = grading.Component{
			Name:     in.Name,
			Score:    in.Score,
			MaxScore: in.MaxScore,
			Weight:   in.Weight,
		}
	}
	return out
}

func toAssessments(inputs []GradeInput) []domain.Assessment {
	out := make([]domain.Assessment, len(inputs))
	for i, in := range inputs {
		out[i] = domain.Assessment{
			ID:       newID(),
			Name:     in.Name,
			Score:    in.Score,
			MaxScore: in.MaxScore,
			Weight:   in.Weight,
		}
	}
	return out
}
