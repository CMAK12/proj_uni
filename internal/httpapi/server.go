// Package httpapi exposes the application over HTTP: a JSON REST API under
// /api and a small server-rendered web UI. It depends on the concrete service
// types and translates between HTTP and the service layer; it contains no
// business logic and no storage detail.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"log/slog"
	"net/http"

	"coursehub/internal/domain"
	"coursehub/internal/service"
)

// Server holds the application services and parsed templates.
type Server struct {
	students    *service.StudentService
	courses     *service.CourseService
	enrollments *service.EnrollmentService
	log         *slog.Logger
	tmpl        *template.Template
}

// NewServer constructs a Server. It panics if the embedded templates fail to
// parse, since that is a programming error detectable at startup.
func NewServer(
	students *service.StudentService,
	courses *service.CourseService,
	enrollments *service.EnrollmentService,
	log *slog.Logger,
) *Server {
	if log == nil {
		log = slog.Default()
	}
	tmpl := template.Must(template.New("").Funcs(templateFuncs).ParseFS(templateFS, "templates/*.html"))
	return &Server{students: students, courses: courses, enrollments: enrollments, log: log, tmpl: tmpl}
}

// --- JSON helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

// writeErr maps a domain error to an HTTP status and a JSON error body.
func (s *Server) writeErr(w http.ResponseWriter, err error) {
	status := statusFor(err)
	if status == http.StatusInternalServerError {
		s.log.Error("request failed", "err", err)
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

// statusFor classifies a domain error into an HTTP status code.
func statusFor(err error) int {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrValidation):
		return http.StatusBadRequest
	case errors.Is(err, domain.ErrDuplicate), errors.Is(err, domain.ErrAlreadyEnrolled):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

// decodeJSON reads a JSON request body into v.
func decodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return domain.ErrValidation
	}
	return nil
}

// reqContext is a tiny helper so handlers read consistently.
func reqContext(r *http.Request) context.Context { return r.Context() }
