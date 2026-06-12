package domain

import "time"

// EnrollmentStatus is the lifecycle state of an enrollment.
type EnrollmentStatus string

const (
	// StatusPending means the student is registered but the course has not started.
	StatusPending EnrollmentStatus = "pending"
	// StatusActive means the course is in progress.
	StatusActive EnrollmentStatus = "active"
	// StatusCompleted means the course finished and a final grade was recorded.
	StatusCompleted EnrollmentStatus = "completed"
	// StatusDropped means the student withdrew.
	StatusDropped EnrollmentStatus = "dropped"
)

// Assessment is a single graded component of an enrollment (exam, lab, project).
type Assessment struct {
	ID       string
	Name     string
	Score    float64 // points earned
	MaxScore float64 // points possible
	Weight   float64 // relative weight, used by weighted strategies
}

// Enrollment links a student to a course and accumulates their assessments,
// progress and final result.
type Enrollment struct {
	ID          string
	StudentID   string
	CourseID    string
	Status      EnrollmentStatus
	FinalGrade  float64
	Letter      string
	Passed      bool
	Progress    float64 // 0..1, maintained by the progress.Tracker observer
	EnrolledAt  time.Time
	Assessments []Assessment
}
