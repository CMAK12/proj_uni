package httpapi

import (
	"net/http"

	"coursehub/internal/domain"
	"coursehub/internal/service"
)

// --- Students ---

type createStudentRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (s *Server) handleCreateStudent(w http.ResponseWriter, r *http.Request) {
	var req createStudentRequest
	if err := decodeJSON(r, &req); err != nil {
		s.writeErr(w, err)
		return
	}
	st, err := s.students.Register(reqContext(r), req.Name, req.Email)
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toStudentDTO(st))
}

func (s *Server) handleListStudents(w http.ResponseWriter, r *http.Request) {
	students, err := s.students.List(reqContext(r))
	if err != nil {
		s.writeErr(w, err)
		return
	}
	out := make([]studentDTO, 0, len(students))
	for _, st := range students {
		out = append(out, toStudentDTO(st))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetStudent(w http.ResponseWriter, r *http.Request) {
	st, err := s.students.Get(reqContext(r), r.PathValue("id"))
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toStudentDTO(st))
}

func (s *Server) handleStudentProgress(w http.ResponseWriter, r *http.Request) {
	views, err := s.enrollments.StudentProgress(reqContext(r), r.PathValue("id"))
	if err != nil {
		s.writeErr(w, err)
		return
	}
	out := make([]progressDTO, 0, len(views))
	for _, v := range views {
		out = append(out, toProgressDTO(v))
	}
	writeJSON(w, http.StatusOK, out)
}

// --- Courses ---

type createCourseRequest struct {
	Code     string   `json:"code"`
	Title    string   `json:"title"`
	Credits  int      `json:"credits"`
	Type     string   `json:"type"`
	Grading  string   `json:"grading"`
	Features []string `json:"features"`
	Platform string   `json:"platform"`
}

func (s *Server) handleCreateCourse(w http.ResponseWriter, r *http.Request) {
	var req createCourseRequest
	if err := decodeJSON(r, &req); err != nil {
		s.writeErr(w, err)
		return
	}
	c, err := s.courses.Create(reqContext(r), domain.CourseRecord{
		Code:     req.Code,
		Title:    req.Title,
		Credits:  req.Credits,
		Type:     domain.CourseType(req.Type),
		Grading:  req.Grading,
		Features: req.Features,
		Platform: req.Platform,
	})
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toCourseDTO(c))
}

func (s *Server) handleListCourses(w http.ResponseWriter, r *http.Request) {
	courses, err := s.courses.List(reqContext(r))
	if err != nil {
		s.writeErr(w, err)
		return
	}
	out := make([]courseDTO, 0, len(courses))
	for _, c := range courses {
		out = append(out, toCourseDTO(c))
	}
	writeJSON(w, http.StatusOK, out)
}

// --- Enrollments ---

type createEnrollmentRequest struct {
	StudentID string `json:"studentId"`
	CourseID  string `json:"courseId"`
}

func (s *Server) handleCreateEnrollment(w http.ResponseWriter, r *http.Request) {
	var req createEnrollmentRequest
	if err := decodeJSON(r, &req); err != nil {
		s.writeErr(w, err)
		return
	}
	e, err := s.enrollments.Enroll(reqContext(r), req.StudentID, req.CourseID)
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toEnrollmentDTO(e))
}

type gradeRequest struct {
	Components []struct {
		Name     string  `json:"name"`
		Score    float64 `json:"score"`
		MaxScore float64 `json:"maxScore"`
		Weight   float64 `json:"weight"`
	} `json:"components"`
	Planned int `json:"planned"`
}

func (s *Server) handleGrade(w http.ResponseWriter, r *http.Request) {
	var req gradeRequest
	if err := decodeJSON(r, &req); err != nil {
		s.writeErr(w, err)
		return
	}
	inputs := make([]service.GradeInput, len(req.Components))
	for i, c := range req.Components {
		inputs[i] = service.GradeInput{Name: c.Name, Score: c.Score, MaxScore: c.MaxScore, Weight: c.Weight}
	}
	e, err := s.enrollments.RecordGrade(reqContext(r), r.PathValue("id"), inputs, req.Planned)
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toEnrollmentDTO(e))
}
