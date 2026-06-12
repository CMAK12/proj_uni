package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"coursehub/internal/domain"
	"coursehub/internal/progress"
	"coursehub/internal/service"
	"coursehub/internal/storage"
)

// newTestServer wires the full stack against an in-memory database and returns
// an httptest.Server plus a no-redirect client.
func newTestServer(t *testing.T) (*httptest.Server, *http.Client) {
	t.Helper()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	studentRepo := storage.NewStudentRepo(db)
	courseRepo := storage.NewCourseRepo(db)
	enrollmentRepo := storage.NewEnrollmentRepo(db)

	pub := progress.NewPublisher(progress.NewProgressTracker(enrollmentRepo))
	students := service.NewStudentService(studentRepo)
	courses := service.NewCourseService(courseRepo)
	enrollments := service.NewEnrollmentService(enrollmentRepo, courses, pub)

	srv := NewServer(students, courses, enrollments, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Routes())
	t.Cleanup(ts.Close)

	client := ts.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return ts, client
}

func doJSON(t *testing.T, client *http.Client, method, urlStr, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, urlStr, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func decodeBody(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode body: %v", err)
	}
}

func TestAPI_Students(t *testing.T) {
	ts, client := newTestServer(t)

	t.Run("create valid", func(t *testing.T) {
		resp := doJSON(t, client, http.MethodPost, ts.URL+"/api/students", `{"name":"Іван","email":"ivan@lnu.ua"}`)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("status = %d, want 201", resp.StatusCode)
		}
		var dto studentDTO
		decodeBody(t, resp, &dto)
		if dto.ID == "" || dto.Email != "ivan@lnu.ua" {
			t.Errorf("dto = %+v", dto)
		}
	})

	t.Run("create invalid email", func(t *testing.T) {
		resp := doJSON(t, client, http.MethodPost, ts.URL+"/api/students", `{"name":"Bob","email":"no-at"}`)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", resp.StatusCode)
		}
		resp.Body.Close()
	})

	t.Run("create malformed json", func(t *testing.T) {
		resp := doJSON(t, client, http.MethodPost, ts.URL+"/api/students", `{not json}`)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", resp.StatusCode)
		}
		resp.Body.Close()
	})

	t.Run("create duplicate email", func(t *testing.T) {
		_ = doJSON(t, client, http.MethodPost, ts.URL+"/api/students", `{"name":"A","email":"dup@lnu.ua"}`).Body.Close()
		resp := doJSON(t, client, http.MethodPost, ts.URL+"/api/students", `{"name":"B","email":"dup@lnu.ua"}`)
		if resp.StatusCode != http.StatusConflict {
			t.Errorf("status = %d, want 409", resp.StatusCode)
		}
		resp.Body.Close()
	})

	t.Run("list", func(t *testing.T) {
		resp := doJSON(t, client, http.MethodGet, ts.URL+"/api/students", "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		var list []studentDTO
		decodeBody(t, resp, &list)
		if len(list) == 0 {
			t.Error("expected at least one student")
		}
	})

	t.Run("get not found", func(t *testing.T) {
		resp := doJSON(t, client, http.MethodGet, ts.URL+"/api/students/missing", "")
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("status = %d, want 404", resp.StatusCode)
		}
		resp.Body.Close()
	})
}

func TestAPI_Students_GetAndProgress(t *testing.T) {
	ts, client := newTestServer(t)
	resp := doJSON(t, client, http.MethodPost, ts.URL+"/api/students", `{"name":"Ola","email":"ola@lnu.ua"}`)
	var st studentDTO
	decodeBody(t, resp, &st)

	t.Run("get found", func(t *testing.T) {
		r := doJSON(t, client, http.MethodGet, ts.URL+"/api/students/"+st.ID, "")
		if r.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", r.StatusCode)
		}
		var got studentDTO
		decodeBody(t, r, &got)
		if got.ID != st.ID {
			t.Errorf("id = %q, want %q", got.ID, st.ID)
		}
	})

	t.Run("progress empty", func(t *testing.T) {
		r := doJSON(t, client, http.MethodGet, ts.URL+"/api/students/"+st.ID+"/progress", "")
		if r.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", r.StatusCode)
		}
		var views []progressDTO
		decodeBody(t, r, &views)
		if len(views) != 0 {
			t.Errorf("views = %v, want empty", views)
		}
	})
}

