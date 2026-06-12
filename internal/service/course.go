package service

import (
	"context"
	"fmt"
	"time"

	"coursehub/internal/course"
	"coursehub/internal/domain"
)

// CourseService creates and reads courses, delegating construction of the
// decorated domain.Course to the course.Factory.
type CourseService struct {
	repo    CourseRepository
	factory course.Factory
	now     func() time.Time
}

// NewCourseService returns a CourseService backed by repo.
func NewCourseService(repo CourseRepository) *CourseService {
	return &CourseService{repo: repo, factory: course.NewFactory(), now: time.Now}
}

// Create validates the spec (by building it through the factory), persists the
// flat record and returns the assembled domain.Course.
func (s *CourseService) Create(ctx context.Context, rec domain.CourseRecord) (domain.Course, error) {
	rec.ID = newID()
	rec.CreatedAt = s.now().UTC()

	c, err := s.factory.Build(rec)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, rec); err != nil {
		return nil, fmt.Errorf("create course: %w", err)
	}
	return c, nil
}

// Get loads a course record and rebuilds its decorated form.
func (s *CourseService) Get(ctx context.Context, id string) (domain.Course, error) {
	rec, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.factory.Build(rec)
}

// List returns every course in decorated form.
func (s *CourseService) List(ctx context.Context) ([]domain.Course, error) {
	recs, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	courses := make([]domain.Course, 0, len(recs))
	for _, rec := range recs {
		c, err := s.factory.Build(rec)
		if err != nil {
			return nil, fmt.Errorf("rebuild course %s: %w", rec.ID, err)
		}
		courses = append(courses, c)
	}
	return courses, nil
}
