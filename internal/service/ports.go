// Package service holds the application's business logic. It orchestrates the
// design patterns (Strategy, Observer, Factory, Decorator) but owns no storage
// detail: it depends only on the repository interfaces declared below. The
// concrete SQLite implementations live in package storage and are injected in
// main, giving the dependency-inversion that makes the storage backend
// swappable (the Repository pattern).
package service

import (
	"context"

	"coursehub/internal/domain"
)

// StudentRepository persists students.
type StudentRepository interface {
	Create(ctx context.Context, s *domain.Student) error
	GetByID(ctx context.Context, id string) (*domain.Student, error)
	List(ctx context.Context) ([]*domain.Student, error)
}

// CourseRepository persists courses as flat records; the service rebuilds the
// decorated domain.Course via the factory.
type CourseRepository interface {
	Create(ctx context.Context, rec domain.CourseRecord) error
	GetByID(ctx context.Context, id string) (domain.CourseRecord, error)
	List(ctx context.Context) ([]domain.CourseRecord, error)
}

// EnrollmentRepository persists enrollments and their assessments.
type EnrollmentRepository interface {
	Create(ctx context.Context, e *domain.Enrollment) error
	GetByID(ctx context.Context, id string) (*domain.Enrollment, error)
	ListByStudent(ctx context.Context, studentID string) ([]*domain.Enrollment, error)
	Exists(ctx context.Context, studentID, courseID string) (bool, error)
	// Update saves status, grade fields and assessments.
	Update(ctx context.Context, e *domain.Enrollment) error
	// SetProgress satisfies progress.Store so the ProgressTracker observer can
	// persist computed progress independently of Update.
	SetProgress(ctx context.Context, enrollmentID string, progress float64) error
}
