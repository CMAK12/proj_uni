package course

import (
	"fmt"

	"coursehub/internal/domain"
	"coursehub/internal/grading"
)

// Factory builds domain.Course values from flat records. It is the single place
// that knows how to assemble a base course, attach the requested decorators and
// validate the grading strategy — centralising construction so callers (HTTP
// handlers, repositories) never wire decorators by hand.
type Factory struct{}

// NewFactory returns a Factory.
func NewFactory() Factory { return Factory{} }

// Build turns a CourseRecord into a fully decorated domain.Course. It is used
// both when a course is first created and when one is rehydrated from storage,
// guaranteeing the in-memory object and the persisted record stay consistent.
func (Factory) Build(rec domain.CourseRecord) (domain.Course, error) {
	if rec.Code == "" || rec.Title == "" {
		return nil, fmt.Errorf("%w: course code and title are required", domain.ErrValidation)
	}
	if _, err := grading.Get(rec.Grading); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrValidation, err)
	}

	var c domain.Course = &base{
		id:       rec.ID,
		code:     rec.Code,
		title:    rec.Title,
		credits:  rec.Credits,
		grading:  rec.Grading,
		features: nil,
	}

	// Apply decorators. Order is deterministic so Describe()/Features() are
	// stable across rebuilds. An online type implies the online decorator even
	// if "online" is not explicitly in Features.
	if rec.Type == domain.CourseTypeOnline || hasFeature(rec.Features, FeatureOnline) {
		c = OnlineCourse{Course: c, Platform: rec.Platform}
	}
	if hasFeature(rec.Features, FeatureCertified) {
		c = CertifiedCourse{Course: c}
	}

	return c, nil
}