func TestAPI_Courses(t *testing.T) {
	ts, client := newTestServer(t)

	t.Run("create valid online certified", func(t *testing.T) {
		resp := doJSON(t, client, http.MethodPost, ts.URL+"/api/courses",
			`{"code":"CS101","title":"Algo","credits":5,"type":"online","grading":"weighted","features":["certified"],"platform":"Moodle"}`)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("status = %d, want 201", resp.StatusCode)
		}
		var dto courseDTO
		decodeBody(t, resp, &dto)
		if len(dto.Features) != 2 {
			t.Errorf("features = %v, want online+certified", dto.Features)
		}
	})

	t.Run("create invalid grading", func(t *testing.T) {
		resp := doJSON(t, client, http.MethodPost, ts.URL+"/api/courses", `{"code":"X","title":"T","grading":"bogus"}`)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", resp.StatusCode)
		}
		resp.Body.Close()
	})

	t.Run("create duplicate code", func(t *testing.T) {
		body := `{"code":"DUP","title":"T","credits":1,"grading":"letter"}`
		_ = doJSON(t, client, http.MethodPost, ts.URL+"/api/courses", body).Body.Close()
		resp := doJSON(t, client, http.MethodPost, ts.URL+"/api/courses", body)
		if resp.StatusCode != http.StatusConflict {
			t.Errorf("status = %d, want 409", resp.StatusCode)
		}
		resp.Body.Close()
	})

	t.Run("create malformed", func(t *testing.T) {
		resp := doJSON(t, client, http.MethodPost, ts.URL+"/api/courses", `nope`)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", resp.StatusCode)
		}
		resp.Body.Close()
	})

	t.Run("list", func(t *testing.T) {
		resp := doJSON(t, client, http.MethodGet, ts.URL+"/api/courses", "")
		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want 200", resp.StatusCode)
		}
		resp.Body.Close()
	})
}

// seedStudentAndCourse creates one student and one course and returns their IDs.
func seedStudentAndCourse(t *testing.T, ts *httptest.Server, client *http.Client) (string, string) {
	t.Helper()
	var st studentDTO
	decodeBody(t, doJSON(t, client, http.MethodPost, ts.URL+"/api/students", `{"name":"S","email":"s@lnu.ua"}`), &st)
	var c courseDTO
	decodeBody(t, doJSON(t, client, http.MethodPost, ts.URL+"/api/courses", `{"code":"CS1","title":"T","credits":3,"grading":"weighted"}`), &c)
	return st.ID, c.ID
}

func TestAPI_Enrollments(t *testing.T) {
	ts, client := newTestServer(t)
	studentID, courseID := seedStudentAndCourse(t, ts, client)

	var enrollmentID string
	t.Run("enroll valid", func(t *testing.T) {
		resp := doJSON(t, client, http.MethodPost, ts.URL+"/api/enrollments",
			fmt.Sprintf(`{"studentId":%q,"courseId":%q}`, studentID, courseID))
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("status = %d, want 201", resp.StatusCode)
		}
		var dto enrollmentDTO
		decodeBody(t, resp, &dto)
		enrollmentID = dto.ID
		if dto.Status != string(domain.StatusPending) {
			t.Errorf("status = %q, want pending", dto.Status)
		}
	})

	t.Run("enroll duplicate", func(t *testing.T) {
		resp := doJSON(t, client, http.MethodPost, ts.URL+"/api/enrollments",
			fmt.Sprintf(`{"studentId":%q,"courseId":%q}`, studentID, courseID))
		if resp.StatusCode != http.StatusConflict {
			t.Errorf("status = %d, want 409", resp.StatusCode)
		}
		resp.Body.Close()
	})

	t.Run("enroll course not found", func(t *testing.T) {
		resp := doJSON(t, client, http.MethodPost, ts.URL+"/api/enrollments",
			fmt.Sprintf(`{"studentId":%q,"courseId":"missing"}`, studentID))
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("status = %d, want 404", resp.StatusCode)
		}
		resp.Body.Close()
	})

	t.Run("enroll malformed", func(t *testing.T) {
		resp := doJSON(t, client, http.MethodPost, ts.URL+"/api/enrollments", `bad`)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", resp.StatusCode)
		}
		resp.Body.Close()
	})

	t.Run("grade valid", func(t *testing.T) {
		resp := doJSON(t, client, http.MethodPut, ts.URL+"/api/enrollments/"+enrollmentID+"/grade",
			`{"components":[{"name":"exam","score":90,"maxScore":100,"weight":1}],"planned":1}`)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		var dto enrollmentDTO
		decodeBody(t, resp, &dto)
		if dto.Letter != "A" || !dto.Passed || dto.Progress != 1 {
			t.Errorf("dto = %+v, want A/passed/progress 1", dto)
		}
	})

	t.Run("grade no components", func(t *testing.T) {
		resp := doJSON(t, client, http.MethodPut, ts.URL+"/api/enrollments/"+enrollmentID+"/grade", `{"components":[]}`)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", resp.StatusCode)
		}
		resp.Body.Close()
	})

	t.Run("grade enrollment not found", func(t *testing.T) {
		resp := doJSON(t, client, http.MethodPut, ts.URL+"/api/enrollments/missing/grade",
			`{"components":[{"name":"x","score":1,"maxScore":1,"weight":1}],"planned":1}`)
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("status = %d, want 404", resp.StatusCode)
		}
		resp.Body.Close()
	})

	t.Run("grade malformed", func(t *testing.T) {
		resp := doJSON(t, client, http.MethodPut, ts.URL+"/api/enrollments/"+enrollmentID+"/grade", `oops`)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", resp.StatusCode)
		}
		resp.Body.Close()
	})
}

