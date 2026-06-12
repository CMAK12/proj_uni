package service

import (
	"context"
	"fmt"
	"time"

	"coursehub/internal/domain"
)

// StudentService manages student records.
type StudentService struct {
	repo StudentRepository
	now  func() time.Time
}

// NewStudentService returns a StudentService backed by repo.
func NewStudentService(repo StudentRepository) *StudentService {
	return &StudentService{repo: repo, now: time.Now}
}

// Register validates and stores a new student.
func (s *StudentService) Register(ctx context.Context, name, email string) (*domain.Student, error) {
	st := &domain.Student{
		ID:        newID(),
		Name:      name,
		Email:     email,
		CreatedAt: s.now().UTC(),
	}
	if err := st.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, st); err != nil {
		return nil, fmt.Errorf("register student: %w", err)
	}
	return st, nil
}

// Get returns a student by ID.
func (s *StudentService) Get(ctx context.Context, id string) (*domain.Student, error) {
	return s.repo.GetByID(ctx, id)
}

// List returns all students.
func (s *StudentService) List(ctx context.Context) ([]*domain.Student, error) {
	return s.repo.List(ctx)
}
