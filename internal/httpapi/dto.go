package httpapi

import (
	"coursehub/internal/domain"
	"coursehub/internal/service"
)

// studentDTO is the JSON shape of a student.
type studentDTO struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func toStudentDTO(s *domain.Student) studentDTO {
	return studentDTO{ID: s.ID, Name: s.Name, Email: s.Email}
}

// courseDTO flattens a decorated domain.Course for JSON output. Features and
// Description reflect any decorators applied (Decorator pattern).
type courseDTO struct {
	ID          string   `json:"id"`
	Code        string   `json:"code"`
	Title       string   `json:"title"`
	Credits     int      `json:"credits"`
	Grading     string   `json:"grading"`
	Features    []string `json:"features"`
	Description string   `json:"description"`
}

func toCourseDTO(c domain.Course) courseDTO {
	feats := c.Features()
	if feats == nil {
		feats = []string{}
	}
	return courseDTO{
		ID:          c.ID(),
		Code:        c.Code(),
		Title:       c.Title(),
		Credits:     c.Credits(),
		Grading:     c.GradingStrategy(),
		Features:    feats,
		Description: c.Describe(),
	}
}

// enrollmentDTO is the JSON shape of an enrollment.
type enrollmentDTO struct {
	ID         string  `json:"id"`
	StudentID  string  `json:"studentId"`
	CourseID   string  `json:"courseId"`
	Status     string  `json:"status"`
	FinalGrade float64 `json:"finalGrade"`
	Letter     string  `json:"letter"`
	Passed     bool    `json:"passed"`
	Progress   float64 `json:"progress"`
}

func toEnrollmentDTO(e *domain.Enrollment) enrollmentDTO {
	return enrollmentDTO{
		ID:         e.ID,
		StudentID:  e.StudentID,
		CourseID:   e.CourseID,
		Status:     string(e.Status),
		FinalGrade: e.FinalGrade,
		Letter:     e.Letter,
		Passed:     e.Passed,
		Progress:   e.Progress,
	}
}

// progressDTO is the JSON shape of one progress row.
type progressDTO struct {
	CourseCode  string  `json:"courseCode"`
	CourseTitle string  `json:"courseTitle"`
	Status      string  `json:"status"`
	FinalGrade  float64 `json:"finalGrade"`
	Letter      string  `json:"letter"`
	Progress    float64 `json:"progress"`
}

func toProgressDTO(v service.ProgressView) progressDTO {
	return progressDTO{
		CourseCode:  v.CourseCode,
		CourseTitle: v.CourseTitle,
		Status:      string(v.Enrollment.Status),
		FinalGrade:  v.Enrollment.FinalGrade,
		Letter:      v.Enrollment.Letter,
		Progress:    v.Enrollment.Progress,
	}
}
