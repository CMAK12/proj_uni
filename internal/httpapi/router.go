package httpapi

import "net/http"

// Routes returns the HTTP handler with all API and web routes registered. The
// Go 1.22+ ServeMux matches on method and path pattern, so no third-party
// router is needed.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// JSON REST API.
	mux.HandleFunc("POST /api/students", s.handleCreateStudent)
	mux.HandleFunc("GET /api/students", s.handleListStudents)
	mux.HandleFunc("GET /api/students/{id}", s.handleGetStudent)
	mux.HandleFunc("GET /api/students/{id}/progress", s.handleStudentProgress)

	mux.HandleFunc("POST /api/courses", s.handleCreateCourse)
	mux.HandleFunc("GET /api/courses", s.handleListCourses)

	mux.HandleFunc("POST /api/enrollments", s.handleCreateEnrollment)
	mux.HandleFunc("PUT /api/enrollments/{id}/grade", s.handleGrade)

	// Web UI (server-rendered HTML).
	mux.HandleFunc("GET /students", s.pageStudents)
	mux.HandleFunc("POST /students", s.formCreateStudent)
	mux.HandleFunc("GET /courses", s.pageCourses)
	mux.HandleFunc("POST /courses", s.formCreateCourse)
	mux.HandleFunc("GET /progress", s.pageProgress)
	mux.HandleFunc("POST /enroll", s.formEnroll)
	mux.HandleFunc("POST /grade", s.formGrade)
	mux.HandleFunc("GET /", s.pageHome)

	return s.withLogging(mux)
}

// withLogging is simple request middleware.
func (s *Server) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.log.Info("request", "method", r.Method, "path", r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
