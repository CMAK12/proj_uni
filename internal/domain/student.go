package domain

import (
	"strings"
	"time"
)

// Student represents a learner enrolled in the institution.
type Student struct {
	ID        string
	Name      string
	Email     string
	CreatedAt time.Time
}

// Validate reports whether the student has the minimum required fields.
func (s *Student) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return wrap("name is required")
	}
	if !strings.Contains(s.Email, "@") {
		return wrap("valid email is required")
	}
	return nil
}

// wrap annotates ErrValidation with a human-readable reason.
func wrap(reason string) error {
	return &validationError{reason: reason}
}

type validationError struct {
	reason string
}

func (e *validationError) Error() string { return "validation failed: " + e.reason }

// Is lets errors.Is(err, ErrValidation) match any validation error.
func (e *validationError) Is(target error) bool { return target == ErrValidation }
