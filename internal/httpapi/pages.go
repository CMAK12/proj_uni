package httpapi

import (
	"net/http"
	"strconv"

	"coursehub/internal/domain"
	"coursehub/internal/service"
)

// render executes a named template with data, reporting failures as 500s.
func (s *Server) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, name, data); err != nil {
		s.log.Error("render template", "name", name, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

// --- GET pages ---

func (s *Server) pageHome(w http.ResponseWriter, r *http.Request) {
	// "/" matches everything; reject anything but the root.
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	ctx := reqContext(r)
	students, err := s.students.List(ctx)
	if err != nil {
		s.writeErr(w, err)
		return
	}
	courses, err := s.courses.List(ctx)
	if err != nil {
		s.writeErr(w, err)
		return
	}
	credits := 0
	for _, c := range courses {
		credits += c.Credits()
	}
	s.render(w, "home.html", map[string]any{
		"Title":         "Панель керування",
		"Subtitle":      "Огляд системи керування курсами та студентами освітнього закладу.",
		"Active":        "home",
		"StudentCount":  len(students),
		"CourseCount":   len(courses),
		"Credits":       credits,
		"StrategyCount": 3,
	})
}

func (s *Server) pageStudents(w http.ResponseWriter, r *http.Request) {
	students, err := s.students.List(reqContext(r))
	if err != nil {
		s.writeErr(w, err)
		return
	}
	s.render(w, "students.html", map[string]any{
		"Title":    "Студенти",
		"Subtitle": "Реєстрація студентів освітнього закладу.",
		"Active":   "students",
		"Students": students,
	})
}

func (s *Server) pageCourses(w http.ResponseWriter, r *http.Request) {
	courses, err := s.courses.List(reqContext(r))
	if err != nil {
		s.writeErr(w, err)
		return
	}
	s.render(w, "courses.html", map[string]any{
		"Title":      "Курси",
		"Subtitle":   "Каталог курсів зі схемами оцінювання та розширеннями.",
		"Active":     "courses",
		"Courses":    courses,
		"Strategies": []string{"weighted", "passfail", "letter"},
	})
}

func (s *Server) pageProgress(w http.ResponseWriter, r *http.Request) {
	ctx := reqContext(r)
	students, err := s.students.List(ctx)
	if err != nil {
		s.writeErr(w, err)
		return
	}
	courses, err := s.courses.List(ctx)
	if err != nil {
		s.writeErr(w, err)
		return
	}

	data := map[string]any{
		"Title":    "Прогрес",
		"Subtitle": "Зарахування, оцінювання та відстеження прогресу студента.",
		"Active":   "progress",
		"Students": students,
		"Courses":  courses,
	}

	if studentID := r.URL.Query().Get("student"); studentID != "" {
		views, err := s.enrollments.StudentProgress(ctx, studentID)
		if err != nil {
			s.writeErr(w, err)
			return
		}
		data["Selected"] = studentID
		data["Views"] = views
	}
	s.render(w, "progress.html", data)
}

// --- POST form handlers (web UI) ---

func (s *Server) formCreateStudent(w http.ResponseWriter, r *http.Request) {
	if _, err := s.students.Register(reqContext(r), r.FormValue("name"), r.FormValue("email")); err != nil {
		s.writeErr(w, err)
		return
	}
	http.Redirect(w, r, "/students", http.StatusSeeOther)
}

func (s *Server) formCreateCourse(w http.ResponseWriter, r *http.Request) {
	credits, _ := strconv.Atoi(r.FormValue("credits"))
	var features []string
	if r.FormValue("certified") != "" {
		features = append(features, "certified")
	}
	_, err := s.courses.Create(reqContext(r), domain.CourseRecord{
		Code:     r.FormValue("code"),
		Title:    r.FormValue("title"),
		Credits:  credits,
		Type:     domain.CourseType(r.FormValue("type")),
		Grading:  r.FormValue("grading"),
		Features: features,
		Platform: r.FormValue("platform"),
	})
	if err != nil {
		s.writeErr(w, err)
		return
	}
	http.Redirect(w, r, "/courses", http.StatusSeeOther)
}

func (s *Server) formEnroll(w http.ResponseWriter, r *http.Request) {
	studentID := r.FormValue("student")
	if _, err := s.enrollments.Enroll(reqContext(r), studentID, r.FormValue("course")); err != nil {
		s.writeErr(w, err)
		return
	}
	http.Redirect(w, r, "/progress?student="+studentID, http.StatusSeeOther)
}

func (s *Server) formGrade(w http.ResponseWriter, r *http.Request) {
	score, _ := strconv.ParseFloat(r.FormValue("score"), 64)
	maxScore, _ := strconv.ParseFloat(r.FormValue("maxScore"), 64)
	weight, _ := strconv.ParseFloat(r.FormValue("weight"), 64)
	planned, _ := strconv.Atoi(r.FormValue("planned"))

	input := service.GradeInput{Name: r.FormValue("name"), Score: score, MaxScore: maxScore, Weight: weight}
	if _, err := s.enrollments.RecordGrade(reqContext(r), r.FormValue("enrollment"), []service.GradeInput{input}, planned); err != nil {
		s.writeErr(w, err)
		return
	}
	http.Redirect(w, r, "/progress?student="+r.FormValue("student"), http.StatusSeeOther)
}
