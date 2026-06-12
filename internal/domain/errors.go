// Package domain defines the core entities of the course-management system:
// students, courses and enrollments. It holds no business logic and depends on
// no other package, so every other layer may import it without creating cycles.
package domain

import "errors"

// Sentinel errors describe expected failure conditions. Callers compare with
// errors.Is rather than matching on strings.
var (
	// ErrNotFound is returned when a requested entity does not exist.
	ErrNotFound = errors.New("entity not found")

	// ErrDuplicate is returned when a unique constraint is violated
	// (e.g. a student e-mail or course code that already exists).
	ErrDuplicate = errors.New("entity already exists")

	// ErrValidation is returned when input fails validation.
	ErrValidation = errors.New("validation failed")

	// ErrAlreadyEnrolled is returned when a student is enrolled in a course
	// they are already enrolled in.
	ErrAlreadyEnrolled = errors.New("student already enrolled in course")
)