func TestWebPages(t *testing.T) {
	ts, client := newTestServer(t)
	seedStudentAndCourse(t, ts, client)

	pages := []struct {
		name string
		path string
	}{
		{"home", "/"},
		{"students", "/students"},
		{"courses", "/courses"},
		{"progress", "/progress"},
	}
	for _, p := range pages {
		t.Run(p.name, func(t *testing.T) {
			resp := doJSON(t, client, http.MethodGet, ts.URL+p.path, "")
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want 200", resp.StatusCode)
			}
			ct := resp.Header.Get("Content-Type")
			if !strings.Contains(ct, "text/html") {
				t.Errorf("content-type = %q, want html", ct)
			}
		})
	}
}

func TestWebPage_NotFound(t *testing.T) {
	ts, client := newTestServer(t)
	resp := doJSON(t, client, http.MethodGet, ts.URL+"/does-not-exist", "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestWebForms(t *testing.T) {
	ts, client := newTestServer(t)

	post := func(path string, form url.Values) *http.Response {
		req, _ := http.NewRequest(http.MethodPost, ts.URL+path, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}

	t.Run("create student redirects", func(t *testing.T) {
		resp := post("/students", url.Values{"name": {"Web"}, "email": {"web@lnu.ua"}})
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusSeeOther {
			t.Errorf("status = %d, want 303", resp.StatusCode)
		}
	})

	t.Run("create course redirects", func(t *testing.T) {
		resp := post("/courses", url.Values{
			"code": {"WEB1"}, "title": {"Web"}, "credits": {"4"},
			"type": {"online"}, "grading": {"letter"}, "platform": {"Moodle"}, "certified": {"on"},
		})
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusSeeOther {
			t.Errorf("status = %d, want 303", resp.StatusCode)
		}
	})

	// Seed proper IDs for enroll/grade forms.
	studentID, courseID := seedStudentAndCourse(t, ts, client)

	t.Run("enroll redirects", func(t *testing.T) {
		resp := post("/enroll", url.Values{"student": {studentID}, "course": {courseID}})
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusSeeOther {
			t.Errorf("status = %d, want 303", resp.StatusCode)
		}
	})

	t.Run("progress for student renders", func(t *testing.T) {
		resp := doJSON(t, client, http.MethodGet, ts.URL+"/progress?student="+studentID, "")
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want 200", resp.StatusCode)
		}
	})

	t.Run("enroll invalid course redirects to error", func(t *testing.T) {
		resp := post("/enroll", url.Values{"student": {studentID}, "course": {"missing"}})
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("status = %d, want 404", resp.StatusCode)
		}
	})
}

func TestStatusFor(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"not found", domain.ErrNotFound, http.StatusNotFound},
		{"validation", domain.ErrValidation, http.StatusBadRequest},
		{"duplicate", domain.ErrDuplicate, http.StatusConflict},
		{"already enrolled", domain.ErrAlreadyEnrolled, http.StatusConflict},
		{"wrapped not found", fmt.Errorf("ctx: %w", domain.ErrNotFound), http.StatusNotFound},
		{"generic", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := statusFor(tt.err); got != tt.want {
				t.Errorf("statusFor(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}

func TestDecodeJSON(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{"valid", `{"name":"x"}`, false},
		{"empty object", `{}`, false},
		{"malformed", `{`, true},
		{"unknown field", `{"name":"x","extra":1}`, true},
		{"not json", `hello`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(tt.body))
			var p payload
			err := decodeJSON(req, &p)
			if tt.wantErr && err == nil {
				t.Errorf("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestTemplateFuncs(t *testing.T) {
	pct := templateFuncs["pct"].(func(float64) string)
	pctTests := []struct {
		in   float64
		want string
	}{
		{0, "0%"}, {0.5, "50%"}, {1, "100%"}, {0.756, "76%"}, {0.754, "75%"},
	}
	for _, tt := range pctTests {
		t.Run(fmt.Sprintf("pct_%.3f", tt.in), func(t *testing.T) {
			if got := pct(tt.in); got != tt.want {
				t.Errorf("pct(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}

	initials := templateFuncs["initials"].(func(string) string)
	initTests := []struct {
		in   string
		want string
	}{
		{"Іван Петренко", "ІП"},
		{"Single", "S"},
		{"", "?"},
		{"   ", "?"},
		{"a b c", "AB"},
		{"john doe smith", "JD"},
	}
	for _, tt := range initTests {
		t.Run("initials_"+tt.in, func(t *testing.T) {
			if got := initials(tt.in); got != tt.want {
				t.Errorf("initials(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestReqContext(t *testing.T) {
	type ctxKey string
	const k ctxKey = "k"
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(context.WithValue(context.Background(), k, "v"))
	if reqContext(req).Value(k) != "v" {
		t.Error("reqContext did not preserve request context")
	}
}
